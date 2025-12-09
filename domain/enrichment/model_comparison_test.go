//go:build integration
// +build integration

package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"backend_v3/llm"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// TestModelComparison tests different model combinations
// Run with: go test -v -tags=integration -timeout=15m ./domain/enrichment/... -run TestModelComparison
func TestModelComparison(t *testing.T) {
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("Failed to load .env: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Fatal("OPENAI_API_KEY not set")
	}

	llmClient := llm.NewClientWithBaseURL(apiKey, "https://openrouter.ai/api/v1")

	// Test company - smaller/less known for harder test
	company := &CompanyInput{
		ID:       uuid.New(),
		Name:     "Conta Azul",
		Industry: strPtr("SaaS / Contabilidade"),
		Location: strPtr("Joinville, SC"),
	}

	combinations := []struct {
		name       string
		stage1     string
		fallback1  string
		stage2     string
		fallback2  string
		singleCall bool // true = use stage1 model only for everything
	}{
		{
			name:      "1. Perplexity + Sonnet 4.5 (current)",
			stage1:    "perplexity/sonar-pro",
			fallback1: "perplexity/sonar",
			stage2:    "anthropic/claude-sonnet-4.5",
			fallback2: "anthropic/claude-sonnet-4",
		},
		{
			name:       "2. Opus 4.5 :online (single call)",
			stage1:     "anthropic/claude-opus-4.5:online",
			fallback1:  "anthropic/claude-sonnet-4.5:online",
			singleCall: true,
		},
		{
			name:      "3. Perplexity + Opus 4.5 (two-stage)",
			stage1:    "perplexity/sonar-pro",
			fallback1: "perplexity/sonar",
			stage2:    "anthropic/claude-opus-4.5",
			fallback2: "anthropic/claude-sonnet-4.5",
		},
		{
			name:       "4. Sonnet 4.5 :online (single call, cheaper)",
			stage1:     "anthropic/claude-sonnet-4.5:online",
			fallback1:  "anthropic/claude-sonnet-4:online",
			singleCall: true,
		},
	}

	results := make([]TestResult, 0, len(combinations))

	for _, combo := range combinations {
		t.Run(combo.name, func(t *testing.T) {
			fmt.Printf("\n\n{'='*60}\n")
			fmt.Printf("🧪 TESTING: %s\n", combo.name)
			fmt.Printf("{'='*60}\n")

			var result *EnrichedCompanyData
			var err error
			var elapsed time.Duration
			var totalCost float64

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			start := time.Now()

			if combo.singleCall {
				// Single model call with web search
				result, totalCost, err = testSingleCall(ctx, llmClient, company, combo.stage1, combo.fallback1)
			} else {
				// Two-stage approach
				result, totalCost, err = testTwoStage(ctx, llmClient, company, combo.stage1, combo.fallback1, combo.stage2, combo.fallback2)
			}

			elapsed = time.Since(start)

			if err != nil {
				fmt.Printf("❌ FAILED: %v\n", err)
				results = append(results, TestResult{
					Name:    combo.name,
					Success: false,
					Error:   err.Error(),
				})
				return
			}

			// Score the results
			score := scoreResult(result)

			testResult := TestResult{
				Name:             combo.name,
				Success:          true,
				Duration:         elapsed,
				Cost:             totalCost,
				ConfidenceScore:  result.ConfidenceScore,
				CompetitorsCount: len(result.Competitors),
				StrengthsCount:   len(result.Strengths),
				WeaknessesCount:  len(result.Weaknesses),
				NewsCount:        len(result.RecentNews),
				SourcesCount:     len(result.Sources),
				QualityScore:     score,
			}
			results = append(results, testResult)

			fmt.Printf("\n✅ SUCCESS in %v (cost: $%.4f)\n", elapsed, totalCost)
			fmt.Printf("   Confidence: %.0f%%\n", result.ConfidenceScore)
			fmt.Printf("   Competitors: %d\n", len(result.Competitors))
			fmt.Printf("   Strengths: %d, Weaknesses: %d\n", len(result.Strengths), len(result.Weaknesses))
			fmt.Printf("   News: %d, Sources: %d\n", len(result.RecentNews), len(result.Sources))
			fmt.Printf("   Quality Score: %d/100\n", score)

			// Print competitors for comparison
			fmt.Println("\n   Competitors found:")
			for i, c := range result.Competitors {
				if i < 5 {
					fmt.Printf("     %d. %s\n", i+1, c)
				}
			}
			if len(result.Competitors) > 5 {
				fmt.Printf("     ... and %d more\n", len(result.Competitors)-5)
			}
		})
	}

	// Print comparison summary
	fmt.Printf("\n\n{'='*80}\n")
	fmt.Printf("📊 COMPARISON SUMMARY\n")
	fmt.Printf("{'='*80}\n\n")
	fmt.Printf("%-45s %8s %8s %6s %6s %6s %6s\n", "Model Combination", "Time", "Cost", "Conf", "Comp", "SWOT", "Score")
	fmt.Printf("%s\n", "─────────────────────────────────────────────────────────────────────────────────")

	for _, r := range results {
		if r.Success {
			fmt.Printf("%-45s %8s $%6.4f %5.0f%% %6d %6d %6d\n",
				r.Name,
				r.Duration.Round(time.Second),
				r.Cost,
				r.ConfidenceScore,
				r.CompetitorsCount,
				r.StrengthsCount+r.WeaknessesCount,
				r.QualityScore,
			)
		} else {
			fmt.Printf("%-45s FAILED: %s\n", r.Name, r.Error)
		}
	}

	// Find best
	var best TestResult
	for _, r := range results {
		if r.Success && r.QualityScore > best.QualityScore {
			best = r
		}
	}
	fmt.Printf("\n🏆 BEST: %s (Score: %d, Cost: $%.4f)\n", best.Name, best.QualityScore, best.Cost)
}

type TestResult struct {
	Name             string
	Success          bool
	Error            string
	Duration         time.Duration
	Cost             float64
	ConfidenceScore  float64
	CompetitorsCount int
	StrengthsCount   int
	WeaknessesCount  int
	NewsCount        int
	SourcesCount     int
	QualityScore     int
}

func testSingleCall(ctx context.Context, client *llm.Client, company *CompanyInput, model, fallback string) (*EnrichedCompanyData, float64, error) {
	prompt := buildSingleCallPrompt(company)

	req := llm.Request{
		Model: model,
		SystemPrompt: `Você é um analista de inteligência de mercado sênior. Pesquise e analise a empresa.

REGRAS:
1. Retorne APENAS JSON válido
2. SEMPRE preencha competitors, strengths, weaknesses (NUNCA arrays vazios)
3. Use web search para dados atualizados
4. Inclua notícias recentes (2024)
5. Liste fontes consultadas`,
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		Temperature: 0.3,
		MaxTokens:   4000,
	}

	resp, err := client.CallWithFallback(ctx, &req, fallback)
	if err != nil {
		return nil, 0, err
	}

	result, err := parseEnrichmentJSON(resp.Content)
	if err != nil {
		return nil, 0, fmt.Errorf("parse error: %w (response: %.200s...)", err, resp.Content)
	}

	// Use actual cost from response
	return result, resp.CostUSD, nil
}

func testTwoStage(ctx context.Context, client *llm.Client, company *CompanyInput,
	stage1Model, stage1Fallback, stage2Model, stage2Fallback string) (*EnrichedCompanyData, float64, error) {

	var totalCost float64

	// Stage 1: Search
	searchPrompt := fmt.Sprintf(`Pesquise informações sobre: %s
Setor: %s
Local: %s

ENCONTRE: CNPJ, fundação, funcionários, produtos, concorrentes, notícias 2024, forças, fraquezas, executivos.
Cite fontes.`, company.Name, strVal(company.Industry), strVal(company.Location))

	searchReq := llm.Request{
		Model:        stage1Model,
		SystemPrompt: "Você é um pesquisador. Encontre dados factuais sobre empresas brasileiras. Cite fontes.",
		Messages:     []llm.Message{{Role: "user", Content: searchPrompt}},
		Temperature:  0.2,
		MaxTokens:    3000,
	}

	searchResp, err := client.CallWithFallback(ctx, &searchReq, stage1Fallback)
	if err != nil {
		return nil, 0, fmt.Errorf("stage1 failed: %w", err)
	}
	totalCost += searchResp.CostUSD

	// Stage 2: Synthesis
	synthesisPrompt := buildSynthesisPromptForTest(company, searchResp.Content)

	synthesisReq := llm.Request{
		Model: stage2Model,
		SystemPrompt: `Analise os dados e retorne JSON estruturado.
REGRAS:
1. APENAS JSON válido
2. SEMPRE preencha competitors, strengths, weaknesses
3. Se dados faltarem, infira do setor`,
		Messages:    []llm.Message{{Role: "user", Content: synthesisPrompt}},
		Temperature: 0.3,
		MaxTokens:   4000,
	}

	synthesisResp, err := client.CallWithFallback(ctx, &synthesisReq, stage2Fallback)
	if err != nil {
		return nil, totalCost, fmt.Errorf("stage2 failed: %w", err)
	}
	totalCost += synthesisResp.CostUSD

	result, err := parseEnrichmentJSON(synthesisResp.Content)
	if err != nil {
		return nil, totalCost, fmt.Errorf("parse error: %w", err)
	}

	return result, totalCost, nil
}

func buildSingleCallPrompt(company *CompanyInput) string {
	return fmt.Sprintf(`# Pesquisa e Análise Estratégica: %s

Setor: %s
Local: %s

## PESQUISE E RETORNE JSON:

{
  "cnpj": "XX.XXX.XXX/XXXX-XX",
  "website": "https://...",
  "industry": "Setor",
  "foundation_year": "YYYY",
  "employees_range": "XX-XX",
  "business_model": "Descrição",
  "main_products": ["Produto 1", "Produto 2"],
  "recent_news": ["2024: Notícia 1", "2024: Notícia 2"],
  "key_executives": ["Nome - Cargo"],
  "competitors": ["Concorrente 1", "Concorrente 2", "Concorrente 3"],
  "competitor_details": ["Desc 1", "Desc 2"],
  "strengths": ["Força 1", "Força 2", "Força 3"],
  "weaknesses": ["Fraqueza 1", "Fraqueza 2"],
  "opportunities": ["Oportunidade 1"],
  "threats": ["Ameaça 1"],
  "industry_trends": ["Tendência 1"],
  "confidence_score": 80,
  "sources": ["fonte1.com", "fonte2.com"]
}

IMPORTANTE: Pesquise dados REAIS e ATUAIS. NUNCA retorne arrays vazios.`,
		company.Name, strVal(company.Industry), strVal(company.Location))
}

func buildSynthesisPromptForTest(company *CompanyInput, rawData string) string {
	return fmt.Sprintf(`# Sintetize em JSON: %s

## DADOS DA PESQUISA:
%s

## RETORNE JSON:
{
  "cnpj": "...", "website": "...", "industry": "...", "foundation_year": "...",
  "employees_range": "...", "business_model": "...", "main_products": [...],
  "recent_news": [...], "key_executives": [...],
  "competitors": [...], "competitor_details": [...],
  "strengths": [...], "weaknesses": [...],
  "opportunities": [...], "threats": [...],
  "industry_trends": [...], "confidence_score": 80, "sources": [...]
}`, company.Name, rawData)
}

// parseEnrichmentJSON delegates to the service's flexible parser
func parseEnrichmentJSON(content string) (*EnrichedCompanyData, error) {
	// Use an instance of the service to access parseEnrichmentResponse
	svc := &Service{}
	return svc.parseEnrichmentResponse(content)
}

func scoreResult(r *EnrichedCompanyData) int {
	score := 0

	// Confidence (max 20)
	score += int(r.ConfidenceScore / 5)

	// Competitors (max 20)
	if len(r.Competitors) >= 5 {
		score += 20
	} else {
		score += len(r.Competitors) * 4
	}

	// Strengths (max 15)
	if len(r.Strengths) >= 5 {
		score += 15
	} else {
		score += len(r.Strengths) * 3
	}

	// Weaknesses (max 15)
	if len(r.Weaknesses) >= 3 {
		score += 15
	} else {
		score += len(r.Weaknesses) * 5
	}

	// News (max 10)
	if len(r.RecentNews) >= 3 {
		score += 10
	} else {
		score += len(r.RecentNews) * 3
	}

	// Sources (max 10)
	if len(r.Sources) >= 3 {
		score += 10
	} else {
		score += len(r.Sources) * 3
	}

	// Bonus for key fields
	if r.CNPJ != nil && *r.CNPJ != "" {
		score += 5
	}
	if r.FoundationYear != nil && *r.FoundationYear != "" {
		score += 5
	}

	return score
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
