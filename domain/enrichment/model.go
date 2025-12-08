package enrichment

// EnrichedCompanyData is the response structure from enrichment
// Contains fields gathered via Gemini 3 Pro with web search
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

	// Products & Services (enrichment v3)
	MainProducts     []string `json:"main_products,omitempty"`      // Key products/services offered
	ServiceAreas     []string `json:"service_areas,omitempty"`      // Geographic or market areas served
	TechStack        []string `json:"tech_stack,omitempty"`         // Known technologies used
	Certifications   []string `json:"certifications,omitempty"`     // ISO, SOC2, etc.
	KeyPartnerships  []string `json:"key_partnerships,omitempty"`   // Strategic partners
	RecentNews       []string `json:"recent_news,omitempty"`        // Recent news headlines/events
	KeyExecutives    []string `json:"key_executives,omitempty"`     // CEO, CTO, etc. with roles
	CompanyHistory   *string  `json:"company_history,omitempty"`    // Brief history/milestones
	CultureValues    *string  `json:"culture_values,omitempty"`     // Company culture/values
	ESGInitiatives   *string  `json:"esg_initiatives,omitempty"`    // Environmental/Social/Governance
	CustomerSegments []string `json:"customer_segments,omitempty"`  // Target customer types
	PricingModel     *string  `json:"pricing_model,omitempty"`      // Subscription, usage-based, etc.
	MarketPosition   *string  `json:"market_position,omitempty"`    // Market positioning statement
	UniqueSellingPts []string `json:"unique_selling_points,omitempty"` // USPs/differentiators

	// Competitive Analysis (enrichment v3)
	Competitors          []string `json:"competitors,omitempty"`
	CompetitorDetails    []string `json:"competitor_details,omitempty"`    // Brief description of each competitor
	CompetitiveAdvantage *string  `json:"competitive_advantage,omitempty"` // Key competitive advantages
	MarketShare          *string  `json:"market_share,omitempty"`          // Estimated market share %

	// SWOT-ready data (enrichment v3)
	Strengths            []string `json:"strengths,omitempty"`
	Weaknesses           []string `json:"weaknesses,omitempty"`
	Opportunities        []string `json:"opportunities,omitempty"`        // Market opportunities
	Threats              []string `json:"threats,omitempty"`              // External threats
	StrategicChallenges  []string `json:"strategic_challenges,omitempty"` // Key challenges facing the company

	// Industry context (enrichment v2)
	IndustryGrowthRate  *string  `json:"industry_growth_rate,omitempty"`
	IndustryTrends      []string `json:"industry_trends,omitempty"`
	RegulatoryContext   *string  `json:"regulatory_context,omitempty"`
	MarketConcentration *string  `json:"market_concentration,omitempty"`

	// Market sizing (enrichment v3)
	TAMEstimate *string `json:"tam_estimate,omitempty"` // Total Addressable Market
	SAMEstimate *string `json:"sam_estimate,omitempty"` // Serviceable Addressable Market
	SOMEstimate *string `json:"som_estimate,omitempty"` // Serviceable Obtainable Market

	// Social links
	LinkedInURL   *string `json:"linkedin_url,omitempty"`
	TwitterHandle *string `json:"twitter_handle,omitempty"`

	// Meta
	ConfidenceScore float64  `json:"confidence_score"` // 0-100
	Sources         []string `json:"sources,omitempty"`
}
