package analysis

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"backend_v3/config"
	"backend_v3/llm"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

// LLMClient defines the contract for the AI service.
type LLMClient interface {
	GenerateStructured(ctx context.Context, model string, prompt string, data interface{}, targetSchema interface{}) error
	GenerateStructuredWithOptions(ctx context.Context, opts llm.GenerationOptions, prompt string, data interface{}, targetSchema interface{}) error
}

// FrameworkService interface for dynamic framework loading
type FrameworkService interface {
	GetExecutionPlan(ctx context.Context, codes []string) ([]*Framework, error)
	GetByCode(ctx context.Context, code string) (*Framework, error)
	ListActive(ctx context.Context) ([]*Framework, error)
}

// Framework represents the framework model from domain/framework
type Framework struct {
	ID                 interface{}
	Code               string
	Name               string
	NamePT             string
	Description        *string
	DescriptionPT      *string
	Category           string
	LayerOrder         int
	IsActive           bool
	RequiresEnrichment bool
	TimeoutSeconds     int
	PromptTemplate     string
	OutputSchema       interface{}
	PreferredModel     *string
	Temperature        float64
	DependsOn          []string
	CreatedAt          interface{}
	UpdatedAt          interface{}
}

// MacroServiceInterface defines the contract for fetching macroeconomic data
// This interface is implemented by macroeconomics.Service (via adapter in main.go)
// Using an interface avoids import cycles between analysis and macroeconomics domains
type MacroServiceInterface interface {
	// GetLatestSnapshot retrieves the most recent values for ALL active indicators
	// Returns nil error if DB is empty (graceful degradation)
	GetLatestSnapshot(ctx context.Context) (*MacroSnapshot, error)
}

// MacroSnapshot represents latest economic indicators from DB
type MacroSnapshot struct {
	Indicators map[string]*MacroIndicator `json:"indicators"`
}

// MacroIndicator represents a single indicator with full metadata
type MacroIndicator struct {
	Code  string  `json:"code"`
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// HasData returns true if at least one indicator has data
func (s *MacroSnapshot) HasData() bool {
	return s != nil && len(s.Indicators) > 0
}

// SubmissionRepository defines the interface for accessing submission data
// We only need GetByID for fetching submission context
type SubmissionRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*SubmissionData, error)
}

// SubmissionData represents the essential submission fields needed for analysis
type SubmissionData struct {
	CompanyName       string
	CompanyWebsite    *string
	CompanyIndustry   *string
	CompanySize       *string
	CompanyLocation   *string
	BusinessChallenge string
	TargetMarket      *string
	AnnualRevenueMin  *float64
	AnnualRevenueMax  *float64
	FundingStage      *string
}

// ReportLookup defines the minimal dependency needed to verify PDF existence
type ReportLookup interface {
	GetBySubmissionID(ctx context.Context, submissionID string) (ReportSummary, error)
}

// ReportSummary captures only the fields analysis needs to validate sending
type ReportSummary interface {
	GetPDFURL() string
}

// Service handles all business analysis operations
type Service struct {
	repo             Repository
	submissionRepo   SubmissionRepository
	llm              LLMClient
	logger           zerolog.Logger
	queueClient      *asynq.Client // For job orchestration
	reportLookup     ReportLookup  // Optional: used to verify PDF before Send
	macroService     MacroServiceInterface // Optional: used to fetch macro data directly from DB
	frameworkService FrameworkService // Optional: used for dynamic framework execution

	// Framework configurations (4-model approach: presearch, enrichment, primary, synthesis)
	frameworks map[string]config.FrameworkConfig
}

// NewService creates a new analysis service instance
func NewService(
	repo Repository,
	submissionRepo SubmissionRepository,
	llm LLMClient,
	logger zerolog.Logger,
	queueClient *asynq.Client,
) *Service {
	return &Service{
		repo:           repo,
		submissionRepo: submissionRepo,
		llm:            llm,
		logger:         logger.With().Str("service", "analysis").Logger(),
		queueClient:    queueClient,
		frameworks:     make(map[string]config.FrameworkConfig), // Populated via SetFrameworks()
	}
}

// SetFrameworks updates the framework configurations (called by main.go)
func (s *Service) SetFrameworks(frameworks map[string]config.FrameworkConfig) {
	s.frameworks = frameworks
}

// SetReportLookup wires a report lookup dependency (used for PDF existence checks)
func (s *Service) SetReportLookup(lookup ReportLookup) {
	s.reportLookup = lookup
}

// SetMacroService wires a macro service dependency (used for fetching macro data directly from DB)
// This is preferred over extracting macro data from enrichment output
func (s *Service) SetMacroService(svc MacroServiceInterface) {
	s.macroService = svc
}

// SetFrameworkService wires a framework service dependency (used for dynamic framework execution)
// This enables RunAnalysisDynamic to load frameworks from database
func (s *Service) SetFrameworkService(fs FrameworkService) {
	s.frameworkService = fs
}

// GetByID retrieves an analysis by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Analysis, error) {
	return s.repo.GetByID(ctx, id)
}

// GetBySubmissionID retrieves an analysis by submission ID
func (s *Service) GetBySubmissionID(ctx context.Context, submissionID string) (*Analysis, error) {
	return s.repo.GetBySubmissionID(ctx, submissionID)
}

// ListAll retrieves all analyses with pagination
func (s *Service) ListAll(ctx context.Context, limit, offset int) ([]*Analysis, error) {
	return s.repo.List(ctx, limit, offset)
}

// Delete removes an analysis record
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// applyEditsToAnalysis applies edits from a map to the analysis structure
func (s *Service) applyEditsToAnalysis(analysis *Analysis, edits map[string]interface{}) {
	// Debug: Log what we received
	s.logger.Debug().
		Interface("edits_keys", getMapKeys(edits)).
		Msg("applyEditsToAnalysis received")

	// Helper to get framework from map and apply edits
	applyFrameworkEdits := func(code string, applyFunc func(interface{}, map[string]interface{})) {
		if frameworkEdits, ok := edits[code].(map[string]interface{}); ok {
			s.logger.Debug().Str("framework", code).Msg("Applying framework edits")

			// Get existing framework data
			var framework interface{}
			switch code {
			case "pestel":
				framework = new(PESTELAnalysis)
			case "porter":
				framework = new(PorterAnalysis)
			case "swot":
				framework = new(SWOTAnalysis)
			case "okrs":
				framework = new(OKRAnalysis)
			case "tam_sam_som":
				framework = new(TamSamSomAnalysis)
			case "benchmarking":
				framework = new(BenchmarkingAnalysis)
			case "blue_ocean":
				framework = new(BlueOceanAnalysis)
			case "growth_hacking":
				framework = new(GrowthHackingAnalysis)
			case "scenarios":
				framework = new(ScenarioAnalysis)
			case "bsc":
				framework = new(BalancedScorecardAnalysis)
			case "decision_matrix":
				framework = new(DecisionMatrixAnalysis)
			case "synthesis":
				framework = new(AnalysisSynthesis)
			default:
				s.logger.Warn().Str("framework", code).Msg("Unknown framework type")
				return
			}

			// Load existing data (ignore errors if not found)
			_ = analysis.GetFramework(code, framework)

			// Apply edits
			applyFunc(framework, frameworkEdits)

			// Store back
			if err := analysis.SetFramework(code, framework); err != nil {
				s.logger.Error().Err(err).Str("framework", code).Msg("Failed to set framework")
			}
		} else if _, exists := edits[code]; exists {
			s.logger.Warn().
				Str("framework", code).
				Str("actual_type", fmt.Sprintf("%T", edits[code])).
				Msg("Framework edits type assertion failed")
		}
	}

	// Apply edits to each framework
	applyFrameworkEdits("pestel", func(fw interface{}, edits map[string]interface{}) {
		s.applyPESTELEdits(fw.(*PESTELAnalysis), edits)
	})
	applyFrameworkEdits("porter", func(fw interface{}, edits map[string]interface{}) {
		s.applyPorterEdits(fw.(*PorterAnalysis), edits)
	})
	applyFrameworkEdits("swot", func(fw interface{}, edits map[string]interface{}) {
		s.applySWOTEdits(fw.(*SWOTAnalysis), edits)
	})
	applyFrameworkEdits("okrs", func(fw interface{}, edits map[string]interface{}) {
		s.applyOKRsEdits(fw.(*OKRAnalysis), edits)
	})
	applyFrameworkEdits("tam_sam_som", func(fw interface{}, edits map[string]interface{}) {
		s.applyTamSamSomEdits(fw.(*TamSamSomAnalysis), edits)
	})
	applyFrameworkEdits("benchmarking", func(fw interface{}, edits map[string]interface{}) {
		s.applyBenchmarkingEdits(fw.(*BenchmarkingAnalysis), edits)
	})
	applyFrameworkEdits("blue_ocean", func(fw interface{}, edits map[string]interface{}) {
		s.applyBlueOceanEdits(fw.(*BlueOceanAnalysis), edits)
	})
	applyFrameworkEdits("growth_hacking", func(fw interface{}, edits map[string]interface{}) {
		s.applyGrowthHackingEdits(fw.(*GrowthHackingAnalysis), edits)
	})
	applyFrameworkEdits("scenarios", func(fw interface{}, edits map[string]interface{}) {
		s.applyScenariosEdits(fw.(*ScenarioAnalysis), edits)
	})
	applyFrameworkEdits("bsc", func(fw interface{}, edits map[string]interface{}) {
		s.applyBSCEdits(fw.(*BalancedScorecardAnalysis), edits)
	})
	applyFrameworkEdits("decision_matrix", func(fw interface{}, edits map[string]interface{}) {
		s.applyDecisionMatrixEdits(fw.(*DecisionMatrixAnalysis), edits)
	})
	applyFrameworkEdits("synthesis", func(fw interface{}, edits map[string]interface{}) {
		s.applySynthesisEdits(fw.(*AnalysisSynthesis), edits)
	})
}

// Helper to get map keys for debugging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Helper methods to apply edits to specific frameworks
func (s *Service) applyPESTELEdits(pestel *PESTELAnalysis, edits map[string]interface{}) {
	if political, ok := edits["political"].([]interface{}); ok {
		pestel.Political = interfaceSliceToStringSlice(political)
	}
	if economic, ok := edits["economic"].([]interface{}); ok {
		pestel.Economic = interfaceSliceToStringSlice(economic)
	}
	if social, ok := edits["social"].([]interface{}); ok {
		pestel.Social = interfaceSliceToStringSlice(social)
	}
	if technological, ok := edits["technological"].([]interface{}); ok {
		pestel.Technological = interfaceSliceToStringSlice(technological)
	}
	if environmental, ok := edits["environmental"].([]interface{}); ok {
		pestel.Environmental = interfaceSliceToStringSlice(environmental)
	}
	if legal, ok := edits["legal"].([]interface{}); ok {
		pestel.Legal = interfaceSliceToStringSlice(legal)
	}
	if summary, ok := edits["summary"].(string); ok {
		pestel.Summary = summary
	}
}

func (s *Service) applyPorterEdits(porter *PorterAnalysis, edits map[string]interface{}) {
	if competitiveRivalry, ok := edits["competitive_rivalry"].(string); ok {
		porter.CompetitiveRivalry = competitiveRivalry
	}
	if supplierPower, ok := edits["supplier_power"].(string); ok {
		porter.SupplierPower = supplierPower
	}
	if buyerPower, ok := edits["buyer_power"].(string); ok {
		porter.BuyerPower = buyerPower
	}
	if threatNewEntrants, ok := edits["threat_new_entrants"].(string); ok {
		porter.ThreatNewEntrants = threatNewEntrants
	}
	if threatSubstitutes, ok := edits["threat_substitutes"].(string); ok {
		porter.ThreatSubstitutes = threatSubstitutes
	}
	if overallAttractiveness, ok := edits["overall_attractiveness"].(string); ok {
		porter.OverallAttractiveness = overallAttractiveness
	}
	if summary, ok := edits["summary"].(string); ok {
		porter.Summary = summary
	}
}

func (s *Service) applySWOTEdits(swot *SWOTAnalysis, edits map[string]interface{}) {
	if strengths, ok := edits["strengths"].([]interface{}); ok {
		swot.Strengths = interfaceSliceToSWOTItemSlice(strengths)
	}
	if weaknesses, ok := edits["weaknesses"].([]interface{}); ok {
		swot.Weaknesses = interfaceSliceToSWOTItemSlice(weaknesses)
	}
	if opportunities, ok := edits["opportunities"].([]interface{}); ok {
		swot.Opportunities = interfaceSliceToSWOTItemSlice(opportunities)
	}
	if threats, ok := edits["threats"].([]interface{}); ok {
		swot.Threats = interfaceSliceToSWOTItemSlice(threats)
	}
	if summary, ok := edits["summary"].(string); ok {
		swot.Summary = summary
	}
}

func (s *Service) applyOKRsEdits(okrs *OKRAnalysis, edits map[string]interface{}) {
	if summary, ok := edits["summary"].(string); ok {
		okrs.Summary = summary
	}
	if totalInvestment, ok := edits["total_investment"].(string); ok {
		okrs.TotalInvestment = totalInvestment
	}
	if successMetrics, ok := edits["success_metrics"].([]interface{}); ok {
		okrs.SuccessMetrics = interfaceSliceToStringSlice(successMetrics)
	}
	// Handle plan_90_days array
	if plan90Days, ok := edits["plan_90_days"].([]interface{}); ok {
		okrs.Plan90Days = interfaceSliceToMonthlyOKRSlice(plan90Days)
	}
}

func (s *Service) applyTamSamSomEdits(tam *TamSamSomAnalysis, edits map[string]interface{}) {
	if tamValue, ok := edits["tam"].(string); ok {
		tam.TAM = tamValue
	}
	if sam, ok := edits["sam"].(string); ok {
		tam.SAM = sam
	}
	if som, ok := edits["som"].(string); ok {
		tam.SOM = som
	}
	if assumptions, ok := edits["assumptions"].([]interface{}); ok {
		tam.Assumptions = interfaceSliceToStringSlice(assumptions)
	}
	if cagr, ok := edits["cagr"].(string); ok {
		tam.CAGR = cagr
	}
	if confidenceLevel, ok := edits["confidence_level"].(float64); ok {
		tam.ConfidenceLevel = int(confidenceLevel)
	}
	if estimationMethod, ok := edits["estimation_method"].(string); ok {
		tam.EstimationMethod = estimationMethod
	}
	if summary, ok := edits["summary"].(string); ok {
		tam.Summary = summary
	}
}

func (s *Service) applyBenchmarkingEdits(bench *BenchmarkingAnalysis, edits map[string]interface{}) {
	if competitors, ok := edits["competitors_analyzed"].([]interface{}); ok {
		bench.Competitors = interfaceSliceToStringSlice(competitors)
	}
	if performanceGaps, ok := edits["performance_gaps"].([]interface{}); ok {
		bench.PerformanceGaps = interfaceSliceToStringSlice(performanceGaps)
	}
	if bestPractices, ok := edits["best_practices"].([]interface{}); ok {
		bench.BestPractices = interfaceSliceToStringSlice(bestPractices)
	}
	if summary, ok := edits["summary"].(string); ok {
		bench.Summary = summary
	}
}

func (s *Service) applyBlueOceanEdits(blueOcean *BlueOceanAnalysis, edits map[string]interface{}) {
	if eliminate, ok := edits["eliminate"].([]interface{}); ok {
		blueOcean.Eliminate = interfaceSliceToStringSlice(eliminate)
	}
	if reduce, ok := edits["reduce"].([]interface{}); ok {
		blueOcean.Reduce = interfaceSliceToStringSlice(reduce)
	}
	if raise, ok := edits["raise"].([]interface{}); ok {
		blueOcean.Raise = interfaceSliceToStringSlice(raise)
	}
	if create, ok := edits["create"].([]interface{}); ok {
		blueOcean.Create = interfaceSliceToStringSlice(create)
	}
	if newValueCurve, ok := edits["new_value_curve"].(string); ok {
		blueOcean.NewValueCurve = newValueCurve
	}
	if summary, ok := edits["summary"].(string); ok {
		blueOcean.Summary = summary
	}
}

func (s *Service) applyGrowthHackingEdits(gh *GrowthHackingAnalysis, edits map[string]interface{}) {
	if leapLoop, ok := edits["leap_loop"].(map[string]interface{}); ok {
		applyGrowthLoopEdits(&gh.LeapLoop, leapLoop)
	}
	if scaleLoop, ok := edits["scale_loop"].(map[string]interface{}); ok {
		applyGrowthLoopEdits(&gh.ScaleLoop, scaleLoop)
	}
	if summary, ok := edits["summary"].(string); ok {
		gh.Summary = summary
	}
}

func applyGrowthLoopEdits(loop *GrowthLoop, edits map[string]interface{}) {
	if name, ok := edits["name"].(string); ok {
		loop.Name = name
	}
	if loopType, ok := edits["type"].(string); ok {
		loop.Type = loopType
	}
	if steps, ok := edits["steps"].([]interface{}); ok {
		loop.Steps = interfaceSliceToStringSlice(steps)
	}
	if metrics, ok := edits["metrics"].([]interface{}); ok {
		loop.Metrics = interfaceSliceToStringSlice(metrics)
	}
	if bottleneck, ok := edits["bottleneck"].(string); ok {
		loop.Bottleneck = bottleneck
	}
}

func (s *Service) applyScenariosEdits(scenarios *ScenarioAnalysis, edits map[string]interface{}) {
	if optimistic, ok := edits["optimistic"].(map[string]interface{}); ok {
		applyScenarioEdits(&scenarios.Optimistic, optimistic)
	}
	if realist, ok := edits["realist"].(map[string]interface{}); ok {
		applyScenarioEdits(&scenarios.Realist, realist)
	}
	if pessimistic, ok := edits["pessimistic"].(map[string]interface{}); ok {
		applyScenarioEdits(&scenarios.Pessimistic, pessimistic)
	}
	if mitigationTactics, ok := edits["mitigation_tactics"].([]interface{}); ok {
		scenarios.MitigationTactics = interfaceSliceToStringSlice(mitigationTactics)
	}
	if earlyWarningSignals, ok := edits["early_warning_signals"].([]interface{}); ok {
		scenarios.EarlyWarningSignals = interfaceSliceToStringSlice(earlyWarningSignals)
	}
	if summary, ok := edits["summary"].(string); ok {
		scenarios.Summary = summary
	}
}

func applyScenarioEdits(scenario *Scenario, edits map[string]interface{}) {
	if name, ok := edits["name"].(string); ok {
		scenario.Name = name
	}
	if probability, ok := edits["probability"].(float64); ok {
		scenario.Probability = int(probability)
	}
	if description, ok := edits["description"].(string); ok {
		scenario.Description = description
	}
	if requiredActions, ok := edits["required_actions"].([]interface{}); ok {
		scenario.RequiredActions = interfaceSliceToStringSlice(requiredActions)
	}
}

func (s *Service) applyBSCEdits(bsc *BalancedScorecardAnalysis, edits map[string]interface{}) {
	if financial, ok := edits["financial"].([]interface{}); ok {
		bsc.Financial = interfaceSliceToStringSlice(financial)
	}
	if customer, ok := edits["customer"].([]interface{}); ok {
		bsc.Customer = interfaceSliceToStringSlice(customer)
	}
	if internal, ok := edits["internal_processes"].([]interface{}); ok {
		bsc.Internal = interfaceSliceToStringSlice(internal)
	}
	if learningGrowth, ok := edits["learning_growth"].([]interface{}); ok {
		bsc.LearningGrowth = interfaceSliceToStringSlice(learningGrowth)
	}
	if summary, ok := edits["summary"].(string); ok {
		bsc.Summary = summary
	}
}

func (s *Service) applyDecisionMatrixEdits(dm *DecisionMatrixAnalysis, edits map[string]interface{}) {
	if alternatives, ok := edits["alternatives"].([]interface{}); ok {
		dm.Alternatives = interfaceSliceToStringSlice(alternatives)
	}
	if criteria, ok := edits["criteria"].([]interface{}); ok {
		dm.Criteria = interfaceSliceToStringSlice(criteria)
	}
	if finalRecommendation, ok := edits["final_recommendation"].(string); ok {
		dm.FinalRecommendation = finalRecommendation
	}
	if recommendedOption, ok := edits["recommended_option"].(string); ok {
		dm.RecommendedOption = recommendedOption
	}
	if score, ok := edits["score"].(string); ok {
		dm.Score = score
	}
	if scoreComparison, ok := edits["score_comparison"].(string); ok {
		dm.ScoreComparison = scoreComparison
	}
	if priorityRecommendations, ok := edits["priority_recommendations"].([]interface{}); ok {
		dm.PriorityRecommendations = interfaceSliceToPriorityRecommendationSlice(priorityRecommendations)
	}
	if monitoringMetrics, ok := edits["monitoring_metrics"].([]interface{}); ok {
		dm.MonitoringMetrics = interfaceSliceToStringSlice(monitoringMetrics)
	}
	if summary, ok := edits["summary"].(string); ok {
		dm.Summary = summary
	}
}

func (s *Service) applySynthesisEdits(synthesis *AnalysisSynthesis, edits map[string]interface{}) {
	if executiveSummary, ok := edits["executive_summary"].(string); ok {
		synthesis.ExecutiveSummary = executiveSummary
	}
	if centralChallenge, ok := edits["central_challenge"].(string); ok {
		synthesis.CentralChallenge = centralChallenge
	}
	if mainFindings, ok := edits["main_findings"].([]interface{}); ok {
		synthesis.MainFindings = interfaceSliceToStringSlice(mainFindings)
	}
	if importantNotes, ok := edits["important_notes"].([]interface{}); ok {
		synthesis.ImportantNotes = interfaceSliceToStringSlice(importantNotes)
	}
	if keyFindings, ok := edits["key_findings"].([]interface{}); ok {
		synthesis.KeyFindings = interfaceSliceToStringSlice(keyFindings)
	}
	if strategicPriorities, ok := edits["strategic_priorities"].([]interface{}); ok {
		synthesis.StrategicPriorities = interfaceSliceToStringSlice(strategicPriorities)
	}
	if roadmap, ok := edits["roadmap"].([]interface{}); ok {
		synthesis.Roadmap = interfaceSliceToStringSlice(roadmap)
	}
	if overallRecommendation, ok := edits["overall_recommendation"].(string); ok {
		synthesis.OverallRecommendation = overallRecommendation
	}
}

// Helper function to convert []interface{} to []string
func interfaceSliceToStringSlice(slice []interface{}) []string {
	result := make([]string, len(slice))
	for i, v := range slice {
		if str, ok := v.(string); ok {
			result[i] = str
		}
	}
	return result
}

// Helper function to convert []interface{} to []SWOTItem
// Handles both old format ([]string) and new format ([]SWOTItem with confidence/source)
func interfaceSliceToSWOTItemSlice(slice []interface{}) []SWOTItem {
	result := make([]SWOTItem, 0, len(slice))
	for _, v := range slice {
		// Check if it's the new format (map with content, confidence, source)
		if itemMap, ok := v.(map[string]interface{}); ok {
			item := SWOTItem{}
			if content, ok := itemMap["content"].(string); ok {
				item.Content = content
			}
			if confidence, ok := itemMap["confidence"].(string); ok {
				item.Confidence = confidence
			}
			if source, ok := itemMap["source"].(string); ok {
				item.Source = source
			}
			result = append(result, item)
		} else if str, ok := v.(string); ok {
			// Backward compatibility: old format was just strings
			result = append(result, SWOTItem{
				Content:    str,
				Confidence: "Média",              // Default confidence for legacy data
				Source:     "análise de mercado", // Default source
			})
		}
	}
	return result
}

// Helper function to convert []interface{} to []MonthlyOKR
func interfaceSliceToMonthlyOKRSlice(slice []interface{}) []MonthlyOKR {
	result := make([]MonthlyOKR, 0, len(slice))
	for _, v := range slice {
		if itemMap, ok := v.(map[string]interface{}); ok {
			item := MonthlyOKR{}
			if month, ok := itemMap["month"].(string); ok {
				item.Month = month
			}
			if focus, ok := itemMap["focus"].(string); ok {
				item.Focus = focus
			}
			if objective, ok := itemMap["objective"].(string); ok {
				item.Objective = objective
			}
			if keyResults, ok := itemMap["key_results"].([]interface{}); ok {
				item.KeyResults = interfaceSliceToStringSlice(keyResults)
			}
			if investment, ok := itemMap["investment"].(string); ok {
				item.Investment = investment
			}
			if alignedRec, ok := itemMap["aligned_recommendation"].(string); ok {
				item.AlignedRecommendation = alignedRec
			}
			result = append(result, item)
		}
	}
	return result
}

// Helper function to convert []interface{} to []PriorityRecommendation
func interfaceSliceToPriorityRecommendationSlice(slice []interface{}) []PriorityRecommendation {
	result := make([]PriorityRecommendation, 0, len(slice))
	for _, v := range slice {
		if itemMap, ok := v.(map[string]interface{}); ok {
			item := PriorityRecommendation{}
			if priority, ok := itemMap["priority"].(float64); ok {
				item.Priority = int(priority)
			}
			if title, ok := itemMap["title"].(string); ok {
				item.Title = title
			}
			if description, ok := itemMap["description"].(string); ok {
				item.Description = description
			}
			if timeline, ok := itemMap["timeline"].(string); ok {
				item.Timeline = timeline
			}
			if budget, ok := itemMap["budget"].(string); ok {
				item.Budget = budget
			}
			result = append(result, item)
		}
	}
	return result
}

// generateAnalysisID generates a new UUID for analysis
func generateAnalysisID() string {
	return uuid.New().String()
}

// UpdateFields updates analysis framework fields (admin edit)
// Status remains unchanged
// IMPORTANT: Admin can only edit when status is "completed" (AI finished processing)
func (s *Service) UpdateFields(ctx context.Context, analysisID string, updateData map[string]interface{}) (*Analysis, error) {
	// Get current analysis
	currentAnalysis, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return nil, err
	}

	// Only allow edits when AI has finished (status = "completed")
	// Block edits during: pending (AI processing)
	if currentAnalysis.Status == string(StatusPending) || currentAnalysis.Status == string(StatusProcessing) {
		return nil, fmt.Errorf("cannot edit analysis while AI is still processing")
	}

	// Apply edits to analysis
	s.applyEditsToAnalysis(currentAnalysis, updateData)

	// Update via repository
	if err := s.repo.Update(ctx, currentAnalysis); err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Msg("Updated analysis fields")

	return currentAnalysis, nil
}

// SetVisibility toggles user visibility for an analysis
// Admin can make a completed analysis visible to the user
func (s *Service) SetVisibility(ctx context.Context, analysisID string, visible bool) error {
	// Get analysis first to validate state
	analysis, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return err
	}

	// Only allow visibility toggle on completed analyses
	if analysis.Status != string(StatusCompleted) {
		return fmt.Errorf("analysis must be completed before toggling visibility, current status: %s", analysis.Status)
	}

	if err := s.repo.SetVisibility(ctx, analysisID, visible); err != nil {
		return err
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Bool("visible", visible).
		Msg("Analysis visibility updated")

	return nil
}

// SetBlurStatus toggles the blur overlay for premium frameworks
// This is independent of visibility - an analysis can be visible but blurred (paywall)
// Admin can unblur to give full access to premium content
func (s *Service) SetBlurStatus(ctx context.Context, analysisID string, blurred bool) error {
	// Get analysis first to validate it exists
	_, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return err
	}

	// No status restriction - blur can be toggled at any time
	// (it's a display control, not a workflow state)

	if err := s.repo.SetBlurStatus(ctx, analysisID, blurred); err != nil {
		return err
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Bool("blurred", blurred).
		Msg("Analysis blur status updated")

	return nil
}

// SetPublicStatus toggles whether the analysis is accessible without login
// When true, anyone with the access code can view the report
// When false, the access code requires authentication
func (s *Service) SetPublicStatus(ctx context.Context, analysisID string, public bool) error {
	// Get analysis first to validate it exists
	_, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return err
	}

	// No status restriction - public flag can be toggled at any time
	// (it's an access control, not a workflow state)

	if err := s.repo.SetPublicStatus(ctx, analysisID, public); err != nil {
		return err
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Bool("public", public).
		Msg("Analysis public status updated")

	return nil
}

// MarkAsFailed updates analysis with error message
// Called by worker ErrorHandler after Asynq exhausts max retries
// Status remains "pending" with error_message populated
func (s *Service) MarkAsFailed(ctx context.Context, submissionID string, errorMsg string) error {
	analysis, err := s.repo.GetBySubmissionID(ctx, submissionID)
	if err != nil {
		return fmt.Errorf("failed to get analysis: %w", err)
	}

	// Set error message (status stays "pending")
	analysis.ErrorMessage = &errorMsg

	if err := s.repo.Update(ctx, analysis); err != nil {
		return fmt.Errorf("failed to update analysis error message: %w", err)
	}

	s.logger.Error().
		Str("analysis_id", analysis.ID).
		Str("submission_id", submissionID).
		Str("error", errorMsg).
		Msg("Analysis marked with error message")

	return nil
}

// GetByAccessCode retrieves an analysis by its public access code
// Returns nil, nil if not found (for 404 handling)
func (s *Service) GetByAccessCode(ctx context.Context, code string) (*Analysis, error) {
	return s.repo.GetByAccessCode(ctx, code)
}

// GenerateAccessCode creates a unique 8-character access code for public sharing
// Handles collisions by regenerating up to 5 times
func (s *Service) GenerateAccessCode(ctx context.Context, analysisID string) (string, error) {
	// Verify analysis exists and is completed
	analysis, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return "", err
	}

	// Only allow access code generation for completed analyses
	if analysis.Status != string(StatusCompleted) {
		return "", fmt.Errorf("analysis must be completed before generating access code, current status: %s", analysis.Status)
	}

	const maxRetries = 5
	for i := 0; i < maxRetries; i++ {
		code := generateSecureCode(8)
		exists, err := s.repo.AccessCodeExists(ctx, code)
		if err != nil {
			return "", fmt.Errorf("failed to check code existence: %w", err)
		}
		if !exists {
			err = s.repo.SetAccessCode(ctx, analysisID, code)
			if err != nil {
				return "", fmt.Errorf("failed to set access code: %w", err)
			}

			s.logger.Info().
				Str("analysis_id", analysisID).
				Str("access_code", code).
				Msg("Generated access code for analysis")

			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique code after %d attempts", maxRetries)
}

// generateSecureCode generates a cryptographically secure alphanumeric code
// Uses only uppercase letters and digits (36 characters) for URL-safety
func generateSecureCode(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[n.Int64()]
	}

	return string(result)
}
