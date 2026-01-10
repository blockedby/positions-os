package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"

	"github.com/blockedby/positions-os/internal/collector"
	"github.com/blockedby/positions-os/internal/database"
	"github.com/blockedby/positions-os/internal/logger"
	"github.com/blockedby/positions-os/internal/publisher"
	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/telegram"
)

func main() {
	// load .env file
	_ = godotenv.Load()

	// initialize logger
	logLevel := getEnv("COLLECTOR_LOG_LEVEL", "info")
	logFile := getEnv("COLLECTOR_LOG_FILE", "./logs/collector.log")
	if err := logger.Init(logLevel, logFile); err != nil {
		fmt.Printf("failed to init logger: %v\n", err)
		os.Exit(1)
	}
	log := logger.Get()

	log.Info().Msg("starting collector service")

	// load config
	dbURL := getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/positions_os")
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")

	tgAppIDStr := getEnv("TG_API_ID", "")
	tgAppHash := getEnv("TG_API_HASH", "")
	tgSession := getEnv("TG_SESSION_STRING", "")

	if tgAppIDStr == "" || tgAppHash == "" || tgSession == "" {
		log.Fatal().Msg("TG_API_ID, TG_API_HASH and TG_SESSION_STRING are required")
	}

	log.Info().Int("session_len", len(tgSession)).Str("session_prefix", tgSession[:10]).Msg("loaded session string")

	tgAppID, err := strconv.Atoi(tgAppIDStr)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid TG_API_ID")
	}

	// connect to database
	log.Info().Msg("connecting to database...")
	db, err := database.New(context.Background(), dbURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// initialize repositories
	targetsRepo := repository.NewTargetsRepository(db.Pool)
	jobsRepo := repository.NewJobsRepository(db.Pool)
	rangesRepo := repository.NewRangesRepository(db.Pool)

	// connect to nats
	log.Info().Msg("connecting to nats...")
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Warn().Err(err).Msg("failed to connect to nats, publishing disabled")
	} else {
		defer nc.Close()
	}

	var pub collector.EventPublisher
	if nc != nil {
		pub = publisher.NewNATSPublisher(nc)
	}

	// initialize telegram client
	log.Info().Msg("initializing telegram client...")
	tgProtoClient, err := gotgproto.NewClient(
		tgAppID,
		tgAppHash,
		gotgproto.ClientTypePhone(""), // empty for session auth
		&gotgproto.ClientOpts{
			Session:          sessionMaker.StringSession(tgSession),
			DisableCopyright: true,
			InMemory:         true,
		},
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create telegram client")
	}

	tgClient := telegram.NewClient(tgProtoClient)
	defer tgClient.Close()

	// create service
	svc := collector.NewService(
		tgClient,
		targetsRepo,
		jobsRepo,
		rangesRepo,
		pub,
		log,
	)

	// create manager and handler
	manager := collector.NewScrapeManager(svc) // inject service as scraper
	handler := collector.NewHandler(manager, targetsRepo)
	router := collector.NewRouter(handler)

	// get port from env
	port := getEnv("COLLECTOR_PORT", "3100")
	addr := ":" + port

	// create server
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// start server in goroutine
	go func() {
		log.Info().Str("addr", addr).Msg("listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	// wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server...")

	// stop any running scrape job
	manager.Stop()

	// graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server stopped")
}

// getEnv returns env variable or default value
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
