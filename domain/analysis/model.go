package analysis

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Status constants for Analysis
// Flow: pending → processing → completed/failed
// - pending: Queued, waiting for worker
// - processing: Worker is actively running analysis
// - completed: Worker finished analysis successfully
// - failed: Worker encountered an error
type Status string

const (
	StatusPending    Status = "pending"    // Queued, waiting for worker
	StatusProcessing Status = "processing" // Worker is actively running analysis
	StatusCompleted  Status = "completed"  // Worker finished analysis successfully
	StatusFailed     Status = "failed"     // Worker encountered an error
)

type Analysis struct {
	ID           string  `db:"id" json:"id"`
	SubmissionID string  `db:"submission_id" json:"submission_id"`
	CompanyID    *string `db:"company_id" json:"company_id,omitempty"` // Direct link to company (migration 026)
	EnrichmentID string  `db:"enrichment_id" json:"enrichment_id"`

	// Status and processing
	Status           string     `db:"status" json:"status"`
	ErrorMessage     *string    `db:"error_message" json:"error_message,omitempty"` // Error details if analysis failed
	ProcessingTimeMs int64      `db:"processing_time_ms" json:"processing_time_ms"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
	CompletedAt      *time.Time `db:"completed_at" json:"completed_at"`

	// PDF storage (moved from reports table - migration 028)
	PdfUrl         *string    `db:"pdf_url" json:"pdf_url,omitempty"`
	PdfGeneratedAt *time.Time `db:"pdf_generated_at" json:"pdf_generated_at,omitempty"`

	// Visibility control - Admin must explicitly make analysis visible to user
	IsVisibleToUser bool `db:"is_visible_to_user" json:"is_visible_to_user"`

	// Blur control - When true, premium frameworks are blurred for users
	// Independent of visibility (can be visible but blurred)
	IsBlurred bool `db:"is_blurred" json:"is_blurred"`

	// Public sharing via access code
	AccessCode          *string    `db:"access_code" json:"access_code,omitempty"`
	AccessCodeCreatedAt *time.Time `db:"access_code_created_at" json:"access_code_created_at,omitempty"`

	// Soft delete support
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`

	// The 11 Frameworks (JSONB in Postgres)
	PESTEL         PESTELAnalysis            `db:"pestel" json:"pestel"`
	Porter         PorterAnalysis            `db:"porter" json:"porter"`
	TamSamSom      TamSamSomAnalysis         `db:"tam_sam_som" json:"tam_sam_som"`
	SWOT           SWOTAnalysis              `db:"swot" json:"swot"`
	Benchmarking   BenchmarkingAnalysis      `db:"benchmarking" json:"benchmarking"`
	BlueOcean      BlueOceanAnalysis         `db:"blue_ocean" json:"blue_ocean"`
	GrowthHacking  GrowthHackingAnalysis     `db:"growth_hacking" json:"growth_hacking"`
	Scenarios      ScenarioAnalysis          `db:"scenarios" json:"scenarios"`
	OKRs           OKRAnalysis               `db:"okrs" json:"okrs"`
	BSC            BalancedScorecardAnalysis `db:"bsc" json:"bsc"`
	DecisionMatrix DecisionMatrixAnalysis    `db:"decision_matrix" json:"decision_matrix"`

	// Final Output
	Synthesis AnalysisSynthesis `db:"synthesis" json:"synthesis"`
}

// --- Framework Structs ---

type PESTELAnalysis struct {
	Political     []string `json:"political"`
	Economic      []string `json:"economic"`
	Social        []string `json:"social"`
	Technological []string `json:"technological"`
	Environmental []string `json:"environmental"`
	Legal         []string `json:"legal"`
	Summary       string   `json:"summary"`
}

type PorterAnalysis struct {
	// Traditional 5 Forces
	CompetitiveRivalry string `json:"competitive_rivalry"`
	SupplierPower      string `json:"supplier_power"`
	BuyerPower         string `json:"buyer_power"`
	ThreatNewEntrants  string `json:"threat_new_entrants"`
	ThreatSubstitutes  string `json:"threat_substitutes"`

	// +2 Modern Forces (2025+)
	PowerPartnershipsEcosystems string `json:"power_partnerships_ecosystems"` // Collaborative networks & platform effects
	DisruptionAIData            string `json:"disruption_ai_data"`            // AI/Data-driven disruption potential

	// Intensity Ratings (Alta/Média/Baixa) for each force
	CompetitiveRivalryIntensity          string `json:"competitive_rivalry_intensity"`
	SupplierPowerIntensity               string `json:"supplier_power_intensity"`
	BuyerPowerIntensity                  string `json:"buyer_power_intensity"`
	ThreatNewEntrantsIntensity           string `json:"threat_new_entrants_intensity"`
	ThreatSubstitutesIntensity           string `json:"threat_substitutes_intensity"`
	PowerPartnershipsEcosystemsIntensity string `json:"power_partnerships_ecosystems_intensity"`
	DisruptionAIDataIntensity            string `json:"disruption_ai_data_intensity"`

	// Strategic Implications (4 key actionable points)
	StrategicImplications []string `json:"strategic_implications"`

	OverallAttractiveness string `json:"overall_attractiveness"`
	Summary               string `json:"summary"`
}

// SWOTItem represents a single SWOT item with confidence level and source attribution
type SWOTItem struct {
	Content    string `json:"content"`
	Confidence string `json:"confidence"` // "Alta" | "Média" | "Baixa"
	Source     string `json:"source"`     // "fato" | "análise de mercado" | "estimativa" | "feedback de clientes"
}

type SWOTAnalysis struct {
	Strengths     []SWOTItem `json:"strengths"`     // Enhanced with confidence & source
	Weaknesses    []SWOTItem `json:"weaknesses"`    // Enhanced with confidence & source
	Opportunities []SWOTItem `json:"opportunities"` // Enhanced with confidence & source
	Threats       []SWOTItem `json:"threats"`       // Enhanced with confidence & source
	Summary       string     `json:"summary"`
}

type BlueOceanAnalysis struct {
	Eliminate     []string `json:"eliminate"`
	Reduce        []string `json:"reduce"`
	Raise         []string `json:"raise"`
	Create        []string `json:"create"`
	NewValueCurve string   `json:"new_value_curve"`
	Summary       string   `json:"summary"`
}

type BalancedScorecardAnalysis struct {
	Financial      []string `json:"financial"`
	Customer       []string `json:"customer"`
	Internal       []string `json:"internal_processes"` // Maps to validator "Internal"
	LearningGrowth []string `json:"learning_growth"`
	Summary        string   `json:"summary"`
}

// QuarterlyOKR represents OKRs for a specific quarter with investment estimates
// DEPRECATED: Legacy format with hardcoded quarters (Q1 2025, Q2 2025, Q3 2025)
// Kept for backwards compatibility with existing analyses
type QuarterlyOKR struct {
	Quarter    string   `json:"quarter"`     // "Q1 2025", "Q2 2025", "Q3 2025"
	Objective  string   `json:"objective"`   // Main objective description
	KeyResults []string `json:"key_results"` // KR1, KR2, KR3 (exactly 3)
	Investment string   `json:"investment"`  // Investment estimate (e.g., "R$ 25 mil" or range "R$ 20-30k")
	Timeline   string   `json:"timeline"`    // Duration (e.g., "3-4 meses")
}

// MonthlyOKR represents OKRs for a month within the 90-day plan
// V2 format: Uses relative dates (Mês 1, Mês 2, Mês 3) instead of hardcoded quarters
type MonthlyOKR struct {
	Month                 string   `json:"month"`                  // "Mês 1", "Mês 2", "Mês 3"
	Focus                 string   `json:"focus"`                  // "Fundação", "Crescimento", "Consolidação"
	Objective             string   `json:"objective"`              // Main objective description
	KeyResults            []string `json:"key_results"`            // KR1, KR2, KR3 (exactly 3)
	Investment            string   `json:"investment"`             // Investment estimate for this month
	AlignedRecommendation string   `json:"aligned_recommendation"` // Links to Decision Matrix recommendation
}

// CapacityRequirements defines resource requirements for an execution phase
type CapacityRequirements struct {
	Team         string   `json:"team"`          // e.g., "5-10 pessoas"
	SkillsNeeded []string `json:"skills_needed"` // Required capabilities
	Dependencies []string `json:"dependencies"`  // External dependencies (partners, approvals)
}

// ExecutionPhase represents a phase in the phased execution model (Foundation/Validation/Scale)
type ExecutionPhase struct {
	Duration             string                `json:"duration"`                        // e.g., "Meses 1-3"
	Focus                string                `json:"focus"`                           // e.g., "Estruturação"
	Prerequisites        []string              `json:"prerequisites,omitempty"`         // What must be completed first
	Objectives           []string              `json:"objectives"`                      // Phase objectives
	KeyResults           []string              `json:"key_results"`                     // Measurable results
	InvestmentRange      string                `json:"investment_range"`                // e.g., "R$ 50-100k"
	CapacityRequirements *CapacityRequirements `json:"capacity_requirements,omitempty"` // Resource needs
}

// CapacityAssessment evaluates organizational readiness for execution
type CapacityAssessment struct {
	TeamReadiness        string   `json:"team_readiness"`        // "Alta" | "Média" | "Baixa"
	BudgetAdequacy       string   `json:"budget_adequacy"`       // "Suficiente" | "Limitado" | "Insuficiente"
	StakeholderAlignment string   `json:"stakeholder_alignment"` // "Alinhado" | "Parcial" | "Resistência"
	Blockers             []string `json:"blockers,omitempty"`    // Identified blockers
}

type OKRAnalysis struct {
	// V2: 90-Day Plan format with monthly milestones (preferred)
	Plan90Days      []MonthlyOKR `json:"plan_90_days,omitempty"` // Monthly OKR structure (Mês 1, 2, 3)
	TotalInvestment string       `json:"total_investment,omitempty"`
	SuccessMetrics  []string     `json:"success_metrics,omitempty"`

	// V1 Legacy: Quarterly format (for backwards compatibility)
	// When reading existing data, check if Quarters has data; if Plan90Days is empty, use Quarters
	Quarters []QuarterlyOKR `json:"quarters,omitempty"` // Quarterly OKR structure (Q1, Q2, Q3)

	// V3: Phased execution model (NeoGovernança realism enhancement)
	// 3-phase structure: Foundation (1-3mo) → Validation (4-6mo) → Scale (7-12mo)
	ExecutionPhases    map[string]*ExecutionPhase `json:"execution_phases,omitempty"`    // "foundation", "validation", "scale"
	CapacityAssessment *CapacityAssessment        `json:"capacity_assessment,omitempty"` // Organizational readiness

	Summary string `json:"summary"`
}

// DEPRECATED: Legacy OKRObjective kept for backward compatibility
type BenchmarkingAnalysis struct {
	Competitors     []string `json:"competitors_analyzed"`
	PerformanceGaps []string `json:"performance_gaps"` // Fixes validator error
	BestPractices   []string `json:"best_practices"`   // Fixes validator error
	Summary         string   `json:"summary"`
}

// GrowthLoop represents a structured growth loop (LEAP or SCALE)
type GrowthLoop struct {
	Name       string   `json:"name"`       // "LEAP Loop" or "SCALE Loop"
	Type       string   `json:"type"`       // "acquisition" or "monetization"
	Steps      []string `json:"steps"`      // 4 steps (e.g., ["Land", "Engage", "Activate", "Propagate"])
	Metrics    []string `json:"metrics"`    // Key metrics to track (e.g., ["CAC", "Taxa de Conversão"])
	Bottleneck string   `json:"bottleneck"` // Identified bottleneck in the loop
}

type GrowthHackingAnalysis struct {
	LeapLoop  GrowthLoop `json:"leap_loop"`  // LEAP Loop (Acquisition): Land, Engage, Activate, Propagate
	ScaleLoop GrowthLoop `json:"scale_loop"` // SCALE Loop (Monetization): Satisfy, Convert, Loop Back, Expand
	Summary   string     `json:"summary"`
}

// Scenario represents a future scenario with probability and required actions
type Scenario struct {
	Name            string   `json:"name"`             // "Cenário Otimista", "Cenário Realista", "Cenário Pessimista"
	Probability     int      `json:"probability"`      // Probability percentage (e.g., 20, 60, 20)
	Description     string   `json:"description"`      // Scenario description (max 450 chars)
	RequiredActions []string `json:"required_actions"` // Actions to take if this scenario materializes
}

type ScenarioAnalysis struct {
	Optimistic  Scenario `json:"optimistic"`  // Optimistic scenario (typically 20%)
	Realist     Scenario `json:"realist"`     // Realistic scenario (typically 60%)
	Pessimistic Scenario `json:"pessimistic"` // Pessimistic scenario (typically 20%)

	MitigationTactics   []string `json:"mitigation_tactics"`    // Risk mitigation strategies
	EarlyWarningSignals []string `json:"early_warning_signals"` // Indicators that signal scenario shifts
	Summary             string   `json:"summary"`
}

type TamSamSomAnalysis struct {
	TAM         string   `json:"tam"`
	SAM         string   `json:"sam"`
	SOM         string   `json:"som"`
	Assumptions []string `json:"assumptions"`
	CAGR        string   `json:"cagr"`

	// V2: Estimation transparency fields
	ConfidenceLevel  int    `json:"confidence_level"`  // 0-100, based on data quality
	EstimationMethod string `json:"estimation_method"` // How the estimate was derived
	CalculationNotes string `json:"calculation_notes"` // Brief explanation of calculation

	// Data sources breakdown for transparency
	DataSourcesUsed *DataSourcesUsed `json:"data_sources_used,omitempty"`

	// Legacy fields (kept for backwards compatibility)
	DataQuality        string   `json:"data_quality,omitempty"`        // "complete" | "partial" | "insufficient"
	NextSteps          []string `json:"next_steps,omitempty"`          // Steps to gather missing data
	ProxyIndicators    []string `json:"proxy_indicators,omitempty"`    // Alternative metrics when data is insufficient
	ExpectedOutputs    []string `json:"expected_outputs,omitempty"`    // What complete analysis should include
	MethodologicalNote string   `json:"methodological_note,omitempty"` // Explanation of methodology or data limitations

	// Caveat for low confidence estimates
	CaveatMessage string `json:"caveat_message,omitempty"` // Warning if confidence < 50

	// V3: 3-tier scenario modeling (NeoGovernança realism enhancement)
	TamScenarios           *TamSamSomScenarios `json:"tam_scenarios,omitempty"`
	SamScenarios           *TamSamSomScenarios `json:"sam_scenarios,omitempty"`
	SomScenarios           *TamSamSomScenarios `json:"som_scenarios,omitempty"`
	BaselineRecommendation string              `json:"baseline_recommendation,omitempty"` // "conservative" | "realistic"

	Summary string `json:"summary"`
}

// DataSourcesUsed provides transparency about estimation methodology
type DataSourcesUsed struct {
	DirectData              []string `json:"direct_data,omitempty"`              // Actual market data found
	ProxyCalculations       []string `json:"proxy_calculations,omitempty"`       // Derived from related metrics
	BenchmarkExtrapolations []string `json:"benchmark_extrapolations,omitempty"` // Industry benchmarks applied
}

// ScenarioEstimate represents a single scenario (conservative/realistic/optimistic)
type ScenarioEstimate struct {
	ValueRange  string `json:"value_range"`            // e.g., "R$ 500k-2M"
	MarketShare string `json:"market_share,omitempty"` // e.g., "2-5%"
	Probability int    `json:"probability"`            // 20, 30, 50
}

// TamSamSomScenarios represents 3-tier scenario modeling for market estimates
type TamSamSomScenarios struct {
	Conservative ScenarioEstimate `json:"conservative"` // Pessimistic: execution challenges
	Realistic    ScenarioEstimate `json:"realistic"`    // Moderate: good execution
	Optimistic   ScenarioEstimate `json:"optimistic"`   // Favorable: exceptional positioning
}

// LegalFeasibility assesses juridical risk for a recommendation
type LegalFeasibility struct {
	RiskLevel               string   `json:"risk_level"`                        // "Baixo" | "Médio" | "Alto" | "Crítico"
	RequiresStatutoryChange bool     `json:"requires_statutory_change"`         // Needs assembly/statute changes
	RequiresLegalOpinion    bool     `json:"requires_legal_opinion"`            // Should get legal counsel
	RegulatoryDependencies  []string `json:"regulatory_dependencies,omitempty"` // Required licenses/approvals
	MitigationPlan          string   `json:"mitigation_plan,omitempty"`         // Risk mitigation strategy
}

// PriorityRecommendation represents a prioritized strategic recommendation
type PriorityRecommendation struct {
	Priority    int    `json:"priority"`    // 1, 2, 3 (ordered by priority)
	Title       string `json:"title"`       // Recommendation title
	Description string `json:"description"` // Detailed description
	Timeline    string `json:"timeline"`    // Expected duration (e.g., "9-12 meses")
	Budget      string `json:"budget"`      // Budget estimate (e.g., "R$150-250k")

	// V3: Legal feasibility assessment (NeoGovernança realism enhancement)
	LegalFeasibility *LegalFeasibility `json:"legal_feasibility,omitempty"`
}

// ReviewCycle defines the review and monitoring cadence
type ReviewCycle struct {
	Frequency             string   `json:"frequency"`              // "Trimestral", "Mensal", etc.
	ExtraordinaryTriggers []string `json:"extraordinary_triggers"` // Events that trigger extraordinary review
}

type DecisionMatrixAnalysis struct {
	Alternatives        []string `json:"alternatives"`
	Criteria            []string `json:"criteria"`
	FinalRecommendation string   `json:"final_recommendation"`

	// Enhanced Decision Support (TUC Glasses alignment)
	RecommendedOption       string                   `json:"recommended_option"`       // Best option name
	Score                   string                   `json:"score"`                    // Score (e.g., "7.3/10")
	ScoreComparison         string                   `json:"score_comparison"`         // Comparison to alternatives (e.g., "+23% above second")
	PriorityRecommendations []PriorityRecommendation `json:"priority_recommendations"` // Top 3 recommendations with budgets & timelines
	ReviewCycle             ReviewCycle              `json:"review_cycle"`             // Review frequency and triggers
	MonitoringMetrics       []string                 `json:"monitoring_metrics"`       // Metrics to track execution

	Summary string `json:"summary"`
}

// ConsistencyValidation checks alignment across analysis frameworks
type ConsistencyValidation struct {
	FinancialAlignment  bool     `json:"financial_alignment"`   // OKRs achievable with conservative SOM?
	CapacityAlignment   bool     `json:"capacity_alignment"`    // Team/resources support the plan?
	LegalAlignment      bool     `json:"legal_alignment"`       // High-risk items have mitigation?
	OverallRealismScore int      `json:"overall_realism_score"` // 1-10 realism rating
	Flags               []string `json:"flags,omitempty"`       // Inconsistencies or warnings
}

type AnalysisSynthesis struct {
	ExecutiveSummary string `json:"executive_summary"`

	// Enhanced Executive Summary Components (TUC Glasses alignment)
	CentralChallenge string   `json:"central_challenge"` // Primary strategic challenge facing the company
	MainFindings     []string `json:"main_findings"`     // 4-point SWOT summary from executive summary
	ImportantNotes   []string `json:"important_notes"`   // Critical observations and warnings

	KeyFindings           []string `json:"key_findings"`
	StrategicPriorities   []string `json:"strategic_priorities"`
	Roadmap               []string `json:"roadmap"`
	OverallRecommendation string   `json:"overall_recommendation"`

	// V3: Cross-framework consistency validation (NeoGovernança realism enhancement)
	ConsistencyValidation *ConsistencyValidation `json:"consistency_validation,omitempty"`
}

// ContextContainer used during processing
type ContextContainer struct {
	SubmissionID   string
	SubmissionData map[string]interface{} // User's company data from submission
	EnrichmentData map[string]interface{} // External intelligence data
	// Pointers to hold interim results
	PESTEL         *PESTELAnalysis
	Porter         *PorterAnalysis
	TamSamSom      *TamSamSomAnalysis
	SWOT           *SWOTAnalysis
	Benchmarking   *BenchmarkingAnalysis
	BlueOcean      *BlueOceanAnalysis
	GrowthHacking  *GrowthHackingAnalysis
	Scenarios      *ScenarioAnalysis
	OKRs           *OKRAnalysis
	BSC            *BalancedScorecardAnalysis
	DecisionMatrix *DecisionMatrixAnalysis
}

// Status management methods for Analysis

// Start marks the analysis as processing (worker started)
func (a *Analysis) Start() {
	a.Status = string(StatusProcessing)
	a.UpdatedAt = time.Now()
}

// Complete marks the analysis as successfully completed
func (a *Analysis) Complete() {
	a.Status = string(StatusCompleted)
	n := time.Now()
	a.CompletedAt = &n
	a.UpdatedAt = n
}

// Fail marks the analysis as failed with an error message
func (a *Analysis) Fail(errorMsg string) {
	a.Status = string(StatusFailed)
	a.ErrorMessage = &errorMsg
	a.UpdatedAt = time.Now()
}

// SetPdf records the PDF URL and generation timestamp
func (a *Analysis) SetPdf(pdfUrl string) {
	a.PdfUrl = &pdfUrl
	now := time.Now()
	a.PdfGeneratedAt = &now
	a.UpdatedAt = now
}

// MakeVisible makes the analysis visible to the user
func (a *Analysis) MakeVisible() {
	a.IsVisibleToUser = true
	a.UpdatedAt = time.Now()
}

// MakeInvisible hides the analysis from the user
func (a *Analysis) MakeInvisible() {
	a.IsVisibleToUser = false
	a.UpdatedAt = time.Now()
}

// SetBlurred enables premium content blurring
func (a *Analysis) SetBlurred(blurred bool) {
	a.IsBlurred = blurred
	a.UpdatedAt = time.Now()
}

// HasPdf returns true if a PDF has been generated
func (a *Analysis) HasPdf() bool {
	return a.PdfUrl != nil && *a.PdfUrl != ""
}

// IsCompleted returns true if analysis completed successfully
func (a *Analysis) IsCompleted() bool {
	return a.Status == string(StatusCompleted)
}

// IsFailed returns true if analysis failed
func (a *Analysis) IsFailed() bool {
	return a.Status == string(StatusFailed)
}

// ============================================
// JSONB Serialization for PostgreSQL
// ============================================
// Implement driver.Valuer and sql.Scanner for all framework types
// This enables PostgreSQL JSONB storage and retrieval

// --- PESTELAnalysis ---

func (p PESTELAnalysis) Value() (driver.Value, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return string(b), nil // Must return string for JSONB compatibility with lib/pq
}

func (p *PESTELAnalysis) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for PESTELAnalysis")
	}
	return json.Unmarshal(b, p)
}

// --- PorterAnalysis ---

func (p PorterAnalysis) Value() (driver.Value, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (p *PorterAnalysis) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for PorterAnalysis")
	}
	return json.Unmarshal(b, p)
}

// --- SWOTAnalysis ---

func (s SWOTAnalysis) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (s *SWOTAnalysis) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for SWOTAnalysis")
	}
	return json.Unmarshal(b, s)
}

// --- BlueOceanAnalysis ---

func (b BlueOceanAnalysis) Value() (driver.Value, error) {
	bytes, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

func (b *BlueOceanAnalysis) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for BlueOceanAnalysis")
	}
	return json.Unmarshal(bytes, b)
}

// --- BalancedScorecardAnalysis ---

func (bsc BalancedScorecardAnalysis) Value() (driver.Value, error) {
	b, err := json.Marshal(bsc)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (bsc *BalancedScorecardAnalysis) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for BalancedScorecardAnalysis")
	}
	return json.Unmarshal(b, bsc)
}

// --- OKRAnalysis ---

func (o OKRAnalysis) Value() (driver.Value, error) {
	b, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (o *OKRAnalysis) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for OKRAnalysis")
	}
	return json.Unmarshal(b, o)
}

// --- BenchmarkingAnalysis ---

func (b BenchmarkingAnalysis) Value() (driver.Value, error) {
	bytes, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

func (b *BenchmarkingAnalysis) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for BenchmarkingAnalysis")
	}
	return json.Unmarshal(bytes, b)
}

// --- GrowthHackingAnalysis ---

func (g GrowthHackingAnalysis) Value() (driver.Value, error) {
	b, err := json.Marshal(g)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (g *GrowthHackingAnalysis) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for GrowthHackingAnalysis")
	}
	return json.Unmarshal(b, g)
}

// --- ScenarioAnalysis ---

func (s ScenarioAnalysis) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (s *ScenarioAnalysis) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for ScenarioAnalysis")
	}
	return json.Unmarshal(b, s)
}

// --- TamSamSomAnalysis ---

func (t TamSamSomAnalysis) Value() (driver.Value, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (t *TamSamSomAnalysis) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for TamSamSomAnalysis")
	}
	return json.Unmarshal(b, t)
}

// --- DecisionMatrixAnalysis ---

func (d DecisionMatrixAnalysis) Value() (driver.Value, error) {
	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (d *DecisionMatrixAnalysis) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for DecisionMatrixAnalysis")
	}
	return json.Unmarshal(b, d)
}

// --- AnalysisSynthesis ---

func (a AnalysisSynthesis) Value() (driver.Value, error) {
	b, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (a *AnalysisSynthesis) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for AnalysisSynthesis")
	}
	return json.Unmarshal(b, a)
}
