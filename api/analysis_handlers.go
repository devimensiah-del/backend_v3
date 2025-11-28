package api

import (
	"encoding/json"
	"net/http"

	"backend_v3/domain/analysis" // Assuming this domain exists
	"backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog" // Needed for logger
)

// AnalysisHandlers holds all service dependencies and configuration for analysis handlers
type AnalysisHandlers struct {
	SubmissionService *submission.Service
	AnalysisService   *analysis.Service
	Logger            zerolog.Logger
}

// NewAnalysisHandlers creates a new analysis handler set with all dependencies
func NewAnalysisHandlers(
	submissionSvc *submission.Service,
	analysisSvc *analysis.Service,
	logger zerolog.Logger,
) *AnalysisHandlers {
	return &AnalysisHandlers{
		SubmissionService: submissionSvc,
		AnalysisService:   analysisSvc,
		Logger:            logger,
	}
}

// GetAnalysis handles GET /api/v1/submissions/:id/analysis
func (h *AnalysisHandlers) GetAnalysis(c *gin.Context) {
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
	submission, err := h.SubmissionService.GetByID(c.Request.Context(), subUUID)
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
	analysis, err := h.AnalysisService.GetBySubmissionID(c.Request.Context(), submissionID)
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
		ID:              analysis.ID,
		SubmissionID:    analysis.SubmissionID,
		Status:          analysis.Status,
		Analysis:        analysisData,
		IsVisibleToUser: analysis.IsVisibleToUser,
		IsBlurred:       analysis.IsBlurred,
		AccessCode:      analysis.AccessCode,
		CreatedAt:       analysis.CreatedAt,
		UpdatedAt:       analysis.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
	})
}

// UpdateAnalysis handles PUT /api/v1/admin/analysis/:id
// Admin can edit analysis framework fields (status remains unchanged)
func (h *AnalysisHandlers) UpdateAnalysis(c *gin.Context) {
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
	updatedAnalysis, err := h.AnalysisService.UpdateFields(c.Request.Context(), analysisID, updateData)
	if err != nil {
		h.Logger.Error().Err(err).Str("analysis_id", analysisID).Msg("Failed to update analysis")
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
		ID:              updatedAnalysis.ID,
		SubmissionID:    updatedAnalysis.SubmissionID,
		Status:          updatedAnalysis.Status,
		Analysis:        analysisData,
		IsVisibleToUser: updatedAnalysis.IsVisibleToUser,
		IsBlurred:       updatedAnalysis.IsBlurred,
		AccessCode:      updatedAnalysis.AccessCode,
		CreatedAt:       updatedAnalysis.CreatedAt,
		UpdatedAt:       updatedAnalysis.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
	})
}

// GetAnalysisAdmin handles GET /api/v1/admin/analysis/:id
// Admin can fetch analysis directly by analysis ID (not just by submission ID)
func (h *AnalysisHandlers) GetAnalysisAdmin(c *gin.Context) {
	analysisID := c.Param("id")

	// Get analysis by ID (admin access, no ownership check needed)
	analysis, err := h.AnalysisService.GetByID(c.Request.Context(), analysisID)
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
		ID:              analysis.ID,
		SubmissionID:    analysis.SubmissionID,
		Status:          analysis.Status,
		Analysis:        analysisData,
		IsVisibleToUser: analysis.IsVisibleToUser,
		IsBlurred:       analysis.IsBlurred,
		AccessCode:      analysis.AccessCode,
		CreatedAt:       analysis.CreatedAt,
		UpdatedAt:       analysis.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
	})
}

// GetAnalysisBySubmissionAdmin handles GET /api/v1/admin/submissions/:id/analysis
// Admin can fetch analysis by submission ID (no ownership check needed)
func (h *AnalysisHandlers) GetAnalysisBySubmissionAdmin(c *gin.Context) {
	submissionID := c.Param("id")

	// Get analysis by submission ID (admin access, no ownership check needed)
	analysis, err := h.AnalysisService.GetBySubmissionID(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Analysis not found for this submission",
		})
		return
	}

	// Transform to response
	analysisData := make(map[string]interface{})
	analysisBytes, _ := json.Marshal(analysis)
	json.Unmarshal(analysisBytes, &analysisData)

	response := AnalysisResponse{
		ID:              analysis.ID,
		SubmissionID:    analysis.SubmissionID,
		Status:          analysis.Status,
		Analysis:        analysisData,
		IsVisibleToUser: analysis.IsVisibleToUser,
		IsBlurred:       analysis.IsBlurred,
		AccessCode:      analysis.AccessCode,
		CreatedAt:       analysis.CreatedAt,
		UpdatedAt:       analysis.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
	})
}

// ToggleVisibility handles POST /api/v1/admin/analysis/:id/visibility
// Admin toggles whether the analysis is visible to end users
func (h *AnalysisHandlers) ToggleVisibility(c *gin.Context) {
	analysisID := c.Param("id")

	// Parse request body
	var req struct {
		Visible bool `json:"visible"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "visible field is required (boolean)",
		})
		return
	}

	// Toggle visibility via service
	err := h.AnalysisService.SetVisibility(c.Request.Context(), analysisID, req.Visible)
	if err != nil {
		h.Logger.Error().Err(err).Str("analysis_id", analysisID).Msg("Failed to toggle visibility")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Toggle failed",
			Message: err.Error(),
		})
		return
	}

	// Fetch updated analysis
	updatedAnalysis, err := h.AnalysisService.GetByID(c.Request.Context(), analysisID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Fetch failed",
			Message: "Visibility toggled but failed to fetch updated analysis",
		})
		return
	}

	// Transform to response
	analysisData := make(map[string]interface{})
	analysisBytes, _ := json.Marshal(updatedAnalysis)
	json.Unmarshal(analysisBytes, &analysisData)

	response := AnalysisResponse{
		ID:              updatedAnalysis.ID,
		SubmissionID:    updatedAnalysis.SubmissionID,
		Status:          updatedAnalysis.Status,
		Analysis:        analysisData,
		IsVisibleToUser: updatedAnalysis.IsVisibleToUser,
		IsBlurred:       updatedAnalysis.IsBlurred,
		AccessCode:      updatedAnalysis.AccessCode,
		CreatedAt:       updatedAnalysis.CreatedAt,
		UpdatedAt:       updatedAnalysis.UpdatedAt,
	}

	action := "hidden from"
	if req.Visible {
		action = "made visible to"
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
		"message":  "Analysis " + action + " user successfully",
	})
}

// ToggleBlur handles POST /api/v1/admin/analysis/:id/blur
// Admin toggles the blur overlay for premium frameworks
func (h *AnalysisHandlers) ToggleBlur(c *gin.Context) {
	analysisID := c.Param("id")

	// Parse request body
	var req struct {
		Blurred bool `json:"blurred"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "blurred field is required (boolean)",
		})
		return
	}

	// Toggle blur status via service
	err := h.AnalysisService.SetBlurStatus(c.Request.Context(), analysisID, req.Blurred)
	if err != nil {
		h.Logger.Error().Err(err).Str("analysis_id", analysisID).Msg("Failed to toggle blur status")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Toggle failed",
			Message: err.Error(),
		})
		return
	}

	// Fetch updated analysis
	updatedAnalysis, err := h.AnalysisService.GetByID(c.Request.Context(), analysisID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Fetch failed",
			Message: "Blur status toggled but failed to fetch updated analysis",
		})
		return
	}

	// Transform to response
	analysisData := make(map[string]interface{})
	analysisBytes, _ := json.Marshal(updatedAnalysis)
	json.Unmarshal(analysisBytes, &analysisData)

	response := AnalysisResponse{
		ID:              updatedAnalysis.ID,
		SubmissionID:    updatedAnalysis.SubmissionID,
		Status:          updatedAnalysis.Status,
		Analysis:        analysisData,
		IsVisibleToUser: updatedAnalysis.IsVisibleToUser,
		IsBlurred:       updatedAnalysis.IsBlurred,
		AccessCode:      updatedAnalysis.AccessCode,
		CreatedAt:       updatedAnalysis.CreatedAt,
		UpdatedAt:       updatedAnalysis.UpdatedAt,
	}

	action := "blurred"
	if !req.Blurred {
		action = "unblurred"
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
		"message":  "Analysis premium frameworks " + action + " successfully",
	})
}

// GenerateAccessCode handles POST /api/v1/admin/analysis/:id/access-code
// Admin generates a unique access code for public sharing
func (h *AnalysisHandlers) GenerateAccessCode(c *gin.Context) {
	analysisID := c.Param("id")

	// Generate access code via service (handles collision retry)
	accessCode, err := h.AnalysisService.GenerateAccessCode(c.Request.Context(), analysisID)
	if err != nil {
		h.Logger.Error().Err(err).Str("analysis_id", analysisID).Msg("Failed to generate access code")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Generation failed",
			Message: err.Error(),
		})
		return
	}

	// Build shareable URL
	baseURL := c.GetHeader("Origin")
	if baseURL == "" {
		baseURL = "https://imenseia.com.br"
	}
	shareableURL := baseURL + "/report/" + accessCode

	c.JSON(http.StatusOK, gin.H{
		"access_code":   accessCode,
		"shareable_url": shareableURL,
		"message":       "Access code generated successfully",
	})
}

// GetPublicReport handles GET /api/v1/public/report/:code
// Public endpoint - no auth required for public users
// Admin users can preview even if visibility is OFF (they need to pass ?preview=admin with valid JWT)
// Returns analysis data for public viewing if access code is valid and analysis is visible
func (h *AnalysisHandlers) GetPublicReport(c *gin.Context) {
	accessCode := c.Param("code")

	// Get analysis by access code
	analysis, err := h.AnalysisService.GetByAccessCode(c.Request.Context(), accessCode)
	if err != nil {
		h.Logger.Error().Err(err).Str("access_code", accessCode).Msg("Error fetching analysis by access code")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal error",
			Message: "Failed to retrieve report",
		})
		return
	}

	// Check if analysis exists (nil, nil means not found)
	if analysis == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Relatório não encontrado. O código de acesso é inválido ou o relatório não está mais disponível.",
		})
		return
	}

	// Check if this is an admin preview request
	isAdminPreview := false
	if c.Query("preview") == "admin" {
		// Check if we have a valid Authorization header with admin role
		role, exists := c.Get("userRole")
		if exists {
			userRole, ok := role.(string)
			if ok && (userRole == "admin" || userRole == "super_admin") {
				isAdminPreview = true
			}
		}
	}

	// Check if analysis is visible to users (skip if admin preview)
	if !analysis.IsVisibleToUser && !isAdminPreview {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Relatório não encontrado. O código de acesso é inválido ou o relatório não está mais disponível.",
		})
		return
	}

	// Transform to response (includes full analysis data)
	analysisData := make(map[string]interface{})
	analysisBytes, _ := json.Marshal(analysis)
	json.Unmarshal(analysisBytes, &analysisData)

	// Build response with essential fields for public report
	response := gin.H{
		"id":            analysis.ID,
		"submission_id": analysis.SubmissionID,
		"status":        analysis.Status,
		"analysis":      analysisData,
		"created_at":    analysis.CreatedAt,
		"is_blurred":    analysis.IsBlurred,
	}

	// Add preview flag for admin preview
	if isAdminPreview {
		response["is_admin_preview"] = true
	}

	c.JSON(http.StatusOK, response)
}
