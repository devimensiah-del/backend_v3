package analysis

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Status constants for Analysis
type Status string

const (
	StatusPending   Status = "pending"   // Initial state, waiting for worker
	StatusCompleted Status = "completed" // Worker finished (version 1)
	StatusApproved  Status = "approved"  // Admin approved, PDF generated
	StatusSent      Status = "sent"      // Made available to user
	// Note: Removed StatusProcessing and StatusFailed
	// Failures keep status as "pending" with error_message populated
)

type Analysis struct {
	ID               string     `db:"id" json:"id"`
	SubmissionID     string     `db:"submission_id" json:"submission_id"`
	EnrichmentID     string     `db:"enrichment_id" json:"enrichment_id"`
	Status           string     `db:"status" json:"status"`
	ErrorMessage     *string    `db:"error_message" json:"error_message,omitempty"` // Error details if analysis failed
	ProcessingTimeMs int64      `db:"processing_time_ms" json:"processing_time_ms"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
	CompletedAt      *time.Time `db:"completed_at" json:"completed_at"`

	// Versioning fields
	Version          int     `db:"version" json:"version"`
	ParentAnalysisID *string `db:"parent_analysis_id" json:"parent_analysis_id,omitempty"`
	IsLatest         bool    `db:"is_latest" json:"is_latest"`

	// Approval and Send tracking
	ApprovedAt *time.Time `db:"approved_at" json:"approved_at,omitempty"`
	ApprovedBy *string    `db:"approved_by" json:"approved_by,omitempty"` // UUID of user who approved
	SentAt     *time.Time `db:"sent_at" json:"sent_at,omitempty"`
	SentTo     *string    `db:"sent_to" json:"sent_to,omitempty"` // Email address report was sent to

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
type QuarterlyOKR struct {
	Quarter    string   `json:"quarter"`     // "Q1 2025", "Q2 2025", "Q3 2025"
	Objective  string   `json:"objective"`   // Main objective description
	KeyResults []string `json:"key_results"` // KR1, KR2, KR3 (exactly 3)
	Investment string   `json:"investment"`  // Investment estimate (e.g., "R$ 25 mil" or range "R$ 20-30k")
	Timeline   string   `json:"timeline"`    // Duration (e.g., "3-4 meses")
}

type OKRAnalysis struct {
	Quarters []QuarterlyOKR `json:"quarters"` // Quarterly OKR structure (Q1, Q2, Q3)
	Summary  string         `json:"summary"`
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

	// Data Quality & Partial Data Support (for "Data Insufficient" scenarios)
	DataQuality        string   `json:"data_quality"`        // "complete" | "partial" | "insufficient"
	NextSteps          []string `json:"next_steps"`          // Steps to gather missing data
	ProxyIndicators    []string `json:"proxy_indicators"`    // Alternative metrics when data is insufficient
	ExpectedOutputs    []string `json:"expected_outputs"`    // What complete analysis should include
	MethodologicalNote string   `json:"methodological_note"` // Explanation of methodology or data limitations

	Summary string `json:"summary"`
}

// PriorityRecommendation represents a prioritized strategic recommendation
type PriorityRecommendation struct {
	Priority    int    `json:"priority"`    // 1, 2, 3 (ordered by priority)
	Title       string `json:"title"`       // Recommendation title
	Description string `json:"description"` // Detailed description
	Timeline    string `json:"timeline"`    // Expected duration (e.g., "9-12 meses")
	Budget      string `json:"budget"`      // Budget estimate (e.g., "R$150-250k")
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

func (a *Analysis) Start() {
	// Keep status as pending, just update timestamp
	a.UpdatedAt = time.Now()
}

func (a *Analysis) Complete() {
	a.Status = string(StatusCompleted)
	n := time.Now()
	a.CompletedAt = &n
	a.UpdatedAt = n
}

func (a *Analysis) Approve(approvedBy *string) {
	a.Status = string(StatusApproved)
	now := time.Now()
	a.ApprovedAt = &now
	a.ApprovedBy = approvedBy
	a.UpdatedAt = now
}

func (a *Analysis) Send(sentTo *string) {
	a.Status = string(StatusSent)
	now := time.Now()
	a.SentAt = &now
	a.SentTo = sentTo
	a.UpdatedAt = now
}

func (a *Analysis) Fail(errorMsg string) {
	// Keep status as pending, set error_message
	a.ErrorMessage = &errorMsg
	a.UpdatedAt = time.Now()
}

// CreateNewVersion creates a new version of this analysis
// Returns a new Analysis with incremented version and parent reference
func (a *Analysis) CreateNewVersion() *Analysis {
	newAnalysis := &Analysis{
		// Copy basic info
		SubmissionID:     a.SubmissionID,
		EnrichmentID:     a.EnrichmentID,
		Status:           a.Status, // Copy status from previous version per user requirement
		ProcessingTimeMs: 0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		CompletedAt:      nil,

		// Versioning
		Version:          a.Version + 1,
		ParentAnalysisID: &a.ID,
		IsLatest:         true,

		// Copy all framework data
		PESTEL:         a.PESTEL,
		Porter:         a.Porter,
		TamSamSom:      a.TamSamSom,
		SWOT:           a.SWOT,
		Benchmarking:   a.Benchmarking,
		BlueOcean:      a.BlueOcean,
		GrowthHacking:  a.GrowthHacking,
		Scenarios:      a.Scenarios,
		OKRs:           a.OKRs,
		BSC:            a.BSC,
		DecisionMatrix: a.DecisionMatrix,
		Synthesis:      a.Synthesis,
	}

	return newAnalysis
}

// ============================================
// JSONB Serialization for PostgreSQL
// ============================================
// Implement driver.Valuer and sql.Scanner for all framework types
// This enables PostgreSQL JSONB storage and retrieval

// --- PESTELAnalysis ---

func (p PESTELAnalysis) Value() (driver.Value, error) {
	return json.Marshal(p)
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
	return json.Marshal(p)
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
	return json.Marshal(s)
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
	return json.Marshal(b)
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
	return json.Marshal(bsc)
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
	return json.Marshal(o)
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
	return json.Marshal(b)
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
	return json.Marshal(g)
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
	return json.Marshal(s)
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
	return json.Marshal(t)
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
	return json.Marshal(d)
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
	return json.Marshal(a)
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
