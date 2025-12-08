# Submission Domain

## Purpose

Entry point for the analysis pipeline. Captures company information and business challenges submitted by users.

## File Structure

```
submission/
├── model.go           # Entity definition + validation
├── types.go           # Input/output types, constants, interfaces
├── errors.go          # Domain errors (ErrNotFound, ErrValidation, etc.)
├── repository.go      # Database access with transaction support
├── service.go         # All business logic (CRUD + SubmitForm workflow)
├── service_test.go    # Unit tests with mocks
└── repository_test.go # Repository tests with sqlmock
```

## Data Flow (Saga Pattern)

```
Frontend Form                        Service                           Database
      │                                │                                 │
      ├── POST /api/v1/submit ────────►│                                 │
      │                                │                                 │
      │                                ├── Validate challenge fields     │
      │                                │                                 │
      │                                ├── Step 1: Create Submission ───►│
      │                                │                                 │
      │                                ├── Step 2: Create Company ──────►│
      │                                │   (triggers enrichment async)   │
      │                                │                                 │
      │                                ├── Step 3: Create Challenge ────►│
      │                                │                                 │
      │◄── {submission_id, company_id, challenge_id} ────────────────────┤
      │                                │                                 │
      │     ON FAILURE AT ANY STEP:    │                                 │
      │     rollbackSaga() deletes     │                                 │
      │     previously created entities │                                 │
```

**Note:** Uses saga pattern (not DB transaction) because each step involves different domains with their own business logic. Rollback deletes entities in reverse order: Challenge → Company → Submission.

## Business Rules (INVARIANTS)

### MUST
- Required fields: `company_name`, `contact_name`, `contact_email`
- Challenge fields are ALL REQUIRED: `challenge_category`, `challenge_type`, `business_challenge`
- Challenge type MUST match its category (e.g., `growth_organic` for `growth`)
- Revenue range: if both provided, `min <= max`
- Company record is created automatically via company service
- Challenge record is created automatically via challenge service
- All three entities (Submission, Company, Challenge) created atomically via saga

### NEVER
- Expose raw database errors to API callers (use domain errors)
- Allow invalid challenge category/type combinations
- Skip validation before persisting
- Hard delete submissions (soft delete only via `deleted_at`)

## Challenge Data (Source of Truth)

**IMPORTANT:** Challenge category and type are stored in the `challenges` table, NOT on submissions.

The Submission entity:
- Stores company info and contact details
- Does NOT have `challenge_category` or `challenge_type` fields
- Links to Challenge entity via `challenges.company_id`

For challenge metadata, query the `challenges` table:
```sql
SELECT c.challenge_category, c.challenge_type, c.business_challenge
FROM challenges c
WHERE c.company_id = (
    SELECT cs.company_id FROM company_submissions cs WHERE cs.submission_id = $1
);
```

## Challenge Categories & Types

| Category | Valid Types |
|----------|-------------|
| `growth` | `growth_organic`, `growth_geographic`, `growth_segment`, `growth_product`, `growth_channel` |
| `transform` | `transform_digital`, `transform_model`, `transform_culture`, `transform_operational` |
| `transition` | `transition_succession`, `transition_exit`, `transition_merger`, `transition_turnaround` |
| `compete` | `compete_differentiate`, `compete_defend`, `compete_reposition` |
| `funding` | `funding_raise`, `funding_debt`, `funding_ipo` |

**NOTE:** Challenge validation is defined in and delegated to the `challenge` domain package.
The submission service uses `ChallengeServiceInterface.ValidateCategory()` and `ValidateType()` methods.

## Key Types

| Type | File | Purpose |
|------|------|---------|
| `Submission` | model.go | Main entity with all company/contact fields |
| `SubmitRequest` | types.go | Public form input (includes challenge fields) |
| `SubmitFormResponse` | types.go | Response with created entity IDs |
| `ListOptions` | types.go | Pagination and filtering for List queries |
| `CreateFromCompanyInput` | types.go | Input for re-analyze workflow |
| `CompanyServiceInterface` | types.go | Interface for company service (avoids import cycle) |
| `ChallengeServiceInterface` | types.go | Interface for challenge service (avoids import cycle) |
| `ValidationError` | errors.go | Field-level validation failures |
| `RepositoryError` | errors.go | Wrapped DB errors (hides internals) |
| `WorkflowError` | errors.go | Multi-step workflow failures |

## Service Methods

| Method | Purpose |
|--------|---------|
| `NewService()` | Constructor |
| `SetCompanyService()` | Inject company service dependency |
| `SetChallengeService()` | Inject challenge service dependency |
| `SubmitForm()` | Main workflow: creates Submission → Company → Challenge |
| `Create()` | Low-level submission creation (prefer SubmitForm) |
| `GetByID()` | Get submission by ID |
| `GetByEmail()` | Get submissions by contact email |
| `List()` | List with pagination/filtering |
| `Delete()` | Soft delete submission |
| `CreateFromCompany()` | Create submission from existing company (re-analyze) |
| `LinkAnonymousToUser()` | Link anonymous submissions after signup |

## Error Handling

```go
// Check error types
if submission.IsNotFound(err) { ... }
if submission.IsValidation(err) { ... }

// Validation errors include field name
var ve *submission.ValidationError
if errors.As(err, &ve) {
    fmt.Printf("Field %s: %s", ve.Field, ve.Message)
}

// Repository errors hide DB details
var re *submission.RepositoryError
if errors.As(err, &re) {
    log.Error().Err(re.Internal()).Msg("DB error")  // For logging
    return re  // Safe to return to API (no SQL details)
}
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/submit` | POST | Create submission (public) |
| `/api/v1/submissions` | GET | List user's submissions (protected) |
| `/api/v1/submissions/:id` | GET | Get submission details (protected) |
| `/api/v1/admin/submissions` | GET | List all submissions (admin) |

## Saga Rollback

If any step fails during `SubmitForm()`, previously created entities are deleted in reverse order:

1. Delete Challenge (if created)
2. Delete Company (if created) - includes unlinking from `company_submissions`
3. Delete Submission (if created)

Rollback is best-effort - failures are logged but don't propagate.

## AI Agent Warnings

### DO NOT
- Add status field to Submission (workflow status derived from Company/Analysis)
- Skip challenge validation
- Skip company/challenge creation in SubmitForm
- Expose SQL errors to API callers
- Replace saga pattern with DB transaction (cross-domain logic needs saga)

### SAFE TO MODIFY
- Add new optional fields (update model.go + repository.go)
- Add new challenge categories/types (update types.go)
- Modify validation rules (service.go)
- Add new query methods to repository
