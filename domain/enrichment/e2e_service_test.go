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

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// TestE2EEnrichmentService tests the enrichment service end-to-end
// This simulates what happens when a company is created and enrichment runs
// Run with: go test -v -tags=integration -timeout=5m ./domain/enrichment/... -run TestE2EEnrichmentService
func TestE2EEnrichmentService(t *testing.T) {
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("Failed to load .env: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Fatal("OPENAI_API_KEY not set")
	}

	// Create LLM client (same as main.go)
	llmClient := llm.NewClientWithBaseURL(apiKey, "https://openrouter.ai/api/v1")

	// Create enrichment service (same as main.go)
	svc := NewService(llmClient)

	// Test case: New company submission (like frontend would send)
	testCases := []struct {
		name     string
		company  *CompanyInput
		checkFor []string // Strings that should appear in results
	}{
		{
			name: "Well-known company (Nubank)",
			company: &CompanyInput{
				ID:       uuid.New(),
				Name:     "Nubank",
				Industry: strPtr("Fintech"),
				Location: strPtr("São Paulo, Brasil"),
			},
			checkFor: []string{"fintech", "digital", "bank"},
		},
		{
			name: "Smaller company (Conta Azul)",
			company: &CompanyInput{
				ID:       uuid.New(),
				Name:     "Conta Azul",
				Industry: strPtr("SaaS / Contabilidade"),
				Location: strPtr("Joinville, SC"),
			},
			checkFor: []string{"erp", "contabil", "software"},
		},
	}

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("🧪 E2E ENRICHMENT SERVICE TEST")
	fmt.Println("   Models: " + ModelStage1Search + " → " + ModelStage2Synthesis)
	fmt.Println(strings.Repeat("=", 70))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Printf("\n🏢 Testing: %s\n", tc.company.Name)

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			start := time.Now()
			result, err := svc.EnrichCompany(ctx, tc.company)
			elapsed := time.Since(start)

			if err != nil {
				t.Fatalf("EnrichCompany failed: %v", err)
			}

			fmt.Printf("   ✅ Completed in %v\n", elapsed.Round(time.Second))
			fmt.Printf("   Confidence: %.0f%%\n", result.ConfidenceScore)
			fmt.Printf("   Competitors: %d\n", len(result.Competitors))
			fmt.Printf("   Strengths: %d, Weaknesses: %d\n", len(result.Strengths), len(result.Weaknesses))
			fmt.Printf("   Sources: %d\n", len(result.Sources))

			// Validate critical fields are populated
			if len(result.Competitors) == 0 {
				t.Error("FAIL: Competitors array is empty")
			}
			if len(result.Strengths) == 0 {
				t.Error("FAIL: Strengths array is empty")
			}
			if len(result.Weaknesses) == 0 {
				t.Error("FAIL: Weaknesses array is empty")
			}

			// Check for expected keywords
			resultJSON := fmt.Sprintf("%v %v %v %v",
				result.Industry, result.BusinessModel, result.Competitors, result.Strengths)
			resultLower := strings.ToLower(resultJSON)

			for _, keyword := range tc.checkFor {
				if !strings.Contains(resultLower, keyword) {
					t.Logf("WARNING: Expected keyword '%s' not found in result", keyword)
				}
			}

			// Print sample data
			fmt.Printf("\n   Sample Competitors:\n")
			for i, c := range result.Competitors {
				if i < 3 {
					fmt.Printf("     - %s\n", c)
				}
			}

			fmt.Printf("\n   Sample Strengths:\n")
			for i, s := range result.Strengths {
				if i < 3 {
					fmt.Printf("     - %s\n", s)
				}
			}

			if len(result.RecentNews) > 0 {
				fmt.Printf("\n   Recent News:\n")
				for i, n := range result.RecentNews {
					if i < 2 {
						fmt.Printf("     - %s\n", n)
					}
				}
			}
		})
	}
}
