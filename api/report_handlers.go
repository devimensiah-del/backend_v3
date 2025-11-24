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
	AsyncService      *report.AsyncService // Optional: For async PDF generation
	SubmissionService *submission.Service
	Logger            zerolog.Logger
}

// NewReportHandlers creates a new report.Handlers instance with all dependencies
// AsyncService is optional - if nil, PDF generation runs synchronously
func NewReportHandlers(
	analysisSvc *analysis.Service,
	reportSvc *report.Service,
	submissionSvc *submission.Service,
	logger zerolog.Logger,
) *ReportHandlers {
	return &ReportHandlers{
		AnalysisService:   analysisSvc,
		ReportService:     reportSvc,
		AsyncService:      nil, // Will be set separately if async is enabled
		SubmissionService: submissionSvc,
		Logger:            logger,
	}
}

// SetAsyncService enables async PDF generation by setting the async service
func (h *ReportHandlers) SetAsyncService(asyncSvc *report.AsyncService) {
	h.AsyncService = asyncSvc
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
// If AsyncService is available, uses background job queue to prevent HTTP timeouts
// Otherwise, generates PDF synchronously
func (h *ReportHandlers) PublishReport(c *gin.Context) {
	submissionID := c.Param("id")

	anal, err := h.AnalysisService.GetBySubmissionID(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Analysis not ready"})
		return
	}

	// If async service is available, enqueue for background processing
	if h.AsyncService != nil {
		taskID, err := h.AsyncService.EnqueuePDFGeneration(c.Request.Context(), submissionID, anal.ID)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Failed to enqueue PDF generation")
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to start PDF generation",
				Message: err.Error(),
			})
			return
		}

		h.Logger.Info().
			Str("submission_id", submissionID).
			Str("analysis_id", anal.ID).
			Str("task_id", taskID).
			Msg("PDF generation task enqueued")

		// Return 202 Accepted with task ID for polling
		c.JSON(http.StatusAccepted, ReportPublishResponse{
			ReportID: anal.ID,
			TaskID:   taskID,
			Status:   "processing",
			Message:  "PDF generation started. Poll /api/submissions/:id/report/download for status.",
		})
		return
	}

	// Fallback: Generate PDF synchronously (may timeout for large reports)
	h.Logger.Warn().
		Str("submission_id", submissionID).
		Msg("AsyncService not configured, generating PDF synchronously")

	pdfURL, err := h.ReportService.Publish(c.Request.Context(), submissionID, anal.ID)
	if err != nil {
		h.Logger.Error().Err(err).Msg("PDF generation failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "PDF generation failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ReportPublishResponse{
		ReportID: anal.ID,
		Status:   "completed",
		PDFURL:   pdfURL,
		Message:  "PDF generated successfully",
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
