package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"backend_v3/config"
	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/submission"
	"backend_v3/llm"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Info().Msg("🚀 Starting Coimma Integration Test")

	// Configuration
	cfg := &config.Config{
		OpenRouterAPIKey: "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",
		Frameworks: map[string]config.FrameworkConfig{
			"enrichment": {
				Model:       "google/gemini-2.0-flash-001",
				Temperature: 0.5,
				MaxTokens:   8000,
			},
			"pestel": {
				Model:       "openai/o3-mini",
				Temperature: 0.2,
				MaxTokens:   1500,
			},
			"porter": {
				Model:       "openai/gpt-4o",
				Temperature: 0.3,
				MaxTokens:   1500,
			},
			"tam_sam_som": {
				Model:       "openai/o3-mini",
				Temperature: 0.1,
				MaxTokens:   1200,
			},
			"swot": {
				Model:       "openai/gpt-4o-mini",
				Temperature: 0.4,
				MaxTokens:   1500,
			},
			"benchmarking": {
				Model:       "openai/gpt-4o-mini",
				Temperature: 0.35,
				MaxTokens:   1500,
			},
			"blue_ocean": {
				Model:       "openai/gpt-4o",
				Temperature: 0.7,
				MaxTokens:   1500,
			},
			"growth_hacking": {
				Model:       "openai/gpt-4o-mini",
				Temperature: 0.6,
				MaxTokens:   1500,
			},
			"scenarios": {
				Model:       "openai/gpt-4o",
				Temperature: 0.6,
				MaxTokens:   1800,
			},
			"okrs": {
				Model:       "openai/o3-mini",
				Temperature: 0.25,
				MaxTokens:   1500,
			},
			"bsc": {
				Model:       "openai/gpt-4o-mini",
				Temperature: 0.35,
				MaxTokens:   1500,
			},
			"decision_matrix": {
				Model:       "openai/o3-mini",
				Temperature: 0.2,
				MaxTokens:   1500,
			},
			"synthesis": {
				Model:       "openai/gpt-4o",
				Temperature: 0.4,
				MaxTokens:   3000,
			},
		},
	}

	// Initialize LLM Client
	llmClient := llm.NewClient(cfg.OpenRouterAPIKey)

	// Create mock repositories (in-memory for testing)
	ctx := context.Background()

	// Create Coimma Submission
	log.Info().Msg("📝 Creating Coimma submission...")

	website := "https://coimma.com.br/"
	industry := "Equipamentos Agropecuários / Sistemas de Pesagem"

	sub := &submission.Submission{
		ID:                uuid.New(),
		CompanyName:       "Coimma",
		CompanyWebsite:    &website,
		CompanyIndustry:   &industry,
		BusinessChallenge: `Sou CEO da Coimma. Somos fabricantes de equipamentos de manejo para gado, balanças bovinas eletronica, balanças rodoviárias, automação de balancas rodoviárias e balanças de fluxo por batelada para grãos.

Atualmente 70% do nosso faturamento é proveniente do mercado pecuária. Os outros 30% sao provenientes de segmentos diversos destacando-se grãos e armazenamento. A prestação de serviços de calibração e manutenção representa 5% do faturamento.

Nossas competências são fabricação de equipamentos, sistemas de pesagem e eletrônica elabore um plano de de negócios para nossa companhia em duas fases:

Fase 1 (Imediata): Expandir as nossas vendas com nosso portfólio atual de produtos e aproveitando nossa estrutura atual de vendas e assistência técnica.

Fase 2 (Desenvolvimento): Desenvolver novos produtos para o segmento industrial que requer investimentos em estrutura de vendas e assistência técnica.`,
		ContactName:  "CEO Coimma",
		ContactEmail: "test@coimma.com.br",
		Status:       submission.StatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	log.Info().Str("submission_id", sub.ID.String()).Msg("✅ Submission created")

	// Create enrichment service (mock repo)
	enrichRepo := newMockEnrichmentRepo()
	subRepo := newMockSubmissionRepo(sub)
	enrichSvc := enrichment.NewService(enrichRepo, subRepo, llmClient, "google/gemini-2.0-flash-001")
	enrichSvc.SetEnrichmentConfig(cfg.Frameworks["enrichment"])

	// Run Enrichment
	log.Info().Msg("🔍 Starting enrichment workflow...")
	startEnrich := time.Now()

	enrichResult, err := enrichSvc.EnrichSubmission(ctx, sub.ID)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Enrichment failed")
	}

	enrichDuration := time.Since(startEnrich)
	log.Info().Dur("duration", enrichDuration).Msg("✅ Enrichment completed")

	// Create analysis service (mock repo)
	analysisRepo := newMockAnalysisRepo()
	analysisSvc := analysis.NewService(analysisRepo, llmClient, log.Logger, "", "")
	analysisSvc.SetFrameworks(cfg.Frameworks)

	// Run Analysis
	log.Info().Msg("📊 Starting analysis workflow...")
	startAnalysis := time.Now()

	enrichedData := enrichResult.EnrichedData
	analysisResult, err := analysisSvc.RunAnalysis(ctx, sub.ID.String(), enrichResult.ID.String(), enrichedData)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Analysis failed")
	}

	analysisDuration := time.Since(startAnalysis)
	log.Info().Dur("duration", analysisDuration).Msg("✅ Analysis completed")

	// Generate Markdown Report
	log.Info().Msg("📄 Generating markdown report...")
	if err := generateMarkdownReport(sub, enrichResult, analysisResult); err != nil {
		log.Fatal().Err(err).Msg("❌ Markdown generation failed")
	}

	log.Info().Msg("🎉 Integration test completed successfully!")
	log.Info().
		Dur("enrichment_time", enrichDuration).
		Dur("analysis_time", analysisDuration).
		Dur("total_time", enrichDuration+analysisDuration).
		Msg("⏱️ Performance metrics")
}

// Mock Repositories for testing without database

type mockSubmissionRepo struct {
	sub *submission.Submission
}

func newMockSubmissionRepo(sub *submission.Submission) *mockSubmissionRepo {
	return &mockSubmissionRepo{sub: sub}
}

func (r *mockSubmissionRepo) GetByID(ctx context.Context, id uuid.UUID) (*submission.Submission, error) {
	return r.sub, nil
}

func (r *mockSubmissionRepo) Update(ctx context.Context, sub *submission.Submission) error {
	r.sub = sub
	return nil
}

func (r *mockSubmissionRepo) Create(ctx context.Context, sub *submission.Submission) error {
	r.sub = sub
	return nil
}

func (r *mockSubmissionRepo) GetByEmail(ctx context.Context, email string) ([]*submission.Submission, error) {
	if r.sub.ContactEmail == email {
		return []*submission.Submission{r.sub}, nil
	}
	return []*submission.Submission{}, nil
}

func (r *mockSubmissionRepo) List(ctx context.Context, opts *submission.ListOptions) ([]*submission.Submission, int, error) {
	return []*submission.Submission{r.sub}, 1, nil
}

func (r *mockSubmissionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (r *mockSubmissionRepo) GetTotalCount(ctx context.Context) (int, error) {
	return 1, nil
}

func (r *mockSubmissionRepo) GetActiveCount(ctx context.Context) (int, error) {
	return 1, nil
}

func (r *mockSubmissionRepo) GetCompletedCount(ctx context.Context) (int, error) {
	return 0, nil
}

type mockEnrichmentRepo struct {
	enrichments map[string]*enrichment.Enrichment
}

func newMockEnrichmentRepo() *mockEnrichmentRepo {
	return &mockEnrichmentRepo{enrichments: make(map[string]*enrichment.Enrichment)}
}

func (r *mockEnrichmentRepo) Create(ctx context.Context, e *enrichment.Enrichment) error {
	r.enrichments[e.ID.String()] = e
	return nil
}

func (r *mockEnrichmentRepo) GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*enrichment.Enrichment, error) {
	for _, e := range r.enrichments {
		if e.SubmissionID == submissionID {
			return e, nil
		}
	}
	return nil, fmt.Errorf("enrichment not found")
}

func (r *mockEnrichmentRepo) UpdateSystem(ctx context.Context, e *enrichment.Enrichment) error {
	r.enrichments[e.ID.String()] = e
	return nil
}

func (r *mockEnrichmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*enrichment.Enrichment, error) {
	if e, ok := r.enrichments[id.String()]; ok {
		return e, nil
	}
	return nil, fmt.Errorf("enrichment not found")
}

func (r *mockEnrichmentRepo) UpdateUser(ctx context.Context, e *enrichment.Enrichment) error {
	r.enrichments[e.ID.String()] = e
	return nil
}

type mockAnalysisRepo struct {
	analyses map[string]*analysis.Analysis
}

func newMockAnalysisRepo() *mockAnalysisRepo {
	return &mockAnalysisRepo{analyses: make(map[string]*analysis.Analysis)}
}

func (r *mockAnalysisRepo) Create(ctx context.Context, a *analysis.Analysis) error {
	r.analyses[a.ID] = a
	return nil
}

func (r *mockAnalysisRepo) Update(ctx context.Context, a *analysis.Analysis) error {
	r.analyses[a.ID] = a
	return nil
}

func (r *mockAnalysisRepo) GetByID(ctx context.Context, id string) (*analysis.Analysis, error) {
	if a, ok := r.analyses[id]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("analysis not found")
}

func (r *mockAnalysisRepo) GetBySubmissionID(ctx context.Context, submissionID string) (*analysis.Analysis, error) {
	for _, a := range r.analyses {
		if a.SubmissionID == submissionID {
			return a, nil
		}
	}
	return nil, fmt.Errorf("analysis not found")
}

func (r *mockAnalysisRepo) List(ctx context.Context, limit, offset int) ([]*analysis.Analysis, error) {
	return nil, nil
}

func (r *mockAnalysisRepo) Delete(ctx context.Context, id string) error {
	delete(r.analyses, id)
	return nil
}

func generateMarkdownReport(sub *submission.Submission, enrich *enrichment.Enrichment, anal *analysis.Analysis) error {
	var report string

	report += "# Análise Estratégica: Coimma\n\n"
	report += fmt.Sprintf("**Gerado em:** %s\n\n", time.Now().Format("02/01/2006 15:04:05"))
	report += "---\n\n"

	// Submission Info
	report += "## 📋 Informações da Empresa\n\n"
	report += fmt.Sprintf("**Nome:** %s\n\n", sub.CompanyName)
	if sub.CompanyWebsite != nil {
		report += fmt.Sprintf("**Website:** %s\n\n", *sub.CompanyWebsite)
	}
	if sub.CompanyIndustry != nil {
		report += fmt.Sprintf("**Setor:** %s\n\n", *sub.CompanyIndustry)
	}
	report += fmt.Sprintf("**Desafio de Negócio:**\n\n%s\n\n", sub.BusinessChallenge)
	report += "---\n\n"

	// Enrichment
	report += "## 🔍 Enriquecimento de Dados (Strategic Profile)\n\n"
	enrichJSON, _ := json.MarshalIndent(enrich.EnrichedData, "", "  ")
	report += "```json\n" + string(enrichJSON) + "\n```\n\n"
	report += "---\n\n"

	// Analysis
	report += "## 📊 Análise Estratégica Completa\n\n"

	report += "### Layer 1: Análise do Ambiente\n\n"

	report += "#### PESTEL Analysis\n\n"
	pestelJSON, _ := json.MarshalIndent(anal.PESTEL, "", "  ")
	report += "```json\n" + string(pestelJSON) + "\n```\n\n"

	report += "#### Porter's Five Forces\n\n"
	porterJSON, _ := json.MarshalIndent(anal.Porter, "", "  ")
	report += "```json\n" + string(porterJSON) + "\n```\n\n"

	report += "#### TAM/SAM/SOM\n\n"
	tamJSON, _ := json.MarshalIndent(anal.TamSamSom, "", "  ")
	report += "```json\n" + string(tamJSON) + "\n```\n\n"

	report += "### Layer 2: Posicionamento\n\n"

	report += "#### SWOT Analysis\n\n"
	swotJSON, _ := json.MarshalIndent(anal.SWOT, "", "  ")
	report += "```json\n" + string(swotJSON) + "\n```\n\n"

	report += "#### Benchmarking\n\n"
	benchJSON, _ := json.MarshalIndent(anal.Benchmarking, "", "  ")
	report += "```json\n" + string(benchJSON) + "\n```\n\n"

	report += "### Layer 3: Estratégia\n\n"

	report += "#### Blue Ocean Strategy (ERRC)\n\n"
	blueJSON, _ := json.MarshalIndent(anal.BlueOcean, "", "  ")
	report += "```json\n" + string(blueJSON) + "\n```\n\n"

	report += "#### Growth Hacking\n\n"
	growthJSON, _ := json.MarshalIndent(anal.GrowthHacking, "", "  ")
	report += "```json\n" + string(growthJSON) + "\n```\n\n"

	report += "#### Scenario Planning\n\n"
	scenariosJSON, _ := json.MarshalIndent(anal.Scenarios, "", "  ")
	report += "```json\n" + string(scenariosJSON) + "\n```\n\n"

	report += "### Layer 4: Execução\n\n"

	report += "#### OKRs (Objectives & Key Results)\n\n"
	okrsJSON, _ := json.MarshalIndent(anal.OKRs, "", "  ")
	report += "```json\n" + string(okrsJSON) + "\n```\n\n"

	report += "#### Balanced Scorecard\n\n"
	bscJSON, _ := json.MarshalIndent(anal.BSC, "", "  ")
	report += "```json\n" + string(bscJSON) + "\n```\n\n"

	report += "#### Decision Matrix\n\n"
	decisionJSON, _ := json.MarshalIndent(anal.DecisionMatrix, "", "  ")
	report += "```json\n" + string(decisionJSON) + "\n```\n\n"

	report += "---\n\n"
	report += "## 🎯 Síntese Executiva\n\n"
	synthesisJSON, _ := json.MarshalIndent(anal.Synthesis, "", "  ")
	report += "```json\n" + string(synthesisJSON) + "\n```\n\n"

	report += "---\n\n"
	report += fmt.Sprintf("**Processing Time:** %dms\n\n", anal.ProcessingTimeMs)
	report += "_Generated by IMENSIAH Strategic Cascade v3_\n"

	// Write to file
	os.MkdirAll("integration_tests", 0755)
	return os.WriteFile("integration_tests/coimma_strategic_analysis.md", []byte(report), 0644)
}
