package api

import (
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Handler holds all service dependencies and configuration, and now composes specialized handlers
type Handler struct {
	// Core dependencies for middleware and global helpers
	db                *sqlx.DB
	redisClient       *redis.Client
	logger            zerolog.Logger
	supabaseURL       string
	supabaseAnonKey   string
	supabaseJWTSecret string

	// Composed Handlers from sub-packages
	AdminHandlers      *AdminHandlers
	AnalysisHandlers   *AnalysisHandlers
	AuthHandlers       *AuthHandlers
	CompanyHandlers    *CompanyHandlers
	EnrichmentHandlers *EnrichmentHandlers
	ReportHandlers     *ReportHandlers
	SubmissionHandlers *SubmissionHandlers
	UserHandlers       *UserHandlers

	// Helper for building detailed submission responses
	SubmissionResponseBuilder *SubmissionResponseBuilder
}

// NewHandler creates a new Handler instance with all dependencies and composed handlers
func NewHandler(
	adminHandlers *AdminHandlers,
	analysisHandlers *AnalysisHandlers,
	authHandlers *AuthHandlers,
	companyHandlers *CompanyHandlers,
	enrichmentHandlers *EnrichmentHandlers,
	reportHandlers *ReportHandlers,
	submissionHandlers *SubmissionHandlers,
	userHandlers *UserHandlers,
	submissionResponseBuilder *SubmissionResponseBuilder,
	db *sqlx.DB,
	redisClient *redis.Client,
	logger zerolog.Logger,
	supabaseURL string,
	supabaseAnonKey string,
	supabaseJWTSecret string,
) *Handler {
	return &Handler{
		db:                db,
		redisClient:       redisClient,
		logger:            logger.With().Str("component", "api").Logger(),
		supabaseURL:       supabaseURL,
		supabaseAnonKey:   supabaseAnonKey,
		supabaseJWTSecret: supabaseJWTSecret,

		AdminHandlers:             adminHandlers,
		AnalysisHandlers:          analysisHandlers,
		AuthHandlers:              authHandlers,
		CompanyHandlers:           companyHandlers,
		EnrichmentHandlers:        enrichmentHandlers,
		ReportHandlers:            reportHandlers,
		SubmissionHandlers:        submissionHandlers,
		UserHandlers:              userHandlers,
		SubmissionResponseBuilder: submissionResponseBuilder,
	}
}
