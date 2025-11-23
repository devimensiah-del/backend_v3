package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetEnrichment handles GET /api/v1/submissions/:id/enrichment
func (h *Handler) GetEnrichment(c *gin.Context) {
	submissionID := c.Param("id")

	// Parse and validate UUID
	subUUID, err := parseUUID(submissionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid ID",
			Message: "Invalid submission ID format",
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Get submission to verify ownership (unless admin)
	submission, err := h.submissionSvc.GetByID(c.Request.Context(), subUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Submission not found",
		})
		return
	}

	// Check if user owns this submission (admins can access any)
	role, _ := c.Get("userRole")
	userRole := ""
	if r, ok := role.(string); ok {
		userRole = r
	}

	// Convert userID string to UUID for comparison
	userIDStr := userID.(string)
	currentUserUUID, err := parseUUID(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal error",
			Message: "Invalid user ID format",
		})
		return
	}

	if submission.UserID != nil && *submission.UserID != currentUserUUID {
		if userRole != "admin" && userRole != "super_admin" {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "Forbidden",
				Message: "You don't have permission to access this enrichment",
			})
			return
		}
	}

	// Get enrichment data
	enrichment, err := h.enrichmentSvc.GetBySubmissionID(c.Request.Context(), subUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Enrichment data not found for this submission",
		})
		return
	}

	// Transform to DTO to fix field name mismatch (enrichedData -> data)
	response := EnrichmentResponse{
		ID:           enrichment.ID.String(),
		SubmissionID: enrichment.SubmissionID.String(),
		Status:       string(enrichment.Status),
		Progress:     enrichment.Progress,
		CurrentStep:  enrichment.CurrentStep,
		Data:         enrichment.EnrichedData, // Maps enrichedData -> data
		CreatedAt:    enrichment.CreatedAt,
		UpdatedAt:    enrichment.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"enrichment": response,
	})
}

// UpdateEnrichment handles PUT /api/v1/admin/enrichment/:id
// Admin can edit enrichment fields BEFORE approval (status remains "completed")
// Cannot edit after status is "approved"
func (h *Handler) UpdateEnrichment(c *gin.Context) {
	enrichmentID := c.Param("id")

	// Parse and validate UUID
	enrichUUID, err := parseUUID(enrichmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid ID",
			Message: "Invalid enrichment ID format",
		})
		return
	}

	// Parse request body (partial update - any fields from EnrichedData)
	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// Update enrichment fields via service
	updatedEnrichment, err := h.enrichmentSvc.UpdateFields(c.Request.Context(), enrichUUID, updateData)
	if err != nil {
		h.logger.Error().Err(err).Str("enrichment_id", enrichmentID).Msg("Failed to update enrichment")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Update failed",
			Message: "Failed to update enrichment fields",
		})
		return
	}

	// Transform to DTO
	response := EnrichmentResponse{
		ID:           updatedEnrichment.ID.String(),
		SubmissionID: updatedEnrichment.SubmissionID.String(),
		Status:       string(updatedEnrichment.Status),
		Progress:     updatedEnrichment.Progress,
		CurrentStep:  updatedEnrichment.CurrentStep,
		Data:         updatedEnrichment.EnrichedData,
		CreatedAt:    updatedEnrichment.CreatedAt,
		UpdatedAt:    updatedEnrichment.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"enrichment": response,
	})
}

// ApproveEnrichment handles POST /api/v1/admin/enrichment/:id/approve
// Changes status from "completed" → "approved" and triggers analysis creation
func (h *Handler) ApproveEnrichment(c *gin.Context) {
	enrichmentID := c.Param("id")

	// Parse and validate UUID
	enrichUUID, err := parseUUID(enrichmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid ID",
			Message: "Invalid enrichment ID format",
		})
		return
	}

	// Approve enrichment (service handles status update AND job enqueueing)
	err = h.enrichmentSvc.Approve(c.Request.Context(), enrichUUID)
	if err != nil {
		h.logger.Error().Err(err).Str("enrichment_id", enrichmentID).Msg("Failed to approve enrichment")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Approval failed",
			Message: err.Error(),
		})
		return
	}

	// Fetch updated enrichment
	updatedEnrichment, err := h.enrichmentSvc.GetByID(c.Request.Context(), enrichUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Fetch failed",
			Message: "Approved but failed to fetch updated enrichment",
		})
		return
	}

	// Transform to DTO
	enrichmentResponse := EnrichmentResponse{
		ID:           updatedEnrichment.ID.String(),
		SubmissionID: updatedEnrichment.SubmissionID.String(),
		Status:       string(updatedEnrichment.Status),
		Progress:     updatedEnrichment.Progress,
		CurrentStep:  updatedEnrichment.CurrentStep,
		Data:         updatedEnrichment.EnrichedData,
		CreatedAt:    updatedEnrichment.CreatedAt,
		UpdatedAt:    updatedEnrichment.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"enrichment": enrichmentResponse,
		"message":    "Enrichment approved, analysis job enqueued",
	})
}

// GetEnrichmentAdmin handles GET /api/v1/admin/enrichment/:id
// Admin can fetch enrichment directly by enrichment ID (not just by submission ID)
func (h *Handler) GetEnrichmentAdmin(c *gin.Context) {
	enrichmentID := c.Param("id")

	// Parse and validate UUID
	enrichUUID, err := parseUUID(enrichmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid ID",
			Message: "Invalid enrichment ID format",
		})
		return
	}

	// Get enrichment by ID (admin access, no ownership check needed)
	enrichment, err := h.enrichmentSvc.GetByID(c.Request.Context(), enrichUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Enrichment not found",
		})
		return
	}

	// Transform to DTO
	response := EnrichmentResponse{
		ID:           enrichment.ID.String(),
		SubmissionID: enrichment.SubmissionID.String(),
		Status:       string(enrichment.Status),
		Progress:     enrichment.Progress,
		CurrentStep:  enrichment.CurrentStep,
		Data:         enrichment.EnrichedData,
		CreatedAt:    enrichment.CreatedAt,
		UpdatedAt:    enrichment.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"enrichment": response,
	})
}
