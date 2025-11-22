package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// --- REPORT ENDPOINTS ---

// PreviewReport generates HTML on-the-fly for the Admin UI
// GET /api/submissions/:id/report/preview
func (h *Handler) PreviewReport(c *gin.Context) {
	submissionID := c.Param("id")

	// Get Analysis ID
	anal, err := h.analysisSvc.GetBySubmissionID(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Analysis not ready", Message: "Wait for AI to finish."})
		return
	}

	// Generate HTML (In Memory)
	pagesMap, err := h.reportSvc.GeneratePreview(c.Request.Context(), submissionID, anal.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Preview generation failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Preview failed", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ReportPreviewResponse{Pages: pagesMap})
}

// PublishReport freezes the report and generates the PDF
// POST /api/submissions/:id/report/publish
func (h *Handler) PublishReport(c *gin.Context) {
	submissionID := c.Param("id")

	anal, err := h.analysisSvc.GetBySubmissionID(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Analysis not ready"})
		return
	}

	pdfURL, err := h.reportSvc.Publish(c.Request.Context(), submissionID, anal.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Publish failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Publish failed", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ReportPublishResponse{
		ReportID: anal.ID, // Or create a new ID
		PDFURL:   pdfURL,
	})
}
