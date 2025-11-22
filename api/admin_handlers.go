package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
func (h *Handler) UpdateSubmissionStatus(c *gin.Context) {
	submissionID := c.Param("id")

	var req UpdateSubmissionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	// Parse UUID
	id, err := uuid.Parse(submissionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid submission ID format"})
		return
	}

	// Validate status (matches frontend SubmissionStatus)
	validStatuses := map[submission.Status]bool{
		submission.StatusPending:          true,
		submission.StatusProcessing:       true,
		submission.StatusEnriching:        true,
		submission.StatusEnriched:         true,
		submission.StatusAnalyzing:        true,
		submission.StatusAnalyzed:         true,
		submission.StatusReadyForReview:   true,
		submission.StatusGeneratingReport: true,
		submission.StatusCompleted:        true,
		submission.StatusEnrichmentFailed: true,
		submission.StatusAnalysisFailed:   true,
		submission.StatusReportFailed:     true,
		submission.StatusFailed:           true,
	}

	newStatus := submission.Status(req.Status)
	if !validStatuses[newStatus] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid status",
			Message: fmt.Sprintf("Status '%s' is not valid", req.Status),
		})
		return
	}

	// Update status
	if err := h.submissionSvc.UpdateStatus(c.Request.Context(), id, newStatus); err != nil {
		h.logger.Error().Err(err).Str("submission_id", submissionID).Msg("Failed to update submission status")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Update failed",
			Message: "Failed to update submission status",
		})
		return
	}

	// Fetch updated submission
	updatedSub, err := h.submissionSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("submission_id", submissionID).Msg("Failed to fetch updated submission")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Update succeeded but fetch failed",
			Message: "Status was updated but could not retrieve updated submission",
		})
		return
	}

	// Frontend expects wrapped response: { submission: {...} }
	resp := buildSubmissionDetailResponse(updatedSub, h, c.Request.Context(), id, submissionID)
	c.JSON(http.StatusOK, gin.H{
		"submission": resp,
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
	sub, err := h.submissionSvc.GetByID(c.Request.Context(), subUUID)
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

	// Update status to analyzing with error checking
	if err := h.submissionSvc.UpdateStatus(c.Request.Context(), sub.ID, submission.StatusAnalyzing); err != nil {
		h.logger.Error().Err(err).Str("submission_id", submissionID).Msg("Failed to update status before analysis retry")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Status update failed",
			Message: "Could not update submission status",
		})
		return
	}

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
