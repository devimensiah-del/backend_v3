package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"backend_v3/api"
	"backend_v3/config"
	"backend_v3/domain/analysis"
	"backend_v3/domain/challenge"
	"backend_v3/domain/company"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/submission"
	"backend_v3/domain/wizard"
	"backend_v3/internal/adapters"
	"backend_v3/jobs"
	"backend_v3/llm"

	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

func main() {
	// Add panic recovery at the top level
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Interface("panic", r).
				Str("stack_trace", string(debug.Stack())).
				Msg("PANIC: Application crashed")
			os.Exit(1)
		}
	}()

	// 1. CONFIGURATION
	log.Info().Msg("Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		// Use standard log here as zerolog isn't setup yet
		fmt.Printf("FATAL: Failed to load config: %v\n", err)
		os.Exit(1)
	}

	log.Info().
		Str("environment", cfg.Environment).
		Str("port", cfg.Port).
		Bool("worker_enabled", cfg.WorkerEnabled).
		Msg("Configuration loaded successfully")

	// 2. DATABASE (PostgreSQL)
	log.Info().Msg("Connecting to PostgreSQL database...")
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		// Print to stdout as well to ensure we see the error
		fmt.Printf("DATABASE CONNECTION ERROR: %v\n", err)
		log.Fatal().Err(err).Str("error_detail", err.Error()).Msg("FATAL: Failed to connect to PostgreSQL database")
	}
	defer db.Close()

	// Configure Connection Pool (configurable via environment variables)
	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)

	log.Info().
		Int("max_open_conns", cfg.DBMaxOpenConns).
		Int("max_idle_conns", cfg.DBMaxIdleConns).
		Dur("conn_max_lifetime", cfg.DBConnMaxLifetime).
		Msg("Database connection pool configured")

	// CRITICAL: Test database connection (health check will fail if this fails)
	log.Info().Msg("Testing database connection...")
	if err := db.Ping(); err != nil {
		log.Fatal().
			Err(err).
			Msg("FATAL: Database connection test failed - cannot reach PostgreSQL")
	}
	log.Info().Msg("✓ Database connection verified")

	// 3. REDIS (Asynq & Cache)
	log.Info().Msg("Connecting to Redis...")
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.RedisURL,
		Password: cfg.RedisPassword,
	}
	// Check Redis connection
	rClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURL,
		Password: cfg.RedisPassword,
	})
	ctx := context.Background()
	if err := rClient.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Str("addr", cfg.RedisURL).Msg("FATAL: Failed to connect to Redis")
	}
	log.Info().Msg("Redis connection verified")
	defer rClient.Close()

	// Asynq Client (for enqueuing background jobs)
	asynqClient := asynq.NewClient(redisOpt)
	defer asynqClient.Close()
	log.Info().Msg("Asynq job queue client initialized")

	// 4. EXTERNAL CLIENTS
	log.Info().Msg("Initializing external service clients...")

	// AI Client (OpenRouter)
	llmClient := llm.NewClient(cfg.OpenRouterAPIKey)
	log.Info().Msg("OpenRouter LLM client initialized")

	// 5. REPOSITORIES
	log.Info().Msg("Initializing data repositories...")
	subRepo := submission.NewRepository(db)
	analysisRepo := analysis.NewPostgresRepository(db)
	companyRepo := company.NewRepository(db)
	log.Info().Msg("All repositories initialized")

	// 6. SERVICES
	log.Info().Msg("Initializing business services...")

	// Company Service (Data persistence for company records)
	companySvc := company.NewService(companyRepo, log.Logger, cfg.EnrichmentTimeout)
	log.Info().
		Int("enrichment_timeout_seconds", cfg.EnrichmentTimeout).
		Msg("Company service initialized (company data persistence enabled)")

	// Challenge Domain (Strategic business challenges)
	challengeRepo := challenge.NewRepository(db)
	challengeSvc := challenge.NewService(challengeRepo)
	log.Info().Msg("Challenge domain initialized (business challenge management)")

	// Submission (Entry point for analysis pipeline)
	subSvc := submission.NewService(subRepo)
	// Inject company service for automatic company creation on submission
	subSvc.SetCompanyService(adapters.NewCompanyServiceAdapterForSubmission(companySvc))
	// Inject challenge service for automatic challenge creation on submission
	subSvc.SetChallengeService(adapters.NewChallengeServiceAdapterForSubmission(challengeSvc))
	log.Info().Msg("Submission service initialized (with company + challenge creation)")

	// Enrichment (Two-Stage Process)
	// Stage 1: Perplexity for web search and data gathering
	// Stage 2: Gemini 3 Pro for strategic analysis and synthesis
	enrichSvc := enrichment.NewService(
		llmClient,
		cfg.Frameworks["presearch"],            // Stage 1: Perplexity
		cfg.Frameworks["enrichment_synthesis"], // Stage 2: Gemini 3 Pro
	)
	log.Info().
		Str("stage1_model", cfg.Frameworks["presearch"].Model).
		Str("stage2_model", cfg.Frameworks["enrichment_synthesis"].Model).
		Msg("Enrichment service initialized (two-stage: Perplexity → Gemini 3 Pro)")

	// Inject enrichment service into company service
	companySvc.SetEnrichmentService(enrichSvc)
	log.Info().Msg("Enrichment service injected into company service (automatic enrichment at company creation)")

	// Analysis (The Strategy Team)
	// Create submission repository adapter for analysis service
	submissionRepoAdapter := analysis.NewSubmissionRepositoryAdapter(subRepo)

	analysisSvc := analysis.NewService(
		analysisRepo,
		submissionRepoAdapter,
		llmClient,
		log.Logger,
		asynqClient,
	)
	// Inject framework configs (4-model approach)
	analysisSvc.SetFrameworks(cfg.Frameworks)
	log.Info().
		Int("framework_count", len(cfg.Frameworks)).
		Msg("Analysis service initialized with 4-model configuration")

	// Inject company service for direct company data access
	analysisSvc.SetCompanyService(adapters.NewCompanyServiceAdapterForAnalysis(companySvc))
	log.Info().Msg("CompanyService injected into analysis (company data enabled)")

	// Inject challenge repository for challenge context in prompts
	analysisSvc.SetChallengeRepo(adapters.NewChallengeRepositoryAdapterForAnalysis(challengeRepo))
	log.Info().Msg("ChallengeRepo injected into analysis (challenge context enabled)")

	// Wizard Service (Human-in-the-Loop Framework Wizard)
	wizardSvc := wizard.NewService(
		analysisRepo,
		llmClient,
		cfg.Frameworks,
		log.Logger,
	)
	// Inject company service for getting company data in wizard (same adapter as analysis)
	wizardSvc.SetCompanyService(adapters.NewCompanyServiceAdapterForAnalysis(companySvc))
	log.Info().Msg("Wizard service initialized (human-in-the-loop enabled with company data)")

	// 7. BACKGROUND WORKER
	log.Info().Msg("Initializing background job worker...")
	worker := jobs.NewWorker(
		cfg,
		subSvc,
		enrichSvc, // Used for reading enrichment data in analysis jobs only
		analysisSvc,
		log.Logger,
	)

	// Start Worker in a separate goroutine
	// STABILITY FIX: Graceful degradation - if worker fails, API stays alive
	if cfg.WorkerEnabled {
		log.Info().
			Str("redis_addr", cfg.RedisURL).
			Bool("has_password", cfg.RedisPassword != "").
			Msg("🚀 WORKER_ENABLED=true - Preparing to start background worker")

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().
						Interface("panic", r).
						Str("component", "worker").
						Msg("PANIC in background worker")
				}
			}()

			log.Info().
				Int("concurrency", cfg.WorkerConcurrency).
				Str("queues", cfg.WorkerQueues).
				Str("redis_addr", cfg.RedisURL).
				Msg("🔧 Starting background worker NOW - listening for jobs")

			if err := worker.Start(); err != nil {
				log.Error().
					Err(err).
					Str("redis_addr", cfg.RedisURL).
					Msg("❌ Worker failed to start - background jobs disabled. Check Redis connection.")
				// API continues running, but background processing won't work
				// Admin must use retry endpoints to manually trigger jobs
			} else {
				log.Info().Msg("✅ Background worker started successfully")
			}
		}()
		defer worker.Stop()
	} else {
		log.Warn().Msg("⚠️ Background worker disabled (WORKER_ENABLED=false). Jobs must be triggered manually.")
	}

	// 8. HTTP API
	log.Info().Msg("Setting up HTTP server and routes...")

	router := api.SetupRouter(
		log.Logger,
		cfg.SupabaseJWTSecret, // For AuthMiddleware
		cfg.AllowedOrigins,    // For CORSMiddleware
		cfg.Environment == "production",
		db,      // For role lookup in AuthMiddleware
		rClient, // Redis client for health and caching
		asynqClient,
		cfg.SupabaseURL,
		cfg.SupabaseAnonKey,   // For auth API calls
		cfg.SupabaseJWTSecret, // For JWT validation
		subSvc,
		enrichSvc,
		analysisSvc,
		companySvc,   // Company service for re-enrich/re-analyze workflows
		wizardSvc,    // Wizard service for human-in-the-loop analysis
		challengeSvc, // Challenge service for challenge management
	)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	log.Info().
		Str("allowed_origins", cfg.AllowedOrigins).
		Bool("production_mode", cfg.Environment == "production").
		Msg("Router configured with middleware")

	// 9. START HTTP SERVER
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Interface("panic", r).
					Str("component", "http_server").
					Msg("PANIC in HTTP server")
			}
		}()

		log.Info().
			Str("port", cfg.Port).
			Str("health_check", "/health").
			Msg("HTTP server starting...")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("FATAL: HTTP server failed to start")
		}
	}()

	// Wait for startup to complete
	time.Sleep(100 * time.Millisecond)
	log.Info().Msg("✓ IMENSIAH Backend V3 started successfully")

	// 10. GRACEFUL SHUTDOWN
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Info().
		Str("signal", sig.String()).
		Msg("Shutdown signal received, starting graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Error during graceful shutdown")
	}

	log.Info().Msg("Server shutdown complete")
}
