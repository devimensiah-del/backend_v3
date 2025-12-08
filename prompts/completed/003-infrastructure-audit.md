<objective>
Audit the infrastructure packages: jobs/worker, llm, pkg, and adapter directories.

This is Phase 3 of the comprehensive codebase audit. Focus on error handling, retries, resilience, logging clarity, and code organization. Nothing in these packages can break in production.
</objective>

<context>
Read @CLAUDE.md for project conventions.
Read @./docs/audit/PHASE1-SUMMARY.md and @./docs/audit/PHASE2-SUMMARY.md for previous findings.

Packages to audit:
- `jobs/` - Asynq background workers, job handlers
- `llm/` - OpenRouter API client with circuit breaker, prompts
- `pkg/` - Shared packages (errors, logging)
- `adapter/` - External service adapters (if present)
</context>

<audit_checklist>

**Jobs/Worker Audit:**
1. **Correct Logic**
   - Is the job execution flow correct?
   - Are there race conditions or concurrency issues?
   - Is job state properly managed?

2. **Error Handling & Retries**
   - How are job failures handled?
   - Is retry logic with exponential backoff implemented?
   - Are permanent failures distinguished from transient ones?
   - What happens when max retries are exhausted?

3. **Logging**
   - Are logs clear and helpful for debugging?
   - Do logs include job IDs, attempt counts, durations?
   - Can developers trace a job's lifecycle?

4. **Code Organization**
   - Are files well-organized?
   - Is there clear separation of concerns?
   - Can this structure scale with more job types?

**LLM Package Audit:**
1. **Circuit Breaker**
   - Is gobreaker properly configured?
   - What are the failure thresholds?
   - How does recovery work?

2. **Retry & Fallback Logic**
   - Are fallback models properly configured?
   - Is retry logic correct for rate limits?
   - Are timeouts appropriate?

3. **Prompts**
   - Are prompts well-structured in prompts.go?
   - Is there duplication that could be consolidated?
   - Are prompts maintainable?

4. **Error Handling**
   - Are LLM errors properly categorized?
   - Is there clear distinction between retryable and permanent errors?

**Pkg Package Audit:**
1. **errors package**
   - Is the error hierarchy well-designed?
   - Are error constructors clear and consistent?
   - Is the package being used correctly across domains?

2. **logging package**
   - Is the logger properly configured?
   - Are log levels used correctly?
   - Is structured logging consistent?

**Adapter Audit:**
- Review any external service adapters
- Check error handling for external calls
- Verify timeout and retry configurations
</audit_checklist>

<output>
Create audit files:
- `./docs/audit/009-jobs-worker-audit.md`
- `./docs/audit/010-llm-audit.md`
- `./docs/audit/011-pkg-audit.md`
- `./docs/audit/012-adapter-audit.md` (if adapters exist)

Create `./docs/audit/PHASE3-SUMMARY.md` with:
- Infrastructure resilience assessment
- Critical failure points identified
- Logging quality assessment
- Recommendations for production stability
</output>

<constraints>
- Do NOT make code changes - audit only
- Focus on "nothing can break" principle
- Identify single points of failure
- Note any missing monitoring or alerting hooks
</constraints>

<verification>
Before completing:
- All infrastructure packages reviewed
- Error handling paths documented
- Retry mechanisms validated
- Logging assessed for debugging clarity
</verification>
