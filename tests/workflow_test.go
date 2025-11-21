package tests

import (
	"context"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/submission"
	"backend_v3/infrastructure/repo/memory"
	"backend_v3/llm"
)

// TestCoimmaWorkflow_Local tests the full workflow with Coimma as test case
func TestCoimmaWorkflow_Local(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	logger.Info().Msg("🚜 STARTING LOCAL WORKFLOW: COIMMA (NO DB)")

	// ========================================================================
	// STEP 1: CREATE SUBMISSION (In-Memory)
	// ========================================================================
	subRepo := memory.NewSubmissionRepo()
	subService := submission.NewService(subRepo, logger)

	sub, _ := subService.Create(context.Background(), &submission.Submission{
		CompanyName:     "Coimma",
		Industry:        "Agronegócio",
		BusinessChallenge: "Modernizar portfólio de equipamentos pecuários para Era 4.0",
		Website:         "https://www.coimma.com.br",
	})
	logger.Info().Str("id", sub.ID).Msg("✅ Step 1: Submission Created (In-Memory)")

	// ========================================================================
	// STEP 2: ENRICHMENT (Agent Discovery)
	// ========================================================================
	logger.Info().Msg("--- STEP 2: RUNNING AGENT (Gemini 2.5 Flash) ---")

	enrichRepo := memory.NewEnrichmentRepo()

	// Use OpenRouter API key from environment
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		t.Skip("OPENROUTER_API_KEY not set, skipping test")
	}

	agent, _ := llm.NewGeminiAgent(apiKey, "gemini-2.0-flash-exp", logger)
	enrichService := enrichment.NewService(enrichRepo, agent, logger)

	enrich, err := enrichService.Enrich(context.Background(), sub.ID, sub)
	if err != nil {
		t.Fatalf("Enrichment failed: %v", err)
	}
	logger.Info().Msgf("Agent Findings:\n%+v", enrich)

	// ========================================================================
	// STEP 3: STRATEGIC CASCADE (Gemini 2.5 Pro + Sonnet 4.5)
	// ========================================================================
	logger.Info().Msg("--- STEP 3: STRATEGIC CASCADE (Gemini 2.5 Pro + Sonnet 4.5) ---")

	analysisRepo := memory.NewAnalysisRepo()

	// Framework Model: Gemini 2.5 Pro
	frameworkModel, _ := llm.NewGeminiAgent(apiKey, "gemini-2.0-flash-thinking-exp-1219", logger)

	// Synthesis Model: Claude Sonnet 4.5
	synthesisModel, _ := llm.NewClaudeAgent(apiKey, "anthropic/claude-3.5-sonnet:beta", logger)

	analysisService := analysis.NewService(analysisRepo, frameworkModel, synthesisModel, logger)

	result, err := analysisService.Analyze(context.Background(), sub.ID, enrich.ID, sub, enrich)
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	logger.Info().Msgf("✅ Analysis Complete. Executive Summary:\n%s\n%s\n%s",
		"---------------------------------------------------",
		result.Synthesis.ExecutiveSummary,
		"---------------------------------------------------")

	// ========================================================================
	// STEP 4: GENERATE HTML & PDF
	// ========================================================================
	logger.Info().Msg("--- STEP 4: GENERATING HTML & PDF ---")

	reportService := analysis.NewReportService(logger)
	htmlPath, pdfPath, err := reportService.PublishReport(context.Background(), result, sub)
	if err != nil {
		t.Fatalf("Report generation failed: %v", err)
	}

	logger.Info().Msg("🎉 WORKFLOW SUCCESS!")
	logger.Info().Msg("📂 FILES SAVED TO: ./backend_v3/tests/test_output/")
	logger.Info().Msg("   - Check 'html_pages' folder to see the design.")
	logger.Info().Msgf("   - 'PDF' Link: file://%s", pdfPath)

	// Verify HTML path exists
	if htmlPath == "" {
		t.Fatal("HTML path is empty")
	}

	// Verify PDF path exists
	if pdfPath == "" {
		t.Fatal("PDF path is empty")
	}
}
