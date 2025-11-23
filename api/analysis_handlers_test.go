package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend_v3/domain/analysis"
	"backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupAnalysisTestHandler() (*Handler, *MockSubmissionService, *MockAnalysisService) {
	gin.SetMode(gin.TestMode)

	mockSubmissionSvc := new(MockSubmissionService)
	mockAnalysisSvc := new(MockAnalysisService)

	handler := &Handler{
		submissionSvc: mockSubmissionSvc,
		analysisSvc:   mockAnalysisSvc,
		logger:        zerolog.Nop(),
	}

	return handler, mockSubmissionSvc, mockAnalysisSvc
}

func TestGetAnalysis_Found(t *testing.T) {
	handler, mockSubmissionSvc, mockAnalysisSvc := setupAnalysisTestHandler()

	submissionID := uuid.New()
	analysisID := uuid.New()
	userID := uuid.New()

	mockSubmission := &submission.Submission{
		ID:     submissionID,
		UserID: &userID,
	}

	mockAnalysis := &analysis.Analysis{
		ID:           analysisID.String(),
		SubmissionID: submissionID.String(),
		Status:       "completed",
		Version:      1,
		PESTEL: &analysis.PESTEL{
			Political:     "Strong government support",
			Economic:      "Growing economy",
			Social:        "Digital transformation trend",
			Technological: "AI adoption increasing",
			Environmental: "Sustainability focus",
			Legal:         "Regulatory compliance required",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockSubmissionSvc.On("GetByID", mock.Anything, submissionID).Return(mockSubmission, nil)
	mockAnalysisSvc.On("GetBySubmissionID", mock.Anything, submissionID.String()).Return(mockAnalysis, nil)

	req, _ := http.NewRequest("GET", "/api/v1/submissions/"+submissionID.String()+"/analysis", nil)

	w := httptest.NewRecorder()
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("userID", userID.String())
		c.Set("userRole", "user")
		c.Next()
	})

	router.GET("/api/v1/submissions/:id/analysis", handler.GetAnalysis)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Contains(t, response, "analysis")
	analysisData := response["analysis"].(map[string]interface{})
	assert.Equal(t, analysisID.String(), analysisData["id"])
	assert.Equal(t, "completed", analysisData["status"])
	assert.Equal(t, float64(1), analysisData["version"])

	mockSubmissionSvc.AssertExpectations(t)
	mockAnalysisSvc.AssertExpectations(t)
}

func TestGetAnalysis_NotFound(t *testing.T) {
	handler, mockSubmissionSvc, mockAnalysisSvc := setupAnalysisTestHandler()

	submissionID := uuid.New()
	userID := uuid.New()

	mockSubmission := &submission.Submission{
		ID:     submissionID,
		UserID: &userID,
	}

	mockSubmissionSvc.On("GetByID", mock.Anything, submissionID).Return(mockSubmission, nil)
	mockAnalysisSvc.On("GetBySubmissionID", mock.Anything, submissionID.String()).Return(nil, errors.New("not found"))

	req, _ := http.NewRequest("GET", "/api/v1/submissions/"+submissionID.String()+"/analysis", nil)

	w := httptest.NewRecorder()
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("userID", userID.String())
		c.Set("userRole", "user")
		c.Next()
	})

	router.GET("/api/v1/submissions/:id/analysis", handler.GetAnalysis)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "Analysis not found")

	mockSubmissionSvc.AssertExpectations(t)
	mockAnalysisSvc.AssertExpectations(t)
}

func TestGetAnalysis_Unauthorized(t *testing.T) {
	handler, mockSubmissionSvc, _ := setupAnalysisTestHandler()

	submissionID := uuid.New()
	ownerID := uuid.New()
	attackerID := uuid.New()

	mockSubmission := &submission.Submission{
		ID:     submissionID,
		UserID: &ownerID,
	}

	mockSubmissionSvc.On("GetByID", mock.Anything, submissionID).Return(mockSubmission, nil)

	req, _ := http.NewRequest("GET", "/api/v1/submissions/"+submissionID.String()+"/analysis", nil)

	w := httptest.NewRecorder()
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("userID", attackerID.String())
		c.Set("userRole", "user")
		c.Next()
	})

	router.GET("/api/v1/submissions/:id/analysis", handler.GetAnalysis)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "You don't have permission")

	mockSubmissionSvc.AssertExpectations(t)
}

func TestUpdateAnalysis_ValidUpdate(t *testing.T) {
	handler, _, mockAnalysisSvc := setupAnalysisTestHandler()

	analysisID := uuid.New()

	updateData := map[string]interface{}{
		"PESTEL": map[string]interface{}{
			"Political": "Updated political analysis",
		},
	}

	updatedAnalysis := &analysis.Analysis{
		ID:           analysisID.String(),
		SubmissionID: uuid.New().String(),
		Status:       "completed",
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mockAnalysisSvc.On("UpdateFields", mock.Anything, analysisID.String(), updateData).Return(updatedAnalysis, nil)

	bodyBytes, _ := json.Marshal(updateData)
	req, _ := http.NewRequest("PUT", "/api/v1/admin/analysis/"+analysisID.String(), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router := gin.New()
	router.PUT("/api/v1/admin/analysis/:id", handler.UpdateAnalysis)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Contains(t, response, "analysis")

	mockAnalysisSvc.AssertExpectations(t)
}

func TestUpdateAnalysis_InvalidJSON(t *testing.T) {
	handler, _, _ := setupAnalysisTestHandler()

	analysisID := uuid.New()

	req, _ := http.NewRequest("PUT", "/api/v1/admin/analysis/"+analysisID.String(), bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router := gin.New()
	router.PUT("/api/v1/admin/analysis/:id", handler.UpdateAnalysis)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Error, "Invalid request")
}

func TestCreateAnalysisVersion_Success(t *testing.T) {
	handler, _, mockAnalysisSvc := setupAnalysisTestHandler()

	parentID := uuid.New()

	edits := map[string]interface{}{
		"SWOT": map[string]interface{}{
			"Strengths": []string{"Updated strength"},
		},
	}

	newVersion := &analysis.Analysis{
		ID:              uuid.New().String(),
		SubmissionID:    uuid.New().String(),
		Status:          "draft",
		Version:         2,
		ParentAnalysisID: stringToPtr(parentID.String()),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	mockAnalysisSvc.On("CreateVersion", mock.Anything, parentID.String(), edits).Return(newVersion, nil)

	reqBody := map[string]interface{}{
		"edits": edits,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/admin/analysis/"+parentID.String()+"/version", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/analysis/:id/version", handler.CreateAnalysisVersion)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Contains(t, response, "analysis")
	analysisData := response["analysis"].(map[string]interface{})
	assert.Equal(t, float64(2), analysisData["version"])
	assert.Equal(t, "draft", analysisData["status"])

	mockAnalysisSvc.AssertExpectations(t)
}

func TestCreateAnalysisVersion_InvalidParentID(t *testing.T) {
	handler, _, mockAnalysisSvc := setupAnalysisTestHandler()

	parentID := uuid.New()

	mockAnalysisSvc.On("CreateVersion", mock.Anything, parentID.String(), mock.Anything).Return(nil, errors.New("parent not found"))

	req, _ := http.NewRequest("POST", "/api/v1/admin/analysis/"+parentID.String()+"/version", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/analysis/:id/version", handler.CreateAnalysisVersion)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Error, "Version creation failed")

	mockAnalysisSvc.AssertExpectations(t)
}

func TestApproveAnalysis_Success(t *testing.T) {
	handler, _, mockAnalysisSvc := setupAnalysisTestHandler()

	analysisID := uuid.New()

	completedAnalysis := &analysis.Analysis{
		ID:           analysisID.String(),
		SubmissionID: uuid.New().String(),
		Status:       "completed",
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	approvedAnalysis := &analysis.Analysis{
		ID:           analysisID.String(),
		SubmissionID: uuid.New().String(),
		Status:       "approved",
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mockAnalysisSvc.On("GetByID", mock.Anything, analysisID.String()).Return(completedAnalysis, nil).Once()
	mockAnalysisSvc.On("Approve", mock.Anything, analysisID.String()).Return(nil)
	mockAnalysisSvc.On("GetByID", mock.Anything, analysisID.String()).Return(approvedAnalysis, nil).Once()

	req, _ := http.NewRequest("POST", "/api/v1/admin/analysis/"+analysisID.String()+"/approve", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/analysis/:id/approve", handler.ApproveAnalysis)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Contains(t, response, "analysis")
	assert.Contains(t, response, "message")
	assert.Contains(t, response["message"], "PDF generation job enqueued")

	analysisData := response["analysis"].(map[string]interface{})
	assert.Equal(t, "approved", analysisData["status"])

	mockAnalysisSvc.AssertExpectations(t)
}

func TestApproveAnalysis_InvalidStatus(t *testing.T) {
	handler, _, mockAnalysisSvc := setupAnalysisTestHandler()

	analysisID := uuid.New()

	draftAnalysis := &analysis.Analysis{
		ID:           analysisID.String(),
		SubmissionID: uuid.New().String(),
		Status:       "draft",
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mockAnalysisSvc.On("GetByID", mock.Anything, analysisID.String()).Return(draftAnalysis, nil)

	req, _ := http.NewRequest("POST", "/api/v1/admin/analysis/"+analysisID.String()+"/approve", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/analysis/:id/approve", handler.ApproveAnalysis)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "Analysis must be in 'completed' status")

	mockAnalysisSvc.AssertExpectations(t)
}

func TestApproveAnalysis_NotFound(t *testing.T) {
	handler, _, mockAnalysisSvc := setupAnalysisTestHandler()

	analysisID := uuid.New()

	mockAnalysisSvc.On("GetByID", mock.Anything, analysisID.String()).Return(nil, errors.New("not found"))

	req, _ := http.NewRequest("POST", "/api/v1/admin/analysis/"+analysisID.String()+"/approve", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/analysis/:id/approve", handler.ApproveAnalysis)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "Analysis not found")

	mockAnalysisSvc.AssertExpectations(t)
}

func TestSendAnalysis_Success(t *testing.T) {
	handler, _, mockAnalysisSvc := setupAnalysisTestHandler()

	analysisID := uuid.New()

	approvedAnalysis := &analysis.Analysis{
		ID:           analysisID.String(),
		SubmissionID: uuid.New().String(),
		Status:       "approved",
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	sentAnalysis := &analysis.Analysis{
		ID:           analysisID.String(),
		SubmissionID: uuid.New().String(),
		Status:       "sent",
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	reqBody := map[string]string{
		"userEmail": "user@example.com",
	}

	mockAnalysisSvc.On("GetByID", mock.Anything, analysisID.String()).Return(approvedAnalysis, nil).Once()
	mockAnalysisSvc.On("Send", mock.Anything, analysisID.String(), "user@example.com").Return(nil)
	mockAnalysisSvc.On("GetByID", mock.Anything, analysisID.String()).Return(sentAnalysis, nil).Once()

	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/admin/analysis/"+analysisID.String()+"/send", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/analysis/:id/send", handler.SendAnalysis)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Contains(t, response, "analysis")
	assert.Contains(t, response, "message")
	assert.Contains(t, response["message"], "Analysis sent to user successfully")

	analysisData := response["analysis"].(map[string]interface{})
	assert.Equal(t, "sent", analysisData["status"])

	mockAnalysisSvc.AssertExpectations(t)
}

func TestSendAnalysis_InvalidStatus(t *testing.T) {
	handler, _, mockAnalysisSvc := setupAnalysisTestHandler()

	analysisID := uuid.New()

	completedAnalysis := &analysis.Analysis{
		ID:           analysisID.String(),
		SubmissionID: uuid.New().String(),
		Status:       "completed",
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	reqBody := map[string]string{
		"userEmail": "user@example.com",
	}

	mockAnalysisSvc.On("GetByID", mock.Anything, analysisID.String()).Return(completedAnalysis, nil)

	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/admin/analysis/"+analysisID.String()+"/send", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/analysis/:id/send", handler.SendAnalysis)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "Analysis must be in 'approved' status")

	mockAnalysisSvc.AssertExpectations(t)
}

func TestSendAnalysis_MissingEmail(t *testing.T) {
	handler, _, _ := setupAnalysisTestHandler()

	analysisID := uuid.New()

	reqBody := map[string]string{}

	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/admin/analysis/"+analysisID.String()+"/send", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/analysis/:id/send", handler.SendAnalysis)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "Valid user email is required")
}

func TestSendAnalysis_NotFound(t *testing.T) {
	handler, _, mockAnalysisSvc := setupAnalysisTestHandler()

	analysisID := uuid.New()

	reqBody := map[string]string{
		"userEmail": "user@example.com",
	}

	mockAnalysisSvc.On("GetByID", mock.Anything, analysisID.String()).Return(nil, errors.New("not found"))

	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/admin/analysis/"+analysisID.String()+"/send", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/analysis/:id/send", handler.SendAnalysis)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "Analysis not found")

	mockAnalysisSvc.AssertExpectations(t)
}
