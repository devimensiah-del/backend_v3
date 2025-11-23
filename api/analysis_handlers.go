package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetAnalysis handles GET /api/v1/submissions/:id/analysis
func (h *Handler) GetAnalysis(c *gin.Context) {
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
				Message: "You don't have permission to access this analysis",
			})
			return
		}
	}

	// Get analysis data by submission ID
	analysis, err := h.analysisSvc.GetBySubmissionID(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Analysis not found for this submission",
		})
		return
	}

	// Transform domain model to map[string]interface{} for frontend
	analysisData := make(map[string]interface{})

	// Marshal the entire analysis to JSON then unmarshal to map
	// This ensures all frameworks are included with their proper structure
	analysisBytes, err := json.Marshal(analysis)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal error",
			Message: "Failed to serialize analysis data",
		})
		return
	}

	if err := json.Unmarshal(analysisBytes, &analysisData); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal error",
			Message: "Failed to deserialize analysis data",
		})
		return
	}

	// Build response DTO
	response := AnalysisResponse{
		ID:           analysis.ID,
		SubmissionID: analysis.SubmissionID,
		Status:       analysis.Status,
		Version:      analysis.Version,
		ParentID:     analysis.ParentAnalysisID,
		Analysis:     analysisData,
		CreatedAt:    analysis.CreatedAt,
		UpdatedAt:    analysis.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
	})
}

// UpdateAnalysis handles PUT /api/v1/admin/analysis/:id
// Admin can edit analysis framework fields (status remains unchanged)
func (h *Handler) UpdateAnalysis(c *gin.Context) {
	analysisID := c.Param("id")

	// Parse request body (partial update - any framework fields)
	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// Update analysis fields via service
	updatedAnalysis, err := h.analysisSvc.UpdateFields(c.Request.Context(), analysisID, updateData)
	if err != nil {
		h.logger.Error().Err(err).Str("analysis_id", analysisID).Msg("Failed to update analysis")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Update failed",
			Message: "Failed to update analysis fields",
		})
		return
	}

	// Transform to response
	analysisData := make(map[string]interface{})
	analysisBytes, _ := json.Marshal(updatedAnalysis)
	json.Unmarshal(analysisBytes, &analysisData)

	response := AnalysisResponse{
		ID:           updatedAnalysis.ID,
		SubmissionID: updatedAnalysis.SubmissionID,
		Status:       updatedAnalysis.Status,
		Version:      updatedAnalysis.Version,
		ParentID:     updatedAnalysis.ParentAnalysisID,
		Analysis:     analysisData,
		CreatedAt:    updatedAnalysis.CreatedAt,
		UpdatedAt:    updatedAnalysis.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
	})
}

// CreateAnalysisVersion handles POST /api/v1/admin/analysis/:id/version
// Creates a new version of the analysis with optional edits
func (h *Handler) CreateAnalysisVersion(c *gin.Context) {
	analysisID := c.Param("id")

	// Parse request body (optional edits to apply to new version)
	var req struct {
		Edits map[string]interface{} `json:"edits"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Empty body is OK - just create version without edits
		req.Edits = nil
	}

	// Create new version via service
	newVersion, err := h.analysisSvc.CreateVersion(c.Request.Context(), analysisID, req.Edits)
	if err != nil {
		h.logger.Error().Err(err).Str("analysis_id", analysisID).Msg("Failed to create analysis version")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Version creation failed",
			Message: "Failed to create new analysis version",
		})
		return
	}

	// Transform to response
	analysisData := make(map[string]interface{})
	analysisBytes, _ := json.Marshal(newVersion)
	json.Unmarshal(analysisBytes, &analysisData)

	response := AnalysisResponse{
		ID:           newVersion.ID,
		SubmissionID: newVersion.SubmissionID,
		Status:       newVersion.Status,
		Version:      newVersion.Version,
		ParentID:     newVersion.ParentAnalysisID,
		Analysis:     analysisData,
		CreatedAt:    newVersion.CreatedAt,
		UpdatedAt:    newVersion.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
	})
}

// ApproveAnalysis handles POST /api/v1/admin/analysis/:id/approve
// Changes status from "completed" → "approved" and triggers PDF generation
func (h *Handler) ApproveAnalysis(c *gin.Context) {
	analysisID := c.Param("id")

	// Get analysis to verify it exists and is in "completed" state
	analysis, err := h.analysisSvc.GetByID(c.Request.Context(), analysisID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Analysis not found",
		})
		return
	}

	// Validate status is "completed"
	if analysis.Status != "completed" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid status",
			Message: "Analysis must be in 'completed' status to approve",
		})
		return
	}

	// Approve analysis (service handles status update AND PDF job enqueueing)
	err = h.analysisSvc.Approve(c.Request.Context(), analysisID)
	if err != nil {
		h.logger.Error().Err(err).Str("analysis_id", analysisID).Msg("Failed to approve analysis")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Approval failed",
			Message: err.Error(),
		})
		return
	}

	// Fetch updated analysis
	updatedAnalysis, err := h.analysisSvc.GetByID(c.Request.Context(), analysisID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Fetch failed",
			Message: "Approved but failed to fetch updated analysis",
		})
		return
	}

	// Transform to response
	analysisData := make(map[string]interface{})
	analysisBytes, _ := json.Marshal(updatedAnalysis)
	json.Unmarshal(analysisBytes, &analysisData)

	response := AnalysisResponse{
		ID:           updatedAnalysis.ID,
		SubmissionID: updatedAnalysis.SubmissionID,
		Status:       updatedAnalysis.Status,
		Version:      updatedAnalysis.Version,
		ParentID:     updatedAnalysis.ParentAnalysisID,
		Analysis:     analysisData,
		CreatedAt:    updatedAnalysis.CreatedAt,
		UpdatedAt:    updatedAnalysis.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
		"message":  "Analysis approved, PDF generation job enqueued",
	})
}

// SendAnalysis handles POST /api/v1/admin/analysis/:id/send
// Changes status from "approved" → "sent" and notifies user
func (h *Handler) SendAnalysis(c *gin.Context) {
	analysisID := c.Param("id")

	// Parse request body (user email to send to)
	var req struct {
		UserEmail string `json:"userEmail" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Valid user email is required",
		})
		return
	}

	// Get analysis to verify it exists and is in "approved" state
	analysis, err := h.analysisSvc.GetByID(c.Request.Context(), analysisID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Analysis not found",
		})
		return
	}

	// Validate status is "approved"
	if analysis.Status != "approved" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid status",
			Message: "Analysis must be in 'approved' status to send",
		})
		return
	}

	// Send analysis via service
	// This will:
	// 1. Verify PDF exists
	// 2. Update analysis status to "sent"
	// 3. Update sent_at timestamp and sent_to email
	// 4. Trigger user notification
	err = h.analysisSvc.Send(c.Request.Context(), analysisID, req.UserEmail)
	if err != nil {
		h.logger.Error().Err(err).Str("analysis_id", analysisID).Msg("Failed to send analysis")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Send failed",
			Message: "Failed to send analysis to user",
		})
		return
	}

	// Fetch updated analysis
	updatedAnalysis, err := h.analysisSvc.GetByID(c.Request.Context(), analysisID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Fetch failed",
			Message: "Sent but failed to fetch updated analysis",
		})
		return
	}

	// Transform to response
	analysisData := make(map[string]interface{})
	analysisBytes, _ := json.Marshal(updatedAnalysis)
	json.Unmarshal(analysisBytes, &analysisData)

	response := AnalysisResponse{
		ID:           updatedAnalysis.ID,
		SubmissionID: updatedAnalysis.SubmissionID,
		Status:       updatedAnalysis.Status,
		Version:      updatedAnalysis.Version,
		ParentID:     updatedAnalysis.ParentAnalysisID,
		Analysis:     analysisData,
		CreatedAt:    updatedAnalysis.CreatedAt,
		UpdatedAt:    updatedAnalysis.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
		"message":  "Analysis sent to user successfully",
	})
}

// GetAnalysisAdmin handles GET /api/v1/admin/analysis/:id
// Admin can fetch analysis directly by analysis ID (not just by submission ID)
func (h *Handler) GetAnalysisAdmin(c *gin.Context) {
	analysisID := c.Param("id")

	// Get analysis by ID (admin access, no ownership check needed)
	analysis, err := h.analysisSvc.GetByID(c.Request.Context(), analysisID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Analysis not found",
		})
		return
	}

	// Transform to response
	analysisData := make(map[string]interface{})
	analysisBytes, _ := json.Marshal(analysis)
	json.Unmarshal(analysisBytes, &analysisData)

	response := AnalysisResponse{
		ID:           analysis.ID,
		SubmissionID: analysis.SubmissionID,
		Status:       analysis.Status,
		Version:      analysis.Version,
		ParentID:     analysis.ParentAnalysisID,
		Analysis:     analysisData,
		CreatedAt:    analysis.CreatedAt,
		UpdatedAt:    analysis.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
	})
}
