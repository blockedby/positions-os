package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/blockedby/positions-os/internal/analyzer"
	"github.com/blockedby/positions-os/internal/config"
	"github.com/blockedby/positions-os/internal/database"
	"github.com/blockedby/positions-os/internal/llm"
	"github.com/blockedby/positions-os/internal/logger"
	"github.com/blockedby/positions-os/internal/nats"
	"github.com/blockedby/positions-os/internal/repository"
)

func main() {
	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// 2. Setup Logger
	logger.Init(cfg.LogLevel, cfg.LogFile)
	log := logger.Get()
	log.Info().Msg("starting analyzer service")

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

	// 4. Setup resources
	// Database
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()
	log.Info().Msg("connected to database")

	// NATS
	natsClient, err := nats.New(ctx, cfg.NatsURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to nats")
	}
	defer natsClient.Close()
	log.Info().Msg("connected to nats")

	// Ensure stream exists (optional, collector usually creates it, but good to ensure)
	if err := natsClient.EnsureStream(ctx, "jobs", []string{"jobs.new"}); err != nil {
		log.Fatal().Err(err).Msg("failed to ensure stream")
	}

	// LLM Client
	llmCfg := llm.Config{
		BaseURL:     cfg.LLMBaseURL,
		Model:       cfg.LLMModel,
		APIKey:      cfg.LLMAPIKey,
		MaxTokens:   cfg.LLMMaxTokens,
		Temperature: float32(cfg.LLMTemperature),
		Timeout:     time.Duration(cfg.LLMTimeoutSec) * time.Second,
	}
	llmClient := llm.NewClient(llmCfg)
	log.Info().Str("model", cfg.LLMModel).Msg("llm client initialized")

	// Load Prompts
	// Assuming running from project root or handling paths
	promptPath := filepath.Join("docs", "prompts", "job-extraction.xml")
	// If running inside docker, path might differ, but Dockerfile should copy.
	// For now assume working directory is project root or files are at relative path.
	prompts, err := llm.LoadPrompt(promptPath)
	if err != nil {
		// Fallback or absolute path check?
		// Try absolute path if relative failed
		cwd, _ := os.Getwd()
		log.Warn().Err(err).Str("cwd", cwd).Msg("failed to load prompt from relative path, trying absolute")
		// For now simple error
		log.Fatal().Err(err).Msg("failed to load prompt")
	}
	log.Info().Msg("prompts loaded")

	// Repositories
	jobsRepo := repository.NewJobsRepository(db.Pool)

	// Get zerolog.Logger for components that need it
	zlog := &log.Logger

	// Processor
	processor := analyzer.NewProcessor(llmClient, jobsRepo, prompts, zlog)

	// Consumer
	consumer := analyzer.NewConsumer(natsClient, processor, zlog)

	// 5. Start Consumer
	if err := consumer.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to start consumer")
	}
	log.Info().Msg("consumer started")

	// Wait for shutdown
	<-ctx.Done()
	log.Info().Msg("shutting down")

	// Allow some time for cleanup if needed (e.g. nats drain)
	time.Sleep(1 * time.Second)
	log.Info().Msg("shutdown complete")
}
