# Challenge Domain

## Purpose

The Challenge domain represents strategic business challenges that companies face. A challenge is a specific problem or opportunity that requires analysis and strategic planning.

## Domain Model

```
Company (1) → Challenge (many) → Analysis (many)
```

- A **Company** can have multiple challenges over time
- Each **Challenge** represents a distinct strategic problem/opportunity
- Each **Challenge** can have multiple analyses (e.g., re-runs with different parameters)

## File Structure

```
challenge/
├── model.go       # Entity definition + validation + constants
├── repository.go  # Database access (PostgreSQL)
└── service.go     # Business logic
```

## Entity Structure

### Challenge

A Challenge contains:
- **ID**: Unique identifier (UUID)
- **CompanyID**: Reference to the company facing this challenge (required)
- **ChallengeCategory**: Top-level category (required)
  - `growth` - Scaling and expansion challenges
  - `transform` - Business model or operational transformation
  - `transition` - Ownership, exit, or major transitions
  - `compete` - Competitive positioning challenges
  - `funding` - Capital raising and financing
- **ChallengeType**: Specific challenge type (required)
  - Growth: `growth_organic`, `growth_geographic`, `growth_segment`, `growth_product`, `growth_channel`
  - Transform: `transform_digital`, `transform_model`, `transform_culture`, `transform_operational`
  - Transition: `transition_succession`, `transition_exit`, `transition_merger`, `transition_turnaround`
  - Compete: `compete_differentiate`, `compete_defend`, `compete_reposition`
  - Funding: `funding_raise`, `funding_debt`, `funding_ipo`
- **BusinessChallenge**: Free-text description of the challenge context (required)
- **Timestamps**: `created_at`, `updated_at`, `deleted_at`

## Domain Invariants

1. **Challenge requires company**: Every challenge MUST be linked to an existing company
2. **Category and type are required**: Both fields must be present and valid
3. **Type must match category**: e.g., `growth_organic` only valid for `growth` category
4. **Business challenge context is mandatory**: Free-text description cannot be empty
5. **Soft deletes**: Challenges are soft-deleted, not permanently removed
6. **No contact info**: Contact info belongs on Submission (the entry point), not Challenge

## Service Methods

| Method | Purpose |
|--------|---------|
| `NewService()` | Constructor |
| `Create()` | Create new challenge with validation |
| `GetByID()` | Retrieve by UUID |
| `ListByCompany()` | Get all challenges for a company |
| `Update()` | Update challenge fields |
| `Delete()` | Soft delete (sets `deleted_at`) |
| `ValidateCategory()` | Check if category string is valid |
| `ValidateType()` | Check if type string is valid for given category |

## Repository Methods

| Method | Purpose |
|--------|---------|
| `Create()` | Insert new challenge |
| `GetByID()` | Retrieve by UUID (excludes deleted) |
| `ListByCompanyID()` | Get all challenges for company (excludes deleted) |
| `Update()` | Update challenge fields |
| `Delete()` | Soft delete (sets `deleted_at`) |

All queries exclude soft-deleted records (`WHERE deleted_at IS NULL`).

## Relationship to Other Domains

### With Submission
- Submission creates the **first** challenge for a company automatically via saga pattern
- Contact info stays on Submission (the entry point) - Challenge only defines the problem
- After submission, Company + Challenge exist independently
- If challenge creation fails, saga rollback deletes the challenge

### With Company
- Challenge MUST reference an existing company (`company_id` required)
- Company can have 0-N challenges over time
- Same company can have multiple challenges (re-analyze scenarios)

### With Analysis
- Analysis links to Challenge (required) for context
- Challenge provides the "what" (problem), Analysis provides the "how" (frameworks)
- Multiple analyses can be run for the same challenge

## Usage Examples

### Creating a challenge from submission (via adapter)
```go
// In main.go adapter
challenge := challenge.NewChallenge(
    companyID,
    challenge.CategoryGrowth,
    challenge.TypeGrowthOrganic,
    "We need to scale from R$5M to R$50M ARR in 3 years",
)

created, err := challengeService.Create(ctx, challenge)
```

### Listing company challenges
```go
challenges, err := challengeService.ListByCompany(ctx, companyID)
// Returns challenges ordered by created_at DESC
```

### Re-analyze with new challenge (admin scenario)
```go
// Create new challenge for existing company
// No contact info needed - Challenge is just the problem definition
newChallenge := challenge.NewChallenge(
    existingCompanyID,
    challenge.CategoryTransform,
    challenge.TypeTransformDigital,
    "New challenge: digital transformation needed",
)
created, err := challengeService.Create(ctx, newChallenge)
// Then trigger analysis with this new challenge
```

## API Endpoints

Challenge domain has no direct API endpoints. Challenges are:
- Created via `/api/v1/submit` (submission flow)
- Created via `/api/v1/admin/companies/:id/re-analyze` (admin re-analyze)
- Queried via Analysis or Company endpoints

## Migration Notes

- Migration `v2_009_challenge_entity.sql` creates the challenges table
- Migration `v2_010_challenge_drop_contact.sql` removes contact columns (if previously added)
- `analyses.challenge_id` is required for new analyses
- Contact info stays on Submission only - Challenge defines the problem, not who requested it

## Exported Validation Helpers

The challenge domain exports validation helpers for use by other domains (e.g., submission):

| Function/Variable | Purpose |
|-------------------|---------|
| `ValidCategories` | Slice of all valid `ChallengeCategory` values |
| `ValidTypesByCategory` | Map from category to its valid types |
| `IsValidCategory(string)` | Check if category string is valid |
| `IsValidType(category, type)` | Check if type is valid for given category |
| `IsValidTypeAny(type)` | Check if type is valid regardless of category |

These are used by the submission domain to validate challenge fields before creating a challenge.

## AI Agent Warnings

### DO NOT
- Remove validation for category/type matching
- Allow challenges without a company reference
- Hard delete challenges (use soft delete)
- Store challenge fields on submission entity (they belong here)
- Remove or modify exported validation functions without updating submission domain

### SAFE TO MODIFY
- Add new challenge categories/types (update model.go constants AND ValidTypesByCategory map)
- Add new fields to Challenge entity
- Add new query methods to repository
- Extend validation rules
