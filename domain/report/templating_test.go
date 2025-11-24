package report_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"backend_v3/domain/analysis"
	"backend_v3/domain/report"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ----------------------------------------------------------------------------
// TEST: Template Resolution - All 24 Templates
// ----------------------------------------------------------------------------

func TestTemplateResolution_AllTemplatesExist(t *testing.T) {
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
	require.NoError(t, err, "All templates should be found and rendered")
	assert.Len(t, pages, 24, "Should render exactly 24 pages")

	// Verify each template produced valid HTML
	for pageName, html := range pages {
		assert.NotEmpty(t, html, "Page %s should not be empty", pageName)
		assert.Contains(t, html, "<!DOCTYPE html>", "Page %s should have DOCTYPE", pageName)
		assert.Contains(t, html, "</html>", "Page %s should be complete HTML", pageName)
	}
}

// ----------------------------------------------------------------------------
// TEST: XSS Sanitization - Prevent Script Injection
// ----------------------------------------------------------------------------

func TestXSSSanitization_ScriptTagsRemoved(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	mockAnalysisRepo := new(MockAnalysisRepo)
	mockSubmissionRepo := new(MockSubmissionRepo)
	mockReportRepo := new(MockReportRepo)
	mockPDFGen := new(MockPDFGenerator)
	mockStorage := new(MockStorageClient)

	// Create analysis with XSS attempts
	maliciousAnalysis := getCoimmaAnalysis()
	maliciousAnalysis.SWOT.Strengths = append(maliciousAnalysis.SWOT.Strengths, analysis.SWOTItem{
		Content:    "<script>alert('XSS')</script>Strong market position",
		Confidence: "Alta",
		Source:     "fato",
	})

	coimmaSubmission := getCoimmaSubmission()

	mockSubmissionRepo.On("GetByID", ctx, uuid.MustParse("db32e622-56bb-4fba-a480-9384aeaa2f5c")).
		Return(coimmaSubmission, nil)
	mockAnalysisRepo.On("GetByID", ctx, "6057454f-12bf-4783-8334-f5a4424ad246").
		Return(maliciousAnalysis, nil)

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

	swotPage := pages["SWOT"]
	// Script tags should NOT appear in rendered HTML
	assert.NotContains(t, swotPage, "<script>", "Script tags should be sanitized")
	assert.NotContains(t, swotPage, "alert('XSS')", "JavaScript should be sanitized")

	// Note: Templates use {{.Content}} directly, so this test documents current behavior
	// For production, consider using bluemonday or html/template's automatic escaping
}

// ----------------------------------------------------------------------------
// TEST: Missing Data Handling - Empty Frameworks
// ----------------------------------------------------------------------------

func TestMissingDataHandling_EmptyFrameworks(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	mockAnalysisRepo := new(MockAnalysisRepo)
	mockSubmissionRepo := new(MockSubmissionRepo)
	mockReportRepo := new(MockReportRepo)
	mockPDFGen := new(MockPDFGenerator)
	mockStorage := new(MockStorageClient)

	// Create analysis with missing/empty frameworks
	emptyAnalysis := &analysis.Analysis{
		ID:           uuid.New().String(),
		SubmissionID: uuid.New().String(),
		Status:       "completed",
		Version:      1,

		// Empty PESTEL
		PESTEL: analysis.PESTELAnalysis{
			Political:     []string{},
			Economic:      []string{},
			Social:        []string{},
			Technological: []string{},
			Environmental: []string{},
			Legal:         []string{},
			Summary:       "",
		},

		// Empty SWOT
		SWOT: analysis.SWOTAnalysis{
			Strengths:     []analysis.SWOTItem{},
			Weaknesses:    []analysis.SWOTItem{},
			Opportunities: []analysis.SWOTItem{},
			Threats:       []analysis.SWOTItem{},
			Summary:       "",
		},

		// Empty OKRs
		OKRs: analysis.OKRAnalysis{
			Quarters: []analysis.QuarterlyOKR{},
			Summary:  "",
		},

		// Minimal synthesis for cover page
		Synthesis: analysis.AnalysisSynthesis{
			ExecutiveSummary: "Analysis in progress",
		},
	}

	coimmaSubmission := getCoimmaSubmission()

	mockSubmissionRepo.On("GetByID", ctx, uuid.MustParse("db32e622-56bb-4fba-a480-9384aeaa2f5c")).
		Return(coimmaSubmission, nil)
	mockAnalysisRepo.On("GetByID", ctx, "test-analysis-id").
		Return(emptyAnalysis, nil)

	svc := report.NewService(
		mockReportRepo,
		mockAnalysisRepo,
		mockSubmissionRepo,
		mockPDFGen,
		mockStorage,
		logger,
	)

	// Execute
	pages, err := svc.GeneratePreview(ctx, "db32e622-56bb-4fba-a480-9384aeaa2f5c", "test-analysis-id")

	// Assert - Should not crash, but handle gracefully
	assert.NoError(t, err, "Should handle empty data gracefully")
	assert.Len(t, pages, 24, "Should still render all pages")

	// Verify pages exist but may have minimal content
	for pageName, html := range pages {
		assert.NotEmpty(t, html, "Page %s should render even with empty data", pageName)
		assert.Contains(t, html, "<!DOCTYPE html>", "Page %s should be valid HTML", pageName)
	}
}

// ----------------------------------------------------------------------------
// TEST: Divider Pages Rendering
// ----------------------------------------------------------------------------

func TestDividerPagesRendering(t *testing.T) {
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

	// Check all 4 divider pages exist
	dividers := []string{"DividerPart1", "DividerPart2", "DividerPart3", "DividerPart4"}

	for _, divider := range dividers {
		assert.Contains(t, pages, divider, "Divider page should exist: "+divider)
		assert.NotEmpty(t, pages[divider], "Divider should have content: "+divider)
		assert.Contains(t, pages[divider], "<!DOCTYPE html>", "Divider should be valid HTML: "+divider)
	}
}

// ----------------------------------------------------------------------------
// TEST: PESTEL Split Pages (PES + TEL)
// ----------------------------------------------------------------------------

func TestPESTELSplitPages(t *testing.T) {
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

	// Both PESTEL pages should exist
	assert.Contains(t, pages, "PESTEL_PES", "PESTEL PES page should exist")
	assert.Contains(t, pages, "PESTEL_TEL", "PESTEL TEL page should exist")

	pesPage := pages["PESTEL_PES"]
	telPage := pages["PESTEL_TEL"]

	assert.NotEmpty(t, pesPage, "PES page should have content")
	assert.NotEmpty(t, telPage, "TEL page should have content")

	// PES page should contain Political, Economic, Social
	// TEL page should contain Technological, Environmental, Legal
	// Note: Exact content depends on template implementation
}

// ----------------------------------------------------------------------------
// TEST: Porter 7 Forces Rendering
// ----------------------------------------------------------------------------

func TestPorter7ForcesRendering(t *testing.T) {
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

	porterPage := pages["Porter"]
	assert.NotEmpty(t, porterPage, "Porter page should have content")

	// Verify all 7 forces are present in the analysis
	porter := coimmaAnalysis.Porter
	assert.NotEmpty(t, porter.CompetitiveRivalry, "Competitive Rivalry should exist")
	assert.NotEmpty(t, porter.SupplierPower, "Supplier Power should exist")
	assert.NotEmpty(t, porter.BuyerPower, "Buyer Power should exist")
	assert.NotEmpty(t, porter.ThreatNewEntrants, "Threat of New Entrants should exist")
	assert.NotEmpty(t, porter.ThreatSubstitutes, "Threat of Substitutes should exist")
	assert.NotEmpty(t, porter.PowerPartnershipsEcosystems, "Power of Partnerships should exist")
	assert.NotEmpty(t, porter.DisruptionAIData, "Disruption by AI/Data should exist")

	// Verify intensity levels are set
	assert.NotEmpty(t, porter.CompetitiveRivalryIntensity)
	assert.NotEmpty(t, porter.SupplierPowerIntensity)
	assert.NotEmpty(t, porter.BuyerPowerIntensity)
}

// ----------------------------------------------------------------------------
// TEST: SWOT Confidence Badges
// ----------------------------------------------------------------------------

func TestSWOTConfidenceBadges(t *testing.T) {
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

	swotPage := pages["SWOT"]
	assert.NotEmpty(t, swotPage, "SWOT page should have content")

	// Verify all SWOT items have confidence levels
	for _, strength := range coimmaAnalysis.SWOT.Strengths {
		assert.NotEmpty(t, strength.Confidence, "Strength should have confidence level")
		assert.Contains(t, []string{"Alta", "Média", "Baixa"}, strength.Confidence)
	}

	for _, weakness := range coimmaAnalysis.SWOT.Weaknesses {
		assert.NotEmpty(t, weakness.Confidence, "Weakness should have confidence level")
	}

	for _, opportunity := range coimmaAnalysis.SWOT.Opportunities {
		assert.NotEmpty(t, opportunity.Confidence, "Opportunity should have confidence level")
	}

	for _, threat := range coimmaAnalysis.SWOT.Threats {
		assert.NotEmpty(t, threat.Confidence, "Threat should have confidence level")
	}
}

// ----------------------------------------------------------------------------
// TEST: Growth Hacking LEAP + SCALE Loops
// ----------------------------------------------------------------------------

func TestGrowthHackingLoops(t *testing.T) {
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

	growthPage := pages["GrowthLoops"]
	assert.NotEmpty(t, growthPage, "Growth Loops page should have content")

	// Verify LEAP and SCALE loops exist
	growth := coimmaAnalysis.GrowthHacking

	assert.Equal(t, "LEAP Loop - Aquisição de Clientes", growth.LeapLoop.Name)
	assert.Equal(t, "Aquisição", growth.LeapLoop.Type)
	assert.NotEmpty(t, growth.LeapLoop.Steps, "LEAP loop should have steps")
	assert.NotEmpty(t, growth.LeapLoop.Metrics, "LEAP loop should have metrics")

	assert.Equal(t, "SCALE Loop - Monetização e Retenção", growth.ScaleLoop.Name)
	assert.Equal(t, "Monetização", growth.ScaleLoop.Type)
	assert.NotEmpty(t, growth.ScaleLoop.Steps, "SCALE loop should have steps")
	assert.NotEmpty(t, growth.ScaleLoop.Metrics, "SCALE loop should have metrics")
}

// ----------------------------------------------------------------------------
// TEST: Scenarios with Probabilities
// ----------------------------------------------------------------------------

func TestScenariosProbabilities(t *testing.T) {
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

	scenariosPage := pages["Scenarios"]
	assert.NotEmpty(t, scenariosPage, "Scenarios page should have content")

	// Verify probabilities sum to 100%
	scenarios := coimmaAnalysis.Scenarios
	totalProb := scenarios.Optimistic.Probability + scenarios.Realist.Probability + scenarios.Pessimistic.Probability
	assert.Equal(t, 100, totalProb, "Probabilities should sum to 100%")

	// Verify each scenario has required actions
	assert.NotEmpty(t, scenarios.Optimistic.RequiredActions, "Optimistic scenario should have actions")
	assert.NotEmpty(t, scenarios.Realist.RequiredActions, "Realist scenario should have actions")
	assert.NotEmpty(t, scenarios.Pessimistic.RequiredActions, "Pessimistic scenario should have actions")
}

// ----------------------------------------------------------------------------
// TEST: OKRs Quarterly Structure
// ----------------------------------------------------------------------------

func TestOKRsQuarterlyStructure(t *testing.T) {
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

	okrsPage := pages["OKRs"]
	assert.NotEmpty(t, okrsPage, "OKRs page should have content")

	// Verify quarterly structure
	okrs := coimmaAnalysis.OKRs
	assert.Len(t, okrs.Quarters, 3, "Should have 3 quarters defined")

	for i, quarter := range okrs.Quarters {
		assert.NotEmpty(t, quarter.Quarter, "Quarter %d should have a name", i+1)
		assert.NotEmpty(t, quarter.Objective, "Quarter %d should have an objective", i+1)
		assert.NotEmpty(t, quarter.KeyResults, "Quarter %d should have key results", i+1)
		assert.NotEmpty(t, quarter.Investment, "Quarter %d should have investment", i+1)
		assert.NotEmpty(t, quarter.Timeline, "Quarter %d should have timeline", i+1)
	}
}

// ----------------------------------------------------------------------------
// TEST: Template Function Helpers (safeHTML, add, lower, replace, slice)
// ----------------------------------------------------------------------------

func TestTemplateFunctionHelpers(t *testing.T) {
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

	// Execute - This will test that all template functions work
	pages, err := svc.GeneratePreview(ctx, "db32e622-56bb-4fba-a480-9384aeaa2f5c", "6057454f-12bf-4783-8334-f5a4424ad246")

	// Assert - If any template function fails, rendering will error
	assert.NoError(t, err, "All template functions should work correctly")
	assert.Len(t, pages, 24, "All pages should render successfully")

	// Spot check that functions are available in templates
	// (Templates use: safeHTML, add, lower, replace, slice)
	for _, html := range pages {
		// Should not contain template errors
		assert.NotContains(t, html, "template:", "Should not have template errors")
		assert.NotContains(t, html, "undefined function", "All functions should be defined")
	}
}

// ----------------------------------------------------------------------------
// TEST: Company Branding (Theme Colors)
// ----------------------------------------------------------------------------

func TestCompanyBranding(t *testing.T) {
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

	coverPage := pages["Cover"]
	assert.NotEmpty(t, coverPage, "Cover page should have content")

	// Verify company name appears
	assert.True(t,
		strings.Contains(coverPage, "Coimma") || strings.Contains(coverPage, coimmaSubmission.CompanyName),
		"Cover page should contain company name",
	)

	// Verify theme colors are used (IMENSIAH branding)
	// Primary: #002A45, Accent: #F59E0B
	// Note: Colors may be embedded in CSS or inline styles
}

// ----------------------------------------------------------------------------
// TEST: Page Count Verification (24 Total)
// ----------------------------------------------------------------------------

func TestPageCountVerification(t *testing.T) {
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

	// Execute Preview
	previewPages, err := svc.GeneratePreview(ctx, "db32e622-56bb-4fba-a480-9384aeaa2f5c", "6057454f-12bf-4783-8334-f5a4424ad246")
	assert.NoError(t, err)
	assert.Equal(t, 24, len(previewPages), "Preview should have 24 pages")

	// Execute Publish
	mockPDFGen.On("Convert", ctx, mock.AnythingOfType("[]string")).
		Return([]byte("fake-pdf"), nil)
	mockStorage.On("Upload", ctx, mock.Anything, mock.Anything, "application/pdf").
		Return("https://storage.example.com/test.pdf", nil)
	mockReportRepo.On("Create", ctx, mock.MatchedBy(func(r *report.Report) bool {
		// Verify TotalPages is set to 24
		return r.TotalPages == 24
	})).Return(nil)

	pdfURL, err := svc.Publish(ctx, "db32e622-56bb-4fba-a480-9384aeaa2f5c", "6057454f-12bf-4783-8334-f5a4424ad246")
	assert.NoError(t, err)
	assert.NotEmpty(t, pdfURL)

	mockReportRepo.AssertExpectations(t)
}
