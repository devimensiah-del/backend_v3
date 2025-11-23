package enrichment

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// JSONMap handles the JSONB interaction with Postgres
type JSONMap map[string]interface{}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = make(JSONMap)
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, m)
}

// =================================================================
// UNIFIED INTELLIGENCE PROFILE
// This matches the output of the UnifiedEnrichmentPrompt
// =================================================================

type UnifiedProfile struct {
	ProfileOverview      ProfileOverview      `json:"profile_overview"`
	MarketPosition       MarketPosition       `json:"market_position"`
	Financials           Financials           `json:"financials"`
	CompetitiveLandscape CompetitiveLandscape `json:"competitive_landscape"`
	StrategicAssessment  StrategicAssessment  `json:"strategic_assessment"`
	MacroContext         *MacroContext        `json:"macro_context,omitempty"` // NEW: Macro-economic and industry context
}

type ProfileOverview struct {
	LegalName      string `json:"legal_name"`
	Website        string `json:"website"`
	FoundationYear string `json:"foundation_year"`
	Headquarters   string `json:"headquarters"`
}

type MarketPosition struct {
	Sector           string `json:"sector"`
	TargetAudience   string `json:"target_audience"`
	ValueProposition string `json:"value_proposition"`
}

type Financials struct {
	EmployeesRange  string `json:"employees_range"`
	RevenueEstimate string `json:"revenue_estimate"`
	BusinessModel   string `json:"business_model"`
}

type CompetitiveLandscape struct {
	Competitors       []string `json:"competitors"`
	MarketShareStatus string   `json:"market_share_status"`
}

type StrategicAssessment struct {
	DigitalMaturity int      `json:"digital_maturity"`
	Strengths       []string `json:"strengths"`
	Weaknesses      []string `json:"weaknesses"`
}

// =================================================================
// MACRO-ECONOMIC & INDUSTRY CONTEXT
// NEW: Addresses the "Brazil blind spot" by capturing real-time
// macro-economic data, industry trends, and regulatory landscape
// =================================================================

type MacroContext struct {
	EconomicIndicators  EconomicIndicators  `json:"economic_indicators"`
	IndustryTrends      IndustryTrends      `json:"industry_trends"`
	RegulatoryLandscape RegulatoryLandscape `json:"regulatory_landscape"`
	MarketSignals       MarketSignals       `json:"market_signals"`
	DataSources         []string            `json:"data_sources"` // URLs/sources used
	LastUpdated         string              `json:"last_updated"` // Timestamp
}

type EconomicIndicators struct {
	Country              string   `json:"country"`               // e.g., "Brazil"
	GDPGrowth            string   `json:"gdp_growth"`            // e.g., "+2.5% (2025 forecast)"
	InflationRate        string   `json:"inflation_rate"`        // e.g., "IPCA: 4.8% a.a."
	InterestRate         string   `json:"interest_rate"`         // e.g., "SELIC: 11.75%"
	ExchangeRate         string   `json:"exchange_rate"`         // e.g., "USD/BRL: 5.20"
	UnemploymentRate     string   `json:"unemployment_rate"`     // e.g., "7.2%"
	PoliticalStability   string   `json:"political_stability"`   // e.g., "Moderate risk due to reforms"
	EconomicOutlook      string   `json:"economic_outlook"`      // Brief summary
	RecentPolicyChanges  []string `json:"recent_policy_changes"` // e.g., ["Tax reform 2025", "New carbon policies"]
}

type IndustryTrends struct {
	IndustrySector       string   `json:"industry_sector"`        // e.g., "Agribusiness Technology"
	GrowthRate           string   `json:"growth_rate"`            // e.g., "+12% CAGR (2024-2028)"
	KeyTrends            []string `json:"key_trends"`             // e.g., ["IoT adoption", "Sustainability focus", "Digital transformation"]
	TechnologyAdoption   string   `json:"technology_adoption"`    // e.g., "High adoption of cloud/AI in sector"
	MarketConcentration  string   `json:"market_concentration"`   // e.g., "Fragmented - no dominant player"
	BarriersToEntry      string   `json:"barriers_to_entry"`      // e.g., "Medium - requires tech expertise and capital"
	MergersAcquisitions  []string `json:"mergers_acquisitions"`   // Recent M&A activity
}

type RegulatoryLandscape struct {
	RecentRegulations    []string `json:"recent_regulations"`    // e.g., ["Lei do Agro 2025", "New environmental compliance"]
	UpcomingChanges      []string `json:"upcoming_changes"`      // e.g., ["Carbon tax proposal", "Data privacy law update"]
	ComplianceRequirements string `json:"compliance_requirements"` // Key compliance needs
	IndustryStandards    []string `json:"industry_standards"`    // Relevant standards (ISO, etc.)
}

type MarketSignals struct {
	CommodityPrices      []string `json:"commodity_prices"`       // e.g., ["Steel prices up 12% YoY", "Copper prices stable"]
	SupplyChainStatus    string   `json:"supply_chain_status"`    // e.g., "Moderate delays in electronics components"
	ConsumerSentiment    string   `json:"consumer_sentiment"`     // e.g., "Cautious due to inflation concerns"
	CompetitorActivity   []string `json:"competitor_activity"`    // e.g., ["Competitor X launched new product", "Competitor Y expanded to new region"]
	EmergingThreats      []string `json:"emerging_threats"`       // e.g., ["New low-cost entrant from China", "Substitute technology gaining traction"]
}

// =================================================================
// Database Entity
// =================================================================

type Enrichment struct {
	ID           uuid.UUID `json:"id" db:"id"`
	SubmissionID uuid.UUID `json:"submissionId" db:"submission_id"`

	Status      Status `json:"status" db:"status"`
	Progress    int    `json:"progress" db:"progress"`
	CurrentStep string `json:"currentStep" db:"current_step"`

	IsLocked bool `json:"isLocked" db:"is_locked"`

	// Generic JSON storage for flexibility
	SourcesStatus JSONMap `json:"sourcesStatus,omitempty" db:"sources_status"`
	EnrichedData  JSONMap `json:"enrichedData,omitempty" db:"enriched_data"`

	StartedAt    *time.Time `json:"startedAt,omitempty" db:"started_at"`
	CompletedAt  *time.Time `json:"completedAt,omitempty" db:"completed_at"`
	ErrorMessage string     `json:"errorMessage,omitempty" db:"error_message"`
	RetryCount   int        `json:"retryCount" db:"retry_count"`
	MaxRetries   int        `json:"maxRetries" db:"max_retries"`

	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type Status string

const (
	StatusPending  Status = "pending"  // Queued, worker not started
	StatusFinished Status = "finished" // Worker completed enrichment
	StatusApproved Status = "approved" // Admin approved, ready for analysis
	// Note: Removed StatusProcessing and StatusFailed
	// Failures keep status as "pending" with error_message populated
)

func NewEnrichment(submissionID uuid.UUID) *Enrichment {
	now := time.Now()
	return &Enrichment{
		ID:            uuid.New(),
		SubmissionID:  submissionID,
		Status:        StatusPending,
		Progress:      0,
		CurrentStep:   "Queued for enrichment",
		IsLocked:      false,
		SourcesStatus: make(JSONMap),
		EnrichedData:  make(JSONMap),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func (e *Enrichment) Start() {
	// NOTE: Removed status transition to "processing" - status stays "pending"
	// Worker starts execution without changing status
	n := time.Now()
	e.StartedAt = &n
	e.UpdatedAt = n
}

func (e *Enrichment) Finish() {
	e.Status = StatusFinished
	n := time.Now()
	e.CompletedAt = &n
	e.Progress = 100
	e.UpdatedAt = n
}

func (e *Enrichment) UpdateProgress(step string, pct int) {
	e.CurrentStep = step
	e.Progress = pct
	e.UpdatedAt = time.Now()
}

func (e *Enrichment) Fail(err error) {
	// NOTE: Keep status as "pending" for retryable failures
	// Error message indicates the failure reason
	e.Status = StatusPending
	e.ErrorMessage = err.Error()
	e.UpdatedAt = time.Now()
}
