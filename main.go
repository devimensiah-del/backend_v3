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

	"backend_v3/adapter/macrodata"
	"backend_v3/api"
	"backend_v3/config"
	"backend_v3/domain/analysis"
	"backend_v3/domain/company"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/macroeconomics"
	"backend_v3/domain/report"
	"backend_v3/domain/submission"
	"backend_v3/infrastructure"
	"backend_v3/jobs"
	"backend_v3/llm"

	"github.com/google/uuid"
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

// macroServiceAdapter implements enrichment.MacroServiceInterface
// Converts macroeconomics.LatestSnapshot to enrichment.MacroSnapshot
type macroServiceAdapter struct {
	svc *macroeconomics.Service
}

func (a macroServiceAdapter) GetLatestSnapshot(ctx context.Context) (*enrichment.MacroSnapshot, error) {
	snapshot, err := a.svc.GetLatestSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, nil
	}

	// Convert macroeconomics types to enrichment types (dynamic map)
	result := &enrichment.MacroSnapshot{
		Indicators: make(map[string]*enrichment.MacroIndicator),
		AsOf:       snapshot.AsOf,
	}

	// Convert each indicator from macroeconomics to enrichment type
	for code, v := range snapshot.Indicators {
		if v != nil {
			result.Indicators[code] = &enrichment.MacroIndicator{
				Code:            v.Code,
				Name:            v.Name,
				Category:        v.Category,
				Value:           v.Value,
				Unit:            v.Unit,
				EffectiveDate:   v.EffectiveDate,
				ReferencePeriod: v.ReferencePeriod,
				SourceCode:      v.SourceCode,
				FetchedAt:       v.FetchedAt,
			}
		}
	}

	return result, nil
}

// companyServiceAdapterForSubmission implements submission.CompanyServiceInterface
type companyServiceAdapterForSubmission struct {
	svc *company.Service
}

func (a companyServiceAdapterForSubmission) CreateFromSubmission(ctx context.Context, input submission.CompanyCreateInput) error {
	_, err := a.svc.CreateFromSubmission(ctx, company.CreateFromSubmissionInput{
		SubmissionID:     input.SubmissionID,
		CompanyName:      input.CompanyName,
		CNPJ:             input.CNPJ,
		Website:          input.Website,
		Industry:         input.Industry,
		CompanySize:      input.CompanySize,
		Location:         input.Location,
		TargetMarket:     input.TargetMarket,
		FundingStage:     input.FundingStage,
		AnnualRevenueMin: input.AnnualRevenueMin,
		AnnualRevenueMax: input.AnnualRevenueMax,
		LinkedInURL:      input.LinkedInURL,
		TwitterHandle:    input.TwitterHandle,
	})
	return err
}

func (a companyServiceAdapterForSubmission) IsVerifiedCNPJExists(ctx context.Context, cnpj string) (bool, error) {
	return a.svc.IsVerifiedCNPJExists(ctx, cnpj)
}

// companyServiceAdapterForEnrichment implements enrichment.CompanyServiceInterface
type companyServiceAdapterForEnrichment struct {
	svc *company.Service
}

func (a companyServiceAdapterForEnrichment) UpdateFromEnrichment(ctx context.Context, submissionID string, input enrichment.CompanyUpdateInput) error {
	// Parse the submission ID
	subID, err := uuid.Parse(submissionID)
	if err != nil {
		return err
	}

	// Get the company by submission ID
	comp, err := a.svc.GetBySubmissionID(ctx, subID)
	if err != nil {
		return err
	}
	if comp == nil {
		log.Warn().Str("submission_id", submissionID).Msg("No company found for submission - skipping enrichment update")
		return nil
	}

	// Parse enrichment ID
	enrichID, err := uuid.Parse(input.EnrichmentID)
	if err != nil {
		return err
	}

	// Update the company (COALESCE behavior)
	return a.svc.UpdateFromEnrichment(ctx, comp.ID, company.UpdateFromEnrichmentInput{
		EnrichmentID:      enrichID,
		FoundationYear:    input.FoundationYear,
		LegalName:         input.LegalName,
		Headquarters:      input.Headquarters,
		Sector:            input.Sector,
		TargetAudience:    input.TargetAudience,
		ValueProposition:  input.ValueProposition,
		EmployeesRange:    input.EmployeesRange,
		RevenueEstimate:   input.RevenueEstimate,
		BusinessModel:     input.BusinessModel,
		Competitors:       input.Competitors,
		MarketShareStatus: input.MarketShareStatus,
		DigitalMaturity:   input.DigitalMaturity,
		Strengths:         input.Strengths,
		Weaknesses:        input.Weaknesses,
		CNPJ:              input.CNPJ,
		Website:           input.Website,
		LinkedInURL:       input.LinkedInURL,
		TwitterHandle:     input.TwitterHandle,
	})
}

// UpdateFromEnrichmentSmartMerge uses smart merge (respects verified fields)
// Includes retry logic to handle race condition with async company creation
func (a companyServiceAdapterForEnrichment) UpdateFromEnrichmentSmartMerge(ctx context.Context, submissionID string, input enrichment.CompanyUpdateInput) error {
	// Parse the submission ID
	subID, err := uuid.Parse(submissionID)
	if err != nil {
		return err
	}

	// Parse enrichment ID early
	enrichID, err := uuid.Parse(input.EnrichmentID)
	if err != nil {
		return err
	}

	// Get the company by submission ID with retry (handles race condition with async company creation)
	// Company creation runs in a goroutine when submission is created, so it may not exist yet
	var comp *company.Company
	maxRetries := 5
	retryDelay := 500 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		comp, err = a.svc.GetBySubmissionID(ctx, subID)
		if err != nil {
			return err
		}
		if comp != nil {
			break // Found it!
		}

		// Company not found yet - wait and retry
		if attempt < maxRetries-1 {
			log.Debug().
				Str("submission_id", submissionID).
				Int("attempt", attempt+1).
				Int("max_retries", maxRetries).
				Dur("delay", retryDelay).
				Msg("Company not found yet, waiting for async creation...")
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff: 500ms, 1s, 2s, 4s
		}
	}

	if comp == nil {
		log.Warn().
			Str("submission_id", submissionID).
			Int("retries", maxRetries).
			Msg("No company found for submission after retries - skipping enrichment update")
		return nil
	}

	log.Info().
		Str("submission_id", submissionID).
		Str("company_id", comp.ID.String()).
		Str("company_name", comp.Name).
		Msg("Company found, updating with enrichment data")

	// Update the company (Smart Merge - respects verified fields)
	return a.svc.UpdateFromEnrichmentSmartMerge(ctx, comp.ID, company.UpdateFromEnrichmentInput{
		EnrichmentID:      enrichID,
		FoundationYear:    input.FoundationYear,
		LegalName:         input.LegalName,
		Headquarters:      input.Headquarters,
		Sector:            input.Sector,
		TargetAudience:    input.TargetAudience,
		ValueProposition:  input.ValueProposition,
		EmployeesRange:    input.EmployeesRange,
		RevenueEstimate:   input.RevenueEstimate,
		BusinessModel:     input.BusinessModel,
		Competitors:       input.Competitors,
		MarketShareStatus: input.MarketShareStatus,
		DigitalMaturity:   input.DigitalMaturity,
		Strengths:         input.Strengths,
		Weaknesses:        input.Weaknesses,
		CNPJ:              input.CNPJ,
		Website:           input.Website,
		LinkedInURL:       input.LinkedInURL,
		TwitterHandle:     input.TwitterHandle,
	})
}

// GetCompanyIDBySubmissionID retrieves the company ID linked to a submission
func (a companyServiceAdapterForEnrichment) GetCompanyIDBySubmissionID(ctx context.Context, submissionID string) (*string, error) {
	// Parse the submission ID
	subID, err := uuid.Parse(submissionID)
	if err != nil {
		return nil, err
	}

	// Get the company by submission ID
	comp, err := a.svc.GetBySubmissionID(ctx, subID)
	if err != nil {
		return nil, err
	}
	if comp == nil {
		return nil, nil
	}

	id := comp.ID.String()
	return &id, nil
}

// GetCompanyDataByID retrieves company data by ID for enrichment purposes
func (a companyServiceAdapterForEnrichment) GetCompanyDataByID(ctx context.Context, companyID string) (*enrichment.CompanyData, error) {
	// Parse the company ID
	compID, err := uuid.Parse(companyID)
	if err != nil {
		return nil, err
	}

	// Get the company by ID
	comp, err := a.svc.GetByID(ctx, compID)
	if err != nil {
		return nil, err
	}
	if comp == nil {
		return nil, nil
	}

	// Convert to CompanyData
	return &enrichment.CompanyData{
		ID:               comp.ID.String(),
		Name:             comp.Name,
		CNPJ:             comp.CNPJ,
		Website:          comp.Website,
		Industry:         comp.Industry,
		CompanySize:      comp.CompanySize,
		Location:         comp.Location,
		TargetMarket:     comp.TargetMarket,
		FundingStage:     comp.FundingStage,
		AnnualRevenueMin: comp.AnnualRevenueMin,
		AnnualRevenueMax: comp.AnnualRevenueMax,
		FoundationYear:   comp.FoundationYear,
		LegalName:        comp.LegalName,
		Headquarters:     comp.Headquarters,
		Sector:           comp.Sector,
		LinkedInURL:      comp.LinkedInURL,
		TwitterHandle:    comp.TwitterHandle,
	}, nil
}

// GetVerifiedFieldNames retrieves the list of verified field names for a company
func (a companyServiceAdapterForEnrichment) GetVerifiedFieldNames(ctx context.Context, companyID string) ([]string, error) {
	// Parse the company ID
	compID, err := uuid.Parse(companyID)
	if err != nil {
		return nil, err
	}

	return a.svc.GetVerifiedFieldNames(ctx, compID)
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
	companyRepo := company.NewRepository(db)
	log.Info().Msg("All repositories initialized")

	// 6. SERVICES
	log.Info().Msg("Initializing business services...")

	// Company Service (Data persistence for company records)
	companySvc := company.NewService(companyRepo, log.Logger)
	log.Info().Msg("Company service initialized (company data persistence enabled)")

	// Submission (Needs Asynq Client to enqueue jobs)
	asynqClient := asynq.NewClient(redisOpt)
	defer asynqClient.Close()
	subSvc := submission.NewService(subRepo, asynqClient)
	// Inject company service for automatic company creation on submission
	subSvc.SetCompanyService(companyServiceAdapterForSubmission{svc: companySvc})
	log.Info().Msg("Submission service initialized (with company creation)")

	// MacroData Provider (BCB/IBGE APIs for real-time economic indicators)
	macroProvider := macrodata.NewMacroDataProvider()
	log.Info().Msg("MacroData provider initialized (BCB/IBGE APIs for SELIC, IPCA, USD/BRL)")

	// Macroeconomics Domain (Persistent storage for economic indicators)
	macroRepo := macroeconomics.NewRepository(db)
	macroSvc := macroeconomics.NewService(macroRepo, macroProvider)
	log.Info().Msg("Macroeconomics domain initialized (DB-backed economic indicators)")

	// Enrichment (The Researcher)
	// Two-phase enrichment: Pre-Search (Perplexity) + Main Enrichment (Gemini)
	// MacroDataProvider injects authoritative economic data from government APIs
	enrichSvc := enrichment.NewService(
		enrichRepo,
		subRepo,
		llmClient,
		asynqClient,
		cfg.Frameworks["enrichment"], // Gemini with Google Search
		cfg.Frameworks["presearch"],  // Perplexity for company identification
		macroProvider,                // BCB/IBGE APIs for SELIC, IPCA, USD/BRL
	)
	log.Info().
		Str("enrichment_model", cfg.Frameworks["enrichment"].Model).
		Str("presearch_model", cfg.Frameworks["presearch"].Model).
		Float64("temperature", cfg.Frameworks["enrichment"].Temperature).
		Msg("Enrichment service initialized with two-phase pipeline + MacroData APIs")

	// Inject macroeconomics service for DB-first macro data fetching
	// Use adapter to convert macroeconomics.LatestSnapshot → enrichment.MacroSnapshot
	enrichSvc.SetMacroService(macroServiceAdapter{svc: macroSvc})
	log.Info().Msg("MacroService injected into enrichment (DB-first enabled)")

	// Inject company service for automatic company updates after enrichment
	enrichSvc.SetCompanyService(companyServiceAdapterForEnrichment{svc: companySvc})
	log.Info().Msg("CompanyService injected into enrichment (company updates enabled)")

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

	// Inject macroeconomics service into worker for scheduled job handling
	worker.SetMacroService(macroSvc)

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

		// Start Macro Scheduler (AFTER worker is started so handlers are ready)
		macroScheduler, err := macroeconomics.NewScheduler(redisOpt, macroSvc)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create macro scheduler - scheduled jobs disabled")
		} else {
			if err := macroScheduler.RegisterTasks(); err != nil {
				log.Error().Err(err).Msg("Failed to register macro scheduler tasks")
			} else {
				go func() {
					defer func() {
						if r := recover(); r != nil {
							log.Error().
								Interface("panic", r).
								Str("component", "macro_scheduler").
								Msg("PANIC in macro scheduler")
						}
					}()
					if err := macroScheduler.Start(); err != nil {
						log.Error().Err(err).Msg("Macro scheduler failed to start")
					}
				}()
				defer macroScheduler.Stop()
				log.Info().Msg("Macro scheduler started (BRT timezone)")
			}
		}
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
		macroSvc,    // Macroeconomics service for admin endpoints
		companySvc,  // Company service for re-enrich/re-analyze workflows
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
