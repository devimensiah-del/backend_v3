// +build integration

package analysis

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"backend_v3/config"
	"backend_v3/domain/challenge"
	"backend_v3/domain/company"
	"backend_v3/domain/framework"
	"backend_v3/domain/submission"
	"backend_v3/llm"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

// TestRunAnalysisIntegration tests the full analysis workflow with real OpenRouter API calls
// Run with: go test -v -tags=integration -timeout=10m ./domain/analysis/... -run TestRunAnalysisIntegration
func TestRunAnalysisIntegration(t *testing.T) {
	// Skip if DATABASE_URL not set
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set - skipping integration test")
	}

	// Skip if OPENAI_API_KEY not set (OpenRouter key)
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set - skipping integration test")
	}

	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Connect to DB
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Initialize repositories
	submissionRepo := submission.NewPostgresRepository(db)
	companyRepo := company.NewPostgresRepository(db)
	challengeRepo := challenge.NewPostgresRepository(db)
	analysisRepo := NewPostgresRepository(db)
	frameworkRepo := framework.NewPostgresRepository(db)

	// Initialize LLM client
	llmClient := llm.NewClient(apiKey, &logger)

	// Initialize services
	companyService := company.NewService(companyRepo, db, nil, logger, 300)
	challengeService := challenge.NewService(challengeRepo)
	frameworkService := framework.NewService(frameworkRepo)

	// Get frameworks config
	frameworks, err := frameworkService.GetFrameworksMap(context.Background())
	if err != nil {
		// Use defaults if not configured
		frameworks = cfg.GetFrameworksMap()
	}

	// Initialize analysis service
	analysisService := NewService(
		analysisRepo,
		submissionRepo,
		nil, // macroRepo not needed for this test
		companyService,
		frameworks,
		llmClient,
		cfg.GetSynthesisModel(),
		logger,
	)

	ctx := context.Background()

	// 1. Create test submission
	t.Log("Creating test submission...")
	testSubmission := &submission.Submission{
		ID:                uuid.New(),
		CompanyName:       "Integration Test Company " + time.Now().Format("150405"),
		BusinessChallenge: "How can we grow our market share by 50% in the B2B SaaS market?",
		ChallengeCategory: stringPtr("growth"),
		ChallengeType:     stringPtr("growth_organic"),
		ContactEmail:      "test@integration.com",
		ContactName:       "Integration Test",
		Status:            "received",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	err = submissionRepo.Create(ctx, testSubmission)
	if err != nil {
		t.Fatalf("Failed to create submission: %v", err)
	}
	t.Logf("Created submission: %s", testSubmission.ID)

	// 2. Create test company
	t.Log("Creating test company...")
	testCompany := &company.Company{
		ID:               uuid.New(),
		SubmissionID:     &testSubmission.ID,
		Name:             testSubmission.CompanyName,
		Industry:         stringPtr("Technology"),
		CompanySize:      stringPtr("50-200 employees"),
		EnrichmentStatus: stringPtr("completed"),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err = companyRepo.Create(ctx, testCompany)
	if err != nil {
		t.Fatalf("Failed to create company: %v", err)
	}
	t.Logf("Created company: %s", testCompany.ID)

	// 3. Create test challenge
	t.Log("Creating test challenge...")
	testChallenge := &challenge.Challenge{
		ID:                uuid.New(),
		CompanyID:         testCompany.ID,
		ChallengeCategory: "growth",
		ChallengeType:     "growth_organic",
		BusinessChallenge: testSubmission.BusinessChallenge,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	err = challengeService.Create(ctx, testChallenge)
	if err != nil {
		t.Fatalf("Failed to create challenge: %v", err)
	}
	t.Logf("Created challenge: %s", testChallenge.ID)

	// 4. Run analysis workflow
	t.Log("Running analysis workflow (this may take several minutes)...")
	startTime := time.Now()

	analysis, err := analysisService.RunAnalysis(
		ctx,
		testSubmission.ID.String(),
		testCompany.ID.String(),
		testChallenge.ID,
	)

	elapsed := time.Since(startTime)
	t.Logf("Analysis completed in %v", elapsed)

	if err != nil {
		t.Fatalf("RunAnalysis failed: %v", err)
	}

	// 5. Verify analysis results
	t.Log("Verifying analysis results...")

	// Check status
	if analysis.Status != string(StatusCompleted) {
		t.Errorf("Expected status 'completed', got '%s'", analysis.Status)
	}

	// Check framework results
	frameworkCodes := []string{
		FrameworkPESTEL,
		FrameworkPorter,
		FrameworkSWOT,
		FrameworkTAMSAMSOM,
		FrameworkBenchmarking,
		FrameworkBlueOcean,
		FrameworkGrowthHacking,
		FrameworkScenarios,
		FrameworkDecisionMatrix,
		FrameworkOKRs,
		FrameworkBSC,
		FrameworkSynthesis,
	}

	for _, code := range frameworkCodes {
		if !analysis.HasFramework(code) {
			t.Errorf("Missing framework: %s", code)
		} else {
			// Get raw data and check it's not empty
			data := analysis.FrameworkResults[code]
			if len(data) < 10 {
				t.Errorf("Framework %s has insufficient data (len=%d)", code, len(data))
			} else {
				t.Logf("✓ Framework %s: %d bytes", code, len(data))
			}
		}
	}

	// Print synthesis summary
	var synthesis AnalysisSynthesis
	if err := analysis.GetFramework(FrameworkSynthesis, &synthesis); err == nil {
		t.Logf("Executive Summary: %s", truncate(synthesis.ExecutiveSummary, 200))
	}

	// Print all framework keys for verification
	t.Log("\n=== All stored frameworks ===")
	allFrameworks := analysis.GetAllFrameworks()
	for code, data := range allFrameworks {
		t.Logf("  %s: %d bytes", code, len(data))
	}

	t.Logf("\n✅ Integration test passed! Analysis ID: %s", analysis.ID)
	t.Logf("Total frameworks: %d", len(allFrameworks))
	t.Logf("Processing time: %dms", analysis.ProcessingTimeMs)
}

func stringPtr(s string) *string {
	return &s
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
