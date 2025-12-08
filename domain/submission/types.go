package submission

import (
	"context"

	"github.com/google/uuid"
)

// =============================================================================
// REQUEST/INPUT TYPES
// =============================================================================

// SubmitRequest is the input for the public submission form.
// All challenge fields are REQUIRED - they drive the analysis.
type SubmitRequest struct {
	// Company Information (required: CompanyName)
	CompanyName     string  `json:"companyName"`
	CNPJ            *string `json:"cnpj,omitempty"`
	CompanyWebsite  *string `json:"companyWebsite,omitempty"`
	CompanyIndustry *string `json:"companyIndustry,omitempty"`
	CompanySize     *string `json:"companySize,omitempty"`
	CompanyLocation *string `json:"companyLocation,omitempty"`

	// Contact Information (all required)
	ContactName     string  `json:"contactName"`
	ContactEmail    string  `json:"contactEmail"`
	ContactPhone    *string `json:"contactPhone,omitempty"`
	ContactPosition *string `json:"contactPosition,omitempty"`

	// Business Context
	TargetMarket     *string  `json:"targetMarket,omitempty"`
	AnnualRevenueMin *float64 `json:"annualRevenueMin,omitempty"`
	AnnualRevenueMax *float64 `json:"annualRevenueMax,omitempty"`
	FundingStage     *string  `json:"fundingStage,omitempty"`

	// Challenge Definition (ALL REQUIRED - drives the analysis)
	// Valid categories: growth, transform, transition, compete, funding
	ChallengeCategory string `json:"challengeCategory"`
	// Valid types depend on category (e.g., growth_organic, transform_digital)
	ChallengeType string `json:"challengeType"`
	// Free-text description of the business challenge
	BusinessChallenge string `json:"businessChallenge"`

	// Additional Information
	AdditionalNotes *string `json:"additionalNotes,omitempty"`
	LinkedInURL     *string `json:"linkedinUrl,omitempty"`
	TwitterHandle   *string `json:"twitterHandle,omitempty"`

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

// SubmitFormResponse contains the IDs of all entities created by SubmitForm.
type SubmitFormResponse struct {
	SubmissionID uuid.UUID `json:"submission_id"`
	CompanyID    uuid.UUID `json:"company_id"`
	ChallengeID  uuid.UUID `json:"challenge_id"`
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
type CompanyCreateInput struct {
	SubmissionID     uuid.UUID
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
	OwnerID          *uuid.UUID
}

// ChallengeServiceInterface defines the contract for challenge service.
type ChallengeServiceInterface interface {
	CreateFromInput(ctx context.Context, input ChallengeCreateInput) (uuid.UUID, error)
	DeleteChallenge(ctx context.Context, id uuid.UUID) error // For saga rollback
	// Validation helpers (delegates to challenge domain)
	ValidateCategory(category string) bool
	ValidateType(category, challengeType string) bool
}

// ChallengeCreateInput contains the data needed to create a challenge.
// NOTE: Contact info is NOT included - it stays on the Submission entity.
type ChallengeCreateInput struct {
	CompanyID         uuid.UUID
	ChallengeCategory string
	ChallengeType     string
	BusinessChallenge string
}

// =============================================================================
// VALIDATION CONSTANTS
// =============================================================================

// NOTE: Challenge category/type validation has been moved to the challenge domain.
// Use challenge.ValidateCategory() and challenge.ValidateType() for validation.

// String length limits for validation.
const (
	MaxCompanyNameLength       = 200
	MaxContactNameLength       = 100
	MaxEmailLength             = 254
	MaxPhoneLength             = 50
	MaxBusinessChallengeLength = 5000
	MaxAdditionalNotesLength   = 10000
	MaxURLLength               = 2048
)
