package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/sessionMaker"

	"github.com/blockedby/positions-os/internal/collector"
	"github.com/blockedby/positions-os/internal/config"
	"github.com/blockedby/positions-os/internal/database"
	"github.com/blockedby/positions-os/internal/logger"
	"github.com/blockedby/positions-os/internal/nats"
	"github.com/blockedby/positions-os/internal/publisher"
	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/telegram"
	"github.com/blockedby/positions-os/internal/web"
	"github.com/blockedby/positions-os/internal/web/handlers"
)

func main() {
	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// 2. Initialize logger
	if err := logger.Init(cfg.LogLevel, cfg.LogFile); err != nil {
		panic("failed to init logger: " + err.Error())
	}
	log := logger.Get()
	log.Info().Msg("starting unified collector & web service")

	// 3. Setup context with graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info().Msg("received shutdown signal")
		cancel()
	}()

	// 4. Connect to database
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// 5. Connect to NATS
	nc, err := nats.New(ctx, cfg.NatsURL)
	if err != nil {
		log.Warn().Err(err).Msg("failed to connect to nats, publishing disabled")
	} else {
		defer nc.Close()
	}

	var pub collector.EventPublisher
	if nc != nil {
		pub = publisher.NewNATSPublisher(nc.Conn)
	}

	// 6. Initialize repositories
	targetsRepo := repository.NewTargetsRepository(db.Pool)
	jobsRepo := repository.NewJobsRepository(db.Pool)
	rangesRepo := repository.NewRangesRepository(db.Pool)
	statsRepo := repository.NewStatsRepository(db.Pool)

	// 7. Initialize telegram client
	if cfg.TGApiID == 0 || cfg.TGApiHash == "" || cfg.TGSessionStr == "" {
		log.Fatal().Msg("TG_API_ID, TG_API_HASH and TG_SESSION_STRING are required")
	}

	log.Info().Msg("initializing telegram client...")
	tgProtoClient, err := gotgproto.NewClient(
		cfg.TGApiID,
		cfg.TGApiHash,
		gotgproto.ClientTypePhone(""),
		&gotgproto.ClientOpts{
			Session:          sessionMaker.StringSession(cfg.TGSessionStr),
			DisableCopyright: true,
			InMemory:         true,
		},
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create telegram client")
	}

	tgClient := telegram.NewClient(tgProtoClient)
	defer tgClient.Close()

	// 8. Initialize Collector Service & Manager
	svc := collector.NewService(
		tgClient,
		targetsRepo,
		jobsRepo,
		rangesRepo,
		pub,
		log,
	)
	scrapeManager := collector.NewScrapeManager(svc)
	collectorHandler := collector.NewHandler(scrapeManager, targetsRepo)

	// 9. Initialize Web UI & WebSocket Hub
	hub := web.NewHub()
	go hub.Run()

	tmpl := web.NewTemplateEngine(cfg.TemplatesDir, true) // reload = true for dev
	if err := tmpl.Load(); err != nil {
		log.Fatal().Err(err).Msg("failed to load templates")
	}

	// 10. Initialize Web Handlers
	pagesHandler := handlers.NewPagesHandler(tmpl, jobsRepo, statsRepo)
	jobsAPIHandler := handlers.NewJobsHandler(jobsRepo, hub)
	targetsAPIHandler := handlers.NewTargetsHandler(targetsRepo, tmpl)
	statsAPIHandler := handlers.NewStatsHandler(statsRepo)

	// 11. Initialize Server
	webCfg := &web.Config{
		Port:         cfg.HTTPPort,
		StaticDir:    cfg.StaticDir,
		TemplatesDir: cfg.TemplatesDir,
	}
	server := web.NewServer(webCfg, nil, hub)

	// 12. Register all handlers
	server.RegisterPagesHandler(pagesHandler)
	server.RegisterJobsHandler(jobsAPIHandler)
	server.RegisterTargetsHandler(targetsAPIHandler)
	server.RegisterStatsHandler(statsAPIHandler)
	server.RegisterCollectorHandler(collectorHandler)

	// 13. Start Server
	log.Info().Int("port", cfg.HTTPPort).Msg("starting web server")
	go func() {
		if err := server.Start(); err != nil {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	// 14. Wait for shutdown
	<-ctx.Done()
	log.Info().Msg("shutting down services...")

	scrapeManager.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	server.Stop(shutdownCtx)

	log.Info().Msg("shutdown complete")
}
