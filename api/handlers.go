package api

import (
	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/report"
	"backend_v3/domain/submission"

	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Handler holds all service dependencies and configuration
type Handler struct {
	submissionSvc     *submission.Service
	enrichmentSvc     *enrichment.Service
	analysisSvc       *analysis.Service
	reportSvc         *report.Service
	asynqClient       *asynq.Client
	db                *sqlx.DB
	redisClient       *redis.Client
	logger            zerolog.Logger
	supabaseURL       string
	supabaseAnonKey   string // For Auth API calls
	supabaseJWTSecret string // For JWT validation
}

// NewHandler creates a new Handler instance with all dependencies
func NewHandler(
	submissionSvc *submission.Service,
	enrichmentSvc *enrichment.Service,
	analysisSvc *analysis.Service,
	reportSvc *report.Service,
	asynqClient *asynq.Client,
	db *sqlx.DB,
	redisClient *redis.Client,
	logger zerolog.Logger,
	supabaseURL string,
	supabaseAnonKey string,
	supabaseJWTSecret string,
) *Handler {
	return &Handler{
		submissionSvc:     submissionSvc,
		enrichmentSvc:     enrichmentSvc,
		analysisSvc:       analysisSvc,
		reportSvc:         reportSvc,
		asynqClient:       asynqClient,
		db:                db,
		redisClient:       redisClient,
		logger:            logger.With().Str("component", "api").Logger(),
		supabaseURL:       supabaseURL,
		supabaseAnonKey:   supabaseAnonKey,
		supabaseJWTSecret: supabaseJWTSecret,
	}
}

// Handler methods are implemented in separate files:
// - auth_handlers.go: Authentication endpoints
// - submission_handlers.go: Public submission endpoints
// - report_handlers.go: Report generation endpoints
// - admin_handlers.go: Admin-only endpoints
// - health_handlers.go: Health check endpoint
// - helpers.go: Shared utility functions
