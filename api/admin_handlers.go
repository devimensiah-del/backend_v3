package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
)

// --- ADMIN ENDPOINTS ---

// ListSubmissions handles GET /api/v1/admin/submissions
func (h *Handler) ListSubmissions(c *gin.Context) {
	// Parse query parameters
	page := 1
	limit := 20
	status := c.Query("status")
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
	if status != "" {
		s := submission.Status(status)
		opts.Status = &s
	}
	if email != "" {
		opts.Email = &email
	}

	// Query submissions
	submissions, total, err := h.submissionSvc.ListAll(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list submissions")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve submissions",
			Message: err.Error(),
		})
		return
	}

	// Calculate total pages
	totalPages := (total + limit - 1) / limit // Ceiling division
	if totalPages < 1 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, SubmissionListResponse{
		Data:       submissions,
		Page:       page,
		PageSize:   limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetSubmissionAdmin handles GET /api/v1/admin/submissions/:id
func (h *Handler) GetSubmissionAdmin(c *gin.Context) {
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
	sub, err := h.submissionSvc.GetByID(c.Request.Context(), subUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Submission not found"})
		return
	}

	// Build detailed response with all fields
	resp := buildSubmissionDetailResponse(sub, h, c.Request.Context(), subUUID, submissionID)

	c.JSON(http.StatusOK, resp)
}

// UpdateSubmissionStatus handles PUT /api/v1/admin/submissions/:id/status
// DEPRECATED: Submission status is always "received" and never changes
// Workflow state is tracked in Enrichment and Analysis entities
func (h *Handler) UpdateSubmissionStatus(c *gin.Context) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error:   "Operation not allowed",
		Message: "Submission status is always 'received' and cannot be changed. Workflow state is tracked in enrichment and analysis entities.",
	})
}

// RetryEnrichment handles POST /api/v1/admin/submissions/:id/retry-enrichment
func (h *Handler) RetryEnrichment(c *gin.Context) {
	submissionID := c.Param("id")

	// Validate submission exists
	_, err := h.submissionSvc.GetByID(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Submission not found"})
		return
	}

	// Create enrichment job payload
	payload := map[string]string{
		"submission_id": submissionID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to marshal enrichment job payload")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Enqueue failed", Message: err.Error()})
		return
	}

	// Enqueue enrichment job
	task := asynq.NewTask("enrichment_job", payloadBytes)
	_, err = h.asynqClient.Enqueue(task)
	if err != nil {
		h.logger.Error().Err(err).Str("submission_id", submissionID).Msg("Failed to enqueue enrichment job")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Enqueue failed",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info().Str("submission_id", submissionID).Msg("Enrichment retry enqueued")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Enrichment retry enqueued successfully",
		Data: map[string]interface{}{
			"id": submissionID,
		},
	})
}

// RetryAnalysis handles POST /api/v1/admin/submissions/:id/retry-analysis
func (h *Handler) RetryAnalysis(c *gin.Context) {
	submissionID := c.Param("id")

	// Parse and validate UUID
	subUUID, err := parseUUID(submissionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid submission ID format"})
		return
	}

	// Validate submission exists
	_, err = h.submissionSvc.GetByID(c.Request.Context(), subUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Submission not found"})
		return
	}

	// Get enrichment record
	enrichmentRecord, err := h.enrichmentSvc.GetBySubmissionID(c.Request.Context(), subUUID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Enrichment required",
			Message: "Cannot retry analysis without enrichment data. Run enrichment first.",
		})
		return
	}

	// NOTE: Removed submission status update - submission status always "received"

	// Create analysis job payload
	payload := map[string]string{
		"submission_id": submissionID,
		"enrichment_id": enrichmentRecord.ID.String(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to marshal analysis job payload")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Enqueue failed", Message: err.Error()})
		return
	}

	// Enqueue analysis job
	task := asynq.NewTask("analysis_job", payloadBytes)
	_, err = h.asynqClient.Enqueue(task)
	if err != nil {
		h.logger.Error().Err(err).Str("submission_id", submissionID).Msg("Failed to enqueue analysis job")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Enqueue failed",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info().Str("submission_id", submissionID).Msg("Analysis retry enqueued")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Analysis retry enqueued successfully",
		Data: map[string]interface{}{
			"id": submissionID,
		},
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
