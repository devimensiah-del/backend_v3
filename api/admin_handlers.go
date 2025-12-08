package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"backend_v3/domain/challenge"
	"backend_v3/domain/company"
	"backend_v3/domain/enrichment" // Added for RetryAnalysis
	"backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// Handlers holds all service dependencies and configuration for admin handlers
type AdminHandlers struct {
	SubmissionService         *submission.Service
	EnrichmentService         *enrichment.Service // For RetryAnalysis
	CompanyService            *company.Service    // For checking company status
	ChallengeService          *challenge.Service  // For getting/creating challenges for analysis
	AsynqClient               *asynq.Client
	Logger                    zerolog.Logger
	SubmissionResponseBuilder *SubmissionResponseBuilder // New field
	db                        *sqlx.DB                   // For GetMetrics queries
}

// NewAdminHandlers creates a new admin handler set with all dependencies
func NewAdminHandlers(
	submissionSvc *submission.Service,
	enrichmentSvc *enrichment.Service,
	companySvc *company.Service,
	challengeSvc *challenge.Service,
	asynqClient *asynq.Client,
	logger zerolog.Logger,
	submissionResponseBuilder *SubmissionResponseBuilder, // New parameter
	db *sqlx.DB, // For GetMetrics
) *AdminHandlers {
	return &AdminHandlers{
		SubmissionService:         submissionSvc,
		EnrichmentService:         enrichmentSvc,
		CompanyService:            companySvc,
		ChallengeService:          challengeSvc,
		AsynqClient:               asynqClient,
		Logger:                    logger,
		SubmissionResponseBuilder: submissionResponseBuilder,
		db:                        db,
	}
}

// --- ADMIN ENDPOINTS ---

// ListSubmissions handles GET /api/v1/admin/submissions
func (h *AdminHandlers) ListSubmissions(c *gin.Context) {
	// Parse query parameters
	page := 1
	limit := 20
	// NOTE: status filter removed - submissions no longer have status
	email := c.Query("email")

	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Build list options
	opts := &submission.ListOptions{
		Limit:   limit,
		Offset:  offset,
		OrderBy: "created_at",
		Order:   "DESC",
	}

	// Add filters if provided
	// NOTE: Status filter removed - submissions no longer have status field
	// Status is now tracked on related entities (Company.enrichment_status, Analysis.status)
	if email != "" {
		opts.Email = &email
	}

	// Query submissions
	submissions, total, err := h.SubmissionService.List(c.Request.Context(), opts)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to list submissions")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve submissions",
			Message: err.Error(),
		})
		return
	}

	// Transform submissions to include relationship IDs (enrichmentId, analysisId, reportId)
	// This is required for frontend filtering (e.g., analysis page filters by analysisId)
	detailedSubmissions := make([]SubmissionDetailResponse, len(submissions))
	for i, sub := range submissions {
		detailedSubmissions[i] = h.SubmissionResponseBuilder.buildSubmissionDetailResponse(sub, c.Request.Context(), sub.ID.String())
	}

	// Calculate total pages
	totalPages := (total + limit - 1) / limit // Ceiling division
	if totalPages < 1 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, SubmissionListResponse{
		Data:       detailedSubmissions,
		Page:       page,
		PageSize:   limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetSubmissionAdmin handles GET /api/v1/admin/submissions/:id
func (h *AdminHandlers) GetSubmissionAdmin(c *gin.Context) {
	// Defensive check: verify admin role (middleware should have verified, but double-check)
	role, exists := c.Get("userRole")
	if !exists || (role != "admin" && role != "super_admin" && role != "service_role") {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Admin access required"})
		return
	}

	submissionID := c.Param("id")

	// Parse and validate UUID
	subUUID, err := parseUUID(submissionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid submission ID format"})
		return
	}

	// Get submission (admin can access any submission)
	sub, err := h.SubmissionService.GetByID(c.Request.Context(), subUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Submission not found"})
		return
	}

	// Build detailed response with all fields
	resp := h.SubmissionResponseBuilder.buildSubmissionDetailResponse(sub, c.Request.Context(), submissionID)

	c.JSON(http.StatusOK, gin.H{"submission": resp})
}

// NOTE: RetryEnrichment endpoint removed - enrichment is now automatic at company creation

// RetryAnalysis handles POST /api/v1/admin/submissions/:id/retry-analysis
func (h *AdminHandlers) RetryAnalysis(c *gin.Context) {
	submissionID := c.Param("id")

	// Parse and validate UUID
	subUUID, err := parseUUID(submissionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid submission ID format"})
		return
	}

	// Validate submission exists
	_, err = h.SubmissionService.GetByID(c.Request.Context(), subUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Submission not found"})
		return
	}

	// Get company linked to submission (required for analysis)
	comp, err := h.CompanyService.GetBySubmissionID(c.Request.Context(), subUUID)
	if err != nil || comp == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Company required",
			Message: "Cannot run analysis without company data. Create company first.",
		})
		return
	}

	// Get existing challenges for this company
	// For retry, we use the most recent challenge
	challenges, err := h.ChallengeService.ListByCompany(c.Request.Context(), comp.ID)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", comp.ID.String()).Msg("Failed to list challenges")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Challenge lookup failed", Message: err.Error()})
		return
	}
	if len(challenges) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Challenge required",
			Message: "No challenges found for this company. Create a challenge first or use re-analyze with a new challenge.",
		})
		return
	}
	// Use the most recent challenge (first in the list, ordered by created_at DESC)
	challengeID := challenges[0].ID

	// Create analysis job payload with challenge_id
	payload := map[string]string{
		"submission_id": submissionID,
		"company_id":    comp.ID.String(),
		"challenge_id":  challengeID.String(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to marshal analysis job payload")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Enqueue failed", Message: err.Error()})
		return
	}

	// Enqueue analysis job
	task := asynq.NewTask("analysis_job", payloadBytes)
	_, err = h.AsynqClient.Enqueue(task)
	if err != nil {
		h.Logger.Error().Err(err).Str("submission_id", submissionID).Msg("Failed to enqueue analysis job")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Enqueue failed",
			Message: err.Error(),
		})
		return
	}

	h.Logger.Info().
		Str("submission_id", submissionID).
		Str("challenge_id", challengeID.String()).
		Msg("Analysis retry enqueued")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Analysis retry enqueued successfully",
		Data: map[string]interface{}{
			"id":           submissionID,
			"challenge_id": challengeID.String(),
		},
	})
}

// NOTE: GetAnalytics endpoint removed - analytics feature needs to be rebuilt with new workflow

// ========================================================================
// SYSTEM METRICS (moved from health_handlers.go)
// ========================================================================

// SystemMetrics represents comprehensive system metrics (aligned with frontend)
type SystemMetrics struct {
	SystemHealth          string         `json:"system_health"`
	Analyses24h           int            `json:"analyses_24h"`
	EnrichmentSuccessRate float64        `json:"enrichment_success_rate"`
	AnalysisSuccessRate   float64        `json:"analysis_success_rate"`
	AvgAnalysisTimeMS     float64        `json:"avg_analysis_time_ms"`
	TotalLLMCost24h       float64        `json:"total_llm_cost_24h"`
	TotalTokens24h        int64          `json:"total_tokens_24h"`
	RecentErrors          []MetricsError `json:"recent_errors"`
}

// MetricsError represents an error in the metrics response
type MetricsError struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// GetMetrics handles GET /api/v1/admin/metrics with system-wide statistics
func (h *AdminHandlers) GetMetrics(c *gin.Context) {
	ctx := c.Request.Context()
	since := time.Now().Add(-24 * time.Hour)

	metrics := SystemMetrics{
		SystemHealth: "healthy", // Default to healthy
		RecentErrors: []MetricsError{},
	}

	// Track if any queries fail to determine system health
	hasErrors := false

	// Query analyses count (not submissions - aligned with frontend)
	var analysisCount int
	err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM analyses WHERE created_at > $1
	`, since).Scan(&analysisCount)
	if err == nil {
		metrics.Analyses24h = analysisCount
	} else {
		hasErrors = true
	}

	// Query enrichment success rate (company-level now)
	var enrichTotal, enrichSuccess int
	err = h.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE enrichment_status = 'completed')
		FROM companies WHERE created_at > $1
	`, since).Scan(&enrichTotal, &enrichSuccess)
	if err == nil {
		metrics.EnrichmentSuccessRate = safePercent(enrichSuccess, enrichTotal)
	} else {
		hasErrors = true
	}

	// Query analysis success rate
	var analysisTotal, analysisSuccess int
	err = h.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status IN ('completed', 'approved', 'sent'))
		FROM analyses WHERE created_at > $1
	`, since).Scan(&analysisTotal, &analysisSuccess)
	if err == nil {
		metrics.AnalysisSuccessRate = safePercent(analysisSuccess, analysisTotal)
	} else {
		hasErrors = true
	}

	// Query avg analysis time (in milliseconds, not seconds)
	var avgTimeMS float64
	err = h.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(processing_time_ms), 0)
		FROM analyses
		WHERE status IN ('completed', 'approved', 'sent') AND created_at > $1
	`, since).Scan(&avgTimeMS)
	if err == nil {
		metrics.AvgAnalysisTimeMS = avgTimeMS
	}

	// Query LLM usage (if table exists)
	var totalCost float64
	var totalTokens int64
	err = h.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(cost_usd), 0),
			COALESCE(SUM(total_tokens), 0)
		FROM llm_usage_logs WHERE created_at > $1
	`, since).Scan(&totalCost, &totalTokens)
	if err == nil {
		metrics.TotalLLMCost24h = totalCost
		metrics.TotalTokens24h = totalTokens
	}

	// Query recent errors with structured format
	rows, err := h.db.QueryContext(ctx, `
		SELECT id, type, message, timestamp FROM (
			SELECT
				id::text,
				'enrichment' as type,
				enrichment_error as message,
				created_at as timestamp
			FROM companies
			WHERE enrichment_status = 'failed' AND created_at > $1 AND enrichment_error IS NOT NULL
			UNION ALL
			SELECT
				id::text,
				'analysis' as type,
				error_message as message,
				created_at as timestamp
			FROM analyses
			WHERE status = 'failed' AND created_at > $1 AND error_message IS NOT NULL
		) e ORDER BY timestamp DESC LIMIT 10
	`, since)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var e MetricsError
			var ts time.Time
			if rows.Scan(&e.ID, &e.Type, &e.Message, &ts) == nil && e.Message != "" {
				e.Timestamp = ts.Format(time.RFC3339)
				metrics.RecentErrors = append(metrics.RecentErrors, e)
			}
		}
	}

	// Determine system health based on success rates and errors
	if hasErrors {
		metrics.SystemHealth = "degraded"
	} else if len(metrics.RecentErrors) > 5 {
		metrics.SystemHealth = "degraded"
	} else if metrics.EnrichmentSuccessRate < 50 || metrics.AnalysisSuccessRate < 50 {
		metrics.SystemHealth = "degraded"
	}

	h.Logger.Info().
		Int("analyses", metrics.Analyses24h).
		Float64("cost_usd", metrics.TotalLLMCost24h).
		Str("health", metrics.SystemHealth).
		Msg("Metrics retrieved")

	c.JSON(http.StatusOK, metrics)
}

// safePercent calculates percentage, returning 100 if denominator is 0
func safePercent(num, denom int) float64 {
	if denom == 0 {
		return 100.0 // No attempts = 100% success (vacuous truth)
	}
	return float64(num) / float64(denom) * 100
}
