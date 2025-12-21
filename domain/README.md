# Domain Layer

Business logic organized by bounded contexts. Each domain is independent with its own models, repository, and service.

## Architecture

```
domain/
├── submission/       # Entry point - form capture
├── company/          # Company records + enrichment
├── challenge/        # Business challenges
├── enrichment/       # Stateless Perplexity client
├── macroeconomics/   # Economic indicators (SELIC, IPCA, USD/BRL)
├── analysis/         # Strategic frameworks execution (batch mode)
└── analysisbysteps/  # Step-by-step analysis with human editing (IAH-2)
```

## Data Flow

```
Submission → Company → Challenge → Analysis
     │           │                    │
     │           └── Enrichment       └── AnalysisBySteps (human editing)
     │               (inline)
     └── Saga rollback on failure
```
---

# Reviewed Domains

## 1. Submission

Entry point for the pipeline. Creates Submission → Company → Challenge via saga pattern.

### Submission Models

| Model | Fields | Notes |
|-------|--------|-------|
| `Submission` | `ID`, `CompanyName`, `CNPJ`, `CompanyWebsite`, `CompanyIndustry`, `CompanySize`, `CompanyLocation`, `ContactName`, `ContactEmail`, `ContactPhone`, `ContactPosition`, `TargetMarket`, `AnnualRevenueMin`, `AnnualRevenueMax`, `FundingStage`, `AdditionalNotes`, `LinkedInURL`, `TwitterHandle`, `UserID`, `CreatedAt`, `UpdatedAt`, `DeletedAt` | Soft delete via `DeletedAt` |
| `SubmitRequest` | Same as above + `ChallengeCategory`, `ChallengeType`, `BusinessChallenge` | API input |
| `SubmitFormResponse` | `SubmissionID`, `CompanyID`, `ChallengeID` | API output |
| `ListOptions` | `Limit`, `Offset`, `Email`, `UserID`, `OrderBy`, `Order` | Query params |
| `CreateFromCompanyInput` | `CompanyID`, company fields, `ContactName`, `ContactEmail` | Re-analyze input |

### Submission Service

| Function | Purpose |
|----------|---------|
| `NewService(repo)` | Constructor |
| `SetCompanyService(svc)` | Inject dependency |
| `SetChallengeService(svc)` | Inject dependency |
| `SubmitForm(ctx, req)` | **Main workflow** - creates all 3 entities with saga rollback |
| `Create(ctx, sub)` | Low-level create (prefer SubmitForm) |
| `GetByID(ctx, id)` | Get by UUID |
| `GetByEmail(ctx, email)` | Get by contact email |
| `List(ctx, opts)` | List with pagination |
| `Delete(ctx, id)` | Soft delete |
| `CreateFromCompany(ctx, input)` | Create from existing company (re-analyze) |
| `LinkAnonymousToUser(ctx, userID, email)` | Link after signup |

### Submission Repository

| Function | SQL Operation |
|----------|---------------|
| `Create(ctx, sub)` | `INSERT INTO submissions` |
| `GetByID(ctx, id)` | `SELECT ... WHERE id = $1 AND deleted_at IS NULL` |
| `GetByEmail(ctx, email)` | `SELECT ... WHERE contact_email = $1` |
| `GetAnonymousByEmail(ctx, email)` | `SELECT ... WHERE user_id IS NULL AND contact_email = $1` |
| `List(ctx, opts)` | `SELECT ... ORDER BY ... LIMIT/OFFSET` |
| `Delete(ctx, id)` | `UPDATE ... SET deleted_at = NOW()` |
| `UpdateUserID(ctx, id, userID)` | `UPDATE ... SET user_id = $1` |
| `WithTx(ctx, fn)` | Transaction wrapper |

### Submission Helpers

| Function | Purpose |
|----------|---------|
| `NewSubmission(name, contact, email, userID)` | Factory with defaults |
| `Validate()` | Field validation |
| `IsDeleted()` | Check soft delete |
| `NewValidationError(field, msg)` | Error factory |
| `NewRepositoryError(op, err)` | Error factory |
| `NewWorkflowError(step, err)` | Error factory |
| `IsNotFound(err)` | Error check |
| `IsValidation(err)` | Error check |

---

## 2. Company

Company records with ownership and one-time enrichment.

### Company Models

| Model | Fields | Notes |
|-------|--------|-------|
| `Company` | `ID`, `Name`, `CNPJ`, `Website`, `Industry`, `CompanySize`, `Location`, `TargetMarket`, `FundingStage`, `AnnualRevenueMin`, `AnnualRevenueMax`, `FoundationYear`, `LegalName`, `Headquarters`, `Sector`, `TargetAudience`, `ValueProposition`, `EmployeesRange`, `RevenueEstimate`, `BusinessModel`, `Competitors`, `MarketShareStatus`, `DigitalMaturity`, `Strengths`, `Weaknesses`, `LinkedInURL`, `TwitterHandle`, `EnrichmentStatus`, `EnrichmentCompletedAt`, `EnrichmentError`, `AllowedUsers`, `OwnerID`, `CreatedAt`, `UpdatedAt` | No soft delete |
| `CompanySubmission` | `CompanyID`, `SubmissionID`, `IsPrimary`, `LinkedAt`, `LinkedBy` | Join table |
| `AnalysisHistoryItem` | `AnalysisID`, `SubmissionID`, `Status`, `BusinessChallenge`, `IsBlurred`, `IsVisibleToUser`, `IsPublic`, `AccessCode`, `PdfUrl`, `CompletedAt`, `CreatedAt`, `UpdatedAt` | Admin view |
| `CreateFromSubmissionInput` | `SubmissionID`, company fields, `OwnerID` | Creation input |
| `StringSlice` | `[]string` | PostgreSQL JSON array |
| `UUIDSlice` | `[]uuid.UUID` | PostgreSQL UUID array |
| `JSONMap` | `map[string]interface{}` | PostgreSQL JSONB |

### Company Service

| Function | Purpose |
|----------|---------|
| `NewService(repo, logger)` | Constructor |
| `SetEnrichmentService(svc)` | Inject dependency |
| `CreateDirect(ctx, input)` | Create without submission link |
| `CreateFromSubmission(ctx, input)` | Create + link + trigger enrichment |
| `GetByID(ctx, id)` | Get by UUID |
| `Delete(ctx, id)` | **Hard delete** (saga rollback only) |
| `GetBySubmissionID(ctx, subID)` | Get by linked submission |
| `LinkSubmission(ctx, compID, subID, isPrimary, linkedBy)` | Link additional submission |
| `GetUserCompanies(ctx, userID)` | Get by owner or allowed_users |
| `ListAll(ctx, limit, offset)` | Admin list |
| `GetAnalysesHistory(ctx, compID)` | Get analyses for company |
| `SetOwnerFromSubmission(ctx, subID, userID)` | Set owner after signup |

### Company Repository

| Function | SQL Operation |
|----------|---------------|
| `Create(ctx, comp)` | `INSERT INTO companies` |
| `GetByID(ctx, id)` | `SELECT ... WHERE id = $1` |
| `GetBySubmissionID(ctx, subID)` | `SELECT ... JOIN company_submissions ... WHERE is_primary = true` |
| `Update(ctx, comp)` | `UPDATE companies SET ...` |
| `Delete(ctx, id)` | `DELETE FROM companies WHERE id = $1` (hard delete) |
| `GetUserCompanies(ctx, userID)` | `SELECT ... WHERE owner_id = $1 OR $1 = ANY(allowed_users)` |
| `ListAll(ctx, limit, offset)` | `SELECT ... LIMIT/OFFSET` |
| `LinkSubmission(ctx, link)` | `INSERT INTO company_submissions ... ON CONFLICT DO UPDATE` |
| `UnlinkSubmissions(ctx, compID)` | `DELETE FROM company_submissions WHERE company_id = $1` |
| `GetAnalysesHistory(ctx, compID)` | `SELECT ... FROM analyses LEFT JOIN challenges ...` |
| `SetEnrichmentProcessing(ctx, id)` | `UPDATE ... SET enrichment_status = 'processing'` |
| `SetEnrichmentCompleted(ctx, id, data)` | `UPDATE ... SET enrichment_status = 'completed', ...` (COALESCE for NULLs) |
| `SetEnrichmentFailed(ctx, id, err)` | `UPDATE ... SET enrichment_status = 'failed', enrichment_error = $1` |
| `WithTx(ctx, fn)` | Transaction wrapper |

### Company Helpers

| Function | Purpose |
|----------|---------|
| `NewCompany(input)` | Factory with defaults |
| `IsOwner(userID)` | Check ownership |
| `CanManageUsers(userID)` | Check if can manage allowed_users |
| `IsUserAllowed(userID)` | Check access (owner OR allowed) |
| `StringSlice.Value/Scan` | PostgreSQL JSON serialization |
| `UUIDSlice.Value/Scan` | PostgreSQL array serialization |
| `UUIDSlice.Contains(id)` | Check if UUID in slice |
| `JSONMap.Value/Scan` | PostgreSQL JSONB serialization |

---

## 3. Challenge

Business challenges linked to companies. Drives analysis context.

**NOTE:** Challenge defines the problem to solve, NOT who requested it. Contact info stays on Submission.

### Challenge Models

| Model | Fields | Notes |
|-------|--------|-------|
| `Challenge` | `ID`, `CompanyID`, `ChallengeCategory`, `ChallengeType`, `BusinessChallenge`, `CreatedAt`, `UpdatedAt`, `DeletedAt` | Soft delete, no contact info |
| `ChallengeCategory` | `growth`, `transform`, `transition`, `compete`, `funding` | Enum (string) |
| `ChallengeType` | `growth_organic`, `growth_geographic`, `transform_digital`, etc. | Enum (string) |

### Challenge Service

| Function | Purpose |
|----------|---------|
| `NewService(repo)` | Constructor |
| `Create(ctx, challenge)` | Create with validation |
| `GetByID(ctx, id)` | Get by UUID |
| `ListByCompany(ctx, compID)` | Get all for company |
| `Update(ctx, challenge)` | Update fields |
| `Delete(ctx, id)` | Soft delete |
| `ValidateCategory(category)` | Check if category string is valid |
| `ValidateType(category, type)` | Check if type is valid for category |

### Challenge Repository

| Function | SQL Operation |
|----------|---------------|
| `Create(ctx, ch)` | `INSERT INTO challenges` |
| `GetByID(ctx, id)` | `SELECT ... WHERE id = $1 AND deleted_at IS NULL` |
| `ListByCompanyID(ctx, compID)` | `SELECT ... WHERE company_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC` |
| `Update(ctx, ch)` | `UPDATE challenges SET ... WHERE deleted_at IS NULL` |
| `Delete(ctx, id)` | `UPDATE ... SET deleted_at = NOW()` |

### Challenge Helpers

| Function | Purpose |
|----------|---------|
| `NewChallenge(compID, category, type, desc)` | Factory with defaults (no contact info) |
| `Validate()` | Field + category/type validation |
| `IsDeleted()` | Check soft delete |
| `IsValidCategory(category)` | **Exported** - Category string check |
| `IsValidType(category, type)` | **Exported** - Type valid for category check |
| `IsValidTypeAny(type)` | **Exported** - Type valid regardless of category |
| `ValidCategories` | **Exported** - Slice of valid categories |
| `ValidTypesByCategory` | **Exported** - Map of category to valid types |

---

## 4. Enrichment

Stateless Perplexity client. No repository - data stored on Company.

### Enrichment Models

| Model | Fields | Notes |
|-------|--------|-------|
| `EnrichedCompanyData` | `CNPJ`, `Website`, `Industry`, `CompanySize`, `Location`, `TargetMarket`, `FundingStage`, `AnnualRevenueMin`, `AnnualRevenueMax`, `FoundationYear`, `LegalName`, `Headquarters`, `Sector`, `TargetAudience`, `ValueProposition`, `EmployeesRange`, `RevenueEstimate`, `BusinessModel`, `MarketShareStatus`, `DigitalMaturity`, `Competitors`, `Strengths`, `Weaknesses`, `LinkedInURL`, `TwitterHandle`, `ConfidenceScore`, `Sources` | Response struct |
| `CompanyInput` | `ID`, `Name`, `CNPJ`, `Website`, `Industry`, `Location` | Input for enrichment |

### Enrichment Service

| Function | Purpose |
|----------|---------|
| `NewService(llmClient, preSearchCfg)` | Constructor |
| `EnrichCompany(ctx, input)` | Call Perplexity, return enriched data |

### Enrichment Storage

**None** - Enrichment is stateless. Data stored on `companies` table by Company domain.

### Enrichment File Structure

```
enrichment/
├── model.go    # EnrichedCompanyData response struct
├── service.go  # Orchestration: EnrichCompany() calls stages
├── prompts.go  # All prompts and prompt builders
└── parser.go   # JSON parsing and normalization
```

### Enrichment Helpers (prompts.go)

| Function | Purpose |
|----------|---------|
| `BuildSearchPrompt(company)` | Build Stage 1 Perplexity search prompt |
| `BuildSynthesisPrompt(company, rawData)` | Build Stage 2 Claude synthesis prompt |

### Enrichment Helpers (parser.go)

| Function | Purpose |
|----------|---------|
| `ParseEnrichmentResponse(content)` | Parse JSON, validate confidence, normalize arrays |

---

## 5. Macroeconomics

Brazilian economic indicators for strategic analysis. DB-first, API-fallback pattern.

### Macroeconomics Models

| Model | Fields | Notes |
|-------|--------|-------|
| `IndicatorType` | `ID`, `Code`, `Name`, `Category`, `Unit`, `Frequency`, `IsActive`, `Metadata` | Indicator definition |
| `DataSource` | `ID`, `Code`, `Name`, `BaseURL`, `Priority`, `IsAuthoritative`, `IsActive` | API source config |
| `IndicatorSource` | `ID`, `IndicatorTypeID`, `SourceID`, `Priority`, `EndpointConfig`, `IsActive`, `LastSuccessAt`, `ConsecutiveFailures` | Indicator-source mapping |
| `IndicatorValue` | `ID`, `IndicatorTypeID`, `SourceID`, `Value`, `EffectiveDate`, `ReferencePeriod`, `RawResponse`, `FetchedAt` | Time-series data |
| `FetchLog` | `ID`, `IndicatorSourceID`, `IndicatorCode`, `SourceCode`, `Status`, `RecordsInserted`, `RecordsUpdated`, `ErrorMessage`, `ResponseTimeMs`, `TriggeredBy` | Audit log |
| `LatestSnapshot` | `Indicators` (map), `AsOf` | All latest values |
| `IndicatorValueSummary` | `Code`, `Name`, `Category`, `Value`, `Unit`, `EffectiveDate`, `ReferencePeriod`, `SourceCode`, `FetchedAt` | Rich value for API |
| `IndicatorHistory` | `IndicatorCode`, `IndicatorName`, `Unit`, `Values`, `From`, `To` | Trend data |

### Macroeconomics Service

| Function | Purpose |
|----------|---------|
| `NewService(repo, macroProvider)` | Constructor |
| `GetLatestSnapshot(ctx)` | **Primary** - Get all latest values (fast DB read) |
| `FetchIndicator(ctx, code)` | Fetch single indicator from API |
| `RefreshAll(ctx)` | Fetch all active indicators |
| `ManualFetch(ctx, code)` | Admin-triggered fetch |
| `GetHistory(ctx, code, from, to)` | Historical values |
| `GetAllIndicatorsWithLatest(ctx)` | All indicators + values |
| `GetIndicatorByCode(ctx, code)` | Get indicator config |
| `GetRecentLogs(ctx, code, limit)` | Audit logs |

### Macroeconomics Repository

| Function | SQL Operation |
|----------|---------------|
| `GetIndicatorByCode(ctx, code)` | `SELECT ... FROM macro_indicator_types WHERE code = $1` |
| `GetActiveIndicators(ctx)` | `SELECT ... WHERE is_active = true` |
| `GetSourceByCode(ctx, code)` | `SELECT ... FROM macro_data_sources WHERE code = $1` |
| `GetActiveSources(ctx)` | `SELECT ... WHERE is_active = true` |
| `GetSourcesForIndicator(ctx, code)` | `SELECT ... JOIN ... ORDER BY priority` |
| `UpdateSourceStatus(ctx, id, success)` | `UPDATE ... SET last_success_at/consecutive_failures` |
| `GetLatestValue(ctx, code)` | `SELECT ... ORDER BY effective_date DESC LIMIT 1` |
| `GetLatestValues(ctx, codes)` | `SELECT DISTINCT ON (code) ...` |
| `GetAllLatestValues(ctx)` | `SELECT DISTINCT ON (code) ... WHERE is_active = true` |
| `InsertValue(ctx, value)` | `INSERT ... ON CONFLICT DO UPDATE` |
| `GetHistory(ctx, code, from, to, limit)` | `SELECT ... WHERE effective_date BETWEEN ...` |
| `LogFetch(ctx, log)` | `INSERT INTO macro_fetch_logs` |
| `GetRecentLogs(ctx, code, limit)` | `SELECT ... ORDER BY created_at DESC LIMIT` |

### Macroeconomics Helpers

| Function | Purpose |
|----------|---------|
| `fetchFromSource(ctx, code, source)` | Route to specific fetcher |
| `fetchSELIC(ctx, source)` | BCB SELIC API |
| `fetchIPCA(ctx, source)` | IBGE IPCA API |
| `fetchUSDoBRL(ctx, source)` | Exchange rate APIs |
| `setEffectiveDate(value, code, raw)` | Parse date from raw response |
| `detailsToSummary(v)` | Convert DB result to API format |
| `LatestSnapshot.HasData()` | Check if snapshot has data |
| `LatestSnapshot.Get(code)` | Get specific indicator |

### Macroeconomics Scheduler

| Function | Purpose |
|----------|---------|
| `NewScheduler(redisOpt, service)` | Constructor (BRT timezone) |
| `RegisterTasks()` | Register cron jobs |
| `Start()` | Begin scheduler |
| `Stop()` | Shutdown scheduler |

---

## 6. Analysis

Strategic analysis with 14 frameworks. Human editing via `analysisbysteps` package (section 7).

See `domain/analysis/README.md` for detailed documentation.

### Analysis Key Points
- Requires `challenge_id` for all analyses (links analysis to specific business problem)
- `RunAnalysis()` executes all 14 frameworks in batch mode
- Results stored in `framework_results` JSONB column
- Human editing via `analysisbysteps` package (IAH-2)

### Analysis Service

| Method | Purpose |
|--------|---------|
| `RunAnalysis(ctx, subID, compID, challengeID)` | Execute all frameworks (batch mode) |
| `GetByID(ctx, id)` | Get analysis by ID |
| `GetBySubmissionID(ctx, subID)` | Get analysis for submission |
| `UpdateStatus(ctx, id, status)` | Update analysis status |

---

## 7. AnalysisBySteps (IAH-2)

Step-by-step analysis with human editing capability.

**Key features:**
- Human can **directly edit** AI output (not just add context for regeneration)
- `ai_output` and `human_edited` stored as TEXT (JSON string)
- `GetEffectiveOutput()` returns `COALESCE(human_edited, ai_output)`
- 14 frameworks (0-13) including `challenge_refinement` and `swotcross`
- `visible` defaults to `true` for public report sections

### AnalysisBySteps Models

| Model | Fields | Notes |
|-------|--------|-------|
| `AnalysisStep` | `ID`, `AnalysisID`, `FrameworkCode`, `StepNumber`, `AIOutput`, `HumanEdited`, `Visible`, `Status`, `GeneratedAt`, `ApprovedAt`, `CreatedAt`, `UpdatedAt` | Table: `analysis_steps_v2` |
| `FrameworkMeta` | `Code`, `Name`, `GuidanceText` | Human checkpoint reflection text |
| `StepStatus` | `pending`, `generating`, `generated`, `approved`, `failed` | Status enum |

### AnalysisBySteps Repository

| Function | SQL Operation |
|----------|---------------|
| `Create(ctx, step)` | `INSERT INTO analysis_steps_v2` |
| `GetByID(ctx, id)` | `SELECT ... WHERE id = $1` |
| `GetByAnalysisID(ctx, analysisID)` | `SELECT ... WHERE analysis_id = $1 ORDER BY step_number` |
| `GetByAnalysisAndFramework(ctx, analysisID, frameworkCode)` | `SELECT ... WHERE analysis_id = $1 AND framework_code = $2` |
| `Update(ctx, step)` | `UPDATE analysis_steps_v2 SET ...` |
| `Upsert(ctx, step)` | `INSERT ... ON CONFLICT (analysis_id, framework_code) DO UPDATE` |
| `SetAIOutput(ctx, id, output)` | Update ai_output + status = generated |
| `SetHumanEdited(ctx, id, edited)` | Update human_edited only |
| `Approve(ctx, id)` | Update status = approved + approved_at |
| `SetVisibility(ctx, id, visible)` | Update visible flag |

### AnalysisBySteps Helpers

| Function | Purpose |
|----------|---------|
| `GetEffectiveOutput()` | Returns `human_edited` if set, else `ai_output` |
| `IsEdited()` | Check if step has human edits |
| `IsApproved()` | Check if status == approved |
| `GetStepNumber(frameworkCode)` | Get step number (0-13) for framework code |
| `GetFrameworkMeta(frameworkCode)` | Get metadata for framework code |
| `TotalSteps()` | Returns 14 (total framework count) |

### Framework Order (14 steps)

| Step | Code | Name | GuidanceText |
|------|------|------|--------------|
| 0 | `challenge_refinement` | Refinamento do Desafio | "Revise se o desafio está claro..." |
| 1 | `pestel` | Análise PESTEL | "Considere quais fatores externos..." |
| 2 | `porter` | 5 Forças de Porter | "Avalie a intensidade competitiva..." |
| 3 | `benchmarking` | Benchmarking | "Os players comparados são relevantes?" |
| 4 | `swot` | Análise SWOT | "As forças listadas geram valor?" |
| 5 | `swotcross` | SWOT Cruzado | "As estratégias cruzadas são viáveis?" |
| 6 | `tam_sam_som` | TAM-SAM-SOM | "O dimensionamento está realista?" |
| 7 | `blue_ocean` | Blue Ocean | "A curva de valor diferencia?" |
| 8 | `growth_hacking` | Growth Hacking | "As táticas são aplicáveis ao estágio?" |
| 9 | `scenarios` | Cenários | "Os cenários cobrem riscos relevantes?" |
| 10 | `decision_matrix` | Matriz de Decisão | "Os critérios refletem prioridades?" |
| 11 | `okrs` | OKRs | "Os objetivos são alcançáveis?" |
| 12 | `bsc` | Balanced Scorecard | "As perspectivas estão balanceadas?" |
| 13 | `synthesis` | Síntese Executiva | "A síntese captura conclusões?" |

### Migration

Table created via `v2_019_analysis_steps_by_human.sql`:
- `analysis_steps_v2` with `UNIQUE(analysis_id, framework_code)`
- `visible BOOLEAN NOT NULL DEFAULT true`
- `ai_output TEXT`, `human_edited TEXT` (JSON strings)

---

## Cross-Domain Interfaces

To avoid import cycles, interfaces are defined in the consuming package:

```go
// submission/types.go
type CompanyServiceInterface interface {
    CreateFromSubmission(ctx, input) (CompanyResult, error)
    SetOwnerFromSubmission(ctx, subID, userID) error
    DeleteCompany(ctx, id) error  // Saga rollback
}

type ChallengeServiceInterface interface {
    CreateFromInput(ctx, input) (uuid.UUID, error)
    DeleteChallenge(ctx, id) error  // Saga rollback
    ValidateCategory(category string) bool  // Validation delegation
    ValidateType(category, type string) bool  // Validation delegation
}
```

Adapters in `main.go` bridge the actual services to these interfaces.

**NOTE:** Challenge validation is centralized in the `challenge` domain. Other domains (like `submission`) use the interface methods to validate without importing the challenge package directly.

---

## Constants

### Challenge Categories & Types

Defined in `challenge/model.go`. Use `challenge.ValidCategories` and `challenge.ValidTypesByCategory` for programmatic access.

| Category | Valid Types |
|----------|-------------|
| `growth` | `growth_organic`, `growth_geographic`, `growth_segment`, `growth_product`, `growth_channel` |
| `transform` | `transform_digital`, `transform_model`, `transform_culture`, `transform_operational` |
| `transition` | `transition_succession`, `transition_exit`, `transition_merger`, `transition_turnaround` |
| `compete` | `compete_differentiate`, `compete_defend`, `compete_reposition` |
| `funding` | `funding_raise`, `funding_debt`, `funding_ipo` |

### Enrichment Status

| Status | Meaning |
|--------|---------|
| `pending` | Company created, enrichment not started |
| `processing` | Enrichment in progress |
| `completed` | Enrichment finished successfully |
| `failed` | Enrichment failed (see `enrichment_error`) |

### Analysis Status

| Status | Meaning |
|--------|---------|
| `pending` | Analysis created, not started |
| `processing` | Frameworks are being executed |
| `completed` | All frameworks completed successfully |
| `failed` | Analysis failed (see error message) |

### AnalysisBySteps Step Status

| Status | Meaning |
|--------|---------|
| `pending` | Step not yet generated |
| `generating` | LLM is running |
| `generated` | Output ready for review |
| `approved` | Human approved step |
| `failed` | AI generation failed |

### Validation Limits

| Constant | Value |
|----------|-------|
| `MaxCompanyNameLength` | 200 |
| `MaxContactNameLength` | 100 |
| `MaxEmailLength` | 254 |
| `MaxPhoneLength` | 50 |
| `MaxBusinessChallengeLength` | 5000 |
| `MaxAdditionalNotesLength` | 10000 |
| `MaxURLLength` | 2048 |

---

## AI Agent Warnings

**DO NOT:**
- Delete or rename `challenge_id` on analyses - it's required
- Remove `enrichment_status` from companies - it tracks async enrichment
- Change framework step order without updating `FrameworkOrder` in `analysisbysteps/constants.go`
- Import domain packages directly - use interfaces in consuming package

**SAFE TO:**
- Add new challenge categories/types (update `types.go`)
- Add new framework steps (update `FrameworkOrder` and prompts)
- Extend enrichment fields (company model + Perplexity prompt)
- Add new macro indicators (via DB config, not code)
- Edit `human_edited` field in `analysis_steps_v2` (user editing is intended)
