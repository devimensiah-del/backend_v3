package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend_v3/domain/enrichment"
	"backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupAdminTestHandler() (*Handler, *MockSubmissionService, *MockEnrichmentService, *MockAsynqClient) {
	gin.SetMode(gin.TestMode)

	mockSubmissionSvc := new(MockSubmissionService)
	mockEnrichmentSvc := new(MockEnrichmentService)
	mockAsynqClient := new(MockAsynqClient)

	handler := &Handler{
		submissionSvc: mockSubmissionSvc,
		enrichmentSvc: mockEnrichmentSvc,
		asynqClient:   mockAsynqClient,
		logger:        zerolog.Nop(),
	}

	return handler, mockSubmissionSvc, mockEnrichmentSvc, mockAsynqClient
}

func TestGetAnalytics_Success(t *testing.T) {
	handler, mockSubmissionSvc, _, _ := setupAdminTestHandler()

	expectedAnalytics := &submission.AnalyticsData{
		TotalSubmissions:     100,
		ActiveSubmissions:    25,
		CompletedSubmissions: 75,
		Revenue:              50000.00,
	}

	mockSubmissionSvc.On("GetAnalytics", mock.Anything).Return(expectedAnalytics, nil)

	req, _ := http.NewRequest("GET", "/api/v1/admin/analytics", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.GET("/api/v1/admin/analytics", handler.GetAnalytics)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response AnalyticsResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, 100, response.TotalSubmissions)
	assert.Equal(t, 25, response.ActiveSubmissions)
	assert.Equal(t, 75, response.CompletedSubmissions)
	assert.Equal(t, 50000.00, response.Revenue)

	mockSubmissionSvc.AssertExpectations(t)
}

func TestGetAnalytics_Error(t *testing.T) {
	handler, mockSubmissionSvc, _, _ := setupAdminTestHandler()

	mockSubmissionSvc.On("GetAnalytics", mock.Anything).Return(nil, errors.New("database error"))

	req, _ := http.NewRequest("GET", "/api/v1/admin/analytics", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.GET("/api/v1/admin/analytics", handler.GetAnalytics)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Error, "Failed to retrieve analytics")

	mockSubmissionSvc.AssertExpectations(t)
}

func TestGetAnalytics_AdminOnly(t *testing.T) {
	handler, mockSubmissionSvc, _, _ := setupAdminTestHandler()

	expectedAnalytics := &submission.AnalyticsData{
		TotalSubmissions:     100,
		ActiveSubmissions:    25,
		CompletedSubmissions: 75,
		Revenue:              50000.00,
	}

	mockSubmissionSvc.On("GetAnalytics", mock.Anything).Return(expectedAnalytics, nil)

	req, _ := http.NewRequest("GET", "/api/v1/admin/analytics", nil)

	w := httptest.NewRecorder()
	router := gin.New()

	// Mock admin auth middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", uuid.New().String())
		c.Set("userRole", "admin")
		c.Next()
	})

	router.GET("/api/v1/admin/analytics", handler.GetAnalytics)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSubmissionSvc.AssertExpectations(t)
}

func TestListSubmissions_AllSubmissions(t *testing.T) {
	handler, mockSubmissionSvc, _, _ := setupAdminTestHandler()

	submissions := []*submission.Submission{
		{ID: uuid.New(), CompanyName: "Company 1"},
		{ID: uuid.New(), CompanyName: "Company 2"},
		{ID: uuid.New(), CompanyName: "Company 3"},
	}

	mockSubmissionSvc.On("ListAll", mock.Anything, mock.MatchedBy(func(opts *submission.ListOptions) bool {
		return opts.Limit == 20 && opts.Offset == 0
	})).Return(submissions, 3, nil)

	req, _ := http.NewRequest("GET", "/api/v1/admin/submissions", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.GET("/api/v1/admin/submissions", handler.ListSubmissions)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SubmissionListResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, 1, response.Page)
	assert.Equal(t, 20, response.PageSize)
	assert.Equal(t, 3, response.Total)
	assert.Equal(t, 1, response.TotalPages)

	mockSubmissionSvc.AssertExpectations(t)
}

func TestListSubmissions_WithFilters(t *testing.T) {
	handler, mockSubmissionSvc, _, _ := setupAdminTestHandler()

	status := submission.StatusReceived
	email := "test@example.com"

	submissions := []*submission.Submission{
		{ID: uuid.New(), CompanyName: "Filtered Company"},
	}

	mockSubmissionSvc.On("ListAll", mock.Anything, mock.MatchedBy(func(opts *submission.ListOptions) bool {
		return opts.Status != nil && *opts.Status == status && opts.Email != nil && *opts.Email == email
	})).Return(submissions, 1, nil)

	req, _ := http.NewRequest("GET", "/api/v1/admin/submissions?status=received&email=test@example.com", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.GET("/api/v1/admin/submissions", handler.ListSubmissions)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SubmissionListResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, 1, response.Total)

	mockSubmissionSvc.AssertExpectations(t)
}

func TestListSubmissions_Pagination(t *testing.T) {
	handler, mockSubmissionSvc, _, _ := setupAdminTestHandler()

	submissions := []*submission.Submission{
		{ID: uuid.New(), CompanyName: "Company 11"},
		{ID: uuid.New(), CompanyName: "Company 12"},
	}

	mockSubmissionSvc.On("ListAll", mock.Anything, mock.MatchedBy(func(opts *submission.ListOptions) bool {
		return opts.Limit == 10 && opts.Offset == 10
	})).Return(submissions, 25, nil)

	req, _ := http.NewRequest("GET", "/api/v1/admin/submissions?page=2&limit=10", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.GET("/api/v1/admin/submissions", handler.ListSubmissions)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SubmissionListResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, 2, response.Page)
	assert.Equal(t, 10, response.PageSize)
	assert.Equal(t, 25, response.Total)
	assert.Equal(t, 3, response.TotalPages)

	mockSubmissionSvc.AssertExpectations(t)
}

func TestGetSubmissionAdmin_Success(t *testing.T) {
	handler, mockSubmissionSvc, _, _ := setupAdminTestHandler()

	submissionID := uuid.New()

	expectedSubmission := &submission.Submission{
		ID:          submissionID,
		CompanyName: "Test Company",
		Status:      submission.StatusReceived,
	}

	mockSubmissionSvc.On("GetByID", mock.Anything, submissionID).Return(expectedSubmission, nil)

	req, _ := http.NewRequest("GET", "/api/v1/admin/submissions/"+submissionID.String(), nil)

	w := httptest.NewRecorder()
	router := gin.New()

	// Mock admin auth
	router.Use(func(c *gin.Context) {
		c.Set("userID", uuid.New().String())
		c.Set("userRole", "admin")
		c.Next()
	})

	router.GET("/api/v1/admin/submissions/:id", handler.GetSubmissionAdmin)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSubmissionSvc.AssertExpectations(t)
}

func TestGetSubmissionAdmin_NotFound(t *testing.T) {
	handler, mockSubmissionSvc, _, _ := setupAdminTestHandler()

	submissionID := uuid.New()

	mockSubmissionSvc.On("GetByID", mock.Anything, submissionID).Return(nil, errors.New("not found"))

	req, _ := http.NewRequest("GET", "/api/v1/admin/submissions/"+submissionID.String(), nil)

	w := httptest.NewRecorder()
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("userID", uuid.New().String())
		c.Set("userRole", "admin")
		c.Next()
	})

	router.GET("/api/v1/admin/submissions/:id", handler.GetSubmissionAdmin)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "Submission not found")

	mockSubmissionSvc.AssertExpectations(t)
}

func TestGetSubmissionAdmin_InvalidUUID(t *testing.T) {
	handler, _, _, _ := setupAdminTestHandler()

	req, _ := http.NewRequest("GET", "/api/v1/admin/submissions/invalid-uuid", nil)

	w := httptest.NewRecorder()
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("userID", uuid.New().String())
		c.Set("userRole", "admin")
		c.Next()
	})

	router.GET("/api/v1/admin/submissions/:id", handler.GetSubmissionAdmin)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "Invalid submission ID format")
}

func TestGetSubmissionAdmin_NonAdminForbidden(t *testing.T) {
	handler, _, _, _ := setupAdminTestHandler()

	submissionID := uuid.New()

	req, _ := http.NewRequest("GET", "/api/v1/admin/submissions/"+submissionID.String(), nil)

	w := httptest.NewRecorder()
	router := gin.New()

	// Mock non-admin user
	router.Use(func(c *gin.Context) {
		c.Set("userID", uuid.New().String())
		c.Set("userRole", "user")
		c.Next()
	})

	router.GET("/api/v1/admin/submissions/:id", handler.GetSubmissionAdmin)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "Admin access required")
}

func TestRetryEnrichment_Success(t *testing.T) {
	handler, mockSubmissionSvc, _, mockAsynqClient := setupAdminTestHandler()

	submissionID := uuid.New().String()

	expectedSubmission := &submission.Submission{
		ID:          uuid.MustParse(submissionID),
		CompanyName: "Test Company",
		Status:      submission.StatusReceived,
	}

	mockSubmissionSvc.On("GetByID", mock.Anything, submissionID).Return(expectedSubmission, nil)
	mockAsynqClient.On("Enqueue", mock.Anything, mock.Anything).Return(&asynq.TaskInfo{}, nil)

	req, _ := http.NewRequest("POST", "/api/v1/admin/submissions/"+submissionID+"/retry-enrichment", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/submissions/:id/retry-enrichment", handler.RetryEnrichment)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response MessageResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "Enrichment retry enqueued successfully")

	mockSubmissionSvc.AssertExpectations(t)
	mockAsynqClient.AssertExpectations(t)
}

func TestRetryEnrichment_SubmissionNotFound(t *testing.T) {
	handler, mockSubmissionSvc, _, _ := setupAdminTestHandler()

	submissionID := uuid.New().String()

	mockSubmissionSvc.On("GetByID", mock.Anything, submissionID).Return(nil, errors.New("not found"))

	req, _ := http.NewRequest("POST", "/api/v1/admin/submissions/"+submissionID+"/retry-enrichment", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/submissions/:id/retry-enrichment", handler.RetryEnrichment)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "Submission not found")

	mockSubmissionSvc.AssertExpectations(t)
}

func TestRetryAnalysis_Success(t *testing.T) {
	handler, mockSubmissionSvc, mockEnrichmentSvc, mockAsynqClient := setupAdminTestHandler()

	submissionID := uuid.New()
	enrichmentID := uuid.New()

	expectedSubmission := &submission.Submission{
		ID:          submissionID,
		CompanyName: "Test Company",
		Status:      submission.StatusReceived,
	}

	expectedEnrichment := &enrichment.Enrichment{
		ID:           enrichmentID,
		SubmissionID: submissionID,
		Status:       enrichment.StatusApproved,
	}

	mockSubmissionSvc.On("GetByID", mock.Anything, submissionID).Return(expectedSubmission, nil)
	mockEnrichmentSvc.On("GetBySubmissionID", mock.Anything, submissionID).Return(expectedEnrichment, nil)
	mockAsynqClient.On("Enqueue", mock.Anything, mock.Anything).Return(&asynq.TaskInfo{}, nil)

	req, _ := http.NewRequest("POST", "/api/v1/admin/submissions/"+submissionID.String()+"/retry-analysis", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/submissions/:id/retry-analysis", handler.RetryAnalysis)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response MessageResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "Analysis retry enqueued successfully")

	mockSubmissionSvc.AssertExpectations(t)
	mockEnrichmentSvc.AssertExpectations(t)
	mockAsynqClient.AssertExpectations(t)
}

func TestRetryAnalysis_EnrichmentRequired(t *testing.T) {
	handler, mockSubmissionSvc, mockEnrichmentSvc, _ := setupAdminTestHandler()

	submissionID := uuid.New()

	expectedSubmission := &submission.Submission{
		ID:          submissionID,
		CompanyName: "Test Company",
		Status:      submission.StatusReceived,
	}

	mockSubmissionSvc.On("GetByID", mock.Anything, submissionID).Return(expectedSubmission, nil)
	mockEnrichmentSvc.On("GetBySubmissionID", mock.Anything, submissionID).Return(nil, errors.New("not found"))

	req, _ := http.NewRequest("POST", "/api/v1/admin/submissions/"+submissionID.String()+"/retry-analysis", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/admin/submissions/:id/retry-analysis", handler.RetryAnalysis)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "Cannot retry analysis without enrichment data")

	mockSubmissionSvc.AssertExpectations(t)
	mockEnrichmentSvc.AssertExpectations(t)
}

