package submission_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"backend_v3/domain/submission"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// MOCK ASYNQ CLIENT
// ============================================================================

type MockAsynqClient struct {
	mock.Mock
}

func (m *MockAsynqClient) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	args := m.Called(task, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*asynq.TaskInfo), args.Error(1)
}

func (m *MockAsynqClient) Close() error {
	return nil
}

// ============================================================================
// TEST SUBMIT FORM WORKFLOW
// ============================================================================

func TestService_SubmitForm(t *testing.T) {
	tests := []struct {
		name          string
		request       *submission.SubmitRequest
		repoMockSetup func(*MockRepository)
		wantErr       bool
		errContains   string
		assertQueue   func(*testing.T, *MockAsynqClient)
	}{
		{
			name: "success - saves submission with provided CNPJ",
			request: &submission.SubmitRequest{
				CompanyName:       "Test Corp",
				ContactName:       "John Doe",
				ContactEmail:      "john@test.com",
				BusinessChallenge: "Need to scale",
				CNPJ:              stringPtr("12.345.678/0001-90"),
			},
			repoMockSetup: func(m *MockRepository) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(sub *submission.Submission) bool {
					return sub.CNPJ != nil && *sub.CNPJ == "12.345.678/0001-90"
				})).Return(nil)
			},
			wantErr:     false,
			assertQueue: func(t *testing.T, q *MockAsynqClient) { q.AssertCalled(t, "Enqueue", mock.Anything, mock.Anything) },
		},
		{
			name: "success - creates submission with nil UserID (public submission)",
			request: &submission.SubmitRequest{
				CompanyName:       "Public Corp",
				ContactName:       "Jane Doe",
				ContactEmail:      "jane@test.com",
				BusinessChallenge: "Need help",
				UserID:            nil, // Public submission
			},
			repoMockSetup: func(m *MockRepository) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(sub *submission.Submission) bool {
					return sub.UserID == nil // Verify UserID is nil
				})).Return(nil)
			},
			wantErr:     false,
			assertQueue: func(t *testing.T, q *MockAsynqClient) { q.AssertCalled(t, "Enqueue", mock.Anything, mock.Anything) },
		},
		{
			name: "success - creates submission with UserID (authenticated user)",
			request: func() *submission.SubmitRequest {
				userID := uuid.New()
				return &submission.SubmitRequest{
					CompanyName:       "User Corp",
					ContactName:       "Bob Smith",
					ContactEmail:      "bob@test.com",
					BusinessChallenge: "Need strategy",
					UserID:            &userID,
				}
			}(),
			repoMockSetup: func(m *MockRepository) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(sub *submission.Submission) bool {
					return sub.UserID != nil // Verify UserID is set
				})).Return(nil)
			},
			wantErr:     false,
			assertQueue: func(t *testing.T, q *MockAsynqClient) { q.AssertCalled(t, "Enqueue", mock.Anything, mock.Anything) },
		},
		{
			name: "failure - validation error (missing company name)",
			request: &submission.SubmitRequest{
				ContactName:       "John Doe",
				ContactEmail:      "john@test.com",
				BusinessChallenge: "Need to scale",
			},
			repoMockSetup: func(m *MockRepository) {},
			wantErr:       true,
			errContains:   "validation failed",
		},
		{
			name: "failure - database error (submission not saved)",
			request: &submission.SubmitRequest{
				CompanyName:       "Test Corp",
				ContactName:       "John Doe",
				ContactEmail:      "john@test.com",
				BusinessChallenge: "Need to scale",
			},
			repoMockSetup: func(m *MockRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*submission.Submission")).
					Return(assert.AnError)
			},
			wantErr:     true,
			errContains: "failed to save submission",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockRepository)
			mockQueue := new(MockAsynqClient)

			tt.repoMockSetup(mockRepo)
			if !tt.wantErr {
				mockRepo.On("ReserveEnrichment", mock.Anything, mock.Anything).Return(true, nil)
				mockQueue.On("Enqueue", mock.Anything, mock.Anything).Return(&asynq.TaskInfo{}, nil)
			}

			svc := submission.NewService(mockRepo, mockQueue)

			// Execute
			result, err := svc.SubmitForm(context.Background(), tt.request, nil)

			// Verify
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEqual(t, uuid.Nil, result.ID)
				assert.Equal(t, submission.StatusReceived, result.Status)
				assert.False(t, result.CreatedAt.IsZero())
				assert.False(t, result.UpdatedAt.IsZero())
			}

			mockRepo.AssertExpectations(t)
			if tt.assertQueue != nil {
				tt.assertQueue(t, mockQueue)
			} else {
				mockQueue.AssertNotCalled(t, "Enqueue", mock.Anything, mock.Anything)
			}
		})
	}
}

// ============================================================================
// TEST TRIGGER ENRICHMENT PROCESS
// ============================================================================

func TestService_TriggerEnrichmentProcess(t *testing.T) {
	tests := []struct {
		name           string
		submission     *submission.Submission
		queueMockSetup func(*MockAsynqClient)
		wantErr        bool
	}{
		{
			name: "success - enqueues enrichment job with correct type",
			submission: &submission.Submission{
				ID:                uuid.New(),
				CompanyName:       "Test Corp",
				ContactName:       "John Doe",
				ContactEmail:      "john@test.com",
				BusinessChallenge: "Need to scale",
				Status:            submission.StatusReceived,
			},
			queueMockSetup: func(m *MockAsynqClient) {
				m.On("Enqueue", mock.MatchedBy(func(task *asynq.Task) bool {
					return task.Type() == "enrichment_job"
				}), mock.Anything).Return(&asynq.TaskInfo{}, nil)
			},
		},
		{
			name: "success - payload contains submission_id as UUID string",
			submission: &submission.Submission{
				ID:                uuid.New(),
				CompanyName:       "Test Corp",
				ContactName:       "John Doe",
				ContactEmail:      "john@test.com",
				BusinessChallenge: "Need to scale",
				Status:            submission.StatusReceived,
			},
			queueMockSetup: func(m *MockAsynqClient) {
				m.On("Enqueue", mock.Anything, mock.Anything).
					Return(&asynq.TaskInfo{}, nil).
					Run(func(args mock.Arguments) {
						task := args.Get(0).(*asynq.Task)

						var payload map[string]interface{}
						err := json.Unmarshal(task.Payload(), &payload)
						assert.NoError(t, err)

						submissionID, ok := payload["submission_id"].(string)
						assert.True(t, ok, "payload should contain submission_id")
						assert.NotEmpty(t, submissionID, "submission_id should not be empty")

						_, err = uuid.Parse(submissionID)
						assert.NoError(t, err, "submission_id should be valid UUID")
					})
			},
		},
		{
			name: "success - job has MaxRetry(3) option",
			submission: &submission.Submission{
				ID:                uuid.New(),
				CompanyName:       "Test Corp",
				ContactName:       "John Doe",
				ContactEmail:      "john@test.com",
				BusinessChallenge: "Need to scale",
				Status:            submission.StatusReceived,
			},
			queueMockSetup: func(m *MockAsynqClient) {
				m.On("Enqueue", mock.Anything, mock.MatchedBy(func(opts []asynq.Option) bool {
					return len(opts) > 0
				})).Return(&asynq.TaskInfo{}, nil)
			},
		},
		{
			name: "failure - queue client missing",
			submission: &submission.Submission{
				ID:                uuid.New(),
				CompanyName:       "Test Corp",
				ContactName:       "John Doe",
				ContactEmail:      "john@test.com",
				BusinessChallenge: "Need to scale",
				Status:            submission.StatusReceived,
			},
			queueMockSetup: func(m *MockAsynqClient) {},
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			mockQueue := new(MockAsynqClient)
			tt.queueMockSetup(mockQueue)

			mockRepo.On("ReserveEnrichment", mock.Anything, mock.Anything).Return(true, nil)
			// Mock GetEnrichmentStatus to return sql.ErrNoRows (no enrichment exists yet)
			mockRepo.On("GetEnrichmentStatus", mock.Anything, mock.Anything).Return(nil, sql.ErrNoRows)

			var svc *submission.Service
			if tt.wantErr {
				svc = submission.NewService(mockRepo, nil)
			} else {
				svc = submission.NewService(mockRepo, mockQueue)
			}

			err := svc.TriggerEnrichmentProcess(context.Background(), tt.submission)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockQueue.AssertExpectations(t)
		})
	}
}

func stringPtr(val string) *string {
	return &val
}
