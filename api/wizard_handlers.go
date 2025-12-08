package api

import (
	"net/http"

	"backend_v3/domain/challenge"
	"backend_v3/domain/wizard"

	"github.com/gin-gonic/gin"
)

// WizardHandlers handles wizard-related HTTP endpoints
type WizardHandlers struct {
	wizardService *wizard.Service
}

// NewWizardHandlers creates a new wizard handlers instance
func NewWizardHandlers(wizardService *wizard.Service) *WizardHandlers {
	return &WizardHandlers{
		wizardService: wizardService,
	}
}

// --- Request/Response Types ---

type StartWizardRequest struct {
	CompanyID   string `json:"company_id" binding:"required"`
	ChallengeID string `json:"challenge_id" binding:"required"`
}

type GenerateStepRequest struct {
	HumanContext string            `json:"human_context"`
	Answers      map[string]string `json:"answers"`
}

type RefineStepRequest struct {
	AdditionalContext string `json:"additional_context"`
	Notes             string `json:"notes"`
}

type WizardResponse struct {
	State   *wizard.State `json:"state"`
	Message string        `json:"message,omitempty"`
}

// --- Handlers ---

// GetWizardState returns the current wizard state for an analysis
// GET /api/v1/analyses/:id/wizard
func (h *WizardHandlers) GetWizardState(c *gin.Context) {
	analysisID := c.Param("id")
	if analysisID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "analysis ID required"})
		return
	}

	state, err := h.wizardService.GetCurrentState(c.Request.Context(), analysisID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, WizardResponse{State: state})
}

// StartWizard initializes or resumes a wizard for a company + challenge
// POST /api/v1/wizard/start
func (h *WizardHandlers) StartWizard(c *gin.Context) {
	var req StartWizardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_id and challenge_id are required"})
		return
	}

	state, err := h.wizardService.StartWizard(c.Request.Context(), req.CompanyID, req.ChallengeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, WizardResponse{
		State:   state,
		Message: "Wizard started successfully",
	})
}

// GenerateStep generates the framework output for the current step
// POST /api/v1/analyses/:id/wizard/generate
func (h *WizardHandlers) GenerateStep(c *gin.Context) {
	analysisID := c.Param("id")
	if analysisID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "analysis ID required"})
		return
	}

	var req GenerateStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body
		req = GenerateStepRequest{}
	}

	state, err := h.wizardService.GenerateStep(c.Request.Context(), analysisID, req.HumanContext, req.Answers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, WizardResponse{
		State:   state,
		Message: "Step generated successfully",
	})
}

// ApproveStep approves the current step and advances to the next
// POST /api/v1/analyses/:id/wizard/approve
func (h *WizardHandlers) ApproveStep(c *gin.Context) {
	analysisID := c.Param("id")
	if analysisID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "analysis ID required"})
		return
	}

	// Get user ID from context if available
	var userID *string
	if uid, exists := c.Get("user_id"); exists {
		if uidStr, ok := uid.(string); ok {
			userID = &uidStr
		}
	}

	state, err := h.wizardService.ApproveStep(c.Request.Context(), analysisID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, WizardResponse{
		State:   state,
		Message: "Step approved successfully",
	})
}

// RefineStep regenerates the current step with additional context
// POST /api/v1/analyses/:id/wizard/refine
func (h *WizardHandlers) RefineStep(c *gin.Context) {
	analysisID := c.Param("id")
	if analysisID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "analysis ID required"})
		return
	}

	var req RefineStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	state, err := h.wizardService.RefineStep(c.Request.Context(), analysisID, req.AdditionalContext, req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, WizardResponse{
		State:   state,
		Message: "Step refined successfully",
	})
}

// GetWizardSummary returns all approved frameworks
// GET /api/v1/analyses/:id/wizard/summary
func (h *WizardHandlers) GetWizardSummary(c *gin.Context) {
	analysisID := c.Param("id")
	if analysisID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "analysis ID required"})
		return
	}

	results, err := h.wizardService.GetWizardSummary(c.Request.Context(), analysisID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"framework_results": results})
}

// GetChallengeTypes returns all available challenge categories and types
// GET /api/v1/challenges/types
func (h *WizardHandlers) GetChallengeTypes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"categories": challenge.ChallengeCategories,
		"types":      challenge.GetAllChallengeTypes(),
	})
}

// GetFrameworkOrder returns the framework wizard order with questions
// GET /api/v1/frameworks/order
func (h *WizardHandlers) GetFrameworkOrder(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"frameworks":  wizard.GetAllFrameworks(),
		"total_steps": wizard.TotalSteps,
	})
}

// GenerateAllSteps generates all frameworks at once, bypassing wizard step-by-step approval
// POST /api/v1/admin/analysis/:id/wizard/generate-all
// Admin-only endpoint for bulk generation
func (h *WizardHandlers) GenerateAllSteps(c *gin.Context) {
	analysisID := c.Param("id")
	if analysisID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "analysis ID required"})
		return
	}

	progress, err := h.wizardService.GenerateAll(c.Request.Context(), analysisID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Bulk generation complete",
		"progress": progress,
	})
}
