package submission

import (
	"net/mail"
	"time"

	"github.com/google/uuid"
)

// Submission represents a company submission for business intelligence analysis
// This is the entry point for the entire workflow
type Submission struct {
	// Primary key
	ID uuid.UUID `json:"id" db:"id"`

	// Company Information (6 fields)
	CompanyName     string  `json:"companyName" db:"company_name"`
	CNPJ            *string `json:"cnpj,omitempty" db:"cnpj"`
	CompanyWebsite  *string `json:"companyWebsite,omitempty" db:"company_website"`
	CompanyIndustry *string `json:"companyIndustry,omitempty" db:"company_industry"`
	CompanySize     *string `json:"companySize,omitempty" db:"company_size"`
	CompanyLocation *string `json:"companyLocation,omitempty" db:"company_location"`

	// Contact Information (4 fields)
	ContactName     string  `json:"contactName" db:"contact_name"`
	ContactEmail    string  `json:"contactEmail" db:"contact_email"`
	ContactPhone    *string `json:"contactPhone,omitempty" db:"contact_phone"`
	ContactPosition *string `json:"contactPosition,omitempty" db:"contact_position"`

	// Business Context (4 fields)
	TargetMarket     *string  `json:"targetMarket,omitempty" db:"target_market"`
	AnnualRevenueMin *float64 `json:"annualRevenueMin,omitempty" db:"annual_revenue_min"`
	AnnualRevenueMax *float64 `json:"annualRevenueMax,omitempty" db:"annual_revenue_max"`
	FundingStage     *string  `json:"fundingStage,omitempty" db:"funding_stage"`

	// Additional Details
	AdditionalNotes *string `json:"additionalNotes,omitempty" db:"additional_notes"`
	LinkedInURL     *string `json:"linkedinUrl,omitempty" db:"linkedin_url"`
	TwitterHandle   *string `json:"twitterHandle,omitempty" db:"twitter_handle"`

	// Metadata
	UserID *uuid.UUID `json:"userId,omitempty" db:"user_id"` // Nullable for public submissions

	// Timestamps
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`
}

// NewSubmission creates a new submission with default values
func NewSubmission(
	companyName string,
	contactName string,
	contactEmail string,
	userID *uuid.UUID,
) *Submission {
	now := time.Now()
	return &Submission{
		ID:           uuid.New(),
		CompanyName:  companyName,
		ContactName:  contactName,
		ContactEmail: contactEmail,
		UserID:       userID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// Validate validates the submission fields
func (s *Submission) Validate() error {
	if s.CompanyName == "" {
		return NewValidationError("company_name", "is required")
	}
	if s.ContactName == "" {
		return NewValidationError("contact_name", "is required")
	}
	if s.ContactEmail == "" {
		return NewValidationError("contact_email", "is required")
	}
	if _, err := mail.ParseAddress(s.ContactEmail); err != nil {
		return NewValidationError("contact_email", "is invalid")
	}
	// Note: ContactPhone validation is done at API layer for new submissions
	// Existing records may have NULL phone values
	return nil
}

// IsDeleted returns true if the submission has been soft deleted
func (s *Submission) IsDeleted() bool {
	return s.DeletedAt != nil
}
