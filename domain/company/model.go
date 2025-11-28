package company

import (
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DataSource represents where company data originated
type DataSource string

const (
	SourceSubmission DataSource = "submission"
	SourceEnrichment DataSource = "enrichment"
	SourceManual     DataSource = "manual"
)

// StringSlice handles JSON array columns in PostgreSQL
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(s)
}

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		*s = []string{}
		return nil
	}
	return json.Unmarshal(b, s)
}

// UUIDSlice handles UUID array columns in PostgreSQL
type UUIDSlice []uuid.UUID

func (s UUIDSlice) Value() (driver.Value, error) {
	if s == nil || len(s) == 0 {
		return "{}", nil
	}
	// Format as PostgreSQL array literal: {uuid1,uuid2,...}
	strs := make([]string, len(s))
	for i, u := range s {
		strs[i] = u.String()
	}
	return "{" + joinStrings(strs, ",") + "}", nil
}

func (s *UUIDSlice) Scan(value interface{}) error {
	if value == nil {
		*s = []uuid.UUID{}
		return nil
	}

	// PostgreSQL returns UUID[] as string like "{uuid1,uuid2}"
	var str string
	switch v := value.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		*s = []uuid.UUID{}
		return nil
	}

	// Parse PostgreSQL array format
	str = strings.Trim(str, "{}")
	if str == "" {
		*s = []uuid.UUID{}
		return nil
	}

	parts := strings.Split(str, ",")
	result := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		u, err := uuid.Parse(p)
		if err != nil {
			continue // Skip invalid UUIDs
		}
		result = append(result, u)
	}
	*s = result
	return nil
}

// Contains checks if the slice contains a specific UUID
func (s UUIDSlice) Contains(id uuid.UUID) bool {
	for _, u := range s {
		if u == id {
			return true
		}
	}
	return false
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}

// Company represents a company entity with all tracked data
// Data comes from submissions, enrichment, or manual edits
type Company struct {
	ID uuid.UUID `json:"id" db:"id"`

	// Core identifiers (from submission)
	Name    string  `json:"name" db:"name"`
	CNPJ    *string `json:"cnpj,omitempty" db:"cnpj"`
	Website *string `json:"website,omitempty" db:"website"`

	// Business context (from submission)
	Industry         *string  `json:"industry,omitempty" db:"industry"`
	CompanySize      *string  `json:"company_size,omitempty" db:"company_size"`
	Location         *string  `json:"location,omitempty" db:"location"`
	TargetMarket     *string  `json:"target_market,omitempty" db:"target_market"`
	FundingStage     *string  `json:"funding_stage,omitempty" db:"funding_stage"`
	AnnualRevenueMin *float64 `json:"annual_revenue_min,omitempty" db:"annual_revenue_min"`
	AnnualRevenueMax *float64 `json:"annual_revenue_max,omitempty" db:"annual_revenue_max"`

	// Enriched data (from AI enrichment)
	FoundationYear    *string     `json:"foundation_year,omitempty" db:"foundation_year"`
	LegalName         *string     `json:"legal_name,omitempty" db:"legal_name"`
	Headquarters      *string     `json:"headquarters,omitempty" db:"headquarters"`
	Sector            *string     `json:"sector,omitempty" db:"sector"`
	TargetAudience    *string     `json:"target_audience,omitempty" db:"target_audience"`
	ValueProposition  *string     `json:"value_proposition,omitempty" db:"value_proposition"`
	EmployeesRange    *string     `json:"employees_range,omitempty" db:"employees_range"`
	RevenueEstimate   *string     `json:"revenue_estimate,omitempty" db:"revenue_estimate"`
	BusinessModel     *string     `json:"business_model,omitempty" db:"business_model"`
	Competitors       StringSlice `json:"competitors,omitempty" db:"competitors"`
	MarketShareStatus *string     `json:"market_share_status,omitempty" db:"market_share_status"`
	DigitalMaturity   *int        `json:"digital_maturity,omitempty" db:"digital_maturity"`
	Strengths         StringSlice `json:"strengths,omitempty" db:"strengths"`
	Weaknesses        StringSlice `json:"weaknesses,omitempty" db:"weaknesses"`

	// Social links
	LinkedInURL   *string `json:"linkedin_url,omitempty" db:"linkedin_url"`
	TwitterHandle *string `json:"twitter_handle,omitempty" db:"twitter_handle"`

	// Verification & Access Control
	IsVerified   bool       `json:"is_verified" db:"is_verified"`     // Admin-verified, CNPJ becomes unique when true
	AllowedUsers UUIDSlice  `json:"allowed_users" db:"allowed_users"` // Users who can access this company's data
	OwnerID      *uuid.UUID `json:"owner_id,omitempty" db:"owner_id"` // User who can manage allowed_users (only owner can add/remove)

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// IsOwner checks if the given user is the owner of this company
func (c *Company) IsOwner(userID uuid.UUID) bool {
	return c.OwnerID != nil && *c.OwnerID == userID
}

// CanManageUsers checks if the given user can manage allowed_users (owner only, or admin via separate check)
func (c *Company) CanManageUsers(userID uuid.UUID) bool {
	return c.IsOwner(userID)
}

// IsUserAllowed checks if a user has access to this company (owner OR in allowed_users)
func (c *Company) IsUserAllowed(userID uuid.UUID) bool {
	return c.IsOwner(userID) || c.AllowedUsers.Contains(userID)
}

// HasLockedCNPJ returns true if the company has a verified CNPJ (unique constraint applies)
func (c *Company) HasLockedCNPJ() bool {
	return c.IsVerified && c.CNPJ != nil && *c.CNPJ != ""
}

// DataHistoryEntry represents a single change to company data
type DataHistoryEntry struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	CompanyID uuid.UUID  `json:"company_id" db:"company_id"`
	FieldName string     `json:"field_name" db:"field_name"`
	OldValue  *string    `json:"old_value,omitempty" db:"old_value"`
	NewValue  *string    `json:"new_value,omitempty" db:"new_value"`
	Source    DataSource `json:"source" db:"source"`
	SourceID  *uuid.UUID `json:"source_id,omitempty" db:"source_id"`
	ChangedBy *uuid.UUID `json:"changed_by,omitempty" db:"changed_by"`
	ChangedAt time.Time  `json:"changed_at" db:"changed_at"`
}

// CompanySubmission represents the link between a company and its submissions
type CompanySubmission struct {
	CompanyID    uuid.UUID  `json:"company_id" db:"company_id"`
	SubmissionID uuid.UUID  `json:"submission_id" db:"submission_id"`
	IsPrimary    bool       `json:"is_primary" db:"is_primary"`
	LinkedAt     time.Time  `json:"linked_at" db:"linked_at"`
	LinkedBy     *uuid.UUID `json:"linked_by,omitempty" db:"linked_by"`
}

// CompanyWithHistory includes the company and its data history
type CompanyWithHistory struct {
	Company *Company            `json:"company"`
	History []*DataHistoryEntry `json:"history"`
}

// CompanyWithSubmissions includes the company and linked submissions
type CompanyWithSubmissions struct {
	Company     *Company             `json:"company"`
	Submissions []*CompanySubmission `json:"submissions"`
}

// CreateFromSubmissionInput contains the data needed to create a company from a submission
type CreateFromSubmissionInput struct {
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
	OwnerID          *uuid.UUID // User who submitted, becomes owner
}

// UpdateFromEnrichmentInput contains the enriched data to update a company
type UpdateFromEnrichmentInput struct {
	EnrichmentID      uuid.UUID
	FoundationYear    *string
	LegalName         *string
	Headquarters      *string
	Sector            *string
	TargetAudience    *string
	ValueProposition  *string
	EmployeesRange    *string
	RevenueEstimate   *string
	BusinessModel     *string
	Competitors       []string
	MarketShareStatus *string
	DigitalMaturity   *int
	Strengths         []string
	Weaknesses        []string
	// Also discovered basic data
	CNPJ          *string
	Website       *string
	LinkedInURL   *string
	TwitterHandle *string
}

// ManualUpdateInput contains fields that can be manually updated by admin
type ManualUpdateInput struct {
	UserID           uuid.UUID
	Name             *string
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
}

// NewCompany creates a new company from submission data
func NewCompany(input CreateFromSubmissionInput) *Company {
	now := time.Now()
	company := &Company{
		ID:               uuid.New(),
		Name:             input.CompanyName,
		CNPJ:             input.CNPJ,
		Website:          input.Website,
		Industry:         input.Industry,
		CompanySize:      input.CompanySize,
		Location:         input.Location,
		TargetMarket:     input.TargetMarket,
		FundingStage:     input.FundingStage,
		AnnualRevenueMin: input.AnnualRevenueMin,
		AnnualRevenueMax: input.AnnualRevenueMax,
		LinkedInURL:      input.LinkedInURL,
		TwitterHandle:    input.TwitterHandle,
		OwnerID:          input.OwnerID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// If owner is set, add them to allowed_users automatically
	if input.OwnerID != nil {
		company.AllowedUsers = UUIDSlice{*input.OwnerID}
	}

	return company
}

// LastSubmissionInfo contains data from the most recent submission for re-enrich/re-analyze
type LastSubmissionInfo struct {
	SubmissionID      uuid.UUID `db:"submission_id"`
	BusinessChallenge string    `db:"business_challenge"`
	LinkedAt          time.Time `db:"linked_at"`
}

// LastEnrichmentInfo contains data from the most recent completed enrichment
// Note: completed_at comes from COALESCE(completed_at, updated_at) in query
type LastEnrichmentInfo struct {
	EnrichmentID uuid.UUID `db:"enrichment_id"`
	SubmissionID uuid.UUID `db:"submission_id"`
	CompletedAt  time.Time `db:"completed_at"` // When enrichment was completed
}

// EnrichmentStatus contains current enrichment status for admin dashboard
// Shows the latest enrichment regardless of status
type EnrichmentStatus struct {
	EnrichmentID uuid.UUID  `json:"enrichment_id" db:"enrichment_id"`
	SubmissionID uuid.UUID  `json:"submission_id" db:"submission_id"`
	Status       string     `json:"status" db:"status"`                      // pending, completed, failed
	Progress     int        `json:"progress" db:"progress"`                  // 0-100
	CurrentStep  string     `json:"current_step" db:"current_step"`          // Current processing step
	StartedAt    *time.Time `json:"started_at,omitempty" db:"started_at"`    // When processing started
	CompletedAt  *time.Time `json:"completed_at,omitempty" db:"completed_at"` // When completed (if done)
	ErrorMessage string     `json:"error_message,omitempty" db:"error_message"` // Error if failed
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// FieldVerification tracks which company fields are verified and protected from re-enrichment
type FieldVerification struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	CompanyID  uuid.UUID  `json:"company_id" db:"company_id"`
	FieldName  string     `json:"field_name" db:"field_name"`
	VerifiedAt time.Time  `json:"verified_at" db:"verified_at"`
	VerifiedBy *uuid.UUID `json:"verified_by,omitempty" db:"verified_by"`
}

// AnalysisHistoryItem represents a single analysis with its associated challenge
// Used in admin dashboard to show all analyses for a company
type AnalysisHistoryItem struct {
	AnalysisID        uuid.UUID  `json:"analysis_id" db:"analysis_id"`
	SubmissionID      uuid.UUID  `json:"submission_id" db:"submission_id"`
	Status            string     `json:"status" db:"status"`                        // pending, processing, completed, failed
	BusinessChallenge string     `json:"business_challenge" db:"business_challenge"` // The challenge this analysis addresses
	IsBlurred         bool       `json:"is_blurred" db:"is_blurred"`                 // Whether report is blurred for non-admins
	IsVisibleToUser   bool       `json:"is_visible_to_user" db:"is_visible_to_user"` // Whether user can see this report
	AccessCode        *string    `json:"access_code,omitempty" db:"access_code"`     // Public access code (if generated)
	PdfUrl            *string    `json:"pdf_url,omitempty" db:"pdf_url"`             // PDF URL if generated
	CompletedAt       *time.Time `json:"completed_at,omitempty" db:"completed_at"`   // When analysis completed
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

// CompanyWithVerifications includes the company and its field verification status
type CompanyWithVerifications struct {
	Company       *Company            `json:"company"`
	Verifications []*FieldVerification `json:"verifications"`
}

// VerifiableFields lists all company fields that can be individually verified
var VerifiableFields = []string{
	"name", "cnpj", "website",
	"industry", "company_size", "location", "target_market", "funding_stage",
	"annual_revenue_min", "annual_revenue_max",
	"foundation_year", "legal_name", "headquarters",
	"sector", "target_audience", "value_proposition",
	"employees_range", "revenue_estimate", "business_model",
	"competitors", "market_share_status", "digital_maturity",
	"strengths", "weaknesses",
	"linkedin_url", "twitter_handle",
}

// IsFieldVerified checks if a specific field is in the verified list
func IsFieldVerified(fieldName string, verifications []*FieldVerification) bool {
	for _, v := range verifications {
		if v.FieldName == fieldName {
			return true
		}
	}
	return false
}

// GetVerifiedFieldSet returns a map for O(1) lookup of verified fields
func GetVerifiedFieldSet(verifications []*FieldVerification) map[string]bool {
	set := make(map[string]bool, len(verifications))
	for _, v := range verifications {
		set[v.FieldName] = true
	}
	return set
}
