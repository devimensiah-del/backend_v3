package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// SetupRouter configures all API routes
// Now accepts allowedOrigins for CORS configuration
func SetupRouter(handler *Handler, logger zerolog.Logger, jwtSecret string, allowedOrigins string, isProd bool, db *sqlx.DB) *gin.Engine {
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

	// Health check endpoint (will bypass CORS)
	router.GET("/health", handler.HealthCheck)

	// Public API routes (v1)
	publicAPI := router.Group("/api/v1")
	{
		publicAPI.POST("/submit", handler.CreateSubmission)
		// Frontend expects /submissions endpoint
		publicAPI.POST("/submissions", handler.CreateSubmission)
	}

	// Public Auth routes (no auth required)
	publicAuthAPI := router.Group("/api/v1/auth")
	{
		publicAuthAPI.POST("/login", handler.Login)
		publicAuthAPI.POST("/signup", handler.Signup)
		publicAuthAPI.POST("/forgot-password", handler.ForgotPassword)
		publicAuthAPI.POST("/reset-password", handler.ResetPassword)
	}

	// Protected Auth routes (authentication required)
	authAPI := router.Group("/api/v1/auth")
	authAPI.Use(AuthMiddleware(jwtSecret, db))
	{
		authAPI.GET("/me", handler.GetCurrentUser)
		authAPI.POST("/logout", handler.Logout)
		authAPI.PUT("/update-password", handler.UpdatePassword)
	}

	// User profile alias (frontend expects /user/profile)
	userAPI := router.Group("/api/v1/user")
	userAPI.Use(AuthMiddleware(jwtSecret, db))
	{
		userAPI.GET("/profile", handler.GetCurrentUser)
		userAPI.PUT("/profile", handler.UpdateUserProfile)
	}

	// Protected User Routes (v1)
	protectedAPI := router.Group("/api/v1")
	protectedAPI.Use(AuthMiddleware(jwtSecret, db))
	{
		// List user's own submissions
		protectedAPI.GET("/submissions", handler.ListUserSubmissions)
		protectedAPI.GET("/submissions/:id", handler.GetSubmission)
		protectedAPI.GET("/submissions/:id/enrichment", handler.GetEnrichment)
		protectedAPI.GET("/submissions/:id/report/preview", handler.PreviewReport)
		protectedAPI.POST("/submissions/:id/report/publish", handler.PublishReport)
	}

	// Admin API routes (v1)
	adminAPI := router.Group("/api/v1/admin")
	adminAPI.Use(AuthMiddleware(jwtSecret, db))
	adminAPI.Use(AdminAuthMiddleware())
	{
		adminAPI.GET("/submissions", handler.ListSubmissions)
		adminAPI.GET("/submissions/:id", handler.GetSubmissionAdmin)
		adminAPI.PUT("/submissions/:id/status", handler.UpdateSubmissionStatus)
		adminAPI.POST("/submissions/:id/retry-enrichment", handler.RetryEnrichment)
		adminAPI.POST("/submissions/:id/retry-analysis", handler.RetryAnalysis)
		adminAPI.GET("/analytics", handler.GetAnalytics)
	}

	return router
}
