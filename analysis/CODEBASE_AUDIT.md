# IMENSIAH Backend v3 - Comprehensive Codebase Audit

**Generated**: 2025-12-02
**Scope**: Backend only (backend_v3/)
**Purpose**: Guide for big refactor effort

---

## Executive Summary

### Critical Issues (Must Address First)
1. **Report domain removal** requires coordinated changes across 31 files - high complexity
2. **Hardcoded 11 frameworks** in analysis domain blocks product flexibility
3. **Scattered logging** uses zerolog but inconsistently across packages
4. **Error messages are English-only** - no i18n support for Portuguese frontend

### High-Impact Improvements
5. Architecture has some inconsistent patterns (adapters in main.go vs interfaces)
6. API endpoints mostly RESTful but some naming could be clearer
7. Tests pass but coverage is uneven across packages

### Priority Order for Refactor
1. Create frameworks domain & table (foundation for analysis flexibility)
2. Refactor analysis to use dynamic frameworks
3. Remove report domain (depends on ensuring analysis handles PDF-related fields)
4. Implement centralized logging
5. Add i18n error handling
6. Clean up dead code and architectural inconsistencies

---

## 1. Report Domain Removal

### Files to Delete (9 files)
| File | Lines | Purpose | Dependencies |
|------|-------|---------|--------------|
| `domain/report/model.go` | ~120 | Report entity, Status enum | Used by service, repository |
| `domain/report/service.go` | ~250 | Business logic, Publish() | Gotenberg, Supabase storage |
| `domain/report/repository.go` | ~200 | CRUD operations | PostgreSQL |
| `domain/report/async_service.go` | ~80 | Async job enqueueing | Asynq |
| `domain/report/templating.go` | ~300 | HTML template rendering | Templates |
| `domain/report/theme.go` | ~50 | PDF styling | Templating |
| `domain/report/integration_test.go` | ~150 | Integration tests | - |
| `domain/report/service_test.go` | ~200 | Unit tests | - |
| `domain/report/repository_test.go` | ~150 | Repository tests | - |
| `domain/report/analysis_validator_test.go` | ~100 | Validator tests | - |
| `domain/report/templating_test.go` | ~100 | Templating tests | - |

### Files to Modify (22 files)

#### main.go (lines 20, 34-51, 468, 545-554, 564)
**Current:**
```go
import "backend_v3/domain/report"  // line 20

type reportLookupAdapter struct { svc *report.Service }  // lines 34-51
reportRepo := report.NewPostgresRepository(db)  // line 468
reportSvc := report.NewService(...)  // lines 545-554
analysisSvc.SetReportLookup(reportLookupAdapter{svc: reportSvc})  // line 554
worker := jobs.NewWorker(..., reportSvc, ...)  // line 558-565
```
**Action:** Remove report import, adapter, repo, service initialization, and worker parameter

#### jobs/worker.go (lines 14, 34-37, 49, 63-64, 374-436)
**Current:**
```go
import "backend_v3/domain/report"  // line 14
type ReportJobPayload struct { ... }  // lines 34-37
reportService *report.Service  // line 49
w.mux.HandleFunc(jobtypes.TypeReport, w.HandleReportJob)  // line 130
func HandleReportJob() { ... }  // lines 374-436
```
**Action:** Remove import, payload struct, service field, handler registration, handler function

#### jobs/types/types.go
**Current:** `const TypeReport = "report:generate"`
**Action:** Remove TypeReport constant

#### api/router.go (lines ~140-160)
**Current:** Report handlers registered in routes
**Action:** Remove report route group and handler references

#### api/handlers.go (lines 25, 40, 65)
**Current:** `ReportHandlers *ReportHandlers` in Handler struct
**Action:** Remove ReportHandlers field and constructor parameter

#### api/report_handlers.go (entire file)
**Action:** Delete entire file (~200 lines)

#### api/types.go
**Current:** May contain report-related request/response types
**Action:** Remove any report-specific types

#### api/admin_handlers.go
**Current:** May have report-related admin endpoints
**Action:** Remove report admin handlers if present

#### api/submission_response_builder.go (report status references)
**Current:** Includes report_status in responses
**Action:** Remove report status from response building

#### config/config.go (line ~180)
**Current:** `GotenbergURL string` - only used by reports
**Action:** Remove GotenbergURL config (or keep if used elsewhere)

#### infrastructure/gotenberg.go (entire file if report-only)
**Current:** Gotenberg client for PDF generation
**Action:** Evaluate if used elsewhere; if report-only, delete

#### infrastructure/supabase.go (report bucket references)
**Current:** May have report-specific bucket handling
**Action:** Remove report bucket references if dedicated

### Database Changes

#### Migration to create: `032_drop_reports_table.sql`
```sql
-- Drop reports table and related indexes
DROP TABLE IF EXISTS reports CASCADE;
DROP INDEX IF EXISTS idx_reports_submission_id;
DROP INDEX IF EXISTS idx_reports_analysis_id;
```

#### Verify existing migrations referencing reports:
- `003_drop_deprecated_report_columns.sql` - Keep (historical)
- `007_report_indexes_and_constraints.sql` - Reference only, table may not exist
- `028_analyses_pdf_columns.sql` - **CRITICAL**: This moved PDF columns TO analyses table

**Important Finding:** Migration 028 moved `pdf_url`, `pdf_generated_at` from reports to analyses. This means analysis domain already stores PDF info. Report domain may be vestigial.

### Breaking Changes
- API endpoints `/api/v1/reports/*` will be removed
- Background job `report:generate` will no longer be processed
- Admin report management endpoints will be removed
- Frontend must stop calling report endpoints

---

## 2. Analysis Flexibility Refactor

### Current Limitations

#### domain/analysis/model.go (hardcoded framework structs)
**Lines 1-400:** 11 framework structs defined as concrete types:
- `PESTELAnalysis`, `PorterAnalysis`, `SWOTAnalysis`, `OKRAnalysis`
- `TamSamSomAnalysis`, `BenchmarkingAnalysis`, `BlueOceanAnalysis`
- `GrowthHackingAnalysis`, `ScenarioAnalysis`, `BalancedScorecardAnalysis`
- `DecisionMatrixAnalysis`, `AnalysisSynthesis`

**Problem:** Adding/removing frameworks requires code changes and migrations.

#### domain/analysis/workflow.go (fixed execution order)
**Lines 80-250:** `RunAnalysis()` executes frameworks in hardcoded layers:
```go
// Layer 1: Environmental Analysis
runPESTEL(), runPorter(), runTamSamSom()
// Layer 2: Positioning
runSWOT(), runBenchmarking()
// Layer 3: Strategy
runBlueOcean(), runGrowthHacking(), runScenarios()
// Layer 4: Execution
runOKRs(), runBSC(), runDecisionMatrix()
// Final: Synthesis
runSynthesis()
```

**Problem:** Cannot run subset of frameworks or add new ones without code changes.

#### domain/analysis/repository.go (11 JSONB columns)
**Lines 42-65:** INSERT/UPDATE uses 11 separate JSONB columns:
```sql
INSERT INTO analyses (swot, pestel, porter, okrs, tam_sam_som, benchmarking,
                      blue_ocean, growth_hacking, scenarios, bsc, decision_matrix, ...)
```

**Problem:** Schema locked to specific frameworks.

#### llm/prompts.go (framework-specific prompts)
**Entire file:** Contains 11+ prompt templates, one per framework.
**Problem:** Prompts coupled to hardcoded framework names.

### Required Changes

1. **Create generic framework result storage:**
   - Replace 11 JSONB columns with single `framework_results JSONB` map
   - Structure: `{"pestel": {...}, "swot": {...}, ...}`

2. **Make workflow data-driven:**
   - Load framework definitions from database
   - Execute only requested frameworks
   - Respect layer/dependency ordering from framework metadata

3. **Decouple prompts:**
   - Store prompts in frameworks table
   - Load at runtime instead of compile-time

### Migration Path

1. Create frameworks table with definitions (Area 3)
2. Add `framework_results JSONB` column to analyses
3. Migrate existing 11 columns data into framework_results
4. Update repository to use generic column
5. Update service/workflow to be data-driven
6. Drop old 11 JSONB columns in final migration

---

## 3. Frameworks Domain Creation

### Proposed Database Schema

```sql
CREATE TABLE frameworks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,        -- e.g., "pestel", "swot", "porter"
    name VARCHAR(100) NOT NULL,               -- Display name: "PESTEL Analysis"
    name_pt VARCHAR(100) NOT NULL,            -- Portuguese: "Analise PESTEL"
    description TEXT,
    description_pt TEXT,
    category VARCHAR(50) NOT NULL,            -- "environment", "positioning", "strategy", "execution"
    layer_order INTEGER NOT NULL DEFAULT 1,   -- Execution order within category

    -- Execution config
    is_active BOOLEAN DEFAULT true,
    requires_enrichment BOOLEAN DEFAULT true,
    timeout_seconds INTEGER DEFAULT 60,

    -- Prompt configuration
    prompt_template TEXT NOT NULL,            -- LLM prompt with {{variables}}
    output_schema JSONB NOT NULL,             -- JSON Schema for structured output

    -- Model preferences
    preferred_model VARCHAR(50),              -- Override default model
    temperature DECIMAL(3,2) DEFAULT 0.7,

    -- Dependencies
    depends_on TEXT[],                        -- Framework codes that must complete first

    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_frameworks_code ON frameworks(code);
CREATE INDEX idx_frameworks_category ON frameworks(category);
CREATE INDEX idx_frameworks_active ON frameworks(is_active) WHERE is_active = true;
```

### Proposed Go Structures

```go
// domain/framework/model.go
package framework

type Framework struct {
    ID              uuid.UUID
    Code            string          // "pestel", "swot"
    Name            string          // English name
    NamePT          string          // Portuguese name
    Description     *string
    DescriptionPT   *string
    Category        string          // "environment", "positioning", "strategy", "execution"
    LayerOrder      int

    IsActive        bool
    RequiresEnrichment bool
    TimeoutSeconds  int

    PromptTemplate  string
    OutputSchema    json.RawMessage // JSON Schema

    PreferredModel  *string
    Temperature     float64
    DependsOn       []string        // Framework codes

    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type Repository interface {
    GetByCode(ctx context.Context, code string) (*Framework, error)
    List(ctx context.Context, activeOnly bool) ([]*Framework, error)
    ListByCategory(ctx context.Context, category string) ([]*Framework, error)
    Create(ctx context.Context, f *Framework) error
    Update(ctx context.Context, f *Framework) error
    Delete(ctx context.Context, id uuid.UUID) error
}

type Service struct {
    repo Repository
    logger zerolog.Logger
}

func (s *Service) GetExecutionPlan(ctx context.Context, requestedCodes []string) ([]*Framework, error) {
    // Returns frameworks in dependency-resolved order
}
```

### API Endpoints

```
GET    /api/v1/frameworks           - List all frameworks (public)
GET    /api/v1/frameworks/:code     - Get framework by code
POST   /api/v1/admin/frameworks     - Create framework (admin)
PUT    /api/v1/admin/frameworks/:id - Update framework (admin)
DELETE /api/v1/admin/frameworks/:id - Deactivate framework (admin)
```

### Integration with Analysis Domain

```go
// domain/analysis/service.go additions

type FrameworkService interface {
    GetExecutionPlan(ctx context.Context, codes []string) ([]*framework.Framework, error)
}

func (s *Service) RunAnalysis(ctx context.Context, submissionID string, enrichmentID string,
    enrichmentData map[string]interface{}, frameworkCodes []string) (*Analysis, error) {

    // Get execution plan (handles dependencies)
    frameworks, err := s.frameworkService.GetExecutionPlan(ctx, frameworkCodes)

    // Execute each framework dynamically
    results := make(map[string]interface{})
    for _, fw := range frameworks {
        result, err := s.executeFramework(ctx, fw, enrichmentData, results)
        results[fw.Code] = result
    }

    // Store in generic JSONB column
    analysis.FrameworkResults = results
}
```

---

## 4. Centralized Logging System

### Current State

**Library:** zerolog (good choice - structured, fast)
**Pattern:** Inconsistent usage across packages

#### Inventory of Logging Patterns

| Location | Pattern | Issues |
|----------|---------|--------|
| `main.go:31,381-423` | `log.Info().Msg()` | Global logger, good |
| `api/*.go` | `h.logger.Info()` | Handler-scoped, good |
| `domain/analysis/service.go:106` | `s.logger.With().Str("service", "analysis")` | Good sub-logger |
| `domain/enrichment/workflow.go` | `log.Info()` | Uses global, should use service logger |
| `jobs/worker.go:83` | `w.logger.With().Str("component", "worker")` | Good |
| `adapter/macrodata/*.go` | `log.Debug()` | Uses global, no context |

### Recommended Architecture

```go
// pkg/logging/logger.go
package logging

import (
    "os"
    "github.com/rs/zerolog"
)

type Logger struct {
    zerolog.Logger
}

func NewLogger(env string) *Logger {
    level := zerolog.InfoLevel
    if env == "development" {
        level = zerolog.DebugLevel
    }

    logger := zerolog.New(os.Stdout).
        With().
        Timestamp().
        Caller().
        Logger().
        Level(level)

    return &Logger{logger}
}

// Contextual fields all logs should have
func (l *Logger) WithRequestID(requestID string) *Logger {
    return &Logger{l.With().Str("request_id", requestID).Logger()}
}

func (l *Logger) WithSubmissionID(id string) *Logger {
    return &Logger{l.With().Str("submission_id", id).Logger()}
}

func (l *Logger) WithUserID(id string) *Logger {
    return &Logger{l.With().Str("user_id", id).Logger()}
}

func (l *Logger) WithComponent(name string) *Logger {
    return &Logger{l.With().Str("component", name).Logger()}
}
```

### File-by-File Changes

| File | Current | Change |
|------|---------|--------|
| `main.go` | `log.Info()` global | Inject logger into all services |
| `api/middleware.go` | Creates request logger | Add request_id, user_id |
| `domain/*/service.go` | Some use injected logger | All must accept logger in constructor |
| `adapter/macrodata/*.go` | Global `log.Debug()` | Accept logger in provider constructor |
| `jobs/worker.go` | Good pattern | Keep, ensure task_id in all logs |

### Structured Fields Standard

All logs should include (where applicable):
- `request_id` - HTTP request correlation
- `submission_id` - Business entity correlation
- `user_id` - Actor identification
- `component` - Service/package name
- `duration_ms` - Operation timing
- `error` - Error details (if applicable)

---

## 5. Error Handling with i18n

### Current Patterns

#### Error Creation Points (43 files with errors.New or fmt.Errorf)

| Location | Example | Issue |
|----------|---------|-------|
| `domain/analysis/repository.go:67` | `fmt.Errorf("failed to create analysis: %w", err)` | English only, technical |
| `domain/analysis/service.go:648` | `fmt.Errorf("cannot edit analysis while AI is still processing")` | English, user-facing |
| `domain/enrichment/service.go` | Various validation errors | English, some user-facing |
| `api/auth_handlers.go` | `errors.New("invalid credentials")` | User-facing, needs i18n |

### Proposed Error Structure

```go
// pkg/errors/errors.go
package errors

type AppError struct {
    Code       string            // Machine-readable: "ERR_ANALYSIS_PROCESSING"
    MessageEN  string            // English message
    MessagePT  string            // Portuguese message (primary)
    HTTPStatus int               // HTTP status code
    Internal   error             // Wrapped internal error (not exposed)
    Details    map[string]string // Additional context
}

func (e *AppError) Error() string {
    return e.MessageEN // For Go error interface
}

func (e *AppError) LocalizedMessage(lang string) string {
    if lang == "pt" || lang == "pt-BR" {
        return e.MessagePT
    }
    return e.MessageEN
}

// Predefined errors
var (
    ErrAnalysisProcessing = &AppError{
        Code:       "ERR_ANALYSIS_PROCESSING",
        MessageEN:  "Cannot edit analysis while AI is processing",
        MessagePT:  "Nao e possivel editar a analise enquanto a IA esta processando",
        HTTPStatus: 409,
    }

    ErrInvalidCredentials = &AppError{
        Code:       "ERR_INVALID_CREDENTIALS",
        MessageEN:  "Invalid email or password",
        MessagePT:  "Email ou senha invalidos",
        HTTPStatus: 401,
    }

    ErrNotFound = &AppError{
        Code:       "ERR_NOT_FOUND",
        MessageEN:  "Resource not found",
        MessagePT:  "Recurso nao encontrado",
        HTTPStatus: 404,
    }

    // ... more predefined errors
)

// For wrapping with context
func Wrap(base *AppError, internal error) *AppError {
    return &AppError{
        Code:       base.Code,
        MessageEN:  base.MessageEN,
        MessagePT:  base.MessagePT,
        HTTPStatus: base.HTTPStatus,
        Internal:   internal,
    }
}
```

### Error Code Catalog (Initial)

| Code | EN Message | PT Message | HTTP |
|------|------------|------------|------|
| `ERR_VALIDATION` | Validation failed | Falha na validacao | 400 |
| `ERR_UNAUTHORIZED` | Unauthorized access | Acesso nao autorizado | 401 |
| `ERR_FORBIDDEN` | Access denied | Acesso negado | 403 |
| `ERR_NOT_FOUND` | Resource not found | Recurso nao encontrado | 404 |
| `ERR_CONFLICT` | Resource conflict | Conflito de recurso | 409 |
| `ERR_RATE_LIMIT` | Too many requests | Muitas requisicoes | 429 |
| `ERR_INTERNAL` | Internal server error | Erro interno do servidor | 500 |
| `ERR_AI_PROCESSING` | AI is processing | IA esta processando | 409 |
| `ERR_AI_FAILED` | AI processing failed | Processamento da IA falhou | 500 |
| `ERR_ENRICHMENT_PENDING` | Enrichment not ready | Enriquecimento nao esta pronto | 409 |

### API Response Format

```go
// api/helpers.go modification
func RespondError(c *gin.Context, err error) {
    lang := c.GetHeader("Accept-Language") // "pt-BR", "en"

    if appErr, ok := err.(*errors.AppError); ok {
        c.JSON(appErr.HTTPStatus, gin.H{
            "error": gin.H{
                "code":    appErr.Code,
                "message": appErr.LocalizedMessage(lang),
            },
        })
        return
    }

    // Fallback for non-app errors
    c.JSON(500, gin.H{
        "error": gin.H{
            "code":    "ERR_INTERNAL",
            "message": errors.ErrInternal.LocalizedMessage(lang),
        },
    })
}
```

---

## 6. Architecture Consistency

### Naming Conventions

#### Inconsistencies Found

| Location | Issue | Recommendation |
|----------|-------|----------------|
| `main.go:34-51` | `reportLookupAdapter` - lowercase | Use `ReportLookupAdapter` for exported |
| `domain/analysis/service.go:91` | `frameworks map[string]config.FrameworkConfig` | Rename to `modelConfigs` (it's model config, not framework) |
| `api/handlers.go` | `AdminHandlers`, `AnalysisHandlers` | Consistent - good |
| Repository naming | `PostgresRepository`, `NewRepository` | Inconsistent - standardize |

#### Recommended Standards

- **Structs:** PascalCase, noun (`SubmissionService`, `AnalysisRepository`)
- **Interfaces:** PascalCase, "-er" suffix or descriptive (`Repository`, `LLMClient`)
- **Methods:** PascalCase exported, camelCase internal
- **Variables:** camelCase
- **Constants:** PascalCase or SCREAMING_SNAKE for env vars

### Dependency Direction Issues

| Issue | Location | Fix |
|-------|----------|-----|
| main.go has adapter code | lines 34-160 | Move adapters to `adapter/` package |
| Circular risk: analysis ↔ report | ReportLookup interface | Already solved with interface - good |
| enrichment needs company | Via interface | Good pattern |

### Missing Interfaces

| Package | Should Have | Reason |
|---------|-------------|--------|
| `domain/submission` | `Repository` interface | Enable testing with mocks |
| `domain/enrichment` | `Repository` interface | Enable testing with mocks |
| `infrastructure` | `PDFGenerator` interface | Decouple from Gotenberg |
| `infrastructure` | `StorageClient` interface | Decouple from Supabase |

### Transaction Handling

Current pattern (good):
```go
// domain/analysis/repository.go:129
func (r *PostgresRepository) UpdateWithTx(ctx context.Context, tx *sqlx.Tx, analysis *Analysis) error
```

Issue: Not all repositories have `*WithTx` variants.

**Recommendation:** Add transactional methods to all repositories that might participate in multi-entity operations.

---

## 7. Design Consistency

### Constructor Patterns

| Pattern | Usage | Standardize To |
|---------|-------|----------------|
| `NewService(repo, logger)` | Most services | Keep |
| `NewRepository(db)` | All repositories | Keep |
| `NewClient(apiKey)` | LLM client | Keep |
| Late injection (`SetXxx`) | main.go adapters | Reduce - prefer constructor injection |

**Issue:** Too many `SetXxx` methods in `main.go` (lines 484-554):
- `subSvc.SetCompanyService()`
- `enrichSvc.SetMacroService()`
- `enrichSvc.SetCompanyService()`
- `analysisSvc.SetFrameworks()`
- `analysisSvc.SetMacroService()`
- `analysisSvc.SetReportLookup()`

**Recommendation:** Use functional options pattern or builder for complex service initialization.

### Error Wrapping

**Good pattern found:**
```go
return fmt.Errorf("failed to create analysis: %w", err)  // Preserves error chain
```

**Inconsistent pattern:**
```go
return fmt.Errorf("validation failed: %v", err)  // Loses error chain (use %w)
```

### Context Propagation

**Good:** All repository methods accept `context.Context`
**Good:** Jobs use context with timeout
**Issue:** Some adapter methods don't propagate context fully

### Nil Checks

| Location | Pattern | Issue |
|----------|---------|-------|
| `main.go:182-184` | `if comp == nil` | Good nil handling |
| Various services | Missing nil checks before dereferencing | Potential panics |

---

## 8. API Endpoint Clarity

### Current Endpoints (from router.go)

#### Public Routes
| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| GET | `/health` | Health check | Good |
| POST | `/api/v1/submit` | CreateSubmission | Should be `/api/v1/submissions` |
| GET | `/api/v1/public/report/:code` | GetPublicReport | Clear |

#### Auth Routes
| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| POST | `/api/v1/auth/login` | Login | Good |
| POST | `/api/v1/auth/signup` | Signup | Good |
| POST | `/api/v1/auth/logout` | Logout | Good |
| POST | `/api/v1/auth/reset-password` | ResetPassword | Good |

#### Protected User Routes
| Method | Path | Handler | Recommendation |
|--------|------|---------|----------------|
| GET | `/api/v1/submissions` | ListUserSubmissions | Good |
| GET | `/api/v1/submissions/:id` | GetSubmission | Good |
| GET | `/api/v1/submissions/:id/enrichment` | GetEnrichment | Good |
| GET | `/api/v1/submissions/:id/analysis` | GetAnalysis | Good |
| GET | `/api/v1/submissions/:id/report` | GetReport | Remove if report domain removed |

#### Admin Routes
| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| GET | `/api/v1/admin/submissions` | AdminListSubmissions | Good |
| PUT | `/api/v1/admin/enrichments/:id/approve` | ApproveEnrichment | Good |
| POST | `/api/v1/admin/enrichments/:id/re-enrich` | TriggerReEnrich | Consider PATCH |
| POST | `/api/v1/admin/analyses/:id/re-analyze` | TriggerReAnalyze | Consider PATCH |

### Recommended Changes

1. **Rename `/api/v1/submit` to `POST /api/v1/submissions`** - RESTful
2. **Remove report endpoints** if report domain removed
3. **Use PATCH for partial updates** instead of PUT where appropriate
4. **Add versioning documentation** in API responses

### Response Format Consistency

Current responses are mostly consistent:
```json
{
  "data": { ... },
  "meta": { "total": 100, "page": 1 }
}
```

Error responses should be standardized (see Area 5).

---

## 9. Testing Overhaul

### Test Results Summary

```
Tests: PASS
Packages tested: ~15
Total test time: ~3 seconds
```

**All tests pass.** No immediate failures to fix.

### Coverage Analysis by Package

| Package | Test Files | Coverage | Notes |
|---------|------------|----------|-------|
| `adapter/macrodata` | 5 tests | Good | Live API tests (may be flaky) |
| `api` | 8+ tests | Medium | Missing handler unit tests |
| `domain/analysis` | 4+ tests | Medium | Missing workflow tests |
| `domain/enrichment` | Integration test | Low | Needs unit tests |
| `domain/submission` | 3+ tests | Medium | Needs more edge cases |
| `domain/report` | 5+ tests | Good | Will be deleted |
| `domain/company` | 0 tests | **None** | **Critical gap** |
| `domain/macroeconomics` | 0 tests | **None** | **Critical gap** |
| `jobs` | 1 test | Low | Needs handler tests |
| `llm` | 0 tests | **None** | **Critical gap** |

### Critical Test Gaps

1. **`domain/company/`** - No tests, complex business logic
2. **`domain/macroeconomics/`** - No tests, scheduler logic
3. **`llm/`** - No tests, core AI integration
4. **`jobs/worker.go`** - Only utility test, needs handler tests

### Recommended Test Standards

1. **Unit tests:** Test each service method in isolation with mocked dependencies
2. **Integration tests:** Test repository with real database (use test containers)
3. **Contract tests:** Verify API response shapes match frontend expectations
4. **Naming:** `TestServiceName_MethodName_Scenario`

### Tests to Add Priority

1. `domain/company/service_test.go` - Company CRUD, verification logic
2. `domain/macroeconomics/service_test.go` - Indicator fetching, caching
3. `llm/client_test.go` - Mock API responses, retry logic
4. `jobs/worker_test.go` - Job handler logic with mocked services

### Tests to Delete

All `domain/report/*_test.go` files when report domain is removed.

---

## 10. Environment Variables and Dead Code

### Environment Variables Audit

#### config/config.go Variables

| Env Var | Used In | Status |
|---------|---------|--------|
| `PORT` | main.go:648 | Active |
| `ENVIRONMENT` | main.go:392 | Active |
| `DATABASE_URL` | main.go:397 | Active |
| `REDIS_URL` | main.go:427 | Active |
| `REDIS_PASSWORD` | main.go:429 | Active |
| `OPENAI_API_KEY` | main.go:447 | Active (OpenRouter) |
| `GOTENBERG_URL` | main.go:451 | **Remove if report deleted** |
| `SUPABASE_URL` | main.go:455 | Active |
| `SUPABASE_ANON_KEY` | main.go:637 | Active |
| `SUPABASE_SERVICE_ROLE_KEY` | main.go:458 | Active |
| `SUPABASE_JWT_SECRET` | main.go:631 | Active |
| `SUPABASE_BUCKET` | main.go:456 | **Verify - may be report-only** |
| `ALLOWED_ORIGINS` | main.go:631 | Active |
| `WORKER_ENABLED` | main.go:572 | Active |
| `WORKER_CONCURRENCY` | main.go:584 | Active |
| `AI_*_MODEL` | config.go | Active |

#### Variables to Remove (if report deleted)

- `GOTENBERG_URL`
- `SUPABASE_BUCKET` (if only for reports)

### Dead Code Inventory

#### Unused Imports (none found critical)

Most files have clean imports.

#### Potentially Dead Functions

| File | Function | Reason |
|------|----------|--------|
| `api/submission_response_builder.go` | Report status building | Dead if report removed |
| `jobs/worker.go:598-640` | `enqueueJob()` | Not called directly (uses asynqClient) |

#### Outdated Comments

| File | Line | Comment | Issue |
|------|------|---------|-------|
| `domain/analysis/workflow.go:80` | "11 frameworks" | Update when flexible |
| `config/config.go:316` | "4-model approach" | Update to match actual config |
| `main.go:404` | "PRODUCTION FIX" | Remove after fix is stable |

### Files to Delete

| File | Reason |
|------|--------|
| `backend_v3.exe` | Build artifact in gitignore |
| `backend_v3.exe~` | Backup file |
| `backend_v3_test.exe` | Test artifact |
| `backend_v3_verify.exe` | Verification artifact |
| `test_build.exe` | Test artifact |
| `templates/techincal/` | Typo directory (should be "technical") |

---

## Appendix A: Complete File Inventory

### Keep (Core Files)
- `main.go` - Modify (remove report)
- `api/*.go` - Modify (remove report handlers)
- `config/config.go` - Modify (remove Gotenberg if unused)
- `domain/analysis/*` - Modify (add flexibility)
- `domain/enrichment/*` - Keep
- `domain/submission/*` - Keep
- `domain/company/*` - Keep, add tests
- `domain/macroeconomics/*` - Keep, add tests
- `jobs/*.go` - Modify (remove report job)
- `llm/*.go` - Keep, add tests
- `infrastructure/supabase.go` - Keep
- `adapter/macrodata/*` - Keep
- `migrations/*.sql` - Keep, add new

### Modify
- All files referencing report domain
- `domain/analysis/` for flexibility

### Delete
- `domain/report/*` - Entire directory
- `api/report_handlers.go` - File
- `infrastructure/gotenberg.go` - If report-only
- Build artifacts (`*.exe`, `*.exe~`)
- Typo directory (`templates/techincal/`)

### Create
- `domain/framework/*` - New domain
- `migrations/032_frameworks_table.sql`
- `migrations/033_analysis_framework_results.sql`
- `migrations/034_drop_reports_table.sql`
- `pkg/errors/errors.go` - i18n errors
- `pkg/logging/logger.go` - Centralized logging
- Tests for company, macroeconomics, llm

---

## Appendix B: Priority Action Items

### Phase 1: Foundation (Low Risk)
1. Create `domain/framework/` package and table
2. Seed frameworks table with current 11 frameworks
3. Add missing tests for company, macroeconomics, llm
4. Create `pkg/logging/` centralized logger

### Phase 2: Analysis Flexibility (Medium Risk)
5. Add `framework_results JSONB` column to analyses
6. Modify workflow to load frameworks dynamically
7. Migrate existing data to new column
8. Update repository for generic storage

### Phase 3: Report Removal (High Risk)
9. Remove report API endpoints
10. Remove report job handler
11. Remove report service initialization
12. Delete report domain files
13. Create migration to drop reports table
14. Remove Gotenberg config (if unused)

### Phase 4: Error Handling (Low Risk)
15. Create `pkg/errors/` with i18n support
16. Define error code catalog
17. Update API responses to use localized errors
18. Update all services to use AppError

### Phase 5: Cleanup (Low Risk)
19. Remove dead code identified
20. Fix outdated comments
21. Delete build artifacts
22. Standardize naming conventions

---

## Appendix C: Dependency Graph

```
Phase 1 (Foundation)
    └── Phase 2 (Analysis Flexibility)
            └── Phase 3 (Report Removal)
                    └── Phase 5 (Cleanup)

Phase 4 (Error Handling) - Independent, can run parallel
```

**Critical Path:** 1 → 2 → 3 → 5

**Parallel:** Phase 4 can run alongside 1-3

---

*End of Audit Document*
