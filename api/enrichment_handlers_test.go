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

	"backend_v3/domain/enrichment"
	"backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupEnrichmentTestHandler() (*Handler, *MockSubmissionService, *MockEnrichmentService) {
	gin.SetMode(gin.TestMode)

	mockSubmissionSvc := new(MockSubmissionService)
	mockEnrichmentSvc := new(MockEnrichmentService)

	handler := &Handler{
		submissionSvc: mockSubmissionSvc,
		enrichmentSvc: mockEnrichmentSvc,
		logger:        zerolog.Nop(),
	}

	return handler, mockSubmissionSvc, mockEnrichmentSvc
}

func TestGetEnrichment_Found(t *testing.T) {
	handler, mockSubmissionSvc, mockEnrichmentSvc := setupEnrichmentTestHandler()

	submissionID := uuid.New()
	enrichmentID := uuid.New()
	userID := uuid.New()

	mockSubmission := &submission.Submission{
		ID:     submissionID,
		UserID: &userID,
	}

	mockEnrichment := &enrichment.Enrichment{
		ID:           enrichmentID,
		SubmissionID: submissionID,
		Status:       enrichment.StatusFinished,
		Progress:     100,
		CurrentStep:  "completed",
		EnrichedData: map[string]interface{}{
			"industry": "Technology",
			"market":   "Global",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockSubmissionSvc.On("GetByID", mock.Anything, submissionID).Return(mockSubmission, nil)
	mockEnrichmentSvc.On("GetBySubmissionID", mock.Anything, submissionID).Return(mockEnrichment, nil)

	req, _ := http.NewRequest("GET", "/api/v1/submissions/"+submissionID.String()+"/enrichment", nil)

	w := httptest.NewRecorder()
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("userID", userID.String())
		c.Set("userRole", "user")
		c.Next()
	})

	router.GET("/api/v1/submissions/:id/enrichment", handler.GetEnrichment)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Contains(t, response, "enrichment")
	enrichmentData := response["enrichment"].(map[string]interface{})
	assert.Equal(t, enrichmentID.String(), enrichmentData["id"])
	assert.Equal(t, "finished", enrichmentData["status"])
	assert.Equal(t, float64(100), enrichmentData["progress"])

	mockSubmissionSvc.AssertExpectations(t)
	mockEnrichmentSvc.AssertExpectations(t)
}

func TestGetEnrichment_NotFound(t *testing.T) {
	handler, mockSubmissionSvc, mockEnrichmentSvc := setupEnrichmentTestHandler()

	submissionID := uuid.New()
	userID := uuid.New()

	mockSubmission := &submission.Submission{
		ID:     submissionID,
		UserID: &userID,
	}

	mockSubmissionSvc.On("GetByID", mock.Anything, submissionID).Return(mockSubmission, nil)
	mockEnrichmentSvc.On("GetBySubmissionID", mock.Anything, submissionID).Return(nil, errors.New("not found"))

	req, _ := http.NewRequest("GET", "/api/v1/submissions/"+submissionID.String()+"/enrichment", nil)

	w := httptest.NewRecorder()
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("userID", userID.String())
		c.Set("userRole", "user")
		c.Next()
	})

	router.GET("/api/v1/submissions/:id/enrichment", handler.GetEnrichment)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "Enrichment data not found")

	mockSubmissionSvc.AssertExpectations(t)
	mockEnrichmentSvc.AssertExpectations(t)
}

func TestGetEnrichment_Unauthorized(t *testing.T) {
	handler, mockSubmissionSvc, _ := setupEnrichmentTestHandler()

	submissionID := uuid.New()
	ownerID := uuid.New()
	attackerID := uuid.New()

	mockSubmission := &submission.Submission{
		ID:     submissionID,
		UserID: &ownerID,
	}

	mockSubmissionSvc.On("GetByID", mock.Anything, submissionID).Return(mockSubmission, nil)

	req, _ := http.NewRequest("GET", "/api/v1/submissions/"+submissionID.String()+"/enrichment", nil)

	w := httptest.NewRecorder()
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("userID", attackerID.String())
		c.Set("userRole", "user")
		c.Next()
	})

	router.GET("/api/v1/submissions/:id/enrichment", handler.GetEnrichment)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "You don't have permission")

	mockSubmissionSvc.AssertExpectations(t)
}

func TestUpdateEnrichment_ValidUpdate(t *testing.T) {
	handler, _, mockEnrichmentSvc := setupEnrichmentTestHandler()

	enrichmentID := uuid.New()
	submissionID := uuid.New()

	updateData := map[string]interface{}{
		"industry": "Healthcare",
		"market":   "Brazil",
	}

	updatedEnrichment := &enrichment.Enrichment{
		ID:           enrichmentID,
		SubmissionID: submissionID,
		Status:       enrichment.StatusFinished,
		Progress:     100,
		CurrentStep:  "completed",
		EnrichedData: updateData,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mockEnrichmentSvc.On("UpdateFields", mock.Anything, enrichmentID, updateData).Return(updatedEnrichment, nil)

	bodyBytes, _ := json.Marshal(updateData)
	req, _ := http.NewRequest("PUT", "/api/v1/admin/enrichment/"+enrichmentID.String(), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router := gin.New()
	router.PUT("/api/v1/admin/enrichment/:id", handler.UpdateEnrichment)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Contains(t, response, "enrichment")
	enrichmentData := response["enrichment"].(map[string]interface{})
	assert.Equal(t, enrichmentID.String(), enrichmentData["id"])

	mockEnrichmentSvc.AssertExpectations(t)
}

func TestUpdateEnrichment_InvalidJSON(t *testing.T) {
	handler, _, _ := setupEnrichmentTestHandler()

	enrichmentID := uuid.New()

	req, _ := http.NewRequest("PUT", "/api/v1/admin/enrichment/"+enrichmentID.String(), bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router := gin.New()
	router.PUT("/api/v1/admin/enrichment/:id", handler.UpdateEnrichment)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Error, "Invalid request")
}

func TestUpdateEnrichment_InvalidUUID(t *testing.T) {
	handler, _, _ := setupEnrichmentTestHandler()

	updateData := map[string]interface{}{"field": "value"}
	bodyBytes, _ := json.Marshal(updateData)

	req, _ := http.NewRequest("PUT", "/api/v1/admin/enrichment/invalid-uuid", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router := gin.New()
	router.PUT("/api/v1/admin/enrichment/:id", handler.UpdateEnrichment)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "Invalid enrichment ID format")
}

func TestApproveEnrichment_Success(t *testing.T) {
	handler, _, mockEnrichmentSvc := setupEnrichmentTestHandler()

	enrichmentID := uuid.New()
	submissionID := uuid.New()

	approvedEnrichment := &enrichment.Enrichment{
		ID:           enrichmentID,
		SubmissionID: submissionID,
		Status:       enrichment.StatusApproved,
		Progress:     100,
		CurrentStep:  "approved",
		EnrichedData: map[string]interface{}{},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mockEnrichmentSvc.On("Approve", mock.Anything, enrichmentID).Return(nil)
	mockEnrichmentSvc.On("GetByID", mock.Anything, enrichmentID).Return(approvedEnrichment, nil)

	req, _ := http.NewRequest("POST", "/api/v1/admin/enrichment/"+enrichmentID.String()+"/approve", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/enrichment/:id/approve", handler.ApproveEnrichment)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Contains(t, response, "enrichment")
	assert.Contains(t, response, "message")
	assert.Contains(t, response["message"], "analysis job enqueued")

	enrichmentData := response["enrichment"].(map[string]interface{})
	assert.Equal(t, "approved", enrichmentData["status"])

	mockEnrichmentSvc.AssertExpectations(t)
}

func TestApproveEnrichment_InvalidStatus(t *testing.T) {
	handler, _, mockEnrichmentSvc := setupEnrichmentTestHandler()

	enrichmentID := uuid.New()

	// Service returns error when status is not "finished"
	mockEnrichmentSvc.On("Approve", mock.Anything, enrichmentID).Return(errors.New("invalid status"))

	req, _ := http.NewRequest("POST", "/api/v1/admin/enrichment/"+enrichmentID.String()+"/approve", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/enrichment/:id/approve", handler.ApproveEnrichment)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Error, "Approval failed")

	mockEnrichmentSvc.AssertExpectations(t)
}

func TestApproveEnrichment_NotFound(t *testing.T) {
	handler, _, mockEnrichmentSvc := setupEnrichmentTestHandler()

	enrichmentID := uuid.New()

	mockEnrichmentSvc.On("Approve", mock.Anything, enrichmentID).Return(errors.New("not found"))

	req, _ := http.NewRequest("POST", "/api/v1/admin/enrichment/"+enrichmentID.String()+"/approve", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/enrichment/:id/approve", handler.ApproveEnrichment)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockEnrichmentSvc.AssertExpectations(t)
}

func TestApproveEnrichment_InvalidUUID(t *testing.T) {
	handler, _, _ := setupEnrichmentTestHandler()

	req, _ := http.NewRequest("POST", "/api/v1/admin/enrichment/invalid-uuid/approve", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/enrichment/:id/approve", handler.ApproveEnrichment)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "Invalid enrichment ID format")
}
