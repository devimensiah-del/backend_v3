# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go REST API backend for IMENSIAH business intelligence platform. Processes company submissions through an AI-powered pipeline: Submission → Enrichment → Analysis (11 strategic frameworks) → PDF Report.

## Build & Development Commands

```bash
# Run server
go run main.go

# Build
go build -o backend_v3.exe

# Tests (via Makefile)
make test                    # Unit tests (non-verbose)
make test-verbose            # All tests with verbose output
make test-unit               # Unit tests only
make test-integration        # Integration tests only
make test-coverage           # Generate coverage report
make test-domain PKG=submission  # Test specific domain

# Run single test
go test -v -timeout 30s ./domain/submission/... -run "TestServiceCreate"

# Clean test cache
make clean
go clean -testcache
```

## Architecture

### Domain-Driven Design Structure
```
domain/
├── submission/      # Entry point - company data capture
├── company/         # Company management, verification & enriched data
├── challenge/       # Business challenge management
├── enrichment/      # Stateless enrichment service (Perplexity-only)
├── analysis/        # 11 strategic frameworks execution
├── framework/       # Dynamic framework configuration
├── wizard/          # Human-in-the-loop step-by-step analysis
└── macroeconomics/  # SELIC, IPCA, USD/BRL indicators
```

**8 domain packages** with consistent patterns:
- `model.go` - Domain entities and value objects
- `repository.go` - Database access with sqlx
- `service.go` - Business logic
- `workflow.go` - Orchestration of complex operations (where applicable)
- `README.md` - Domain-specific documentation

### Key Files
- `api/router.go` - All HTTP routes and middleware composition
- `config/config.go` - Environment configuration and AI model settings
- `llm/client.go` - OpenRouter API client with retry/fallback logic
- `jobs/worker.go` - Asynq background job handlers

### Handler Composition Pattern
API handlers are split by domain in `api/` (21 files total):

**Domain Handlers (9 files):**
- `admin_handlers.go` - Admin operations (4 handlers)
- `analysis_handlers.go` - Analysis CRUD and visibility (10 handlers)
- `auth_handlers.go` - Authentication with Supabase (7 handlers)
- `company_handlers.go` - Company CRUD + re-analysis (6 handlers)
- `framework_handlers.go` - Framework management (6 handlers)
- `macro_handlers.go` - Macroeconomic indicators (4 handlers)
- `submission_handlers.go` - Public submission endpoint (3 handlers)
- `user_handlers.go` - User profile management (2 handlers)
- `wizard_handlers.go` - Human-in-the-loop wizard (6 handlers)

**Infrastructure (8 files):**
- `router.go` - Route setup and composition
- `handlers.go` - Main Handler struct
- `middleware.go` - Auth, CORS, logging, rate limit
- `types.go` - Request/response DTOs
- `helpers.go` - Utility functions
- `security_events.go` - Security audit logging
- `submission_response_builder.go` - Status derivation
- `health_handlers.go` - Health check endpoint

**Tests (5 files):**
- `auth_handlers_test.go`, `middleware_test.go`, `user_handlers_test.go`
- `submission_handlers_contract_test.go`, `submission_response_builder_test.go`

Handlers are composed via `NewHandler()` in `router.go`. See `api/README.md` for details.

## Workflow Pipeline

**Status Flow:**
- Submission: Always `received` (never changes)
- Company: `enrichment_status` tracks Perplexity enrichment (inline at creation)
- Analysis: `pending` → `processing` → `completed` (or `failed`)

**Background Jobs (Asynq/Redis):**
- `AnalysisJob` - Executes 11 frameworks in parallel

**Inline Operations:**
- Company enrichment runs synchronously via Perplexity at company creation

## AI Model Configuration

3-model approach via OpenRouter:
1. **PreSearch** (`AI_PRESEARCH_MODEL`): Perplexity sonar-pro for company enrichment (inline)
2. **Primary** (`AI_PRIMARY_MODEL`): All 11 analysis frameworks
3. **Synthesis** (`AI_SYNTHESIS_MODEL`): Premium model for executive summary

Each has a fallback model (`_FALLBACK` suffix) for automatic retry on rate limits.

## Analysis Frameworks (11 Total)

Defined in `domain/analysis/model.go`:
- Layer 1 (Environment): PESTEL, Porter's 7 Forces, TAM-SAM-SOM
- Layer 2 (Positioning): SWOT (with confidence/source), Benchmarking
- Layer 3 (Strategy): Blue Ocean, Growth Hacking, Scenarios
- Layer 4 (Execution): OKRs, BSC, Decision Matrix
- Final: Synthesis (executive summary)

Prompts in `llm/prompts.go`.

## Database

PostgreSQL via Supabase. Key tables:
- `submissions` - Entry data, linked to optional `user_id`
- `companies` - Verified company records with enriched data (includes enrichment_status)
- `analyses` - Framework outputs in `framework_results` JSONB
- `frameworks` - Dynamic framework configuration (v2+)
- `analysis_steps` - Wizard step tracking (v2+)

### Migrations Structure
```
migrations/
├── 000_baseline.sql       # Production schema snapshot (001-031)
├── archive/               # Historical reference only (DO NOT RUN)
│   ├── 01_initial_schema.sql
│   ├── 02_constraints_triggers.sql
│   └── ...
├── v2_001_frameworks_table.sql    # Dynamic frameworks
├── v2_002_framework_results.sql   # Consolidate JSONB
├── v2_003_drop_legacy_columns.sql # Remove old columns
├── v2_004_wizard_system.sql       # Human-in-the-loop
├── v2_005_company_enrichment.sql  # Enriched data → companies
├── v2_006_submission_challenges.sql
└── v2_007_cleanup.sql
```

**For fresh setups:** Run `000_baseline.sql` then `v2_*` migrations.
**For production:** Only run `v2_*` migrations (001-031 already applied).

## Testing Conventions

- Unit tests: `*_test.go` alongside source
- Integration tests: `integration_test.go` (require DB)
- Contract tests: `*_contract_test.go` for API/model compatibility
- Mocking: `github.com/DATA-DOG/go-sqlmock` for DB, `testify/mock` for services

**Current Test Coverage:** 17.7% overall
- High coverage: `enrichment` (68%), `submission` (60.8%), `infrastructure` (69.4%), `pkg/errors` (100%)
- Medium coverage: `challenge` (28.1%), `company` (20.7%), `framework` (37.6%)
- Low coverage: `analysis` (9%), `api` (9.7%), `llm` (14.1%), `wizard` (0%)
- **Priority:** Add tests for wizard, analysis, and API handlers

## API Route Groups

```
/health                     # Health check
/api/v1/submit             # Public submission
/api/v1/public/report/:code # Public report via access code
/api/v1/auth/*             # Login, signup, password reset
/api/v1/submissions/*      # Protected user routes
/api/v1/companies/*        # User's companies
/api/v1/admin/*            # Admin-only operations
```

Auth uses Supabase JWT tokens validated via `SUPABASE_JWT_SECRET`.

## Required Environment Variables

```
DATABASE_URL              # Supabase PostgreSQL
OPENAI_API_KEY            # OpenRouter API key (sk-or-...)
SUPABASE_URL, SUPABASE_ANON_KEY, SUPABASE_JWT_SECRET, SUPABASE_SERVICE_ROLE_KEY
REDIS_ADDR or REDIS_URL   # For Asynq job queue
GOTENBERG_URL             # PDF generation service
```

## Key Patterns

- **Repository transactions**: Use `*sqlx.Tx` parameter for atomic operations
- **LLM calls**: Always use `GenerateStructuredWithOptions` with fallback models
- **Circuit breaker**: LLM client uses gobreaker for fault tolerance
- **Job retries**: Exponential backoff configured in `config.go`
- **Soft deletes**: Entities have `deleted_at` column, use `WHERE deleted_at IS NULL`
