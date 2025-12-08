# IMENSIAH API Reference

> **Version**: v1.0 | **Base URL**: `/api/v1` | **Updated**: 2025-12-06

Complete API reference for frontend developers. All routes are under `/api/v1` unless noted.

---

## Quick Reference

| Category | Endpoints | Auth |
|----------|-----------|------|
| [Auth](#auth) | 7 | Mixed |
| [Submissions](#submissions) | 4 | Mixed |
| [Companies](#companies) | 3 | Required |
| [Wizard](#wizard-human-in-the-loop) | 6 | Required |
| [Frameworks](#frameworks) | 1 | Required |
| [Public Report](#public-report) | 1 | Optional |
| [Admin](#admin-endpoints) | 13 | Admin |

---

## Authentication

All protected endpoints require a Bearer token in the `Authorization` header:

```
Authorization: Bearer <access_token>
```

Tokens are JWTs signed with the Supabase JWT secret. Token payload includes `sub` (user ID) and `role`.

### Roles

| Role | Access |
|------|--------|
| `user` | Own submissions, companies, analyses |
| `admin` | All user access + admin endpoints |
| `super_admin` | All admin access |
| `service_role` | Backend service access |

---

## Auth

### `POST /auth/login`

Authenticate user and receive access token.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response `200`:**
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "role": "user"
  },
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "expires_at": 1733500000
}
```

### `POST /auth/signup`

Register a new user. Links any anonymous submissions with matching email.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "password123",
  "fullName": "John Doe"
}
```

**Response `201`:**
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com"
  },
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

### `POST /auth/forgot-password`

Request password reset email.

**Request:**
```json
{
  "email": "user@example.com"
}
```

**Response `200`:**
```json
{
  "message": "If your email exists, you will receive a reset link"
}
```

### `POST /auth/reset-password`

Reset password using token from email.

**Request:**
```json
{
  "token": "reset-token-from-email",
  "newPassword": "newpassword123"
}
```

**Response `200`:**
```json
{
  "message": "Password reset successfully"
}
```

### `GET /auth/me` *(Auth Required)*

Get current user profile.

**Response `200`:**
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "fullName": "John Doe",
    "role": "user",
    "isActive": true,
    "createdAt": "2025-01-01T00:00:00Z",
    "updatedAt": "2025-01-01T00:00:00Z"
  }
}
```

### `POST /auth/logout` *(Auth Required)*

Logout current user (client should discard token).

**Response `200`:**
```json
{
  "message": "Logged out successfully"
}
```

### `PUT /auth/update-password` *(Auth Required)*

Update password for authenticated user.

**Request:**
```json
{
  "currentPassword": "oldpassword",
  "newPassword": "newpassword123"
}
```

**Response `200`:**
```json
{
  "message": "Password updated successfully"
}
```

---

## Submissions

### `POST /submissions` *(Public)*

Create a new submission. This creates:
1. **Submission** record (entry point)
2. **Company** record (with automatic Perplexity enrichment)
3. **Challenge** record (linked to company)

**Request:**
```json
{
  "companyName": "Acme Corp",
  "challengeCategory": "growth",
  "challengeType": "growth_organic",
  "businessChallenge": "We need to increase market share by 50%",
  "cnpj": "12.345.678/0001-90",
  "industry": "Technology",
  "companySize": "medium",
  "website": "https://acme.com",
  "additionalInfo": "{\"contactName\":\"John\",\"contactEmail\":\"john@acme.com\"}"
}
```

**Required Fields:**
- `companyName` - Company name
- `challengeCategory` - One of: `growth`, `transform`, `transition`, `compete`, `funding`
- `challengeType` - See [Challenge Types](#challenge-types)
- `businessChallenge` - Free-text description of the challenge

**Optional Fields:**
- `cnpj` - Brazilian company ID
- `industry` - Industry sector
- `companySize` - small, medium, large, enterprise
- `website` - Company website URL

**AdditionalInfo (JSON string):**
```json
{
  "contactName": "string",
  "contactEmail": "string (required)",
  "contactPhone": "string",
  "contactPosition": "string",
  "companyLocation": "string",
  "targetMarket": "string",
  "annualRevenueMin": "number",
  "annualRevenueMax": "number",
  "fundingStage": "string",
  "additionalNotes": "string",
  "linkedinUrl": "string",
  "twitterHandle": "string"
}
```

**Response `201`:**
```json
{
  "id": "submission-uuid",
  "companyId": "company-uuid",
  "challengeId": "challenge-uuid",
  "createdAt": "2025-01-01T00:00:00Z"
}
```

### `GET /submissions` *(Auth Required)*

List current user's submissions.

**Query Parameters:**
- `page` - Page number (default: 1)
- `pageSize` - Items per page (default: 10, max: 100)

**Response `200`:**
```json
{
  "data": [
    {
      "id": "uuid",
      "companyName": "Acme Corp",
      "status": "analyzing",
      "companyId": "uuid",
      "challengeId": "uuid",
      "analysisId": "uuid",
      "createdAt": "2025-01-01T00:00:00Z"
    }
  ],
  "page": 1,
  "pageSize": 10,
  "total": 25,
  "totalPages": 3
}
```

### `GET /submissions/:id` *(Auth Required)*

Get submission details with related entity IDs.

**Response `200`:**
```json
{
  "id": "uuid",
  "userId": "uuid",
  "companyName": "Acme Corp",
  "cnpj": "12.345.678/0001-90",
  "companyWebsite": "https://acme.com",
  "companyIndustry": "Technology",
  "companySize": "medium",
  "companyLocation": "São Paulo, SP",
  "contactName": "John Doe",
  "contactEmail": "john@acme.com",
  "contactPhone": "+55 11 99999-9999",
  "contactPosition": "CEO",
  "targetMarket": "B2B",
  "annualRevenueMin": 1000000,
  "annualRevenueMax": 5000000,
  "fundingStage": "Series A",
  "additionalNotes": "Additional context...",
  "linkedinUrl": "https://linkedin.com/company/acme",
  "twitterHandle": "@acme",
  "createdAt": "2025-01-01T00:00:00Z",
  "updatedAt": "2025-01-01T00:00:00Z",
  "companyId": "uuid",
  "challengeId": "uuid",
  "analysisId": "uuid",
  "pdfUrl": "https://storage.supabase.co/...",
  "challengeCategory": "growth",
  "challengeType": "growth_organic",
  "businessChallenge": "We need to increase market share...",
  "status": "completed"
}
```

**Status Values (derived):**
| Status | Meaning |
|--------|---------|
| `pending` | Submission received, enrichment starting |
| `enriching` | Company enrichment in progress |
| `enriched` | Enrichment complete, awaiting wizard |
| `analyzing` | Wizard/analysis in progress |
| `completed` | Analysis complete |
| `failed` | Processing failed |

### `GET /submissions/:id/analysis` *(Auth Required)*

Get analysis for a submission.

**Response `200`:**
```json
{
  "id": "uuid",
  "submissionId": "uuid",
  "companyId": "uuid",
  "challengeId": "uuid",
  "status": "completed",
  "analysis": {
    "pestel": { ... },
    "porter": { ... },
    "swot": { ... },
    "tam_sam_som": { ... },
    "benchmarking": { ... },
    "blue_ocean": { ... },
    "growth_hacking": { ... },
    "scenarios": { ... },
    "okrs": { ... },
    "bsc": { ... },
    "decision_matrix": { ... },
    "synthesis": { ... }
  },
  "is_visible_to_user": true,
  "is_public": false,
  "access_code": "ABC123",
  "createdAt": "2025-01-01T00:00:00Z",
  "updatedAt": "2025-01-01T00:00:00Z"
}
```

---

## Companies

### `POST /companies` *(Auth Required)*

Create a company directly (without submission).

**Request:**
```json
{
  "name": "Acme Corp",
  "website": "https://acme.com",
  "cnpj": "12.345.678/0001-90",
  "industry": "Technology",
  "company_size": "medium",
  "location": "São Paulo, SP",
  "target_market": "B2B",
  "funding_stage": "Series A"
}
```

**Response `201`:**
```json
{
  "id": "uuid",
  "name": "Acme Corp",
  "enrichmentStatus": "pending",
  "createdAt": "2025-01-01T00:00:00Z"
}
```

### `GET /companies` *(Auth Required)*

List user's companies (owned or in allowed_users).

**Response `200`:**
```json
{
  "companies": [
    {
      "id": "uuid",
      "name": "Acme Corp",
      "website": "https://acme.com",
      "industry": "Technology",
      "enrichmentStatus": "completed",
      "ownerId": "uuid",
      "createdAt": "2025-01-01T00:00:00Z"
    }
  ]
}
```

### `GET /companies/:id` *(Auth Required)*

Get company details (must be owner or in allowed_users).

**Response `200`:**
```json
{
  "id": "uuid",
  "name": "Acme Corp",
  "cnpj": "12.345.678/0001-90",
  "website": "https://acme.com",
  "industry": "Technology",
  "companySize": "medium",
  "location": "São Paulo, SP",
  "targetMarket": "B2B",
  "fundingStage": "Series A",
  "annualRevenueMin": 1000000,
  "annualRevenueMax": 5000000,
  "foundationYear": "2015",
  "legalName": "Acme Corp Ltda",
  "headquarters": "São Paulo, SP, Brazil",
  "sector": "SaaS",
  "targetAudience": "SMBs",
  "valueProposition": "Best-in-class software",
  "employeesRange": "50-100",
  "revenueEstimate": "R$ 1M - 5M",
  "businessModel": "B2B SaaS",
  "competitors": ["Competitor A", "Competitor B"],
  "marketShareStatus": "Challenger",
  "digitalMaturity": 7,
  "strengths": ["Strong team", "Innovative product"],
  "weaknesses": ["Limited brand awareness"],
  "linkedinUrl": "https://linkedin.com/company/acme",
  "twitterHandle": "@acme",
  "enrichmentStatus": "completed",
  "enrichmentCompletedAt": "2025-01-01T00:00:00Z",
  "ownerId": "uuid",
  "allowedUsers": ["uuid1", "uuid2"],
  "createdAt": "2025-01-01T00:00:00Z",
  "updatedAt": "2025-01-01T00:00:00Z"
}
```

---

## Wizard (Human-in-the-Loop)

The wizard enables step-by-step analysis with human validation at each stage.

### Workflow

```
Start Wizard → Generate Step → Review → Approve/Refine → Next Step → ... → Complete
```

**Key Rules:**
- Wizard only moves **forward** - no going back
- Human provides **context**, AI **regenerates** - no direct edits
- Each refinement creates a **version snapshot**
- 12 steps (0-11) + auto-generated synthesis

### `POST /wizard/start` *(Auth Required)*

Start or resume wizard for a company + challenge.

**Request:**
```json
{
  "company_id": "uuid",
  "challenge_id": "uuid"
}
```

**Response `200`:**
```json
{
  "analysisId": "uuid",
  "currentStep": 0,
  "totalSteps": 12,
  "framework": {
    "step": 0,
    "code": "challenge_refinement",
    "name": "Refinamento do Desafio",
    "description": "Refine o desafio de negócio...",
    "questions": [
      { "id": "q1", "question": "Qual o principal impacto esperado?" },
      { "id": "q2", "question": "Quais métricas definiriam sucesso?" }
    ]
  },
  "stepStatus": "pending"
}
```

### `GET /analyses/:id/wizard` *(Auth Required)*

Get current wizard state.

**Response `200`:**
```json
{
  "analysisId": "uuid",
  "currentStep": 3,
  "totalSteps": 12,
  "framework": {
    "step": 3,
    "code": "benchmarking",
    "name": "Benchmarking",
    "description": "Compare com concorrentes...",
    "questions": [...]
  },
  "stepStatus": "generated",
  "output": { ... },
  "humanContext": "Previous context provided",
  "humanAnswers": { "q1": "answer1" },
  "previousSteps": [
    { "step": 0, "frameworkCode": "challenge_refinement", "status": "approved" },
    { "step": 1, "frameworkCode": "pestel", "status": "approved" },
    { "step": 2, "frameworkCode": "porter", "status": "approved" }
  ],
  "iterationCount": 1
}
```

### `POST /analyses/:id/wizard/generate` *(Auth Required)*

Generate output for current step.

**Request:**
```json
{
  "humanContext": "Optional context to guide AI",
  "humanAnswers": {
    "q1": "Answer to question 1",
    "q2": "Answer to question 2"
  }
}
```

**Response `200`:**
```json
{
  "analysisId": "uuid",
  "currentStep": 3,
  "stepStatus": "generated",
  "output": {
    "competitors_analyzed": ["Competitor A", "Competitor B"],
    "performance_gaps": ["Gap 1", "Gap 2"],
    "best_practices": ["Practice 1", "Practice 2"],
    "summary": "Executive summary..."
  }
}
```

### `POST /analyses/:id/wizard/approve` *(Auth Required)*

Approve current step and advance to next.

**Response `200`:**
```json
{
  "analysisId": "uuid",
  "previousStep": 3,
  "currentStep": 4,
  "stepStatus": "pending",
  "framework": {
    "step": 4,
    "code": "swot",
    "name": "SWOT",
    "description": "Análise SWOT...",
    "questions": [...]
  }
}
```

### `POST /analyses/:id/wizard/refine` *(Auth Required)*

Add context and regenerate current step (creates version).

**Request:**
```json
{
  "humanContext": "Please focus more on...",
  "humanAnswers": {
    "q1": "Updated answer"
  }
}
```

**Response `200`:**
```json
{
  "analysisId": "uuid",
  "currentStep": 3,
  "stepStatus": "generated",
  "iterationCount": 2,
  "output": { ... }
}
```

### `GET /analyses/:id/wizard/summary` *(Auth Required)*

Get summary of all completed steps.

**Response `200`:**
```json
{
  "analysisId": "uuid",
  "completedSteps": [
    { "step": 0, "frameworkCode": "challenge_refinement", "frameworkName": "Refinamento do Desafio", "status": "approved", "approvedAt": "2025-01-01T10:00:00Z" },
    { "step": 1, "frameworkCode": "pestel", "frameworkName": "PESTEL", "status": "approved", "approvedAt": "2025-01-01T10:15:00Z" }
  ],
  "currentStep": 2,
  "totalSteps": 12,
  "progress": 16
}
```

---

## Frameworks

> **Note**: Framework definitions are hardcoded in the backend. The `/frameworks` and `/frameworks/:code` public endpoints have been removed. Use `/frameworks/order` to get the wizard execution order.

### `GET /frameworks/order` *(Auth Required)*

Get framework execution order for wizard.

**Response `200`:**
```json
{
  "frameworks": [
    { "step": 0, "code": "challenge_refinement", "name": "Refinamento do Desafio" },
    { "step": 1, "code": "pestel", "name": "PESTEL" },
    { "step": 2, "code": "porter", "name": "Porter 5 Forças" },
    { "step": 3, "code": "benchmarking", "name": "Benchmarking" },
    { "step": 4, "code": "swot", "name": "SWOT" },
    { "step": 5, "code": "tam_sam_som", "name": "TAM-SAM-SOM" },
    { "step": 6, "code": "blue_ocean", "name": "Blue Ocean" },
    { "step": 7, "code": "growth_hacking", "name": "Growth Loops" },
    { "step": 8, "code": "scenarios", "name": "Cenários" },
    { "step": 9, "code": "decision_matrix", "name": "Matriz de Decisão" },
    { "step": 10, "code": "okrs", "name": "Plano 90 Dias" },
    { "step": 11, "code": "bsc", "name": "BSC" }
  ],
  "total_steps": 12
}
```

---

## Challenge Types

### `GET /challenges/types` *(Auth Required)*

Get all valid challenge categories and types.

**Response `200`:**
```json
{
  "categories": ["growth", "transform", "transition", "compete", "funding"],
  "types": {
    "growth": ["growth_organic", "growth_geographic", "growth_segment", "growth_product", "growth_channel"],
    "transform": ["transform_digital", "transform_model", "transform_culture", "transform_operational"],
    "transition": ["transition_succession", "transition_exit", "transition_merger", "transition_turnaround"],
    "compete": ["compete_differentiate", "compete_defend", "compete_reposition"],
    "funding": ["funding_raise", "funding_debt", "funding_ipo"]
  }
}
```

---

## Public Report

### `GET /public/report/:code` *(Optional Auth)*

Access analysis via public access code.

**Path Parameters:**
- `code` - Access code (e.g., "ABC123")

**Response `200`:** Same as `GET /submissions/:id/analysis`

**Note:** If authenticated as admin, bypasses visibility checks for preview.

---

## Admin Endpoints

All admin endpoints require `admin`, `super_admin`, or `service_role` role.

### Submissions

#### `GET /admin/submissions`

List all submissions with filters.

**Query Parameters:**
- `page` - Page number
- `pageSize` - Items per page
- `email` - Filter by contact email
- `status` - Filter by derived status

**Response `200`:** Same as `GET /submissions`

#### `GET /admin/submissions/:id`

Get any submission (no ownership check).

**Response `200`:** Same as `GET /submissions/:id`

#### `GET /admin/submissions/:id/analysis`

Get analysis for any submission.

**Response `200`:** Same as `GET /submissions/:id/analysis`

#### `POST /admin/submissions/:id/retry-analysis`

Retry failed analysis job.

**Response `202`:**
```json
{
  "analysisId": "uuid",
  "status": "pending"
}
```

### Analysis Management

#### `GET /admin/analysis/:id`

Get analysis by analysis ID.

**Response `200`:** Same as analysis response

#### `PUT /admin/analysis/:id`

Update analysis fields (framework results).

**Request:**
```json
{
  "pestel": { "political": ["Updated point"] },
  "swot": { "strengths": [{ "content": "New strength", "confidence": "Alta", "source": "Admin" }] }
}
```

**Response `200`:** Updated analysis

#### `POST /admin/analysis/:id/visibility`

Toggle user visibility.

**Response `200`:**
```json
{
  "id": "uuid",
  "is_visible_to_user": true
}
```

#### `POST /admin/analysis/:id/public`

Toggle public access (via access code).

**Response `200`:**
```json
{
  "id": "uuid",
  "is_public": true
}
```

#### `POST /admin/analysis/:id/access-code`

Generate new access code.

**Response `200`:**
```json
{
  "id": "uuid",
  "access_code": "XYZ789"
}
```

#### `POST /admin/analysis/:id/wizard/generate-all`

Bypass wizard step-by-step approval and generate all 12 frameworks at once.

**Response `200`:**
```json
{
  "message": "Bulk generation complete",
  "progress": {
    "current_step": 12,
    "total_steps": 12,
    "completed_steps": [
      "challenge_refinement", "pestel", "porter", "benchmarking",
      "swot", "tam_sam_som", "blue_ocean", "growth_hacking",
      "scenarios", "decision_matrix", "okrs", "bsc"
    ],
    "failed_steps": [],
    "status": "completed",
    "processing_time_ms": 45000,
    "results": {
      "challenge_refinement": "generated",
      "pestel": "generated",
      "porter": "generated",
      "benchmarking": "generated",
      "swot": "generated",
      "tam_sam_som": "generated",
      "blue_ocean": "generated",
      "growth_hacking": "generated",
      "scenarios": "generated",
      "decision_matrix": "generated",
      "okrs": "generated",
      "bsc": "generated"
    }
  }
}
```

**Notes:**
- Generates all 12 frameworks sequentially with auto-approval
- Synthesis is generated automatically after all steps complete
- Analysis status is set to `completed` upon success
- If any step fails, the process stops and returns partial progress

### Company Management

#### `GET /admin/companies`

List all companies.

**Query Parameters:**
- `limit` - Max results (default: 100)
- `offset` - Offset for pagination

**Response `200`:**
```json
{
  "companies": [...]
}
```

#### `GET /admin/companies/:id`

Get any company details.

**Response `200`:** Same as `GET /companies/:id`

#### `POST /admin/companies/:id/re-analyze`

Trigger new analysis for existing company with new challenge.

**Request:**
```json
{
  "challengeCategory": "transform",
  "challengeType": "transform_digital",
  "businessChallenge": "New challenge description"
}
```

**Response `200`:**
```json
{
  "message": "Re-analysis started",
  "data": {
    "submission_id": "uuid",
    "company_id": "uuid",
    "challenge_id": "uuid",
    "challenge_category": "transform",
    "challenge_type": "transform_digital"
  }
}
```

#### `POST /admin/companies/:id/retry-enrichment`

Re-run Perplexity enrichment for a company. Uses **"fill gaps only"** logic - only fills NULL/empty fields, preserving any manually edited values.

**Response `200`:**
```json
{
  "message": "Enrichment retry started (fill gaps only)",
  "data": {
    "company_id": "uuid",
    "company_name": "Acme Corp",
    "previous_status": "failed",
    "mode": "fill_gaps_only"
  }
}
```

**Notes:**
- Runs asynchronously (returns immediately)
- Only updates fields that are NULL or empty
- Preserves manual edits to company data
- Check `enrichmentStatus` on company to monitor progress

### System Metrics

#### `GET /admin/metrics`

Get system-wide metrics.

**Response `200`:**
```json
{
  "submissions_last_24h": 42,
  "enrichment_success_rate": "95%",
  "analysis_success_rate": "92%",
  "avg_analysis_time_seconds": 45.3,
  "total_cost_last_24h_usd": 12.45,
  "total_tokens_last_24h": 150000,
  "llm_requests_last_24h": 120,
  "errors_last_24h": ["Error 1"],
  "last_updated": "2025-12-06T12:00:00Z"
}
```

---

## User Profile

### `GET /user/profile` *(Auth Required)*

Alias for `GET /auth/me`.

### `PUT /user/profile` *(Auth Required)*

Update user profile.

**Request:**
```json
{
  "fullName": "John Updated"
}
```

**Response `200`:**
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "fullName": "John Updated",
    "role": "user",
    "isActive": true
  }
}
```

### `DELETE /user` *(Auth Required)*

Deactivate user account (soft delete).

**Response `200`:**
```json
{
  "message": "Account deactivated"
}
```

---

## Health Check

### `GET /health` *(Public, No Rate Limit)*

System health check.

**Response `200`:**
```json
{
  "status": "healthy",
  "services": {
    "database": "healthy",
    "redis": "healthy"
  }
}
```

---

## Error Responses

All errors return:

```json
{
  "error": "Error Type",
  "message": "Human-readable description"
}
```

### Status Codes

| Code | Meaning |
|------|---------|
| 400 | Bad Request - Validation failed, invalid parameters |
| 401 | Unauthorized - Missing or invalid auth token |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource doesn't exist |
| 429 | Too Many Requests - Rate limited |
| 500 | Internal Server Error - Server error |

### Rate Limits

| Endpoint | Limit |
|----------|-------|
| Global | 100 requests/minute per IP |

---

## Workflow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        USER FLOW                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  [Submit Form]                                                   │
│       │                                                          │
│       ▼                                                          │
│  POST /submissions ─────► Creates: Submission + Company + Challenge
│       │                                                          │
│       │  (Automatic: Company enrichment via Perplexity)          │
│       │                                                          │
│       ▼                                                          │
│  [Start Wizard]                                                  │
│       │                                                          │
│       ▼                                                          │
│  POST /wizard/start { company_id, challenge_id } ► Creates Analysis
│       │                                                          │
│       ▼                                                          │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  WIZARD LOOP (12 steps)                                  │    │
│  │                                                          │    │
│  │  1. GET /analyses/:id/wizard     ◄── Get current state   │    │
│  │  2. POST /.../wizard/generate    ◄── Generate AI output  │    │
│  │  3. Review output                                        │    │
│  │  4. POST /.../wizard/approve     ◄── Accept & advance    │    │
│  │     OR                                                   │    │
│  │     POST /.../wizard/refine      ◄── Add context & redo  │    │
│  │                                                          │    │
│  │  Repeat until step 11 (BSC) approved                     │    │
│  └─────────────────────────────────────────────────────────┘    │
│       │                                                          │
│       ▼                                                          │
│  [Synthesis Auto-Generated]                                      │
│       │                                                          │
│       ▼                                                          │
│  [Analysis Complete] ─────► Status: "completed"                  │
│       │                                                          │
│       ▼                                                          │
│  GET /submissions/:id/analysis ─────► Full analysis data         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                       ADMIN FLOW                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  [Manage Visibility]                                             │
│  POST /admin/analysis/:id/visibility  ◄── Show to user           │
│  POST /admin/analysis/:id/public      ◄── Enable public link     │
│  POST /admin/analysis/:id/access-code ◄── Generate share code    │
│                                                                  │
│  [Company Management]                                            │
│  POST /admin/companies/:id/re-analyze       ◄── New challenge    │
│  POST /admin/companies/:id/retry-enrichment ◄── Fill data gaps   │
│                                                                  │
│  [Monitor System]                                                │
│  GET /admin/metrics                   ◄── LLM costs, success rates
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## TypeScript Types

```typescript
// Auth
interface LoginRequest {
  email: string;
  password: string;
}

interface AuthResponse {
  user: { id: string; email: string; role?: string };
  access_token: string;
  token_type: string;
  expires_in: number;
  expires_at?: number;
}

// Submission
interface CreateSubmissionRequest {
  companyName: string;
  challengeCategory: ChallengeCategory;
  challengeType: string;
  businessChallenge: string;
  cnpj?: string;
  industry?: string;
  companySize?: string;
  website?: string;
  additionalInfo?: string; // JSON string of AdditionalInfoData
}

interface AdditionalInfoData {
  contactName: string;
  contactEmail: string;
  contactPhone?: string;
  contactPosition?: string;
  companyLocation?: string;
  targetMarket?: string;
  annualRevenueMin?: number;
  annualRevenueMax?: number;
  fundingStage?: string;
  additionalNotes?: string;
  linkedinUrl?: string;
  twitterHandle?: string;
}

type ChallengeCategory = 'growth' | 'transform' | 'transition' | 'compete' | 'funding';

type SubmissionStatus = 'pending' | 'enriching' | 'enriched' | 'analyzing' | 'completed' | 'failed';

// Wizard
interface WizardState {
  analysisId: string;
  currentStep: number;
  totalSteps: number;
  framework: FrameworkStep;
  stepStatus: 'pending' | 'generating' | 'generated' | 'approved';
  output?: Record<string, any>;
  humanContext?: string;
  humanAnswers?: Record<string, string>;
  previousSteps: StepSummary[];
  iterationCount: number;
}

interface FrameworkStep {
  step: number;
  code: string;
  name: string;
  description: string;
  questions: { id: string; question: string }[];
}

interface StepSummary {
  step: number;
  frameworkCode: string;
  frameworkName: string;
  status: string;
  approvedAt?: string;
}

// Analysis
interface AnalysisResponse {
  id: string;
  submissionId?: string;
  companyId?: string;
  challengeId: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  analysis: FrameworkResults;
  is_visible_to_user: boolean;
  is_public: boolean;
  access_code?: string;
  createdAt: string;
  updatedAt: string;
}

interface FrameworkResults {
  pestel: PESTELResult;
  porter: PorterResult;
  swot: SWOTResult;
  tam_sam_som: TAMResult;
  benchmarking: BenchmarkingResult;
  blue_ocean: BlueOceanResult;
  growth_hacking: GrowthHackingResult;
  scenarios: ScenariosResult;
  okrs: OKRsResult;
  bsc: BSCResult;
  decision_matrix: DecisionMatrixResult;
  synthesis: SynthesisResult;
}

// Error
interface ErrorResponse {
  error: string;
  message: string;
}
```

---

## Migration Notes

### From Old API (v0)

| Old Endpoint | New Endpoint | Notes |
|--------------|--------------|-------|
| `GET /submissions/:id/enrichment` | N/A | Enrichment data now on Company |
| `POST /admin/enrichment/:id/approve` | N/A | Use wizard flow |
| `GET /admin/enrichment/:id` | N/A | Enrichment is automatic |
| `POST /submissions/:id/retry-enrichment` | N/A | Enrichment is one-time |

### Key Changes

1. **Enrichment is automatic** - Runs at company creation via Perplexity
2. **Wizard replaces batch analysis** - Human-in-the-loop step-by-step
3. **Challenge is a separate entity** - Not embedded in submission
4. **Company has enriched data** - Not a separate enrichment entity
5. **Status is derived** - From company enrichment_status + analysis status

---

**Document Version**: 2.0
**Last Updated**: 2025-12-06
**Maintainer**: Backend Team
