package analysis_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"backend_v3/config"
	"backend_v3/domain/analysis"
	"backend_v3/llm"

	"github.com/jmoiron/sqlx"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ----------------------------------------------------------------------------
// CONFIGURATION
// ----------------------------------------------------------------------------

// TODO: Replace this with your real key to test against the real API.
// If left invalid, the test will automatically mock the responses.
const OpenRouterKey = ""

const TestModel = "google/gemini-2.0-flash-001"

// ----------------------------------------------------------------------------
// DATA SETUP (From your previous test results)
// ----------------------------------------------------------------------------

var coimmaSubmission = &analysis.SubmissionData{
	CompanyName:       "Coimma",
	CompanyWebsite:    stringPtr("https://www.coimma.com.br"),
	BusinessChallenge: "Sou CEO da Coimma. Somos fabricantes de equipamentos de manejo para gado, balanças bovinas eletronica, balanças rodoviárias, automação de balancas rodoviárias e balanças de fluxo por batelada para grãos. Atualmente 70% do nosso faturamento é proveniente do mercado pecuária. Os outros 30% sao provenientes de segmentos diversos destacando-se grãos e armazenamento. A prestação de serviços de calibração e manutenção representa 5% do faturamento.Nossas competências são fabricação de equipamentos, sistemas de pesagem e eletrônica elabore um plano de de negócios para nossa companhia em duas fases. A primeira fase mais imediata visa expandir as nossas vendas com nosso portfólio atual de produtos e aproveitando nossa estrutura atual de vendas e assistência técnica. Na segunda fase queremos desenvolver novos produtos para o segmento industrial que requer investimentos em estrutura de vendas e assistência técnica.",
}

var coimmaEnrichmentJSON = `{
  "profile_overview": {
    "legal_name": "Coimma Industria e Comercio Ltda",
    "website": "https://www.coimma.com.br",
    "foundation_year": "N/A",
    "headquarters": "Dracena, São Paulo, Brasil"
  },
  "market_position": {
    "sector": "Fabricação de equipamentos para pecuária, sistemas de pesagem e automação industrial",
    "target_audience": "Pecuaristas, produtores de grãos, indústrias de alimentos e bebidas, empresas de logística e transporte.",
    "value_proposition": "Equipamentos de alta qualidade e precisão para manejo de gado, pesagem e automação, com foco em aumentar a eficiência e produtividade dos clientes."
  },
  "financials": {
    "employees_range": "50-200",
    "revenue_estimate": "R$ 10M - R$ 30M/ano (estimated)",
    "business_model": "B2B, Venda de equipamentos, serviços de manutenção e calibração."
  },
  "competitive_landscape": {
    "competitors": [
      "Beckhauser",
      "Balanças Açores",
      "Ramtec"
    ],
    "market_share_status": "Desafiador"
  },
  "strategic_assessment": {
    "digital_maturity": 5,
    "strengths": [
      "Forte presença no mercado pecuário",
      "Experiência em sistemas de pesagem",
      "Estrutura de vendas e assistência técnica estabelecida"
    ],
    "weaknesses": [
      "Dependência do mercado pecuário",
      "Necessidade de diversificação para o setor industrial",
      "Adoção de novas tecnologias pode ser lenta"
    ]
  },
  "macro_context": {
    "economic_indicators": {
      "country": "Brasil",
      "gdp_growth": "+2.0% (2025 previsão)",
      "inflation_rate": "IPCA: 4.5% a.a.",
      "interest_rate": "SELIC: 10.5%",
      "exchange_rate": "USD/BRL: 5.05",
      "unemployment_rate": "7.8%",
      "political_stability": "Moderada, com reformas em discussão",
      "economic_outlook": "Crescimento gradual com inflação controlada",
      "recent_policy_changes": [
        "Reforma tributária em andamento",
        "Incentivos para o agronegócio"
      ]
    },
    "industry_trends": {
      "industry_sector": "Agronegócio e Automação Industrial",
      "growth_rate": "+8% CAGR (2024-2028) para agronegócio, +6% para automação industrial",
      "key_trends": [
        "Automação de processos",
        "Agricultura de precisão",
        "Rastreabilidade e segurança alimentar",
        "IoT e conectividade"
      ],
      "technology_adoption": "Crescente adoção de tecnologias digitais no agronegócio e na indústria",
      "market_concentration": "Moderadamente concentrado, com alguns players dominantes",
      "barriers_to_entry": "Médias, exigindo conhecimento técnico e investimento em P&D",
      "mergers_acquisitions": [
        "Aquisições de startups de tecnologia por grandes empresas do setor",
        "Consolidação de empresas de automação industrial"
      ]
    },
    "regulatory_landscape": {
      "recent_regulations": [
        "Novas normas para rastreabilidade de produtos agropecuários",
        "Regulamentação da Lei Geral de Proteção de Dados (LGPD)"
      ],
      "upcoming_changes": [
        "Discussão sobre novas normas ambientais para o setor agropecuário",
        "Atualização das normas de segurança do trabalho na indústria"
      ],
      "compliance_requirements": "Certificações ISO 9001, ISO 14001, Boas Práticas de Fabricação (BPF)",
      "industry_standards": [
        "Normas ABNT para equipamentos industriais",
        "Padrões de segurança alimentar"
      ]
    },
    "market_signals": {
      "commodity_prices": [
        "Preços de grãos em alta devido à demanda global",
        "Aço com preços voláteis devido a questões geopolíticas"
      ],
      "supply_chain_status": "Cadeias de suprimentos ainda enfrentam desafios devido à pandemia e conflitos internacionais",
      "consumer_sentiment": "Consumidores mais exigentes em relação à qualidade e segurança dos alimentos",
      "competitor_activity": [
        "Lançamento de novos produtos com tecnologias embarcadas",
        "Expansão de empresas para mercados internacionais"
      ],
      "emerging_threats": [
        "Aumento da concorrência de empresas estrangeiras",
        "Riscos climáticos para a produção agropecuária"
      ]
    },
    "data_sources": [
      "https://www.coimma.com.br",
      "https://www.abimaq.org.br/",
      "https://www.canalrural.com.br/"
    ],
    "last_updated": "2024-11-22"
  }
}`

// ----------------------------------------------------------------------------
// TEST
// ----------------------------------------------------------------------------

func TestAnalysisWorkflow_Integration(t *testing.T) {
	// 1. Setup Logger
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	// 2. Setup Mocks
	mockRepo := new(MockRepository)
	mockSubRepo := new(MockSubmissionRepo)

	// Initialize the Resilient Client (Real calls -> Fallback to Mock)
	resilientLLM := NewResilientLLM(OpenRouterKey, logger)

	// Create asynq client for job orchestration (dummy Redis address for testing)
	// The test doesn't actually call Approve() so this won't be used
	queueClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr: "localhost:6379",
	})
	defer queueClient.Close()

	// 3. Setup Service
	service := analysis.NewService(
		mockRepo,
		mockSubRepo,
		resilientLLM,
		logger,
		queueClient,
		"", "", // Deprecated fields
	)

	// 4. Configure Frameworks
	frameworks := make(map[string]config.FrameworkConfig)
	frameworkList := []string{"pestel", "porter", "tam_sam_som", "swot", "benchmarking", "blue_ocean", "growth_hacking", "scenarios", "okrs", "bsc", "decision_matrix", "synthesis"}
	for _, f := range frameworkList {
		frameworks[f] = config.FrameworkConfig{
			Model:       TestModel,
			Temperature: 0.4,
			MaxTokens:   2000,
		}
	}
	service.SetFrameworks(frameworks)

	// 5. Prepare Test Data
	subID := uuid.New()
	enrichID := uuid.New().String()

	// Parse enrichment JSON into map
	var enrichmentData map[string]interface{}
	err := json.Unmarshal([]byte(coimmaEnrichmentJSON), &enrichmentData)
	assert.NoError(t, err)

	// Expectation: Submission Repo returns Coimma data
	mockSubRepo.On("GetByID", mock.Anything, subID).Return(coimmaSubmission, nil)

	// Expectation: Repo creates and updates analysis
	// We use mock.Anything because the ID is generated inside the service
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*analysis.Analysis")).Return(nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*analysis.Analysis")).Return(nil)

	// 6. EXECUTE
	logger.Info().Msg("🚀 Starting Analysis Integration Test...")
	result, err := service.RunAnalysis(context.Background(), subID.String(), enrichID, enrichmentData)

	// 7. VALIDATION
	assert.NoError(t, err)
	assert.NotNil(t, result)

	logger.Info().Msgf("✅ Analysis Completed with Status: %s", result.Status)

	// Validation: Check if key frameworks were populated
	assert.Equal(t, "completed", result.Status)
	assert.NotEmpty(t, result.PESTEL.Summary, "PESTEL should have a summary")
	assert.NotEmpty(t, result.SWOT.Strengths, "SWOT should have strengths")
	assert.NotEmpty(t, result.Synthesis.ExecutiveSummary, "Synthesis should be generated")

	// NEW: Validate macro-context is being used in analysis
	logger.Info().Msg("\n🔍 Validating Macro-Context Usage...")

	// Check PESTEL for specific macro-context references
	pestelOutput := strings.ToLower(strings.Join(result.PESTEL.Economic, " "))
	hasSELIC := strings.Contains(pestelOutput, "selic") || strings.Contains(pestelOutput, "10.5") || strings.Contains(pestelOutput, "10,5")
	hasGDP := strings.Contains(pestelOutput, "pib") || strings.Contains(pestelOutput, "2.0") || strings.Contains(pestelOutput, "2,0")
	hasInflation := strings.Contains(pestelOutput, "inflação") || strings.Contains(pestelOutput, "ipca") || strings.Contains(pestelOutput, "4.5") || strings.Contains(pestelOutput, "4,5")

	if hasSELIC || hasGDP || hasInflation {
		logger.Info().Msgf("✅ PESTEL Economic factors cite macro-context data (SELIC: %v, GDP: %v, Inflation: %v)", hasSELIC, hasGDP, hasInflation)
	} else {
		logger.Warn().Msg("⚠️  PESTEL Economic factors may not be using macro-context data")
	}

	// Check TAM-SAM-SOM for growth rate reference
	tamOutput := strings.ToLower(result.TamSamSom.CAGR + " " + strings.Join(result.TamSamSom.Assumptions, " "))
	hasCAGR := strings.Contains(tamOutput, "8%") || strings.Contains(tamOutput, "6%") || strings.Contains(tamOutput, "cagr")

	if hasCAGR {
		logger.Info().Msg("✅ TAM-SAM-SOM references industry growth rates from macro-context")
	} else {
		logger.Warn().Msg("⚠️  TAM-SAM-SOM may not be using macro-context growth rates")
	}

	// Print the COMPLETE analysis output
	printCompleteAnalysis(result, logger)
}

// printCompleteAnalysis prints all 11 frameworks and synthesis in formatted output
func printCompleteAnalysis(result *analysis.Analysis, logger zerolog.Logger) {
	logger.Info().Msg("\n" + strings.Repeat("=", 100))
	logger.Info().Msg("📊 COMPLETE STRATEGIC CASCADE ANALYSIS - COIMMA")
	logger.Info().Msg(strings.Repeat("=", 100))

	// LAYER 1: ENVIRONMENT
	logger.Info().Msg("\n🌍 LAYER 1: THE ENVIRONMENT (Macro + Industry)")
	logger.Info().Msg(strings.Repeat("-", 100))

	logger.Info().Msg("\n1️⃣ PESTEL ANALYSIS")
	logger.Info().Msgf("Political: %v", result.PESTEL.Political)
	logger.Info().Msgf("Economic: %v", result.PESTEL.Economic)
	logger.Info().Msgf("Social: %v", result.PESTEL.Social)
	logger.Info().Msgf("Technological: %v", result.PESTEL.Technological)
	logger.Info().Msgf("Environmental: %v", result.PESTEL.Environmental)
	logger.Info().Msgf("Legal: %v", result.PESTEL.Legal)
	logger.Info().Msgf("📝 Summary: %s", result.PESTEL.Summary)

	logger.Info().Msg("\n2️⃣ PORTER'S FIVE FORCES")
	logger.Info().Msgf("Competitive Rivalry: %s", result.Porter.CompetitiveRivalry)
	logger.Info().Msgf("Supplier Power: %s", result.Porter.SupplierPower)
	logger.Info().Msgf("Buyer Power: %s", result.Porter.BuyerPower)
	logger.Info().Msgf("Threat of New Entrants: %s", result.Porter.ThreatNewEntrants)
	logger.Info().Msgf("Threat of Substitutes: %s", result.Porter.ThreatSubstitutes)
	logger.Info().Msgf("Overall Attractiveness: %s", result.Porter.OverallAttractiveness)
	logger.Info().Msgf("📝 Summary: %s", result.Porter.Summary)

	logger.Info().Msg("\n3️⃣ TAM-SAM-SOM ANALYSIS")
	logger.Info().Msgf("TAM (Total Addressable Market): %s", result.TamSamSom.TAM)
	logger.Info().Msgf("SAM (Serviceable Addressable Market): %s", result.TamSamSom.SAM)
	logger.Info().Msgf("SOM (Serviceable Obtainable Market): %s", result.TamSamSom.SOM)
	logger.Info().Msgf("CAGR: %s", result.TamSamSom.CAGR)
	logger.Info().Msgf("Assumptions: %v", result.TamSamSom.Assumptions)
	logger.Info().Msgf("📝 Summary: %s", result.TamSamSom.Summary)

	// LAYER 2: POSITIONING
	logger.Info().Msg("\n🎯 LAYER 2: POSITIONING (Internal Fit)")
	logger.Info().Msg(strings.Repeat("-", 100))

	logger.Info().Msg("\n4️⃣ SWOT ANALYSIS")
	logger.Info().Msgf("💪 Strengths: %v", result.SWOT.Strengths)
	logger.Info().Msgf("⚠️ Weaknesses: %v", result.SWOT.Weaknesses)
	logger.Info().Msgf("🚀 Opportunities: %v", result.SWOT.Opportunities)
	logger.Info().Msgf("⚡ Threats: %v", result.SWOT.Threats)
	logger.Info().Msgf("📝 Summary: %s", result.SWOT.Summary)

	logger.Info().Msg("\n5️⃣ BENCHMARKING ANALYSIS")
	logger.Info().Msgf("Competitors Analyzed: %v", result.Benchmarking.Competitors)
	logger.Info().Msgf("Performance Gaps: %v", result.Benchmarking.PerformanceGaps)
	logger.Info().Msgf("Best Practices: %v", result.Benchmarking.BestPractices)
	logger.Info().Msgf("📝 Summary: %s", result.Benchmarking.Summary)

	// LAYER 3: STRATEGY
	logger.Info().Msg("\n🧭 LAYER 3: STRATEGY (Direction)")
	logger.Info().Msg(strings.Repeat("-", 100))

	logger.Info().Msg("\n6️⃣ BLUE OCEAN STRATEGY")
	logger.Info().Msgf("❌ Eliminate: %v", result.BlueOcean.Eliminate)
	logger.Info().Msgf("⬇️ Reduce: %v", result.BlueOcean.Reduce)
	logger.Info().Msgf("⬆️ Raise: %v", result.BlueOcean.Raise)
	logger.Info().Msgf("✨ Create: %v", result.BlueOcean.Create)
	logger.Info().Msgf("New Value Curve: %s", result.BlueOcean.NewValueCurve)
	logger.Info().Msgf("📝 Summary: %s", result.BlueOcean.Summary)

	logger.Info().Msg("\n7️⃣ GROWTH HACKING")
	logger.Info().Msgf("Hypotheses: %v", result.GrowthHacking.Hypotheses)
	logger.Info().Msgf("Experiments: %v", result.GrowthHacking.Experiments)
	logger.Info().Msgf("Key Metrics: %v", result.GrowthHacking.KeyMetrics)
	logger.Info().Msgf("📝 Summary: %s", result.GrowthHacking.Summary)

	logger.Info().Msg("\n8️⃣ SCENARIO PLANNING")
	logger.Info().Msgf("😊 Optimistic: %s", result.Scenarios.ScenarioOptimistic)
	logger.Info().Msgf("😐 Realistic: %s", result.Scenarios.ScenarioRealist)
	logger.Info().Msgf("😰 Pessimistic: %s", result.Scenarios.ScenarioPessimistic)
	logger.Info().Msgf("⚠️ Early Warning Signals: %v", result.Scenarios.EarlyWarningSignals)
	logger.Info().Msgf("📝 Summary: %s", result.Scenarios.Summary)

	// LAYER 4: EXECUTION
	logger.Info().Msg("\n⚡ LAYER 4: EXECUTION (Roadmap)")
	logger.Info().Msg(strings.Repeat("-", 100))

	logger.Info().Msg("\n9️⃣ OKRs (Quarterly Objectives & Key Results)")
	for i, quarter := range result.OKRs.Quarters {
		logger.Info().Msgf("Quarter %d (%s): %s", i+1, quarter.Quarter, quarter.Objective)
		for j, kr := range quarter.KeyResults {
			logger.Info().Msgf("  KR %d.%d: %s", i+1, j+1, kr)
		}
		logger.Info().Msgf("  Investment: %s | Timeline: %s", quarter.Investment, quarter.Timeline)
	}
	logger.Info().Msgf("📝 Summary: %s", result.OKRs.Summary)

	logger.Info().Msg("\n🔟 BALANCED SCORECARD")
	logger.Info().Msgf("💰 Financial: %v", result.BSC.Financial)
	logger.Info().Msgf("👥 Customer: %v", result.BSC.Customer)
	logger.Info().Msgf("⚙️ Internal Processes: %v", result.BSC.Internal)
	logger.Info().Msgf("📚 Learning & Growth: %v", result.BSC.LearningGrowth)
	logger.Info().Msgf("📝 Summary: %s", result.BSC.Summary)

	logger.Info().Msg("\n1️⃣1️⃣ DECISION MATRIX")
	logger.Info().Msgf("Alternatives: %v", result.DecisionMatrix.Alternatives)
	logger.Info().Msgf("Criteria: %v", result.DecisionMatrix.Criteria)
	logger.Info().Msgf("✅ Final Recommendation: %s", result.DecisionMatrix.FinalRecommendation)
	logger.Info().Msgf("📝 Summary: %s", result.DecisionMatrix.Summary)

	// FINAL SYNTHESIS
	logger.Info().Msg("\n🎯 FINAL SYNTHESIS (The Senior Partner)")
	logger.Info().Msg(strings.Repeat("-", 100))
	logger.Info().Msgf("\n📋 Executive Summary:\n%s", result.Synthesis.ExecutiveSummary)
	logger.Info().Msgf("\n🔍 Key Findings: %v", result.Synthesis.KeyFindings)
	logger.Info().Msgf("\n🎯 Strategic Priorities: %v", result.Synthesis.StrategicPriorities)
	logger.Info().Msgf("\n🗺️ Roadmap: %v", result.Synthesis.Roadmap)
	logger.Info().Msgf("\n💡 Overall Recommendation:\n%s", result.Synthesis.OverallRecommendation)

	logger.Info().Msg("\n" + strings.Repeat("=", 100))
	logger.Info().Msg("✅ END OF ANALYSIS")
	logger.Info().Msg(strings.Repeat("=", 100))
}

// ----------------------------------------------------------------------------
// MOCKS AND HELPERS
// ----------------------------------------------------------------------------

// ResilientLLM acts as a proxy. It tries the real API first.
// If that fails (bad key/network), it fills the struct with dummy data.
type ResilientLLM struct {
	realClient *llm.Client
	apiKey     string
	logger     zerolog.Logger
}

func NewResilientLLM(apiKey string, logger zerolog.Logger) *ResilientLLM {
	// If you have a factory for your LLM client, use it here.
	// For this test file, I'll assume standard config initialization or just rely on the Mock logic if key is bad.
	var client *llm.Client
	if len(apiKey) > 10 && apiKey != "sk-or-YOUR_ACTUAL_KEY_HERE" {
		client = llm.NewClientWithBaseURL(apiKey, "https://openrouter.ai/api/v1")
	}
	return &ResilientLLM{realClient: client, apiKey: apiKey, logger: logger}
}

func (r *ResilientLLM) GenerateStructured(ctx context.Context, model string, prompt string, data interface{}, targetSchema interface{}) error {
	// We reuse the Options method with default options
	return r.GenerateStructuredWithOptions(ctx, llm.GenerationOptions{Model: model}, prompt, data, targetSchema)
}

func (r *ResilientLLM) GenerateStructuredWithOptions(ctx context.Context, opts llm.GenerationOptions, prompt string, data interface{}, targetSchema interface{}) error {

	// 1. Try Real Call if client exists
	if r.realClient != nil {
		r.logger.Info().Str("model", opts.Model).Msg("🤖 Attempting Real AI Call...")
		err := r.realClient.GenerateStructuredWithOptions(ctx, opts, prompt, data, targetSchema)
		if err == nil {
			return nil
		}
		r.logger.Warn().Err(err).Msg("⚠️ Real AI Call failed. Falling back to Mock.")
	} else {
		r.logger.Info().Msg("⚠️ No valid API Key. Using Mock Response.")
	}

	// 2. Fallback: Mock Data Injection via JSON unmarshal
	// This ensures the workflow continues even without AI
	mockJSON := r.getMockJSONForSchema(targetSchema)
	return json.Unmarshal([]byte(mockJSON), targetSchema)
}

// Helper to generate valid dummy JSON based on what the struct looks like
// In a robust test, we might check the type of targetSchema, but here we make generic "safe" json.
func (r *ResilientLLM) getMockJSONForSchema(v interface{}) string {
	switch v.(type) {
	case *analysis.PESTELAnalysis:
		return `{"political":["Regulamentações ambientais rígidas"],"economic":["Custo de crédito alto"],"social":["Maior consumo de carne"],"technological":["Agricultura 4.0"],"environmental":["Sustentabilidade"],"legal":["Leis trabalhistas"],"summary":"Análise PESTEL Mockada"}`
	case *analysis.PorterAnalysis:
		return `{"competitive_rivalry":"Alta","supplier_power":"Média","buyer_power":"Alta","threat_new_entrants":"Baixa","threat_substitutes":"Média","overall_attractiveness":"Média","summary":"Análise Porter Mockada"}`
	case *analysis.SWOTAnalysis:
		return `{"strengths":["Marca forte"],"weaknesses":["Baixa digitalização"],"opportunities":["Expansão industrial"],"threats":["Novos entrantes"],"summary":"Análise SWOT Mockada"}`
	case *analysis.AnalysisSynthesis:
		return `{"executive_summary":"A Coimma está bem posicionada mas precisa de digitalização.","key_findings":["Líder em pecuária","Precisa diversificar"],"strategic_priorities":["Digitalização","Novos produtos"],"roadmap":["Q1: Planejamento","Q2: Execução"],"overall_recommendation":"Investir em tecnologia"}`
	default:
		// Generic fallback for other structs (BlueOcean, OKRs, etc) to prevent nil pointers
		return `{"summary": "Mocked Data due to missing/invalid API Key"}`
	}
}

// --- Repo Mocks ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, a *analysis.Analysis) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}
func (m *MockRepository) Update(ctx context.Context, a *analysis.Analysis) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

// Stubs for interface compliance
func (m *MockRepository) GetByID(ctx context.Context, id string) (*analysis.Analysis, error) {
	return nil, nil
}
func (m *MockRepository) GetBySubmissionID(ctx context.Context, id string) (*analysis.Analysis, error) {
	return nil, nil
}
func (m *MockRepository) GetLatestVersionBySubmissionID(ctx context.Context, id string) (*analysis.Analysis, error) {
	return nil, nil
}
func (m *MockRepository) GetAllVersionsBySubmissionID(ctx context.Context, id string) ([]*analysis.Analysis, error) {
	return nil, nil
}
func (m *MockRepository) List(ctx context.Context, limit, offset int) ([]*analysis.Analysis, error) {
	return nil, nil
}
func (m *MockRepository) Delete(ctx context.Context, id string) error { return nil }
func (m *MockRepository) BeginTx(ctx context.Context) (*sqlx.Tx, error) { return nil, nil }
func (m *MockRepository) UpdateWithTx(ctx context.Context, tx *sqlx.Tx, analysis *analysis.Analysis) error { return nil }

type MockSubmissionRepo struct {
	mock.Mock
}

func (m *MockSubmissionRepo) GetByID(ctx context.Context, id uuid.UUID) (*analysis.SubmissionData, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*analysis.SubmissionData), args.Error(1)
}

// Helper
func stringPtr(s string) *string { return &s }
