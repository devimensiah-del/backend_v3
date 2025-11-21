package report

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Report represents a complete business analysis report with 13 HTML pages
// This is what the frontend will display to end users
type Report struct {
	ID           string    `db:"id" json:"id"`
	SubmissionID string    `db:"submission_id" json:"submission_id"`
	AnalysisID   string    `db:"analysis_id" json:"analysis_id"`

	// 13 HTML Pages - Each page is a complete HTML document
	// Frontend developers: These are ready to display in iframe or render directly
	CoverPage              string `db:"cover_page" json:"cover_page"`                           // Page 1: Title, company info, date
	ExecutiveSummary       string `db:"executive_summary" json:"executive_summary"`             // Page 2: High-level overview from synthesis
	TableOfContents        string `db:"table_of_contents" json:"table_of_contents"`             // Page 3: Navigation links to all sections
	SWOTPage               string `db:"swot_page" json:"swot_page"`                             // Page 4: SWOT analysis visualization
	PESTELPage             string `db:"pestel_page" json:"pestel_page"`                         // Page 5: PESTEL analysis
	PorterPage             string `db:"porter_page" json:"porter_page"`                         // Page 6: Porter's Five Forces
	OKRPage                string `db:"okr_page" json:"okr_page"`                               // Page 7: OKRs with timelines
	BCGMatrixPage          string `db:"bcg_matrix_page" json:"bcg_matrix_page"`                // Page 8: BCG Matrix visualization
	ValuePropositionPage   string `db:"value_proposition_page" json:"value_proposition_page"`   // Page 9: Value Proposition Canvas
	BusinessModelPage      string `db:"business_model_page" json:"business_model_page"`         // Page 10: Business Model Canvas
	StrategicPrioritiesPage string `db:"strategic_priorities_page" json:"strategic_priorities_page"` // Page 11: Top priorities and action plan
	RisksAndMitigationPage string `db:"risks_mitigation_page" json:"risks_mitigation_page"`     // Page 12: Critical risks and mitigation strategies
	AppendixPage           string `db:"appendix_page" json:"appendix_page"`                     // Page 13: Raw data, sources, methodology

	// PDF Generation
	PDFURL               string     `db:"pdf_url" json:"pdf_url"`                         // Cloud storage URL for generated PDF
	PDFGeneratedAt       *time.Time `db:"pdf_generated_at" json:"pdf_generated_at"`       // When PDF was created
	PDFGenerationStatus  string     `db:"pdf_generation_status" json:"pdf_generation_status"` // pending, processing, completed, failed

	// Metadata
	Status               string     `db:"status" json:"status"`                 // pending, processing, completed, failed
	ErrorMessage         string     `db:"error_message" json:"error_message"`   // Error details if generation failed
	GenerationTimeMs     int64      `db:"generation_time_ms" json:"generation_time_ms"` // Time to generate HTML pages
	TotalPages           int        `db:"total_pages" json:"total_pages"`       // Should always be 13
	CreatedAt            time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at" json:"updated_at"`
	CompletedAt          *time.Time `db:"completed_at" json:"completed_at"`
}

// ReportMetadata contains summary information about the report
// Frontend developers: Use this for report listing and previews
type ReportMetadata struct {
	ID                  string     `json:"id"`
	SubmissionID        string     `json:"submission_id"`
	CompanyName         string     `json:"company_name"`         // From submission
	IndustryName        string     `json:"industry_name"`        // From submission
	Status              string     `json:"status"`
	PDFURL              string     `json:"pdf_url"`
	PDFGenerationStatus string     `json:"pdf_generation_status"`
	TotalPages          int        `json:"total_pages"`
	CreatedAt           time.Time  `json:"created_at"`
	CompletedAt         *time.Time `json:"completed_at"`
}

// PageTemplate defines the structure for HTML page generation
// Frontend developers: This shows what data each page contains
type PageTemplate struct {
	PageNumber int                    `json:"page_number"`
	PageTitle  string                 `json:"page_title"`
	Data       map[string]interface{} `json:"data"`       // Page-specific data
	Styles     string                 `json:"styles"`     // CSS styles for the page
	Template   string                 `json:"template"`   // HTML template name
}

// PDFGenerationRequest contains parameters for PDF generation
type PDFGenerationRequest struct {
	ReportID     string            `json:"report_id"`
	HTMLPages    []string          `json:"html_pages"`    // All 13 pages in order
	Options      PDFOptions        `json:"options"`       // PDF generation options
	OutputPath   string            `json:"output_path"`   // Where to save/upload PDF
	Metadata     map[string]string `json:"metadata"`      // PDF metadata (author, title, etc.)
}

// PDFOptions defines PDF generation parameters
type PDFOptions struct {
	Format            string  `json:"format"`             // A4, Letter, Legal
	Orientation       string  `json:"orientation"`        // portrait, landscape
	Margin            string  `json:"margin"`             // CSS margin value (e.g., "1cm")
	PrintBackground   bool    `json:"print_background"`   // Include background colors/images
	DisplayHeaderFooter bool  `json:"display_header_footer"` // Show page numbers
	Scale             float64 `json:"scale"`              // Scale factor (0.1 to 2)
	PreferCSSPageSize bool    `json:"prefer_css_page_size"` // Use CSS @page size
}

// DefaultPDFOptions returns sensible defaults for PDF generation
func DefaultPDFOptions() PDFOptions {
	return PDFOptions{
		Format:              "A4",
		Orientation:         "portrait",
		Margin:              "1cm",
		PrintBackground:     true,
		DisplayHeaderFooter: true,
		Scale:               1.0,
		PreferCSSPageSize:   false,
	}
}

// ReportProgress tracks report generation progress
// Frontend developers: Use this for real-time progress updates
type ReportProgress struct {
	ReportID         string    `json:"report_id"`
	CurrentPage      int       `json:"current_page"`      // Which page is being generated (1-13)
	TotalPages       int       `json:"total_pages"`       // Always 13
	Status           string    `json:"status"`            // processing, completed, failed
	PercentComplete  float64   `json:"percent_complete"`  // 0-100
	EstimatedTimeMs  int64     `json:"estimated_time_ms"` // Estimated remaining time
	LastUpdated      time.Time `json:"last_updated"`
}

// ReportSummary provides a high-level overview for dashboards
type ReportSummary struct {
	TotalReports      int            `json:"total_reports"`
	CompletedReports  int            `json:"completed_reports"`
	FailedReports     int            `json:"failed_reports"`
	PendingReports    int            `json:"pending_reports"`
	AverageGenTime    int64          `json:"average_gen_time_ms"`   // Average generation time
	TotalPDFsGenerated int           `json:"total_pdfs_generated"`
	RecentReports     []ReportMetadata `json:"recent_reports"`       // Last 10 reports
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
