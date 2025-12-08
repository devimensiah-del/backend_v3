<objective>
Consolidate and finalize API documentation for frontend developers.

This is Phase 7 (final phase) of the comprehensive codebase audit. Create comprehensive, trustworthy API documentation in /docs that frontend developers can rely on completely.
</objective>

<context>
Read @./docs/audit/016-api-endpoints-matrix.md for complete endpoint inventory.
Read @./docs/audit/PHASE4-SUMMARY.md for handler audit findings.
Read @CLAUDE.md for project overview context.

This documentation will be the single source of truth for frontend developers.
</context>

<documentation_requirements>

**Main API Documentation (`./docs/API.md`):**

1. **Overview Section**
   - Base URL configuration
   - Authentication requirements
   - Common headers
   - Rate limiting information
   - Error response format

2. **Authentication Endpoints**
   - Login, signup, password reset
   - Token format and expiration
   - Refresh flow if applicable

3. **Public Endpoints**
   - Submission endpoint
   - Public report access
   - No-auth requirements

4. **Protected Endpoints (User)**
   - All submission-related endpoints
   - Company endpoints
   - Analysis endpoints
   - Wizard endpoints

5. **Admin Endpoints**
   - All admin operations
   - Required permissions

**For Each Endpoint Document:**
- HTTP method and path
- Authentication requirements
- Request body schema with types
- Query parameters if any
- Response schema with examples
- Error responses with codes
- Business logic notes (what happens on success)

**Additional Documentation Files:**

`./docs/SCHEMAS.md`:
- All request/response TypeScript interfaces
- Frontend can copy these directly

`./docs/WORKFLOWS.md`:
- Complete submission workflow
- Analysis execution flow
- Wizard step-by-step flow
- Status transitions

`./docs/ERRORS.md`:
- Error code reference
- How to handle each error type
- User-friendly message mappings
</documentation_requirements>

<output>
Create/update documentation files:
- `./docs/API.md` - Complete API reference
- `./docs/SCHEMAS.md` - TypeScript interfaces
- `./docs/WORKFLOWS.md` - Business workflows
- `./docs/ERRORS.md` - Error handling guide

Update existing if present, otherwise create new.

Create `./docs/audit/PHASE7-SUMMARY.md` with:
- Documentation created/updated
- Coverage assessment
- Any gaps or TODOs
</output>

<quality_standards>
- Documentation must be accurate and match current code
- Examples must be valid and tested
- No placeholder text or TODOs in final docs
- TypeScript schemas must be syntactically correct
- Every endpoint must be documented
</quality_standards>

<verification>
Before completing:
- Every endpoint from audit matrix is documented
- All schemas are complete with types
- Workflow diagrams/descriptions are accurate
- Error codes are comprehensive
- Frontend developer could implement integration using only these docs
</verification>
