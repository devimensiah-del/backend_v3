package api

import (
	domainanalysis "backend_v3/domain/analysis"
	domainenrichment "backend_v3/domain/enrichment"
	domainreport "backend_v3/domain/report"
	domainsubmission "backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// SetupRouter configures all API routes
// Now accepts allowedOrigins for CORS configuration
// The function signature has been changed to reflect the new handler composition
func SetupRouter(
	logger zerolog.Logger,
	jwtSecret string,
	allowedOrigins string,
	isProd bool,
	db *sqlx.DB,
	redisClient *redis.Client,
	asynqClient *asynq.Client,
	supabaseURL string,
	supabaseAnonKey string,
	supabaseJWTSecret string,
	submissionSvc *domainsubmission.Service, // Pass domain services directly
	enrichmentSvc *domainenrichment.Service,
	analysisSvc *domainanalysis.Service,
	reportSvc *domainreport.Service,
) *gin.Engine {
	if isProd {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Create authentication rate limiter (5 attempts per 15 minutes)
	authLimiter := NewAuthRateLimiter()

	// Global middleware (CORS now includes health check bypass)
	router.Use(CORSMiddleware(allowedOrigins, logger))
	router.Use(RequestIDMiddleware())
	router.Use(LoggingMiddleware(logger))
	router.Use(RecoveryMiddleware(logger))
	router.Use(RateLimitMiddleware(100))

	// --- Instantiate Specialized Handlers ---
	// Instantiate the SubmissionResponseBuilder
	submissionResponseBuilder := NewSubmissionResponseBuilder(
		enrichmentSvc,
		analysisSvc,
		reportSvc,
	)

	// Instantiate handlers
	adminHandlers := NewAdminHandlers(
		submissionSvc,
		enrichmentSvc,
		asynqClient,
		logger,
		submissionResponseBuilder, // Pass the builder
	)
	analysisHandlers := NewAnalysisHandlers(
		submissionSvc,
		analysisSvc,
		logger,
	)
	authHandlers := NewAuthHandlers(
		db,
		logger,
		supabaseURL,
		supabaseAnonKey,
		supabaseJWTSecret,
	)
	enrichmentHandlers := NewEnrichmentHandlers(
		submissionSvc,
		enrichmentSvc,
		logger,
	)
	reportHandlers := NewReportHandlers(
		analysisSvc,
		reportSvc,
		submissionSvc,
		logger,
	)
	submissionHandlers := NewSubmissionHandlers(
		submissionSvc,
		enrichmentSvc,
		asynqClient,
		logger,
		submissionResponseBuilder, // Pass the builder
	)
	userHandlers := NewUserHandlers(
		db,
		logger,
	)

	// Create the main API Handler instance
	// This Handler now composes all specialized handlers
	mainHandler := NewHandler(
		adminHandlers,
		analysisHandlers,
		authHandlers,
		enrichmentHandlers,
		reportHandlers,
		submissionHandlers,
		userHandlers,
		submissionResponseBuilder,
		db,
		redisClient,
		logger,
		supabaseURL,
		supabaseAnonKey,
		supabaseJWTSecret,
	)

	// Health check endpoint (will bypass CORS)
	router.GET("/health", mainHandler.HealthCheck)

	// Public API routes (v1)
	publicAPI := router.Group("/api/v1")
	{
		publicAPI.POST("/submit", mainHandler.SubmissionHandlers.CreateSubmission)
		// Frontend expects /submissions endpoint
		publicAPI.POST("/submissions", mainHandler.SubmissionHandlers.CreateSubmission)
	}

	// Public Auth routes (no auth required)
	// SECURITY: Apply stricter rate limiting to prevent brute force attacks
	publicAuthAPI := router.Group("/api/v1/auth")
	publicAuthAPI.Use(AuthRateLimitMiddleware(authLimiter)) // This middleware might need adjustment if it uses `handler` directly
	{
		publicAuthAPI.POST("/login", mainHandler.AuthHandlers.Login)
		publicAuthAPI.POST("/signup", mainHandler.AuthHandlers.Signup)
		publicAuthAPI.POST("/forgot-password", mainHandler.AuthHandlers.ForgotPassword)
		publicAuthAPI.POST("/reset-password", mainHandler.AuthHandlers.ResetPassword)
	}

	// Protected Auth routes (authentication required)
	// AuthMiddleware will need to be updated to inject the correct user ID/role
	authAPI := router.Group("/api/v1/auth")
	authAPI.Use(AuthMiddleware(jwtSecret, db)) // AuthMiddleware might need to be adjusted
	{
		authAPI.GET("/me", mainHandler.AuthHandlers.GetCurrentUser)
		authAPI.POST("/logout", mainHandler.AuthHandlers.Logout)
		authAPI.PUT("/update-password", mainHandler.AuthHandlers.UpdatePassword)
	}

	// User profile alias (frontend expects /user/profile)
	userAPI := router.Group("/api/v1/user")
	userAPI.Use(AuthMiddleware(jwtSecret, db)) // AuthMiddleware might need to be adjusted
	{
		userAPI.GET("/profile", mainHandler.AuthHandlers.GetCurrentUser)
		userAPI.PUT("/profile", mainHandler.UserHandlers.UpdateUserProfile)
	}

	// Protected User Routes (v1)
	protectedAPI := router.Group("/api/v1")
	protectedAPI.Use(AuthMiddleware(jwtSecret, db)) // AuthMiddleware might need to be adjusted
	{
		// List user's own submissions
		protectedAPI.GET("/submissions", mainHandler.SubmissionHandlers.ListUserSubmissions)
		protectedAPI.GET("/submissions/:id", mainHandler.SubmissionHandlers.GetSubmission)
		protectedAPI.GET("/submissions/:id/enrichment", mainHandler.EnrichmentHandlers.GetEnrichment)
		protectedAPI.GET("/submissions/:id/analysis", mainHandler.AnalysisHandlers.GetAnalysis)
		protectedAPI.GET("/submissions/:id/report/preview", mainHandler.ReportHandlers.PreviewReport)
		protectedAPI.POST("/submissions/:id/report/publish", mainHandler.ReportHandlers.PublishReport)
		protectedAPI.GET("/submissions/:id/report/download", mainHandler.ReportHandlers.DownloadReport)
	}

	// Admin API routes (v1)
	adminAPI := router.Group("/api/v1/admin")
	adminAPI.Use(AuthMiddleware(jwtSecret, db)) // AuthMiddleware might need to be adjusted
	adminAPI.Use(AdminAuthMiddleware())         // AdminAuthMiddleware might need to be adjusted
	{
		// Submission management
		adminAPI.GET("/submissions", mainHandler.AdminHandlers.ListSubmissions)
		adminAPI.GET("/submissions/:id", mainHandler.AdminHandlers.GetSubmissionAdmin)
		adminAPI.GET("/submissions/:id/enrichment", mainHandler.EnrichmentHandlers.GetEnrichmentBySubmissionAdmin)
		// REMOVED: PUT /submissions/:id/status - Violates "Single Status Rule"
		// Submissions always have status "received". Status is derived from Enrichment/Analysis.
		adminAPI.POST("/submissions/:id/retry-enrichment", mainHandler.AdminHandlers.RetryEnrichment)
		adminAPI.POST("/submissions/:id/retry-analysis", mainHandler.AdminHandlers.RetryAnalysis)
		adminAPI.GET("/analytics", mainHandler.AdminHandlers.GetAnalytics)

		// Enrichment management
		adminAPI.GET("/enrichment/:id", mainHandler.EnrichmentHandlers.GetEnrichmentAdmin)
		adminAPI.PUT("/enrichment/:id", mainHandler.EnrichmentHandlers.UpdateEnrichment)
		adminAPI.POST("/enrichment/:id/approve", mainHandler.EnrichmentHandlers.ApproveEnrichment)
		adminAPI.POST("/enrichment/:id/unlock", mainHandler.EnrichmentHandlers.UnlockEnrichment)

		// Analysis management
		adminAPI.GET("/analysis/:id", mainHandler.AnalysisHandlers.GetAnalysisAdmin)
		adminAPI.PUT("/analysis/:id", mainHandler.AnalysisHandlers.UpdateAnalysis)
		adminAPI.POST("/analysis/:id/version", mainHandler.AnalysisHandlers.CreateAnalysisVersion)
		adminAPI.POST("/analysis/:id/approve", mainHandler.AnalysisHandlers.ApproveAnalysis)
		adminAPI.POST("/analysis/:id/send", mainHandler.AnalysisHandlers.SendAnalysis)
	}

	return router
}
