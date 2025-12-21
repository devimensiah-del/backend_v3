package enrichment

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// FlexibleString handles JSON fields that might be returned as string or number
// LLMs sometimes return "foundation_year": 2023 instead of "foundation_year": "2023"
type FlexibleString string

// UnmarshalJSON handles both string and number JSON values
func (fs *FlexibleString) UnmarshalJSON(data []byte) error {
	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*fs = FlexibleString(s)
		return nil
	}

	// Try number (int or float)
	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		// Format as integer if it's a whole number (e.g., 2023)
		if n == float64(int64(n)) {
			*fs = FlexibleString(fmt.Sprintf("%d", int64(n)))
		} else {
			*fs = FlexibleString(fmt.Sprintf("%g", n))
		}
		return nil
	}

	// Check for null
	if string(data) == "null" {
		*fs = ""
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into FlexibleString", string(data))
}

// MarshalJSON outputs as a regular string
func (fs FlexibleString) MarshalJSON() ([]byte, error) {
	if fs == "" {
		return []byte("null"), nil
	}
	return json.Marshal(string(fs))
}

// String returns the string value
func (fs FlexibleString) String() string {
	return string(fs)
}

// ToStringPtr returns a *string pointer (nil if empty)
func (fs FlexibleString) ToStringPtr() *string {
	if fs == "" {
		return nil
	}
	s := string(fs)
	return &s
}

// =============================================================================
// STEP 1: BASIC INFO
// Fields: CNPJ, razão social, fundação, sede, funcionários, website, redes sociais, executivos
// Trigger: Automatic at company creation
// =============================================================================

// Step1BasicInfo contains data from Step 1 enrichment
// Static company data that rarely changes
type Step1BasicInfo struct {
	// Core identifiers
	CNPJ    *string `json:"cnpj,omitempty"`
	Website *string `json:"website,omitempty"`

	// Company details
	LegalName      *string        `json:"legal_name,omitempty"`      // Razão social
	TradeName      *string        `json:"trade_name,omitempty"`      // Nome fantasia
	FoundationYear FlexibleString `json:"foundation_year,omitempty"` // Ano de fundação (handles number or string from LLM)
	Headquarters   *string        `json:"headquarters,omitempty"`    // Sede principal
	EmployeesRange *string        `json:"employees_range,omitempty"` // Faixa de funcionários

	// Contact info (from CNPJ registry)
	Phone *string `json:"phone,omitempty"` // Telefone corporativo
	Email *string `json:"email,omitempty"` // Email corporativo

	// CNAE - Official business classification
	CNAEPrimary *string  `json:"cnae_primary,omitempty"` // Atividade principal (code + description)
	CNAECodes   []string `json:"cnae_codes,omitempty"`   // All CNAE codes (primary + secondary)

	// Financial
	CapitalSocial *string `json:"capital_social,omitempty"` // Capital social (e.g., "R$ 1.000,00")

	// Partners/Shareholders (from CNPJ registry - more accurate than key_executives for BR companies)
	Partners []string `json:"partners,omitempty"` // Sócios with roles (e.g., "João Silva - Sócio-Administrador")

	// Social links
	LinkedInURL   *string `json:"linkedin_url,omitempty"`
	TwitterHandle *string `json:"twitter_handle,omitempty"`
	InstagramURL  *string `json:"instagram_url,omitempty"`
	FacebookURL   *string `json:"facebook_url,omitempty"`

	// Leadership
	KeyExecutives []string `json:"key_executives,omitempty"` // CEO, fundadores, principais líderes

	// Meta
	ConfidenceScore float64  `json:"confidence_score"` // 0-100
	Sources         []string `json:"sources,omitempty"`
	CNPJVerified    bool     `json:"cnpj_verified"` // True if data came from official CNPJ registry
}

// =============================================================================
// STEP 2: BUSINESS MODEL
// Fields: modelo de negócio, produtos/serviços, público alvo, proposta de valor, região geográfica
// Trigger: Human-triggered after Step 1 completed
// =============================================================================

// Step2BusinessModel contains data from Step 2 enrichment
// Business context and go-to-market information
type Step2BusinessModel struct {
	// Business model
	BusinessModel *string `json:"business_model,omitempty"` // SaaS, Marketplace, Serviços, etc.
	Industry      *string `json:"industry,omitempty"`       // Setor
	Sector        *string `json:"sector,omitempty"`         // Subsetor

	// Products & Services
	MainProducts []string `json:"main_products,omitempty"` // Principais produtos/serviços
	PricingModel *string  `json:"pricing_model,omitempty"` // Assinatura, uso, freemium, etc.

	// Target market
	TargetMarket     *string  `json:"target_market,omitempty"`     // B2B, B2C, B2B2C
	TargetAudience   *string  `json:"target_audience,omitempty"`   // Descrição do público-alvo
	CustomerSegments []string `json:"customer_segments,omitempty"` // Segmentos de clientes

	// Value & positioning
	ValueProposition    *string  `json:"value_proposition,omitempty"`     // Proposta de valor
	UniqueSellingPoints []string `json:"unique_selling_points,omitempty"` // Diferenciais

	// Geographic reach
	GeographicRegions []string `json:"geographic_regions,omitempty"` // Regiões de atuação (Brasil, LATAM, etc.)
	ServiceAreas      []string `json:"service_areas,omitempty"`      // Áreas de serviço específicas

	// Meta
	ConfidenceScore float64  `json:"confidence_score"` // 0-100
	Sources         []string `json:"sources,omitempty"`
}

// =============================================================================
// STEP 3: COMPETITIVE INTELLIGENCE
// Fields: concorrentes, informações do setor, reputação, notícias recentes
// Trigger: Human-triggered after Step 2 completed
// Note: SWOT and TAM/SAM/SOM are handled by the analysis domain, not enrichment
// =============================================================================

// Step3CompetitiveIntel contains data from Step 3 enrichment
// Competitive landscape and market intelligence
type Step3CompetitiveIntel struct {
	// Competitors
	Competitors       []string `json:"competitors,omitempty"`        // Lista de concorrentes
	CompetitorDetails []string `json:"competitor_details,omitempty"` // Descrição breve de cada concorrente

	// Industry information
	IndustryGrowthRate  *string  `json:"industry_growth_rate,omitempty"` // Taxa de crescimento do setor
	IndustryTrends      []string `json:"industry_trends,omitempty"`      // Tendências do setor
	MarketConcentration *string  `json:"market_concentration,omitempty"` // Fragmentado/Concentrado
	RegulatoryContext   *string  `json:"regulatory_context,omitempty"`   // Contexto regulatório
	MarketPosition      *string  `json:"market_position,omitempty"`      // Posição no mercado

	// Reputation
	GlassdoorRating   *string `json:"glassdoor_rating,omitempty"`    // Nota Glassdoor
	ReclameAquiRating *string `json:"reclame_aqui_rating,omitempty"` // Nota Reclame Aqui
	ReputationSummary *string `json:"reputation_summary,omitempty"`  // Resumo da reputação

	// Recent news
	RecentNews []string `json:"recent_news,omitempty"` // Notícias recentes (últimos 12 meses)

	// Meta
	ConfidenceScore float64  `json:"confidence_score"` // 0-100
	Sources         []string `json:"sources,omitempty"`
}

// =============================================================================
// COMPANY ENRICHMENT ENTITY
// Database entity for company_enrichment table
// =============================================================================

// CompanyEnrichment represents the enrichment status and data for a company
type CompanyEnrichment struct {
	ID        uuid.UUID `json:"id" db:"id"`
	CompanyID uuid.UUID `json:"company_id" db:"company_id"`

	// Step 1: Basic Info
	Step1Status      string     `json:"step1_status" db:"step1_status"`
	Step1CompletedAt *time.Time `json:"step1_completed_at,omitempty" db:"step1_completed_at"`
	Step1Error       *string    `json:"step1_error,omitempty" db:"step1_error"`
	Step1Data        []byte     `json:"-" db:"step1_data"` // JSONB stored as bytes

	// Step 2: Business Model
	Step2Status      string     `json:"step2_status" db:"step2_status"`
	Step2CompletedAt *time.Time `json:"step2_completed_at,omitempty" db:"step2_completed_at"`
	Step2Error       *string    `json:"step2_error,omitempty" db:"step2_error"`
	Step2Data        []byte     `json:"-" db:"step2_data"` // JSONB stored as bytes

	// Step 3: Competitive Intel
	Step3Status      string     `json:"step3_status" db:"step3_status"`
	Step3CompletedAt *time.Time `json:"step3_completed_at,omitempty" db:"step3_completed_at"`
	Step3Error       *string    `json:"step3_error,omitempty" db:"step3_error"`
	Step3Data        []byte     `json:"-" db:"step3_data"` // JSONB stored as bytes

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Status constants
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// CanTriggerStep2 returns true if Step 2 can be triggered
func (e *CompanyEnrichment) CanTriggerStep2() bool {
	return e.Step1Status == StatusCompleted && e.Step2Status == StatusPending
}

// CanTriggerStep3 returns true if Step 3 can be triggered
func (e *CompanyEnrichment) CanTriggerStep3() bool {
	return e.Step2Status == StatusCompleted && e.Step3Status == StatusPending
}

// NewCompanyEnrichment creates a new enrichment record for a company
func NewCompanyEnrichment(companyID uuid.UUID) *CompanyEnrichment {
	now := time.Now()
	return &CompanyEnrichment{
		ID:          uuid.New(),
		CompanyID:   companyID,
		Step1Status: StatusPending,
		Step2Status: StatusPending,
		Step3Status: StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// =============================================================================
// LEGACY SUPPORT
// Keep EnrichedCompanyData for backward compatibility during migration
// =============================================================================

// EnrichedCompanyData is DEPRECATED - use Step1BasicInfo, Step2BusinessModel, Step3CompetitiveIntel
// Kept for backward compatibility with existing code
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

	// Products & Services
	MainProducts     []string `json:"main_products,omitempty"`
	ServiceAreas     []string `json:"service_areas,omitempty"`
	TechStack        []string `json:"tech_stack,omitempty"`
	Certifications   []string `json:"certifications,omitempty"`
	KeyPartnerships  []string `json:"key_partnerships,omitempty"`
	RecentNews       []string `json:"recent_news,omitempty"`
	KeyExecutives    []string `json:"key_executives,omitempty"`
	CompanyHistory   *string  `json:"company_history,omitempty"`
	CultureValues    *string  `json:"culture_values,omitempty"`
	ESGInitiatives   *string  `json:"esg_initiatives,omitempty"`
	CustomerSegments []string `json:"customer_segments,omitempty"`
	PricingModel     *string  `json:"pricing_model,omitempty"`
	MarketPosition   *string  `json:"market_position,omitempty"`
	UniqueSellingPts []string `json:"unique_selling_points,omitempty"`

	// Competitive Analysis
	Competitors          []string `json:"competitors,omitempty"`
	CompetitorDetails    []string `json:"competitor_details,omitempty"`
	CompetitiveAdvantage *string  `json:"competitive_advantage,omitempty"`
	MarketShare          *string  `json:"market_share,omitempty"`

	// SWOT-ready data
	Strengths           []string `json:"strengths,omitempty"`
	Weaknesses          []string `json:"weaknesses,omitempty"`
	Opportunities       []string `json:"opportunities,omitempty"`
	Threats             []string `json:"threats,omitempty"`
	StrategicChallenges []string `json:"strategic_challenges,omitempty"`

	// Industry context
	IndustryGrowthRate  *string  `json:"industry_growth_rate,omitempty"`
	IndustryTrends      []string `json:"industry_trends,omitempty"`
	RegulatoryContext   *string  `json:"regulatory_context,omitempty"`
	MarketConcentration *string  `json:"market_concentration,omitempty"`

	// Market sizing
	TAMEstimate *string `json:"tam_estimate,omitempty"`
	SAMEstimate *string `json:"sam_estimate,omitempty"`
	SOMEstimate *string `json:"som_estimate,omitempty"`

	// CNPJ Registry data (from casadosdados.com.br)
	TradeName     *string  `json:"trade_name,omitempty"`
	Phone         *string  `json:"phone,omitempty"`
	Email         *string  `json:"email,omitempty"`
	CNAEPrimary   *string  `json:"cnae_primary,omitempty"`
	CNAECodes     []string `json:"cnae_codes,omitempty"`
	CapitalSocial *string  `json:"capital_social,omitempty"`
	Partners      []string `json:"partners,omitempty"`
	CNPJVerified  bool     `json:"cnpj_verified"`

	// Social links
	LinkedInURL   *string `json:"linkedin_url,omitempty"`
	TwitterHandle *string `json:"twitter_handle,omitempty"`
	InstagramURL  *string `json:"instagram_url,omitempty"`
	FacebookURL   *string `json:"facebook_url,omitempty"`

	// Meta
	ConfidenceScore float64  `json:"confidence_score"`
	Sources         []string `json:"sources,omitempty"`
}
