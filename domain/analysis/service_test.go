package analysis

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend_v3/llm"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// =============================================================================
// MOCKS
// =============================================================================

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, analysis *Analysis) error {
	args := m.Called(ctx, analysis)
	return args.Error(0)
}

func (m *MockRepository) Update(ctx context.Context, analysis *Analysis) error {
	args := m.Called(ctx, analysis)
	return args.Error(0)
}

func (m *MockRepository) UpdateWithTx(ctx context.Context, tx *sqlx.Tx, analysis *Analysis) error {
	args := m.Called(ctx, tx, analysis)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id string) (*Analysis, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Analysis), args.Error(1)
}

func (m *MockRepository) GetBySubmissionID(ctx context.Context, submissionID string) (*Analysis, error) {
	args := m.Called(ctx, submissionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Analysis), args.Error(1)
}

func (m *MockRepository) List(ctx context.Context, limit, offset int) ([]*Analysis, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Analysis), args.Error(1)
}

func (m *MockRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqlx.Tx), args.Error(1)
}

func (m *MockRepository) SetVisibility(ctx context.Context, id string, visible bool) error {
	args := m.Called(ctx, id, visible)
	return args.Error(0)
}

func (m *MockRepository) SetBlurStatus(ctx context.Context, id string, blurred bool) error {
	args := m.Called(ctx, id, blurred)
	return args.Error(0)
}

func (m *MockRepository) GetByAccessCode(ctx context.Context, code string) (*Analysis, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Analysis), args.Error(1)
}

func (m *MockRepository) SetAccessCode(ctx context.Context, id string, code string) error {
	args := m.Called(ctx, id, code)
	return args.Error(0)
}

func (m *MockRepository) AccessCodeExists(ctx context.Context, code string) (bool, error) {
	args := m.Called(ctx, code)
	return args.Bool(0), args.Error(1)
}

type MockSubmissionRepository struct {
	mock.Mock
}

func (m *MockSubmissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*SubmissionData, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SubmissionData), args.Error(1)
}

type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) GenerateStructured(ctx context.Context, model string, prompt string, data interface{}, targetSchema interface{}) error {
	args := m.Called(ctx, model, prompt, data, targetSchema)
	return args.Error(0)
}

func (m *MockLLMClient) GenerateStructuredWithOptions(ctx context.Context, opts llm.GenerationOptions, prompt string, data interface{}, targetSchema interface{}) error {
	args := m.Called(ctx, opts, prompt, data, targetSchema)
	return args.Error(0)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func createTestService() (*Service, *MockRepository, *MockSubmissionRepository, *MockLLMClient) {
	mockRepo := new(MockRepository)
	mockSubRepo := new(MockSubmissionRepository)
	mockLLM := new(MockLLMClient)

	logger := zerolog.New(zerolog.NewTestWriter(nil)).Level(zerolog.Disabled)

	// Create mock asynq client (won't actually connect to Redis)
	queueClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr: "localhost:6379",
	})

	service := NewService(
		mockRepo,
		mockSubRepo,
		mockLLM,
		logger,
		queueClient,
	)

	return service, mockRepo, mockSubRepo, mockLLM
}

func createTestAnalysis() *Analysis {
	now := time.Now()
	analysisID := uuid.New().String()
	submissionID := uuid.New().String()
	enrichmentID := uuid.New().String()

	return &Analysis{
		ID:           analysisID,
		SubmissionID: submissionID,
		EnrichmentID: enrichmentID,
		Status:       string(StatusCompleted),
		CreatedAt:    now,
		UpdatedAt:    now,
		CompletedAt:  &now,
		PESTEL: PESTELAnalysis{
			Political:     []string{"Regulatory stability"},
			Economic:      []string{"GDP growth 2%"},
			Social:        []string{"Urbanization trend"},
			Technological: []string{"AI adoption"},
			Environmental: []string{"Sustainability pressure"},
			Legal:         []string{"LGPD compliance"},
			Summary:       "PESTEL summary",
		},
		Porter: PorterAnalysis{
			CompetitiveRivalry:    "High",
			SupplierPower:         "Medium",
			BuyerPower:            "High",
			ThreatNewEntrants:     "Low",
			ThreatSubstitutes:     "Medium",
			OverallAttractiveness: "Medium",
			Summary:               "Porter summary",
		},
		SWOT: SWOTAnalysis{
			Strengths:     []SWOTItem{{Content: "Strong brand", Confidence: "Alta", Source: "fato"}},
			Weaknesses:    []SWOTItem{{Content: "Low tech", Confidence: "Média", Source: "análise de mercado"}},
			Opportunities: []SWOTItem{{Content: "Market expansion", Confidence: "Alta", Source: "fato"}},
			Threats:       []SWOTItem{{Content: "New competitors", Confidence: "Média", Source: "estimativa"}},
			Summary:       "SWOT summary",
		},
	}
}

// =============================================================================
// TESTS: GetByID
// =============================================================================

func TestService_GetByID_Success(t *testing.T) {
	service, mockRepo, _, _ := createTestService()
	defer service.queueClient.Close()

	testAnalysis := createTestAnalysis()

	mockRepo.On("GetByID", mock.Anything, testAnalysis.ID).Return(testAnalysis, nil)

	result, err := service.GetByID(context.Background(), testAnalysis.ID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testAnalysis.ID, result.ID)
	mockRepo.AssertExpectations(t)
}

func TestService_GetByID_NotFound(t *testing.T) {
	service, mockRepo, _, _ := createTestService()
	defer service.queueClient.Close()

	analysisID := uuid.New().String()

	mockRepo.On("GetByID", mock.Anything, analysisID).Return(nil, errors.New("analysis not found"))

	result, err := service.GetByID(context.Background(), analysisID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
	mockRepo.AssertExpectations(t)
}

func TestService_GetByID_DatabaseError(t *testing.T) {
	service, mockRepo, _, _ := createTestService()
	defer service.queueClient.Close()

	analysisID := uuid.New().String()

	mockRepo.On("GetByID", mock.Anything, analysisID).Return(nil, errors.New("database connection error"))

	result, err := service.GetByID(context.Background(), analysisID)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// TESTS: GetBySubmissionID
// =============================================================================

func TestService_GetBySubmissionID_ReturnsAnalysis(t *testing.T) {
	service, mockRepo, _, _ := createTestService()
	defer service.queueClient.Close()

	testAnalysis := createTestAnalysis()

	mockRepo.On("GetBySubmissionID", mock.Anything, testAnalysis.SubmissionID).Return(testAnalysis, nil)

	result, err := service.GetBySubmissionID(context.Background(), testAnalysis.SubmissionID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testAnalysis.SubmissionID, result.SubmissionID)
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// TESTS: Delete (Soft Delete)
// =============================================================================

func TestService_Delete_Success(t *testing.T) {
	service, mockRepo, _, _ := createTestService()
	defer service.queueClient.Close()

	analysisID := uuid.New().String()

	mockRepo.On("Delete", mock.Anything, analysisID).Return(nil)

	err := service.Delete(context.Background(), analysisID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_Delete_NotFound(t *testing.T) {
	service, mockRepo, _, _ := createTestService()
	defer service.queueClient.Close()

	analysisID := uuid.New().String()

	mockRepo.On("Delete", mock.Anything, analysisID).Return(errors.New("analysis not found or already deleted"))

	err := service.Delete(context.Background(), analysisID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// TESTS: UpdateFields (Framework Merge Logic)
// =============================================================================

func TestService_UpdateFields_Success(t *testing.T) {
	service, mockRepo, _, _ := createTestService()
	defer service.queueClient.Close()

	testAnalysis := createTestAnalysis()

	updateData := map[string]interface{}{
		"pestel": map[string]interface{}{
			"summary": "Updated PESTEL summary",
		},
	}

	mockRepo.On("GetByID", mock.Anything, testAnalysis.ID).Return(testAnalysis, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *Analysis) bool {
		return a.PESTEL.Summary == "Updated PESTEL summary"
	})).Return(nil)

	result, err := service.UpdateFields(context.Background(), testAnalysis.ID, updateData)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Updated PESTEL summary", result.PESTEL.Summary)
	mockRepo.AssertExpectations(t)
}

func TestService_UpdateFields_ValidationPreservesStatus(t *testing.T) {
	service, mockRepo, _, _ := createTestService()
	defer service.queueClient.Close()

	testAnalysis := createTestAnalysis()
	originalStatus := testAnalysis.Status

	updateData := map[string]interface{}{
		"swot": map[string]interface{}{
			"summary": "Updated SWOT",
		},
	}

	mockRepo.On("GetByID", mock.Anything, testAnalysis.ID).Return(testAnalysis, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *Analysis) bool {
		return a.Status == originalStatus // Status should not change
	})).Return(nil)

	result, err := service.UpdateFields(context.Background(), testAnalysis.ID, updateData)

	assert.NoError(t, err)
	assert.Equal(t, originalStatus, result.Status)
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// TESTS: MarkAsFailed
// =============================================================================

func TestService_MarkAsFailed_KeepsPendingStatus(t *testing.T) {
	service, mockRepo, _, _ := createTestService()
	defer service.queueClient.Close()

	testAnalysis := createTestAnalysis()
	testAnalysis.Status = string(StatusPending)
	submissionID := testAnalysis.SubmissionID

	errorMsg := "LLM timeout error"

	mockRepo.On("GetBySubmissionID", mock.Anything, submissionID).Return(testAnalysis, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *Analysis) bool {
		return a.Status == string(StatusPending) && a.ErrorMessage != nil && *a.ErrorMessage == errorMsg
	})).Return(nil)

	err := service.MarkAsFailed(context.Background(), submissionID, errorMsg)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_MarkAsFailed_SetsErrorMessage(t *testing.T) {
	service, mockRepo, _, _ := createTestService()
	defer service.queueClient.Close()

	testAnalysis := createTestAnalysis()
	testAnalysis.Status = string(StatusPending)
	testAnalysis.ErrorMessage = nil

	errorMsg := "Database connection lost during analysis"

	mockRepo.On("GetBySubmissionID", mock.Anything, testAnalysis.SubmissionID).Return(testAnalysis, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *Analysis) bool {
		return a.ErrorMessage != nil && *a.ErrorMessage == errorMsg
	})).Return(nil)

	err := service.MarkAsFailed(context.Background(), testAnalysis.SubmissionID, errorMsg)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// TESTS: Table-Driven Tests for Multiple Scenarios
// =============================================================================

func TestService_GetByID_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		analysisID  string
		mockReturn  *Analysis
		mockError   error
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Success",
			analysisID:  uuid.New().String(),
			mockReturn:  createTestAnalysis(),
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "Not Found",
			analysisID:  uuid.New().String(),
			mockReturn:  nil,
			mockError:   errors.New("analysis not found"),
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name:        "Database Error",
			analysisID:  uuid.New().String(),
			mockReturn:  nil,
			mockError:   errors.New("connection timeout"),
			expectError: true,
			errorMsg:    "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mockRepo, _, _ := createTestService()
			defer service.queueClient.Close()

			mockRepo.On("GetByID", mock.Anything, tt.analysisID).Return(tt.mockReturn, tt.mockError)

			result, err := service.GetByID(context.Background(), tt.analysisID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

