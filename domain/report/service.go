package report

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
	"time"

	"backend_v3/domain/analysis"
	"backend_v3/domain/submission"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// --- Interfaces (Dependencies) ---

type PDFGenerator interface {
	// Convert takes a list of HTML strings and returns PDF bytes
	Convert(ctx context.Context, htmlPages []string) ([]byte, error)
}

type StorageClient interface {
	// Upload saves bytes to S3/Supabase and returns the public URL
	Upload(ctx context.Context, path string, data []byte, contentType string) (string, error)
}

// We need the Submission repo to get the Company Name for the cover page
type SubmissionRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*submission.Submission, error)
}

type AnalysisRepository interface {
	GetByID(ctx context.Context, id string) (*analysis.Analysis, error)
}

// --- Service Definition ---

type Service struct {
	repo           Repository
	analysisRepo   AnalysisRepository
	submissionRepo SubmissionRepository
	pdfGen         PDFGenerator
	storage        StorageClient
	logger         zerolog.Logger
	templates      map[string]*template.Template // Cache of parsed templates
}

func NewService(
	repo Repository,
	analysisRepo AnalysisRepository,
	subRepo SubmissionRepository,
	pdfGen PDFGenerator,
	storage StorageClient,
	logger zerolog.Logger,
) *Service {
	svc := &Service{
		repo:           repo,
		analysisRepo:   analysisRepo,
		submissionRepo: subRepo,
		pdfGen:         pdfGen,
		storage:        storage,
		logger:         logger.With().Str("service", "report").Logger(),
		templates:      make(map[string]*template.Template),
	}

	// Pre-parse and cache all templates at startup for performance
	if err := svc.initTemplateCache(); err != nil {
		logger.Warn().Err(err).Msg("Failed to initialize template cache, will parse on-demand")
	}

	return svc
}

// initTemplateCache pre-parses all 24 templates and caches them in memory
func (s *Service) initTemplateCache() error {
	// Define template function map (same as renderPage)
	funcMap := template.FuncMap{
		"add":     func(a, b int) int { return a + b },
		"lower":   func(s string) string { return strings.ToLower(s) },
		"replace": func(s, old, new string) string { return strings.ReplaceAll(s, old, new) },
		"slice": func(start, end int, s string) string {
			if start < 0 {
				start = 0
			}
			if end > len(s) {
				end = len(s)
			}
			if start >= len(s) {
				return ""
			}
			return s[start:end]
		},
	}

	// List of all template files
	templateFiles := []string{
		"01_cover.html",
		"02_exec_summary.html",
		"03_toc.html",
		"03a_divider_part1.html",
		"04a_pestel_pes.html",
		"04b_pestel_tel.html",
		"05a_porter_7forces.html",
		"06_swot.html",
		"08a_divider_part2.html",
		"07_tam_sam_som.html",
		"08_ocean.html",
		"11a_divider_part3.html",
		"12a_okrs_quarterly.html",
		"13a_growth_loops.html",
		"14a_divider_part4.html",
		"15a_scenarios.html",
		"16a_recommendations_review.html",
		"10_business_model.html",
		"11_competitive_analysis.html",
		"12_financial_projections.html",
		"13_gtm_strategy.html",
		"14_risk_assessment.html",
		"15_roadmap.html",
		"16_appendix.html",
	}

	// Template search paths
	templatePaths := []string{
		filepath.Join("templates", "report_v2"),
		filepath.Join("backend_v3", "templates", "report_v2"),
		filepath.Join("..", "templates", "report_v2"),
		filepath.Join("..", "..", "templates", "report_v2"),
	}

	// Try to parse each template from available paths
	for _, templateName := range templateFiles {
		var tmpl *template.Template
		var err error

		for _, basePath := range templatePaths {
			fullPath := filepath.Join(basePath, templateName)
			tmpl, err = template.New(templateName).Funcs(funcMap).ParseFiles(fullPath)
			if err == nil {
				s.templates[templateName] = tmpl
				s.logger.Debug().
					Str("template", templateName).
					Str("path", fullPath).
					Msg("Template cached successfully")
				break
			}
		}

		if err != nil {
			// Log warning but don't fail - will try on-demand parsing
			s.logger.Warn().
				Str("template", templateName).
				Err(err).
				Msg("Failed to cache template, will parse on-demand")
		}
	}

	s.logger.Info().
		Int("cached", len(s.templates)).
		Int("total", len(templateFiles)).
		Msg("Template cache initialized")

	return nil
}

// --- Public Methods ---

// GetBySubmissionID retrieves a report by submission ID
func (s *Service) GetBySubmissionID(ctx context.Context, submissionID string) (*Report, error) {
	return s.repo.GetBySubmissionID(ctx, submissionID)
}

// GeneratePreview generates the HTML for the UI (Draft Mode).
// It does NOT save a PDF. It returns a map of "Section Name" -> "HTML String".
func (s *Service) GeneratePreview(ctx context.Context, submissionID, analysisID string) (map[string]string, error) {
	// 1. Fetch Data
	data, err := s.gatherReportData(ctx, submissionID, analysisID)
	if err != nil {
		return nil, err
	}

	// 2. Render Pages
	// We return a map so the Frontend can render them in tabs (e.g., "Show me the SWOT tab")
	pages := make(map[string]string)

	// Define the page mapping - Using new report_v2 templates with TUC Glasses structure
	renderList := []struct {
		Key      string
		Template string
		Content  interface{}
	}{
		// Executive Overview
		{"Cover", "01_cover.html", nil},
		{"ExecutiveSummary", "02_exec_summary.html", data.Analysis.Synthesis},
		{"TableOfContents", "03_toc.html", nil},

		// Part I: Onde Estamos?
		{"DividerPart1", "03a_divider_part1.html", nil},
		{"PESTEL_PES", "04a_pestel_pes.html", data.Analysis.PESTEL},
		{"PESTEL_TEL", "04b_pestel_tel.html", data.Analysis.PESTEL},
		{"Porter", "05a_porter_7forces.html", data.Analysis.Porter},
		{"SWOT", "06_swot.html", data.Analysis.SWOT},

		// Part II: Onde Queremos Ir?
		{"DividerPart2", "08a_divider_part2.html", nil},
		{"MarketSizing", "07_tam_sam_som.html", data.Analysis.TamSamSom},
		{"BlueOcean", "08_ocean.html", data.Analysis.BlueOcean},

		// Part III: Como Chegar Lá?
		{"DividerPart3", "11a_divider_part3.html", nil},
		{"OKRs", "12a_okrs_quarterly.html", data.Analysis.OKRs},
		{"GrowthLoops", "13a_growth_loops.html", data.Analysis.GrowthHacking},

		// Part IV: O Que Fazer Agora?
		{"DividerPart4", "14a_divider_part4.html", nil},
		{"Scenarios", "15a_scenarios.html", data.Analysis.Scenarios},
		{"Recommendations", "16a_recommendations_review.html", data.Analysis.DecisionMatrix},

		// Appendices
		{"BusinessModel", "10_business_model.html", data.Analysis.BSC},
		{"CompetitiveAnalysis", "11_competitive_analysis.html", data.Analysis.Benchmarking},
		{"FinancialProjections", "12_financial_projections.html", data.Analysis.Scenarios},
		{"GTMStrategy", "13_gtm_strategy.html", data.Analysis.GrowthHacking},
		{"RiskAssessment", "14_risk_assessment.html", data.Analysis.Scenarios},
		{"Roadmap", "15_roadmap.html", data.Analysis.DecisionMatrix},
		{"Appendix", "16_appendix.html", nil},
	}

	for _, item := range renderList {
		html, err := s.renderPage(item.Template, data, item.Content)
		if err != nil {
			return nil, fmt.Errorf("error rendering %s: %w", item.Key, err)
		}
		pages[item.Key] = html
	}

	return pages, nil
}

// Publish freezes the report, generates a PDF, and returns the URL (Final Mode).
func (s *Service) Publish(ctx context.Context, submissionID, analysisID string) (string, error) {
	s.logger.Info().Str("analysis_id", analysisID).Msg("Publishing Report")

	// 1. Generate HTML for all pages (Ordered List for PDF)
	data, err := s.gatherReportData(ctx, submissionID, analysisID)
	if err != nil {
		return "", err
	}

	// Order matters for the PDF! Following TUC Glasses 4-part narrative structure
	templateOrder := []struct {
		Template string
		Content  interface{}
	}{
		// Executive Overview
		{"01_cover.html", nil},
		{"02_exec_summary.html", data.Analysis.Synthesis},
		{"03_toc.html", nil},

		// Part I: Onde Estamos? (Where are we?)
		{"03a_divider_part1.html", nil},
		{"04a_pestel_pes.html", data.Analysis.PESTEL},
		{"04b_pestel_tel.html", data.Analysis.PESTEL},
		{"05a_porter_7forces.html", data.Analysis.Porter},
		{"06_swot.html", data.Analysis.SWOT},

		// Part II: Onde Queremos Ir? (Where do we want to go?)
		{"08a_divider_part2.html", nil},
		{"07_tam_sam_som.html", data.Analysis.TamSamSom},
		{"08_ocean.html", data.Analysis.BlueOcean},

		// Part III: Como Chegar Lá? (How to get there?)
		{"11a_divider_part3.html", nil},
		{"12a_okrs_quarterly.html", data.Analysis.OKRs},
		{"13a_growth_loops.html", data.Analysis.GrowthHacking},

		// Part IV: O Que Fazer Agora? (What to do now?)
		{"14a_divider_part4.html", nil},
		{"15a_scenarios.html", data.Analysis.Scenarios},
		{"16a_recommendations_review.html", data.Analysis.DecisionMatrix},

		// Appendices
		{"10_business_model.html", data.Analysis.BSC},
		{"11_competitive_analysis.html", data.Analysis.Benchmarking},
		{"12_financial_projections.html", data.Analysis.Scenarios},
		{"13_gtm_strategy.html", data.Analysis.GrowthHacking},
		{"14_risk_assessment.html", data.Analysis.Scenarios},
		{"15_roadmap.html", data.Analysis.DecisionMatrix},
		{"16_appendix.html", nil},
	}

	var htmlPages []string
	for _, item := range templateOrder {
		html, err := s.renderPage(item.Template, data, item.Content)
		if err != nil {
			return "", err
		}
		htmlPages = append(htmlPages, html)
	}

	// 2. Generate PDF (Expensive operation)
	pdfBytes, err := s.pdfGen.Convert(ctx, htmlPages)
	if err != nil {
		return "", fmt.Errorf("pdf generation failed: %w", err)
	}

	// 3. Upload to Cloud Storage
	fileName := fmt.Sprintf("reports/%s_%d.pdf", analysisID, time.Now().Unix())
	pdfURL, err := s.storage.Upload(ctx, fileName, pdfBytes, "application/pdf")
	if err != nil {
		return "", fmt.Errorf("storage upload failed: %w", err)
	}

	now := time.Now()

	// 4. Save Record in DB with ALL generated HTML pages
	// CRITICAL FIX: Assign all generated HTML to the Report struct before saving
	report := &Report{
		ID:                  uuid.New().String(),
		SubmissionID:        submissionID,
		AnalysisID:          analysisID,
		Status:              "completed",
		PDFURL:              pdfURL,
		PDFGeneratedAt:      &now,
		PDFGenerationStatus: "completed",
		TotalPages:          24, // Updated for new 4-part structure with dividers and appendices
		CreatedAt:           now,
		UpdatedAt:           now,
		CompletedAt:         &now,

		// Assign all 24 generated HTML pages (in order from templateOrder)
		CoverPage:                htmlPages[0],  // 01_cover.html
		ExecutiveSummary:         htmlPages[1],  // 02_exec_summary.html
		TableOfContents:          htmlPages[2],  // 03_toc.html
		DividerPart1Page:         htmlPages[3],  // 03a_divider_part1.html
		PESTELPesPage:            htmlPages[4],  // 04a_pestel_pes.html
		PESTELTelPage:            htmlPages[5],  // 04b_pestel_tel.html
		PorterPage:               htmlPages[6],  // 05a_porter_7forces.html
		SWOTPage:                 htmlPages[7],  // 06_swot.html
		DividerPart2Page:         htmlPages[8],  // 08a_divider_part2.html
		TamSamSomPage:            htmlPages[9],  // 07_tam_sam_som.html
		BlueOceanPage:            htmlPages[10], // 08_ocean.html
		DividerPart3Page:         htmlPages[11], // 11a_divider_part3.html
		OKRPage:                  htmlPages[12], // 12a_okrs_quarterly.html
		GrowthLoopsPage:          htmlPages[13], // 13a_growth_loops.html
		DividerPart4Page:         htmlPages[14], // 14a_divider_part4.html
		ScenariosPage:            htmlPages[15], // 15a_scenarios.html
		RecommendationsPage:      htmlPages[16], // 16a_recommendations_review.html
		BSCPage:                  htmlPages[17], // 10_business_model.html
		BenchmarkingPage:         htmlPages[18], // 11_competitive_analysis.html
		FinancialProjectionsPage: htmlPages[19], // 12_financial_projections.html
		GrowthHackingPage:        htmlPages[20], // 13_gtm_strategy.html
		RiskAssessmentPage:       htmlPages[21], // 14_risk_assessment.html
		RoadmapPage:              htmlPages[22], // 15_roadmap.html
		AppendixPage:             htmlPages[23], // 16_appendix.html
	}

	// Use Upsert for idempotency - allows re-publishing reports
	if err := s.repo.Upsert(ctx, report); err != nil {
		return "", fmt.Errorf("failed to save report: %w", err)
	}

	return pdfURL, nil
}

// =================================================================================
// INTERNAL HELPERS
// =================================================================================

// ReportData is the "Master Context" passed to templates
type ReportData struct {
	Theme    ThemeConfig
	Company  string
	Industry string
	Date     string
	Analysis *analysis.Analysis
}

// gatherReportData fetches everything needed from the DB
func (s *Service) gatherReportData(ctx context.Context, subID, analysisID string) (*ReportData, error) {
	sub, err := s.submissionRepo.GetByID(ctx, uuid.MustParse(subID))
	if err != nil {
		return nil, err
	}

	an, err := s.analysisRepo.GetByID(ctx, analysisID)
	if err != nil {
		return nil, err
	}

	return &ReportData{
		Theme:    DefaultTheme(), // IMENSIAH Branding
		Company:  sub.CompanyName,
		Industry: *sub.CompanyIndustry,
		Date:     time.Now().Format("January 2006"),
		Analysis: an,
	}, nil
}

// Whitelist of allowed template names to prevent path traversal attacks
var allowedTemplates = map[string]bool{
	"01_cover.html":                   true,
	"02_exec_summary.html":            true,
	"03_toc.html":                     true,
	"03a_divider_part1.html":          true,
	"04a_pestel_pes.html":             true,
	"04b_pestel_tel.html":             true,
	"05a_porter_7forces.html":         true,
	"06_swot.html":                    true,
	"08a_divider_part2.html":          true,
	"07_tam_sam_som.html":             true,
	"08_ocean.html":                   true,
	"11a_divider_part3.html":          true,
	"12a_okrs_quarterly.html":         true,
	"13a_growth_loops.html":           true,
	"14a_divider_part4.html":          true,
	"15a_scenarios.html":              true,
	"16a_recommendations_review.html": true,
	"10_business_model.html":          true,
	"11_competitive_analysis.html":    true,
	"12_financial_projections.html":   true,
	"13_gtm_strategy.html":            true,
	"14_risk_assessment.html":         true,
	"15_roadmap.html":                 true,
	"16_appendix.html":                true,
}

// renderPage merges the template with data and theme
func (s *Service) renderPage(templateName string, globalData *ReportData, specificContent interface{}) (string, error) {
	// SECURITY: Validate template name against whitelist to prevent path traversal
	cleanName := filepath.Base(templateName) // Remove any path components
	if !allowedTemplates[cleanName] {
		return "", fmt.Errorf("invalid template name: %s (not in whitelist)", templateName)
	}
	templateName = cleanName // Use only the clean filename
	// Construct full payload with all necessary data
	payload := map[string]interface{}{
		"Theme":          globalData.Theme,
		"Company":        globalData.Company,
		"CompanyName":    globalData.Company,
		"Industry":       globalData.Industry,
		"Date":           globalData.Date,
		"ReportDate":     globalData.Date,
		"Title":          "Strategic Analysis Report",
		"ReportTitle":    "Strategic Analysis Report",
		"ReportSubtitle": "Comprehensive Business Intelligence",
		"Year":           time.Now().Year(),
		"PageNumber":     1,
		"Content":        specificContent,
		"Market":         "Brazil",
		"Version":        "1.0",

		// Add all the specific framework data directly so templates can access them
		"PESTEL":         globalData.Analysis.PESTEL,
		"Porter":         globalData.Analysis.Porter,
		"SWOT":           globalData.Analysis.SWOT,
		"TamSamSom":      globalData.Analysis.TamSamSom,
		"BlueOcean":      globalData.Analysis.BlueOcean,
		"OKRs":           globalData.Analysis.OKRs,
		"BSC":            globalData.Analysis.BSC,
		"Benchmarking":   globalData.Analysis.Benchmarking,
		"Scenarios":      globalData.Analysis.Scenarios,
		"GrowthHacking":  globalData.Analysis.GrowthHacking,
		"DecisionMatrix": globalData.Analysis.DecisionMatrix,
		"Synthesis":      globalData.Analysis.Synthesis,

		// Cover page needs these
		"Overview":        make([]string, 0),
		"Recommendations": make([]string, 0),
		"KeyFindings":     make([]interface{}, 0),
		"Metrics":         make([]interface{}, 0),
	}

	// Add synthesis data if available
	if globalData.Analysis != nil && globalData.Analysis.Synthesis.ExecutiveSummary != "" {
		payload["Overview"] = []string{globalData.Analysis.Synthesis.ExecutiveSummary}
		if len(globalData.Analysis.Synthesis.KeyFindings) > 0 {
			payload["KeyFindings"] = globalData.Analysis.Synthesis.KeyFindings
		}
		if len(globalData.Analysis.Synthesis.StrategicPriorities) > 0 {
			payload["Recommendations"] = globalData.Analysis.Synthesis.StrategicPriorities
		}
	}

	// Templates are now self-contained - parse individually
	// Using report_v2 for TUC Glasses aligned templates
	// SECURITY: Use filepath.Join to safely construct paths, preventing traversal
	templatePaths := []string{
		filepath.Join("templates", "report_v2", templateName),               // Production / From root
		filepath.Join("backend_v3", "templates", "report_v2", templateName), // From parent dir
		filepath.Join("..", "templates", "report_v2", templateName),         // From tests in subdirs
		filepath.Join("..", "..", "templates", "report_v2", templateName),   // From deep test dirs
	}

	// PERFORMANCE: Try to use cached template first
	tmpl, cached := s.templates[templateName]

	// If not cached, parse on-demand (fallback)
	if !cached {
		var err error

		// SECURITY: Removed safeHTML function to prevent XSS vulnerabilities
		// Go's html/template automatically escapes all content by default
		funcMap := template.FuncMap{
			"add":     func(a, b int) int { return a + b },
			"lower":   func(s string) string { return strings.ToLower(s) },
			"replace": func(s, old, new string) string { return strings.ReplaceAll(s, old, new) },
			"slice": func(start, end int, s string) string {
				if start < 0 {
					start = 0
				}
				if end > len(s) {
					end = len(s)
				}
				if start >= len(s) {
					return ""
				}
				return s[start:end]
			},
		}

		for _, path := range templatePaths {
			tmpl, err = template.New(templateName).Funcs(funcMap).ParseFiles(path)
			if err == nil {
				// Cache for next time
				s.templates[templateName] = tmpl
				s.logger.Debug().
					Str("template", templateName).
					Msg("Template parsed and cached on-demand")
				break
			}
		}

		if err != nil {
			return "", fmt.Errorf("template %s not found in any location: %w", templateName, err)
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, payload); err != nil {
		return "", fmt.Errorf("failed to render %s: %w", templateName, err)
	}

	return buf.String(), nil
}
