package api

import (
	"encoding/json"
	"net/http"

	"backend_v3/domain/analysis"
	"backend_v3/domain/challenge"
	"backend_v3/domain/company"
	"backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// AnalysisHandlers holds all service dependencies and configuration for analysis handlers
type AnalysisHandlers struct {
	SubmissionService *submission.Service
	AnalysisService   *analysis.Service
	CompanyService    *company.Service
	ChallengeService  *challenge.Service
	Logger            zerolog.Logger
}

// NewAnalysisHandlers creates a new analysis handler set with all dependencies
func NewAnalysisHandlers(
	submissionSvc *submission.Service,
	analysisSvc *analysis.Service,
	companySvc *company.Service,
	challengeSvc *challenge.Service,
	logger zerolog.Logger,
) *AnalysisHandlers {
	return &AnalysisHandlers{
		SubmissionService: submissionSvc,
		AnalysisService:   analysisSvc,
		CompanyService:    companySvc,
		ChallengeService:  challengeSvc,
		Logger:            logger,
	}
}

// NOTE: GetAnalysis (by submission ID) was removed in v2_013 schema cleanup
// Use GetAnalysisUser (by analysis ID) or GetAnalysisByChallengeID instead

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

	// Unwrap "analysis" key if frontend sends nested structure
	// Frontend may send: {"analysis": {"swot": {...}}} instead of {"swot": {...}}
	if nested, ok := updateData["analysis"].(map[string]interface{}); ok {
		h.Logger.Debug().Msg("Unwrapping nested 'analysis' key from request")
		updateData = nested
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

	response := buildAnalysisResponse(updatedAnalysis, analysisData)

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
	})
}

// GetAnalysisUser handles GET /api/v1/analyses/:id
// User can fetch analysis by ID if they have access to the associated company
func (h *AnalysisHandlers) GetAnalysisUser(c *gin.Context) {
	analysisID := c.Param("id")

	// Verify user is authenticated (userID needed for future company ownership checks)
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Get analysis by ID
	analysis, err := h.AnalysisService.GetByID(c.Request.Context(), analysisID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Analysis not found",
		})
		return
	}

	// Check if user has access via admin role
	role, _ := c.Get("userRole")
	userRole := ""
	if r, ok := role.(string); ok {
		userRole = r
	}

	// Admin can access any analysis; non-admin users must have visibility enabled
	// In v2, access control is based on company ownership (to be implemented via company service)
	// For now, non-admin users can only access visible analyses
	if userRole != "admin" && userRole != "super_admin" && !analysis.IsVisibleToUser {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Not available",
			Message: "This analysis is not yet available for viewing",
		})
		return
	}

	// Transform to response
	analysisData := make(map[string]interface{})
	analysisBytes, _ := json.Marshal(analysis)
	json.Unmarshal(analysisBytes, &analysisData)

	response := buildAnalysisResponse(analysis, analysisData)

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

	response := buildAnalysisResponse(analysis, analysisData)

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
	})
}

// NOTE: GetAnalysisBySubmissionAdmin was removed in v2_013 schema cleanup
// Use GetAnalysisAdmin (by analysis ID) or GetAnalysisByChallengeID instead

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

	response := buildAnalysisResponse(updatedAnalysis, analysisData)

	action := "hidden from"
	if req.Visible {
		action = "made visible to"
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
		"message":  "Analysis " + action + " user successfully",
	})
}

// TogglePublic handles POST /api/v1/admin/analysis/:id/public
// Admin toggles whether the analysis is accessible without login
// When true: anyone with access code can view (public)
// When false: access code requires authentication (private)
func (h *AnalysisHandlers) TogglePublic(c *gin.Context) {
	analysisID := c.Param("id")

	// Parse request body
	var req struct {
		Public bool `json:"public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "public field is required (boolean)",
		})
		return
	}

	// Toggle public status via service
	err := h.AnalysisService.SetPublicStatus(c.Request.Context(), analysisID, req.Public)
	if err != nil {
		h.Logger.Error().Err(err).Str("analysis_id", analysisID).Msg("Failed to toggle public status")
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
			Message: "Public status toggled but failed to fetch updated analysis",
		})
		return
	}

	// Transform to response
	analysisData := make(map[string]interface{})
	analysisBytes, _ := json.Marshal(updatedAnalysis)
	json.Unmarshal(analysisBytes, &analysisData)

	response := buildAnalysisResponse(updatedAnalysis, analysisData)

	action := "made private (login required)"
	if req.Public {
		action = "made public (no login required)"
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
		"message":  "Analysis " + action + " successfully",
	})
}

// UpdateVisibility handles PATCH /api/v1/admin/analysis/:id/visibility
// Admin sets visibility and public status in one call
func (h *AnalysisHandlers) UpdateVisibility(c *gin.Context) {
	analysisID := c.Param("id")

	// Parse request body - both fields are optional
	var req struct {
		IsVisibleToUser *bool `json:"is_visible_to_user"`
		IsPublic        *bool `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Expected JSON with is_visible_to_user and/or is_public fields",
		})
		return
	}

	// Update visibility if provided
	if req.IsVisibleToUser != nil {
		if err := h.AnalysisService.SetVisibility(c.Request.Context(), analysisID, *req.IsVisibleToUser); err != nil {
			h.Logger.Error().Err(err).Str("analysis_id", analysisID).Msg("Failed to set visibility")
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Update failed",
				Message: err.Error(),
			})
			return
		}
	}

	// Update public status if provided
	if req.IsPublic != nil {
		if err := h.AnalysisService.SetPublicStatus(c.Request.Context(), analysisID, *req.IsPublic); err != nil {
			h.Logger.Error().Err(err).Str("analysis_id", analysisID).Msg("Failed to set public status")
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Update failed",
				Message: err.Error(),
			})
			return
		}
	}

	// Fetch updated analysis to return current state
	updatedAnalysis, err := h.AnalysisService.GetByID(c.Request.Context(), analysisID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Fetch failed",
			Message: "Visibility updated but failed to fetch analysis",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                 updatedAnalysis.ID,
		"is_visible_to_user": updatedAnalysis.IsVisibleToUser,
		"is_public":          updatedAnalysis.IsPublic,
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
// Public endpoint - no auth required for public users IF is_public=true
// Admin users can preview even if visibility is OFF (they need to pass ?preview=admin with valid JWT)
// Authenticated users can access even if is_public=false (private mode requires login)
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
	isAuthenticated := false
	if c.Query("preview") == "admin" {
		// Check if we have a valid Authorization header with admin role
		role, exists := c.Get("userRole")
		if exists {
			userRole, ok := role.(string)
			if ok && (userRole == "admin" || userRole == "super_admin") {
				isAdminPreview = true
				isAuthenticated = true
			}
		}
	}

	// Check if user is authenticated (non-admin)
	if !isAuthenticated {
		_, exists := c.Get("userID")
		isAuthenticated = exists
	}

	// Check if analysis is visible to users (skip if admin preview)
	if !analysis.IsVisibleToUser && !isAdminPreview {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Relatório não encontrado. O código de acesso é inválido ou o relatório não está mais disponível.",
		})
		return
	}

	// Check public access: if is_public=false, user must be authenticated
	// This implements the 4-state visibility matrix
	if !analysis.IsPublic && !isAuthenticated && !isAdminPreview {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Message: "Este relatório requer autenticação. Por favor, faça login para acessar.",
		})
		return
	}

	// Get company info for display
	var companyData *company.Company
	if analysis.CompanyID != nil {
		h.Logger.Debug().Str("company_id", *analysis.CompanyID).Msg("Fetching company for report")
		companyUUID, parseErr := uuid.Parse(*analysis.CompanyID)
		if parseErr != nil {
			h.Logger.Warn().Err(parseErr).Str("company_id", *analysis.CompanyID).Msg("Failed to parse company UUID")
		} else {
			companyData, err = h.CompanyService.GetByID(c.Request.Context(), companyUUID)
			if err != nil {
				h.Logger.Warn().Err(err).Str("company_id", *analysis.CompanyID).Msg("Could not fetch company for report")
			} else if companyData != nil {
				h.Logger.Debug().Str("company_name", companyData.Name).Msg("Company found for report")
			}
		}
	} else {
		h.Logger.Warn().Str("analysis_id", analysis.ID).Msg("Analysis has no company_id")
	}

	// Get challenge info for business context (ChallengeID is required, never nil)
	var businessChallenge string
	if analysis.ChallengeID != uuid.Nil {
		challengeData, err := h.ChallengeService.GetByID(c.Request.Context(), analysis.ChallengeID)
		if err == nil && challengeData != nil {
			businessChallenge = challengeData.BusinessChallenge
		}
	}

	// Extract framework_results from the analysis
	frameworkResults := make(map[string]interface{})
	frameworkKeys := []string{}
	for key, value := range analysis.FrameworkResults {
		var parsed interface{}
		if err := json.Unmarshal(value, &parsed); err == nil {
			frameworkResults[key] = parsed
			frameworkKeys = append(frameworkKeys, key)
		}
	}
	h.Logger.Debug().Strs("frameworks", frameworkKeys).Int("count", len(frameworkResults)).Msg("Parsed framework results")

	// Build response with framework_results at root level (matching PublicReportData interface)
	response := gin.H{
		"id":                analysis.ID,
		"company_id":        analysis.CompanyID,
		"challenge_id":      analysis.ChallengeID.String(),
		"status":            analysis.Status,
		"framework_results": frameworkResults,
		"created_at":        analysis.CreatedAt,
		"is_public":         analysis.IsPublic,
	}

	// Add company info for display
	if companyData != nil {
		response["company_name"] = companyData.Name
		response["industry"] = companyData.Industry
	}

	// Add business challenge
	if businessChallenge != "" {
		response["business_challenge"] = businessChallenge
	}

	// Add preview flag for admin preview
	if isAdminPreview {
		response["is_admin_preview"] = true
	}

	c.JSON(http.StatusOK, response)
}

// buildAnalysisResponse is a helper to construct AnalysisResponse from domain.Analysis
// Ensures consistent handling of nullable fields and ChallengeID
func buildAnalysisResponse(analysis *analysis.Analysis, analysisData map[string]interface{}) AnalysisResponse {
	return AnalysisResponse{
		ID:              analysis.ID,
		CompanyID:       analysis.CompanyID,
		ChallengeID:     analysis.ChallengeID.String(),
		Status:          analysis.Status,
		Analysis:        analysisData,
		IsVisibleToUser: analysis.IsVisibleToUser,
		IsPublic:        analysis.IsPublic,
		AccessCode:      analysis.AccessCode,
		CreatedAt:       analysis.CreatedAt,
		UpdatedAt:       analysis.UpdatedAt,
	}
}
