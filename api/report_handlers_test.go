package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend_v3/domain/analysis"
	"backend_v3/domain/report"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupReportTestHandler() (*Handler, *MockAnalysisService, *MockReportService) {
	gin.SetMode(gin.TestMode)

	mockAnalysisSvc := new(MockAnalysisService)
	mockReportSvc := new(MockReportService)

	handler := &Handler{
		analysisSvc: mockAnalysisSvc,
		reportSvc:   mockReportSvc,
		logger:      zerolog.Nop(),
	}

	return handler, mockAnalysisSvc, mockReportSvc
}

func TestPreviewReport_Success(t *testing.T) {
	handler, mockAnalysisSvc, mockReportSvc := setupReportTestHandler()

	submissionID := uuid.New()
	analysisID := uuid.New()

	mockAnalysis := &analysis.Analysis{
		ID:           analysisID.String(),
		SubmissionID: submissionID.String(),
		Status:       "approved",
		Version:      1,
	}

	expectedPages := map[string]string{
		"SWOT":   "<html><body>SWOT Analysis</body></html>",
		"PESTEL": "<html><body>PESTEL Analysis</body></html>",
		"Porter": "<html><body>Porter Analysis</body></html>",
	}

	mockAnalysisSvc.On("GetBySubmissionID", mock.Anything, submissionID.String()).Return(mockAnalysis, nil)
	mockReportSvc.On("GeneratePreview", mock.Anything, submissionID.String(), analysisID.String()).Return(expectedPages, nil)

	req, _ := http.NewRequest("GET", "/api/submissions/"+submissionID.String()+"/report/preview", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.GET("/api/submissions/:id/report/preview", handler.PreviewReport)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ReportPreviewResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, 3, len(response.Pages))
	assert.Contains(t, response.Pages, "SWOT")
	assert.Contains(t, response.Pages, "PESTEL")
	assert.Contains(t, response.Pages["SWOT"], "SWOT Analysis")

	mockAnalysisSvc.AssertExpectations(t)
	mockReportSvc.AssertExpectations(t)
}

func TestPreviewReport_AnalysisNotReady(t *testing.T) {
	handler, mockAnalysisSvc, _ := setupReportTestHandler()

	submissionID := uuid.New()

	mockAnalysisSvc.On("GetBySubmissionID", mock.Anything, submissionID.String()).Return(nil, errors.New("not found"))

	req, _ := http.NewRequest("GET", "/api/submissions/"+submissionID.String()+"/report/preview", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.GET("/api/submissions/:id/report/preview", handler.PreviewReport)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Error, "Analysis not ready")
	assert.Contains(t, response.Message, "Wait for AI to finish")

	mockAnalysisSvc.AssertExpectations(t)
}

func TestPreviewReport_GenerationFailed(t *testing.T) {
	handler, mockAnalysisSvc, mockReportSvc := setupReportTestHandler()

	submissionID := uuid.New()
	analysisID := uuid.New()

	mockAnalysis := &analysis.Analysis{
		ID:           analysisID.String(),
		SubmissionID: submissionID.String(),
		Status:       "approved",
		Version:      1,
	}

	mockAnalysisSvc.On("GetBySubmissionID", mock.Anything, submissionID.String()).Return(mockAnalysis, nil)
	mockReportSvc.On("GeneratePreview", mock.Anything, submissionID.String(), analysisID.String()).Return(nil, errors.New("template error"))

	req, _ := http.NewRequest("GET", "/api/submissions/"+submissionID.String()+"/report/preview", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.GET("/api/submissions/:id/report/preview", handler.PreviewReport)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Error, "Preview failed")

	mockAnalysisSvc.AssertExpectations(t)
	mockReportSvc.AssertExpectations(t)
}

func TestPublishReport_Success(t *testing.T) {
	handler, mockAnalysisSvc, mockReportSvc := setupReportTestHandler()

	submissionID := uuid.New()
	analysisID := uuid.New()

	mockAnalysis := &analysis.Analysis{
		ID:           analysisID.String(),
		SubmissionID: submissionID.String(),
		Status:       "approved",
		Version:      1,
	}

	expectedPDFURL := "https://storage.example.com/reports/" + submissionID.String() + ".pdf"

	mockAnalysisSvc.On("GetBySubmissionID", mock.Anything, submissionID.String()).Return(mockAnalysis, nil)
	mockReportSvc.On("Publish", mock.Anything, submissionID.String(), analysisID.String()).Return(expectedPDFURL, nil)

	req, _ := http.NewRequest("POST", "/api/submissions/"+submissionID.String()+"/report/publish", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/submissions/:id/report/publish", handler.PublishReport)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ReportPublishResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, analysisID.String(), response.ReportID)
	assert.Equal(t, expectedPDFURL, response.PDFURL)

	mockAnalysisSvc.AssertExpectations(t)
	mockReportSvc.AssertExpectations(t)
}

func TestPublishReport_AnalysisNotReady(t *testing.T) {
	handler, mockAnalysisSvc, _ := setupReportTestHandler()

	submissionID := uuid.New()

	mockAnalysisSvc.On("GetBySubmissionID", mock.Anything, submissionID.String()).Return(nil, errors.New("not found"))

	req, _ := http.NewRequest("POST", "/api/submissions/"+submissionID.String()+"/report/publish", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/submissions/:id/report/publish", handler.PublishReport)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Error, "Analysis not ready")

	mockAnalysisSvc.AssertExpectations(t)
}

func TestPublishReport_PublishFailed(t *testing.T) {
	handler, mockAnalysisSvc, mockReportSvc := setupReportTestHandler()

	submissionID := uuid.New()
	analysisID := uuid.New()

	mockAnalysis := &analysis.Analysis{
		ID:           analysisID.String(),
		SubmissionID: submissionID.String(),
		Status:       "approved",
		Version:      1,
	}

	mockAnalysisSvc.On("GetBySubmissionID", mock.Anything, submissionID.String()).Return(mockAnalysis, nil)
	mockReportSvc.On("Publish", mock.Anything, submissionID.String(), analysisID.String()).Return("", errors.New("PDF generation failed"))

	req, _ := http.NewRequest("POST", "/api/submissions/"+submissionID.String()+"/report/publish", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/submissions/:id/report/publish", handler.PublishReport)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Error, "Publish failed")

	mockAnalysisSvc.AssertExpectations(t)
	mockReportSvc.AssertExpectations(t)
}

func TestPublishReport_AnalysisNotApproved(t *testing.T) {
	handler, mockAnalysisSvc, mockReportSvc := setupReportTestHandler()

	submissionID := uuid.New()
	analysisID := uuid.New()

	mockAnalysis := &analysis.Analysis{
		ID:           analysisID.String(),
		SubmissionID: submissionID.String(),
		Status:       "completed", // Not approved
		Version:      1,
	}

	mockAnalysisSvc.On("GetBySubmissionID", mock.Anything, submissionID.String()).Return(mockAnalysis, nil)
	mockReportSvc.On("Publish", mock.Anything, submissionID.String(), analysisID.String()).Return("", errors.New("analysis not approved"))

	req, _ := http.NewRequest("POST", "/api/submissions/"+submissionID.String()+"/report/publish", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/submissions/:id/report/publish", handler.PublishReport)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Error, "Publish failed")

	mockAnalysisSvc.AssertExpectations(t)
	mockReportSvc.AssertExpectations(t)
}
