package submission

import (
	"context"

	"github.com/google/uuid"
)

// =============================================================================
// REQUEST/INPUT TYPES
// =============================================================================

// SubmitRequest is the input for the public submission form.
// Simplified: company info + contact info only. Challenge created separately.
type SubmitRequest struct {
	// Company Information
	CompanyName    string  `json:"companyName"`
	CNPJ           *string `json:"cnpj,omitempty"`
	CompanyWebsite *string `json:"companyWebsite,omitempty"`

	// Contact Information (all required at API layer)
	ContactName  string `json:"contactName"`
	ContactEmail string `json:"contactEmail"`
	ContactPhone string `json:"contactPhone"`

	// Metadata (set by backend, not from JSON)
	UserID *uuid.UUID `json:"-"`
}

// ListOptions configures pagination and filtering for List queries.
type ListOptions struct {
	Limit   int
	Offset  int
	Email   *string
	UserID  *uuid.UUID
	OrderBy string
	Order   string
}

// CreateFromCompanyInput contains data needed to create a submission from existing company data.
// Used by admin re-enrich/re-analyze workflows.
type CreateFromCompanyInput struct {
	CompanyID        uuid.UUID
	CompanyName      string
	CNPJ             *string
	Website          *string
	Industry         *string
	CompanySize      *string
	Location         *string
	TargetMarket     *string
	FundingStage     *string
	AnnualRevenueMin *float64
	AnnualRevenueMax *float64
	LinkedInURL      *string
	TwitterHandle    *string
	ContactName      string
	ContactEmail     string
	UserID           *uuid.UUID
}

// =============================================================================
// RESPONSE/OUTPUT TYPES
// =============================================================================

// SubmitFormResponse contains the IDs of entities created by SubmitForm.
// Note: Challenge is no longer created automatically from submission.
type SubmitFormResponse struct {
	SubmissionID uuid.UUID `json:"submission_id"`
	CompanyID    uuid.UUID `json:"company_id"`
}

// =============================================================================
// SERVICE INTERFACE TYPES (for dependency injection)
// =============================================================================

// CompanyServiceInterface defines the contract for company service.
// Using an interface avoids import cycles.
type CompanyServiceInterface interface {
	CreateFromSubmission(ctx context.Context, input CompanyCreateInput) (CompanyResult, error)
	SetOwnerFromSubmission(ctx context.Context, submissionID, userID uuid.UUID) error
	DeleteCompany(ctx context.Context, id uuid.UUID) error // For saga rollback
}

// CompanyResult is a minimal struct to receive company creation result.
type CompanyResult struct {
	ID   uuid.UUID
	Name string
}

// CompanyCreateInput contains the data needed to create a company from a submission.
// Simplified to match the new submission form fields.
type CompanyCreateInput struct {
	SubmissionID uuid.UUID
	CompanyName  string
	CNPJ         *string
	Website      *string
	OwnerID      *uuid.UUID
	ContactEmail string // Submitter's email for CNPJ duplicate detection
}

// =============================================================================
// VALIDATION CONSTANTS
// =============================================================================

// String length limits for validation.
const (
	MaxCompanyNameLength = 200
	MaxContactNameLength = 100
	MaxEmailLength       = 254
	MaxPhoneLength       = 50
	MaxURLLength         = 2048
)
