<objective>
Perform a comprehensive line-by-line analysis of the backend_v3 Go codebase to identify exactly what needs to change across 10 critical areas.

**IMPORTANT**: This is an ANALYSIS-ONLY task. Do NOT write any code, make any edits, or implement any changes. Your deliverable is a detailed audit document that will guide a subsequent big refactor effort.

The analysis must be thorough enough that a developer could take the document and execute all changes without needing additional clarification.
</objective>

<context>
This is the backend_v3 Go REST API for IMENSIAH - a strategic business intelligence platform. The codebase follows Domain-Driven Design with packages in `domain/` for submission, enrichment, analysis, report, company, and macroeconomics.

**Current state issues being addressed:**
- Report domain is being removed (PDF generation moving elsewhere or deprecated)
- Analysis is hardcoded to 11 frameworks but needs to support flexible combinations
- No centralized logging - scattered log statements
- Error handling inconsistent, needs Portuguese-first i18n support
- Various architectural inconsistencies accumulated over time
- Dead code and outdated comments from previous iterations

Read the following files first to understand the project:
@CLAUDE.md
@api/router.go
@config/config.go
</context>

<analysis_areas>

<area_1 title="Report Domain Removal">
**Goal**: Identify everything that needs to be removed/modified when deleting the report domain.

Analyze:
- `domain/report/` - All files and their dependencies
- Database references to report tables in `migrations/`
- API handlers in `api/` that reference reports
- Job handlers in `jobs/` related to reports
- Any imports of report package across the codebase
- Config references to report-related settings (GOTENBERG_URL, etc.)
- Template files in `templates/` if report-specific

Output for each finding:
- File path and line numbers
- What references the report domain
- Recommended action (delete, modify, or keep with changes)
- Dependencies that would break if removed
</area_1>

<area_2 title="Analysis Flexibility Refactor">
**Goal**: Identify changes needed to make analysis support any combination of frameworks instead of hardcoded 11.

Analyze:
- `domain/analysis/model.go` - Current framework definitions
- `domain/analysis/workflow.go` - How frameworks are executed
- `domain/analysis/service.go` - Framework orchestration logic
- `domain/analysis/repository.go` - How framework results are stored
- `llm/prompts.go` - Framework-specific prompts
- Database schema in `migrations/` - JSONB structure for framework results
- Any hardcoded framework lists or switch statements

Output for each finding:
- Current implementation (file:line)
- Why it's inflexible
- Recommended change to support dynamic framework selection
</area_2>

<area_3 title="Frameworks Domain Creation">
**Goal**: Design the new frameworks domain and database table.

Analyze existing patterns:
- How other domains are structured (model.go, repository.go, service.go)
- Current framework definitions scattered in analysis domain
- What metadata each framework needs (name, description, category, dependencies, prompt template, etc.)

Output:
- Proposed database table schema for `frameworks`
- Proposed Go structs for the frameworks domain
- Migration strategy from hardcoded to database-driven frameworks
- API endpoints needed for framework management
- How analysis domain will consume the frameworks domain
</area_3>

<area_4 title="Centralized Logging System">
**Goal**: Design a comprehensive logging system and identify all logging changes needed.

Analyze:
- Current logging patterns across all packages (search for `log.`, `fmt.Print`, `fmt.Errorf`)
- Log levels used (or not used)
- Structured vs unstructured logging
- Request/response logging in API handlers
- Error logging patterns
- Performance/timing logs
- Background job logging in `jobs/`

Output:
- Inventory of current logging statements (file:line, current pattern)
- Recommended logging library and configuration
- Proposed log levels for different scenarios
- Structured logging fields to include (request_id, user_id, submission_id, etc.)
- Changes needed per file to adopt centralized logging
</area_4>

<area_5 title="Error Handling with i18n">
**Goal**: Design error handling that supports Portuguese-primary with English fallback.

Analyze:
- Current error handling patterns (`errors.New`, `fmt.Errorf`, custom error types)
- How errors propagate from domain → service → handler → response
- API error response formats
- Error codes vs error messages
- Where user-facing error messages are constructed

Output:
- Inventory of all error creation points (file:line, current message)
- Proposed error structure with i18n support
- Error code catalog needed
- Portuguese translations required
- How frontend will receive/display errors
- Changes needed per file to adopt i18n errors
</area_5>

<area_6 title="Architecture Consistency">
**Goal**: Identify architectural inconsistencies and standardize patterns.

Analyze:
- Naming conventions: variables, functions, structs, packages
- Dependency direction: who imports whom, circular risks
- Layer boundaries: handler → service → repository patterns
- Transaction handling: consistency across domains
- Interface usage: where interfaces should exist but don't
- Dependency injection patterns

Output:
- Naming inconsistencies with recommended fixes
- Dependency violations with recommended restructuring
- Missing interfaces that should be added
- Transaction handling issues
- Pattern violations with examples and fixes
</area_6>

<area_7 title="Design Consistency">
**Goal**: Ensure consistent design patterns across the codebase.

Analyze:
- Constructor patterns (`New*` functions)
- Option patterns (functional options vs config structs)
- Error wrapping patterns
- Context propagation
- Pointer vs value receivers
- Nil checks and validation patterns
- Comment styles and documentation

Output:
- Inconsistent patterns with file:line references
- Recommended standard for each pattern type
- Files requiring updates to match standards
</area_7>

<area_8 title="API Endpoint Clarity">
**Goal**: Ensure API endpoints are RESTful, clear, and well-documented.

Analyze:
- `api/router.go` - All route definitions
- Handler functions - naming, parameters, responses
- HTTP methods usage (GET/POST/PUT/PATCH/DELETE appropriateness)
- URL path conventions (/api/v1/*, plural vs singular, nesting)
- Query parameter patterns
- Response format consistency
- Error response format consistency
- Authentication/authorization patterns per endpoint

Output:
- Each endpoint with current path, method, and purpose
- Recommended changes for clarity/RESTfulness
- Response format inconsistencies
- Missing or unclear endpoints
- Documentation gaps
</area_8>

<area_9 title="Testing Overhaul">
**Goal**: Ensure tests are lean, consistent, passing, and prevent production breaks.

Analyze:
- Test file locations and naming patterns
- Test coverage per package
- Test types: unit, integration, contract
- Mocking patterns (sqlmock, testify/mock)
- Test data setup and teardown
- Assertion patterns
- Tests that are flaky or skip conditions
- Dead/commented-out tests

Run tests and capture results:
!go test -v -timeout 60s ./... 2>&1

Output:
- Test inventory per package
- Failing tests with reasons
- Inconsistent test patterns
- Missing test coverage for critical paths
- Recommended test structure standards
- Tests to delete, fix, or add
</area_9>

<area_10 title="Environment Variables and Dead Code">
**Goal**: Align config with actual env vars and remove all dead code.

Analyze:
- `config/config.go` - All env var references
- `.env.example` or documentation for expected env vars
- Actual usage of each config value across codebase
- Unused imports in each file
- Unused functions, methods, types, constants
- Commented-out code blocks
- Outdated comments that don't match current code
- Files that are entirely unused

Output:
- Env vars in config not used anywhere
- Env vars referenced but not in config
- Unused code inventory (file:line, what, why it's dead)
- Outdated comments to remove/update
- Files to delete entirely
</area_10>

</analysis_areas>

<output_format>
Create a single comprehensive markdown document at `./analysis/CODEBASE_AUDIT.md` with the following structure:

```markdown
# IMENSIAH Backend v3 - Comprehensive Codebase Audit

**Generated**: [date]
**Scope**: Backend only (backend_v3/)
**Purpose**: Guide for big refactor effort

## Executive Summary
[High-level findings, critical issues, recommended priority order]

## 1. Report Domain Removal
### Files to Delete
### Files to Modify
### Database Changes
### Breaking Changes

## 2. Analysis Flexibility Refactor
### Current Limitations
### Required Changes
### Migration Path

## 3. Frameworks Domain Creation
### Proposed Schema
### Proposed Go Structures
### API Endpoints
### Integration Points

## 4. Centralized Logging System
### Current State
### Proposed Architecture
### File-by-File Changes

## 5. Error Handling with i18n
### Current Patterns
### Proposed Error Structure
### Error Code Catalog
### File-by-File Changes

## 6. Architecture Consistency
### Naming Conventions
### Dependency Issues
### Interface Gaps
### Transaction Handling

## 7. Design Consistency
### Pattern Inventory
### Recommended Standards
### Files Requiring Updates

## 8. API Endpoint Clarity
### Current Endpoints
### Recommended Changes
### Documentation Needs

## 9. Testing Overhaul
### Test Results
### Coverage Analysis
### Recommended Changes
### Test Standards

## 10. Environment Variables and Dead Code
### Env Var Alignment
### Dead Code Inventory
### Files to Delete

## Appendix A: Complete File Inventory
[List of all files examined with status: keep/modify/delete]

## Appendix B: Priority Action Items
[Ordered list of changes for the refactor, grouped by risk/complexity]

## Appendix C: Dependency Graph
[Which areas depend on which - ordering for implementation]
```
</output_format>

<execution_approach>
1. **Read project documentation first**: CLAUDE.md files at all levels
2. **Map the codebase structure**: Use Glob to inventory all files
3. **Read each file systematically**: Go through domain/, api/, jobs/, config/, llm/, migrations/
4. **Run tests**: Capture current test state
5. **Search for patterns**: Use Grep to find logging, error, and other patterns
6. **Cross-reference findings**: Connect issues across areas
7. **Prioritize recommendations**: Based on dependencies and risk

Thoroughly analyze every file. Do not skip or summarize. The goal is a complete inventory that leaves no surprises during implementation.
</execution_approach>

<verification>
Before completing the audit document, verify:
- [ ] Every Go file in backend_v3/ has been read and analyzed
- [ ] All 10 analysis areas have detailed findings
- [ ] File paths and line numbers are accurate
- [ ] Recommendations are actionable (specific enough to implement)
- [ ] Dependencies between changes are mapped
- [ ] Priority order considers dependencies
- [ ] Test results are included
- [ ] No area is left with just "looks fine" - either document what's good or what needs changing
</verification>

<constraints>
- DO NOT write any code or make any edits to source files
- DO NOT create any files except the audit document
- DO NOT skip files or use sampling - read everything
- DO NOT make assumptions about what code does - read and verify
- DO include specific file:line references for every finding
- DO explain WHY each change is needed, not just WHAT
</constraints>
