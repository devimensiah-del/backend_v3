package report_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"backend_v3/domain/analysis"
	"backend_v3/domain/report"
	"backend_v3/domain/submission"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ----------------------------------------------------------------------------
// MOCK IMPLEMENTATIONS
// ----------------------------------------------------------------------------

type MockAnalysisRepo struct {
	mock.Mock
}

func (m *MockAnalysisRepo) GetByID(ctx context.Context, id string) (*analysis.Analysis, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*analysis.Analysis), args.Error(1)
}

type MockSubmissionRepo struct {
	mock.Mock
}

func (m *MockSubmissionRepo) GetByID(ctx context.Context, id uuid.UUID) (*submission.Submission, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*submission.Submission), args.Error(1)
}

type MockReportRepo struct {
	mock.Mock
}

func (m *MockReportRepo) Create(ctx context.Context, r *report.Report) error {
	args := m.Called(ctx, r)
	return args.Error(0)
}

func (m *MockReportRepo) GetBySubmissionID(ctx context.Context, subID string) (*report.Report, error) {
	args := m.Called(ctx, subID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*report.Report), args.Error(1)
}

func (m *MockReportRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockReportRepo) GetByID(ctx context.Context, id string) (*report.Report, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*report.Report), args.Error(1)
}

func (m *MockReportRepo) List(ctx context.Context, limit, offset int) ([]*report.ReportSummary, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*report.ReportSummary), args.Error(1)
}

func (m *MockReportRepo) Update(ctx context.Context, r *report.Report) error {
	args := m.Called(ctx, r)
	return args.Error(0)
}

func (m *MockReportRepo) Upsert(ctx context.Context, r *report.Report) error {
	args := m.Called(ctx, r)
	return args.Error(0)
}

type MockPDFGenerator struct {
	mock.Mock
}

func (m *MockPDFGenerator) Convert(ctx context.Context, htmlPages []string) ([]byte, error) {
	args := m.Called(ctx, htmlPages)
	return args.Get(0).([]byte), args.Error(1)
}

type MockStorageClient struct {
	mock.Mock
}

func (m *MockStorageClient) Upload(ctx context.Context, path string, data []byte, contentType string) (string, error) {
	args := m.Called(ctx, path, data, contentType)
	return args.Get(0).(string), args.Error(1)
}

// ----------------------------------------------------------------------------
// TEST DATA - COIMMA ANALYSIS FROM WORKFLOW TEST
// ----------------------------------------------------------------------------

func getCoimmaAnalysis() *analysis.Analysis {
	return &analysis.Analysis{
		ID:           "6057454f-12bf-4783-8334-f5a4424ad246",
		SubmissionID: "db32e622-56bb-4fba-a480-9384aeaa2f5c",
		Status:       "completed",

		// PESTEL Analysis
		PESTEL: analysis.PESTELAnalysis{
			Political: []string{
				"Reformaz tributária: Impacto nos custos e investimentos.",
				"Incentivos ao agronegócio: Oportunidades de expansão.",
				"Estabilidade política moderada: Risco de mudanças.",
			},
			Economic: []string{
				"SELIC a 10.5%: Custo de crédito para expansão.",
				"Crescimento do PIB +2.0%: Expansão do mercado.",
				"Inflação 4.5%: Aumento de custos operacionais.",
			},
			Social: []string{
				"Consumidor exigente: Qualidade e rastreabilidade.",
				"Adoção de tecnologia: Demanda por inovação.",
				"Segurança alimentar: Rastreabilidade crucial.",
			},
			Technological: []string{
				"Automação: Oportunidade em sistemas Coimma.",
				"Agricultura de precisão: Demanda por tecnologia.",
				"IoT e conectividade: Integração de equipamentos.",
			},
			Environmental: []string{
				"Normas ambientais: Adaptação da produção.",
				"Riscos climáticos: Impacto na agropecuária.",
				"Sustentabilidade: Demanda por práticas verdes.",
			},
			Legal: []string{
				"LGPD: Adequação na coleta de dados.",
				"Normas ABNT: Conformidade dos equipamentos.",
				"Rastreabilidade: Novas regras agropecuárias.",
			},
			Summary: "A Coimma deve aproveitar incentivos no agro, investir em automação e rastreabilidade, adaptar-se às normas ambientais e à LGPD, e mitigar riscos da inflação e instabilidade política, focando na expansão para o setor industrial.",
		},

		// Porter's 7 Forces Analysis
		Porter: analysis.PorterAnalysis{
			CompetitiveRivalry:                   "Alta rivalidade devido a lançamentos de produtos com tecnologia embarcada e expansão internacional dos concorrentes. Concentração moderada exige diferenciação.",
			SupplierPower:                        "Média. Preços voláteis do aço e desafios na cadeia de suprimentos (pandemia/conflitos) impactam custos. Buscar diversificação de fornecedores.",
			BuyerPower:                           "Média. Consumidores exigentes (qualidade/segurança) e inflação (4.5% a.a.) aumentam pressão por preços competitivos e valor agregado.",
			ThreatNewEntrants:                    "Média. Barreiras de entrada (conhecimento técnico/P&D) são atenuadas por incentivos ao agronegócio. Conformidade com normas (ISO, ABNT) é crucial.",
			ThreatSubstitutes:                    "Baixa. Adoção crescente de tecnologias digitais e agricultura de precisão podem substituir equipamentos tradicionais. Monitorar inovações.",
			PowerPartnershipsEcosystems:          "Parcerias com startups de AgTech e fornecedores de IoT podem fortalecer a posição competitiva e criar ecossistemas de valor.",
			DisruptionAIData:                     "IA e análise de dados podem revolucionar a pecuária de precisão, criando oportunidades para novos modelos de negócio baseados em dados.",
			CompetitiveRivalryIntensity:          "Alta",
			SupplierPowerIntensity:               "Média",
			BuyerPowerIntensity:                  "Média",
			ThreatNewEntrantsIntensity:           "Média",
			ThreatSubstitutesIntensity:           "Baixa",
			PowerPartnershipsEcosystemsIntensity: "Alta",
			DisruptionAIDataIntensity:            "Alta",
			StrategicImplications: []string{
				"Focar em diferenciação através de inovação e qualidade",
				"Otimizar cadeia de suprimentos para mitigar riscos de fornecedores",
				"Desenvolver parcerias estratégicas com AgTech e IoT",
				"Investir em soluções baseadas em dados e IA",
			},
			OverallAttractiveness: "Média",
			Summary:               "A Coimma enfrenta um mercado com rivalidade alta, poder de barganha dos compradores e fornecedores moderados, e ameaças de novos entrantes e substitutos médias e baixas, respectivamente. Oportunidades em parcerias e disrupção por IA/dados são importantes. A empresa deve focar em diferenciação, otimização da cadeia de suprimentos e parcerias estratégicas para manter a competitividade.",
		},

		// SWOT Analysis with Confidence Badges
		SWOT: analysis.SWOTAnalysis{
			Strengths: []analysis.SWOTItem{
				{Content: "Forte presença no mercado pecuário.", Confidence: "Alta", Source: "fato"},
				{Content: "Experiência em sistemas de pesagem.", Confidence: "Alta", Source: "fato"},
				{Content: "Estrutura de vendas e assistência técnica.", Confidence: "Alta", Source: "fato"},
				{Content: "Reputação de qualidade no setor.", Confidence: "Média", Source: "análise de mercado"},
			},
			Weaknesses: []analysis.SWOTItem{
				{Content: "Dependência do mercado pecuário.", Confidence: "Alta", Source: "fato"},
				{Content: "Diversificação industrial é necessária.", Confidence: "Alta", Source: "fato"},
				{Content: "Adoção de novas tecnologias lenta.", Confidence: "Média", Source: "análise de mercado"},
				{Content: "Faturamento em serviços é baixo (5%).", Confidence: "Média", Source: "fato"},
			},
			Opportunities: []analysis.SWOTItem{
				{Content: "Incentivos fiscais para o agronegócio.", Confidence: "Alta", Source: "análise de mercado"},
				{Content: "Crescimento da automação e rastreabilidade.", Confidence: "Alta", Source: "análise de mercado"},
				{Content: "Expansão para o setor industrial.", Confidence: "Média", Source: "análise de mercado"},
				{Content: "Parcerias estratégicas (IA/Dados).", Confidence: "Média", Source: "análise de mercado"},
			},
			Threats: []analysis.SWOTItem{
				{Content: "Alta rivalidade no mercado.", Confidence: "Média", Source: "análise de mercado"},
				{Content: "Inflação e instabilidade política.", Confidence: "Média", Source: "análise de mercado"},
				{Content: "Novos entrantes no mercado.", Confidence: "Baixa", Source: "análise de mercado"},
				{Content: "Normas ambientais e LGPD.", Confidence: "Média", Source: "análise de mercado"},
			},
			Summary: "Focar na expansão industrial e otimizar a cadeia de suprimentos, aproveitando a reputação e estrutura existente no mercado pecuário.",
		},

		// TAM-SAM-SOM Analysis
		TamSamSom: analysis.TamSamSomAnalysis{
			TAM:  "R$ 120B",
			SAM:  "R$ 40B",
			SOM:  "R$ 300M",
			CAGR: "+7% a.a.",
			Assumptions: []string{
				"Agronegócio cresce +8% CAGR (2024-2028).",
				"Automação industrial +6% CAGR.",
				"PIB Brasil +2% em 2025.",
			},
			Summary: "Mercado em expansão, com automação e agronegócio impulsionando o crescimento. Coimma tem potencial significativo.",
		},

		// Blue Ocean Strategy
		BlueOcean: analysis.BlueOceanAnalysis{
			Eliminate: []string{
				"Vendas reativas (esperar o cliente)",
				"Foco excessivo em features complexas",
				"Marketing genérico (sem segmentação)",
			},
			Reduce: []string{
				"Custos de manutenção corretiva",
				"Complexidade da linha de produtos",
				"Tempo de resposta ao cliente",
			},
			Raise: []string{
				"Qualidade do atendimento ao cliente",
				"Investimento em P&D (automação)",
				"Treinamento da equipe de vendas",
			},
			Create: []string{
				"Soluções integradas (hardware/software)",
				"Plataforma de dados para pecuaristas",
				"Serviços de consultoria (otimização)",
			},
			NewValueCurve: "Foco em soluções completas, dados e consultoria para pecuaristas, com atendimento premium e inovação constante em automação e software.",
			Summary:       "Coimma se transforma em provedora de soluções integradas, impulsionada por dados e consultoria, para otimizar a pecuária e expandir para a indústria.",
		},

		// Quarterly OKRs
		OKRs: analysis.OKRAnalysis{
			Quarters: []analysis.QuarterlyOKR{
				{
					Quarter:   "Q1 2025",
					Objective: "Validar a expansão para o segmento de grãos e armazenamento.",
					KeyResults: []string{
						"Aumentar as vendas no segmento grãos em 15%.",
						"Realizar 20 visitas técnicas a clientes potenciais.",
						"Obter 3 novos contratos de serviço de calibração.",
					},
					Investment: "R$ 25 mil",
					Timeline:   "3 meses",
				},
				{
					Quarter:   "Q2 2025",
					Objective: "Crescer a receita e a base de clientes no setor pecuário.",
					KeyResults: []string{
						"Aumentar a receita total em 10%.",
						"Adicionar 15 novos clientes no setor pecuário.",
						"Elevar a satisfação do cliente para 4.5/5.",
					},
					Investment: "R$ 50 mil",
					Timeline:   "3 meses",
				},
				{
					Quarter:   "Q3 2025",
					Objective: "Escalar a oferta de serviços e explorar o mercado industrial.",
					KeyResults: []string{
						"Aumentar a receita de serviços em 20%.",
						"Desenvolver 2 protótipos para o setor industrial.",
						"Realizar 10 estudos de mercado no setor industrial.",
					},
					Investment: "R$ 100 mil",
					Timeline:   "3 meses",
				},
			},
			Summary: "Expansão gradual, focando em pecuária, grãos e explorando o setor industrial.",
		},

		// Growth Hacking with LEAP + SCALE Loops
		GrowthHacking: analysis.GrowthHackingAnalysis{
			LeapLoop: analysis.GrowthLoop{
				Name: "LEAP Loop - Aquisição de Clientes",
				Type: "Aquisição",
				Steps: []string{
					"Lead Generation: Criar conteúdo educativo sobre automação pecuária",
					"Engagement: Oferecer webinars e demos técnicas gratuitas",
					"Activation: Trial de 30 dias com assistência técnica dedicada",
					"Purchase: Conversão com desconto para early adopters",
				},
				Metrics: []string{
					"Custo por Lead (CPL)",
					"Taxa de Conversão Trial → Venda",
					"Lifetime Value (LTV) do Cliente",
				},
				Bottleneck: "Baixa conversão de leads para trials devido à falta de awareness sobre automação",
			},
			ScaleLoop: analysis.GrowthLoop{
				Name: "SCALE Loop - Monetização e Retenção",
				Type: "Monetização",
				Steps: []string{
					"Service Adoption: Cliente utiliza serviços de calibração mensal",
					"Cross-sell: Apresentar produtos complementares (software, sensores IoT)",
					"Annual Contract: Migrar para contrato anual com desconto",
					"Loyalty & Expansion: Programa de fidelidade com upgrades automáticos",
					"Referral: Cliente indica outros pecuaristas (incentivo de 10%)",
				},
				Metrics: []string{
					"Monthly Recurring Revenue (MRR)",
					"Churn Rate",
					"Net Revenue Retention (NRR)",
					"Referral Rate",
				},
				Bottleneck: "Baixo MRR devido aos serviços representarem apenas 5% do faturamento",
			},
			Summary: "Fase 1: Expansão no mercado pecuário e grãos com foco em conteúdo e canais digitais. Fase 2: Diversificação para indústria com novos produtos.",
		},

		// Scenario Analysis with Probabilities
		Scenarios: analysis.ScenarioAnalysis{
			Optimistic: analysis.Scenario{
				Name:        "Cenário Otimista: Boom do Agronegócio",
				Probability: 20,
				Description: "Incentivos fiscais robustos, PIB crescendo +3.5%, inflação controlada em 3%, e alta adoção de tecnologia no setor pecuário impulsionam vendas da Coimma.",
				RequiredActions: []string{
					"Expandir capacidade produtiva em 30%",
					"Contratar 50 novos colaboradores",
					"Investir R$ 5M em P&D para produtos industriais",
				},
			},
			Realist: analysis.Scenario{
				Name:        "Cenário Realista: Crescimento Gradual",
				Probability: 60,
				Description: "PIB +2%, inflação em 4.5%, adoção moderada de tecnologia. Coimma cresce no mercado pecuário e inicia testes no setor industrial.",
				RequiredActions: []string{
					"Manter estrutura atual e otimizar processos",
					"Desenvolver 2 protótipos para setor industrial",
					"Investir R$ 1M em marketing digital",
				},
			},
			Pessimistic: analysis.Scenario{
				Name:        "Cenário Pessimista: Recessão e Instabilidade",
				Probability: 20,
				Description: "PIB estagnado, inflação acima de 6%, instabilidade política e redução de investimentos no agronegócio.",
				RequiredActions: []string{
					"Reduzir custos operacionais em 20%",
					"Focar em contratos de serviço recorrente",
					"Adiar investimentos em expansão industrial",
				},
			},
			MitigationTactics: []string{
				"Diversificar portfólio para reduzir dependência do setor pecuário",
				"Criar reserva de caixa para 6 meses de operação",
				"Estabelecer parcerias estratégicas para compartilhar riscos",
			},
			EarlyWarningSignals: []string{
				"Aumento da inflação acima da meta (4.5%)",
				"Queda nos preços das commodities agrícolas",
				"Aumento da taxa de desemprego",
			},
			Summary: "A Coimma deve equilibrar a expansão industrial com a solidez no mercado pecuário, monitorando de perto os indicadores econômicos e regulatórios para ajustar suas estratégias.",
		},

		// Decision Matrix with Recommendations
		DecisionMatrix: analysis.DecisionMatrixAnalysis{
			RecommendedOption: "Abordagem Híbrida: Expansão Gradual e Diversificação",
			Score:             "8.5/10",
			ScoreComparison:   "Superior às alternativas A (7.2/10) e B (6.8/10)",
			PriorityRecommendations: []analysis.PriorityRecommendation{
				{
					Priority:    1,
					Title:       "Fortalecer a Presença Digital e Marketing de Conteúdo",
					Description: "Criar blog técnico, webinars mensais e cases de sucesso para atrair leads qualificados no mercado pecuário e grãos.",
					Timeline:    "3-6 meses",
					Budget:      "R$ 150 mil",
				},
				{
					Priority:    2,
					Title:       "Desenvolver Protótipos para o Setor Industrial",
					Description: "Iniciar P&D de 2 produtos-piloto para automação industrial, com testes em clientes-beta.",
					Timeline:    "6-12 meses",
					Budget:      "R$ 800 mil",
				},
				{
					Priority:    3,
					Title:       "Otimizar Cadeia de Suprimentos e Reduzir Custos",
					Description: "Diversificar fornecedores de aço, implementar controle de estoque JIT e renegociar contratos com fornecedores.",
					Timeline:    "3-9 meses",
					Budget:      "R$ 200 mil",
				},
			},
			MonitoringMetrics: []string{
				"Receita Total (meta: +15% a.a.)",
				"Margem Bruta (meta: 30%)",
				"NPS (Net Promoter Score) (meta: >50)",
				"% Receita de Serviços (meta: 10%)",
				"Custo de Aquisição de Cliente (CAC)",
			},
			ReviewCycle: analysis.ReviewCycle{
				Frequency: "Trimestral",
				ExtraordinaryTriggers: []string{
					"Inflação acima de 6% por 2 trimestres consecutivos",
					"Queda de receita >10% em qualquer trimestre",
					"Entrada de novo concorrente com market share >15%",
				},
			},
			Alternatives: []string{
				"A: Expansão Acelerada para o Setor Industrial",
				"B: Consolidação e Otimização no Mercado Pecuário",
				"C: Abordagem Híbrida: Expansão Gradual e Diversificação",
			},
			Criteria: []string{
				"Retorno sobre o Investimento (ROI)",
				"Risco de Mercado e Operacional",
				"Tempo para Implementação e Resultados",
			},
			FinalRecommendation: "A abordagem híbrida permite diversificar riscos, capitalizar no mercado atual e preparar a Coimma para o futuro industrial. Foco inicial no mercado pecuário, enquanto se investe em P&D para o setor industrial.",
			Summary:             "A Coimma deve equilibrar a expansão industrial com a solidez no mercado pecuário, monitorando de perto os indicadores econômicos e regulatórios para ajustar suas estratégias. Próximo passo: iniciar o desenvolvimento dos produtos piloto para indústria.",
		},

		// Balanced Scorecard
		BSC: analysis.BalancedScorecardAnalysis{
			Financial: []string{
				"Aumentar receita total em 15%",
				"Expandir margem bruta para 30%",
			},
			Customer: []string{
				"Aumentar satisfação do cliente",
				"Expandir base de clientes em 10%",
			},
			Internal: []string{
				"Otimizar processos produtivos",
				"Reduzir tempo de entrega em 20%",
			},
			LearningGrowth: []string{
				"Capacitar equipe em novas técn.",
				"Investir em P&D (5% da receita)",
			},
			Summary: "Coimma focará expansão, otimização e inovação para se tornar líder em soluções integradas.",
		},

		// Benchmarking
		Benchmarking: analysis.BenchmarkingAnalysis{
			Competitors: []string{"Beckhauser", "Balanças Açores", "Ramtec"},
			PerformanceGaps: []string{
				"Dependência excessiva do mercado pecuário.",
				"Lenta adoção de novas tecnologias.",
				"Necessidade de diversificação para o setor industrial.",
			},
			BestPractices: []string{
				"Forte presença no mercado pecuário.",
				"Experiência consolidada em sistemas de pesagem.",
				"Estrutura de vendas e assistência técnica estabelecida.",
			},
			Summary: "Coimma deve diversificar para indústria, acelerar adoção de tecnologia e reduzir dependência do setor pecuário.",
		},

		// Final Synthesis
		Synthesis: analysis.AnalysisSynthesis{
			CentralChallenge: "Coimma precisa equilibrar a expansão para o setor industrial com a consolidação da liderança no mercado pecuário, mantendo margens saudáveis em um cenário de inflação e alta rivalidade competitiva.",
			ExecutiveSummary: "Coimma expande para indústria, aproveitando expertise em sistemas de pesagem e estrutura existente. Foco em inovação e diversificação para crescimento sustentável.",
			MainFindings: []string{
				"Expansão gradual: pecuária, grãos e indústria.",
				"Conteúdo e canais digitais são cruciais.",
				"Foco em diferenciação e parcerias estratégicas.",
				"Otimização da cadeia de suprimentos é prioritária.",
			},
			ImportantNotes: []string{
				"Monitorar inflação e taxa SELIC para ajustar estratégia de investimento",
				"Normas ambientais e LGPD requerem adaptação imediata",
			},
			KeyFindings: []string{
				"Mercado pecuário representa 70% da receita e oferece estabilidade",
				"Setor industrial apresenta alto potencial de crescimento (+6% CAGR)",
				"Parcerias com AgTech podem acelerar inovação",
			},
			StrategicPriorities: []string{
				"Expandir vendas no mercado pecuário.",
				"Desenvolver novos produtos industriais.",
				"Otimizar cadeia de suprimentos.",
			},
			Roadmap: []string{
				"Curto Prazo (3-6 meses): Fortalecer presença digital e otimizar processos internos.",
				"Médio Prazo (6-12 meses): Lançar novos produtos para o segmento de grãos e iniciar testes no setor industrial.",
				"Longo Prazo (12+ meses): Expansão para o mercado industrial com estrutura de vendas e assistência técnica dedicada.",
			},
			OverallRecommendation: "Diversificar para indústria, mantendo a liderança no mercado pecuário.",
		},

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func getCoimmaSubmission() *submission.Submission {
	companyWebsite := "https://www.coimma.com.br"
	companyIndustry := "Fabricação de equipamentos para pecuária"
	companySize := "50-200 funcionários"
	companyLocation := "Dracena, São Paulo, Brasil"

	return &submission.Submission{
		ID:                uuid.MustParse("db32e622-56bb-4fba-a480-9384aeaa2f5c"),
		CompanyName:       "Coimma",
		CompanyWebsite:    &companyWebsite,
		CompanyIndustry:   &companyIndustry,
		CompanySize:       &companySize,
		CompanyLocation:   &companyLocation,
		BusinessChallenge: "Sou CEO da Coimma. Somos fabricantes de equipamentos de manejo para gado, balanças bovinas eletronica, balanças rodoviárias, automação de balancas rodoviárias e balanças de fluxo por batelada para grãos. Atualmente 70% do nosso faturamento é proveniente do mercado pecuária. Os outros 30% sao provenientes de segmentos diversos destacando-se grãos e armazenamento. A prestação de serviços de calibração e manutenção representa 5% do faturamento.Nossas competências são fabricação de equipamentos, sistemas de pesagem e eletrônica elabore um plano de de negócios para nossa companhia em duas fases. A primeira fase mais imediata visa expandir as nossas vendas com nosso portfólio atual de produtos e aproveitando nossa estrutura atual de vendas e assistência técnica. Na segunda fase queremos desenvolver novos produtos para o segmento industrial que requer investimentos em estrutura de vendas e assistência técnica.",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// ----------------------------------------------------------------------------
// TEST: Generate Preview (All HTML Templates)
// ----------------------------------------------------------------------------

func TestReportService_GeneratePreview_Coimma(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Create mocks
	mockAnalysisRepo := new(MockAnalysisRepo)
	mockSubmissionRepo := new(MockSubmissionRepo)
	mockReportRepo := new(MockReportRepo)
	mockPDFGen := new(MockPDFGenerator)
	mockStorage := new(MockStorageClient)

	// Setup test data
	coimmaAnalysis := getCoimmaAnalysis()
	coimmaSubmission := getCoimmaSubmission()

	// Mock expectations
	mockSubmissionRepo.On("GetByID", ctx, uuid.MustParse("db32e622-56bb-4fba-a480-9384aeaa2f5c")).
		Return(coimmaSubmission, nil)
	mockAnalysisRepo.On("GetByID", ctx, "6057454f-12bf-4783-8334-f5a4424ad246").
		Return(coimmaAnalysis, nil)

	// Create service
	svc := report.NewService(
		mockReportRepo,
		mockAnalysisRepo,
		mockSubmissionRepo,
		mockPDFGen,
		mockStorage,
		logger,
	)

	// Execute
	logger.Info().Msg("🚀 Generating HTML Preview for Coimma Report...")
	htmlPages, err := svc.GeneratePreview(ctx, "db32e622-56bb-4fba-a480-9384aeaa2f5c", "6057454f-12bf-4783-8334-f5a4424ad246")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, htmlPages)

	// Log results
	logger.Info().Msgf("✅ Generated %d HTML pages", len(htmlPages))
	for pageName, html := range htmlPages {
		logger.Info().Msgf("  📄 %s: %d characters", pageName, len(html))

		// Save HTML to file for manual inspection
		fileName := "test_artifacts/html_previews/" + pageName + ".html"
		err := os.WriteFile(fileName, []byte(html), 0644)
		if err == nil {
			logger.Info().Msgf("    💾 Saved to: %s", fileName)
		}
	}

	// Verify all expected pages are present (24 total with new TUC Glasses structure)
	expectedPages := []string{
		// Executive Overview
		"Cover", "ExecutiveSummary", "TableOfContents",
		// Part I
		"DividerPart1", "PESTEL_PES", "PESTEL_TEL", "Porter", "SWOT",
		// Part II
		"DividerPart2", "MarketSizing", "BlueOcean",
		// Part III
		"DividerPart3", "OKRs", "GrowthLoops",
		// Part IV
		"DividerPart4", "Scenarios", "Recommendations",
		// Appendices
		"BusinessModel", "CompetitiveAnalysis", "FinancialProjections",
		"GTMStrategy", "RiskAssessment", "Roadmap", "Appendix",
	}

	for _, page := range expectedPages {
		assert.Contains(t, htmlPages, page, "Missing page: "+page)
		assert.NotEmpty(t, htmlPages[page], "Empty HTML for page: "+page)
	}

	mockAnalysisRepo.AssertExpectations(t)
	mockSubmissionRepo.AssertExpectations(t)
}

// ----------------------------------------------------------------------------
// TEST: Publish Report (Generate PDF)
// ----------------------------------------------------------------------------

func TestReportService_Publish_Coimma(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Create mocks
	mockAnalysisRepo := new(MockAnalysisRepo)
	mockSubmissionRepo := new(MockSubmissionRepo)
	mockReportRepo := new(MockReportRepo)
	mockPDFGen := new(MockPDFGenerator)
	mockStorage := new(MockStorageClient)

	// Setup test data
	coimmaAnalysis := getCoimmaAnalysis()
	coimmaSubmission := getCoimmaSubmission()

	// Mock expectations
	mockSubmissionRepo.On("GetByID", ctx, uuid.MustParse("db32e622-56bb-4fba-a480-9384aeaa2f5c")).
		Return(coimmaSubmission, nil)
	mockAnalysisRepo.On("GetByID", ctx, "6057454f-12bf-4783-8334-f5a4424ad246").
		Return(coimmaAnalysis, nil)

	// Mock PDF generation
	mockPDFBytes := []byte("fake-pdf-content-for-testing")
	mockPDFGen.On("Convert", ctx, mock.AnythingOfType("[]string")).
		Return(mockPDFBytes, nil)

	// Mock storage upload
	mockPDFURL := "https://storage.example.com/reports/coimma-report.pdf"
	mockStorage.On("Upload", ctx, mock.AnythingOfType("string"), mockPDFBytes, "application/pdf").
		Return(mockPDFURL, nil)

	// Mock report upsert (service uses Upsert for idempotency)
	mockReportRepo.On("Upsert", ctx, mock.AnythingOfType("*report.Report")).
		Return(nil)

	// Create service
	svc := report.NewService(
		mockReportRepo,
		mockAnalysisRepo,
		mockSubmissionRepo,
		mockPDFGen,
		mockStorage,
		logger,
	)

	// Execute
	logger.Info().Msg("🚀 Publishing Coimma Report (Generating PDF)...")
	pdfURL, err := svc.Publish(ctx, "db32e622-56bb-4fba-a480-9384aeaa2f5c", "6057454f-12bf-4783-8334-f5a4424ad246")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, mockPDFURL, pdfURL)

	logger.Info().Msgf("✅ PDF Published Successfully: %s", pdfURL)

	mockAnalysisRepo.AssertExpectations(t)
	mockSubmissionRepo.AssertExpectations(t)
	mockPDFGen.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockReportRepo.AssertExpectations(t)
}

// ----------------------------------------------------------------------------
// TEST: Publish - PDF Generation Failure
// ----------------------------------------------------------------------------

func TestReportService_Publish_PDFGenerationFails(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	mockAnalysisRepo := new(MockAnalysisRepo)
	mockSubmissionRepo := new(MockSubmissionRepo)
	mockReportRepo := new(MockReportRepo)
	mockPDFGen := new(MockPDFGenerator)
	mockStorage := new(MockStorageClient)

	coimmaAnalysis := getCoimmaAnalysis()
	coimmaSubmission := getCoimmaSubmission()

	mockSubmissionRepo.On("GetByID", ctx, uuid.MustParse("db32e622-56bb-4fba-a480-9384aeaa2f5c")).
		Return(coimmaSubmission, nil)
	mockAnalysisRepo.On("GetByID", ctx, "6057454f-12bf-4783-8334-f5a4424ad246").
		Return(coimmaAnalysis, nil)

	// Mock PDF generation failure
	mockPDFGen.On("Convert", ctx, mock.AnythingOfType("[]string")).
		Return([]byte{}, fmt.Errorf("gotenberg service unavailable"))

	svc := report.NewService(
		mockReportRepo,
		mockAnalysisRepo,
		mockSubmissionRepo,
		mockPDFGen,
		mockStorage,
		logger,
	)

	// Execute
	pdfURL, err := svc.Publish(ctx, "db32e622-56bb-4fba-a480-9384aeaa2f5c", "6057454f-12bf-4783-8334-f5a4424ad246")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, pdfURL)
	assert.Contains(t, err.Error(), "pdf generation failed")

	mockAnalysisRepo.AssertExpectations(t)
	mockSubmissionRepo.AssertExpectations(t)
	mockPDFGen.AssertExpectations(t)
	// Storage should NOT be called
	mockStorage.AssertNotCalled(t, "Upload")
}

// ----------------------------------------------------------------------------
// TEST: Publish - Storage Upload Failure
// ----------------------------------------------------------------------------

func TestReportService_Publish_StorageUploadFails(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	mockAnalysisRepo := new(MockAnalysisRepo)
	mockSubmissionRepo := new(MockSubmissionRepo)
	mockReportRepo := new(MockReportRepo)
	mockPDFGen := new(MockPDFGenerator)
	mockStorage := new(MockStorageClient)

	coimmaAnalysis := getCoimmaAnalysis()
	coimmaSubmission := getCoimmaSubmission()

	mockSubmissionRepo.On("GetByID", ctx, uuid.MustParse("db32e622-56bb-4fba-a480-9384aeaa2f5c")).
		Return(coimmaSubmission, nil)
	mockAnalysisRepo.On("GetByID", ctx, "6057454f-12bf-4783-8334-f5a4424ad246").
		Return(coimmaAnalysis, nil)

	mockPDFBytes := []byte("fake-pdf-content")
	mockPDFGen.On("Convert", ctx, mock.AnythingOfType("[]string")).
		Return(mockPDFBytes, nil)

	// Mock storage upload failure
	mockStorage.On("Upload", ctx, mock.AnythingOfType("string"), mockPDFBytes, "application/pdf").
		Return("", fmt.Errorf("supabase connection timeout"))

	svc := report.NewService(
		mockReportRepo,
		mockAnalysisRepo,
		mockSubmissionRepo,
		mockPDFGen,
		mockStorage,
		logger,
	)

	// Execute
	pdfURL, err := svc.Publish(ctx, "db32e622-56bb-4fba-a480-9384aeaa2f5c", "6057454f-12bf-4783-8334-f5a4424ad246")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, pdfURL)
	assert.Contains(t, err.Error(), "storage upload failed")

	mockAnalysisRepo.AssertExpectations(t)
	mockSubmissionRepo.AssertExpectations(t)
	mockPDFGen.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	// Report should NOT be created
	mockReportRepo.AssertNotCalled(t, "Create")
}

// ----------------------------------------------------------------------------
// TEST: Publish - Missing Submission
// ----------------------------------------------------------------------------

func TestReportService_Publish_SubmissionNotFound(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	mockAnalysisRepo := new(MockAnalysisRepo)
	mockSubmissionRepo := new(MockSubmissionRepo)
	mockReportRepo := new(MockReportRepo)
	mockPDFGen := new(MockPDFGenerator)
	mockStorage := new(MockStorageClient)

	// Mock submission not found
	mockSubmissionRepo.On("GetByID", ctx, uuid.MustParse("db32e622-56bb-4fba-a480-9384aeaa2f5c")).
		Return(nil, fmt.Errorf("submission not found"))

	svc := report.NewService(
		mockReportRepo,
		mockAnalysisRepo,
		mockSubmissionRepo,
		mockPDFGen,
		mockStorage,
		logger,
	)

	// Execute
	pdfURL, err := svc.Publish(ctx, "db32e622-56bb-4fba-a480-9384aeaa2f5c", "6057454f-12bf-4783-8334-f5a4424ad246")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, pdfURL)

	mockSubmissionRepo.AssertExpectations(t)
	// Analysis should NOT be fetched
	mockAnalysisRepo.AssertNotCalled(t, "GetByID")
}

// ----------------------------------------------------------------------------
// TEST: Publish - Missing Analysis
// ----------------------------------------------------------------------------

func TestReportService_Publish_AnalysisNotFound(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	mockAnalysisRepo := new(MockAnalysisRepo)
	mockSubmissionRepo := new(MockSubmissionRepo)
	mockReportRepo := new(MockReportRepo)
	mockPDFGen := new(MockPDFGenerator)
	mockStorage := new(MockStorageClient)

	coimmaSubmission := getCoimmaSubmission()

	mockSubmissionRepo.On("GetByID", ctx, uuid.MustParse("db32e622-56bb-4fba-a480-9384aeaa2f5c")).
		Return(coimmaSubmission, nil)

	// Mock analysis not found
	mockAnalysisRepo.On("GetByID", ctx, "6057454f-12bf-4783-8334-f5a4424ad246").
		Return(nil, fmt.Errorf("analysis not found"))

	svc := report.NewService(
		mockReportRepo,
		mockAnalysisRepo,
		mockSubmissionRepo,
		mockPDFGen,
		mockStorage,
		logger,
	)

	// Execute
	pdfURL, err := svc.Publish(ctx, "db32e622-56bb-4fba-a480-9384aeaa2f5c", "6057454f-12bf-4783-8334-f5a4424ad246")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, pdfURL)

	mockSubmissionRepo.AssertExpectations(t)
	mockAnalysisRepo.AssertExpectations(t)
}

// ----------------------------------------------------------------------------
// TEST: GeneratePreview - All 24 Pages Rendered
// ----------------------------------------------------------------------------

func TestReportService_GeneratePreview_AllPagesRendered(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	mockAnalysisRepo := new(MockAnalysisRepo)
	mockSubmissionRepo := new(MockSubmissionRepo)
	mockReportRepo := new(MockReportRepo)
	mockPDFGen := new(MockPDFGenerator)
	mockStorage := new(MockStorageClient)

	coimmaAnalysis := getCoimmaAnalysis()
	coimmaSubmission := getCoimmaSubmission()

	mockSubmissionRepo.On("GetByID", ctx, uuid.MustParse("db32e622-56bb-4fba-a480-9384aeaa2f5c")).
		Return(coimmaSubmission, nil)
	mockAnalysisRepo.On("GetByID", ctx, "6057454f-12bf-4783-8334-f5a4424ad246").
		Return(coimmaAnalysis, nil)

	svc := report.NewService(
		mockReportRepo,
		mockAnalysisRepo,
		mockSubmissionRepo,
		mockPDFGen,
		mockStorage,
		logger,
	)

	// Execute
	pages, err := svc.GeneratePreview(ctx, "db32e622-56bb-4fba-a480-9384aeaa2f5c", "6057454f-12bf-4783-8334-f5a4424ad246")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, pages)

	// Verify all 24 pages exist
	expectedPages := []string{
		"Cover", "ExecutiveSummary", "TableOfContents",
		"DividerPart1", "PESTEL_PES", "PESTEL_TEL", "Porter", "SWOT",
		"DividerPart2", "MarketSizing", "BlueOcean",
		"DividerPart3", "OKRs", "GrowthLoops",
		"DividerPart4", "Scenarios", "Recommendations",
		"BusinessModel", "CompetitiveAnalysis", "FinancialProjections",
		"GTMStrategy", "RiskAssessment", "Roadmap", "Appendix",
	}

	assert.Equal(t, len(expectedPages), len(pages), "Should have exactly 24 pages")

	for _, pageName := range expectedPages {
		assert.Contains(t, pages, pageName, "Missing page: "+pageName)
		assert.NotEmpty(t, pages[pageName], "Page should have HTML content: "+pageName)
		assert.Contains(t, pages[pageName], "<!DOCTYPE html>", "Page should be complete HTML: "+pageName)
	}

	mockAnalysisRepo.AssertExpectations(t)
	mockSubmissionRepo.AssertExpectations(t)
}

// ----------------------------------------------------------------------------
// TEST: GetBySubmissionID
// ----------------------------------------------------------------------------

func TestReportService_GetBySubmissionID_Found(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	mockAnalysisRepo := new(MockAnalysisRepo)
	mockSubmissionRepo := new(MockSubmissionRepo)
	mockReportRepo := new(MockReportRepo)
	mockPDFGen := new(MockPDFGenerator)
	mockStorage := new(MockStorageClient)

	submissionID := uuid.New().String()
	now := time.Now()

	expectedReport := &report.Report{
		ID:           uuid.New().String(),
		SubmissionID: submissionID,
		AnalysisID:   uuid.New().String(),
		Status:       "completed",
		PDFURL:       "https://storage.example.com/report.pdf",
		TotalPages:   24,
		CreatedAt:    now,
	}

	mockReportRepo.On("GetBySubmissionID", ctx, submissionID).
		Return(expectedReport, nil)

	svc := report.NewService(
		mockReportRepo,
		mockAnalysisRepo,
		mockSubmissionRepo,
		mockPDFGen,
		mockStorage,
		logger,
	)

	// Execute
	result, err := svc.GetBySubmissionID(ctx, submissionID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedReport.ID, result.ID)
	assert.Equal(t, submissionID, result.SubmissionID)

	mockReportRepo.AssertExpectations(t)
}

func TestReportService_GetBySubmissionID_NotFound(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	mockAnalysisRepo := new(MockAnalysisRepo)
	mockSubmissionRepo := new(MockSubmissionRepo)
	mockReportRepo := new(MockReportRepo)
	mockPDFGen := new(MockPDFGenerator)
	mockStorage := new(MockStorageClient)

	submissionID := uuid.New().String()

	mockReportRepo.On("GetBySubmissionID", ctx, submissionID).
		Return(nil, fmt.Errorf("report not found"))

	svc := report.NewService(
		mockReportRepo,
		mockAnalysisRepo,
		mockSubmissionRepo,
		mockPDFGen,
		mockStorage,
		logger,
	)

	// Execute
	result, err := svc.GetBySubmissionID(ctx, submissionID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)

	mockReportRepo.AssertExpectations(t)
}
