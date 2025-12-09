//go:build integration
// +build integration

package enrichment

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"backend_v3/llm"

	"github.com/joho/godotenv"
)

// TestSelicRealTimeData tests if models return current SELIC (15% as of Dec 2024)
// This validates whether :online suffix and web search actually work
// Run with: go test -v -tags=integration -timeout=10m ./domain/enrichment/... -run TestSelicRealTimeData
func TestSelicRealTimeData(t *testing.T) {
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("Failed to load .env: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Fatal("OPENAI_API_KEY not set")
	}

	client := llm.NewClientWithBaseURL(apiKey, "https://openrouter.ai/api/v1")

	models := []struct {
		name         string
		model        string
		hasWebSearch bool
	}{
		{"Perplexity sonar-pro (native search)", "perplexity/sonar-pro", true},
		{"Claude Opus 4.5 :online", "anthropic/claude-opus-4.5:online", true},
		{"Claude Sonnet 4.5 :online", "anthropic/claude-sonnet-4.5:online", true},
		{"Claude Sonnet 4.5 (no search)", "anthropic/claude-sonnet-4.5", false},
		{"Gemini 2.5 Flash :online", "google/gemini-2.5-flash-preview:online", true},
	}

	prompt := `Qual é a taxa SELIC atual no Brasil? Responda APENAS com o número percentual, exemplo: "14.25%" ou "15%". Nada mais.`

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("🎯 SELIC REAL-TIME DATA TEST")
	fmt.Println("   Expected: 15% (current rate as of Dec 2024)")
	fmt.Println(strings.Repeat("=", 70))

	results := make([]struct {
		name      string
		model     string
		result    string
		elapsed   time.Duration
		cost      float64
		isCorrect bool
	}, 0)

	for _, m := range models {
		fmt.Printf("\n🧪 Testing: %s\n", m.name)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

		req := llm.Request{
			Model:        m.model,
			SystemPrompt: "Você responde perguntas sobre economia brasileira de forma concisa.",
			Messages:     []llm.Message{{Role: "user", Content: prompt}},
			Temperature:  0.0, // Deterministic
			MaxTokens:    50,
		}

		start := time.Now()
		resp, err := client.Call(ctx, &req)
		elapsed := time.Since(start)
		cancel()

		if err != nil {
			fmt.Printf("   ❌ ERROR: %v\n", err)
			results = append(results, struct {
				name      string
				model     string
				result    string
				elapsed   time.Duration
				cost      float64
				isCorrect bool
			}{m.name, m.model, "ERROR: " + err.Error(), elapsed, 0, false})
			continue
		}

		answer := strings.TrimSpace(resp.Content)
		isCorrect := strings.Contains(answer, "15")

		icon := "❌"
		if isCorrect {
			icon = "✅"
		}

		fmt.Printf("   %s Response: %s (%.1fs, $%.4f)\n", icon, answer, elapsed.Seconds(), resp.CostUSD)

		results = append(results, struct {
			name      string
			model     string
			result    string
			elapsed   time.Duration
			cost      float64
			isCorrect bool
		}{m.name, m.model, answer, elapsed, resp.CostUSD, isCorrect})
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("📊 RESULTS SUMMARY")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("%-40s %10s %8s %8s\n", "Model", "SELIC", "Time", "Cost")
	fmt.Println(strings.Repeat("-", 70))

	correctCount := 0
	for _, r := range results {
		status := "❌ WRONG"
		if r.isCorrect {
			status = "✅ 15%"
			correctCount++
		}
		if strings.HasPrefix(r.result, "ERROR") {
			status = "⚠️ ERROR"
		}
		fmt.Printf("%-40s %10s %7.1fs $%6.4f\n", r.name, status, r.elapsed.Seconds(), r.cost)
	}

	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("Models with correct SELIC: %d/%d\n", correctCount, len(results))
	fmt.Println()

	// Fail if none got it right
	if correctCount == 0 {
		t.Error("No model returned correct SELIC rate (15%)")
	}
}

// TestModelComparisonSimple tests enrichment quality across models
// Run with: go test -v -tags=integration -timeout=10m ./domain/enrichment/... -run TestModelComparisonSimple
func TestModelComparisonSimple(t *testing.T) {
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("Failed to load .env: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Fatal("OPENAI_API_KEY not set")
	}

	client := llm.NewClientWithBaseURL(apiKey, "https://openrouter.ai/api/v1")

	// Test with Conta Azul - less known company
	companyName := "Conta Azul"
	industry := "SaaS Contabilidade"

	prompt := fmt.Sprintf(`Pesquise sobre a empresa brasileira "%s" (%s).

Retorne APENAS JSON válido:
{
  "nome": "...",
  "cnpj": "...",
  "fundacao": "YYYY",
  "funcionarios": "XXX-YYY",
  "sede": "Cidade, Estado",
  "produtos": ["produto1", "produto2"],
  "concorrentes": ["conc1", "conc2", "conc3"],
  "forcas": ["forca1", "forca2"],
  "fraquezas": ["fraq1", "fraq2"],
  "noticias_2024": ["noticia1", "noticia2"]
}

IMPORTANTE: Liste concorrentes REAIS. Nunca arrays vazios.`, companyName, industry)

	models := []struct {
		name  string
		model string
	}{
		{"1. Perplexity → Sonnet 4.5 (two-stage)", "perplexity/sonar-pro"},
		{"2. Opus 4.5 :online (single)", "anthropic/claude-opus-4.5:online"},
		{"3. Sonnet 4.5 :online (single)", "anthropic/claude-sonnet-4.5:online"},
	}

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Printf("🏢 ENRICHMENT TEST: %s\n", companyName)
	fmt.Println(strings.Repeat("=", 70))

	for _, m := range models {
		fmt.Printf("\n🧪 %s\n", m.name)
		fmt.Printf("   Model: %s\n", m.model)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)

		var finalResp *llm.Response
		var err error
		start := time.Now()

		if m.name == "1. Perplexity → Sonnet 4.5 (two-stage)" {
			// Two-stage: Perplexity search, then Sonnet synthesis
			searchReq := llm.Request{
				Model:        "perplexity/sonar-pro",
				SystemPrompt: "Pesquise dados factuais sobre empresas brasileiras.",
				Messages:     []llm.Message{{Role: "user", Content: prompt}},
				Temperature:  0.2,
				MaxTokens:    2000,
			}
			searchResp, searchErr := client.Call(ctx, &searchReq)
			if searchErr != nil {
				fmt.Printf("   ❌ Stage 1 ERROR: %v\n", searchErr)
				cancel()
				continue
			}
			fmt.Printf("   Stage 1: %.1fs, $%.4f\n", time.Since(start).Seconds(), searchResp.CostUSD)

			// Stage 2
			synthPrompt := fmt.Sprintf("Baseado nestes dados, retorne JSON estruturado:\n\n%s\n\nJSON:", searchResp.Content)
			synthReq := llm.Request{
				Model:        "anthropic/claude-sonnet-4.5",
				SystemPrompt: "Retorne apenas JSON válido.",
				Messages:     []llm.Message{{Role: "user", Content: synthPrompt}},
				Temperature:  0.2,
				MaxTokens:    2000,
			}
			finalResp, err = client.Call(ctx, &synthReq)
		} else {
			// Single call
			req := llm.Request{
				Model:        m.model,
				SystemPrompt: "Pesquise e retorne dados em JSON.",
				Messages:     []llm.Message{{Role: "user", Content: prompt}},
				Temperature:  0.2,
				MaxTokens:    2000,
			}
			finalResp, err = client.Call(ctx, &req)
		}

		elapsed := time.Since(start)
		cancel()

		if err != nil {
			fmt.Printf("   ❌ ERROR: %v\n", err)
			continue
		}

		// Count key data points
		content := finalResp.Content
		concCount := strings.Count(content, `"conc`) + strings.Count(content, `"Conc`)
		hasNews := strings.Contains(content, "2024")
		hasCNPJ := strings.Contains(content, "CNPJ") || strings.Contains(content, "cnpj")

		fmt.Printf("   ✅ Done: %.1fs, $%.4f\n", elapsed.Seconds(), finalResp.CostUSD)
		fmt.Printf("   Competitors mentioned: ~%d\n", concCount)
		fmt.Printf("   Has 2024 news: %v\n", hasNews)
		fmt.Printf("   Has CNPJ: %v\n", hasCNPJ)

		// Print first 500 chars
		preview := content
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		fmt.Printf("   Preview: %s\n", preview)
	}
}
