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
	"github.com/rs/zerolog/log"
)

// reportLookupAdapter implements analysis.ReportLookup without creating package cycles
type reportLookupAdapter struct {
	svc *report.Service
}

type reportSummary struct {
	rep *report.Report
}

func (r reportSummary) GetPDFURL() string { return r.rep.PDFURL }

func (a reportLookupAdapter) GetBySubmissionID(ctx context.Context, submissionID string) (analysis.ReportSummary, error) {
	rep, err := a.svc.GetBySubmissionID(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	return reportSummary{rep: rep}, nil
}

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
		log.Fatal().Err(err).Msg("FATAL: Failed to connect to PostgreSQL database")
	}
	defer db.Close()

	// Configure Connection Pool
	// PRODUCTION FIX: Use configurable connection pool settings instead of hardcoded values
	// This allows tuning for Railway/production limits without code changes
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

	// 4. EXTERNAL CLIENTS
	log.Info().Msg("Initializing external service clients...")

	// AI Client (OpenRouter)
	llmClient := llm.NewClient(cfg.OpenRouterAPIKey)
	log.Info().Msg("OpenRouter LLM client initialized")

	// PDF Generator (Gotenberg)
	pdfGen := infrastructure.NewGotenbergClient(cfg.GotenbergURL)
	log.Info().Str("gotenberg_url", cfg.GotenbergURL).Msg("Gotenberg PDF client initialized")

	// Cloud Storage (Supabase)
	storage := infrastructure.NewSupabaseStorageClient(
		cfg.SupabaseURL,
		cfg.SupabaseBucket,
		cfg.SupabaseServiceKey,
		cfg.SupabaseSignedTTL,
	)
	log.Info().Str("bucket", "reports").Msg("Supabase storage client initialized")

	// 5. REPOSITORIES
	log.Info().Msg("Initializing data repositories...")
	subRepo := submission.NewRepository(db)
	enrichRepo := enrichment.NewRepository(db)
	analysisRepo := analysis.NewPostgresRepository(db)
	reportRepo := report.NewPostgresRepository(db)
	log.Info().Msg("All repositories initialized")

	// 6. SERVICES
	log.Info().Msg("Initializing business services...")

	// Submission (Needs Asynq Client to enqueue jobs)
	asynqClient := asynq.NewClient(redisOpt)
	defer asynqClient.Close()
	subSvc := submission.NewService(subRepo, asynqClient)
	log.Info().Msg("Submission service initialized")

	// Enrichment (The Researcher)
	enrichSvc := enrichment.NewService(
		enrichRepo,
		subRepo,
		llmClient,
		asynqClient,
		cfg.Frameworks["enrichment"],
	)
	log.Info().
		Str("model", cfg.Frameworks["enrichment"].Model).
		Float64("temperature", cfg.Frameworks["enrichment"].Temperature).
		Msg("Enrichment service initialized")

	// Analysis (The Strategy Team)
	// Create submission repository adapter for analysis service
	submissionRepoAdapter := analysis.NewSubmissionRepositoryAdapter(subRepo)

	analysisSvc := analysis.NewService(
		analysisRepo,
		submissionRepoAdapter,
		llmClient,
		log.Logger,
		asynqClient,
		cfg.AnalysisModel,
		cfg.SynthesisModel,
	)
	// Inject all framework-specific configs (heterogeneous routing)
	analysisSvc.SetFrameworks(cfg.Frameworks)
	log.Info().
		Int("framework_count", len(cfg.Frameworks)).
		Msg("Analysis service initialized with heterogeneous model routing")

	// Report (The Publisher)
	reportSvc := report.NewService(
		reportRepo,
		analysisRepo,
		subRepo,
		pdfGen,
		storage,
		log.Logger,
	)
	log.Info().Msg("Report service initialized")
	analysisSvc.SetReportLookup(reportLookupAdapter{svc: reportSvc})

	// 7. BACKGROUND WORKER
	log.Info().Msg("Initializing background job worker...")
	worker := jobs.NewWorker(
		cfg,
		subSvc,
		enrichSvc,
		analysisSvc,
		reportSvc,
		log.Logger,
	)

	// Start Worker in a separate goroutine
	// STABILITY FIX: Graceful degradation - if worker fails, API stays alive
	if cfg.WorkerEnabled {
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
				Msg("Starting background worker")

			if err := worker.Start(); err != nil {
				log.Error().Err(err).Msg("Worker failed to start - background jobs disabled. Check Redis connection.")
				// API continues running, but background processing won't work
				// Admin must use retry endpoints to manually trigger jobs
			}
		}()
		defer worker.Stop()
	} else {
		log.Warn().Msg("Background worker disabled (WORKER_ENABLED=false). Jobs must be triggered manually.")
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
		reportSvc,
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
