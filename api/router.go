package api

import (
	domainanalysis "backend_v3/domain/analysis"
	domaincompany "backend_v3/domain/company"
	domainenrichment "backend_v3/domain/enrichment"
	domainframework "backend_v3/domain/framework"
	domainmacro "backend_v3/domain/macroeconomics"
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
	macroSvc *domainmacro.Service,         // Macroeconomics service for DB-backed indicators
	companySvc *domaincompany.Service,     // Company service for re-enrich/re-analyze workflows
	frameworkSvc *domainframework.Service, // Framework service for framework management
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
		submissionSvc, // For linking anonymous submissions on signup
	)
	enrichmentHandlers := NewEnrichmentHandlers(
		submissionSvc,
		enrichmentSvc,
		analysisSvc,
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
	macroHandlers := NewMacroHandlers(
		macroSvc,
		logger,
	)
	companyHandlers := NewCompanyHandlers(
		companySvc,
		submissionSvc,
		enrichmentSvc,
		asynqClient,
		logger,
	)
	frameworkHandlers := NewFrameworkHandlers(
		frameworkSvc,
		logger,
	)

	// Create the main API Handler instance
	// This Handler now composes all specialized handlers
	mainHandler := NewHandler(
		adminHandlers,
		analysisHandlers,
		authHandlers,
		companyHandlers,
		enrichmentHandlers,
		frameworkHandlers,
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
		// Public report access via access code
		// Uses OptionalAuthMiddleware to allow admin preview (bypasses visibility check)
		publicAPI.GET("/public/report/:code", OptionalAuthMiddleware(jwtSecret, db), mainHandler.AnalysisHandlers.GetPublicReport)

		// Framework routes (public - active frameworks only)
		publicAPI.GET("/frameworks", mainHandler.FrameworkHandlers.List)
		publicAPI.GET("/frameworks/:code", mainHandler.FrameworkHandlers.GetByCode)
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
		userAPI.DELETE("", mainHandler.UserHandlers.DeleteAccount)
	}

	// Protected User Routes (v1)
	protectedAPI := router.Group("/api/v1")
	protectedAPI.Use(AuthMiddleware(jwtSecret, db)) // AuthMiddleware might need to be adjusted
	{
		// List user's own submissions (legacy - kept for backwards compatibility)
		protectedAPI.GET("/submissions", mainHandler.SubmissionHandlers.ListUserSubmissions)
		protectedAPI.GET("/submissions/:id", mainHandler.SubmissionHandlers.GetSubmission)
		protectedAPI.GET("/submissions/:id/enrichment", mainHandler.EnrichmentHandlers.GetEnrichment)
		protectedAPI.GET("/submissions/:id/analysis", mainHandler.AnalysisHandlers.GetAnalysis)
		protectedAPI.GET("/submissions/:id/report/preview", mainHandler.ReportHandlers.PreviewReport)
		protectedAPI.POST("/submissions/:id/report/publish", mainHandler.ReportHandlers.PublishReport)
		protectedAPI.GET("/submissions/:id/report/download", mainHandler.ReportHandlers.DownloadReport)

		// Company routes - user's companies (owner or in allowed_users)
		protectedAPI.GET("/companies", mainHandler.CompanyHandlers.GetMyCompanies)
		protectedAPI.GET("/companies/:id", mainHandler.CompanyHandlers.GetCompany)
		protectedAPI.PUT("/companies/:id", mainHandler.CompanyHandlers.UpdateCompanyUser)
		protectedAPI.POST("/companies/:id/users", mainHandler.CompanyHandlers.AddUserToCompany)
		protectedAPI.DELETE("/companies/:id/users/:userId", mainHandler.CompanyHandlers.RemoveUserFromCompany)

		// User-level field verification - users can verify their own company data
		protectedAPI.GET("/companies/:id/verifications", mainHandler.CompanyHandlers.GetFieldVerificationsUser)
		protectedAPI.POST("/companies/:id/verifications", mainHandler.CompanyHandlers.VerifyFieldUser)
		protectedAPI.POST("/companies/:id/verifications/all", mainHandler.CompanyHandlers.VerifyAllFieldsUser)
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
		adminAPI.GET("/submissions/:id/analysis", mainHandler.AnalysisHandlers.GetAnalysisBySubmissionAdmin)
		// REMOVED: PUT /submissions/:id/status - Violates "Single Status Rule"
		// Submissions always have status "received". Status is derived from Enrichment/Analysis.
		adminAPI.POST("/submissions/:id/retry-enrichment", mainHandler.AdminHandlers.RetryEnrichment)
		adminAPI.POST("/submissions/:id/retry-analysis", mainHandler.AdminHandlers.RetryAnalysis)
		adminAPI.GET("/analytics", mainHandler.AdminHandlers.GetAnalytics)
		adminAPI.GET("/metrics", mainHandler.GetMetrics) // System-wide metrics (LLM costs, success rates)

		// Enrichment management
		adminAPI.GET("/enrichment/:id", mainHandler.EnrichmentHandlers.GetEnrichmentAdmin)
		adminAPI.PUT("/enrichment/:id", mainHandler.EnrichmentHandlers.UpdateEnrichment)
		adminAPI.POST("/enrichment/:id/unlock", mainHandler.EnrichmentHandlers.UnlockEnrichment)

		// Analysis management
		adminAPI.GET("/analysis/:id", mainHandler.AnalysisHandlers.GetAnalysisAdmin)
		adminAPI.PUT("/analysis/:id", mainHandler.AnalysisHandlers.UpdateAnalysis)
		adminAPI.POST("/analysis/:id/visibility", mainHandler.AnalysisHandlers.ToggleVisibility)
		adminAPI.POST("/analysis/:id/blur", mainHandler.AnalysisHandlers.ToggleBlur)
		adminAPI.POST("/analysis/:id/public", mainHandler.AnalysisHandlers.TogglePublic)
		adminAPI.POST("/analysis/:id/access-code", mainHandler.AnalysisHandlers.GenerateAccessCode)

		// Macroeconomics management (SELIC, IPCA, USD/BRL)
		adminAPI.GET("/macro/latest", macroHandlers.GetLatestSnapshot)
		adminAPI.POST("/macro/refresh", macroHandlers.RefreshAll)
		adminAPI.POST("/macro/refresh/:code", macroHandlers.RefreshIndicator)
		adminAPI.GET("/macro/history/:code", macroHandlers.GetHistory)

		// Company management (admin)
		adminAPI.GET("/companies", mainHandler.CompanyHandlers.ListAllCompanies)
		adminAPI.GET("/companies/:id", mainHandler.CompanyHandlers.GetCompanyAdmin)
		adminAPI.PUT("/companies/:id", mainHandler.CompanyHandlers.UpdateCompany)
		adminAPI.POST("/companies/:id/verify", mainHandler.CompanyHandlers.VerifyCompany)
		adminAPI.POST("/companies/:id/unverify", mainHandler.CompanyHandlers.UnverifyCompany)
		adminAPI.POST("/companies/:id/users", mainHandler.CompanyHandlers.AddUserToCompany)
		adminAPI.DELETE("/companies/:id/users/:userId", mainHandler.CompanyHandlers.RemoveUserFromCompany)
		adminAPI.POST("/companies/:id/transfer-ownership", mainHandler.CompanyHandlers.TransferOwnership)

		// Company-based workflows (re-enrich, re-analyze)
		// Access: admins OR users in company.allowed_users
		adminAPI.POST("/companies/:id/re-enrich", mainHandler.CompanyHandlers.ReEnrichCompany)
		adminAPI.POST("/companies/:id/re-analyze", mainHandler.CompanyHandlers.ReAnalyzeCompany)
		adminAPI.POST("/companies/:id/enrich-and-analyze", mainHandler.CompanyHandlers.EnrichAndAnalyzeCompany)

		// Field verification (protect fields from re-enrichment)
		// Admin only - manage which fields are verified/protected
		adminAPI.GET("/companies/:id/verifications", mainHandler.CompanyHandlers.GetFieldVerifications)
		adminAPI.POST("/companies/:id/verifications", mainHandler.CompanyHandlers.VerifyField)
		adminAPI.DELETE("/companies/:id/verifications/:field_name", mainHandler.CompanyHandlers.UnverifyField)
		adminAPI.POST("/companies/:id/verifications/bulk", mainHandler.CompanyHandlers.VerifyFields)
		adminAPI.POST("/companies/:id/verifications/all", mainHandler.CompanyHandlers.VerifyAllFields)
		adminAPI.DELETE("/companies/:id/verifications/all", mainHandler.CompanyHandlers.UnverifyAllFields)

		// Framework management (admin)
		adminAPI.GET("/frameworks", mainHandler.FrameworkHandlers.AdminList)
		adminAPI.POST("/frameworks", mainHandler.FrameworkHandlers.AdminCreate)
		adminAPI.PUT("/frameworks/:id", mainHandler.FrameworkHandlers.AdminUpdate)
		adminAPI.DELETE("/frameworks/:id", mainHandler.FrameworkHandlers.AdminDeactivate)
	}

	return router
}
