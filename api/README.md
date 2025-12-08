# API Layer Documentation

**Last Updated:** 2025-12-07 (MVP Simplification)
**Total Files:** 19 Go files
**Purpose:** HTTP handlers, routing, middleware, and request/response handling for IMENSIAH backend

**Recent Changes:**
- ✅ Consolidated company handlers (2 files → 1 file)
- ✅ Moved admin metrics to proper location
- ✅ Removed deprecated endpoints
- ✅ Consistent file naming applied

---

## Table of Contents
1. [Current Structure](#current-structure)
2. [Route Map](#route-map)
3. [Handler Composition](#handler-composition)
4. [Middleware Chain](#middleware-chain)
5. [Issues Found](#issues-found)
6. [Recommended Changes](#recommended-changes)

---

## Current Structure

### Handler Files (by Domain)

| File | Purpose | Domain | Handlers | Status |
|------|---------|--------|----------|--------|
| **admin_handlers.go** | Admin operations + system metrics | Admin | 4 | Active |
| **analysis_handlers.go** | Analysis CRUD and visibility | Analysis | 9 | Active |
| **auth_handlers.go** | Authentication with Supabase | Auth | 7 | Active |
| **company_handlers.go** | Company CRUD + re-analysis + retry-enrichment | Company | 7 | Active |
| **submission_handlers.go** | Public submission endpoint | Submission | 3 | Active |
| **user_handlers.go** | User profile management | User | 2 | Active |
| **wizard_handlers.go** | Human-in-the-loop wizard | Wizard | 7 | Active |

### Infrastructure Files

| File | Purpose | LOC | Notes |
|------|---------|-----|-------|
| **router.go** | Route definitions and setup | 268 | Central router configuration |
| **handlers.go** | Main Handler struct composition | 67 | Orchestrates sub-handlers |
| **middleware.go** | Auth, CORS, logging, rate limit | 333 | 6 middleware functions |
| **health_handlers.go** | Health check endpoint only | 77 | 1 endpoint |
| **helpers.go** | Utility functions | 142 | UUID parsing, error handling |
| **types.go** | Request/response DTOs | 223 | 30+ type definitions |
| **submission_response_builder.go** | Builds submission details | 173 | Status derivation logic |
| **security_events.go** | Security audit logging | 167 | 9 logging functions |

### Test Files

| File | Purpose | Coverage |
|------|---------|----------|
| **auth_handlers_test.go** | Auth handler unit tests | Auth endpoints |
| **middleware_test.go** | Middleware unit tests | Auth/CORS middleware |
| **submission_handlers_contract_test.go** | Contract tests | Submission API |
| **submission_response_builder_test.go** | Builder unit tests | Response building |
| **user_handlers_test.go** | User handler unit tests | User endpoints |

---

## Route Map

### Public Routes (No Auth Required)

| Method | Path | Handler File | Function | Purpose |
|--------|------|--------------|----------|---------|
| GET | `/health` | health_handlers.go | `HealthCheck()` | System health check |
| POST | `/api/v1/submit` | submission_handlers.go | `CreateSubmission()` | Legacy submission endpoint |
| POST | `/api/v1/submissions` | submission_handlers.go | `CreateSubmission()` | Create new submission |
| GET | `/api/v1/public/report/:code` | analysis_handlers.go | `GetPublicReport()` | Public report access |
| POST | `/api/v1/auth/login` | auth_handlers.go | `Login()` | User login |
| POST | `/api/v1/auth/signup` | auth_handlers.go | `Signup()` | User registration |
| POST | `/api/v1/auth/forgot-password` | auth_handlers.go | `ForgotPassword()` | Request password reset |
| POST | `/api/v1/auth/reset-password` | auth_handlers.go | `ResetPassword()` | Reset password with token |

### Protected Routes (Auth Required)

| Method | Path | Handler File | Function | Purpose |
|--------|------|--------------|----------|---------|
| GET | `/api/v1/auth/me` | auth_handlers.go | `GetCurrentUser()` | Get current user profile |
| POST | `/api/v1/auth/logout` | auth_handlers.go | `Logout()` | Logout user |
| PUT | `/api/v1/auth/update-password` | auth_handlers.go | `UpdatePassword()` | Update password |
| GET | `/api/v1/user/profile` | auth_handlers.go | `GetCurrentUser()` | Alias for /auth/me |
| PUT | `/api/v1/user/profile` | user_handlers.go | `UpdateUserProfile()` | Update user profile |
| DELETE | `/api/v1/user` | user_handlers.go | `DeleteAccount()` | Deactivate account |
| GET | `/api/v1/submissions` | submission_handlers.go | `ListUserSubmissions()` | List user's submissions |
| GET | `/api/v1/submissions/:id` | submission_handlers.go | `GetSubmission()` | Get submission details |
| GET | `/api/v1/submissions/:id/analysis` | analysis_handlers.go | `GetAnalysis()` | Get analysis for submission |
| POST | `/api/v1/companies` | company_handlers.go | `CreateCompany()` | Create company directly |
| GET | `/api/v1/companies` | company_handlers.go | `GetMyCompanies()` | List user's companies |
| GET | `/api/v1/companies/:id` | company_handlers.go | `GetCompany()` | Get company details |
| GET | `/api/v1/challenges/types` | Inline handler | Returns challenge types | Challenge metadata |
| GET | `/api/v1/frameworks/order` | Inline handler | Returns framework order | Wizard metadata |
| POST | `/api/v1/wizard/start` | wizard_handlers.go | `StartWizard()` | Start wizard (company_id + challenge_id in body) |
| GET | `/api/v1/analyses/:id/wizard` | wizard_handlers.go | `GetWizardState()` | Get wizard state |
| POST | `/api/v1/analyses/:id/wizard/generate` | wizard_handlers.go | `GenerateStep()` | Generate wizard step |
| POST | `/api/v1/analyses/:id/wizard/approve` | wizard_handlers.go | `ApproveStep()` | Approve wizard step |
| POST | `/api/v1/analyses/:id/wizard/refine` | wizard_handlers.go | `RefineStep()` | Refine wizard step |
| GET | `/api/v1/analyses/:id/wizard/summary` | wizard_handlers.go | `GetWizardSummary()` | Get wizard summary |

### Admin Routes (Auth + Admin Role Required)

| Method | Path | Handler File | Function | Purpose |
|--------|------|--------------|----------|---------|
| GET | `/api/v1/admin/submissions` | admin_handlers.go | `ListSubmissions()` | List all submissions |
| GET | `/api/v1/admin/submissions/:id` | admin_handlers.go | `GetSubmissionAdmin()` | Get submission (admin) |
| GET | `/api/v1/admin/submissions/:id/analysis` | analysis_handlers.go | `GetAnalysisBySubmissionAdmin()` | Get analysis (admin) |
| POST | `/api/v1/admin/submissions/:id/retry-analysis` | admin_handlers.go | `RetryAnalysis()` | Retry analysis job |
| GET | `/api/v1/admin/metrics` | admin_handlers.go | `GetMetrics()` | System metrics |
| GET | `/api/v1/admin/analysis/:id` | analysis_handlers.go | `GetAnalysisAdmin()` | Get analysis by ID |
| PUT | `/api/v1/admin/analysis/:id` | analysis_handlers.go | `UpdateAnalysis()` | Update analysis fields |
| POST | `/api/v1/admin/analysis/:id/visibility` | analysis_handlers.go | `ToggleVisibility()` | Toggle visibility |
| POST | `/api/v1/admin/analysis/:id/public` | analysis_handlers.go | `TogglePublic()` | Toggle public access |
| POST | `/api/v1/admin/analysis/:id/access-code` | analysis_handlers.go | `GenerateAccessCode()` | Generate access code |
| POST | `/api/v1/admin/analysis/:id/wizard/generate-all` | wizard_handlers.go | `GenerateAllSteps()` | Bulk generate all frameworks |
| GET | `/api/v1/admin/companies` | company_handlers.go | `ListAllCompanies()` | List all companies |
| GET | `/api/v1/admin/companies/:id` | company_handlers.go | `GetCompanyAdmin()` | Get company (admin) |
| POST | `/api/v1/admin/companies/:id/re-analyze` | company_handlers.go | `ReAnalyzeCompany()` | Re-run analysis with new challenge |
| POST | `/api/v1/admin/companies/:id/retry-enrichment` | company_handlers.go | `RetryEnrichment()` | Re-run enrichment (fill gaps only) |

---

## Handler Composition

The API layer uses a **composition pattern** to organize handlers:

```go
// Main Handler struct (handlers.go)
type Handler struct {
    // Core dependencies
    db                *sqlx.DB
    redisClient       *redis.Client
    logger            zerolog.Logger
    supabaseURL       string
    supabaseAnonKey   string
    supabaseJWTSecret string

    // Composed Handlers (sub-packages)
    AdminHandlers      *AdminHandlers
    AnalysisHandlers   *AnalysisHandlers
    AuthHandlers       *AuthHandlers
    CompanyHandlers    *CompanyHandlers
    SubmissionHandlers *SubmissionHandlers
    UserHandlers       *UserHandlers

    // Helper for building detailed responses
    SubmissionResponseBuilder *SubmissionResponseBuilder
}
```

### Initialization Flow (router.go)

1. **Create specialized handlers** (lines 58-114):
   - `SubmissionResponseBuilder` - Composes analysis/company/challenge data
   - `AdminHandlers` - Admin operations
   - `AnalysisHandlers` - Analysis management
   - `AuthHandlers` - Authentication
   - `SubmissionHandlers` - Public submissions
   - `UserHandlers` - User profiles
   - `CompanyHandlers` - Company CRUD, re-analysis, retry-enrichment
   - `WizardHandlers` - Human-in-the-loop workflow

2. **Create main Handler** (lines 118-133):
   - Composes all specialized handlers
   - Shares common dependencies (DB, Redis, Logger)

3. **Register routes** (lines 136-264):
   - Uses `mainHandler.SubHandler.Method()` pattern
   - Example: `mainHandler.AnalysisHandlers.GetAnalysis()`

### Benefits of This Pattern

- **Separation of Concerns**: Each domain has its own handler file
- **Testability**: Handlers can be tested in isolation
- **Dependency Injection**: Services injected at initialization
- **Scalability**: Easy to add new handler groups

---

## Middleware Chain

Middleware is applied in this order (router.go lines 49-54):

1. **CORSMiddleware** (middleware.go:20-84)
   - Handles CORS preflight requests
   - Special handling for `/health` endpoint (allows all origins)
   - Rejects non-GET requests without Origin header
   - Configured via `allowedOrigins` parameter

2. **RequestIDMiddleware** (middleware.go:325-332)
   - Adds `X-Request-ID` header for tracing
   - Sets `request_id` in Gin context

3. **LoggingMiddleware** (middleware.go:226-269)
   - Logs all HTTP requests with detailed context
   - Logs START and COMPLETION events
   - Color-coded by status code (Info/Warn/Error)

4. **RecoveryMiddleware** (middleware.go:272-288)
   - Recovers from panics
   - Logs panic details with stack trace
   - Returns 500 Internal Server Error

5. **RateLimitMiddleware** (middleware.go:291-322)
   - Limits requests per IP (default: 100 req/min)
   - Uses `golang.org/x/time/rate` token bucket
   - Cleanup goroutine prevents memory leak

### Route-Specific Middleware

- **AuthMiddleware** (middleware.go:153-179)
  - Validates Supabase JWT tokens
  - Fetches user role from database
  - Sets `userID`, `userEmail`, `userRole` in context
  - Returns 401 Unauthorized on failure

- **OptionalAuthMiddleware** (middleware.go:183-189)
  - Parses token if present but doesn't require it
  - Used for public endpoints that need admin preview mode
  - Example: `/api/v1/public/report/:code`

- **AdminAuthMiddleware** (middleware.go:192-223)
  - Checks if user has admin/super_admin/service_role
  - Returns 403 Forbidden if insufficient privileges
  - Logs authorization failures for audit

---

## Issues Found

### 1. File Organization Issues

#### ~~1.1 Company Handlers Split Across Two Files~~ ✅ FIXED
- **Status**: RESOLVED - Merged into `company_handlers.go`
- Files `company_crud_handlers.go` and `company_analysis_handlers.go` have been consolidated
- Single `CompanyHandlers` struct with 6 methods (CRUD + re-analysis)

#### ~~1.2 Inconsistent Handler Naming~~ ✅ FIXED
- **Status**: RESOLVED - Renamed to `company_handlers.go`
- All handler files now follow `{domain}_handlers.go` pattern

#### ~~1.3 Health Handlers Mixed Concerns~~ ✅ FIXED
- **Status**: RESOLVED - `GetMetrics()` moved to `admin_handlers.go`
- `health_handlers.go` now only contains public health check endpoint
- System metrics are properly scoped as admin-only

### 2. Deprecated/Dead Code

#### ~~2.1 Deprecated Endpoints~~ ✅ FIXED
- **Status**: RESOLVED - Deprecated endpoints removed
- `POST /api/v1/admin/submissions/:id/retry-enrichment` - REMOVED (enrichment is automatic)
- `GET /api/v1/admin/analytics` - REMOVED (needs rebuild with new workflow)
- Routes cleaned from router.go with explanatory comments

#### 2.2 Removed Routes (Comments Only)
- `PUT /api/v1/admin/submissions/:id/status` - Removed (router.go:231)
- `PUT /api/v1/companies/:id` - Removed (router.go:195, 255)
- Notes say "edit in Supabase directly" but no user guidance

### 3. Missing Documentation

#### 3.1 No Route Documentation
- Routes only documented in code comments
- No API specification (OpenAPI/Swagger)
- Frontend relies on code inspection

#### 3.2 Missing Error Code Documentation
- `ErrorResponse` type is generic
- No standardized error codes
- Portuguese messages hardcoded in handlers

#### 3.3 No Rate Limit Documentation
- Global rate limit (100 req/min) not documented
- No endpoint-specific limits
- No guidance for API consumers

### 4. Security Concerns

#### 4.1 CORS Configuration
- Allows credentials (`AllowCredentials: true`)
- 12-hour MaxAge is very long
- Health endpoint allows all origins (`*`)
- **Recommendation**: Review CORS policy

#### 4.2 Missing Input Validation
- `CreateSubmissionRequest` relies on frontend validation
- Server-side validation is minimal
- **Recommendation**: Add server-side validation layer

#### 4.3 Security Event Logging Incomplete
- `SecurityEventLogger` defined but underutilized
- Only used in `AuthMiddleware` and `AdminAuthMiddleware`
- **Recommendation**: Add to all sensitive operations

### 5. Testing Gaps

#### 5.1 Missing Handler Tests
- ❌ No tests for: admin_handlers.go
- ❌ No tests for: analysis_handlers.go
- ❌ No tests for: company_crud_handlers.go
- ❌ No tests for: company_analysis_handlers.go
- ❌ No tests for: framework_handlers.go
- ❌ No tests for: macro_handlers.go
- ❌ No tests for: wizard_handlers.go
- ❌ No tests for: health_handlers.go
- ✅ Tests exist for: auth, middleware, submission (partial)

#### 5.2 Missing Integration Tests
- No end-to-end API tests
- Contract tests only for submission
- **Recommendation**: Add integration test suite

### 6. Code Duplication

#### 6.1 Error Response Handling
- Multiple error response patterns:
  - `respondError()` in helpers.go
  - `RespondAppError()` in helpers.go
  - `RespondValidationError()` in helpers.go
  - Inline `c.JSON(status, ErrorResponse{...})` in handlers
- **Recommendation**: Standardize on one pattern

#### 6.2 UUID Parsing
- `parseUUID()` helper exists but not used consistently
- Some handlers use `uuid.Parse()` directly
- **Recommendation**: Enforce helper usage

#### 6.3 Status Derivation
- `deriveStatus()` in submission_response_builder.go
- Status logic scattered across handlers
- **Recommendation**: Centralize in domain layer

### 7. Type System Issues

#### 7.1 Inconsistent Response Structures
- Some endpoints wrap data: `gin.H{"submission": resp}`
- Some return data directly: `c.JSON(200, resp)`
- **Recommendation**: Standardize response format

#### 7.2 Nullable Field Confusion
- Mix of `*string` and `string` for optional fields
- No clear pattern for when to use pointers
- **Recommendation**: Document nullable field policy

#### 7.3 Time Format Inconsistency
- Some use `time.Time` directly
- Some use `Format(time.RFC3339)`
- **Recommendation**: Use consistent serialization

---

## Recommended Changes

### ~~Phase 1: Quick Wins~~ ✅ COMPLETED (2025-12-06)

1. ✅ **Merge company handler files**
   - Merged `company_crud_handlers.go` + `company_analysis_handlers.go` → `company_handlers.go`
   - Single `CompanyHandlers` struct with 6 methods
   - Consistent with other domain handlers

2. ✅ **Remove deprecated endpoints**
   - Removed `RetryEnrichment()` (enrichment is automatic)
   - Removed `GetAnalytics()` (needs rebuild)
   - Routes removed from router with comments

3. ✅ **Move GetMetrics to admin handlers**
   - Moved from `health_handlers.go` to `admin_handlers.go`
   - Properly scoped as admin-only endpoint
   - `health_handlers.go` now only contains public health check

4. **Standardize error responses** (TODO)
   ```go
   // Use RespondAppError() everywhere
   // Remove inline c.JSON() error responses
   ```

5. **Add route documentation** (TODO)
   ```markdown
   # Create docs/api_reference.md with all endpoints
   # Or generate from OpenAPI spec
   ```

### Phase 2: Testing (Medium Effort)

5. **Add missing handler tests**
   ```bash
   # Create test files for all handlers
   # Target: 80% code coverage
   ```

6. **Add integration tests**
   ```bash
   # Create tests/integration/ directory
   # Test full request/response flows
   ```

### Phase 3: Architecture Improvements (Higher Effort)

7. **Standardize response format**
   ```go
   // Define standard envelope:
   // { "data": {...}, "meta": {...}, "errors": [...] }
   ```

8. **Add OpenAPI/Swagger spec**
   ```bash
   # Generate from code or write manually
   # Integrate with swagger-ui for docs
   ```

9. **Improve security event logging**
   ```go
   // Add security logging to all sensitive operations:
   // - Data modifications
   // - Access to sensitive data
   // - Admin actions
   ```

10. **Refactor status derivation**
    ```go
    // Move deriveStatus() to domain layer
    // Create submission.DerivedStatus() method
    ```

### Current File Structure (After Consolidation)

```
api/
├── admin_handlers.go              # Admin operations + system metrics (4 handlers)
├── analysis_handlers.go           # Analysis CRUD and visibility (9 handlers)
├── auth_handlers.go               # Authentication with Supabase (7 handlers)
├── company_handlers.go            # Company CRUD + re-analysis + retry-enrichment (7 handlers)
├── health_handlers.go             # Health check only (1 handler)
├── submission_handlers.go         # Public submission endpoint (3 handlers)
├── user_handlers.go               # User profile management (2 handlers)
├── wizard_handlers.go             # Human-in-the-loop wizard (7 handlers)
├── handlers.go                    # Main Handler composition
├── middleware.go                  # Auth, CORS, logging, rate limit
├── router.go                      # Route definitions
├── types.go                       # Request/response DTOs
├── helpers.go                     # Utility functions
├── security_events.go             # Security audit logging
├── submission_response_builder.go # Status derivation logic
└── README.md                      # This file
```

**Key Changes (2025-12-07 - MVP Simplification):**
- ✅ Removed `framework_handlers.go` (frameworks now hardcoded in code)
- ✅ Removed `macro_handlers.go` (macro indicators now hardcoded)
- ✅ Removed `ToggleBlur()` handler (is_blurred field removed)
- ✅ Added `RetryEnrichment()` handler for "fill gaps only" re-enrichment
- ✅ Updated route table to reflect current endpoints

**Key Changes (2025-12-06):**
- ✅ Merged `company_crud_handlers.go` + `company_analysis_handlers.go` → `company_handlers.go`
- ✅ Moved `GetMetrics()` from `health_handlers.go` to `admin_handlers.go`
- ✅ Consistent naming: All handlers follow `{domain}_handlers.go` pattern

### Naming Conventions (Recommended)

- **Handler Files**: `{domain}_handlers.go`
- **Handler Structs**: `{Domain}Handlers`
- **Handler Functions**: `{Action}{Entity}()` (e.g., `GetAnalysis()`, `CreateSubmission()`)
- **Request Types**: `{Action}{Entity}Request` (e.g., `CreateSubmissionRequest`)
- **Response Types**: `{Entity}Response` or `{Entity}ListResponse`
- **Test Files**: `{domain}_handlers_test.go`

---

## API Versioning

**Current Version**: v1
**Base Path**: `/api/v1`

### Future Considerations

- v2 API should introduce breaking changes:
  - Standardized response envelope
  - Consistent error codes
  - OpenAPI spec
  - Deprecate legacy endpoints

- Maintain v1 compatibility for 6 months after v2 release

---

## Dependencies

### External Packages

- **Gin Framework**: `github.com/gin-gonic/gin` - HTTP router
- **Gin CORS**: `github.com/gin-contrib/cors` - CORS middleware
- **JWT**: `github.com/golang-jwt/jwt/v5` - Token validation
- **SQLX**: `github.com/jmoiron/sqlx` - Database access
- **Redis**: `github.com/redis/go-redis/v9` - Caching
- **Asynq**: `github.com/hibiken/asynq` - Background jobs
- **Zerolog**: `github.com/rs/zerolog` - Structured logging
- **UUID**: `github.com/google/uuid` - UUID generation
- **Rate**: `golang.org/x/time/rate` - Rate limiting

### Internal Dependencies

- **Domain Services**: All domain packages (`submission`, `analysis`, etc.)
- **App Errors**: `backend_v3/pkg/errors` - Error handling

---

## Metrics & Monitoring

### Health Check Endpoint

**GET** `/health`

Returns:
```json
{
  "status": "healthy",
  "services": {
    "database": "healthy",
    "redis": "healthy"
  }
}
```

### Metrics Endpoint (Admin Only)

**GET** `/api/v1/admin/metrics`

Returns:
```json
{
  "submissions_last_24h": 42,
  "enrichment_success_rate": "95%",
  "analysis_success_rate": "92%",
  "avg_analysis_time_seconds": 45.3,
  "total_cost_last_24h_usd": 12.45,
  "total_tokens_last_24h": 150000,
  "llm_requests_last_24h": 120,
  "errors_last_24h": ["Error 1", "Error 2"],
  "last_updated": "2025-12-06T12:00:00Z"
}
```

---

## Contributing

### Adding a New Handler

1. Create `{domain}_handlers.go` file
2. Define `{Domain}Handlers` struct with dependencies
3. Create `New{Domain}Handlers()` constructor
4. Implement handler functions with `(h *{Domain}Handlers) {Action}(c *gin.Context)`
5. Add to main `Handler` struct in `handlers.go`
6. Register routes in `router.go`
7. Create `{domain}_handlers_test.go` with tests

### Adding a New Endpoint

1. Add handler function to appropriate handler file
2. Register route in `router.go` under correct group (public/protected/admin)
3. Add request/response types to `types.go`
4. Update this README with new route
5. Add tests
6. Update API documentation

---

## Security Best Practices

1. **Always use AuthMiddleware** for protected routes
2. **Validate all inputs** on the server side
3. **Use prepared statements** to prevent SQL injection (handled by SQLX)
4. **Log security events** with SecurityEventLogger
5. **Rate limit sensitive endpoints** (auth, submissions)
6. **Never log passwords or tokens**
7. **Use HTTPS in production** (enforced by Railway)

---

## Troubleshooting

### Common Issues

1. **CORS errors in browser**
   - Check `allowedOrigins` includes frontend URL
   - Verify Origin header is sent by frontend

2. **401 Unauthorized errors**
   - Check JWT token is valid and not expired
   - Verify `SUPABASE_JWT_SECRET` matches token signing key
   - Ensure user exists in `user_profiles` table

3. **403 Forbidden errors**
   - Check user has correct role in database
   - Verify AdminAuthMiddleware is applied to route

4. **500 Internal Server errors**
   - Check server logs for panic messages
   - Verify database connection is healthy
   - Check Redis connection (if used)

---

## Performance Considerations

### Current Bottlenecks

1. **Database queries in handlers** - Should be in services
2. **Synchronous enrichment calls** - Now handled via background jobs
3. **No caching layer** - Redis available but underutilized
4. **No request batching** - Each request hits DB independently

### Optimization Opportunities

1. **Add Redis caching** for frequently accessed data
2. **Implement connection pooling** (already configured in SQLX)
3. **Add database query optimization** (indexes, query analysis)
4. **Consider GraphQL** for flexible data fetching

---

## Related Documentation

- [API Reference (Frontend)](../docs/API.md) - Complete endpoint documentation
- [Domain Layer README](../domain/README.md) - Business logic documentation

---

**Document Version**: 1.0
**Maintainer**: Backend Team
**Last Reviewed**: 2025-12-06
