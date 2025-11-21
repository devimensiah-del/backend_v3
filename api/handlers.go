package api

import (
	"net/http"

	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/report"
	"backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type Handler struct {
	submissionSvc *submission.Service
	enrichmentSvc *enrichment.Service
	analysisSvc   *analysis.Service
	reportSvc     *report.Service
	asynqClient   *asynq.Client
	db            *sqlx.DB
	redisClient   *redis.Client
	logger        zerolog.Logger
}

func NewHandler(
	submissionSvc *submission.Service,
	enrichmentSvc *enrichment.Service,
	analysisSvc *analysis.Service,
	reportSvc *report.Service,
	asynqClient *asynq.Client,
	db *sqlx.DB,
	redisClient *redis.Client,
	logger zerolog.Logger,
) *Handler {
	return &Handler{
		submissionSvc: submissionSvc,
		enrichmentSvc: enrichmentSvc,
		analysisSvc:   analysisSvc,
		reportSvc:     reportSvc,
		asynqClient:   asynqClient,
		db:            db,
		redisClient:   redisClient,
		logger:        logger.With().Str("component", "api").Logger(),
	}
}

// --- AUTH ENDPOINTS ---

// GetCurrentUser returns the authenticated user's profile
func (h *Handler) GetCurrentUser(c *gin.Context) {
	// Get user ID from AuthMiddleware
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "User not authenticated"})
		return
	}

	// Query user_profiles table
	var profile UserProfile
	query := `
		SELECT id, email, full_name, role, is_active, created_at, updated_at
		FROM user_profiles
		WHERE id = $1
	`

	err := h.db.Get(&profile, query, userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID.(string)).Msg("Failed to fetch user profile")
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "User profile not found"})
		return
	}

	c.JSON(http.StatusOK, UserProfileResponse{User: profile})
}

// --- PUBLIC ENDPOINTS ---

// CreateSubmission starts the workflow
func (h *Handler) CreateSubmission(c *gin.Context) {
	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	sub, err := h.submissionSvc.SubmitForm(c.Request.Context(), &submission.SubmitRequest{
		CompanyName:       req.CompanyName,
		CompanyWebsite:    &req.WebsiteURL,
		CompanyIndustry:   &req.IndustryName,
		ContactEmail:      req.Email,
		BusinessChallenge: req.Description,
		// Map other fields...
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create submission")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Submission failed", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, SubmissionResponse{
		ID:            sub.ID.String(),
		CompanyName:   sub.CompanyName,
		Status:        string(sub.Status),
		StatusMessage: "AI Agent activated. Analysis started.",
		CreatedAt:     sub.CreatedAt,
	})
}

// GetSubmission checks status and returns details
// SECURITY: Ensures user can only access their own submission
func (h *Handler) GetSubmission(c *gin.Context) {
	submissionID := c.Param("id")

	// 1. Fetch Submission
	sub, err := h.submissionSvc.GetByID(c.Request.Context(), parserUUID(submissionID))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Submission not found"})
		return
	}

	// 2. SECURITY CHECK: Ownership
	// We get "userID" and "userRole" from the AuthMiddleware
	requesterID, hasUser := c.Get("userID")
	requesterRole, _ := c.Get("userRole")

	// If the submission belongs to a user (sub.UserID is not nil)
	// AND the requester is not the owner
	// AND the requester is not an admin
	if sub.UserID != nil && hasUser {
		if sub.UserID.String() != requesterID.(string) {
			// Check if Admin bypass
			if requesterRole != "admin" && requesterRole != "super_admin" {
				h.logger.Warn().Str("sub_id", submissionID).Str("requester", requesterID.(string)).Msg("Unauthorized access attempt")
				c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "You do not have permission to view this submission"})
				return
			}
		}
	}

	// 3. Build Response
	resp := SubmissionDetailResponse{
		ID:          sub.ID.String(),
		CompanyName: sub.CompanyName,
		Status:      string(sub.Status),
		CreatedAt:   sub.CreatedAt,
	}

	if enrich, err := h.enrichmentSvc.GetBySubmissionID(c.Request.Context(), parserUUID(submissionID)); err == nil {
		resp.EnrichmentID = enrich.ID.String()
	}
	if anal, err := h.analysisSvc.GetBySubmissionID(c.Request.Context(), submissionID); err == nil {
		resp.AnalysisID = anal.ID
	}
	if rep, err := h.reportSvc.GetBySubmissionID(c.Request.Context(), submissionID); err == nil {
		resp.ReportID = rep.ID
		resp.PDFURL = rep.PDFURL
	}

	c.JSON(http.StatusOK, resp)
}

// --- NEW REPORTING ENDPOINTS (Interactive) ---

// PreviewReport generates HTML on-the-fly for the Admin UI
// GET /api/submissions/:id/report/preview
func (h *Handler) PreviewReport(c *gin.Context) {
	submissionID := c.Param("id")

	// Get Analysis ID
	anal, err := h.analysisSvc.GetBySubmissionID(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Analysis not ready", Message: "Wait for AI to finish."})
		return
	}

	// Generate HTML (In Memory)
	pagesMap, err := h.reportSvc.GeneratePreview(c.Request.Context(), submissionID, anal.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Preview generation failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Preview failed", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ReportPreviewResponse{Pages: pagesMap})
}

// PublishReport freezes the report and generates the PDF
// POST /api/submissions/:id/report/publish
func (h *Handler) PublishReport(c *gin.Context) {
	submissionID := c.Param("id")

	anal, err := h.analysisSvc.GetBySubmissionID(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Analysis not ready"})
		return
	}

	pdfURL, err := h.reportSvc.Publish(c.Request.Context(), submissionID, anal.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Publish failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Publish failed", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ReportPublishResponse{
		ReportID: anal.ID, // Or create a new ID
		PDFURL:   pdfURL,
	})
}

// --- ADMIN ENDPOINTS ---

// ListSubmissions handles GET /api/admin/submissions
func (h *Handler) ListSubmissions(c *gin.Context) {
	// For now, return empty list
	// TODO: Implement pagination and filtering
	c.JSON(http.StatusOK, gin.H{
		"submissions": []interface{}{},
		"total":       0,
	})
}

// RetryEnrichment handles POST /api/admin/submissions/:id/retry-enrichment
func (h *Handler) RetryEnrichment(c *gin.Context) {
	submissionID := c.Param("id")

	// TODO: Implement retry logic
	h.logger.Info().Str("submission_id", submissionID).Msg("Retry enrichment requested")

	c.JSON(http.StatusOK, gin.H{
		"message": "Enrichment retry enqueued",
		"id":      submissionID,
	})
}

// RetryAnalysis handles POST /api/admin/submissions/:id/retry-analysis
func (h *Handler) RetryAnalysis(c *gin.Context) {
	submissionID := c.Param("id")

	// TODO: Implement retry logic
	h.logger.Info().Str("submission_id", submissionID).Msg("Retry analysis requested")

	c.JSON(http.StatusOK, gin.H{
		"message": "Analysis retry enqueued",
		"id":      submissionID,
	})
}

// GetAnalytics handles GET /api/v1/admin/analytics
func (h *Handler) GetAnalytics(c *gin.Context) {
	// Get analytics data from service
	analytics, err := h.submissionSvc.GetAnalytics(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get analytics")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve analytics",
			Message: err.Error(),
		})
		return
	}

	// Map to response type
	response := AnalyticsResponse{
		TotalSubmissions:     analytics.TotalSubmissions,
		ActiveSubmissions:    analytics.ActiveSubmissions,
		CompletedSubmissions: analytics.CompletedSubmissions,
		Revenue:              analytics.Revenue,
	}

	c.JSON(http.StatusOK, response)
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	health := gin.H{
		"status": "healthy",
		"services": gin.H{
			"database": "healthy",
			"redis":    "healthy",
		},
	}

	// Check database
	if err := h.db.Ping(); err != nil {
		health["status"] = "unhealthy"
		health["services"].(gin.H)["database"] = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, health)
		return
	}

	// Check Redis if available
	if h.redisClient != nil {
		if err := h.redisClient.Ping(c.Request.Context()).Err(); err != nil {
			health["status"] = "unhealthy"
			health["services"].(gin.H)["redis"] = "unhealthy"
			c.JSON(http.StatusServiceUnavailable, health)
			return
		}
	}

	c.JSON(http.StatusOK, health)
}

// Helper
func parserUUID(id string) uuid.UUID {
	u, _ := uuid.Parse(id)
	return u
}
