package api

import (
	"net/http"

	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog" // Changed from 'log' to 'zerolog' for explicit logger usage
)

// EnrichmentHandlers holds all service dependencies and configuration for enrichment handlers
type EnrichmentHandlers struct {
	SubmissionService *submission.Service
	EnrichmentService *enrichment.Service
	AnalysisService   *analysis.Service
	Logger            zerolog.Logger
}

// NewEnrichmentHandlers creates a new enrichment.Handlers instance with all dependencies
func NewEnrichmentHandlers(
	submissionSvc *submission.Service,
	enrichmentSvc *enrichment.Service,
	analysisSvc *analysis.Service,
	logger zerolog.Logger,
) *EnrichmentHandlers {
	return &EnrichmentHandlers{
		SubmissionService: submissionSvc,
		EnrichmentService: enrichmentSvc,
		AnalysisService:   analysisSvc,
		Logger:            logger,
	}
}

// GetEnrichment handles GET /api/v1/submissions/:id/enrichment
func (h *EnrichmentHandlers) GetEnrichment(c *gin.Context) {
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
				Message: "You don't have permission to access this enrichment",
			})
			return
		}
	}

	// Get enrichment data
	enrichment, err := h.EnrichmentService.GetBySubmissionID(c.Request.Context(), subUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Enrichment data not found for this submission",
		})
		return
	}

	// Transform to DTO to fix field name mismatch (enrichedData -> data)
	// Use helper function for inferred progress
	response := h.buildEnrichmentResponse(enrichment)

	c.JSON(http.StatusOK, gin.H{
		"enrichment": response,
	})
}

// UpdateEnrichment handles PUT /api/v1/admin/enrichment/:id
// Admin can edit enrichment fields BEFORE approval (status remains "completed")
// Cannot edit after status is "approved"
func (h *EnrichmentHandlers) UpdateEnrichment(c *gin.Context) {
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
	updatedEnrichment, err := h.EnrichmentService.UpdateFields(c.Request.Context(), enrichUUID, updateData)
	if err != nil {
		h.Logger.Error().Err(err).Str("enrichment_id", enrichmentID).Msg("Failed to update enrichment")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Update failed",
			Message: "Failed to update enrichment fields",
		})
		return
	}

	// Transform to DTO
	response := h.buildEnrichmentResponse(updatedEnrichment)

	c.JSON(http.StatusOK, gin.H{
		"enrichment": response,
	})
}

// GetEnrichmentProgress handles GET /api/v1/enrichment/:id/progress
// Public endpoint (or protected) for polling progress
func (h *EnrichmentHandlers) GetEnrichmentProgress(c *gin.Context) {
	enrichmentID := c.Param("id")

	// Parse UUID
	enrichUUID, err := parseUUID(enrichmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid enrichment ID format"})
		return
	}

	// Get enrichment
	enrichment, err := h.EnrichmentService.GetByID(c.Request.Context(), enrichUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Enrichment not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"progress":    enrichment.Progress,
		"currentStep": enrichment.CurrentStep,
		"status":      enrichment.Status,
	})
}

// ApproveEnrichment handles POST /api/v1/admin/enrichment/:id/approve
// Changes status from "completed" → "approved" and triggers analysis creation
func (h *EnrichmentHandlers) ApproveEnrichment(c *gin.Context) {
	enrichmentID := c.Param("id")

	// Parse and validate UUID
	enrichUUID, err := parseUUID(enrichmentID)
	if err != nil {
		respondError(c, h.Logger, http.StatusBadRequest, err, "ID de enriquecimento inválido.")
		return
	}

	// Approve enrichment (service handles status update AND job enqueueing)
	err = h.EnrichmentService.Approve(c.Request.Context(), enrichUUID)
	if err != nil {
		respondError(c, h.Logger, http.StatusBadRequest, err, "Não foi possível aprovar o enriquecimento.")
		return
	}

	// Fetch updated enrichment
	updatedEnrichment, err := h.EnrichmentService.GetByID(c.Request.Context(), enrichUUID)
	if err != nil {
		respondError(c, h.Logger, http.StatusInternalServerError, err, "Aprovado, mas não foi possível carregar o enriquecimento atualizado.")
		return
	}

	// Transform to DTO
	enrichmentResponse := h.buildEnrichmentResponse(updatedEnrichment)

	// Best-effort: fetch latest analysis for this submission (may still be queued)
	var analysisID *string
	var analysisStatus *string
	if h.AnalysisService != nil {
		if a, err := h.AnalysisService.GetBySubmissionID(c.Request.Context(), updatedEnrichment.SubmissionID.String()); err == nil && a != nil {
			id := a.ID
			analysisID = &id
			status := a.Status
			analysisStatus = &status
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"enrichment":     enrichmentResponse,
		"analysisId":     analysisID,
		"analysisStatus": analysisStatus,
		"message":        "Enriquecimento aprovado, analise iniciada.",
	})
}

// UnlockEnrichment handles POST /api/v1/admin/enrichment/:id/unlock
// Manually unlocks a stuck enrichment (recovery endpoint)
func (h *EnrichmentHandlers) UnlockEnrichment(c *gin.Context) {
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

	// Get enrichment
	enrichment, err := h.EnrichmentService.GetByID(c.Request.Context(), enrichUUID)
	if err != nil {
		h.Logger.Error().Err(err).Str("enrichment_id", enrichmentID).Msg("Failed to fetch enrichment")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Fetch failed",
			Message: err.Error(),
		})
		return
	}

	// Unlock the enrichment
	enrichment.IsLocked = false
	err = h.EnrichmentService.UpdateUnlock(c.Request.Context(), enrichment)
	if err != nil {
		h.Logger.Error().Err(err).Str("enrichment_id", enrichmentID).Msg("Failed to unlock enrichment")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Unlock failed",
			Message: err.Error(),
		})
		return
	}

	// Transform to DTO
	enrichmentResponse := h.buildEnrichmentResponse(enrichment)

	c.JSON(http.StatusOK, gin.H{
		"enrichment": enrichmentResponse,
		"message":    "Enrichment unlocked successfully",
	})
}

// GetEnrichmentAdmin handles GET /api/v1/admin/enrichment/:id
// Admin can fetch enrichment directly by enrichment ID (not just by submission ID)
func (h *EnrichmentHandlers) GetEnrichmentAdmin(c *gin.Context) {
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
	enrichment, err := h.EnrichmentService.GetByID(c.Request.Context(), enrichUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Enrichment not found",
		})
		return
	}

	// Transform to DTO
	response := h.buildEnrichmentResponse(enrichment)

	c.JSON(http.StatusOK, gin.H{
		"enrichment": response,
	})
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// calculateInferredProgress dynamically calculates progress based on status and current_step
// This implements the "inferred progress" pattern to avoid storing redundant state in DB
func (h *EnrichmentHandlers) calculateInferredProgress(status string, currentStep string) int {
	// Completed or approved = 100%
	if status == "completed" || status == "approved" {
		return 100
	}

	// Pending = infer from current step
	if status == "pending" {
		switch currentStep {
		case "Scanning digital footprint (Transient)...":
			return 10
		case "Agent is synthesizing Intelligence Profile...":
			return 50
		case "Finalizing profile...":
			return 90
		default:
			return 5 // Just started
		}
	}

	return 0
}

// buildEnrichmentResponse creates an EnrichmentResponse from domain model
// Uses inferred progress instead of database field for better accuracy
func (h *EnrichmentHandlers) buildEnrichmentResponse(e *enrichment.Enrichment) EnrichmentResponse {
	// DEBUG: Log the enriched data to verify it's being populated
	h.Logger.Debug().
		Str("enrichment_id", e.ID.String()).
		Interface("enriched_data", e.EnrichedData).
		Int("data_size", len(e.EnrichedData)).
		Msg("Building enrichment response")

	// Calculate progress and determine the appropriate step message
	// NullableString must be converted to string for these helper functions
	progress := h.calculateInferredProgress(string(e.Status), string(e.CurrentStep))
	currentStep := h.getDisplayStep(string(e.Status), string(e.CurrentStep))

	return EnrichmentResponse{
		ID:           e.ID.String(),
		SubmissionID: e.SubmissionID.String(),
		Status:       string(e.Status),
		Progress:     progress,
		CurrentStep:  currentStep,
		Data:         e.EnrichedData,
		IsLocked:     e.IsLocked,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}

// getDisplayStep returns a user-friendly step message based on status
// This ensures completed/approved statuses show appropriate messages even if
// the database has stale step data
func (h *EnrichmentHandlers) getDisplayStep(status string, dbStep string) string {
	switch status {
	case "completed":
		return "Enrichment complete - awaiting review"
	case "approved":
		return "Approved - analysis in progress"
	default:
		// For pending status, use the database step
		if dbStep == "" {
			return "Queued for enrichment"
		}
		return dbStep
	}
}

// GetEnrichmentBySubmissionAdmin handles GET /api/v1/admin/submissions/:id/enrichment
// Admin can fetch enrichment by submission ID (for admin edit page)
func (h *EnrichmentHandlers) GetEnrichmentBySubmissionAdmin(c *gin.Context) {
	submissionID := c.Param("id")

	// Parse and validate UUID
	subUUID, err := parseUUID(submissionID)
	if err != nil {
		h.Logger.Error().Err(err).Str("submission_id", submissionID).Msg("Invalid submission ID format")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid ID",
			Message: "Invalid submission ID format",
		})
		return
	}

	h.Logger.Debug().Str("submission_id", subUUID.String()).Msg("Fetching enrichment by submission ID")

	// Get enrichment by submission ID (admin access, no ownership check needed)
	enrichment, err := h.EnrichmentService.GetBySubmissionID(c.Request.Context(), subUUID)
	if err != nil {
		h.Logger.Error().Err(err).Str("submission_id", subUUID.String()).Msg("Failed to fetch enrichment")
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Enrichment not found for this submission",
		})
		return
	}

	h.Logger.Debug().
		Str("enrichment_id", enrichment.ID.String()).
		Str("submission_id", subUUID.String()).
		Str("status", string(enrichment.Status)).
		Msg("Enrichment fetched successfully")

	// Transform to DTO
	response := h.buildEnrichmentResponse(enrichment)
	c.JSON(http.StatusOK, response)
}
