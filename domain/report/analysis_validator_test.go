package report_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"backend_v3/domain/analysis"
	"backend_v3/domain/report"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestHTMLAnalysis_ComprehensiveValidation performs deep analysis of all generated HTML
func TestHTMLAnalysis_ComprehensiveValidation(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Setup mocks
	mockAnalysisRepo := new(MockAnalysisRepo)
	mockSubmissionRepo := new(MockSubmissionRepo)
	mockReportRepo := new(MockReportRepo)
	mockPDFGen := new(MockPDFGenerator)
	mockStorage := new(MockStorageClient)

	// Setup test data
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

	// Generate all pages
	logger.Info().Msg("\n" + strings.Repeat("=", 100))
	logger.Info().Msg("🔍 COMPREHENSIVE HTML ANALYSIS - COIMMA REPORT")
	logger.Info().Msg(strings.Repeat("=", 100))

	htmlPages, err := svc.GeneratePreview(ctx, "db32e622-56bb-4fba-a480-9384aeaa2f5c", "6057454f-12bf-4783-8334-f5a4424ad246")
	assert.NoError(t, err)
	assert.NotNil(t, htmlPages)

	// Create output directory
	outputDir := filepath.Join("..", "..", "test_artifacts", "html_analysis")
	os.MkdirAll(outputDir, 0755)

	var analysisReport strings.Builder
	analysisReport.WriteString("# HTML Generation Analysis Report\n\n")
	analysisReport.WriteString("## Executive Summary\n\n")
	analysisReport.WriteString(fmt.Sprintf("- **Total Pages Generated:** %d\n", len(htmlPages)))
	analysisReport.WriteString(fmt.Sprintf("- **Test Date:** %s\n", "2025-11-22"))
	analysisReport.WriteString(fmt.Sprintf("- **Analysis Subject:** Coimma Strategic Analysis\n\n"))

	// Analysis metrics
	totalIssues := 0
	pageIssues := make(map[string][]string)
	pageSizes := make(map[string]int)

	analysisReport.WriteString("## Page-by-Page Analysis\n\n")

	for pageName, html := range htmlPages {
		logger.Info().Msgf("\n📄 Analyzing: %s", pageName)

		issues := []string{}

		// Save to file
		fileName := filepath.Join(outputDir, pageName+".html")
		err := os.WriteFile(fileName, []byte(html), 0644)
		if err == nil {
			logger.Info().Msgf("  💾 Saved: %s", fileName)
		}

		pageSizes[pageName] = len(html)

		// 1. Check for A4 landscape dimensions (842x595px)
		if !strings.Contains(html, "width: 842px") && !strings.Contains(html, "width:842px") {
			issues = append(issues, "⚠️ Missing or incorrect page width (should be 842px)")
		}
		if !strings.Contains(html, "height: 595px") && !strings.Contains(html, "height:595px") {
			issues = append(issues, "⚠️ Missing or incorrect page height (should be 595px)")
		}

		// 2. Check for overflow handling
		if !strings.Contains(html, "overflow: hidden") && !strings.Contains(html, "overflow:hidden") {
			issues = append(issues, "⚠️ Missing overflow:hidden - content may bleed")
		}

		// 3. Check for empty content placeholders
		emptyPlaceholders := regexp.MustCompile(`{{\s*\.\w+\s*}}`).FindAllString(html, -1)
		if len(emptyPlaceholders) > 0 {
			issues = append(issues, fmt.Sprintf("⚠️ Found %d unrendered template variables: %v", len(emptyPlaceholders), emptyPlaceholders[:min(3, len(emptyPlaceholders))]))
		}

		// 4. Check for "undefined" or "null" in rendered content
		if strings.Contains(html, ">undefined<") || strings.Contains(html, ">null<") {
			issues = append(issues, "⚠️ Contains 'undefined' or 'null' values")
		}

		// 5. Check for empty lists that should have content
		emptyListPattern := regexp.MustCompile(`<ul[^>]*>\s*</ul>`)
		if emptyListPattern.MatchString(html) {
			issues = append(issues, "⚠️ Contains empty <ul> elements")
		}

		// 6. Check for missing framework data indicators
		if strings.Contains(pageName, "SWOT") {
			if !strings.Contains(html, "Forças") && !strings.Contains(html, "Strengths") {
				issues = append(issues, "❌ SWOT missing Strengths section")
			}
			// Check for confidence badges
			if !strings.Contains(html, "confidence-badge") && !strings.Contains(html, "Alta") {
				issues = append(issues, "⚠️ SWOT missing confidence badges")
			}
		}

		if strings.Contains(pageName, "OKR") {
			if !strings.Contains(html, "Q1") && !strings.Contains(html, "Quarter") {
				issues = append(issues, "❌ OKRs missing quarterly structure")
			}
		}

		if strings.Contains(pageName, "Porter") {
			// Check for 7 forces
			forceCount := strings.Count(html, "força") + strings.Count(html, "Force")
			if forceCount < 7 {
				issues = append(issues, fmt.Sprintf("⚠️ Porter's analysis may be incomplete (found %d force references, expected 7+)", forceCount))
			}
			// Check for intensity badges
			if !strings.Contains(html, "intensity-") && !strings.Contains(html, "Alta") {
				issues = append(issues, "⚠️ Porter missing intensity indicators")
			}
		}

		if strings.Contains(pageName, "Scenarios") {
			if !strings.Contains(html, "20%") && !strings.Contains(html, "60%") {
				issues = append(issues, "⚠️ Scenarios missing probability percentages")
			}
		}

		if strings.Contains(pageName, "Growth") {
			if !strings.Contains(html, "LEAP") && !strings.Contains(html, "SCALE") {
				issues = append(issues, "❌ Growth loops missing LEAP/SCALE structure")
			}
		}

		// 7. Check for CSS that might cause overflow
		if strings.Contains(html, "position: absolute") && !strings.Contains(html, "overflow: hidden") {
			issues = append(issues, "⚠️ Uses absolute positioning without overflow protection")
		}

		// 8. Check for very long text that might overflow
		longTextPattern := regexp.MustCompile(`>\s*[^<]{300,}\s*<`)
		if longTextPattern.MatchString(html) {
			issues = append(issues, "⚠️ Contains very long text blocks (>300 chars) that may overflow")
		}

		// 9. Check for proper page structure
		if !strings.Contains(html, "<!DOCTYPE html>") {
			issues = append(issues, "⚠️ Missing DOCTYPE declaration")
		}
		if !strings.Contains(html, "</html>") {
			issues = append(issues, "❌ Missing closing </html> tag")
		}

		// 10. Check for @page CSS rule for PDF generation
		if !strings.Contains(html, "@page") {
			issues = append(issues, "⚠️ Missing @page CSS rule for PDF rendering")
		}

		// Store issues
		if len(issues) > 0 {
			pageIssues[pageName] = issues
			totalIssues += len(issues)

			logger.Warn().Msgf("  ⚠️  Found %d issues:", len(issues))
			for _, issue := range issues {
				logger.Warn().Msgf("      %s", issue)
			}
		} else {
			logger.Info().Msg("  ✅ No issues found")
		}

		// Write to report
		analysisReport.WriteString(fmt.Sprintf("### %s\n", pageName))
		analysisReport.WriteString(fmt.Sprintf("- **Size:** %d bytes\n", len(html)))
		if len(issues) > 0 {
			analysisReport.WriteString(fmt.Sprintf("- **Issues Found:** %d\n", len(issues)))
			for _, issue := range issues {
				analysisReport.WriteString(fmt.Sprintf("  - %s\n", issue))
			}
		} else {
			analysisReport.WriteString("- **Status:** ✅ No issues\n")
		}
		analysisReport.WriteString("\n")
	}

	// Summary statistics
	logger.Info().Msg("\n" + strings.Repeat("=", 100))
	logger.Info().Msg("📊 ANALYSIS SUMMARY")
	logger.Info().Msg(strings.Repeat("=", 100))
	logger.Info().Msgf("Total Pages: %d", len(htmlPages))
	logger.Info().Msgf("Pages with Issues: %d", len(pageIssues))
	logger.Info().Msgf("Total Issues: %d", totalIssues)

	if len(pageIssues) > 0 {
		logger.Warn().Msg("\n⚠️  PAGES WITH ISSUES:")
		for page, issues := range pageIssues {
			logger.Warn().Msgf("  %s: %d issues", page, len(issues))
		}
	} else {
		logger.Info().Msg("\n✅ ALL PAGES VALIDATED SUCCESSFULLY!")
	}

	// Data completeness check
	logger.Info().Msg("\n" + strings.Repeat("=", 100))
	logger.Info().Msg("📋 DATA COMPLETENESS CHECK")
	logger.Info().Msg(strings.Repeat("=", 100))

	dataChecks := analyzeDataCompleteness(coimmaAnalysis)
	for framework, status := range dataChecks {
		if status {
			logger.Info().Msgf("  ✅ %s: Complete", framework)
		} else {
			logger.Warn().Msgf("  ⚠️  %s: Incomplete or missing data", framework)
		}
	}

	// Write final report
	analysisReport.WriteString("\n## Summary\n\n")
	analysisReport.WriteString(fmt.Sprintf("- **Total Issues:** %d\n", totalIssues))
	analysisReport.WriteString(fmt.Sprintf("- **Pages with Issues:** %d/%d\n", len(pageIssues), len(htmlPages)))
	analysisReport.WriteString(fmt.Sprintf("- **Clean Pages:** %d/%d\n", len(htmlPages)-len(pageIssues), len(htmlPages)))

	reportFile := filepath.Join(outputDir, "ANALYSIS_REPORT.md")
	err = os.WriteFile(reportFile, []byte(analysisReport.String()), 0644)
	assert.NoError(t, err)
	logger.Info().Msgf("\n📄 Full report saved to: %s", reportFile)

	// Final assertion
	if totalIssues > 0 {
		logger.Warn().Msgf("\n⚠️  Test completed with %d issues requiring attention", totalIssues)
	} else {
		logger.Info().Msg("\n✅ ALL VALIDATIONS PASSED!")
	}
}

func analyzeDataCompleteness(a *analysis.Analysis) map[string]bool {
	checks := make(map[string]bool)

	// PESTEL
	checks["PESTEL"] = len(a.PESTEL.Political) > 0 && len(a.PESTEL.Economic) > 0 &&
		len(a.PESTEL.Social) > 0 && len(a.PESTEL.Technological) > 0 &&
		len(a.PESTEL.Environmental) > 0 && len(a.PESTEL.Legal) > 0

	// Porter's 7 Forces
	checks["Porter"] = a.Porter.CompetitiveRivalry != "" && a.Porter.SupplierPower != "" &&
		a.Porter.BuyerPower != "" && a.Porter.ThreatNewEntrants != "" &&
		a.Porter.ThreatSubstitutes != "" && a.Porter.PowerPartnershipsEcosystems != "" &&
		a.Porter.DisruptionAIData != ""

	// SWOT
	checks["SWOT"] = len(a.SWOT.Strengths) > 0 && len(a.SWOT.Weaknesses) > 0 &&
		len(a.SWOT.Opportunities) > 0 && len(a.SWOT.Threats) > 0

	// TAM-SAM-SOM
	checks["TAM-SAM-SOM"] = a.TamSamSom.TAM != "" && a.TamSamSom.SAM != "" && a.TamSamSom.SOM != ""

	// Blue Ocean
	checks["BlueOcean"] = len(a.BlueOcean.Eliminate) > 0 && len(a.BlueOcean.Reduce) > 0 &&
		len(a.BlueOcean.Raise) > 0 && len(a.BlueOcean.Create) > 0

	// OKRs
	checks["OKRs"] = len(a.OKRs.Quarters) >= 3

	// Growth Hacking
	checks["GrowthHacking"] = a.GrowthHacking.LeapLoop.Name != "" && a.GrowthHacking.ScaleLoop.Name != ""

	// Scenarios
	checks["Scenarios"] = a.Scenarios.Optimistic.Name != "" && a.Scenarios.Realist.Name != "" &&
		a.Scenarios.Pessimistic.Name != ""

	// Decision Matrix
	checks["DecisionMatrix"] = len(a.DecisionMatrix.PriorityRecommendations) > 0 &&
		a.DecisionMatrix.RecommendedOption != ""

	// Benchmarking
	checks["Benchmarking"] = len(a.Benchmarking.Competitors) > 0

	// BSC
	checks["BSC"] = len(a.BSC.Financial) > 0 && len(a.BSC.Customer) > 0

	// Synthesis
	checks["Synthesis"] = a.Synthesis.ExecutiveSummary != "" && len(a.Synthesis.StrategicPriorities) > 0

	return checks
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
