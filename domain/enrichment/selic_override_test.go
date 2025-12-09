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

// TestSelicDataOverride verifies that Claude doesn't override Perplexity's real-time data
// with its own training data. This is critical for data integrity.
// Run with: go test -v -tags=integration -timeout=5m ./domain/enrichment/... -run TestSelicDataOverride
func TestSelicDataOverride(t *testing.T) {
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("Failed to load .env: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Fatal("OPENAI_API_KEY not set")
	}

	client := llm.NewClientWithBaseURL(apiKey, "https://openrouter.ai/api/v1")

	// Simulate Stage 1 output with CORRECT SELIC (15% as of Dec 2024)
	perplexityData := `## Dados Macroeconômicos Brasil (Dezembro 2024)

**Taxa SELIC**: 15,00% ao ano (última decisão do COPOM em dezembro de 2024)
**IPCA**: 4,87% acumulado 12 meses
**USD/BRL**: R$ 6,05

Fonte: Banco Central do Brasil, dezembro 2024`

	// Stage 2 synthesis prompt
	prompt := fmt.Sprintf(`# TAREFA: Extrair dados macroeconômicos

## DADOS DA PESQUISA (Perplexity - dezembro 2024):
%s

## INSTRUÇÕES:
1. Extraia os dados EXATAMENTE como aparecem no texto acima
2. NUNCA use seu conhecimento prévio para "corrigir" valores
3. Retorne APENAS JSON válido

## FORMATO:
{"selic": "X%%", "ipca": "X%%", "usd_brl": "R$ X.XX"}`, perplexityData)

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("🧪 SELIC DATA OVERRIDE TEST")
	fmt.Println("   Perplexity says: SELIC = 15% (CORRECT - Dec 2024)")
	fmt.Println("   Claude training: SELIC = 10.75% (OLD - April 2024)")
	fmt.Println("   Test: Does Claude preserve or override the data?")
	fmt.Println(strings.Repeat("=", 70))

	models := []struct {
		name  string
		model string
	}{
		{"Claude Sonnet 4.5", "anthropic/claude-sonnet-4.5"},
		{"Claude Opus 4.5", "anthropic/claude-opus-4.5"},
	}

	results := make([]struct {
		name      string
		preserved bool
		response  string
	}, 0)

	for _, m := range models {
		fmt.Printf("\n🧪 Testing: %s\n", m.name)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

		req := llm.Request{
			Model: m.model,
			SystemPrompt: `Você é um extrator de dados. Sua ÚNICA tarefa é extrair dados do texto fornecido.

REGRA ABSOLUTA: Use APENAS os dados do texto. NUNCA use seu conhecimento prévio.
Se o texto diz "SELIC = 15%", retorne 15%. NÃO "corrija" para valores antigos.`,
			Messages:    []llm.Message{{Role: "user", Content: prompt}},
			Temperature: 0.0,
			MaxTokens:   200,
		}

		resp, err := client.Call(ctx, &req)
		cancel()

		if err != nil {
			fmt.Printf("   ❌ ERROR: %v\n", err)
			continue
		}

		answer := strings.TrimSpace(resp.Content)
		preserved := strings.Contains(answer, "15")

		icon := "❌ OVERRODE (BAD)"
		if preserved {
			icon = "✅ PRESERVED (GOOD)"
		}

		fmt.Printf("   %s\n", icon)
		fmt.Printf("   Response: %s\n", answer)

		results = append(results, struct {
			name      string
			preserved bool
			response  string
		}{m.name, preserved, answer})
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("📊 RESULTS SUMMARY")
	fmt.Println(strings.Repeat("=", 70))

	preservedCount := 0
	for _, r := range results {
		status := "❌ OVERRODE"
		if r.preserved {
			status = "✅ PRESERVED"
			preservedCount++
		}
		fmt.Printf("%-20s: %s\n", r.name, status)
	}

	fmt.Printf("\nModels that preserved data: %d/%d\n", preservedCount, len(results))

	if preservedCount < len(results) {
		t.Errorf("Some models overrode Perplexity's real-time data with training data!")
	}
}
