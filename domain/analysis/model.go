package analysis

import (
	"time"
)

// Status constants for Analysis
type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusApproved   Status = "approved"
	StatusSent       Status = "sent"
)

type Analysis struct {
	ID               string     `db:"id" json:"id"`
	SubmissionID     string     `db:"submission_id" json:"submission_id"`
	EnrichmentID     string     `db:"enrichment_id" json:"enrichment_id"`
	Status           string     `db:"status" json:"status"`
	ProcessingTimeMs int64      `db:"processing_time_ms" json:"processing_time_ms"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
	CompletedAt      *time.Time `db:"completed_at" json:"completed_at"`

	// Versioning fields
	Version          int     `db:"version" json:"version"`
	ParentAnalysisID *string `db:"parent_analysis_id" json:"parent_analysis_id,omitempty"`

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
	CompetitiveRivalry    string `json:"competitive_rivalry"`
	SupplierPower         string `json:"supplier_power"`
	BuyerPower            string `json:"buyer_power"`
	ThreatNewEntrants     string `json:"threat_new_entrants"`
	ThreatSubstitutes     string `json:"threat_substitutes"`
	OverallAttractiveness string `json:"overall_attractiveness"`
	Summary               string `json:"summary"`
}

type SWOTAnalysis struct {
	Strengths     []string `json:"strengths"`
	Weaknesses    []string `json:"weaknesses"`
	Opportunities []string `json:"opportunities"`
	Threats       []string `json:"threats"`
	Summary       string   `json:"summary"`
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

type OKRAnalysis struct {
	Objectives []OKRObjective `json:"objectives"`
	Summary    string         `json:"summary"`
}

type OKRObjective struct {
	Title      string   `json:"title"`
	KeyResults []string `json:"key_results"`
}

type BenchmarkingAnalysis struct {
	Competitors     []string `json:"competitors_analyzed"`
	PerformanceGaps []string `json:"performance_gaps"` // Fixes validator error
	BestPractices   []string `json:"best_practices"`   // Fixes validator error
	Summary         string   `json:"summary"`
}

type GrowthHackingAnalysis struct {
	Hypotheses  []string `json:"hypotheses"`  // Fixes validator error
	Experiments []string `json:"experiments"` // Fixes validator error
	KeyMetrics  []string `json:"key_metrics"` // Fixes validator error
	Summary     string   `json:"summary"`
}

type ScenarioAnalysis struct {
	ScenarioOptimistic  string   `json:"scenario_optimistic"`
	ScenarioRealist     string   `json:"scenario_realist"`
	ScenarioPessimistic string   `json:"scenario_pessimistic"`
	EarlyWarningSignals []string `json:"early_warning_signals"` // Fixes validator error
	Summary             string   `json:"summary"`
}

type TamSamSomAnalysis struct {
	TAM         string   `json:"tam"`
	SAM         string   `json:"sam"`
	SOM         string   `json:"som"`
	Assumptions []string `json:"assumptions"`
	CAGR        string   `json:"cagr"`
	Summary     string   `json:"summary"`
}

type DecisionMatrixAnalysis struct {
	Alternatives        []string `json:"alternatives"`
	Criteria            []string `json:"criteria"`
	FinalRecommendation string   `json:"final_recommendation"`
	Summary             string   `json:"summary"`
}

type AnalysisSynthesis struct {
	ExecutiveSummary      string   `json:"executive_summary"`
	KeyFindings           []string `json:"key_findings"`
	StrategicPriorities   []string `json:"strategic_priorities"`
	Roadmap               []string `json:"roadmap"`
	OverallRecommendation string   `json:"overall_recommendation"`
}

// ContextContainer used during processing
type ContextContainer struct {
	SubmissionID   string
	EnrichmentData map[string]interface{}
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
	a.Status = string(StatusProcessing)
	a.UpdatedAt = time.Now()
}

func (a *Analysis) Complete() {
	a.Status = string(StatusCompleted)
	n := time.Now()
	a.CompletedAt = &n
	a.UpdatedAt = n
}

func (a *Analysis) Approve() {
	a.Status = string(StatusApproved)
	a.UpdatedAt = time.Now()
}

func (a *Analysis) Send() {
	a.Status = string(StatusSent)
	a.UpdatedAt = time.Now()
}

func (a *Analysis) Fail() {
	a.Status = string(StatusFailed)
	a.UpdatedAt = time.Now()
}

// CreateNewVersion creates a new version of this analysis
// Returns a new Analysis with incremented version and parent reference
func (a *Analysis) CreateNewVersion() *Analysis {
	newAnalysis := &Analysis{
		// Copy basic info
		SubmissionID:     a.SubmissionID,
		EnrichmentID:     a.EnrichmentID,
		Status:           string(StatusPending),
		ProcessingTimeMs: 0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		CompletedAt:      nil,

		// Versioning
		Version:          a.Version + 1,
		ParentAnalysisID: &a.ID,

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
