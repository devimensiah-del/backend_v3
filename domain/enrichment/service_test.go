package enrichment_test

import (
	"backend_v3/config"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/submission"
	"backend_v3/llm"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// TABLE-DRIVEN TESTS FOR SERVICE METHODS
// ============================================================================

// TestGetByID_TableDriven tests GetByID with various scenarios
func TestGetByID_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		enrichID    uuid.UUID
		setupMock   func(*MockRepository)
		expectError bool
		errorMsg    string
		validate    func(*testing.T, *enrichment.Enrichment)
	}{
		{
			name:     "Success - Found",
			enrichID: uuid.New(),
			setupMock: func(repo *MockRepository) {
				e := &enrichment.Enrichment{
					ID:           uuid.New(),
					SubmissionID: uuid.New(),
					Status:       enrichment.StatusFinished,
					Progress:     100,
				}
				repo.On("GetByID", mock.Anything, mock.Anything).Return(e, nil)
			},
			expectError: false,
			validate: func(t *testing.T, e *enrichment.Enrichment) {
				assert.NotNil(t, e)
				assert.Equal(t, enrichment.StatusFinished, e.Status)
				assert.Equal(t, 100, e.Progress)
			},
		},
		{
			name:     "NotFound - Returns Error",
			enrichID: uuid.New(),
			setupMock: func(repo *MockRepository) {
				repo.On("GetByID", mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("enrichment not found"))
			},
			expectError: true,
			errorMsg:    "enrichment not found",
		},
		{
			name:     "DatabaseError - Connection Failed",
			enrichID: uuid.New(),
			setupMock: func(repo *MockRepository) {
				repo.On("GetByID", mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("database connection failed"))
			},
			expectError: true,
			errorMsg:    "database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repo := new(MockRepository)
			tt.setupMock(repo)

			svc := createTestService(repo, nil, nil)

			// Execute
			result, err := svc.GetByID(context.Background(), tt.enrichID)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}

			repo.AssertExpectations(t)
		})
	}
}

// TestGetBySubmissionID_TableDriven tests GetBySubmissionID with various scenarios
func TestGetBySubmissionID_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		submissionID uuid.UUID
		setupMock    func(*MockRepository)
		expectError  bool
		errorMsg     string
		validate     func(*testing.T, *enrichment.Enrichment)
	}{
		{
			name:         "Success - Found Single",
			submissionID: uuid.New(),
			setupMock: func(repo *MockRepository) {
				e := &enrichment.Enrichment{
					ID:           uuid.New(),
					SubmissionID: uuid.New(),
					Status:       enrichment.StatusPending,
					Progress:     0,
				}
				repo.On("GetBySubmissionID", mock.Anything, mock.Anything).Return(e, nil)
			},
			expectError: false,
			validate: func(t *testing.T, e *enrichment.Enrichment) {
				assert.NotNil(t, e)
				assert.Equal(t, enrichment.StatusPending, e.Status)
			},
		},
		{
			name:         "NotFound - No Enrichment",
			submissionID: uuid.New(),
			setupMock: func(repo *MockRepository) {
				repo.On("GetBySubmissionID", mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("enrichment not found"))
			},
			expectError: true,
			errorMsg:    "enrichment not found",
		},
		{
			name:         "Success - Multiple Enrichments Returns Latest",
			submissionID: uuid.New(),
			setupMock: func(repo *MockRepository) {
				// Repository should return the latest one (ORDER BY created_at DESC LIMIT 1)
				latestE := &enrichment.Enrichment{
					ID:           uuid.New(),
					SubmissionID: uuid.New(),
					Status:       enrichment.StatusFinished,
					Progress:     100,
					CreatedAt:    time.Now(),
				}
				repo.On("GetBySubmissionID", mock.Anything, mock.Anything).Return(latestE, nil)
			},
			expectError: false,
			validate: func(t *testing.T, e *enrichment.Enrichment) {
				assert.NotNil(t, e)
				assert.Equal(t, enrichment.StatusFinished, e.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repo := new(MockRepository)
			tt.setupMock(repo)

			svc := createTestService(repo, nil, nil)

			// Execute
			result, err := svc.GetBySubmissionID(context.Background(), tt.submissionID)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}

			repo.AssertExpectations(t)
		})
	}
}

// TestUpdateEnrichmentData_TableDriven tests UpdateEnrichmentData
func TestUpdateEnrichmentData_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		enrichID    uuid.UUID
		updateData  map[string]interface{}
		setupMock   func(*MockRepository)
		expectError bool
		errorMsg    string
		validate    func(*testing.T, *MockRepository)
	}{
		{
			name:     "Success - Valid Update",
			enrichID: uuid.New(),
			updateData: map[string]interface{}{
				"profile_overview": map[string]interface{}{
					"legal_name": "Updated Corp",
					"website":    "https://updated.com",
				},
			},
			setupMock: func(repo *MockRepository) {
				existingE := &enrichment.Enrichment{
					ID:           uuid.New(),
					SubmissionID: uuid.New(),
					Status:       enrichment.StatusFinished,
					EnrichedData: enrichment.JSONMap{},
				}
				repo.On("GetByID", mock.Anything, mock.Anything).Return(existingE, nil)
				repo.On("UpdateUser", mock.Anything, mock.MatchedBy(func(e *enrichment.Enrichment) bool {
					// Verify data was replaced
					_, ok := e.EnrichedData["profile_overview"]
					return ok
				})).Return(nil)
			},
			expectError: false,
			validate: func(t *testing.T, repo *MockRepository) {
				repo.AssertCalled(t, "UpdateUser", mock.Anything, mock.Anything)
			},
		},
		{
			name:     "Error - Enrichment Not Found",
			enrichID: uuid.New(),
			updateData: map[string]interface{}{
				"test": "data",
			},
			setupMock: func(repo *MockRepository) {
				repo.On("GetByID", mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("enrichment not found"))
			},
			expectError: true,
			errorMsg:    "enrichment not found",
		},
		{
			name:     "Error - Update Fails",
			enrichID: uuid.New(),
			updateData: map[string]interface{}{
				"test": "data",
			},
			setupMock: func(repo *MockRepository) {
				e := &enrichment.Enrichment{
					ID:           uuid.New(),
					SubmissionID: uuid.New(),
					EnrichedData: enrichment.JSONMap{},
				}
				repo.On("GetByID", mock.Anything, mock.Anything).Return(e, nil)
				repo.On("UpdateUser", mock.Anything, mock.Anything).
					Return(fmt.Errorf("database error"))
			},
			expectError: true,
			errorMsg:    "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repo := new(MockRepository)
			tt.setupMock(repo)

			svc := createTestService(repo, nil, nil)

			// Execute
			err := svc.UpdateEnrichmentData(context.Background(), tt.enrichID, tt.updateData)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, repo)
				}
			}

			repo.AssertExpectations(t)
		})
	}
}

// TestUpdateFields_TableDriven tests UpdateFields with deep merge logic
func TestUpdateFields_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		enrichID     uuid.UUID
		existingData map[string]interface{}
		updateData   map[string]interface{}
		setupMock    func(*MockRepository)
		expectError  bool
		errorMsg     string
		validateData func(*testing.T, map[string]interface{})
	}{
		{
			name:     "Success - Deep Merge Nested Objects",
			enrichID: uuid.New(),
			existingData: map[string]interface{}{
				"profile_overview": map[string]interface{}{
					"legal_name":      "Original Corp",
					"website":         "https://original.com",
					"foundation_year": "2015",
				},
				"financials": map[string]interface{}{
					"revenue_estimate": "$1M",
				},
			},
			updateData: map[string]interface{}{
				"profile_overview": map[string]interface{}{
					"legal_name": "Updated Corp", // Should update
					// website and foundation_year should remain
				},
				"market_position": map[string]interface{}{ // Should add new section
					"sector": "Technology",
				},
			},
			setupMock: func(repo *MockRepository) {
				e := &enrichment.Enrichment{
					ID:           uuid.New(),
					SubmissionID: uuid.New(),
					Status:       enrichment.StatusFinished,
					EnrichedData: enrichment.JSONMap{
						"profile_overview": map[string]interface{}{
							"legal_name":      "Original Corp",
							"website":         "https://original.com",
							"foundation_year": "2015",
						},
						"financials": map[string]interface{}{
							"revenue_estimate": "$1M",
						},
					},
				}
				repo.On("GetByID", mock.Anything, mock.Anything).Return(e, nil)
				repo.On("UpdateUser", mock.Anything, mock.Anything).Return(nil)
			},
			expectError: false,
			validateData: func(t *testing.T, data map[string]interface{}) {
				// Check deep merge worked
				profileOverview := data["profile_overview"].(map[string]interface{})
				assert.Equal(t, "Updated Corp", profileOverview["legal_name"]) // Updated
				assert.Equal(t, "https://original.com", profileOverview["website"]) // Preserved
				assert.Equal(t, "2015", profileOverview["foundation_year"]) // Preserved

				// Check financials preserved
				financials := data["financials"].(map[string]interface{})
				assert.Equal(t, "$1M", financials["revenue_estimate"])

				// Check new section added
				marketPosition := data["market_position"].(map[string]interface{})
				assert.Equal(t, "Technology", marketPosition["sector"])
			},
		},
		{
			name:     "Success - Replace Array Values",
			enrichID: uuid.New(),
			existingData: map[string]interface{}{
				"competitors": []interface{}{"CompA", "CompB"},
			},
			updateData: map[string]interface{}{
				"competitors": []interface{}{"CompX", "CompY", "CompZ"}, // Should replace entire array
			},
			setupMock: func(repo *MockRepository) {
				e := &enrichment.Enrichment{
					ID:           uuid.New(),
					SubmissionID: uuid.New(),
					EnrichedData: enrichment.JSONMap{
						"competitors": []interface{}{"CompA", "CompB"},
					},
				}
				repo.On("GetByID", mock.Anything, mock.Anything).Return(e, nil)
				repo.On("UpdateUser", mock.Anything, mock.Anything).Return(nil)
			},
			expectError: false,
			validateData: func(t *testing.T, data map[string]interface{}) {
				competitors := data["competitors"].([]interface{})
				assert.Len(t, competitors, 3)
				assert.Equal(t, "CompX", competitors[0])
			},
		},
		{
			name:     "Error - Enrichment Not Found",
			enrichID: uuid.New(),
			updateData: map[string]interface{}{
				"test": "data",
			},
			setupMock: func(repo *MockRepository) {
				repo.On("GetByID", mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("enrichment not found"))
			},
			expectError: true,
			errorMsg:    "enrichment not found",
		},
		{
			name:     "Success - Concurrent Updates (Status Unchanged)",
			enrichID: uuid.New(),
			existingData: map[string]interface{}{
				"status_field": "original",
			},
			updateData: map[string]interface{}{
				"status_field": "updated",
			},
			setupMock: func(repo *MockRepository) {
				e := &enrichment.Enrichment{
					ID:           uuid.New(),
					SubmissionID: uuid.New(),
					Status:       enrichment.StatusFinished, // Should stay finished
					EnrichedData: enrichment.JSONMap{
						"status_field": "original",
					},
				}
				repo.On("GetByID", mock.Anything, mock.Anything).Return(e, nil)
				repo.On("UpdateUser", mock.Anything, mock.MatchedBy(func(e *enrichment.Enrichment) bool {
					// Verify status remains finished
					return e.Status == enrichment.StatusFinished
				})).Return(nil)
			},
			expectError: false,
			validateData: func(t *testing.T, data map[string]interface{}) {
				assert.Equal(t, "updated", data["status_field"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repo := new(MockRepository)
			tt.setupMock(repo)

			svc := createTestService(repo, nil, nil)

			// Execute
			result, err := svc.UpdateFields(context.Background(), tt.enrichID, tt.updateData)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validateData != nil {
					tt.validateData(t, result.EnrichedData)
				}
			}

			repo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// INDIVIDUAL TESTS FOR APPROVE METHOD
// ============================================================================

// TestApprove_Success tests successful approval (job creation tested in integration tests)
func TestApprove_Success(t *testing.T) {
	t.Skip("Approve() requires Redis connection for Asynq - tested in integration tests")

	// This test validates:
	// 1. Status must be "finished" before approval
	// 2. Status changes from "finished" to "approved"
	// 3. UpdateSystem is called with approved status
	// 4. Analysis job is enqueued with correct payload

	// Integration test should verify full flow with real Redis/Asynq
}

// TestApprove_RejectsNonFinishedStatus tests rejection when status is not "finished"
func TestApprove_RejectsNonFinishedStatus(t *testing.T) {
	tests := []struct {
		name   string
		status enrichment.Status
	}{
		{"Pending Status", enrichment.StatusPending},
		{"Already Approved", enrichment.StatusApproved},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			enrichID := uuid.New()
			repo := new(MockRepository)

			existingE := &enrichment.Enrichment{
				ID:           enrichID,
				SubmissionID: uuid.New(),
				Status:       tt.status,
			}

			repo.On("GetByID", mock.Anything, enrichID).Return(existingE, nil)

			svc := createTestService(repo, nil, nil)

			// Execute
			err := svc.Approve(context.Background(), enrichID)

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "enrichment must be in 'finished' status to approve")

			// Verify UpdateSystem was NOT called
			repo.AssertNotCalled(t, "UpdateSystem", mock.Anything, mock.Anything)
		})
	}
}

// TestApprove_TriggersAnalysisJob tests that analysis job is created with correct payload
// NOTE: This test is a placeholder - actual job creation is tested in integration tests
func TestApprove_TriggersAnalysisJob(t *testing.T) {
	t.Skip("Analysis job creation requires real Asynq queue - tested in integration tests")

	// This test would verify:
	// 1. Task type is "analysis"
	// 2. Payload contains both submission_id and enrichment_id
	// 3. Job is enqueued successfully

	// Integration test should verify:
	// - Asynq.Enqueue is called with correct task
	// - Task payload: {"submission_id": "...", "enrichment_id": "..."}
	// - Analysis worker receives the job
}

// ============================================================================
// INDIVIDUAL TESTS FOR MARK AS FAILED METHOD
// ============================================================================

// TestMarkAsFailed_UpdatesErrorMessage tests that error message is set
func TestMarkAsFailed_UpdatesErrorMessage(t *testing.T) {
	// Setup
	submissionID := uuid.New()
	errorMsg := "LLM timeout after 3 retries"

	repo := new(MockRepository)

	existingE := &enrichment.Enrichment{
		ID:           uuid.New(),
		SubmissionID: submissionID,
		Status:       enrichment.StatusPending,
		ErrorMessage: "",
	}

	repo.On("GetBySubmissionID", mock.Anything, submissionID).Return(existingE, nil)
	repo.On("UpdateSystem", mock.Anything, mock.MatchedBy(func(e *enrichment.Enrichment) bool {
		return e.ErrorMessage == errorMsg && e.Status == enrichment.StatusPending
	})).Return(nil)

	svc := createTestService(repo, nil, nil)

	// Execute
	err := svc.MarkAsFailed(context.Background(), submissionID, errorMsg)

	// Assert
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// TestMarkAsFailed_KeepsStatusPending tests that status remains "pending"
func TestMarkAsFailed_KeepsStatusPending(t *testing.T) {
	// Setup
	submissionID := uuid.New()

	repo := new(MockRepository)

	existingE := &enrichment.Enrichment{
		ID:           uuid.New(),
		SubmissionID: submissionID,
		Status:       enrichment.StatusPending,
	}

	repo.On("GetBySubmissionID", mock.Anything, submissionID).Return(existingE, nil)
	repo.On("UpdateSystem", mock.Anything, mock.MatchedBy(func(e *enrichment.Enrichment) bool {
		// Verify status stays pending even after failure
		return e.Status == enrichment.StatusPending
	})).Return(nil)

	svc := createTestService(repo, nil, nil)

	// Execute
	err := svc.MarkAsFailed(context.Background(), submissionID, "some error")

	// Assert
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// ============================================================================
// HELPER FUNCTIONS AND MOCKS
// ============================================================================

// createTestService creates a service for testing
func createTestService(repo enrichment.Repository, subRepo submission.Repository, llmClient *llm.Client) *enrichment.Service {
	if subRepo == nil {
		subRepo = new(MockSubmissionRepo)
	}
	if llmClient == nil {
		llmClient = &llm.Client{} // Dummy client
	}

	cfg := config.FrameworkConfig{
		Model:       "test-model",
		Temperature: 0.5,
		MaxTokens:   4000,
	}

	// Use dummy queue client
	queueClient := asynq.NewClient(asynq.RedisClientOpt{Addr: "localhost:6379"})

	return enrichment.NewService(repo, subRepo, llmClient, queueClient, cfg)
}

// NOTE: Asynq client mocking is complex and requires interface abstraction
// Job creation tests should be done in integration tests with real Redis/Asynq
// These unit tests focus on business logic and status transitions
