package company

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

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

	// Enriched data v3 (additional fields from Perplexity + Claude)
	MainProducts         StringSlice `json:"main_products,omitempty" db:"main_products"`
	RecentNews           StringSlice `json:"recent_news,omitempty" db:"recent_news"`
	KeyExecutives        StringSlice `json:"key_executives,omitempty" db:"key_executives"`
	Opportunities        StringSlice `json:"opportunities,omitempty" db:"opportunities"`
	Threats              StringSlice `json:"threats,omitempty" db:"threats"`
	StrategicChallenges  StringSlice `json:"strategic_challenges,omitempty" db:"strategic_challenges"`
	CompetitorDetails    StringSlice `json:"competitor_details,omitempty" db:"competitor_details"`
	CompetitiveAdvantage *string     `json:"competitive_advantage,omitempty" db:"competitive_advantage"`
	MarketShare          *string     `json:"market_share,omitempty" db:"market_share"`
	TAMEstimate          *string     `json:"tam_estimate,omitempty" db:"tam_estimate"`
	SAMEstimate          *string     `json:"sam_estimate,omitempty" db:"sam_estimate"`
	SOMEstimate          *string     `json:"som_estimate,omitempty" db:"som_estimate"`
	CompanyHistory       *string     `json:"company_history,omitempty" db:"company_history"`
	CustomerSegments     StringSlice `json:"customer_segments,omitempty" db:"customer_segments"`
	PricingModel         *string     `json:"pricing_model,omitempty" db:"pricing_model"`
	UniqueSellingPoints  StringSlice `json:"unique_selling_points,omitempty" db:"unique_selling_points"`

	// Industry context (enrichment v2 - deeper market intelligence)
	IndustryGrowthRate  *string     `json:"industry_growth_rate,omitempty" db:"industry_growth_rate"`
	IndustryTrends      StringSlice `json:"industry_trends,omitempty" db:"industry_trends"`
	RegulatoryContext   *string     `json:"regulatory_context,omitempty" db:"regulatory_context"`
	MarketConcentration *string     `json:"market_concentration,omitempty" db:"market_concentration"`
	EnrichmentSources   StringSlice `json:"enrichment_sources,omitempty" db:"enrichment_sources"`

	// Social links
	LinkedInURL   *string `json:"linkedin_url,omitempty" db:"linkedin_url"`
	TwitterHandle *string `json:"twitter_handle,omitempty" db:"twitter_handle"`

	// Enrichment status (company-level tracking)
	EnrichmentStatus      string     `json:"enrichment_status" db:"enrichment_status"` // pending, processing, completed, failed
	EnrichmentCompletedAt *time.Time `json:"enrichment_completed_at,omitempty" db:"enrichment_completed_at"`
	EnrichmentError       *string    `json:"enrichment_error,omitempty" db:"enrichment_error"`

	// Access Control (no verification)
	AllowedUsers UUIDSlice  `json:"allowed_users" db:"allowed_users"` // Users who can access this company's data
	OwnerID      *uuid.UUID `json:"owner_id,omitempty" db:"owner_id"` // User who can manage allowed_users (only owner can add/remove)

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Validate validates required company fields
func (c *Company) Validate() error {
	if c.Name == "" {
		return ErrNameRequired
	}
	if len(c.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	// Field length validation
	if c.Website != nil && len(*c.Website) > MaxWebsiteLength {
		return ErrWebsiteTooLong
	}
	if c.Industry != nil && len(*c.Industry) > MaxIndustryLength {
		return ErrIndustryTooLong
	}
	if c.Location != nil && len(*c.Location) > MaxLocationLength {
		return ErrLocationTooLong
	}
	if c.LinkedInURL != nil && len(*c.LinkedInURL) > MaxURLLength {
		return ErrLinkedInURLTooLong
	}
	if c.TwitterHandle != nil && len(*c.TwitterHandle) > MaxNameLength {
		return ErrTwitterHandleTooLong
	}
	// Revenue range validation
	if c.AnnualRevenueMin != nil && c.AnnualRevenueMax != nil {
		if *c.AnnualRevenueMin > *c.AnnualRevenueMax {
			return ErrRevenueRangeInvalid
		}
	}
	if c.AnnualRevenueMin != nil && *c.AnnualRevenueMin < 0 {
		return ErrRevenueNegative
	}
	if c.AnnualRevenueMax != nil && *c.AnnualRevenueMax < 0 {
		return ErrRevenueNegative
	}
	return nil
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

// CompanySubmission represents the link between a company and its submissions
type CompanySubmission struct {
	CompanyID    uuid.UUID  `json:"company_id" db:"company_id"`
	SubmissionID uuid.UUID  `json:"submission_id" db:"submission_id"`
	IsPrimary    bool       `json:"is_primary" db:"is_primary"`
	LinkedAt     time.Time  `json:"linked_at" db:"linked_at"`
	LinkedBy     *uuid.UUID `json:"linked_by,omitempty" db:"linked_by"`
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
		EnrichmentStatus: EnrichmentPending, // Start with pending status
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// If owner is set, add them to allowed_users automatically
	if input.OwnerID != nil {
		company.AllowedUsers = UUIDSlice{*input.OwnerID}
	}

	return company
}

// AnalysisHistoryItem represents a single analysis with its associated challenge
// Used in admin dashboard to show all analyses for a company
type AnalysisHistoryItem struct {
	AnalysisID        uuid.UUID  `json:"analysis_id" db:"analysis_id"`
	SubmissionID      uuid.UUID  `json:"submission_id" db:"submission_id"`
	Status            string     `json:"status" db:"status"`                         // pending, processing, completed, failed
	BusinessChallenge string     `json:"business_challenge" db:"business_challenge"` // The challenge this analysis addresses
	IsBlurred         bool       `json:"is_blurred" db:"is_blurred"`                 // Whether report is blurred for non-admins
	IsVisibleToUser   bool       `json:"is_visible_to_user" db:"is_visible_to_user"` // Whether user can see this report
	IsPublic          bool       `json:"is_public" db:"is_public"`                   // Whether accessible without login
	AccessCode        *string    `json:"access_code,omitempty" db:"access_code"`     // Public access code (if generated)
	PdfUrl            *string    `json:"pdf_url,omitempty" db:"pdf_url"`             // PDF URL if generated
	CompletedAt       *time.Time `json:"completed_at,omitempty" db:"completed_at"`   // When analysis completed
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

// DataSource represents where company data originated
type DataSource string

const (
	SourceSubmission DataSource = "submission"
	SourceEnrichment DataSource = "enrichment"
	SourceManual     DataSource = "manual"
)

// Enrichment status constants
const (
	EnrichmentPending    = "pending"
	EnrichmentProcessing = "processing"
	EnrichmentCompleted  = "completed"
	EnrichmentFailed     = "failed"
)

// Validation constants
const (
	MaxNameLength     = 200
	MaxWebsiteLength  = 2048
	MaxIndustryLength = 200
	MaxLocationLength = 200
	MaxURLLength      = 2048
	MaxUserCompanies  = 100 // Default limit for GetUserCompanies query
)

// Validation errors
var (
	ErrNameRequired         = errors.New("company name is required")
	ErrNameTooLong          = errors.New("company name exceeds 200 characters")
	ErrWebsiteTooLong       = errors.New("website URL exceeds 2048 characters")
	ErrIndustryTooLong      = errors.New("industry exceeds 200 characters")
	ErrLocationTooLong      = errors.New("location exceeds 200 characters")
	ErrLinkedInURLTooLong   = errors.New("LinkedIn URL exceeds 2048 characters")
	ErrTwitterHandleTooLong = errors.New("Twitter handle exceeds 200 characters")
	ErrRevenueRangeInvalid  = errors.New("annual revenue min cannot be greater than max")
	ErrRevenueNegative      = errors.New("annual revenue cannot be negative")
)

// StringSlice handles JSON array columns in PostgreSQL
type StringSlice []string

// JSONMap handles JSONB columns in PostgreSQL
type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		*j = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal(b, j)
}

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
	if len(s) == 0 {
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

// joinStrings is a simple wrapper for strings.Join
// Kept for backward compatibility but uses the efficient standard library implementation
func joinStrings(strs []string, sep string) string {
	return strings.Join(strs, sep)
}
