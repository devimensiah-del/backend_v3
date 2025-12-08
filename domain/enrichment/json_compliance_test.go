// +build integration

package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"backend_v3/llm"

	"github.com/joho/godotenv"
)

// TestJSONCompliance tests which model follows flat JSON schema better
// Run with: go test -v -tags=integration -timeout=5m ./domain/enrichment/... -run TestJSONCompliance
func TestJSONCompliance(t *testing.T) {
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("Failed to load .env: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Fatal("OPENAI_API_KEY not set")
	}

	client := llm.NewClientWithBaseURL(apiKey, "https://openrouter.ai/api/v1")

	// Sample research data
	researchData := `## Empresa: TechCorp Brasil
- CNPJ: 12.345.678/0001-90
- Fundação: 2015
- Funcionários: 200-500
- Sede: São Paulo, SP
- Setor: SaaS
- Concorrentes: CompA, CompB, CompC
- Notícias 2024: Lançou produto X em janeiro, Expandiu para México em março
- Forças: Tecnologia proprietária, Equipe experiente
- Fraquezas: Pouca presença internacional`

	prompt := fmt.Sprintf(`# Extraia dados da pesquisa abaixo

## DADOS:
%s

## RETORNE JSON PLANO (sem objetos aninhados):
{
  "cnpj": "string",
  "foundation_year": "string",
  "employees_range": "string",
  "headquarters": "string",
  "industry": "string",
  "competitors": ["string", "string"],
  "recent_news": ["string", "string"],
  "strengths": ["string", "string"],
  "weaknesses": ["string"],
  "confidence_score": 85
}

REGRAS:
- Arrays DEVEM conter APENAS strings
- NÃO use objetos aninhados
- ✅ CORRETO: "competitors": ["CompA", "CompB"]
- ❌ ERRADO: "competitors": [{"name": "CompA"}]`, researchData)

	systemPrompt := `Extraia dados e retorne JSON PLANO. Arrays devem conter APENAS strings, NUNCA objetos.`

	models := []struct {
		name  string
		model string
	}{
		{"Sonnet 4.5", "anthropic/claude-sonnet-4.5"},
		{"Opus 4.5", "anthropic/claude-opus-4.5"},
	}

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("🧪 JSON COMPLIANCE TEST")
	fmt.Println("   Testing: Which model follows flat JSON schema better?")
	fmt.Println(strings.Repeat("=", 70))

	for _, m := range models {
		fmt.Printf("\n🧪 Testing: %s\n", m.name)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

		req := llm.Request{
			Model:        m.model,
			SystemPrompt: systemPrompt,
			Messages:     []llm.Message{{Role: "user", Content: prompt}},
			Temperature:  0.2,
			MaxTokens:    1000,
		}

		resp, err := client.Call(ctx, &req)
		cancel()

		if err != nil {
			fmt.Printf("   ❌ ERROR: %v\n", err)
			continue
		}

		answer := strings.TrimSpace(resp.Content)

		// Extract JSON
		jsonStr := answer
		if start := strings.Index(jsonStr, "{"); start != -1 {
			if end := strings.LastIndex(jsonStr, "}"); end != -1 {
				jsonStr = jsonStr[start : end+1]
			}
		}

		// Try to parse
		var result map[string]interface{}
		parseErr := json.Unmarshal([]byte(jsonStr), &result)

		if parseErr != nil {
			fmt.Printf("   ❌ JSON PARSE ERROR: %v\n", parseErr)
			fmt.Printf("   Response: %.200s...\n", answer)
			continue
		}

		// Check for nested objects in arrays
		hasNestedObjects := false
		problematicFields := []string{}

		arrayFields := []string{"competitors", "recent_news", "strengths", "weaknesses"}
		for _, field := range arrayFields {
			if arr, ok := result[field].([]interface{}); ok {
				for _, item := range arr {
					if _, isObject := item.(map[string]interface{}); isObject {
						hasNestedObjects = true
						problematicFields = append(problematicFields, field)
						break
					}
				}
			}
		}

		if hasNestedObjects {
			fmt.Printf("   ❌ HAS NESTED OBJECTS in: %v\n", problematicFields)
		} else {
			fmt.Printf("   ✅ FLAT JSON - No nested objects!\n")
		}

		// Print key fields
		if competitors, ok := result["competitors"]; ok {
			fmt.Printf("   competitors: %v\n", competitors)
		}
		if news, ok := result["recent_news"]; ok {
			fmt.Printf("   recent_news: %v\n", news)
		}

		fmt.Printf("   Cost: $%.4f\n", resp.CostUSD)
	}
}
