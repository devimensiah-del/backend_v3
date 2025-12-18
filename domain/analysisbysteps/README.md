# AnalysisBySteps Domain (IAH-2, IAH-3)

Step-by-step analysis execution with human editing capability and AI generation. Replaces the deprecated `wizard` package.

## Key Differences from Wizard

| Feature | Wizard (deprecated) | AnalysisBySteps |
|---------|---------------------|-----------------|
| Human input | Add context → AI regenerates | Direct edit of AI output |
| Storage | `analysis_steps` table | `analysis_steps_v2` table |
| Output field | Single `output` JSONB | `ai_output` + `human_edited` TEXT |
| Effective value | N/A | `COALESCE(human_edited, ai_output)` |
| Framework count | 12 | 14 (+ swotcross, challenge_refinement) |
| Visible default | N/A | `true` |
| GuidanceText | Clarifying questions | Human checkpoint reflection |

## Files

| File | Purpose |
|------|---------|
| `model.go` | `AnalysisStep` struct and `StepStatus` constants |
| `constants.go` | `FrameworkMeta` struct and 14-step `FrameworkOrder` |
| `repository.go` | CRUD + specialized update methods |
| `service.go` | Business logic (start, generate, edit, approve, state) - IAH-3 |
| `types.go` | Response DTOs (`StartResponse`, `ApproveResponse`, `StepStateResponse`) |
| `prompts.go` | Framework code to LLM prompt mapping |

## Model

```go
type AnalysisStep struct {
    ID            string      // UUID
    AnalysisID    string      // FK to analyses.id
    FrameworkCode string      // e.g., "pestel", "swot"
    StepNumber    int         // 0-13
    AIOutput      *string     // JSON string from LLM
    HumanEdited   *string     // Human's edited JSON string
    Visible       bool        // Show in public report (default: true)
    Status        StepStatus  // pending, generating, generated, approved, failed
    GeneratedAt   *time.Time  // When AI generated output
    ApprovedAt    *time.Time  // When human approved
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

### Key Methods

- `GetEffectiveOutput()` - Returns `human_edited` if set, else `ai_output`
- `IsEdited()` - Check if step has human edits
- `IsApproved()` - Check if status == approved

## Framework Order (14 Steps)

```go
var FrameworkOrder = []FrameworkMeta{
    {Code: "challenge_refinement", Name: "Refinamento do Desafio", GuidanceText: "..."},
    {Code: "pestel",               Name: "Análise PESTEL",        GuidanceText: "..."},
    {Code: "porter",               Name: "5 Forças de Porter",    GuidanceText: "..."},
    {Code: "benchmarking",         Name: "Benchmarking",          GuidanceText: "..."},
    {Code: "swot",                 Name: "Análise SWOT",          GuidanceText: "..."},
    {Code: "swotcross",            Name: "SWOT Cruzado",          GuidanceText: "..."},
    {Code: "tam_sam_som",          Name: "TAM-SAM-SOM",           GuidanceText: "..."},
    {Code: "blue_ocean",           Name: "Blue Ocean",            GuidanceText: "..."},
    {Code: "growth_hacking",       Name: "Growth Hacking",        GuidanceText: "..."},
    {Code: "scenarios",            Name: "Cenários",              GuidanceText: "..."},
    {Code: "decision_matrix",      Name: "Matriz de Decisão",     GuidanceText: "..."},
    {Code: "okrs",                 Name: "OKRs",                  GuidanceText: "..."},
    {Code: "bsc",                  Name: "Balanced Scorecard",    GuidanceText: "..."},
    {Code: "synthesis",            Name: "Síntese Executiva",     GuidanceText: "..."},
}
```

### Helper Functions

- `GetStepNumber(frameworkCode)` - Returns step number (0-13) or -1
- `GetFrameworkMeta(frameworkCode)` - Returns `*FrameworkMeta` or nil
- `TotalSteps()` - Returns 14

## Repository Methods

| Method | Purpose |
|--------|---------|
| `Create(ctx, step)` | Insert new step |
| `GetByID(ctx, id)` | Get by UUID |
| `GetByAnalysisID(ctx, analysisID)` | Get all steps ordered by step_number |
| `GetByAnalysisAndFramework(ctx, analysisID, frameworkCode)` | Get specific step |
| `Update(ctx, step)` | Full update |
| `Upsert(ctx, step)` | Insert or update on conflict |
| `SetAIOutput(ctx, id, output)` | Set AI output + status=generated |
| `SetHumanEdited(ctx, id, edited)` | Set human edit only |
| `Approve(ctx, id)` | Set status=approved + approved_at |
| `SetVisibility(ctx, id, visible)` | Toggle visibility |

## Database Table

Created by `v2_019_analysis_steps_by_human.sql`:

```sql
CREATE TABLE IF NOT EXISTS analysis_steps_v2 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analysis_id UUID NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,
    framework_code TEXT NOT NULL,
    step_number INTEGER NOT NULL,
    ai_output TEXT,           -- JSON string from LLM
    human_edited TEXT,        -- Human's edited JSON string
    visible BOOLEAN NOT NULL DEFAULT true,
    status TEXT NOT NULL DEFAULT 'pending',
    generated_at TIMESTAMPTZ,
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(analysis_id, framework_code)
);
```

## Status Flow

```
pending → generating → generated ─┬→ approved
                                   │
                                   └→ (human edits) → approved
```

- `pending`: Step created, AI not yet called
- `generating`: LLM request in progress
- `generated`: AI output ready for review
- `approved`: Human approved (with or without edits)
- `failed`: AI generation failed

## Usage Pattern

```go
// 1. Create step when analysis starts
step := &AnalysisStep{
    AnalysisID:    analysisID,
    FrameworkCode: "pestel",
    StepNumber:    GetStepNumber("pestel"), // 1
    Status:        StatusPending,
}
repo.Create(ctx, step)

// 2. Set AI output after generation
repo.SetAIOutput(ctx, step.ID, jsonOutput)

// 3. Human edits (optional)
repo.SetHumanEdited(ctx, step.ID, editedJson)

// 4. Approve
repo.Approve(ctx, step.ID)

// 5. Get effective output for rendering
output := step.GetEffectiveOutput() // human_edited if set, else ai_output
```

## Frontend Integration

Frontend editors use `react-hook-form` to edit individual framework outputs:

1. Fetch step via `GET /api/v1/analyses/:id/steps/:frameworkCode`
2. Parse `ai_output` or `human_edited` as JSON into form
3. User edits fields
4. Submit via `PUT /api/v1/analyses/:id/steps/:frameworkCode`
5. Backend calls `SetHumanEdited()` with JSON string

## Service Methods (IAH-3)

### StartAnalysisBySteps(ctx, challengeID) → StartResponse
Creates a new step-by-step analysis:
- Fetches challenge to get company_id
- Creates (or reuses) parent `Analysis` record
- Creates all 14 `AnalysisStep` records with `status=pending`
- Returns analysis ID and all steps
- **Idempotent**: Safe to call multiple times

### GenerateStep(ctx, analysisID, stepNumber) → AnalysisStep
Calls LLM to generate AI output for a step:
- Validates step number (0-13)
- Validates all previous steps are approved (enforces sequential flow)
- Sets status to `generating`
- Builds LLM context (company, challenge, macro, previous frameworks)
- Calls LLM with framework-specific prompt and config
- On success: sets `ai_output`, `status=generated`, `generated_at`
- On failure: sets `status=failed`, logs error
- **Preserves** existing `human_edited` (doesn't overwrite)

### SaveHumanEdit(ctx, stepID, editedJSON) → AnalysisStep
Saves human edits without calling LLM:
- Validates JSON structure
- Updates `human_edited` field only
- **Does NOT change status** (stays `generated` or `approved`)

### ApproveAndAdvance(ctx, stepID) → ApproveResponse
Approves a step and advances to next:
- Validates step has content (ai_output OR human_edited)
- Sets `status=approved`, `approved_at=now()`
- If not last step: returns next step info
- If last step (synthesis): marks parent Analysis as `completed`

### GetStepState(ctx, analysisID) → StepStateResponse
Returns current state for UI:
- **Current step**: First non-approved (or last if all approved)
- **Previous steps**: All approved steps (read-only context)
- **Framework metadata**: Guidance text for current step

### GetAnalysisSteps(ctx, analysisID) → []AnalysisStep
Returns all 14 steps ordered by step_number.

## LLM Context Building

Each step receives structured context via `buildStepContext()`:

```go
{
  "company_data": { /* Full company record */ },
  "challenge_context": "The business challenge description",
  "challenge_type": "growth_organic",
  "challenge_category": "growth",
  "macro_context": {
    "economic_indicators": {
      "interest_rate": "15.00% a.a.",       // SELIC
      "inflation_rate": "4.68% (12 meses)", // IPCA
      "exchange_rate": "R$ 5,44/USD",
      "as_of": "2025-12"
    }
  },
  "previous_frameworks": {
    "pestel": { /* JSON output from step 1 */ },
    "porter": { /* JSON output from step 2 */ },
    // ... all approved previous steps
  }
}
```

**Key Rules**:
- Macro indicators are hardcoded (not from database)
- Previous frameworks include only **approved** steps
- Each framework uses its effective output (`human_edited` if exists, else `ai_output`)

## Service Dependencies

The service requires:
- `*Repository` - Database operations
- `analysis.Repository` - To fetch/create parent Analysis records
- `*company.Service` - To get company data for LLM context
- `challenge.Repository` - To get challenge data
- `*llm.Client` - For AI generation
- `map[string]config.FrameworkConfig` - Model routing (primary + fallback)
- `zerolog.Logger` - Structured logging

Initialization in `main.go`:

```go
analysisByStepsSvc := analysisbysteps.NewService(
    analysisByStepsRepo,
    analysisRepo,
    companySvc,
    challengeRepo,
    llmClient,
    cfg.Frameworks,
    log.Logger,
)
```

## Jira Reference

- **IAH-2**: "Domínio e modelo do fluxo por etapas" - Domain package created ✅
- **IAH-3**: "Service e API do fluxo por etapas" - Service layer + API handlers ✅

## API Endpoints (IAH-3)

| Method | Endpoint | Handler | Purpose |
|--------|----------|---------|---------|
| POST | `/api/v1/analyses/steps/start` | `StartAnalysisBySteps` | Start new analysis |
| POST | `/api/v1/analyses/:id/steps/:step/generate` | `GenerateStep` | Generate AI output |
| PUT | `/api/v1/analyses/:id/steps/:step/edit` | `SaveHumanEdit` | Save human edit |
| POST | `/api/v1/analyses/:id/steps/:step/approve` | `ApproveAndAdvance` | Approve and advance |
| GET | `/api/v1/analyses/:id/steps/state` | `GetStepState` | Get current state |
| GET | `/api/v1/analyses/:id/steps` | `GetAnalysisSteps` | Get all steps |

See `docs/API.md` for full frontend integration documentation.
