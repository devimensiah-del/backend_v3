<objective>
Audit all API handlers and endpoints for correctness, reliability, and alignment with domain business logic.

This is Phase 4 of the comprehensive codebase audit. Verify that handlers correctly implement the business logic documented in domain audits and that all API endpoints are production-ready.
</objective>

<context>
Read @CLAUDE.md for project conventions and API structure.
Read all previous audit summaries in @./docs/audit/ to understand domain business logic.

API structure to audit:
```
api/
├── router.go            # Route setup and composition
├── handlers.go          # Main Handler struct
├── middleware.go        # Auth, CORS, logging, rate limit
├── types.go             # Request/response DTOs
├── helpers.go           # Utility functions
├── security_events.go   # Security audit logging

Domain Handlers (9 files):
├── admin_handlers.go         # 4 handlers
├── analysis_handlers.go      # 10 handlers
├── auth_handlers.go          # 7 handlers
├── company_handlers.go       # 6 handlers
├── framework_handlers.go     # 6 handlers
├── macro_handlers.go         # 4 handlers
├── submission_handlers.go    # 3 handlers
├── user_handlers.go          # 2 handlers
├── wizard_handlers.go        # 6 handlers
```

Route groups:
- `/health` - Health check
- `/api/v1/submit` - Public submission
- `/api/v1/public/report/:code` - Public report via access code
- `/api/v1/auth/*` - Authentication
- `/api/v1/submissions/*` - Protected user routes
- `/api/v1/companies/*` - User's companies
- `/api/v1/admin/*` - Admin-only operations
</context>

<audit_checklist>

**For Each Handler File:**

1. **Business Logic Alignment**
   - Does the handler correctly call domain services?
   - Is the business logic from domain audits properly implemented?
   - Are there any bypasses or shortcuts that violate domain rules?

2. **Request Validation**
   - Are all request parameters validated?
   - Are error messages clear and helpful?
   - Is validation consistent across similar endpoints?

3. **Response Formatting**
   - Are responses consistent in structure?
   - Are HTTP status codes appropriate?
   - Are error responses informative without leaking internals?

4. **Authentication & Authorization**
   - Are protected routes properly secured?
   - Is user context correctly extracted and used?
   - Are admin routes properly restricted?

5. **Error Handling**
   - Are domain errors properly translated to HTTP errors?
   - Is error logging appropriate?
   - Are errors returned to clients helpful?

**For Middleware:**
- Auth middleware correctness
- Rate limiting configuration
- CORS settings for production
- Request logging quality

**For Router:**
- Route organization and clarity
- Middleware application correctness
- No orphaned or dead routes

**Endpoint Reliability:**
- Can each endpoint handle edge cases?
- Are there timeout configurations?
- Is graceful degradation implemented where appropriate?
</audit_checklist>

<output>
Create audit files:
- `./docs/audit/013-handlers-audit.md` (covers all handler files)
- `./docs/audit/014-middleware-audit.md`
- `./docs/audit/015-router-audit.md`
- `./docs/audit/016-api-endpoints-matrix.md` (comprehensive endpoint inventory)

The endpoints matrix should list:
- All endpoints with HTTP method and path
- Authentication requirements
- Related domain service
- Current test coverage
- Production readiness status (ready/needs-work/blocked)

Create `./docs/audit/PHASE4-SUMMARY.md` with:
- Handler quality assessment
- Endpoints requiring attention
- Inconsistencies found
- Security concerns if any
</output>

<constraints>
- Do NOT make code changes - audit only
- Cross-reference with domain audit findings
- Note any handlers with low or no test coverage
- Flag any security concerns immediately
</constraints>

<verification>
Before completing:
- All 9 handler files reviewed
- Middleware thoroughly audited
- Complete endpoint inventory created
- Business logic alignment verified against domain audits
</verification>
