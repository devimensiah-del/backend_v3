# IMENSIAH Business Workflows

**Complete workflow documentation for frontend integration**

This document describes all business workflows, status transitions, and integration patterns.

---

## Table of Contents

- [Submission Workflow](#submission-workflow)
- [Company Enrichment Workflow](#company-enrichment-workflow)
- [Wizard Analysis Workflow](#wizard-analysis-workflow)
- [Re-Analysis Workflow](#re-analysis-workflow)
- [Admin Approval Workflow](#admin-approval-workflow)
- [Public Sharing Workflow](#public-sharing-workflow)
- [Status Derivation Logic](#status-derivation-logic)
- [Error Handling Workflows](#error-handling-workflows)

---

## Submission Workflow

### Overview

User submits company information → Company enrichment → Wizard-based analysis → Admin review → Public sharing

### Step-by-Step Flow

```
┌─────────────────────────────────────────────────────────────────┐
│  STEP 1: USER SUBMITS FORM                                       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    POST /api/v1/submissions
                              │
                              │
         ┌────────────────────┼────────────────────┐
         │                    │                    │
         ▼                    ▼                    ▼
  Create Submission    Create Company      Create Challenge
         │                    │                    │
         │              enrichment_status:         │
         │                  "pending"              │
         │                                         │
         └────────────────────┬────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 2: INLINE COMPANY ENRICHMENT (AUTOMATIC)                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    Perplexity API Call
                 (enrichment_status: "processing")
                              │
                              │
         ┌────────────────────┴────────────────────┐
         │                                         │
         ▼                                         ▼
    SUCCESS                                    FAILURE
enrichment_status:                      enrichment_status:
  "completed"                               "failed"
         │                                enrichment_error:
         │                                  "Error details"
         │                                         │
         │                                         │
         ▼                                         ▼
  Enriched data populated                 User notified
  (foundation_year,                       Admin can retry
   competitors,
   digital_maturity, etc.)
         │
         │
┌─────────────────────────────────────────────────────────────────┐
│  STEP 3: WIZARD INITIALIZATION (MANUAL TRIGGER)                 │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
  POST /api/v1/wizard/start
  Body: { "company_id": "...", "challenge_id": "..." }
         │
         ▼
  Create Analysis record with company_id + challenge_id
  (status: "pending", wizard_mode: true)
         │
         │
┌─────────────────────────────────────────────────────────────────┐
│  STEP 4: WIZARD EXECUTION (HUMAN-IN-THE-LOOP)                  │
│  See "Wizard Analysis Workflow" section                         │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
  Analysis completed
  (status: "completed")
         │
         │
┌─────────────────────────────────────────────────────────────────┐
│  STEP 5: ADMIN REVIEW & VISIBILITY CONTROL                      │
│  See "Admin Approval Workflow" section                          │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
  is_visible_to_user: true
  is_blurred: false/true
  is_public: true/false
  access_code: "ABC12345"
         │
         │
┌─────────────────────────────────────────────────────────────────┐
│  STEP 6: USER/PUBLIC ACCESS                                     │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
  GET /api/v1/submissions/:id/analysis (authenticated)
  GET /api/v1/public/report/:code (public access code)
```

### Authentication Scenarios

#### Anonymous Submission
```
1. User submits form without login
2. System creates submission (user_id: NULL)
3. contact_email stored for later linking
4. User receives submission ID in response
5. User can check status via GET /api/v1/submissions/:id (no auth)
6. If user signs up with matching email:
   - Backend auto-links submission to user (async)
   - User can now see submission in dashboard
```

#### Authenticated Submission
```
1. User logs in (JWT token)
2. User submits form
3. System creates submission (user_id: <user UUID>)
4. Submission appears in user's dashboard immediately
5. User can track progress via GET /api/v1/submissions
```

### Data Flow

```typescript
// Frontend: Create submission
const response = await fetch('/api/v1/submissions', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    companyName: "Acme Corp",
    challengeCategory: "growth",
    challengeType: "growth_organic",
    businessChallenge: "Need to scale...",
    additionalInfo: JSON.stringify({
      contactName: "John",
      contactEmail: "john@acme.com"
    })
  })
});

// Backend creates:
// 1. Submission (id: UUID)
// 2. Company (id: UUID, enrichment_status: "pending")
// 3. Challenge (id: UUID)

// Frontend receives:
{
  "submission": {
    "id": "submission-uuid",
    "companyId": "company-uuid",
    "challengeId": "challenge-uuid",
    "createdAt": "2025-12-06T10:00:00Z"
  }
}

// Frontend can poll for enrichment completion:
GET /api/v1/companies/:companyId
// Check: enrichment_status === "completed"
```

---

## Company Enrichment Workflow

### Inline Enrichment (Default)

Enrichment happens **synchronously** at company creation:

```
Create Company
      │
      ▼
enrichment_status = "processing"
      │
      ▼
Perplexity API Call (Sonar-Pro)
      │
      │
  ┌───┴───┐
  │       │
  ▼       ▼
SUCCESS  FAILURE
      │
      ▼
enrichment_status = "completed"
      │
      ▼
Enriched data populated:
- foundation_year
- legal_name
- headquarters
- sector
- target_audience
- value_proposition
- employees_range
- revenue_estimate
- business_model
- competitors (array)
- market_share_status
- digital_maturity (1-10)
- strengths (array)
- weaknesses (array)
```

### Enrichment Data Structure

```typescript
interface EnrichedCompanyData {
  // Identity
  foundation_year?: string;
  legal_name?: string;
  headquarters?: string;

  // Business
  sector?: string;
  target_audience?: string;
  value_proposition?: string;

  // Size & Performance
  employees_range?: string;
  revenue_estimate?: string;
  business_model?: string;

  // Market Position
  competitors?: string[];
  market_share_status?: string;

  // Digital Maturity
  digital_maturity?: number; // 1-10 scale

  // SWOT Preview
  strengths?: string[];
  weaknesses?: string[];
}
```

### Enrichment Status Tracking

```typescript
// Poll for enrichment completion
async function waitForEnrichment(companyId: string) {
  const pollInterval = 2000; // 2 seconds
  const maxAttempts = 30; // 1 minute total

  for (let i = 0; i < maxAttempts; i++) {
    const company = await getCompany(companyId);

    if (company.enrichment_status === "completed") {
      return company;
    }

    if (company.enrichment_status === "failed") {
      throw new Error(company.enrichment_error || "Enrichment failed");
    }

    await sleep(pollInterval);
  }

  throw new Error("Enrichment timeout");
}
```

---

## Wizard Analysis Workflow

### Overview

Step-by-step human-in-the-loop analysis with 12 frameworks.

### Wizard Steps

```
Step 0:  Challenge Refinement (clarify business challenge)
Step 1:  PESTEL Analysis
Step 2:  Porter's 7 Forces
Step 3:  Benchmarking
Step 4:  SWOT Analysis
Step 5:  TAM-SAM-SOM
Step 6:  Blue Ocean Strategy
Step 7:  Growth Hacking (LEAP + SCALE loops)
Step 8:  Scenario Planning
Step 9:  Decision Matrix
Step 10: OKRs (90-day plan)
Step 11: Balanced Scorecard
─────────────────────────
Auto:    Synthesis (executive summary)
```

### Wizard Loop

```
┌─────────────────────────────────────────────────────────────────┐
│  WIZARD INITIALIZATION                                           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
       POST /api/v1/wizard/start
       Body: { "company_id": "...", "challenge_id": "..." }
                              │
                              ▼
                  Create Analysis record
                  (company_id + challenge_id set)
                  (wizard_mode: true)
                  (current_step: 0)
                  (status: "pending")
                              │
                              │
┌─────────────────────────────────────────────────────────────────┐
│  FOR EACH STEP (0-11):                                          │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────┐
│ 1. GET CURRENT STATE    │
│ GET /analyses/:id/wizard│
└─────────────────────────┘
    │
    ▼
  Display framework questions
  Collect human context
    │
    ▼
┌─────────────────────────┐
│ 2. GENERATE OUTPUT      │
│ POST .../wizard/generate│
└─────────────────────────┘
    │
    ▼
  AI generates framework output
  based on:
  - Company data
  - Challenge context
  - Previous frameworks
  - Human answers
    │
    ▼
  Display output to human
    │
    ├──────────────┬──────────────┐
    │              │              │
    ▼              ▼              ▼
┌─────────┐  ┌─────────┐  ┌─────────────┐
│ APPROVE │  │ REFINE  │  │ REGENERATE  │
│ & NEXT  │  │ + REGEN │  │ (NO CONTEXT)│
└─────────┘  └─────────┘  └─────────────┘
    │              │              │
    ▼              ▼              │
POST .../approve  POST .../refine │
    │              │              │
    │              ▼              │
    │    Save version snapshot    │
    │    iteration_count++        │
    │              │              │
    └──────────────┴──────────────┘
                   │
                   ▼
              current_step++
                   │
                   │
┌─────────────────────────────────────────────────────────────────┐
│  REPEAT UNTIL STEP 11 APPROVED                                  │
└─────────────────────────────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────────┐
│  AUTO-GENERATE SYNTHESIS                                        │
│  (Executive summary based on all frameworks)                    │
└─────────────────────────────────────────────────────────────────┘
                   │
                   ▼
          status = "completed"
          completed_at = NOW()
```

### Step State Transitions

```
pending → generating → generated → approved
                 │           │
                 │           └──> refine → generating
                 │
                 └──────────────> failed
```

### Frontend Integration Example

```typescript
// Example: Complete wizard flow
async function runWizard(companyId: string, challengeId: string) {
  // 1. Start wizard with company + challenge
  const { state } = await startWizard({ company_id: companyId, challenge_id: challengeId });
  let analysisId = state.analysis_id;

  // 2. Loop through all steps
  while (state.current_step < state.total_steps) {
    // Get current state
    const wizard = await getWizardState(analysisId);

    // Display questions to user
    const answers = await promptUser(wizard.framework.questions);

    // Generate output
    await generateStep(analysisId, {
      human_context: "Focus on Brazilian market",
      answers
    });

    // Get generated output
    const updated = await getWizardState(analysisId);

    // Show output to user for review
    const approved = await reviewOutput(updated.output);

    if (approved) {
      // Approve and move to next step
      await approveStep(analysisId);
    } else {
      // Request refinement
      const feedback = await getFeedback();
      await refineStep(analysisId, {
        additional_context: feedback,
        notes: "User requested changes"
      });
    }
  }

  // Synthesis is auto-generated
  console.log("Analysis complete!");
}
```

---

## Re-Analysis Workflow

### Overview

Admin can trigger new analysis for existing company with a different challenge.

### Flow

```
┌─────────────────────────────────────────────────────────────────┐
│  STEP 1: ADMIN INITIATES RE-ANALYSIS                            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
     POST /api/v1/admin/companies/:id/re-analyze
     {
       "challenge_category": "transform",
       "challenge_type": "transform_digital",
       "business_challenge": "Accelerate digital transformation"
     }
                              │
                              │
┌─────────────────────────────────────────────────────────────────┐
│  STEP 2: VALIDATE PRECONDITIONS                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
         ┌─────────────────────────────────────┐
         │ 1. Company exists?                  │
         │ 2. Enrichment completed?            │
         │    (enrichment_status === "completed") │
         └─────────────────────────────────────┘
                              │
                              │
         ┌────────────────────┴────────────────────┐
         │                                         │
         ▼                                         ▼
       PASS                                      FAIL
         │                                         │
         │                                         ▼
         │                                  Return 400 Error
         │                                  "Enrichment not completed"
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 3: CREATE NEW ENTITIES                                    │
└─────────────────────────────────────────────────────────────────┘
         │
         │
         ├──> Create new Challenge
         │    (linked to existing Company)
         │
         ├──> Create new Submission
         │    (for tracking purposes)
         │
         └──> Link Submission to Company
              (company_submissions table)
         │
         │
┌─────────────────────────────────────────────────────────────────┐
│  STEP 4: ENQUEUE ANALYSIS JOB                                   │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
  Enqueue Analysis Job (Asynq/Redis)
  Payload: {
    submission_id,
    company_id,
    challenge_id
  }
         │
         │
┌─────────────────────────────────────────────────────────────────┐
│  STEP 5: WIZARD FLOW (SAME AS NEW SUBMISSION)                  │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
  Analysis proceeds through wizard steps
  User can track via:
  - GET /api/v1/submissions/:id
  - GET /api/v1/analyses/:id/wizard
```

### Re-Analysis Use Cases

1. **Different Challenge Type**
   - Company was analyzed for "growth_organic"
   - Now needs analysis for "transform_digital"
   - Same company data, different strategic focus

2. **Updated Company Data**
   - Company updated their information
   - Admin wants fresh analysis with new context

3. **Failed Previous Analysis**
   - Previous analysis failed mid-way
   - Admin retries with corrections

### Frontend Integration

```typescript
// Admin: Re-analyze company
async function reAnalyzeCompany(companyId: string) {
  const response = await fetch(`/api/v1/admin/companies/${companyId}/re-analyze`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${adminToken}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      challenge_category: "transform",
      challenge_type: "transform_digital",
      business_challenge: "Need to accelerate digital transformation initiatives"
    })
  });

  const { data } = await response.json();
  // data.company_id - Company ID
  // data.challenge_id - New challenge ID

  // Start wizard with company + challenge
  navigateToWizard(data.company_id, data.challenge_id);
}
```

---

## Admin Approval Workflow

### Overview

Admin controls analysis visibility and sharing via 3 independent flags.

### Visibility States

```
┌─────────────────────────────────────────────────────────────────┐
│  VISIBILITY CONTROL (3 INDEPENDENT FLAGS)                       │
└─────────────────────────────────────────────────────────────────┘

  1. is_visible_to_user (boolean)
     └─> Controls if user can see analysis at all
         false: Analysis hidden from user (admin preview only)
         true:  User can view analysis

  2. is_blurred (boolean)
     └─> Controls blur overlay on premium frameworks
         false: All frameworks visible (no blur)
         true:  Premium frameworks blurred (paywall)

  3. is_public (boolean)
     └─> Controls public access via access code
         false: Access code requires login (private sharing)
         true:  Access code works without login (public sharing)
```

### Visibility Matrix

| `is_visible` | `is_public` | `is_blurred` | Authenticated User | Anonymous User |
|--------------|-------------|--------------|-------------------|----------------|
| `false` | `false` | N/A | ❌ 404 Not found | ❌ 404 Not found |
| `false` | `true` | N/A | ❌ 404 Not found | ❌ 404 Not found |
| `true` | `false` | `false` | ✅ Full access | ❌ 401 Login required |
| `true` | `false` | `true` | ✅ Blurred access | ❌ 401 Login required |
| `true` | `true` | `false` | ✅ Full access | ✅ Full access |
| `true` | `true` | `true` | ✅ Blurred access | ✅ Blurred access |

**Admin Preview:**
- Admin with `?preview=admin` can view **any** analysis regardless of flags
- Used for QA before making visible to user

### Admin Actions

```
┌─────────────────────────────────────────────────────────────────┐
│  STEP 1: REVIEW COMPLETED ANALYSIS                              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
         GET /api/v1/admin/analysis/:id
         (Returns full analysis with all flags)
                              │
                              │
┌─────────────────────────────────────────────────────────────────┐
│  STEP 2: EDIT IF NEEDED (OPTIONAL)                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
        PUT /api/v1/admin/analysis/:id
        {
          "swot": {
            "strengths": [{ "content": "...", ... }]
          }
        }
                              │
                              │
┌─────────────────────────────────────────────────────────────────┐
│  STEP 3: CONTROL VISIBILITY                                     │
└─────────────────────────────────────────────────────────────────┘
         │
         ├──> POST /admin/analysis/:id/visibility
         │    { "visible": true }
         │    (Show to user)
         │
         ├──> POST /admin/analysis/:id/blur
         │    { "blurred": false }
         │    (Disable blur overlay)
         │
         └──> POST /admin/analysis/:id/public
              { "public": true }
              (Enable public access)
                              │
                              │
┌─────────────────────────────────────────────────────────────────┐
│  STEP 4: GENERATE ACCESS CODE (IF PUBLIC)                       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
       POST /admin/analysis/:id/access-code
                              │
                              ▼
         Returns:
         {
           "access_code": "ABC12345",
           "shareable_url": "https://imenseia.com.br/report/ABC12345"
         }
                              │
                              │
┌─────────────────────────────────────────────────────────────────┐
│  STEP 5: NOTIFY USER (FRONTEND RESPONSIBILITY)                  │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
  Send email with:
  - Link to analysis
  - Access code (if public)
  - Instructions
```

### Frontend Integration

```typescript
// Admin: Complete approval workflow
async function approveAnalysis(analysisId: string) {
  // 1. Review analysis
  const analysis = await getAnalysis(analysisId);

  // 2. Make visible to user
  await toggleVisibility(analysisId, { visible: true });

  // 3. Disable blur (full access)
  await toggleBlur(analysisId, { blurred: false });

  // 4. Enable public access
  await togglePublic(analysisId, { public: true });

  // 5. Generate access code
  const { access_code, shareable_url } = await generateAccessCode(analysisId);

  // 6. Notify user
  await sendEmail({
    to: analysis.submission.contact_email,
    subject: "Seu relatório está pronto!",
    body: `Acesse em: ${shareable_url}`
  });
}
```

---

## Public Sharing Workflow

### Access Code Flow

```
┌─────────────────────────────────────────────────────────────────┐
│  ADMIN GENERATES ACCESS CODE                                    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
    POST /api/v1/admin/analysis/:id/access-code
                              │
                              ▼
         Generate 8-char code (ABC12345)
         (Retry on collision, max 5 attempts)
                              │
                              ▼
         access_code = "ABC12345"
         access_code_created_at = NOW()
                              │
                              │
┌─────────────────────────────────────────────────────────────────┐
│  USER/PUBLIC ACCESSES REPORT                                    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
     GET /api/v1/public/report/:code
     Example: GET /public/report/ABC12345
                              │
                              │
┌─────────────────────────────────────────────────────────────────┐
│  VALIDATION CHECKS                                              │
└─────────────────────────────────────────────────────────────────┘
         │
         ├──> 1. Access code valid?
         │    └─> Invalid: 404 Not Found
         │
         ├──> 2. is_visible_to_user === true?
         │    └─> false: 404 Not Found
         │
         └──> 3. is_public || authenticated?
              └─> Neither: 401 Authentication Required
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  RETURN ANALYSIS DATA                                           │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
  Return full analysis:
  - All framework results
  - is_blurred flag (frontend applies blur CSS)
  - Metadata
```

### Access Code Format

- **Length:** 8 characters
- **Character set:** A-Z, 0-9 (uppercase)
- **Example:** `ABC12345`
- **Collision handling:** Retry up to 5 times
- **Lifespan:** Permanent (no expiration)

### Blur Overlay Behavior

```typescript
// Frontend: Apply blur based on flag
function renderFramework(framework: any, isBlurred: boolean) {
  const isPremium = [
    "blue_ocean",
    "growth_hacking",
    "scenarios",
    "decision_matrix",
    "okrs",
    "bsc"
  ].includes(framework.code);

  if (isPremium && isBlurred) {
    return (
      <div className="relative">
        <div className="blur-sm">{renderFrameworkContent(framework)}</div>
        <div className="absolute inset-0 flex items-center justify-center">
          <button onClick={handleUpgrade}>
            Upgrade para ver completo
          </button>
        </div>
      </div>
    );
  }

  return renderFrameworkContent(framework);
}
```

---

## Status Derivation Logic

### Submission Status (Derived)

Submission doesn't have a `status` column. Status is derived from related entities:

```typescript
function deriveSubmissionStatus(
  submission: Submission,
  company?: Company,
  analysis?: Analysis
): SubmissionStatus {
  // No company created yet
  if (!company) {
    return "pending";
  }

  // Company enrichment in progress
  if (company.enrichment_status === "processing") {
    return "enriching";
  }

  // Company enrichment failed
  if (company.enrichment_status === "failed") {
    return "failed";
  }

  // Company enriched, but no analysis started
  if (!analysis) {
    return "enriched";
  }

  // Analysis in progress
  if (analysis.status === "pending" || analysis.status === "processing") {
    return "analyzing";
  }

  // Analysis failed
  if (analysis.status === "failed") {
    return "failed";
  }

  // Analysis completed
  if (analysis.status === "completed") {
    return "completed";
  }

  return "pending";
}
```

### Status Transition Diagram

```
         ┌───────────┐
         │  pending  │
         └─────┬─────┘
               │
               ▼
        ┌──────────────┐
        │  enriching   │
        └──┬────────┬──┘
           │        │
           ▼        ▼
      ┌─────────┐  ┌────────┐
      │enriched │  │ failed │
      └────┬────┘  └────────┘
           │
           ▼
      ┌──────────┐
      │analyzing │
      └──┬───┬───┘
         │   │
         ▼   ▼
    ┌─────────┐  ┌────────┐
    │completed│  │ failed │
    └─────────┘  └────────┘
```

---

## Error Handling Workflows

### Enrichment Failure Recovery

```
┌─────────────────────────────────────────────────────────────────┐
│  ENRICHMENT FAILS                                               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
         company.enrichment_status = "failed"
         company.enrichment_error = "Rate limit exceeded"
                              │
                              │
┌─────────────────────────────────────────────────────────────────┐
│  ADMIN NOTIFICATION                                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
         Admin dashboard shows:
         - Failed enrichment
         - Error message
         - Retry button
                              │
                              │
┌─────────────────────────────────────────────────────────────────┐
│  ADMIN RETRIES (MANUAL)                                         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
      POST /admin/companies/:id/retry-enrichment
      (Not implemented yet - manual fix required)
                              │
                              ▼
         Alternative: Delete company, re-submit
```

### Analysis Failure Recovery

```
┌─────────────────────────────────────────────────────────────────┐
│  ANALYSIS FAILS                                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
         analysis.status = "failed"
         analysis.error_message = "LLM timeout"
                              │
                              │
┌─────────────────────────────────────────────────────────────────┐
│  ADMIN NOTIFICATION                                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
         Admin dashboard shows:
         - Failed analysis
         - Error message
         - Retry button
                              │
                              │
┌─────────────────────────────────────────────────────────────────┐
│  ADMIN RETRIES                                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
      POST /admin/submissions/:id/retry-analysis
                              │
                              ▼
         Creates new analysis job
         Reuses same company/challenge
                              │
                              ▼
         Analysis proceeds through wizard
```

### Wizard Step Failure

```
┌─────────────────────────────────────────────────────────────────┐
│  STEP GENERATION FAILS                                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
         step.status = "failed"
         step.error_message = "Token limit exceeded"
                              │
                              │
┌─────────────────────────────────────────────────────────────────┐
│  USER NOTIFICATION                                              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
         Frontend displays:
         "Erro ao gerar framework: [error_message]"
         "Tente novamente com contexto mais específico"
                              │
                              │
┌─────────────────────────────────────────────────────────────────┐
│  USER RETRIES                                                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
      POST /analyses/:id/wizard/generate
      (With different human_context)
                              │
                              ▼
         Retry generation
         step.iteration_count++
```

---

## Polling Patterns

### Poll for Enrichment Completion

```typescript
async function pollEnrichment(companyId: string, maxAttempts = 30) {
  for (let i = 0; i < maxAttempts; i++) {
    const company = await getCompany(companyId);

    if (company.enrichment_status === "completed") {
      return company;
    }

    if (company.enrichment_status === "failed") {
      throw new Error(company.enrichment_error);
    }

    await sleep(2000); // 2 seconds
  }

  throw new Error("Enrichment timeout");
}
```

### Poll for Analysis Progress

```typescript
async function pollWizardProgress(analysisId: string) {
  while (true) {
    const state = await getWizardState(analysisId);

    if (state.status === "completed") {
      return state;
    }

    if (state.status === "failed") {
      throw new Error("Wizard failed");
    }

    // Update UI with current step
    updateProgress(state.current_step, state.total_steps);

    await sleep(3000); // 3 seconds
  }
}
```

---

**Workflow Documentation Version:** 1.0
**Last Updated:** 2025-12-06
**Compatible with API:** v1
**Maintained By:** IMENSIAH Engineering Team
