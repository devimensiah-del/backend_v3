package api

import (
	"net/http"
	"strconv"

	"backend_v3/domain/analysisbysteps"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// AnalysisByStepsHandlers holds all service dependencies for step-by-step analysis handlers
type AnalysisByStepsHandlers struct {
	AnalysisByStepsService *analysisbysteps.Service
	Logger                 zerolog.Logger
}

// NewAnalysisByStepsHandlers creates a new handler instance with all dependencies
func NewAnalysisByStepsHandlers(
	analysisByStepsSvc *analysisbysteps.Service,
	logger zerolog.Logger,
) *AnalysisByStepsHandlers {
	return &AnalysisByStepsHandlers{
		AnalysisByStepsService: analysisByStepsSvc,
		Logger:                 logger,
	}
}

// StartAnalysisBySteps handles POST /api/v1/analyses/steps/start
// Starts a new step-by-step analysis for a challenge
// Request: { "challenge_id": "uuid" }
// Response: StartResponse with all 14 pending steps
func (h *AnalysisByStepsHandlers) StartAnalysisBySteps(c *gin.Context) {
	// Verify user is authenticated
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Parse request body
	var req StartAnalysisByStepsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// Validate UUID (Gin doesn't have native uuid validator)
	challengeID, err := uuid.Parse(req.ChallengeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "challenge_id must be a valid UUID",
		})
		return
	}

	// TODO: Verify user owns the challenge or is admin (company ownership check TBD)

	// Call service
	result, err := h.AnalysisByStepsService.StartAnalysisBySteps(c.Request.Context(), challengeID)
	if err != nil {
		h.Logger.Error().Err(err).Str("challenge_id", challengeID.String()).Msg("Failed to start analysis by steps")

		// Check for specific error types
		if err.Error() == "challenge not found" || err.Error() == "analysis not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Not found",
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to start analysis",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GenerateStep handles POST /api/v1/analyses/:id/steps/:step/generate
// Generates AI output for a specific step
// Path params: id (analysis UUID), step (0-13)
// Response: AnalysisStep with ai_output populated
func (h *AnalysisByStepsHandlers) GenerateStep(c *gin.Context) {
	// Verify user is authenticated
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Parse path parameters
	analysisID := c.Param("id")
	if analysisID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Analysis ID is required",
		})
		return
	}

	stepStr := c.Param("step")
	stepNumber, err := strconv.Atoi(stepStr)
	if err != nil || stepNumber < 0 || stepNumber > 13 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Step number must be 0-13",
		})
		return
	}

	// TODO: Verify user owns the analysis or is admin

	// Call service
	result, err := h.AnalysisByStepsService.GenerateStep(c.Request.Context(), analysisID, stepNumber)
	if err != nil {
		h.Logger.Error().
			Err(err).
			Str("analysis_id", analysisID).
			Int("step_number", stepNumber).
			Msg("Failed to generate step")

		// Check for specific error types
		if err.Error() == "analysis not found" || err.Error() == "step not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Not found",
				Message: err.Error(),
			})
			return
		}

		// Check for validation errors (previous step not approved)
		if err.Error() == "cannot generate step" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid request",
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to generate step",
			Message: err.Error(),
		})
		return
	}

	// Convert to response type
	response := buildAnalysisStepResponse(result)
	c.JSON(http.StatusOK, response)
}

// SaveHumanEdit handles PUT /api/v1/analyses/:id/steps/:step/edit
// Saves human edit for a step without regenerating
// Request: { "edited_content": "json string" }
// Response: Updated AnalysisStep
func (h *AnalysisByStepsHandlers) SaveHumanEdit(c *gin.Context) {
	// Verify user is authenticated
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Parse path parameters
	analysisID := c.Param("id")
	if analysisID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Analysis ID is required",
		})
		return
	}

	stepStr := c.Param("step")
	stepNumber, err := strconv.Atoi(stepStr)
	if err != nil || stepNumber < 0 || stepNumber > 13 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Step number must be 0-13",
		})
		return
	}

	// Parse request body
	var req SaveHumanEditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// TODO: Verify user owns the analysis or is admin

	// Get the step ID by analysis ID and step number
	allSteps, err := h.AnalysisByStepsService.GetStepState(c.Request.Context(), analysisID)
	if err != nil {
		h.Logger.Error().
			Err(err).
			Str("analysis_id", analysisID).
			Msg("Failed to get step state")

		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Analysis not found",
		})
		return
	}

	// Find the step with matching step number
	var stepID string
	if allSteps.CurrentStepData != nil && allSteps.CurrentStepData.StepNumber == stepNumber {
		stepID = allSteps.CurrentStepData.ID
	} else {
		for _, prevStep := range allSteps.PreviousSteps {
			if prevStep.StepNumber == stepNumber {
				stepID = prevStep.ID
				break
			}
		}
	}

	if stepID == "" {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Step not found",
		})
		return
	}

	// Call service to save human edit
	result, err := h.AnalysisByStepsService.SaveHumanEdit(c.Request.Context(), stepID, req.EditedContent)
	if err != nil {
		h.Logger.Error().
			Err(err).
			Str("step_id", stepID).
			Int("step_number", stepNumber).
			Msg("Failed to save human edit")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to save edit",
			Message: err.Error(),
		})
		return
	}

	// Convert to response type
	response := buildAnalysisStepResponse(result)
	c.JSON(http.StatusOK, response)
}

// ApproveAndAdvance handles POST /api/v1/analyses/:id/steps/:step/approve
// Approves step and advances to next
// Response: ApproveResponse with approved step + next step info
func (h *AnalysisByStepsHandlers) ApproveAndAdvance(c *gin.Context) {
	// Verify user is authenticated
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Parse path parameters
	analysisID := c.Param("id")
	if analysisID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Analysis ID is required",
		})
		return
	}

	stepStr := c.Param("step")
	stepNumber, err := strconv.Atoi(stepStr)
	if err != nil || stepNumber < 0 || stepNumber > 13 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Step number must be 0-13",
		})
		return
	}

	// TODO: Verify user owns the analysis or is admin

	// Get the step ID by analysis ID and step number
	allSteps, err := h.AnalysisByStepsService.GetStepState(c.Request.Context(), analysisID)
	if err != nil {
		h.Logger.Error().
			Err(err).
			Str("analysis_id", analysisID).
			Msg("Failed to get step state")

		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Analysis not found",
		})
		return
	}

	// Find the step with matching step number
	var stepID string
	if allSteps.CurrentStepData != nil && allSteps.CurrentStepData.StepNumber == stepNumber {
		stepID = allSteps.CurrentStepData.ID
	} else {
		for _, prevStep := range allSteps.PreviousSteps {
			if prevStep.StepNumber == stepNumber {
				stepID = prevStep.ID
				break
			}
		}
	}

	if stepID == "" {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Step not found",
		})
		return
	}

	// Call service to approve and advance
	result, err := h.AnalysisByStepsService.ApproveAndAdvance(c.Request.Context(), stepID)
	if err != nil {
		h.Logger.Error().
			Err(err).
			Str("step_id", stepID).
			Int("step_number", stepNumber).
			Msg("Failed to approve and advance")

		// Check for validation errors (no content)
		if err.Error() == "cannot approve step" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid request",
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to approve step",
			Message: err.Error(),
		})
		return
	}

	// Convert to response type
	response := gin.H{
		"approved_step": buildAnalysisStepResponse(result.ApprovedStep),
		"is_complete":   result.IsComplete,
		"current_step":  result.CurrentStep,
	}

	if result.NextStep != nil {
		nextStepResp := buildAnalysisStepResponse(result.NextStep)
		response["next_step"] = nextStepResp
	}

	c.JSON(http.StatusOK, response)
}

// GetStepState handles GET /api/v1/analyses/:id/steps/state
// Gets current step state with previous steps as context
// Response: StepStateResponse with current step + read-only previous steps
func (h *AnalysisByStepsHandlers) GetStepState(c *gin.Context) {
	// Verify user is authenticated
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Parse path parameter
	analysisID := c.Param("id")
	if analysisID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Analysis ID is required",
		})
		return
	}

	// TODO: Verify user owns the analysis or is admin

	// Call service
	result, err := h.AnalysisByStepsService.GetStepState(c.Request.Context(), analysisID)
	if err != nil {
		h.Logger.Error().
			Err(err).
			Str("analysis_id", analysisID).
			Msg("Failed to get step state")

		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetAnalysisSteps handles GET /api/v1/analyses/:id/steps
// Gets all steps for an analysis
// Response: { "steps": []AnalysisStep }
func (h *AnalysisByStepsHandlers) GetAnalysisSteps(c *gin.Context) {
	// Verify user is authenticated
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Parse path parameter
	analysisID := c.Param("id")
	if analysisID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Analysis ID is required",
		})
		return
	}

	// TODO: Verify user owns the analysis or is admin

	// Get the step state (which includes all steps)
	result, err := h.AnalysisByStepsService.GetStepState(c.Request.Context(), analysisID)
	if err != nil {
		h.Logger.Error().
			Err(err).
			Str("analysis_id", analysisID).
			Msg("Failed to get analysis steps")

		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: err.Error(),
		})
		return
	}

	// Build response with all steps (previous + current)
	allSteps := make([]AnalysisStepResponse, 0, len(result.PreviousSteps)+1)

	for _, step := range result.PreviousSteps {
		allSteps = append(allSteps, buildAnalysisStepResponse(&step))
	}

	if result.CurrentStepData != nil {
		allSteps = append(allSteps, buildAnalysisStepResponse(result.CurrentStepData))
	}

	c.JSON(http.StatusOK, gin.H{
		"steps": allSteps,
	})
}

// buildAnalysisStepResponse converts domain AnalysisStep to API response
func buildAnalysisStepResponse(step *analysisbysteps.AnalysisStep) AnalysisStepResponse {
	resp := AnalysisStepResponse{
		ID:            step.ID,
		AnalysisID:    step.AnalysisID,
		FrameworkCode: step.FrameworkCode,
		StepNumber:    step.StepNumber,
		AIOutput:      step.AIOutput,
		HumanEdited:   step.HumanEdited,
		Visible:       step.Visible,
		Status:        string(step.Status),
		GeneratedAt:   step.GeneratedAt,
		ApprovedAt:    step.ApprovedAt,
		CreatedAt:     step.CreatedAt,
		UpdatedAt:     step.UpdatedAt,
		EffectiveOutput: step.GetEffectiveOutput(),
		IsEdited:        step.IsEdited(),
	}
	return resp
}
