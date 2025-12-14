# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go REST API backend for IMENSIAH business intelligence platform. Processes company submissions through an AI-powered pipeline: Submission → Enrichment → Analysis (14 strategic frameworks) → PDF Report.

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
├── submission/        # Entry point - company data capture
├── company/           # Company management, verification & enriched data
├── challenge/         # Business challenge management
├── enrichment/        # Stateless enrichment service (Perplexity-only)
├── analysis/          # 14 strategic frameworks execution (batch mode)
├── analysisbysteps/   # Step-by-step analysis with human editing (IAH-2)
└── macroeconomics/    # SELIC, IPCA, USD/BRL indicators
```

**7 domain packages** with consistent patterns:
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
API handlers are split by domain in `api/`:

**Domain Handlers:**
- `admin_handlers.go` - Admin operations
- `analysis_handlers.go` - Analysis CRUD and visibility
- `auth_handlers.go` - Authentication with Supabase
- `company_handlers.go` - Company CRUD + re-analysis
- `macro_handlers.go` - Macroeconomic indicators
- `submission_handlers.go` - Public submission endpoint
- `user_handlers.go` - User profile management

**Infrastructure:**
- `router.go` - Route setup and composition
- `handlers.go` - Main Handler struct
- `middleware.go` - Auth, CORS, logging, rate limit
- `types.go` - Request/response DTOs
- `helpers.go` - Utility functions
- `security_events.go` - Security audit logging
- `submission_response_builder.go` - Status derivation
- `health_handlers.go` - Health check endpoint

Handlers are composed via `NewHandler()` in `router.go`. See `api/README.md` for details.

## Workflow Pipeline

**Status Flow:**
- Submission: Always `received` (never changes)
- Company: `enrichment_status` tracks Perplexity enrichment (inline at creation)
- Analysis: `pending` → `processing` → `completed` (or `failed`)

**Background Jobs (Asynq/Redis):**
- `AnalysisJob` - Executes 14 frameworks sequentially

**Inline Operations:**
- Company enrichment runs synchronously via Perplexity at company creation

## AI Model Configuration

3-model approach via OpenRouter:
1. **PreSearch** (`AI_PRESEARCH_MODEL`): Perplexity sonar-pro for company enrichment (inline)
2. **Primary** (`AI_PRIMARY_MODEL`): All 14 analysis frameworks
3. **Synthesis** (`AI_SYNTHESIS_MODEL`): Premium model for executive summary

Each has a fallback model (`_FALLBACK` suffix) for automatic retry on rate limits.

## Analysis Frameworks (14 Total)

Defined in `domain/analysisbysteps/constants.go`:
- **Step 0**: Challenge Refinement (problem validation)
- **Layer 1** (Environment): PESTEL, Porter's 5 Forces, Benchmarking
- **Layer 2** (Positioning): SWOT, SWOT Cross (cross-quadrant strategies), TAM-SAM-SOM
- **Layer 3** (Strategy): Blue Ocean, Growth Hacking, Scenarios
- **Layer 4** (Execution): Decision Matrix, OKRs, BSC
- **Final**: Synthesis (executive summary)

Framework order and guidance text in `domain/analysisbysteps/constants.go`.
Prompts in `llm/prompts.go`.

## Database

PostgreSQL via Supabase. Key tables:
- `submissions` - Entry data, linked to optional `user_id`
- `companies` - Verified company records with enriched data (includes enrichment_status)
- `analyses` - Framework outputs in `framework_results` JSONB
- `analysis_steps_v2` - Human-editable step storage (IAH-2)

### Migrations Structure
```
migrations/
├── 000_baseline.sql       # Production schema snapshot (001-031)
├── archive/               # Historical reference only (DO NOT RUN)
├── v2_001 - v2_018        # Schema evolution
├── v2_019_analysis_steps_by_human.sql  # IAH-2: analysis_steps_v2 table
└── v2_020_drop_legacy_wizard_tables.sql # Cleanup deprecated tables
```

**For fresh setups:** Run `000_baseline.sql` then `v2_*` migrations.
**For production:** Only run `v2_*` migrations (001-031 already applied).

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
/api/v1/frameworks/order   # Framework metadata (IAH-2)
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

## Jira Tickets

- **IAH-2**: AnalysisBySteps domain + migration (completed)
- **IAH-3**: API handlers for human editing (pending)
