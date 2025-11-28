package enrichment

import (
	"context"
	"time"
)

// MacroSnapshot represents latest economic indicators from DB (dynamic)
// This is the interface type used by enrichment service
// Contains ALL available indicators - LLM decides which are relevant
type MacroSnapshot struct {
	Indicators map[string]*MacroIndicator `json:"indicators"`
	AsOf       time.Time                   `json:"as_of"`
}

// MacroIndicator represents a single indicator with full metadata
// Includes name/category so LLM can understand the indicator
type MacroIndicator struct {
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	Category        string    `json:"category"`
	Value           float64   `json:"value"`
	Unit            string    `json:"unit"`
	EffectiveDate   time.Time `json:"effective_date"`
	ReferencePeriod *string   `json:"reference_period,omitempty"`
	SourceCode      string    `json:"source_code"`
	FetchedAt       time.Time `json:"fetched_at"`
}

// HasData returns true if at least one indicator has data
func (s *MacroSnapshot) HasData() bool {
	return len(s.Indicators) > 0
}

// Get returns a specific indicator by code (for backwards compatibility)
func (s *MacroSnapshot) Get(code string) *MacroIndicator {
	if s.Indicators == nil {
		return nil
	}
	return s.Indicators[code]
}

// MacroServiceInterface defines the contract for macroeconomics service
// This interface is implemented by macroeconomics.Service
// Using an interface avoids import cycles
type MacroServiceInterface interface {
	// GetLatestSnapshot retrieves the most recent values for ALL active indicators
	// Returns nil error if DB is empty (graceful degradation)
	GetLatestSnapshot(ctx context.Context) (*MacroSnapshot, error)
}

// =================================================================
// COMPANY SERVICE INTERFACE
// =================================================================

// CompanyUpdateInput contains enriched data to update a company
type CompanyUpdateInput struct {
	EnrichmentID      string
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
	// Discovered basic data
	CNPJ          *string
	Website       *string
	LinkedInURL   *string
	TwitterHandle *string
}

// CompanyData represents the essential company fields needed for enrichment
// This is a simplified view to avoid importing the company package
type CompanyData struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	CNPJ             *string `json:"cnpj,omitempty"`
	Website          *string `json:"website,omitempty"`
	Industry         *string `json:"industry,omitempty"`
	CompanySize      *string `json:"company_size,omitempty"`
	Location         *string `json:"location,omitempty"`
	TargetMarket     *string `json:"target_market,omitempty"`
	FundingStage     *string `json:"funding_stage,omitempty"`
	AnnualRevenueMin *float64 `json:"annual_revenue_min,omitempty"`
	AnnualRevenueMax *float64 `json:"annual_revenue_max,omitempty"`
	FoundationYear   *string `json:"foundation_year,omitempty"`
	LegalName        *string `json:"legal_name,omitempty"`
	Headquarters     *string `json:"headquarters,omitempty"`
	Sector           *string `json:"sector,omitempty"`
	LinkedInURL      *string `json:"linkedin_url,omitempty"`
	TwitterHandle    *string `json:"twitter_handle,omitempty"`
}

// CompanyServiceInterface defines the contract for company service
// This interface is implemented by company.Service
// Using an interface avoids import cycles
type CompanyServiceInterface interface {
	// UpdateFromEnrichment updates company with enriched data (COALESCE behavior - fills empty fields only)
	// DEPRECATED: Use UpdateFromEnrichmentSmartMerge instead for field-level verification support
	UpdateFromEnrichment(ctx context.Context, submissionID string, input CompanyUpdateInput) error

	// UpdateFromEnrichmentSmartMerge updates company with enriched data using smart merge
	// Only updates fields that are NOT verified (verified fields are protected from re-enrichment)
	UpdateFromEnrichmentSmartMerge(ctx context.Context, submissionID string, input CompanyUpdateInput) error

	// GetCompanyIDBySubmissionID retrieves the company ID linked to a submission
	// Returns nil if no company is linked
	GetCompanyIDBySubmissionID(ctx context.Context, submissionID string) (*string, error)

	// GetCompanyDataByID retrieves company data by ID for enrichment purposes
	// Returns simplified CompanyData to avoid import cycles
	GetCompanyDataByID(ctx context.Context, companyID string) (*CompanyData, error)

	// GetVerifiedFieldNames returns the list of field names that are verified for a company
	// Verified fields should NOT be overwritten during re-enrichment
	GetVerifiedFieldNames(ctx context.Context, companyID string) ([]string, error)
}
