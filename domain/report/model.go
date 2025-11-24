package report

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Report represents a complete business analysis report with 24 HTML pages
// This is what the frontend will display to end users
type Report struct {
	ID           string `db:"id" json:"id"`
	SubmissionID string `db:"submission_id" json:"submission_id"`
	AnalysisID   string `db:"analysis_id" json:"analysis_id"`

	// 24 HTML Pages - Strategic Cascade Framework (v2)
	CoverPage                string `db:"cover_page" json:"cover_page"`                                 // Page 1
	ExecutiveSummary         string `db:"executive_summary" json:"executive_summary"`                   // Page 2
	TableOfContents          string `db:"table_of_contents" json:"table_of_contents"`                   // Page 3
	DividerPart1Page         string `db:"divider_part1_page" json:"divider_part1_page"`                 // Page 4 (NEW)
	PESTELPesPage            string `db:"pestel_pes_page" json:"pestel_pes_page"`                       // Page 5 (NEW)
	PESTELTelPage            string `db:"pestel_tel_page" json:"pestel_tel_page"`                       // Page 6 (NEW)
	PorterPage               string `db:"porter_page" json:"porter_page"`                               // Page 7
	SWOTPage                 string `db:"swot_page" json:"swot_page"`                                   // Page 8
	DividerPart2Page         string `db:"divider_part2_page" json:"divider_part2_page"`                 // Page 9 (NEW)
	TamSamSomPage            string `db:"tam_sam_som_page" json:"tam_sam_som_page"`                     // Page 10
	BlueOceanPage            string `db:"blue_ocean_page" json:"blue_ocean_page"`                       // Page 11
	DividerPart3Page         string `db:"divider_part3_page" json:"divider_part3_page"`                 // Page 12 (NEW)
	OKRPage                  string `db:"okr_page" json:"okr_page"`                                     // Page 13
	GrowthLoopsPage          string `db:"growth_loops_page" json:"growth_loops_page"`                   // Page 14 (NEW)
	DividerPart4Page         string `db:"divider_part4_page" json:"divider_part4_page"`                 // Page 15 (NEW)
	ScenariosPage            string `db:"scenarios_page" json:"scenarios_page"`                         // Page 16 (Renamed from ScenariosPage in logic, but keeping DB column if possible or mapping correctly)
	RecommendationsPage      string `db:"recommendations_page" json:"recommendations_page"`             // Page 17 (NEW)
	BSCPage                  string `db:"bsc_page" json:"bsc_page"`                                     // Page 18
	BenchmarkingPage         string `db:"benchmarking_page" json:"benchmarking_page"`                   // Page 19
	FinancialProjectionsPage string `db:"financial_projections_page" json:"financial_projections_page"` // Page 20
	GrowthHackingPage        string `db:"growth_hacking_page" json:"growth_hacking_page"`               // Page 21
	RiskAssessmentPage       string `db:"risk_assessment_page" json:"risk_assessment_page"`             // Page 22
	RoadmapPage              string `db:"roadmap_page" json:"roadmap_page"`                             // Page 23 (NEW)
	AppendixPage             string `db:"appendix_page" json:"appendix_page"`                           // Page 24

	// PDF Generation
	PDFURL              string     `db:"pdf_url" json:"pdf_url"`                             // Cloud storage URL for generated PDF
	PDFGeneratedAt      *time.Time `db:"pdf_generated_at" json:"pdf_generated_at"`           // When PDF was created
	PDFGenerationStatus string     `db:"pdf_generation_status" json:"pdf_generation_status"` // pending, processing, completed, failed

	// Metadata
	Status           string     `db:"status" json:"status"`                         // pending, processing, completed, failed
	ErrorMessage     string     `db:"error_message" json:"error_message"`           // Error details if generation failed
	GenerationTimeMs int64      `db:"generation_time_ms" json:"generation_time_ms"` // Time to generate HTML pages
	TotalPages       int        `db:"total_pages" json:"total_pages"`               // Should always be 24
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
	CompletedAt      *time.Time `db:"completed_at" json:"completed_at"`
}

// GetPDFURL returns the stored PDF URL (implements analysis.ReportSummary)
func (r *Report) GetPDFURL() string {
	return r.PDFURL
}

// ReportMetadata contains summary information about the report
// Frontend developers: Use this for report listing and previews
type ReportMetadata struct {
	ID                  string     `json:"id"`
	SubmissionID        string     `json:"submission_id"`
	CompanyName         string     `json:"company_name"`  // From submission
	IndustryName        string     `json:"industry_name"` // From submission
	Status              string     `json:"status"`
	PDFURL              string     `json:"pdf_url"`
	PDFGenerationStatus string     `json:"pdf_generation_status"`
	TotalPages          int        `json:"total_pages"`
	CreatedAt           time.Time  `json:"created_at"`
	CompletedAt         *time.Time `json:"completed_at"`
}

// ReportSummary is a lightweight struct for listing reports
// PERFORMANCE: Excludes the 24 HTML page columns to avoid massive bandwidth usage
// Each report can have ~100KB+ of HTML, so listing 50 reports could pull 5MB+ without this optimization
type ReportSummary struct {
	ID                  string     `db:"id" json:"id"`
	SubmissionID        string     `db:"submission_id" json:"submission_id"`
	AnalysisID          string     `db:"analysis_id" json:"analysis_id"`
	PDFURL              string     `db:"pdf_url" json:"pdf_url"`
	PDFGeneratedAt      *time.Time `db:"pdf_generated_at" json:"pdf_generated_at"`
	PDFGenerationStatus string     `db:"pdf_generation_status" json:"pdf_generation_status"`
	Status              string     `db:"status" json:"status"`
	ErrorMessage        string     `db:"error_message" json:"error_message"`
	GenerationTimeMs    int64      `db:"generation_time_ms" json:"generation_time_ms"`
	TotalPages          int        `db:"total_pages" json:"total_pages"`
	CreatedAt           time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at" json:"updated_at"`
	CompletedAt         *time.Time `db:"completed_at" json:"completed_at"`
}

// PageTemplate defines the structure for HTML page generation
// Frontend developers: This shows what data each page contains
type PageTemplate struct {
	PageNumber int                    `json:"page_number"`
	PageTitle  string                 `json:"page_title"`
	Data       map[string]interface{} `json:"data"`     // Page-specific data
	Styles     string                 `json:"styles"`   // CSS styles for the page
	Template   string                 `json:"template"` // HTML template name
}

// PDFGenerationRequest contains parameters for PDF generation
type PDFGenerationRequest struct {
	ReportID   string            `json:"report_id"`
	HTMLPages  []string          `json:"html_pages"`  // All 24 pages in order
	Options    PDFOptions        `json:"options"`     // PDF generation options
	OutputPath string            `json:"output_path"` // Where to save/upload PDF
	Metadata   map[string]string `json:"metadata"`    // PDF metadata (author, title, etc.)
}

// PDFOptions defines PDF generation parameters
type PDFOptions struct {
	Format              string  `json:"format"`                // A4, Letter, Legal
	Orientation         string  `json:"orientation"`           // portrait, landscape
	Margin              string  `json:"margin"`                // CSS margin value (e.g., "1cm")
	PrintBackground     bool    `json:"print_background"`      // Include background colors/images
	DisplayHeaderFooter bool    `json:"display_header_footer"` // Show page numbers
	Scale               float64 `json:"scale"`                 // Scale factor (0.1 to 2)
	PreferCSSPageSize   bool    `json:"prefer_css_page_size"`  // Use CSS @page size
}

// DefaultPDFOptions returns sensible defaults for PDF generation
// NOTE: Templates use A4 Landscape (842px × 595px) format
func DefaultPDFOptions() PDFOptions {
	return PDFOptions{
		Format:              "A4",
		Orientation:         "landscape", // Changed from portrait to match templates
		Margin:              "0",         // Changed from 1cm to match template design
		PrintBackground:     true,
		DisplayHeaderFooter: false, // Changed from true - templates don't use headers/footers
		Scale:               1.0,
		PreferCSSPageSize:   true, // Changed to true - respect @page CSS rules
	}
}

// ReportProgress tracks report generation progress
// Frontend developers: Use this for real-time progress updates
type ReportProgress struct {
	ReportID        string    `json:"report_id"`
	CurrentPage     int       `json:"current_page"`      // Which page is being generated (1-24)
	TotalPages      int       `json:"total_pages"`       // Always 24
	Status          string    `json:"status"`            // processing, completed, failed
	PercentComplete float64   `json:"percent_complete"`  // 0-100
	EstimatedTimeMs int64     `json:"estimated_time_ms"` // Estimated remaining time
	LastUpdated     time.Time `json:"last_updated"`
}

// ReportStats provides a high-level overview for dashboards
type ReportStats struct {
	TotalReports       int              `json:"total_reports"`
	CompletedReports   int              `json:"completed_reports"`
	FailedReports      int              `json:"failed_reports"`
	PendingReports     int              `json:"pending_reports"`
	AverageGenTime     int64            `json:"average_gen_time_ms"` // Average generation time
	TotalPDFsGenerated int              `json:"total_pdfs_generated"`
	RecentReports      []ReportMetadata `json:"recent_reports"` // Last 10 reports
}

// Scan implements sql.Scanner for JSONB compatibility (if needed for metadata)
func (p *PageTemplate) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, p)
}

func (p PageTemplate) Value() (driver.Value, error) {
	return json.Marshal(p)
}
