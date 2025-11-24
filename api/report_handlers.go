package api

import (
	"net/http"

	"backend_v3/domain/analysis"
	"backend_v3/domain/report"
	"backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog" // Needed for logger
)

// Handlers holds all service dependencies and configuration for report handlers
type ReportHandlers struct {
	AnalysisService   *analysis.Service
	ReportService     *report.Service
	SubmissionService *submission.Service
	Logger            zerolog.Logger
}

// NewReportHandlers creates a new report.Handlers instance with all dependencies
func NewReportHandlers(
	analysisSvc *analysis.Service,
	reportSvc *report.Service,
	submissionSvc *submission.Service,
	logger zerolog.Logger,
) *ReportHandlers {
	return &ReportHandlers{
		AnalysisService:   analysisSvc,
		ReportService:     reportSvc,
		SubmissionService: submissionSvc,
		Logger:            logger,
	}
}

// --- REPORT ENDPOINTS ---

// PreviewReport generates HTML on-the-fly for the Admin UI
// GET /api/submissions/:id/report/preview
func (h *ReportHandlers) PreviewReport(c *gin.Context) {
	submissionID := c.Param("id")

	// Get Analysis ID
	anal, err := h.AnalysisService.GetBySubmissionID(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Analysis not ready", Message: "Wait for AI to finish."})
		return
	}

	// Generate HTML (In Memory)
	pagesMap, err := h.ReportService.GeneratePreview(c.Request.Context(), submissionID, anal.ID)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Preview generation failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Preview failed", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ReportPreviewResponse{Pages: pagesMap})
}

// PublishReport freezes the report and generates the PDF
// POST /api/submissions/:id/report/publish
func (h *ReportHandlers) PublishReport(c *gin.Context) {
	submissionID := c.Param("id")

	anal, err := h.AnalysisService.GetBySubmissionID(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Analysis not ready"})
		return
	}

	pdfURL, err := h.ReportService.Publish(c.Request.Context(), submissionID, anal.ID)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Publish failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Publish failed", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ReportPublishResponse{
		ReportID: anal.ID, // Or create a new ID
		PDFURL:   pdfURL,
	})
}

// DownloadReport retrieves the PDF URL for a submission's report
// GET /api/v1/submissions/:id/report/download
func (h *ReportHandlers) DownloadReport(c *gin.Context) {
	submissionID := c.Param("id")

	// Parse and validate UUID
	subUUID, err := parseUUID(submissionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid ID",
			Message: "Invalid submission ID format",
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Get submission to verify ownership (unless admin)
	submission, err := h.SubmissionService.GetByID(c.Request.Context(), subUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Submission not found",
		})
		return
	}

	// Check if user owns this submission (admins can access any)
	role, _ := c.Get("userRole")
	userRole := ""
	if r, ok := role.(string); ok {
		userRole = r
	}

	// Convert userID string to UUID for comparison
	userIDStr := userID.(string)
	currentUserUUID, err := parseUUID(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal error",
			Message: "Invalid user ID format",
		})
		return
	}

	if submission.UserID != nil && *submission.UserID != currentUserUUID {
		if userRole != "admin" && userRole != "super_admin" {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "Forbidden",
				Message: "You don't have permission to download this report",
			})
			return
		}
	}

	// Get report by submission ID
	report, err := h.ReportService.GetBySubmissionID(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "Report not found for this submission",
		})
		return
	}

	// Check if PDF has been generated
	if report.PDFURL == "" {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "PDF not ready",
			Message: "Report PDF has not been generated yet. Please wait for analysis approval.",
		})
		return
	}

	// Check PDF generation status
	if report.PDFGenerationStatus != "completed" {
		c.JSON(http.StatusAccepted, gin.H{
			"status":  report.PDFGenerationStatus,
			"message": "PDF generation is still in progress. Please try again later.",
		})
		return
	}

	// Return the PDF URL (client can download directly from cloud storage)
	c.JSON(http.StatusOK, gin.H{
		"pdf_url":    report.PDFURL,
		"report_id":  report.ID,
		"created_at": report.CreatedAt,
	})
}
