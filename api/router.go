package api

import (
	domainanalysis "backend_v3/domain/analysis"
	domainanalysisbysteps "backend_v3/domain/analysisbysteps"
	domainchallenge "backend_v3/domain/challenge"
	domaincompany "backend_v3/domain/company"
	domainenrichment "backend_v3/domain/enrichment"
	domainframework "backend_v3/domain/framework"
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
	companySvc *domaincompany.Service,       // Company service for re-enrich/re-analyze workflows
	challengeSvc *domainchallenge.Service,   // Challenge service for challenge management
	frameworkSvc *domainframework.Service,   // Framework service for V2 framework management (nil if disabled)
) *gin.Engine {
	if isProd {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware (CORS now includes health check bypass)
	router.Use(CORSMiddleware(allowedOrigins, logger))
	router.Use(RequestIDMiddleware())
	router.Use(LoggingMiddleware(logger))
	router.Use(RecoveryMiddleware(logger))
	router.Use(RateLimitMiddleware(100))

	// --- Instantiate Specialized Handlers ---
	// Instantiate the SubmissionResponseBuilder with all required services
	submissionResponseBuilder := NewSubmissionResponseBuilder(
		analysisSvc,
		companySvc,
		challengeSvc,
	)

	// Instantiate handlers
	adminHandlers := NewAdminHandlers(
		submissionSvc,
		enrichmentSvc,
		companySvc,
		challengeSvc,
		asynqClient,
		logger,
		submissionResponseBuilder, // Pass the builder
		db,                        // For GetMetrics
	)
	analysisHandlers := NewAnalysisHandlers(
		submissionSvc,
		analysisSvc,
		companySvc,
		challengeSvc,
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
	submissionHandlers := NewSubmissionHandlers(
		submissionSvc,
		logger,
		submissionResponseBuilder, // Pass the builder
		authHandlers,              // For auto-creating users on submission
	)
	userHandlers := NewUserHandlers(
		db,
		logger,
	)
	companyHandlers := NewCompanyHandlers(
		companySvc,
		submissionSvc,
		enrichmentSvc,
		challengeSvc,
		analysisSvc,
		asynqClient,
		logger,
	)

	// Framework handlers (V2) - only functional if frameworkSvc is not nil
	var frameworkHandlers *FrameworkHandlers
	if frameworkSvc != nil {
		// Create dependency-aware services
		repo := frameworkSvc.GetRepo()
		depService := domainframework.NewDependencyService(repo, logger)
		promptBuilder := domainframework.NewPromptBuilder(repo, depService, logger)
		execService := domainframework.NewExecutionService(repo, depService, promptBuilder, logger)

		frameworkHandlers = NewFrameworkHandlers(frameworkSvc, depService, execService, companySvc, asynqClient, logger)
	}

	// Create the main API Handler instance
	// This Handler now composes all specialized handlers
	mainHandler := NewHandler(
		adminHandlers,
		analysisHandlers,
		authHandlers,
		companyHandlers,
		frameworkHandlers,
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
	// Queue diagnostic (temporary - for debugging)
	router.GET("/queue-diagnostic", mainHandler.QueueDiagnostic)

	// Public API routes (v1)
	publicAPI := router.Group("/api/v1")
	{
		publicAPI.POST("/submit", mainHandler.SubmissionHandlers.CreateSubmission)
		// Frontend expects /submissions endpoint
		publicAPI.POST("/submissions", mainHandler.SubmissionHandlers.CreateSubmission)
		// Public report access via access code
		// Uses OptionalAuthMiddleware to allow admin preview (bypasses visibility check)
		publicAPI.GET("/public/report/:code", OptionalAuthMiddleware(jwtSecret, db), mainHandler.AnalysisHandlers.GetPublicReport)
	}

	// Public Auth routes (no auth required)
	// Global rate limiting already applies (100 req/min), which is sufficient for auth endpoints
	publicAuthAPI := router.Group("/api/v1/auth")
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
		authAPI.POST("/set-password", mainHandler.AuthHandlers.SetPassword) // For users auto-created without password
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
		// NOTE: GET /submissions/:id/analysis removed in v2_013 - use GET /analyses/:id instead

		// Company routes - user's companies (owner or in allowed_users)
		protectedAPI.POST("/companies", mainHandler.CompanyHandlers.CreateCompany)
		protectedAPI.GET("/companies", mainHandler.CompanyHandlers.GetMyCompanies)
		protectedAPI.GET("/companies/:id", mainHandler.CompanyHandlers.GetCompany)
		protectedAPI.GET("/companies/:id/challenges", mainHandler.CompanyHandlers.GetCompanyChallenges)
		// NOTE: PUT /companies/:id removed - edit company in Supabase directly

		// Enrichment step triggers (human-validated enrichment pipeline)
		protectedAPI.GET("/companies/:id/enrichment-status", mainHandler.CompanyHandlers.GetEnrichmentStatus)
		protectedAPI.POST("/companies/:id/enrich/step1/retry", mainHandler.CompanyHandlers.RetryStep1)
		protectedAPI.POST("/companies/:id/enrich/step2", mainHandler.CompanyHandlers.TriggerStep2)
		protectedAPI.POST("/companies/:id/enrich/step2/retry", mainHandler.CompanyHandlers.RetryStep2)
		protectedAPI.POST("/companies/:id/enrich/step3", mainHandler.CompanyHandlers.TriggerStep3)
		protectedAPI.POST("/companies/:id/enrich/step3/retry", mainHandler.CompanyHandlers.RetryStep3)

		// Challenge routes - create challenges for existing companies
		protectedAPI.POST("/challenges", mainHandler.CompanyHandlers.CreateChallenge)

		// User-facing company update (with ownership check + field whitelist)
		protectedAPI.PUT("/companies/:id", mainHandler.CompanyHandlers.UpdateCompanyUser)

		// User-facing analysis triggering (reuse admin handlers - they already do checkCompanyAccess)
		protectedAPI.POST("/companies/:id/re-analyze", mainHandler.CompanyHandlers.ReAnalyzeCompany)
		protectedAPI.POST("/challenges/:id/analyze", mainHandler.CompanyHandlers.AnalyzeChallenge)

		// Analysis routes (user can view their own analyses)
		protectedAPI.GET("/analyses/:id", mainHandler.AnalysisHandlers.GetAnalysisUser)
		protectedAPI.POST("/analyses/:id/access-code", mainHandler.AnalysisHandlers.GenerateAccessCodeUser)

		// Challenge types (public info, but under protected for consistency)
		protectedAPI.GET("/challenges/types", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"categories": domainchallenge.ChallengeCategories,
				"types":      domainchallenge.GetAllChallengeTypes(),
			})
		})

		// Framework order for step-by-step analysis (IAH-2)
		protectedAPI.GET("/frameworks/order", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"frameworks":  domainanalysisbysteps.FrameworkOrder,
				"total_steps": domainanalysisbysteps.TotalSteps(),
			})
		})

		// Framework V2 routes (only available if FRAMEWORKS_V2_ENABLED=true)
		if mainHandler.FrameworkHandlers != nil {
			// Original framework result routes
			protectedAPI.GET("/companies/:id/framework-results", mainHandler.FrameworkHandlers.GetCompanyFrameworkResults)
			protectedAPI.POST("/companies/:id/frameworks/:code/run", mainHandler.FrameworkHandlers.RunFrameworkForCompany)
			protectedAPI.GET("/companies/:id/framework-results/:resultId", mainHandler.FrameworkHandlers.GetFrameworkResultStatus)
			protectedAPI.PUT("/companies/:id/framework-results/:resultId", mainHandler.FrameworkHandlers.UpdateFrameworkResult)

			// 5-Step Analysis & Dependency Management routes
			protectedAPI.GET("/companies/:id/analysis-state", mainHandler.FrameworkHandlers.GetAnalysisState)
			protectedAPI.GET("/companies/:id/stale-frameworks", mainHandler.FrameworkHandlers.GetStaleFrameworks)
			protectedAPI.POST("/companies/:id/acknowledge-stale", mainHandler.FrameworkHandlers.AcknowledgeStale)
			protectedAPI.GET("/companies/:id/frameworks/:code/readiness", mainHandler.FrameworkHandlers.CheckFrameworkReadiness)
			protectedAPI.POST("/companies/:id/frameworks/:code/execute", mainHandler.FrameworkHandlers.ExecuteFramework)

			// Framework dependency info (public for visualization)
			protectedAPI.GET("/frameworks/:code/dependencies", mainHandler.FrameworkHandlers.GetDependencyChain)
		}

		// TODO: IAH-3 - Add analysisbysteps API handlers for human editing
	}

	// Admin API routes (v1)
	adminAPI := router.Group("/api/v1/admin")
	adminAPI.Use(AuthMiddleware(jwtSecret, db)) // AuthMiddleware might need to be adjusted
	adminAPI.Use(AdminAuthMiddleware())         // AdminAuthMiddleware might need to be adjusted
	{
		// Submission management
		adminAPI.GET("/submissions", mainHandler.AdminHandlers.ListSubmissions)
		adminAPI.GET("/submissions/:id", mainHandler.AdminHandlers.GetSubmissionAdmin)
		// NOTE: GET /submissions/:id/analysis removed in v2_013 - use GET /admin/analysis/:id instead
		// REMOVED: PUT /submissions/:id/status - Violates "Single Status Rule"
		// Submissions always have status "received". Status is derived from Enrichment/Analysis.
		// REMOVED: POST /submissions/:id/retry-enrichment - Enrichment is now automatic at company creation
		adminAPI.POST("/submissions/:id/retry-analysis", mainHandler.AdminHandlers.RetryAnalysis)
		// REMOVED: GET /analytics - Needs to be rebuilt with new workflow
		adminAPI.GET("/metrics", mainHandler.AdminHandlers.GetMetrics) // System-wide metrics (LLM costs, success rates)
		adminAPI.GET("/queue-diagnostic", mainHandler.QueueDiagnostic) // Queue status and cleanup (add ?clear=true to clear)

		// Analysis management
		adminAPI.GET("/analysis/:id", mainHandler.AnalysisHandlers.GetAnalysisAdmin)
		adminAPI.PUT("/analysis/:id", mainHandler.AnalysisHandlers.UpdateAnalysis)
		adminAPI.POST("/analysis/:id/visibility", mainHandler.AnalysisHandlers.ToggleVisibility)
		adminAPI.PATCH("/analysis/:id/visibility", mainHandler.AnalysisHandlers.UpdateVisibility)
		adminAPI.POST("/analysis/:id/public", mainHandler.AnalysisHandlers.TogglePublic)
		adminAPI.POST("/analysis/:id/access-code", mainHandler.AnalysisHandlers.GenerateAccessCode)
		// NOTE: POST /analysis/:id/wizard/generate-all removed - wizard package deprecated (IAH-2)

		// Company management (admin)
		adminAPI.GET("/companies", mainHandler.CompanyHandlers.ListAllCompanies)
		adminAPI.GET("/companies/:id", mainHandler.CompanyHandlers.GetCompanyAdmin)
		adminAPI.PUT("/companies/:id", mainHandler.CompanyHandlers.UpdateCompanyAdmin)
		adminAPI.POST("/companies/:id/re-analyze", mainHandler.CompanyHandlers.ReAnalyzeCompany)
		adminAPI.POST("/companies/:id/retry-enrichment", mainHandler.CompanyHandlers.RetryEnrichment)
		adminAPI.POST("/companies/:id/re-enrich", mainHandler.CompanyHandlers.ReEnrichCompany)
		adminAPI.POST("/challenges/:id/analyze", mainHandler.CompanyHandlers.AnalyzeChallenge)

		// CNPJ duplicate management (admin)
		adminAPI.GET("/companies/duplicates", mainHandler.AdminHandlers.ListCNPJDuplicates)
		adminAPI.POST("/companies/:id/merge", mainHandler.AdminHandlers.MergeCompanies)
		// NOTE: Multi-user management removed - manage allowed_users in Supabase directly

		// Framework V2 admin routes (only available if FRAMEWORKS_V2_ENABLED=true)
		if mainHandler.FrameworkHandlers != nil {
			adminAPI.GET("/frameworks", mainHandler.FrameworkHandlers.ListFrameworks)
			adminAPI.GET("/frameworks/execution-plan", mainHandler.FrameworkHandlers.GetExecutionPlan)
			adminAPI.GET("/frameworks/:id", mainHandler.FrameworkHandlers.GetFramework)
			adminAPI.PUT("/frameworks/:id", mainHandler.FrameworkHandlers.UpdateFramework)
		}
	}

	return router
}
