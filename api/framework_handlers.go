package api

import (
	"encoding/json"
	"net/http"

	"backend_v3/domain/framework"
	jobtypes "backend_v3/jobs/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

// FrameworkHandlers holds all service dependencies for framework handlers
type FrameworkHandlers struct {
	FrameworkService *framework.Service
	AsynqClient      *asynq.Client
	Logger           zerolog.Logger
}

// NewFrameworkHandlers creates a new framework handler set with all dependencies
func NewFrameworkHandlers(
	frameworkSvc *framework.Service,
	asynqClient *asynq.Client,
	logger zerolog.Logger,
) *FrameworkHandlers {
	return &FrameworkHandlers{
		FrameworkService: frameworkSvc,
		AsynqClient:      asynqClient,
		Logger:           logger,
	}
}

// =============================================================================
// Framework CRUD (Admin)
// =============================================================================

// ListFrameworks handles GET /api/v1/admin/frameworks
// Returns all active frameworks ordered by layer
func (h *FrameworkHandlers) ListFrameworks(c *gin.Context) {
	frameworks, err := h.FrameworkService.GetActive(c.Request.Context())
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to list frameworks")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to list frameworks",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"frameworks": frameworks,
		"total":      len(frameworks),
	})
}

// GetFramework handles GET /api/v1/admin/frameworks/:id
// Returns a single framework by ID or code
func (h *FrameworkHandlers) GetFramework(c *gin.Context) {
	idOrCode := c.Param("id")
	ctx := c.Request.Context()

	var fw *framework.Framework
	var err error

	// Try UUID first, then code
	if id, parseErr := uuid.Parse(idOrCode); parseErr == nil {
		fw, err = h.FrameworkService.GetByID(ctx, id)
	} else {
		fw, err = h.FrameworkService.GetByCode(ctx, idOrCode)
	}

	if err != nil {
		h.Logger.Error().Err(err).Str("id_or_code", idOrCode).Msg("Failed to get framework")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get framework",
			Message: err.Error(),
		})
		return
	}

	if fw == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Framework not found",
			Message: "No framework found with ID or code: " + idOrCode,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"framework": fw})
}

// UpdateFrameworkRequest represents the request body for updating a framework
type UpdateFrameworkRequest struct {
	Name               *string                  `json:"name,omitempty"`
	Description        *string                  `json:"description,omitempty"`
	Layer              *int                     `json:"layer,omitempty"`
	IsBase             *bool                    `json:"is_base,omitempty"`
	IsActive           *bool                    `json:"is_active,omitempty"`
	PromptSystem       *string                  `json:"prompt_system,omitempty"`
	PromptUser         *string                  `json:"prompt_user,omitempty"`
	PromptJSONTemplate *string                  `json:"prompt_json_template,omitempty"`
	ModelConfig        *framework.ModelConfig   `json:"model_config,omitempty"`
}

// UpdateFramework handles PUT /api/v1/admin/frameworks/:id
// Updates a framework's configuration
func (h *FrameworkHandlers) UpdateFramework(c *gin.Context) {
	idOrCode := c.Param("id")
	ctx := c.Request.Context()

	var fw *framework.Framework
	var err error

	// Try UUID first, then code
	if id, parseErr := uuid.Parse(idOrCode); parseErr == nil {
		fw, err = h.FrameworkService.GetByID(ctx, id)
	} else {
		fw, err = h.FrameworkService.GetByCode(ctx, idOrCode)
	}

	if err != nil || fw == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Framework not found",
			Message: "No framework found with ID or code: " + idOrCode,
		})
		return
	}

	var req UpdateFrameworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// Apply updates
	if req.Name != nil {
		fw.Name = *req.Name
	}
	if req.Description != nil {
		fw.Description = req.Description
	}
	if req.Layer != nil {
		fw.Layer = *req.Layer
	}
	if req.IsBase != nil {
		fw.IsBase = *req.IsBase
	}
	if req.IsActive != nil {
		fw.IsActive = *req.IsActive
	}
	if req.PromptSystem != nil {
		fw.PromptSystem = req.PromptSystem
	}
	if req.PromptUser != nil {
		fw.PromptUser = *req.PromptUser
	}
	if req.PromptJSONTemplate != nil {
		fw.PromptJSONTemplate = req.PromptJSONTemplate
	}
	if req.ModelConfig != nil {
		fw.ModelConfig = *req.ModelConfig
	}

	if err := h.FrameworkService.Update(ctx, fw); err != nil {
		h.Logger.Error().Err(err).Str("framework_id", fw.ID.String()).Msg("Failed to update framework")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update framework",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"framework": fw})
}

// =============================================================================
// Company Framework Results
// =============================================================================

// GetCompanyFrameworkResults handles GET /api/v1/companies/:id/framework-results
// Returns all framework results for a company
func (h *FrameworkHandlers) GetCompanyFrameworkResults(c *gin.Context) {
	companyIDStr := c.Param("id")
	ctx := c.Request.Context()

	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid company ID",
			Message: "Company ID must be a valid UUID",
		})
		return
	}

	// Optional challenge_id filter
	var challengeID *uuid.UUID
	if challengeIDStr := c.Query("challenge_id"); challengeIDStr != "" {
		if id, err := uuid.Parse(challengeIDStr); err == nil {
			challengeID = &id
		}
	}

	results, err := h.FrameworkService.GetCompanyResultsWithFramework(ctx, companyID, challengeID)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyIDStr).Msg("Failed to get framework results")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get framework results",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(results),
	})
}

// GetFrameworkResultStatus handles GET /api/v1/companies/:id/framework-results/:resultId
// Returns status of a specific framework result (for polling)
func (h *FrameworkHandlers) GetFrameworkResultStatus(c *gin.Context) {
	resultIDStr := c.Param("resultId")
	ctx := c.Request.Context()

	resultID, err := uuid.Parse(resultIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid result ID",
			Message: "Result ID must be a valid UUID",
		})
		return
	}

	result, err := h.FrameworkService.GetResultByID(ctx, resultID)
	if err != nil {
		h.Logger.Error().Err(err).Str("result_id", resultIDStr).Msg("Failed to get framework result")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get framework result",
			Message: err.Error(),
		})
		return
	}

	if result == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Result not found",
			Message: "No framework result found with ID: " + resultIDStr,
		})
		return
	}

	// Get framework details
	fw, _ := h.FrameworkService.GetByID(ctx, result.FrameworkID)

	c.JSON(http.StatusOK, gin.H{
		"result":    result,
		"framework": fw,
	})
}

// UpdateFrameworkResultRequest represents the request to update a framework result
type UpdateFrameworkResultRequest struct {
	Result json.RawMessage `json:"result" binding:"required"`
}

// UpdateFrameworkResult handles PUT /api/v1/companies/:id/framework-results/:resultId
// Allows users to edit the framework result JSON (like enrichment steps)
func (h *FrameworkHandlers) UpdateFrameworkResult(c *gin.Context) {
	companyIDStr := c.Param("id")
	resultIDStr := c.Param("resultId")
	ctx := c.Request.Context()

	// Parse company ID
	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid company ID",
			Message: "Company ID must be a valid UUID",
		})
		return
	}

	// Parse result ID
	resultID, err := uuid.Parse(resultIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid result ID",
			Message: "Result ID must be a valid UUID",
		})
		return
	}

	// Parse request body
	var req UpdateFrameworkResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// Get existing result to verify ownership
	existingResult, err := h.FrameworkService.GetResultByID(ctx, resultID)
	if err != nil {
		h.Logger.Error().Err(err).Str("result_id", resultIDStr).Msg("Failed to get framework result")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get framework result",
			Message: err.Error(),
		})
		return
	}

	if existingResult == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Result not found",
			Message: "No framework result found with ID: " + resultIDStr,
		})
		return
	}

	// Verify company ownership
	if existingResult.CompanyID != companyID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Access denied",
			Message: "Result does not belong to this company",
		})
		return
	}

	// Update the result
	if err := h.FrameworkService.UpdateResult(ctx, resultID, req.Result); err != nil {
		h.Logger.Error().Err(err).Str("result_id", resultIDStr).Msg("Failed to update framework result")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update result",
			Message: err.Error(),
		})
		return
	}

	h.Logger.Info().
		Str("result_id", resultIDStr).
		Str("company_id", companyIDStr).
		Msg("Framework result updated")

	c.JSON(http.StatusOK, gin.H{
		"message":   "Result updated successfully",
		"result_id": resultID.String(),
	})
}

// RunFrameworkRequest represents the request to run a framework
type RunFrameworkRequest struct {
	ChallengeID *string `json:"challenge_id,omitempty"`
}

// RunFrameworkForCompany handles POST /api/v1/companies/:id/frameworks/:code/run
// Triggers async execution of a single framework for a company
// Returns 202 Accepted with result_id for polling
func (h *FrameworkHandlers) RunFrameworkForCompany(c *gin.Context) {
	companyIDStr := c.Param("id")
	frameworkCode := c.Param("code")
	ctx := c.Request.Context()

	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid company ID",
			Message: "Company ID must be a valid UUID",
		})
		return
	}

	// Get framework by code
	fw, err := h.FrameworkService.GetByCode(ctx, frameworkCode)
	if err != nil || fw == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Framework not found",
			Message: "No framework found with code: " + frameworkCode,
		})
		return
	}

	// Parse optional challenge_id
	var req RunFrameworkRequest
	c.ShouldBindJSON(&req) // Ignore error - body is optional

	var challengeID *uuid.UUID
	if req.ChallengeID != nil {
		if id, err := uuid.Parse(*req.ChallengeID); err == nil {
			challengeID = &id
		}
	}

	// Check dependencies are met
	met, missing, err := h.FrameworkService.CheckDependenciesMet(ctx, fw.ID, companyID, challengeID)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to check dependencies")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to check dependencies",
			Message: err.Error(),
		})
		return
	}

	if !met {
		c.JSON(http.StatusPreconditionFailed, ErrorResponse{
			Error:   "Dependencies not met",
			Message: "Missing required frameworks: " + joinStrings(missing),
		})
		return
	}

	// Create pending result
	result, err := h.FrameworkService.CreatePendingResult(ctx, companyID, fw.ID, challengeID)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to create pending result")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create result",
			Message: err.Error(),
		})
		return
	}

	// Enqueue job for async execution
	payload := FrameworkExecutionPayload{
		CompanyID:     companyID.String(),
		FrameworkCode: frameworkCode,
		ResultID:      result.ID.String(),
	}
	if challengeID != nil {
		challengeIDStr := challengeID.String()
		payload.ChallengeID = &challengeIDStr
	}

	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask(jobtypes.TypeFrameworkExecution, payloadBytes)

	if h.AsynqClient != nil {
		_, err = h.AsynqClient.Enqueue(task,
			asynq.MaxRetry(3),
			asynq.Queue("default"),
			asynq.Retention(24*60*60), // 24 hours
		)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Failed to enqueue framework execution job")
			// Don't fail - result is created, worker can retry
		}
	}

	h.Logger.Info().
		Str("company_id", companyIDStr).
		Str("framework_code", frameworkCode).
		Str("result_id", result.ID.String()).
		Msg("Framework execution job enqueued")

	c.JSON(http.StatusAccepted, gin.H{
		"message":   "Framework execution started",
		"result_id": result.ID.String(),
		"status":    result.Status,
		"framework": fw.Code,
	})
}

// FrameworkExecutionPayload is the payload for async framework execution
type FrameworkExecutionPayload struct {
	CompanyID     string  `json:"company_id"`
	FrameworkCode string  `json:"framework_code"`
	ChallengeID   *string `json:"challenge_id,omitempty"`
	ResultID      string  `json:"result_id"`
}

// GetExecutionPlan handles GET /api/v1/admin/frameworks/execution-plan
// Returns the execution plan for all frameworks
func (h *FrameworkHandlers) GetExecutionPlan(c *gin.Context) {
	ctx := c.Request.Context()

	plan, err := h.FrameworkService.BuildExecutionPlan(ctx)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to build execution plan")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to build execution plan",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"execution_plan": plan})
}

// =============================================================================
// Helpers
// =============================================================================

func joinStrings(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += ", " + strs[i]
	}
	return result
}
