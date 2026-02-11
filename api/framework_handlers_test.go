package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend_v3/domain/framework"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// =============================================================================
// Mock Framework Service
// =============================================================================

type MockFrameworkService struct {
	mock.Mock
}

func (m *MockFrameworkService) GetActive(ctx interface{}) ([]*framework.Framework, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*framework.Framework), args.Error(1)
}

func (m *MockFrameworkService) GetByID(ctx interface{}, id uuid.UUID) (*framework.Framework, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*framework.Framework), args.Error(1)
}

func (m *MockFrameworkService) GetByCode(ctx interface{}, code string) (*framework.Framework, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*framework.Framework), args.Error(1)
}

func (m *MockFrameworkService) Update(ctx interface{}, f *framework.Framework) error {
	args := m.Called(ctx, f)
	return args.Error(0)
}

func (m *MockFrameworkService) BuildExecutionPlan(ctx interface{}) (*framework.ExecutionPlan, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*framework.ExecutionPlan), args.Error(1)
}

func (m *MockFrameworkService) GetCompanyResultsWithFramework(ctx interface{}, companyID uuid.UUID, challengeID *uuid.UUID) ([]*framework.CompanyFrameworkResultWithDetails, error) {
	args := m.Called(ctx, companyID, challengeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*framework.CompanyFrameworkResultWithDetails), args.Error(1)
}

func (m *MockFrameworkService) GetResultByID(ctx interface{}, id uuid.UUID) (*framework.CompanyFrameworkResult, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*framework.CompanyFrameworkResult), args.Error(1)
}

func (m *MockFrameworkService) CheckDependenciesMet(ctx interface{}, frameworkID, companyID uuid.UUID, challengeID *uuid.UUID) (bool, []string, error) {
	args := m.Called(ctx, frameworkID, companyID, challengeID)
	return args.Bool(0), args.Get(1).([]string), args.Error(2)
}

func (m *MockFrameworkService) CreatePendingResult(ctx interface{}, companyID, frameworkID uuid.UUID, challengeID *uuid.UUID) (*framework.CompanyFrameworkResult, error) {
	args := m.Called(ctx, companyID, frameworkID, challengeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*framework.CompanyFrameworkResult), args.Error(1)
}

func (m *MockFrameworkService) UpdateResult(ctx interface{}, resultID uuid.UUID, result json.RawMessage) error {
	args := m.Called(ctx, resultID, result)
	return args.Error(0)
}

// =============================================================================
// Test Helpers
// =============================================================================

func newTestFrameworkHandler() *FrameworkHandlers {
	logger := zerolog.Nop()
	// Note: We can't use the mock directly because FrameworkHandlers expects *framework.Service
	// For these tests, we'll test the handler behavior with nil checks
	return &FrameworkHandlers{
		FrameworkService: nil, // Will be set per test
		AsynqClient:      nil,
		Logger:           logger,
	}
}

func setupTestRouter(handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

// =============================================================================
// ListFrameworks Tests
// =============================================================================

func TestListFrameworks_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create test frameworks
	frameworks := []*framework.Framework{
		{
			ID:       uuid.New(),
			Code:     "pestel",
			Name:     "PESTEL Analysis",
			Layer:    1,
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			Code:     "porter",
			Name:     "Porter's 5 Forces",
			Layer:    1,
			IsActive: true,
		},
	}

	// Create mock service using the actual service with a mock repo
	mockRepo := new(framework.MockRepository)
	mockRepo.On("GetActive", mock.Anything).Return(frameworks, nil)

	logger := zerolog.Nop()
	svc := framework.NewService(mockRepo, logger)

	handler := &FrameworkHandlers{
		FrameworkService: svc,
		Logger:           logger,
	}

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/frameworks", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call handler
	handler.ListFrameworks(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), response["total"])

	fwList := response["frameworks"].([]interface{})
	assert.Len(t, fwList, 2)

	mockRepo.AssertExpectations(t)
}

// =============================================================================
// GetFramework Tests
// =============================================================================

func TestGetFramework_ByCode_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fw := &framework.Framework{
		ID:       uuid.New(),
		Code:     "pestel",
		Name:     "PESTEL Analysis",
		Layer:    1,
		IsActive: true,
	}

	mockRepo := new(framework.MockRepository)
	mockRepo.On("GetByCode", mock.Anything, "pestel").Return(fw, nil)

	logger := zerolog.Nop()
	svc := framework.NewService(mockRepo, logger)

	handler := &FrameworkHandlers{
		FrameworkService: svc,
		Logger:           logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/frameworks/pestel", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "pestel"}}

	handler.GetFramework(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	fwResp := response["framework"].(map[string]interface{})
	assert.Equal(t, "pestel", fwResp["code"])
	assert.Equal(t, "PESTEL Analysis", fwResp["name"])

	mockRepo.AssertExpectations(t)
}

func TestGetFramework_ByUUID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fwID := uuid.New()
	fw := &framework.Framework{
		ID:       fwID,
		Code:     "porter",
		Name:     "Porter's 5 Forces",
		Layer:    1,
		IsActive: true,
	}

	mockRepo := new(framework.MockRepository)
	mockRepo.On("GetByID", mock.Anything, fwID).Return(fw, nil)

	logger := zerolog.Nop()
	svc := framework.NewService(mockRepo, logger)

	handler := &FrameworkHandlers{
		FrameworkService: svc,
		Logger:           logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/frameworks/"+fwID.String(), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: fwID.String()}}

	handler.GetFramework(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	fwResp := response["framework"].(map[string]interface{})
	assert.Equal(t, "porter", fwResp["code"])

	mockRepo.AssertExpectations(t)
}

func TestGetFramework_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(framework.MockRepository)
	mockRepo.On("GetByCode", mock.Anything, "nonexistent").Return(nil, nil)

	logger := zerolog.Nop()
	svc := framework.NewService(mockRepo, logger)

	handler := &FrameworkHandlers{
		FrameworkService: svc,
		Logger:           logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/frameworks/nonexistent", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.GetFramework(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	mockRepo.AssertExpectations(t)
}

// =============================================================================
// UpdateFramework Tests
// =============================================================================

func TestUpdateFramework_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fwID := uuid.New()
	fw := &framework.Framework{
		ID:          fwID,
		Code:        "pestel",
		Name:        "PESTEL Analysis",
		Layer:       1,
		IsActive:    true,
		PromptUser:  "Original prompt",
		ModelConfig: framework.ModelConfig{Model: "gpt-4"},
	}

	mockRepo := new(framework.MockRepository)
	mockRepo.On("GetByCode", mock.Anything, "pestel").Return(fw, nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*framework.Framework")).Return(nil)

	logger := zerolog.Nop()
	svc := framework.NewService(mockRepo, logger)

	handler := &FrameworkHandlers{
		FrameworkService: svc,
		Logger:           logger,
	}

	newName := "Updated PESTEL"
	updateReq := UpdateFrameworkRequest{
		Name: &newName,
	}
	body, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/frameworks/pestel", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "pestel"}}

	handler.UpdateFramework(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	fwResp := response["framework"].(map[string]interface{})
	assert.Equal(t, "Updated PESTEL", fwResp["name"])

	mockRepo.AssertExpectations(t)
}

// =============================================================================
// GetCompanyFrameworkResults Tests
// =============================================================================

func TestGetCompanyFrameworkResults_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	companyID := uuid.New()
	fwID := uuid.New()

	results := []*framework.CompanyFrameworkResultWithDetails{
		{
			Result: &framework.CompanyFrameworkResult{
				ID:          uuid.New(),
				CompanyID:   companyID,
				FrameworkID: fwID,
				Status:      framework.StatusCompleted,
				IsCurrent:   true,
			},
			Framework: &framework.Framework{
				ID:   fwID,
				Code: "pestel",
				Name: "PESTEL",
			},
		},
	}

	mockRepo := new(framework.MockRepository)
	mockRepo.On("GetCompanyResults", mock.Anything, companyID, (*uuid.UUID)(nil)).Return([]*framework.CompanyFrameworkResult{results[0].Result}, nil)
	mockRepo.On("GetByID", mock.Anything, fwID).Return(results[0].Framework, nil)

	logger := zerolog.Nop()
	svc := framework.NewService(mockRepo, logger)

	handler := &FrameworkHandlers{
		FrameworkService: svc,
		Logger:           logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/"+companyID.String()+"/framework-results", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: companyID.String()}}

	handler.GetCompanyFrameworkResults(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), response["total"])

	mockRepo.AssertExpectations(t)
}

func TestGetCompanyFrameworkResults_InvalidCompanyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := zerolog.Nop()
	handler := &FrameworkHandlers{
		FrameworkService: nil,
		Logger:           logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/invalid-uuid/framework-results", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetCompanyFrameworkResults(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// =============================================================================
// GetFrameworkResultStatus Tests
// =============================================================================

func TestGetFrameworkResultStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resultID := uuid.New()
	fwID := uuid.New()
	companyID := uuid.New()

	result := &framework.CompanyFrameworkResult{
		ID:          resultID,
		CompanyID:   companyID,
		FrameworkID: fwID,
		Status:      framework.StatusCompleted,
		IsCurrent:   true,
		GeneratedAt: func() *time.Time { t := time.Now(); return &t }(),
	}

	fw := &framework.Framework{
		ID:   fwID,
		Code: "pestel",
		Name: "PESTEL",
	}

	mockRepo := new(framework.MockRepository)
	mockRepo.On("GetResultByID", mock.Anything, resultID).Return(result, nil)
	mockRepo.On("GetByID", mock.Anything, fwID).Return(fw, nil)

	logger := zerolog.Nop()
	svc := framework.NewService(mockRepo, logger)

	handler := &FrameworkHandlers{
		FrameworkService: svc,
		Logger:           logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/"+companyID.String()+"/framework-results/"+resultID.String(), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{
		{Key: "id", Value: companyID.String()},
		{Key: "resultId", Value: resultID.String()},
	}

	handler.GetFrameworkResultStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	resultResp := response["result"].(map[string]interface{})
	assert.Equal(t, framework.StatusCompleted, resultResp["status"])

	mockRepo.AssertExpectations(t)
}

func TestGetFrameworkResultStatus_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resultID := uuid.New()
	companyID := uuid.New()

	mockRepo := new(framework.MockRepository)
	mockRepo.On("GetResultByID", mock.Anything, resultID).Return(nil, nil)

	logger := zerolog.Nop()
	svc := framework.NewService(mockRepo, logger)

	handler := &FrameworkHandlers{
		FrameworkService: svc,
		Logger:           logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/"+companyID.String()+"/framework-results/"+resultID.String(), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{
		{Key: "id", Value: companyID.String()},
		{Key: "resultId", Value: resultID.String()},
	}

	handler.GetFrameworkResultStatus(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	mockRepo.AssertExpectations(t)
}

// =============================================================================
// GetExecutionPlan Tests
// =============================================================================

func TestGetExecutionPlan_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	plan := &framework.ExecutionPlan{
		Layers: []framework.ExecutionLayer{
			{
				LayerNumber: 1,
				Frameworks: []*framework.Framework{
					{ID: uuid.New(), Code: "pestel", Name: "PESTEL", Layer: 1},
					{ID: uuid.New(), Code: "porter", Name: "Porter", Layer: 1},
				},
			},
			{
				LayerNumber: 2,
				Frameworks: []*framework.Framework{
					{ID: uuid.New(), Code: "swot", Name: "SWOT", Layer: 2},
				},
			},
		},
	}

	frameworks := []*framework.Framework{
		plan.Layers[0].Frameworks[0],
		plan.Layers[0].Frameworks[1],
		plan.Layers[1].Frameworks[0],
	}

	mockRepo := new(framework.MockRepository)
	mockRepo.On("GetActive", mock.Anything).Return(frameworks, nil)

	logger := zerolog.Nop()
	svc := framework.NewService(mockRepo, logger)

	handler := &FrameworkHandlers{
		FrameworkService: svc,
		Logger:           logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/frameworks/execution-plan", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.GetExecutionPlan(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	execPlan := response["execution_plan"].(map[string]interface{})
	layers := execPlan["layers"].([]interface{})
	assert.Len(t, layers, 2)

	mockRepo.AssertExpectations(t)
}

// =============================================================================
// UpdateFrameworkResult Tests
// =============================================================================

func TestUpdateFrameworkResult_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resultID := uuid.New()
	companyID := uuid.New()
	fwID := uuid.New()

	existingResult := &framework.CompanyFrameworkResult{
		ID:          resultID,
		CompanyID:   companyID,
		FrameworkID: fwID,
		Status:      framework.StatusCompleted,
		Result:      json.RawMessage(`{"summary": "original"}`),
		IsCurrent:   true,
	}

	mockRepo := new(framework.MockRepository)
	mockRepo.On("GetResultByID", mock.Anything, resultID).Return(existingResult, nil)
	mockRepo.On("UpdateResultData", mock.Anything, resultID, mock.AnythingOfType("json.RawMessage")).Return(nil)

	logger := zerolog.Nop()
	svc := framework.NewService(mockRepo, logger)

	handler := &FrameworkHandlers{
		FrameworkService: svc,
		Logger:           logger,
	}

	updateReq := map[string]interface{}{
		"result": map[string]interface{}{
			"summary": "updated by user",
			"edited":  true,
		},
	}
	body, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/companies/"+companyID.String()+"/framework-results/"+resultID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{
		{Key: "id", Value: companyID.String()},
		{Key: "resultId", Value: resultID.String()},
	}

	handler.UpdateFrameworkResult(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Result updated successfully", response["message"])

	mockRepo.AssertExpectations(t)
}

func TestUpdateFrameworkResult_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resultID := uuid.New()
	companyID := uuid.New()

	mockRepo := new(framework.MockRepository)
	mockRepo.On("GetResultByID", mock.Anything, resultID).Return(nil, nil)

	logger := zerolog.Nop()
	svc := framework.NewService(mockRepo, logger)

	handler := &FrameworkHandlers{
		FrameworkService: svc,
		Logger:           logger,
	}

	updateReq := map[string]interface{}{
		"result": map[string]interface{}{"summary": "test"},
	}
	body, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/companies/"+companyID.String()+"/framework-results/"+resultID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{
		{Key: "id", Value: companyID.String()},
		{Key: "resultId", Value: resultID.String()},
	}

	handler.UpdateFrameworkResult(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	mockRepo.AssertExpectations(t)
}

func TestUpdateFrameworkResult_InvalidResultID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	companyID := uuid.New()

	logger := zerolog.Nop()
	handler := &FrameworkHandlers{
		FrameworkService: nil,
		Logger:           logger,
	}

	updateReq := map[string]interface{}{
		"result": map[string]interface{}{"summary": "test"},
	}
	body, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/companies/"+companyID.String()+"/framework-results/invalid-uuid", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{
		{Key: "id", Value: companyID.String()},
		{Key: "resultId", Value: "invalid-uuid"},
	}

	handler.UpdateFrameworkResult(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateFrameworkResult_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resultID := uuid.New()
	companyID := uuid.New()

	logger := zerolog.Nop()
	handler := &FrameworkHandlers{
		FrameworkService: nil,
		Logger:           logger,
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/companies/"+companyID.String()+"/framework-results/"+resultID.String(), bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{
		{Key: "id", Value: companyID.String()},
		{Key: "resultId", Value: resultID.String()},
	}

	handler.UpdateFrameworkResult(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateFrameworkResult_CompanyMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resultID := uuid.New()
	companyID := uuid.New()
	differentCompanyID := uuid.New()
	fwID := uuid.New()

	existingResult := &framework.CompanyFrameworkResult{
		ID:          resultID,
		CompanyID:   differentCompanyID, // Different company
		FrameworkID: fwID,
		Status:      framework.StatusCompleted,
		IsCurrent:   true,
	}

	mockRepo := new(framework.MockRepository)
	mockRepo.On("GetResultByID", mock.Anything, resultID).Return(existingResult, nil)

	logger := zerolog.Nop()
	svc := framework.NewService(mockRepo, logger)

	handler := &FrameworkHandlers{
		FrameworkService: svc,
		Logger:           logger,
	}

	updateReq := map[string]interface{}{
		"result": map[string]interface{}{"summary": "test"},
	}
	body, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/companies/"+companyID.String()+"/framework-results/"+resultID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{
		{Key: "id", Value: companyID.String()},
		{Key: "resultId", Value: resultID.String()},
	}

	handler.UpdateFrameworkResult(c)

	assert.Equal(t, http.StatusForbidden, w.Code)

	mockRepo.AssertExpectations(t)
}
