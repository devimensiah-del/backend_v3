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
├── submission/   # Entry point - company data capture
├── enrichment/   # AI data gathering (Perplexity + Gemini)
├── analysis/     # 11 strategic frameworks execution
├── report/       # PDF generation via Gotenberg
├── company/      # Company management & verification
└── macroeconomics/  # SELIC, IPCA, USD/BRL indicators
```

Each domain package follows the pattern:
- `model.go` - Domain entities and value objects
- `repository.go` - Database access with sqlx
- `service.go` - Business logic
- `workflow.go` - Orchestration of complex operations

### Key Files
- `api/router.go` - All HTTP routes and middleware composition
- `config/config.go` - Environment configuration and AI model settings
- `llm/client.go` - OpenRouter API client with retry/fallback logic
- `jobs/worker.go` - Asynq background job handlers

### Handler Composition Pattern
API handlers are split by domain in `api/`:
- `submission_handlers.go`, `enrichment_handlers.go`, `analysis_handlers.go`, `report_handlers.go`
- `admin_handlers.go` - Admin-only operations
- `company_handlers.go` - Company CRUD and verification
- `macro_handlers.go` - Macroeconomic indicators

Handlers are composed via `NewHandler()` in `router.go:120-136`.

## Workflow Pipeline

**Status Flow:**
- Submission: Always `received` (never changes)
- Enrichment: `pending` → `processing` → `completed` → `approved` (or `failed`)
- Analysis: `pending` → `processing` → `completed` (or `failed`)

**Background Jobs (Asynq/Redis):**
- `EnrichmentJob` - Runs Perplexity pre-search + Gemini enrichment
- `AnalysisJob` - Executes 11 frameworks in parallel
- `ReportJob` - Generates PDF via Gotenberg

## AI Model Configuration

6-model approach via OpenRouter (`config/config.go:317-391`):
1. **PreSearch** (`AI_PRESEARCH_MODEL`): Perplexity for company identification
2. **Enrichment** (`AI_ENRICHMENT_MODEL`): Gemini with Google Search
3. **Primary** (`AI_PRIMARY_MODEL`): All 11 analysis frameworks
4. **Synthesis** (`AI_SYNTHESIS_MODEL`): Executive summary (premium model)

Each has a fallback model for automatic retry on rate limits.

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
- `enrichments` - AI-gathered context (one per submission)
- `analyses` - Framework outputs as JSONB columns
- `companies` - Verified company records with field verification
- `macro_indicators` - SELIC, IPCA, USD/BRL snapshots

Migrations in `migrations/` (001-029).

## Testing Conventions

- Unit tests: `*_test.go` alongside source
- Integration tests: `integration_test.go` (require DB)
- Contract tests: `*_contract_test.go` for API/model compatibility
- Mocking: `github.com/DATA-DOG/go-sqlmock` for DB, `testify/mock` for services

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
