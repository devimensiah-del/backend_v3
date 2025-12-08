package enrichment

// EnrichedCompanyData is the response structure from Perplexity enrichment
// Contains only the fields that Perplexity can confidently fill
type EnrichedCompanyData struct {
	// Core identifiers
	CNPJ    *string `json:"cnpj,omitempty"`
	Website *string `json:"website,omitempty"`

	// Business context
	Industry         *string  `json:"industry,omitempty"`
	CompanySize      *string  `json:"company_size,omitempty"`
	Location         *string  `json:"location,omitempty"`
	TargetMarket     *string  `json:"target_market,omitempty"`
	FundingStage     *string  `json:"funding_stage,omitempty"`
	AnnualRevenueMin *float64 `json:"annual_revenue_min,omitempty"`
	AnnualRevenueMax *float64 `json:"annual_revenue_max,omitempty"`

	// Enriched data
	FoundationYear    *string `json:"foundation_year,omitempty"`
	LegalName         *string `json:"legal_name,omitempty"`
	Headquarters      *string `json:"headquarters,omitempty"`
	Sector            *string `json:"sector,omitempty"`
	TargetAudience    *string `json:"target_audience,omitempty"`
	ValueProposition  *string `json:"value_proposition,omitempty"`
	EmployeesRange    *string `json:"employees_range,omitempty"`
	RevenueEstimate   *string `json:"revenue_estimate,omitempty"`
	BusinessModel     *string `json:"business_model,omitempty"`
	MarketShareStatus *string `json:"market_share_status,omitempty"`
	DigitalMaturity   *int    `json:"digital_maturity,omitempty"`

	// Arrays
	Competitors []string `json:"competitors,omitempty"`
	Strengths   []string `json:"strengths,omitempty"`
	Weaknesses  []string `json:"weaknesses,omitempty"`

	// Industry context (enrichment v2)
	IndustryGrowthRate  *string  `json:"industry_growth_rate,omitempty"`
	IndustryTrends      []string `json:"industry_trends,omitempty"`
	RegulatoryContext   *string  `json:"regulatory_context,omitempty"`
	MarketConcentration *string  `json:"market_concentration,omitempty"`

	// Social links
	LinkedInURL   *string `json:"linkedin_url,omitempty"`
	TwitterHandle *string `json:"twitter_handle,omitempty"`

	// Meta
	ConfidenceScore float64  `json:"confidence_score"` // 0-100
	Sources         []string `json:"sources,omitempty"`
}
