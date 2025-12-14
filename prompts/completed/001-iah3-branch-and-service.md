<objective>
Create the IAH-3 feature branch and implement the AnalysisBySteps service layer with all business logic for the step-by-step analysis flow with human editing.

This prompt addresses IAH-3 Jira ticket: "Service e API do fluxo por etapas" - implementing the service layer that orchestrates step-by-step analysis with human-in-the-loop editing capability.
</objective>

<context>
The `analysisbysteps` domain already exists with:
- `model.go` - AnalysisStep struct with AIOutput, HumanEdited, Status fields
- `constants.go` - FrameworkOrder (14 steps), FrameworkMeta, helper functions
- `repository.go` - Full CRUD + SetAIOutput, SetHumanEdited, Approve, Upsert methods
- `service.go` - Empty stub with only NewService constructor

The existing `analysis/workflow.go` shows how LLM calls work with context from previous steps.

Key patterns to follow:
- LLM calls use `llm.GenerateStructuredWithOptions` with fallback models
- Context includes company_data, challenge_data, and previous framework outputs
- Repository pattern with `*sqlx.Tx` for transactions
- Status flow: pending → generating → generated → approved (or failed)
</context>

<research>
Before implementing, read these files to understand patterns:
- `@domain/analysisbysteps/model.go` - AnalysisStep model
- `@domain/analysisbysteps/constants.go` - FrameworkOrder and helpers
- `@domain/analysisbysteps/repository.go` - Repository methods
- `@domain/analysis/workflow.go` - How LLM context is built (lines 600-900)
- `@domain/analysis/service.go` - Service structure
- `@llm/client.go` - GenerateStructuredWithOptions signature
- `@config/config.go` - FrameworkConfig structure
</research>

<requirements>
## 1. Create Feature Branch

Create branch `feature/IAH-3-analysisbysteps-api` from master and switch to it.

## 2. Implement Service Methods in `domain/analysisbysteps/service.go`

### Dependencies
The service needs:
- `repo *Repository` (already exists)
- `analysisRepo analysis.Repository` - To fetch/create Analysis records (use the interface from domain/analysis/repository.go)
- `companyService *company.Service` - To fetch company data for LLM context
- `challengeRepo challenge.Repository` - To fetch challenge data (interface from domain/challenge/repository.go)
- `llmClient *llm.Client` - For generating step outputs
- `frameworks map[string]config.FrameworkConfig` - For model routing
- `logger zerolog.Logger`

**Note**: Macro context is hardcoded (not from a service). Copy `extractMacroContext()` from `analysis/workflow.go:1171-1182`.

### Required Service Methods

```go
// StartAnalysisBySteps creates a new step-by-step analysis with status='draft', current_step=0
// Creates all 14 AnalysisStep records in pending status
func (s *Service) StartAnalysisBySteps(ctx context.Context, challengeID uuid.UUID) (*StartResponse, error)

// GenerateStep calls LLM for current step with context from all previous approved steps
// Preserves existing human_edited if present (doesn't overwrite)
// Sets status to 'generating' before LLM call, then 'generated' on success
func (s *Service) GenerateStep(ctx context.Context, analysisID string, stepNumber int) (*AnalysisStep, error)

// SaveHumanEdit saves human edit without calling LLM
// Only updates human_edited field, does NOT change status
func (s *Service) SaveHumanEdit(ctx context.Context, stepID string, editedJSON string) (*AnalysisStep, error)

// ApproveAndAdvance validates output, sets approved_at, status=approved
// Returns the approved step and next step info
func (s *Service) ApproveAndAdvance(ctx context.Context, stepID string) (*ApproveResponse, error)

// GetStepState returns current step + all previous steps as read-only context
func (s *Service) GetStepState(ctx context.Context, analysisID string) (*StepStateResponse, error)

// GetAnalysisSteps returns all steps for an analysis ordered by step_number
func (s *Service) GetAnalysisSteps(ctx context.Context, analysisID string) ([]AnalysisStep, error)
```

### Response Types (add to model.go or create types.go)

```go
type StartResponse struct {
    AnalysisID    string         `json:"analysis_id"`
    ChallengeID   string         `json:"challenge_id"`
    TotalSteps    int            `json:"total_steps"`
    CurrentStep   int            `json:"current_step"`
    Steps         []AnalysisStep `json:"steps"`
}

type ApproveResponse struct {
    ApprovedStep  *AnalysisStep  `json:"approved_step"`
    NextStep      *AnalysisStep  `json:"next_step,omitempty"`
    IsComplete    bool           `json:"is_complete"`
    CurrentStep   int            `json:"current_step"`
}

type StepStateResponse struct {
    AnalysisID      string          `json:"analysis_id"`
    CurrentStep     int             `json:"current_step"`
    TotalSteps      int             `json:"total_steps"`
    CurrentStepData *AnalysisStep   `json:"current_step_data"`
    PreviousSteps   []AnalysisStep  `json:"previous_steps"` // Read-only context
    FrameworkMeta   *FrameworkMeta  `json:"framework_meta"`
}
```

## 3. Implement LLM Context Building

Follow the pattern from `analysis/workflow.go`:

```go
// buildStepContext builds LLM context for a framework step
// Includes: company_data, challenge_data, macro_context, and all approved previous steps
func (s *Service) buildStepContext(ctx context.Context, analysis *analysis.Analysis, previousSteps []AnalysisStep) (map[string]interface{}, error)
```

Key context fields:
- `company_data` - From companyService.GetByID
- `challenge_context` - Business challenge description
- `challenge_type` - From challenge entity
- `challenge_category` - From challenge entity
- `macro_context` - Hardcoded economic indicators (see workflow.go:1171)
- `previous_frameworks` - Map of frameworkCode → effectiveOutput for all approved steps

## 4. Framework-Specific Prompt Selection

Use the existing prompts from `llm/prompts.go` based on framework code:
- `challenge_refinement` → `llm.ChallengeRefinementPrompt`
- `pestel` → `llm.FrameworkPESTELPrompt`
- `porter` → `llm.FrameworkPorterPrompt`
- etc.

Add a helper function:
```go
func getPromptForFramework(frameworkCode string) string
```
</requirements>

<implementation>
### Service Constructor

```go
func NewService(
    repo *Repository,
    analysisRepo analysis.Repository,      // Interface from domain/analysis/repository.go
    companyService *company.Service,       // Concrete service (has GetByID)
    challengeRepo challenge.Repository,    // Interface from domain/challenge/repository.go
    llmClient *llm.Client,
    frameworks map[string]config.FrameworkConfig,
    logger zerolog.Logger,
) *Service
```

### Macro Context (Hardcoded)

Copy this function from `analysis/workflow.go:1171-1182`:
```go
// extractMacroContext returns hardcoded Brazilian economic indicators for MVP
func (s *Service) extractMacroContext() map[string]interface{} {
    return map[string]interface{}{
        "economic_indicators": map[string]interface{}{
            "interest_rate":  "15.00% a.a.",      // SELIC
            "inflation_rate": "4.68% (12 meses)", // IPCA
            "exchange_rate":  "R$ 5,44/USD",      // Dólar Comercial
            "as_of":          "2025-12",
        },
    }
}
```

### Key Implementation Notes

1. **StartAnalysisBySteps**:
   - Check if analysis already exists for challenge (reuse or create new)
   - Create all 14 AnalysisStep records with status=pending
   - Use transaction for atomicity

2. **GenerateStep**:
   - Validate step number is valid (0-13)
   - Validate all previous steps are approved (can't skip)
   - Set status=generating before LLM call
   - Build context with all approved previous steps
   - Call LLM with framework-specific prompt and config
   - On success: set ai_output, status=generated, generated_at
   - On failure: set status=failed, log error
   - If human_edited exists, preserve it (don't clear)

3. **SaveHumanEdit**:
   - Simply update human_edited field
   - Does NOT change status (stays generated or approved)

4. **ApproveAndAdvance**:
   - Validate step has content (ai_output or human_edited)
   - Set status=approved, approved_at=now()
   - If not last step, return next step info
   - If last step (synthesis), mark analysis as complete

5. **GetStepState**:
   - Find current step (first non-approved, or last if all approved)
   - Return current step data + all previous as read-only
</implementation>

<output>
Create/modify these files:

1. `domain/analysisbysteps/service.go` - Full service implementation
2. `domain/analysisbysteps/types.go` - Response types (StartResponse, ApproveResponse, etc.)
3. `domain/analysisbysteps/prompts.go` - getPromptForFramework helper

Run tests after implementation:
```bash
npm run test:unit -- -run "TestAnalysisBySteps"
```
</output>

<verification>
Before completing, verify:
- [ ] Branch `feature/IAH-3-analysisbysteps-api` created and checked out
- [ ] Service struct has all required dependencies
- [ ] All 5 main service methods implemented
- [ ] Response types defined
- [ ] LLM context building follows workflow.go patterns
- [ ] Prompt selection covers all 14 frameworks
- [ ] Status transitions match: pending → generating → generated → approved
- [ ] Previous step validation prevents skipping
- [ ] Code compiles without errors: `go build ./...`
</verification>

<success_criteria>
- All service methods compile and follow existing patterns
- Status transitions are enforced correctly
- LLM context includes company, challenge, macro, and previous frameworks
- Human edits are preserved when regenerating
- Service can be instantiated with proper dependency injection
</success_criteria>
