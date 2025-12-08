# Company Domain

## Purpose

Manages company records with ownership, one-time enrichment at creation, and editable fields.

## File Structure

```
company/
├── model.go       # Entity definition + helper types (StringSlice, UUIDSlice, JSONMap)
├── repository.go  # Database access (PostgreSQL) with transaction support
└── service.go     # Business logic including async enrichment
```

## Data Flow

```
Submission Created                   Company Domain                  Enrichment
      │                                    │                            │
      ├── CreateFromSubmission() ─────────►│                            │
      │                                    ├── Create company record    │
      │                                    │   (owner_id = user_id)     │
      │                                    │   (enrichment_status=pending)
      │                                    │                            │
      │                                    ├── Link to submission ──────┤
      │                                    │   (company_submissions)    │
      │                                    │                            │
      │                                    ├── runEnrichment() async ──►│
      │                                    │   (fire-and-forget)        │
      │                                    │                            │
      │◄── Company created ────────────────┤                            │
      │                                    │                            │
      │                                    │◄── Enriched data ─────────┤
      │                                    │   (fills NULL fields)      │
      │                                    │   (enrichment_status=completed)
```

## Business Rules

### Core Principles
- **One-time enrichment**: Enrichment runs automatically at company creation, never again
- **Enrichment fills gaps**: Only populates NULL fields, never overwrites user input
- **All fields editable**: Owner/admin can edit any field in Supabase
- **Multiple submissions per company**: Company can be linked to multiple submissions (re-analyze)
- **Async enrichment with timeout**: Enrichment runs in background goroutine with 5-minute timeout

### Ownership & Access
- `owner_id`: User who submitted (has edit access)
- `allowed_users`: Additional users with view access
- Owner is automatically added to `allowed_users` on creation
- Access managed in Supabase directly (no PUT endpoints)

### Saga Rollback
- `Delete()` method exists for saga rollback only
- Hard deletes company and unlinks from `company_submissions`
- Not exposed via API - only used internally

## Key Types

| Type | Purpose |
|------|---------|
| `Company` | Main entity with all company data |
| `CompanySubmission` | Links company to submissions (many-to-many) |
| `AnalysisHistoryItem` | Analysis records for admin view |
| `CreateFromSubmissionInput` | Input for company creation |
| `ErrNameRequired`, `ErrNameTooLong`, etc. | Validation errors |
| `StringSlice` | JSON array type for PostgreSQL |
| `UUIDSlice` | UUID array type for PostgreSQL |
| `JSONMap` | JSONB type for PostgreSQL |

## Model Fields

```go
// Core identifiers
ID               UUID     // Primary key
Name             string   // Company name (required)
CNPJ             *string  // Brazilian tax ID
Website          *string  // Official website

// Business context (from submission)
Industry         *string  // Sector/industry
CompanySize      *string  // Employee count range
Location         *string  // City/Country
TargetMarket     *string  // B2B/B2C, segments
FundingStage     *string  // Seed, Series A, etc.
AnnualRevenueMin *float64 // Revenue range min
AnnualRevenueMax *float64 // Revenue range max

// Enriched data (from AI - fills NULL fields only)
FoundationYear   *string  // Year founded
LegalName        *string  // Legal/registered name
Headquarters     *string  // HQ location
Sector           *string  // More specific sector
TargetAudience   *string  // Target customer profile
ValueProposition *string  // Unique value prop
EmployeesRange   *string  // Employee count estimate
RevenueEstimate  *string  // Revenue estimate
BusinessModel    *string  // B2B, B2C, marketplace, etc.
Competitors      []string // Known competitors
MarketShareStatus *string // Market position
DigitalMaturity  *int     // 1-10 scale
Strengths        []string // Company strengths
Weaknesses       []string // Company weaknesses

// Social links
LinkedInURL      *string  // LinkedIn profile
TwitterHandle    *string  // Twitter/X handle

// Enrichment tracking
EnrichmentStatus      string     // pending, processing, completed, failed
EnrichmentCompletedAt *time.Time // When enrichment finished
EnrichmentError       *string    // Error message if failed

// Access control
OwnerID          *UUID     // Primary owner
AllowedUsers     []UUID    // Users with access

// Timestamps
CreatedAt        time.Time
UpdatedAt        time.Time
```

## Model Methods

| Method | Purpose |
|--------|---------|
| `Validate()` | Validates required fields (Name, revenue range) |
| `IsOwner(userID)` | Check if user is owner |
| `CanManageUsers(userID)` | Check if user can manage allowed_users |
| `IsUserAllowed(userID)` | Check if user has access (owner OR allowed) |

## Service Methods (13 total)

| Method | Purpose |
|--------|---------|
| `NewService()` | Constructor |
| `SetEnrichmentService()` | Inject enrichment dependency |
| `CreateDirect()` | Create company directly (no submission link) |
| `CreateFromSubmission()` | Create company from submission, link, trigger enrichment |
| `GetByID()` | Get company by ID |
| `Delete()` | Hard delete for saga rollback (internal use only) |
| `GetBySubmissionID()` | Get company linked to a submission |
| `LinkSubmission()` | Link additional submission (for re-analyze) |
| `GetUserCompanies()` | Get all companies for a user (owner or allowed) |
| `ListAll()` | Admin: list all companies with pagination |
| `GetAnalysesHistory()` | Get all analyses for a company |
| `SetOwnerFromSubmission()` | Link anonymous submission to new user |
| `runEnrichment()` | Private: async enrichment execution |

## Repository Methods

| Method | Purpose |
|--------|---------|
| `Create()` | Insert new company |
| `GetByID()` | Get by UUID |
| `GetBySubmissionID()` | Get company linked to submission |
| `Update()` | Update all fields |
| `Delete()` | Hard delete (for saga rollback) |
| `GetUserCompanies()` | Get companies by owner or allowed_users |
| `ListAll()` | List with pagination |
| `LinkSubmission()` | Create company_submissions record |
| `UnlinkSubmissions()` | Delete company_submissions records |
| `GetAnalysesHistory()` | Get analyses with challenge info |
| `SetEnrichmentProcessing()` | Update status to processing |
| `SetEnrichmentCompleted()` | Update status + fill enriched fields |
| `SetEnrichmentFailed()` | Update status + error message |
| `WithTx()` | Execute function in transaction |

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/companies` | GET | List user's companies |
| `/api/v1/companies` | POST | Create company directly (authenticated) |
| `/api/v1/companies/:id` | GET | Get company details |
| `/api/v1/admin/companies` | GET | List all companies |
| `/api/v1/admin/companies/:id` | GET | Get company + analyses history |
| `/api/v1/admin/companies/:id/re-analyze` | POST | Trigger new analysis |

**Note:** PUT endpoints removed - edit company data directly in Supabase.

## Related Domains

- **Submission**: Creates company record via saga pattern
- **Enrichment**: Stateless service called once at company creation
- **Challenge**: Business challenge linked to company via `company_id`
- **Analysis**: Framework outputs linked via `company_id`

## What Was Removed (Simplified)

| Removed | Reason |
|---------|--------|
| Company verification (`is_verified`) | Not needed |
| Re-enrichment | Enrichment is one-time only |
| Field-level history tracking | Not needed |
| Multi-user management endpoints | Manage `allowed_users` in Supabase |
| Transfer ownership endpoint | Not needed |
| Sector intelligence fields | Never used |
| Data quality tracking | Never used |
| Company update endpoints (PUT) | Edit in Supabase directly |
| `company_data_history` table | Code removed, table dropped in v2_011 |

## AI Agent Warnings

### DO NOT
- Add re-enrichment capability (enrichment is one-time)
- Add field-level history tracking (removed for simplicity)
- Add verification system back (removed)
- Create separate enrichments table (enrichment is inline on company)
- Expose Delete() via API (saga rollback only)
- Make enrichment synchronous (must be async)

### SAFE TO MODIFY
- Add new company fields (update model + repository queries)
- Extend enrichment to populate new fields
- Add new query methods for specific use cases
- Modify access control logic
