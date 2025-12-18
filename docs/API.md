# IMENSIAH API Reference

> **Version**: v3.0 (IAH-3) | **Base URL**: `/api/v1` | **Updated**: 2025-12-14

Complete API reference for frontend developers. All routes are under `/api/v1` unless noted.

---

## Quick Reference

| Category | Endpoints | Auth |
|----------|-----------|------|
| [Auth](#auth) | 7 | Mixed |
| [Submissions](#submissions) | 4 | Mixed |
| [Companies](#companies) | 3 | Required |
| [**Analysis By Steps (IAH-3)**](#analysis-by-steps-iah-3---human-editable-analysis) | 6 | Required |
| [Frameworks](#frameworks) | 1 | Required |
| [Public Report](#public-report) | 1 | Optional |
| [Admin](#admin-endpoints) | 13 | Admin |
| [Wizard (Legacy)](#wizard-legacy--deprecated) | - | Deprecated |

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

## Analysis By Steps (IAH-3) - Human-Editable Analysis

> **⚠️ IMPORTANT**: This is the NEW step-by-step analysis API (IAH-3). It replaces the legacy Wizard API.

The Analysis By Steps feature enables step-by-step strategic analysis with **direct human editing** of AI-generated outputs. Unlike the legacy Wizard (which only allowed adding context for regeneration), this API allows users to directly edit the JSON output of each framework.

### 🎯 Key Concepts for Frontend Developers

1. **14 Steps** (0-13): Each step corresponds to a strategic framework
2. **Strict Order**: Cannot generate step N until step N-1 is **approved**
3. **Direct Editing**: Human can directly edit AI output (not just add context)
4. **Effective Output**: Always use `effective_output` field (returns human edit if exists, else AI output)
5. **Idempotent Start**: Safe to call `/start` multiple times - returns existing analysis

### 📋 The 14 Frameworks (Step Order)

| Step | Code | Name (PT-BR) | Guidance Text |
|------|------|--------------|---------------|
| 0 | `challenge_refinement` | Refinamento do Desafio | Revise se o desafio está claro e específico |
| 1 | `pestel` | Análise PESTEL | Considere quais fatores externos impactam |
| 2 | `porter` | 5 Forças de Porter | Avalie a intensidade competitiva |
| 3 | `benchmarking` | Benchmarking | Os players comparados são relevantes? |
| 4 | `swot` | Análise SWOT | As forças listadas geram valor? |
| 5 | `swotcross` | SWOT Cruzado | As estratégias cruzadas são viáveis? |
| 6 | `tam_sam_som` | TAM-SAM-SOM | O dimensionamento está realista? |
| 7 | `blue_ocean` | Blue Ocean | A curva de valor diferencia? |
| 8 | `growth_hacking` | Growth Hacking | Táticas aplicáveis ao estágio atual? |
| 9 | `scenarios` | Cenários | Os cenários cobrem riscos relevantes? |
| 10 | `decision_matrix` | Matriz de Decisão | Critérios refletem prioridades? |
| 11 | `okrs` | OKRs | Objetivos ambiciosos mas alcançáveis? |
| 12 | `bsc` | Balanced Scorecard | Perspectivas balanceadas? |
| 13 | `synthesis` | Síntese Executiva | Captura as principais conclusões? |

### 🔄 Step Status State Machine

```
pending ──► generating ──► generated ──┬──► approved
                │                       │
                │         (human edits) │
                ▼                       │
              failed                    │
                                        │
              ◄─────────────────────────┘
        (can re-generate after approved,
         resets status to "generated")
```

**Status Values:**
| Status | Meaning | Can Generate? | Can Approve? | Can Edit? |
|--------|---------|---------------|--------------|-----------|
| `pending` | Step created, AI not called | ✅ Yes | ❌ No | ❌ No |
| `generating` | LLM request in progress | ❌ No | ❌ No | ❌ No |
| `generated` | AI output ready for review | ✅ Yes (re-gen) | ✅ Yes | ✅ Yes |
| `approved` | Human approved the output | ✅ Yes (re-gen) | ❌ Already | ✅ Yes |
| `failed` | AI generation failed | ✅ Yes (retry) | ❌ No | ❌ No |

### 🚨 Order Enforcement (CRITICAL)

**The API enforces strict sequential order:**

```
❌ REJECTED: Generate step 3 when step 2 is not approved
   Error: "cannot generate step 3: previous step 2 (porter) is not approved yet"

✅ ALLOWED: Generate step 0 (first step, no prerequisites)
✅ ALLOWED: Generate step 3 when steps 0, 1, 2 are approved
✅ ALLOWED: Re-generate step 0 even if already generated/approved
```

**Frontend UX Implication**:
- Disable "Generate" button for step N if step N-1 is not approved
- Show clear message: "Complete previous step first"

---

### Endpoints

Base path: `/api/v1/analyses`

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/steps/start` | Start new analysis (returns all 14 steps) |
| POST | `/:id/steps/:step/generate` | Generate AI output for step |
| PUT | `/:id/steps/:step/edit` | Save human edit |
| POST | `/:id/steps/:step/approve` | Approve step and advance |
| GET | `/:id/steps/state` | Get current state (for UI) |
| GET | `/:id/steps` | Get all steps |

---

### `POST /analyses/steps/start` *(Auth Required)*

Start a new step-by-step analysis for a challenge. **Idempotent** - safe to call multiple times.

**Request:**
```json
{
  "challenge_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response `200`:**
```json
{
  "analysis_id": "660e8400-e29b-41d4-a716-446655440001",
  "challenge_id": "550e8400-e29b-41d4-a716-446655440000",
  "total_steps": 14,
  "current_step": 0,
  "steps": [
    {
      "id": "step-uuid-0",
      "analysis_id": "660e8400-e29b-41d4-a716-446655440001",
      "framework_code": "challenge_refinement",
      "step_number": 0,
      "status": "pending",
      "visible": true,
      "created_at": "2025-01-01T10:00:00Z",
      "updated_at": "2025-01-01T10:00:00Z",
      "is_edited": false
    },
    {
      "id": "step-uuid-1",
      "analysis_id": "660e8400-e29b-41d4-a716-446655440001",
      "framework_code": "pestel",
      "step_number": 1,
      "status": "pending",
      "visible": true,
      "created_at": "2025-01-01T10:00:00Z",
      "updated_at": "2025-01-01T10:00:00Z",
      "is_edited": false
    }
    // ... 12 more steps (steps 2-13)
  ]
}
```

**Error Responses:**
| Code | Error | When |
|------|-------|------|
| 400 | `challenge_id must be a valid UUID` | Invalid UUID format |
| 404 | `challenge not found` | Challenge doesn't exist |

---

### `POST /analyses/:id/steps/:step/generate` *(Auth Required)*

Generate AI output for a specific step. Requires all previous steps to be approved.

**Path Parameters:**
- `id` - Analysis UUID
- `step` - Step number (0-13)

**Request:** No body required

**Response `200`:**
```json
{
  "id": "step-uuid-1",
  "analysis_id": "660e8400-e29b-41d4-a716-446655440001",
  "framework_code": "pestel",
  "step_number": 1,
  "ai_output": "{\"political\":[...],\"economic\":[...],\"social\":[...]}",
  "status": "generated",
  "visible": true,
  "generated_at": "2025-01-01T10:05:00Z",
  "created_at": "2025-01-01T10:00:00Z",
  "updated_at": "2025-01-01T10:05:00Z",
  "effective_output": "{\"political\":[...],\"economic\":[...],\"social\":[...]}",
  "is_edited": false
}
```

**Error Responses:**
| Code | Error | When |
|------|-------|------|
| 400 | `Step number must be 0-13` | Invalid step number |
| 400 | `cannot generate step N: previous step N-1 (code) is not approved yet` | Order violation |
| 404 | `analysis not found` | Analysis doesn't exist |

**⚠️ IMPORTANT for Frontend:**
```typescript
// Before calling generate, check if previous step is approved:
if (stepNumber > 0) {
  const prevStep = steps[stepNumber - 1];
  if (prevStep.status !== 'approved') {
    showError('Complete previous step first');
    return;
  }
}
```

---

### `PUT /analyses/:id/steps/:step/edit` *(Auth Required)*

Save human edits to a step. Does NOT regenerate - just saves the edit.

**Path Parameters:**
- `id` - Analysis UUID
- `step` - Step number (0-13)

**Request:**
```json
{
  "edited_content": "{\"political\":[\"Updated point 1\"],\"economic\":[\"Updated point 2\"]}"
}
```

**⚠️ CRITICAL**: `edited_content` must be valid JSON string. The backend validates JSON structure.

**Response `200`:**
```json
{
  "id": "step-uuid-1",
  "analysis_id": "660e8400-e29b-41d4-a716-446655440001",
  "framework_code": "pestel",
  "step_number": 1,
  "ai_output": "{\"political\":[\"Original AI output\"]}",
  "human_edited": "{\"political\":[\"Updated point 1\"],\"economic\":[\"Updated point 2\"]}",
  "status": "generated",
  "visible": true,
  "generated_at": "2025-01-01T10:05:00Z",
  "created_at": "2025-01-01T10:00:00Z",
  "updated_at": "2025-01-01T10:07:00Z",
  "effective_output": "{\"political\":[\"Updated point 1\"],\"economic\":[\"Updated point 2\"]}",
  "is_edited": true
}
```

**Error Responses:**
| Code | Error | When |
|------|-------|------|
| 400 | `invalid JSON` | `edited_content` is not valid JSON |
| 404 | `Step not found` | Step doesn't exist |

**Frontend Integration Pattern:**
```typescript
// Parse effective_output to edit
const content = JSON.parse(step.effective_output);

// User edits in form...

// Save back as JSON string
const edited = JSON.stringify(formValues);
await api.put(`/analyses/${analysisId}/steps/${stepNumber}/edit`, {
  edited_content: edited
});
```

---

### `POST /analyses/:id/steps/:step/approve` *(Auth Required)*

Approve the current step and advance to the next. Step must have content (AI or human edited).

**Path Parameters:**
- `id` - Analysis UUID
- `step` - Step number (0-13)

**Request:** No body required

**Response `200`:**
```json
{
  "approved_step": {
    "id": "step-uuid-1",
    "framework_code": "pestel",
    "step_number": 1,
    "status": "approved",
    "approved_at": "2025-01-01T10:10:00Z",
    "effective_output": "{...}",
    "is_edited": true
  },
  "next_step": {
    "id": "step-uuid-2",
    "framework_code": "porter",
    "step_number": 2,
    "status": "pending"
  },
  "is_complete": false,
  "current_step": 2
}
```

**When last step (13 - synthesis) is approved:**
```json
{
  "approved_step": {
    "id": "step-uuid-13",
    "framework_code": "synthesis",
    "step_number": 13,
    "status": "approved",
    "approved_at": "2025-01-01T12:00:00Z"
  },
  "is_complete": true,
  "current_step": 13
}
```

**Error Responses:**
| Code | Error | When |
|------|-------|------|
| 400 | `cannot approve step: no content` | Step has no AI or human output |
| 404 | `Step not found` | Step doesn't exist |

---

### `GET /analyses/:id/steps/state` *(Auth Required)*

Get the current state of the analysis. **Use this for UI rendering.**

**Path Parameters:**
- `id` - Analysis UUID

**Response `200`:**
```json
{
  "analysis_id": "660e8400-e29b-41d4-a716-446655440001",
  "current_step": 2,
  "total_steps": 14,
  "current_step_data": {
    "id": "step-uuid-2",
    "framework_code": "porter",
    "step_number": 2,
    "status": "pending",
    "visible": true,
    "is_edited": false
  },
  "previous_steps": [
    {
      "id": "step-uuid-0",
      "framework_code": "challenge_refinement",
      "step_number": 0,
      "status": "approved",
      "approved_at": "2025-01-01T10:05:00Z",
      "effective_output": "{...}",
      "is_edited": false
    },
    {
      "id": "step-uuid-1",
      "framework_code": "pestel",
      "step_number": 1,
      "status": "approved",
      "approved_at": "2025-01-01T10:10:00Z",
      "effective_output": "{...}",
      "is_edited": true
    }
  ],
  "framework_meta": {
    "Code": "porter",
    "Name": "5 Forças de Porter",
    "GuidanceText": "Avalie a intensidade competitiva. Os concorrentes listados estão corretos? Algum foi esquecido?"
  }
}
```

**Frontend Usage:**
```typescript
// Fetch state on page load
const state = await api.get(`/analyses/${analysisId}/steps/state`);

// Render progress bar
const progress = (state.current_step / state.total_steps) * 100;

// Show guidance text to user
showGuidance(state.framework_meta.GuidanceText);

// Show previous steps as read-only accordion
state.previous_steps.forEach(step => {
  renderPreviousStep(step);
});

// Show current step form
if (state.current_step_data) {
  renderCurrentStepForm(state.current_step_data);
}
```

---

### `GET /analyses/:id/steps` *(Auth Required)*

Get all 14 steps for an analysis.

**Path Parameters:**
- `id` - Analysis UUID

**Response `200`:**
```json
{
  "steps": [
    {
      "id": "step-uuid-0",
      "framework_code": "challenge_refinement",
      "step_number": 0,
      "status": "approved",
      "effective_output": "{...}",
      "is_edited": false
    },
    {
      "id": "step-uuid-1",
      "framework_code": "pestel",
      "step_number": 1,
      "status": "generated",
      "effective_output": "{...}",
      "is_edited": true
    },
    {
      "id": "step-uuid-2",
      "framework_code": "porter",
      "step_number": 2,
      "status": "pending",
      "is_edited": false
    }
    // ... 11 more steps
  ]
}
```

---

### 🎨 Complete Frontend Integration Example

```typescript
// types.ts
interface AnalysisStep {
  id: string;
  analysis_id: string;
  framework_code: string;
  step_number: number;
  ai_output?: string;
  human_edited?: string;
  effective_output?: string;
  visible: boolean;
  status: 'pending' | 'generating' | 'generated' | 'approved' | 'failed';
  generated_at?: string;
  approved_at?: string;
  created_at: string;
  updated_at: string;
  is_edited: boolean;
}

interface StepState {
  analysis_id: string;
  current_step: number;
  total_steps: number;
  current_step_data?: AnalysisStep;
  previous_steps: AnalysisStep[];
  framework_meta?: {
    Code: string;
    Name: string;
    GuidanceText: string;
  };
}

// api.ts
const analysisByStepsApi = {
  // Start new analysis
  start: (challengeId: string) =>
    fetch('/api/v1/analyses/steps/start', {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ challenge_id: challengeId })
    }),

  // Get current state (use this for UI)
  getState: (analysisId: string) =>
    fetch(`/api/v1/analyses/${analysisId}/steps/state`, {
      headers: { 'Authorization': `Bearer ${token}` }
    }),

  // Generate AI output for step
  generate: (analysisId: string, stepNumber: number) =>
    fetch(`/api/v1/analyses/${analysisId}/steps/${stepNumber}/generate`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}` }
    }),

  // Save human edit
  edit: (analysisId: string, stepNumber: number, editedContent: object) =>
    fetch(`/api/v1/analyses/${analysisId}/steps/${stepNumber}/edit`, {
      method: 'PUT',
      headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ edited_content: JSON.stringify(editedContent) })
    }),

  // Approve and advance
  approve: (analysisId: string, stepNumber: number) =>
    fetch(`/api/v1/analyses/${analysisId}/steps/${stepNumber}/approve`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}` }
    }),
};

// StepWorkflow.tsx - React component example
function StepWorkflow({ analysisId }: { analysisId: string }) {
  const [state, setState] = useState<StepState | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    loadState();
  }, [analysisId]);

  const loadState = async () => {
    const res = await analysisByStepsApi.getState(analysisId);
    setState(await res.json());
  };

  const handleGenerate = async () => {
    if (!state?.current_step_data) return;

    setLoading(true);
    try {
      await analysisByStepsApi.generate(analysisId, state.current_step);
      await loadState(); // Refresh UI
    } catch (err) {
      // Handle "previous step not approved" error
      alert(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleSaveEdit = async (editedContent: object) => {
    if (!state?.current_step_data) return;

    await analysisByStepsApi.edit(analysisId, state.current_step, editedContent);
    await loadState();
  };

  const handleApprove = async () => {
    if (!state?.current_step_data) return;

    const res = await analysisByStepsApi.approve(analysisId, state.current_step);
    const result = await res.json();

    if (result.is_complete) {
      // Redirect to completed analysis view
      router.push(`/analysis/${analysisId}/complete`);
    } else {
      await loadState();
    }
  };

  return (
    <div>
      {/* Progress bar */}
      <ProgressBar
        current={state?.current_step ?? 0}
        total={state?.total_steps ?? 14}
      />

      {/* Previous steps (read-only) */}
      <Accordion>
        {state?.previous_steps.map(step => (
          <AccordionItem key={step.id} title={`${step.step_number}. ${step.framework_code}`}>
            <pre>{step.effective_output}</pre>
            {step.is_edited && <Badge>Edited</Badge>}
          </AccordionItem>
        ))}
      </Accordion>

      {/* Current step */}
      {state?.current_step_data && (
        <CurrentStepCard
          step={state.current_step_data}
          meta={state.framework_meta}
          onGenerate={handleGenerate}
          onSaveEdit={handleSaveEdit}
          onApprove={handleApprove}
          loading={loading}
        />
      )}
    </div>
  );
}
```

---

### 🔁 Re-Generation Behavior

**Important**: Re-generating an already-approved step **resets its status** to `generated`:

```
Step 0: approved → call generate → status becomes "generated"
                                   → requires re-approval before step 1
```

**Frontend should warn users:**
```typescript
const handleRegenerate = async () => {
  if (step.status === 'approved') {
    const confirmed = await confirm(
      'Re-generating will require re-approval. Continue?'
    );
    if (!confirmed) return;
  }
  await generate();
};
```

---

## Wizard (Legacy) ⚠️ DEPRECATED

> **Note**: The legacy Wizard API is deprecated. Use [Analysis By Steps (IAH-3)](#analysis-by-steps-iah-3---human-editable-analysis) instead.

The legacy wizard endpoints remain available for backwards compatibility but will be removed in a future version.

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
│                    USER FLOW (IAH-3)                             │
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
│  [Start Analysis By Steps]                                       │
│       │                                                          │
│       ▼                                                          │
│  POST /analyses/steps/start { challenge_id } ► Creates Analysis  │
│       │                                        + 14 steps        │
│       ▼                                                          │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  STEP-BY-STEP LOOP (14 steps: 0-13)                      │    │
│  │                                                          │    │
│  │  1. GET /analyses/:id/steps/state   ◄── Get current UI   │    │
│  │  2. POST /.../steps/:n/generate     ◄── Generate AI      │    │
│  │  3. Review AI output                                     │    │
│  │  4. PUT /.../steps/:n/edit          ◄── Edit JSON (opt)  │    │
│  │  5. POST /.../steps/:n/approve      ◄── Approve & next   │    │
│  │                                                          │    │
│  │  ⚠️ Order enforced: step N requires N-1 approved         │    │
│  │                                                          │    │
│  │  Repeat until step 13 (synthesis) approved               │    │
│  └─────────────────────────────────────────────────────────┘    │
│       │                                                          │
│       ▼                                                          │
│  [Analysis Complete] ─────► Status: "completed"                  │
│       │                                                          │
│       ▼                                                          │
│  GET /submissions/:id/analysis ─────► Full analysis data         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│              STEP STATE MACHINE (each step)                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│    pending ──► generating ──► generated ──┬──► approved          │
│                     │                      │                     │
│                     ▼                      │ (edit optional)     │
│                   failed                   │                     │
│                                            │                     │
│              ◄─────────────────────────────┘                     │
│         (re-generate resets to "generated")                      │
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

// ==================== Analysis By Steps (IAH-3) ====================

// Step status values
type StepStatus = 'pending' | 'generating' | 'generated' | 'approved' | 'failed';

// Individual analysis step
interface AnalysisStep {
  id: string;
  analysis_id: string;
  framework_code: FrameworkCode;
  step_number: number;                    // 0-13
  ai_output?: string;                     // JSON string from AI
  human_edited?: string;                  // JSON string from human edit
  effective_output?: string;              // human_edited ?? ai_output (USE THIS)
  visible: boolean;
  status: StepStatus;
  generated_at?: string;                  // ISO timestamp
  approved_at?: string;                   // ISO timestamp
  created_at: string;
  updated_at: string;
  is_edited: boolean;                     // true if human_edited exists
}

// Framework codes in order (0-13)
type FrameworkCode =
  | 'challenge_refinement'  // Step 0
  | 'pestel'                // Step 1
  | 'porter'                // Step 2
  | 'benchmarking'          // Step 3
  | 'swot'                  // Step 4
  | 'swotcross'             // Step 5
  | 'tam_sam_som'           // Step 6
  | 'blue_ocean'            // Step 7
  | 'growth_hacking'        // Step 8
  | 'scenarios'             // Step 9
  | 'decision_matrix'       // Step 10
  | 'okrs'                  // Step 11
  | 'bsc'                   // Step 12
  | 'synthesis';            // Step 13

// Framework metadata
interface FrameworkMeta {
  Code: FrameworkCode;
  Name: string;                           // Portuguese display name
  GuidanceText: string;                   // Human reflection prompt
}

// POST /analyses/steps/start - Request
interface StartAnalysisRequest {
  challenge_id: string;                   // UUID
}

// POST /analyses/steps/start - Response
interface StartAnalysisResponse {
  analysis_id: string;
  challenge_id: string;
  total_steps: number;                    // Always 14
  current_step: number;                   // Initially 0
  steps: AnalysisStep[];                  // All 14 steps (pending)
}

// GET /analyses/:id/steps/state - Response (USE THIS FOR UI)
interface StepStateResponse {
  analysis_id: string;
  current_step: number;                   // First non-approved step
  total_steps: number;                    // Always 14
  current_step_data?: AnalysisStep;       // Current step to work on
  previous_steps: AnalysisStep[];         // All approved steps (read-only)
  framework_meta?: FrameworkMeta;         // Metadata for current step
}

// PUT /analyses/:id/steps/:step/edit - Request
interface SaveHumanEditRequest {
  edited_content: string;                 // MUST be valid JSON string
}

// POST /analyses/:id/steps/:step/approve - Response
interface ApproveStepResponse {
  approved_step: AnalysisStep;
  next_step?: AnalysisStep;               // Undefined if last step
  is_complete: boolean;                   // True when synthesis approved
  current_step: number;
}

// GET /analyses/:id/steps - Response
interface GetAllStepsResponse {
  steps: AnalysisStep[];                  // All 14 steps
}

// ==================== Wizard (DEPRECATED) ====================
// Use Analysis By Steps (IAH-3) types above instead

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

### From Wizard API to Analysis By Steps (IAH-3)

| Old Wizard Endpoint | New IAH-3 Endpoint | Notes |
|---------------------|-------------------|-------|
| `POST /wizard/start` | `POST /analyses/steps/start` | Takes `challenge_id` only (not company_id) |
| `GET /analyses/:id/wizard` | `GET /analyses/:id/steps/state` | Returns `StepStateResponse` |
| `POST /analyses/:id/wizard/generate` | `POST /analyses/:id/steps/:step/generate` | Step number in URL path |
| `POST /analyses/:id/wizard/approve` | `POST /analyses/:id/steps/:step/approve` | Step number in URL path |
| `POST /analyses/:id/wizard/refine` | `PUT /analyses/:id/steps/:step/edit` | Direct JSON editing (no regeneration) |
| N/A | `GET /analyses/:id/steps` | Get all 14 steps |

### Key Differences

| Feature | Wizard (Deprecated) | Analysis By Steps (IAH-3) |
|---------|---------------------|---------------------------|
| Human input | Add context → AI regenerates | Direct JSON editing |
| Steps | 12 frameworks | 14 frameworks (+swotcross, challenge_refinement) |
| Storage | `analysis_steps` table | `analysis_steps_v2` table |
| Output fields | Single `output` | `ai_output` + `human_edited` + `effective_output` |
| Edit detection | N/A | `is_edited` boolean field |

### From Old API (v0)

| Old Endpoint | New Endpoint | Notes |
|--------------|--------------|-------|
| `GET /submissions/:id/enrichment` | N/A | Enrichment data now on Company |
| `POST /admin/enrichment/:id/approve` | N/A | Use Analysis By Steps flow |
| `GET /admin/enrichment/:id` | N/A | Enrichment is automatic |
| `POST /submissions/:id/retry-enrichment` | N/A | Enrichment is one-time |

### Key Changes

1. **Enrichment is automatic** - Runs at company creation via Perplexity
2. **Analysis By Steps replaces Wizard** - Direct human editing of JSON outputs
3. **Challenge is a separate entity** - Not embedded in submission
4. **Company has enriched data** - Not a separate enrichment entity
5. **Status is derived** - From company enrichment_status + analysis status
6. **14 frameworks** - Added `challenge_refinement` (step 0) and `swotcross` (step 5)

---

**Document Version**: 3.0 (IAH-3)
**Last Updated**: 2025-12-14
**Maintainer**: Backend Team
