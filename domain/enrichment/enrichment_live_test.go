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

// TestEnrichmentLive tests the two-stage enrichment pipeline with real API calls
// Run with: go test -v -tags=integration -timeout=5m ./domain/enrichment/... -run TestEnrichmentLive
func TestEnrichmentLive(t *testing.T) {
	// Load .env from project root
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("Failed to load .env: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Fatal("OPENAI_API_KEY not set")
	}

	// Create LLM client
	llmClient := llm.NewClientWithBaseURL(apiKey, "https://openrouter.ai/api/v1")

	// Create enrichment service
	svc := NewService(llmClient)

	// Test company - use a well-known Brazilian company
	company := &CompanyInput{
		ID:       uuid.New(),
		Name:     "Nubank",
		Industry: strPtr("Fintech"),
		Location: strPtr("São Paulo, Brasil"),
	}

	fmt.Printf("\n🚀 Testing enrichment for: %s\n", company.Name)
	fmt.Printf("   Stage 1 model: %s\n", ModelStage1Search)
	fmt.Printf("   Stage 2 model: %s\n", ModelStage2Synthesis)
	fmt.Println("   This may take 1-2 minutes...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	start := time.Now()
	result, err := svc.EnrichCompany(ctx, company)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("EnrichCompany failed: %v", err)
	}

	fmt.Printf("\n✅ Enrichment completed in %v\n\n", elapsed)

	// Print key results
	fmt.Println("=== KEY RESULTS ===")
	fmt.Printf("Confidence Score: %.0f%%\n", result.ConfidenceScore)
	fmt.Printf("Sources: %d\n", len(result.Sources))

	if result.Website != nil {
		fmt.Printf("Website: %s\n", *result.Website)
	}
	if result.Industry != nil {
		fmt.Printf("Industry: %s\n", *result.Industry)
	}
	if result.FoundationYear != nil {
		fmt.Printf("Founded: %s\n", *result.FoundationYear)
	}
	if result.EmployeesRange != nil {
		fmt.Printf("Employees: %s\n", *result.EmployeesRange)
	}
	if result.BusinessModel != nil {
		fmt.Printf("Business Model: %s\n", *result.BusinessModel)
	}

	fmt.Println("\n=== COMPETITORS ===")
	if len(result.Competitors) > 0 {
		for i, c := range result.Competitors {
			fmt.Printf("  %d. %s\n", i+1, c)
		}
	} else {
		fmt.Println("  ⚠️ No competitors found!")
	}

	fmt.Println("\n=== STRENGTHS ===")
	if len(result.Strengths) > 0 {
		for i, s := range result.Strengths {
			fmt.Printf("  %d. %s\n", i+1, s)
		}
	} else {
		fmt.Println("  ⚠️ No strengths found!")
	}

	fmt.Println("\n=== WEAKNESSES ===")
	if len(result.Weaknesses) > 0 {
		for i, w := range result.Weaknesses {
			fmt.Printf("  %d. %s\n", i+1, w)
		}
	} else {
		fmt.Println("  ⚠️ No weaknesses found!")
	}

	fmt.Println("\n=== RECENT NEWS ===")
	if len(result.RecentNews) > 0 {
		for i, n := range result.RecentNews {
			fmt.Printf("  %d. %s\n", i+1, n)
		}
	} else {
		fmt.Println("  (none)")
	}

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

	// Print full JSON for inspection
	fmt.Println("\n=== FULL JSON OUTPUT ===")
	jsonBytes, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(jsonBytes))
}

func strPtr(s string) *string {
	return &s
}
