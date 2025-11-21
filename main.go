package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend_v3/api"
	"backend_v3/config"
	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/report"
	"backend_v3/domain/submission"
	"backend_v3/infrastructure"
	"backend_v3/jobs"
	"backend_v3/llm"

	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// 1. CONFIGURATION
	cfg, err := config.Load()
	if err != nil {
		// Use standard log here as zerolog isn't setup yet
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. LOGGING
	if cfg.Environment == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	log.Info().Msg("Starting IMENSIAH Backend V3")

	// 3. DATABASE (PostgreSQL)
	log.Info().Str("url", cfg.DatabaseURL).Msg("Connecting to Database...")
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to DB")
	}
	defer db.Close()

	// Configure Connection Pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 4. REDIS (Asynq & Cache)
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.RedisURL,
		Password: cfg.RedisPassword,
	}
	// Check Redis connection
	rClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURL,
		Password: cfg.RedisPassword,
	})
	if err := rClient.Ping(context.Background()).Err(); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	rClient.Close()

	// 5. EXTERNAL CLIENTS

	// AI Client (OpenRouter)
	llmClient := llm.NewClient(cfg.OpenRouterAPIKey)

	// PDF Generator (Gotenberg)
	pdfGen := infrastructure.NewGotenbergClient(cfg.GotenbergURL)

	// Cloud Storage (Supabase)
	// We use the Service Role Key (mapped to JWT Secret usually) for uploads
	storage := infrastructure.NewSupabaseStorageClient(
		cfg.SupabaseURL,
		"reports", // Bucket Name
		cfg.SupabaseJWTSecret,
	)

	// 6. REPOSITORIES
	subRepo := submission.NewRepository(db)
	enrichRepo := enrichment.NewRepository(db)
	analysisRepo := analysis.NewPostgresRepository(db)
	reportRepo := report.NewPostgresRepository(db)

	// 7. SERVICES

	// Submission (Needs Asynq Client to enqueue jobs)
	asynqClient := asynq.NewClient(redisOpt)
	defer asynqClient.Close()
	subSvc := submission.NewService(subRepo, asynqClient)

	// Enrichment (The Researcher)
	enrichSvc := enrichment.NewService(
		enrichRepo,
		subRepo,
		llmClient,
		cfg.EnrichmentModel,
	)

	// Analysis (The Strategy Team)
	analysisSvc := analysis.NewService(
		analysisRepo,
		llmClient,
		log.Logger,
		cfg.AnalysisModel,
		cfg.SynthesisModel,
	)

	// Report (The Publisher)
	reportSvc := report.NewService(
		reportRepo,
		analysisRepo,
		subRepo,
		pdfGen,
		storage,
		log.Logger,
	)

	// 8. BACKGROUND WORKER
	worker := jobs.NewWorker(
		cfg.RedisURL,
		cfg.RedisPassword,
		subSvc,
		enrichSvc,
		analysisSvc,
		reportSvc,
		log.Logger,
	)

	// Start Worker in a separate goroutine
	if cfg.WorkerEnabled {
		go func() {
			log.Info().Msg("Starting Background Worker")
			if err := worker.Start(); err != nil {
				log.Fatal().Err(err).Msg("Failed to start worker")
			}
		}()
		defer worker.Stop()
	}

	// 9. HTTP API
	handler := api.NewHandler(
		subSvc,
		enrichSvc,
		analysisSvc,
		reportSvc,
		asynqClient,
		db,
		nil,
		log.Logger,
	)

	// CRITICAL: Pass Config values to Router
	router := api.SetupRouter(
		handler,
		log.Logger,
		cfg.SupabaseJWTSecret, // For AuthMiddleware
		cfg.AllowedOrigins,    // For CORSMiddleware
		cfg.Environment == "production",
	)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	// 10. RUN
	go func() {
		log.Info().Msgf("HTTP Server listening on %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}
}
