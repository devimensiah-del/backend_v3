package submission

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Submission represents a company submission for business intelligence analysis
// This is the entry point for the entire workflow
type Submission struct {
	// Primary key
	ID uuid.UUID `json:"id" db:"id"`

	// Company Information (5 fields)
	CompanyName     string  `json:"companyName" db:"company_name"`
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

	// Submission Details (4 fields)
	BusinessChallenge string  `json:"businessChallenge" db:"business_challenge"`
	AdditionalNotes   *string `json:"additionalNotes,omitempty" db:"additional_notes"`
	LinkedInURL       *string `json:"linkedinUrl,omitempty" db:"linkedin_url"`
	TwitterHandle     *string `json:"twitterHandle,omitempty" db:"twitter_handle"`

	// Metadata
	Status Status     `json:"status" db:"status"`
	UserID *uuid.UUID `json:"userId,omitempty" db:"user_id"` // Nullable for public submissions

	// Timestamps
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`
}

// Status represents the submission workflow status
type Status string

const (
	StatusPending          Status = "pending"
	StatusEnriching        Status = "enriching"
	StatusEnriched         Status = "enriched"
	StatusAnalyzing        Status = "analyzing"
	StatusAnalyzed         Status = "analyzed"
	StatusGeneratingReport Status = "generating_report"
	StatusCompleted        Status = "completed"

	// Failure States (Must match DB constraints)
	StatusEnrichmentFailed Status = "enrichment_failed"
	StatusAnalysisFailed   Status = "analysis_failed"
	StatusReportFailed     Status = "report_failed"
	StatusFailed           Status = "failed"
)

// NewSubmission creates a new submission with default values
func NewSubmission(
	companyName string,
	contactName string,
	contactEmail string,
	businessChallenge string,
	userID *uuid.UUID,
) *Submission {
	now := time.Now()
	return &Submission{
		ID:                uuid.New(),
		CompanyName:       companyName,
		ContactName:       contactName,
		ContactEmail:      contactEmail,
		BusinessChallenge: businessChallenge,
		UserID:            userID,
		Status:            StatusPending,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// Validate validates the submission fields
func (s *Submission) Validate() error {
	if s.CompanyName == "" {
		return errors.New("company name is required")
	}
	if s.ContactName == "" {
		return errors.New("contact name is required")
	}
	if s.ContactEmail == "" {
		return errors.New("contact email is required")
	}
	if s.BusinessChallenge == "" {
		return errors.New("business challenge is required")
	}
	return nil
}

// IsDeleted returns true if the submission has been soft deleted
func (s *Submission) IsDeleted() bool {
	return s.DeletedAt != nil
}

// CanEnrich returns true if the submission can be enriched
func (s *Submission) CanEnrich() bool {
	return s.Status == StatusPending || s.Status == StatusFailed
}

// CanAnalyze returns true if the submission can be analyzed
func (s *Submission) CanAnalyze() bool {
	return s.Status == StatusEnriched
}

// SetStatus updates the status and updated_at timestamp
func (s *Submission) SetStatus(status Status) {
	s.Status = status
	s.UpdatedAt = time.Now()
}
