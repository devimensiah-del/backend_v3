package report

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
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
	templates      *template.Template
}

func NewService(
	repo Repository,
	analysisRepo AnalysisRepository,
	subRepo SubmissionRepository,
	pdfGen PDFGenerator,
	storage StorageClient,
	logger zerolog.Logger,
) *Service {
	// Templates are now self-contained and parsed individually in renderPage
	// No need to parse them all at startup
	return &Service{
		repo:           repo,
		analysisRepo:   analysisRepo,
		submissionRepo: subRepo,
		pdfGen:         pdfGen,
		storage:        storage,
		logger:         logger.With().Str("service", "report").Logger(),
		templates:      nil, // Not used anymore
	}
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

	// Ideally use Upsert logic here
	_ = s.repo.Create(ctx, report)

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

// renderPage merges the template with data and theme
func (s *Service) renderPage(templateName string, globalData *ReportData, specificContent interface{}) (string, error) {
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
	templatePaths := []string{
		"templates/report_v2/" + templateName,            // Production / From root
		"backend_v3/templates/report_v2/" + templateName, // From parent dir
		"../templates/report_v2/" + templateName,         // From tests in subdirs
		"../../templates/report_v2/" + templateName,      // From deep test dirs
	}

	var tmpl *template.Template
	var err error

	funcMap := template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		"add":      func(a, b int) int { return a + b },
		"lower":    func(s string) string { return strings.ToLower(s) },
		"replace":  func(s, old, new string) string { return strings.ReplaceAll(s, old, new) },
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
			break
		}
	}

	if err != nil {
		return "", fmt.Errorf("template %s not found in any location: %w", templateName, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, payload); err != nil {
		return "", fmt.Errorf("failed to render %s: %w", templateName, err)
	}

	return buf.String(), nil
}
