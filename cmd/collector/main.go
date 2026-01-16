package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blockedby/positions-os/internal/api"
	"github.com/blockedby/positions-os/internal/collector"
	"github.com/blockedby/positions-os/internal/config"
	"github.com/blockedby/positions-os/internal/database"
	"github.com/blockedby/positions-os/internal/logger"
	"github.com/blockedby/positions-os/internal/migrator"
	"github.com/blockedby/positions-os/internal/nats"
	"github.com/blockedby/positions-os/internal/publisher"
	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/telegram"
	"github.com/blockedby/positions-os/internal/web"
	"github.com/blockedby/positions-os/internal/web/handlers"
	"github.com/blockedby/positions-os/migrations"
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
	log.Info().Msg("starting unified collector & API service")

	// 2a. Run database migrations
	log.Info().Msg("running database migrations")
	m, err := migrator.NewWithFS(migrations.FS)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create migrator")
	}

	if err := m.Up(context.Background(), cfg.DatabaseURL); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	version, dirty, err := m.Version(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Warn().Err(err).Msg("failed to get migration version")
	} else {
		log.Info().Uint("version", version).Bool("dirty", dirty).Msg("migrations complete")
	}

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

	// 7. Initialize telegram manager (optional for API-only mode)
	var tgClient *telegram.Client
	if cfg.TGApiID == 0 || cfg.TGApiHash == "" {
		log.Warn().Msg("TG_API_ID and TG_API_HASH not set, running in API-only mode (Telegram scraping disabled)")
	} else {
		tgManager := telegram.NewManager(cfg, db.GORM)
		if err := tgManager.Init(ctx); err != nil {
			log.Error().Err(err).Msg("telegram manager init failed")
			// We continue, status will be Error/Unauthorized
		}

		// Create the high-level Client wrapper
		tgClient = telegram.NewClient(tgManager)
		defer tgClient.Close()
	}

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

	// 9. Initialize WebSocket Hub
	hub := web.NewHub()
	go hub.Run()

	// 10. Initialize API Handlers
	jobsAPIHandler := handlers.NewJobsHandler(jobsRepo, hub)
	targetsAPIHandler := handlers.NewTargetsHandler(targetsRepo)
	statsAPIHandler := handlers.NewStatsHandler(statsRepo)
	authHandler := handlers.NewAuthHandler(tgClient, hub)

	// 11. Initialize Server
	webCfg := &web.Config{
		Port:      cfg.HTTPPort,
		StaticDir: cfg.StaticDir,
	}
	server := web.NewServer(webCfg, nil, hub)

	// 12. Register API handlers
	server.RegisterJobsHandler(jobsAPIHandler)
	server.RegisterTargetsHandler(targetsAPIHandler)
	server.RegisterStatsHandler(statsAPIHandler)
	server.RegisterCollectorHandler(collectorHandler)
	server.RegisterAuthHandler(authHandler)

	// 13. Create Fuego API server for OpenAPI documentation
	// Note: Fuego is used only for OpenAPI spec generation and Scalar UI.
	// The actual API handlers are registered via Chi above.
	// Services can be nil since Fuego handlers won't be called (Chi routes take precedence).
	apiCfg := &api.Config{
		Port:        cfg.HTTPPort,
		Title:       "Positions OS API",
		Description: "Job search automation system API",
		Version:     "1.0.0",
	}
	apiDeps := &api.Dependencies{
		JobsRepo:          jobsRepo,
		TargetsRepo:       targetsRepo,
		StatsRepo:         statsRepo,
		ApplicationsRepo:  nil, // Not needed for OpenAPI generation
		TelegramClient:    tgClient,
		CollectorService:  nil, // Chi handlers handle actual requests
		DispatcherService: nil,
		Hub:               hub,
	}
	apiServer := api.NewServer(apiCfg, apiDeps)

	// Mount OpenAPI documentation routes on the main router
	apiServer.MountDocsOn(server.Router(), apiCfg.Title, apiCfg.Description)

	// Setup SPA fallback (must be last, after all routes are registered)
	server.SetupSPAFallback()

	// 14. Start Server
	log.Info().Int("port", cfg.HTTPPort).Msg("starting API server")
	go func() {
		if err := server.Start(); err != nil {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	// 15. Wait for shutdown
	<-ctx.Done()
	log.Info().Msg("shutting down services...")

	scrapeManager.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	server.Stop(shutdownCtx)

	log.Info().Msg("shutdown complete")
}
