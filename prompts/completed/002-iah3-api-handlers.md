<objective>
Implement HTTP API handlers for the AnalysisBySteps step-by-step analysis flow, following existing handler patterns in the codebase.

This is Part 2 of IAH-3: After the service layer (Prompt 001), implement the API handlers that expose the step-by-step analysis functionality via HTTP endpoints.
</objective>

<context>
The service layer from Prompt 001 provides:
- `StartAnalysisBySteps(ctx, challengeID)` → StartResponse
- `GenerateStep(ctx, analysisID, stepNumber)` → AnalysisStep
- `SaveHumanEdit(ctx, stepID, editedJSON)` → AnalysisStep
- `ApproveAndAdvance(ctx, stepID)` → ApproveResponse
- `GetStepState(ctx, analysisID)` → StepStateResponse

Existing handler patterns:
- `api/analysis_handlers.go` - AnalysisHandlers struct with service dependencies
- `api/company_handlers.go` - Shows handler composition pattern
- `api/types.go` - Request/Response DTOs
- `api/router.go` - Route registration and middleware

Handler convention:
- Handler struct holds service dependencies
- Constructor `NewXxxHandlers(...)` returns pointer
- Methods use `*gin.Context` parameter
- JSON binding with `c.ShouldBindJSON(&req)`
- Error responses use `ErrorResponse{Error, Message}`
</context>

<research>
Read these files to understand handler patterns:
- `@api/analysis_handlers.go` - Existing analysis handler patterns
- `@api/handlers.go` - Main Handler struct composition
- `@api/types.go` - Request/Response type definitions
- `@api/router.go` - How handlers are wired (lines 50-120)
- `@api/middleware.go` - AuthMiddleware pattern
</research>

<requirements>
## 1. Create Handler Struct

Create `api/analysisbysteps_handlers.go`:

```go
type AnalysisByStepsHandlers struct {
    AnalysisByStepsService *analysisbysteps.Service
    Logger                 zerolog.Logger
}

func NewAnalysisByStepsHandlers(
    analysisByStepsSvc *analysisbysteps.Service,
    logger zerolog.Logger,
) *AnalysisByStepsHandlers
```

## 2. Implement HTTP Handlers

### POST /api/v1/analyses/steps/start
Starts a new step-by-step analysis for a challenge.

```go
// StartAnalysisBySteps handles POST /api/v1/analyses/steps/start
// Request: { "challenge_id": "uuid" }
// Response: StartResponse from service
func (h *AnalysisByStepsHandlers) StartAnalysisBySteps(c *gin.Context)
```

Request type:
```go
type StartAnalysisByStepsRequest struct {
    ChallengeID string `json:"challenge_id" binding:"required"`
}
```

**Note**: Gin doesn't have native UUID validator. Validate manually with `uuid.Parse()` in handler.

### POST /api/v1/analyses/:id/steps/:step/generate
Generates AI output for a specific step.

```go
// GenerateStep handles POST /api/v1/analyses/:id/steps/:step/generate
// Path params: id (analysis UUID), step (0-13)
// Response: AnalysisStep with ai_output populated
func (h *AnalysisByStepsHandlers) GenerateStep(c *gin.Context)
```

### PUT /api/v1/analyses/:id/steps/:step/edit
Saves human edit for a step without regenerating.

```go
// SaveHumanEdit handles PUT /api/v1/analyses/:id/steps/:step/edit
// Request: { "edited_content": "json string" }
// Response: Updated AnalysisStep
func (h *AnalysisByStepsHandlers) SaveHumanEdit(c *gin.Context)
```

Request type:
```go
type SaveHumanEditRequest struct {
    EditedContent string `json:"edited_content" binding:"required"`
}
```

### POST /api/v1/analyses/:id/steps/:step/approve
Approves step and advances to next.

```go
// ApproveAndAdvance handles POST /api/v1/analyses/:id/steps/:step/approve
// Response: ApproveResponse with approved step + next step info
func (h *AnalysisByStepsHandlers) ApproveAndAdvance(c *gin.Context)
```

### GET /api/v1/analyses/:id/steps/state
Gets current step state with previous steps as context.

```go
// GetStepState handles GET /api/v1/analyses/:id/steps/state
// Response: StepStateResponse with current step + read-only previous steps
func (h *AnalysisByStepsHandlers) GetStepState(c *gin.Context)
```

### GET /api/v1/analyses/:id/steps
Gets all steps for an analysis.

```go
// GetAnalysisSteps handles GET /api/v1/analyses/:id/steps
// Response: { "steps": []AnalysisStep }
func (h *AnalysisByStepsHandlers) GetAnalysisSteps(c *gin.Context)
```

## 3. Add Request/Response Types to api/types.go

Add these types at the end of the file:

```go
// ==================== ANALYSIS BY STEPS REQUEST TYPES ====================

type StartAnalysisByStepsRequest struct {
    ChallengeID string `json:"challenge_id" binding:"required"` // Validate UUID manually in handler
}

type SaveHumanEditRequest struct {
    EditedContent string `json:"edited_content" binding:"required"`
}

// ==================== ANALYSIS BY STEPS RESPONSE TYPES ====================

// AnalysisStepResponse maps domain AnalysisStep to API response
type AnalysisStepResponse struct {
    ID            string     `json:"id"`
    AnalysisID    string     `json:"analysis_id"`
    FrameworkCode string     `json:"framework_code"`
    StepNumber    int        `json:"step_number"`
    AIOutput      *string    `json:"ai_output,omitempty"`
    HumanEdited   *string    `json:"human_edited,omitempty"`
    Visible       bool       `json:"visible"`
    Status        string     `json:"status"`
    GeneratedAt   *time.Time `json:"generated_at,omitempty"`
    ApprovedAt    *time.Time `json:"approved_at,omitempty"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
    // Computed fields
    EffectiveOutput *string `json:"effective_output,omitempty"`
    IsEdited        bool    `json:"is_edited"`
}
```
</requirements>

<implementation>
### Error Handling Pattern

Follow the existing pattern from analysis_handlers.go:

```go
// UUID Validation (gin doesn't have native uuid validator)
var req StartAnalysisByStepsRequest
if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, ErrorResponse{
        Error:   "Invalid request",
        Message: err.Error(),
    })
    return
}

challengeID, err := uuid.Parse(req.ChallengeID)
if err != nil {
    c.JSON(http.StatusBadRequest, ErrorResponse{
        Error:   "Invalid request",
        Message: "challenge_id must be a valid UUID",
    })
    return
}

// Parameter validation
analysisID := c.Param("id")
if analysisID == "" {
    c.JSON(http.StatusBadRequest, ErrorResponse{
        Error:   "Invalid request",
        Message: "Analysis ID is required",
    })
    return
}

// Step number parsing
stepStr := c.Param("step")
stepNumber, err := strconv.Atoi(stepStr)
if err != nil || stepNumber < 0 || stepNumber > 13 {
    c.JSON(http.StatusBadRequest, ErrorResponse{
        Error:   "Invalid request",
        Message: "Step number must be 0-13",
    })
    return
}

// Service error handling
result, err := h.AnalysisByStepsService.SomeMethod(ctx, params)
if err != nil {
    h.Logger.Error().Err(err).Str("analysis_id", analysisID).Msg("Operation failed")

    // Check for specific error types
    if strings.Contains(err.Error(), "not found") {
        c.JSON(http.StatusNotFound, ErrorResponse{
            Error:   "Not found",
            Message: err.Error(),
        })
        return
    }

    c.JSON(http.StatusInternalServerError, ErrorResponse{
        Error:   "Operation failed",
        Message: err.Error(),
    })
    return
}
```

### Response Building

Create helper to convert domain AnalysisStep to API response:

```go
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
        IsEdited:       step.IsEdited(),
    }
    return resp
}
```

### User Authorization Check

All endpoints should verify user has access to the analysis:

```go
// Get user ID from context (set by AuthMiddleware)
userID, exists := c.Get("userID")
if !exists {
    c.JSON(http.StatusUnauthorized, ErrorResponse{
        Error:   "Unauthorized",
        Message: "User not authenticated",
    })
    return
}

// TODO: Verify user owns the analysis or is admin
// For now, auth check is sufficient (company ownership TBD)
```
</implementation>

<output>
Create/modify these files:

1. `api/analysisbysteps_handlers.go` - New handler file with all 6 endpoints
2. `api/types.go` - Add request/response types at the end

Files to NOT modify yet (will be done in Prompt 003):
- `api/router.go` - Route registration
- `api/handlers.go` - Handler composition
</output>

<verification>
Before completing, verify:
- [ ] Handler struct created with proper constructor
- [ ] All 6 endpoint handlers implemented
- [ ] Request types with proper validation tags
- [ ] Response helper function for AnalysisStep conversion
- [ ] Error handling follows existing patterns
- [ ] Logging matches existing conventions
- [ ] Code compiles: `go build ./api/...`
</verification>

<success_criteria>
- All handlers compile without errors
- Request binding uses proper validation tags
- Error responses match existing ErrorResponse format
- Logging includes relevant context (analysis_id, step_number)
- Response types match service layer types
</success_criteria>
