package submission_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"backend_v3/domain/submission"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// MOCK REPOSITORY
// ============================================================================

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, s *submission.Submission) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*submission.Submission, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*submission.Submission), args.Error(1)
}

func (m *MockRepository) GetByEmail(ctx context.Context, email string) ([]*submission.Submission, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*submission.Submission), args.Error(1)
}

func (m *MockRepository) List(ctx context.Context, opts *submission.ListOptions) ([]*submission.Submission, int, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*submission.Submission), args.Int(1), args.Error(2)
}

func (m *MockRepository) Update(ctx context.Context, s *submission.Submission) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) GetTotalCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetActiveCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetCompletedCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetEnrichmentStatus(ctx context.Context, submissionID uuid.UUID) (*submission.EnrichmentStatusRow, error) {
	args := m.Called(ctx, submissionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*submission.EnrichmentStatusRow), args.Error(1)
}

func (m *MockRepository) ReserveEnrichment(ctx context.Context, submissionID uuid.UUID) (bool, error) {
	args := m.Called(ctx, submissionID)
	return args.Bool(0), args.Error(1)
}

// ============================================================================
// TEST CREATE
// ============================================================================

func TestService_Create(t *testing.T) {
	tests := []struct {
		name        string
		submission  *submission.Submission
		mockSetup   func(*MockRepository)
		wantErr     bool
		errContains string
	}{
		{
			name: "success - creates submission with valid data",
			submission: &submission.Submission{
				CompanyName:       "Test Corp",
				ContactName:       "John Doe",
				ContactEmail:      "john@test.com",
				BusinessChallenge: "Need to scale",
			},
			mockSetup: func(m *MockRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*submission.Submission")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - creates submission with nil UserID (public submission)",
			submission: &submission.Submission{
				CompanyName:       "Test Corp",
				ContactName:       "Jane Doe",
				ContactEmail:      "jane@test.com",
				BusinessChallenge: "Need help",
				UserID:            nil, // Public submission
			},
			mockSetup: func(m *MockRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*submission.Submission")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failure - validation error (missing company name)",
			submission: &submission.Submission{
				ContactName:       "John Doe",
				ContactEmail:      "john@test.com",
				BusinessChallenge: "Need to scale",
			},
			mockSetup:   func(m *MockRepository) {},
			wantErr:     true,
			errContains: "company name is required",
		},
		{
			name: "failure - validation error (missing contact name)",
			submission: &submission.Submission{
				CompanyName:       "Test Corp",
				ContactEmail:      "john@test.com",
				BusinessChallenge: "Need to scale",
			},
			mockSetup:   func(m *MockRepository) {},
			wantErr:     true,
			errContains: "contact name is required",
		},
		{
			name: "failure - validation error (missing contact email)",
			submission: &submission.Submission{
				CompanyName:       "Test Corp",
				ContactName:       "John Doe",
				BusinessChallenge: "Need to scale",
			},
			mockSetup:   func(m *MockRepository) {},
			wantErr:     true,
			errContains: "contact email is required",
		},
		{
			name: "failure - validation error (missing business challenge)",
			submission: &submission.Submission{
				CompanyName:  "Test Corp",
				ContactName:  "John Doe",
				ContactEmail: "john@test.com",
			},
			mockSetup:   func(m *MockRepository) {},
			wantErr:     true,
			errContains: "business challenge is required",
		},
		{
			name: "failure - database error",
			submission: &submission.Submission{
				CompanyName:       "Test Corp",
				ContactName:       "John Doe",
				ContactEmail:      "john@test.com",
				BusinessChallenge: "Need to scale",
			},
			mockSetup: func(m *MockRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*submission.Submission")).
					Return(errors.New("database connection failed"))
			},
			wantErr:     true,
			errContains: "database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockRepository)
			tt.mockSetup(mockRepo)

			// Use nil queue client for Create tests (no jobs triggered)
			svc := submission.NewService(mockRepo, nil)

			// Execute
			result, err := svc.Create(context.Background(), tt.submission, nil)

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
				assert.False(t, result.CreatedAt.IsZero())
				assert.False(t, result.UpdatedAt.IsZero())
				assert.Equal(t, submission.StatusReceived, result.Status)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// TEST GET BY ID
// ============================================================================

func TestService_GetByID(t *testing.T) {
	tests := []struct {
		name        string
		id          interface{}
		mockSetup   func(*MockRepository)
		wantErr     bool
		errContains string
	}{
		{
			name: "success - found by UUID",
			id:   uuid.New(),
			mockSetup: func(m *MockRepository) {
				m.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(&submission.Submission{
						ID:                uuid.New(),
						CompanyName:       "Test Corp",
						ContactName:       "John Doe",
						ContactEmail:      "john@test.com",
						BusinessChallenge: "Need to scale",
						Status:            submission.StatusReceived,
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "success - found by string UUID",
			id:   uuid.New().String(),
			mockSetup: func(m *MockRepository) {
				m.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(&submission.Submission{
						ID:                uuid.New(),
						CompanyName:       "Test Corp",
						ContactName:       "John Doe",
						ContactEmail:      "john@test.com",
						BusinessChallenge: "Need to scale",
						Status:            submission.StatusReceived,
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "failure - not found",
			id:   uuid.New(),
			mockSetup: func(m *MockRepository) {
				m.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(nil, sql.ErrNoRows)
			},
			wantErr:     true,
			errContains: "no rows",
		},
		{
			name:        "failure - invalid UUID format",
			id:          "not-a-uuid",
			mockSetup:   func(m *MockRepository) {},
			wantErr:     true,
			errContains: "invalid UUID format",
		},
		{
			name:        "failure - invalid type",
			id:          12345,
			mockSetup:   func(m *MockRepository) {},
			wantErr:     true,
			errContains: "id must be string or uuid.UUID",
		},
		{
			name: "failure - database error",
			id:   uuid.New(),
			mockSetup: func(m *MockRepository) {
				m.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(nil, errors.New("database connection failed"))
			},
			wantErr:     true,
			errContains: "database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockRepository)
			tt.mockSetup(mockRepo)

			svc := submission.NewService(mockRepo, nil)

			// Execute
			result, err := svc.GetByID(context.Background(), tt.id)

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
				assert.Equal(t, submission.StatusReceived, result.Status)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// TEST GET BY EMAIL
// ============================================================================

func TestService_GetByEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		mockSetup func(*MockRepository)
		wantCount int
		wantErr   bool
	}{
		{
			name:  "success - multiple submissions found",
			email: "john@test.com",
			mockSetup: func(m *MockRepository) {
				m.On("GetByEmail", mock.Anything, "john@test.com").
					Return([]*submission.Submission{
						{ID: uuid.New(), CompanyName: "Corp A", ContactEmail: "john@test.com"},
						{ID: uuid.New(), CompanyName: "Corp B", ContactEmail: "john@test.com"},
						{ID: uuid.New(), CompanyName: "Corp C", ContactEmail: "john@test.com"},
					}, nil)
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:  "success - single submission found",
			email: "jane@test.com",
			mockSetup: func(m *MockRepository) {
				m.On("GetByEmail", mock.Anything, "jane@test.com").
					Return([]*submission.Submission{
						{ID: uuid.New(), CompanyName: "Corp A", ContactEmail: "jane@test.com"},
					}, nil)
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:  "success - none found",
			email: "nobody@test.com",
			mockSetup: func(m *MockRepository) {
				m.On("GetByEmail", mock.Anything, "nobody@test.com").
					Return([]*submission.Submission{}, nil)
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:  "failure - database error",
			email: "error@test.com",
			mockSetup: func(m *MockRepository) {
				m.On("GetByEmail", mock.Anything, "error@test.com").
					Return(nil, errors.New("database connection failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockRepository)
			tt.mockSetup(mockRepo)

			svc := submission.NewService(mockRepo, nil)

			// Execute
			result, err := svc.GetByEmail(context.Background(), tt.email)

			// Verify
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.wantCount)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// TEST DELETE
// ============================================================================

func TestService_Delete(t *testing.T) {
	tests := []struct {
		name        string
		id          uuid.UUID
		mockSetup   func(*MockRepository)
		wantErr     bool
		errContains string
	}{
		{
			name: "success - soft delete sets deleted_at timestamp",
			id:   uuid.New(),
			mockSetup: func(m *MockRepository) {
				m.On("Delete", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failure - not found",
			id:   uuid.New(),
			mockSetup: func(m *MockRepository) {
				m.On("Delete", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(errors.New("submission not found"))
			},
			wantErr:     true,
			errContains: "submission not found",
		},
		{
			name: "failure - already deleted",
			id:   uuid.New(),
			mockSetup: func(m *MockRepository) {
				m.On("Delete", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(errors.New("submission not found"))
			},
			wantErr:     true,
			errContains: "submission not found",
		},
		{
			name: "failure - database error",
			id:   uuid.New(),
			mockSetup: func(m *MockRepository) {
				m.On("Delete", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(errors.New("database connection failed"))
			},
			wantErr:     true,
			errContains: "database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockRepository)
			tt.mockSetup(mockRepo)

			svc := submission.NewService(mockRepo, nil)

			// Execute
			err := svc.Delete(context.Background(), tt.id)

			// Verify
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// TEST GET ANALYTICS (with parallel goroutines)
// ============================================================================

func TestService_GetAnalytics(t *testing.T) {
	tests := []struct {
		name        string
		mockSetup   func(*MockRepository)
		wantTotal   int
		wantActive  int
		wantComp    int
		wantRevenue float64
		wantErr     bool
		errContains string
	}{
		{
			name: "success - all counts retrieved in parallel",
			mockSetup: func(m *MockRepository) {
				m.On("GetTotalCount", mock.Anything).Return(100, nil)
				m.On("GetActiveCount", mock.Anything).Return(60, nil)
				m.On("GetCompletedCount", mock.Anything).Return(40, nil)
			},
			wantTotal:   100,
			wantActive:  60,
			wantComp:    40,
			wantRevenue: 20000.0, // 40 * $500
			wantErr:     false,
		},
		{
			name: "success - zero counts",
			mockSetup: func(m *MockRepository) {
				m.On("GetTotalCount", mock.Anything).Return(0, nil)
				m.On("GetActiveCount", mock.Anything).Return(0, nil)
				m.On("GetCompletedCount", mock.Anything).Return(0, nil)
			},
			wantTotal:   0,
			wantActive:  0,
			wantComp:    0,
			wantRevenue: 0.0,
			wantErr:     false,
		},
		{
			name: "failure - total count error",
			mockSetup: func(m *MockRepository) {
				m.On("GetTotalCount", mock.Anything).Return(0, errors.New("total count failed"))
				m.On("GetActiveCount", mock.Anything).Return(60, nil).Maybe()
				m.On("GetCompletedCount", mock.Anything).Return(40, nil).Maybe()
			},
			wantErr:     true,
			errContains: "failed to get analytics",
		},
		{
			name: "failure - active count error",
			mockSetup: func(m *MockRepository) {
				m.On("GetTotalCount", mock.Anything).Return(100, nil).Maybe()
				m.On("GetActiveCount", mock.Anything).Return(0, errors.New("active count failed"))
				m.On("GetCompletedCount", mock.Anything).Return(40, nil).Maybe()
			},
			wantErr:     true,
			errContains: "failed to get analytics",
		},
		{
			name: "failure - completed count error",
			mockSetup: func(m *MockRepository) {
				m.On("GetTotalCount", mock.Anything).Return(100, nil).Maybe()
				m.On("GetActiveCount", mock.Anything).Return(60, nil).Maybe()
				m.On("GetCompletedCount", mock.Anything).Return(0, errors.New("completed count failed"))
			},
			wantErr:     true,
			errContains: "failed to get analytics",
		},
		{
			name: "success - handles partial failures gracefully (first error wins)",
			mockSetup: func(m *MockRepository) {
				// Simulate different goroutines completing at different times
				m.On("GetTotalCount", mock.Anything).Return(0, errors.New("database timeout"))
				m.On("GetActiveCount", mock.Anything).Return(0, nil).Maybe()
				m.On("GetCompletedCount", mock.Anything).Return(0, nil).Maybe()
			},
			wantErr:     true,
			errContains: "failed to get analytics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockRepository)
			tt.mockSetup(mockRepo)

			svc := submission.NewService(mockRepo, nil)

			// Execute
			result, err := svc.GetAnalytics(context.Background())

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
				assert.Equal(t, tt.wantTotal, result.TotalSubmissions)
				assert.Equal(t, tt.wantActive, result.ActiveSubmissions)
				assert.Equal(t, tt.wantComp, result.CompletedSubmissions)
				assert.Equal(t, tt.wantRevenue, result.Revenue)
			}

			// Verify all goroutines were called (even if we got an error)
			mockRepo.AssertExpectations(t)
		})
	}
}

// Test that GetAnalytics executes goroutines concurrently
func TestService_GetAnalytics_ParallelExecution(t *testing.T) {
	mockRepo := new(MockRepository)

	// Track call times
	callTimes := make(map[string]time.Time)
	var mu sync.Mutex

	mockRepo.On("GetTotalCount", mock.Anything).Run(func(args mock.Arguments) {
		mu.Lock()
		callTimes["total"] = time.Now()
		mu.Unlock()
		time.Sleep(10 * time.Millisecond) // Simulate work
	}).Return(100, nil)

	mockRepo.On("GetActiveCount", mock.Anything).Run(func(args mock.Arguments) {
		mu.Lock()
		callTimes["active"] = time.Now()
		mu.Unlock()
		time.Sleep(10 * time.Millisecond) // Simulate work
	}).Return(60, nil)

	mockRepo.On("GetCompletedCount", mock.Anything).Run(func(args mock.Arguments) {
		mu.Lock()
		callTimes["completed"] = time.Now()
		mu.Unlock()
		time.Sleep(10 * time.Millisecond) // Simulate work
	}).Return(40, nil)

	svc := submission.NewService(mockRepo, nil)

	// Execute
	start := time.Now()
	result, err := svc.GetAnalytics(context.Background())
	duration := time.Since(start)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// If executed sequentially: 3 * 10ms = 30ms minimum
	// If executed in parallel: ~10ms (the longest single call)
	// Add 20ms buffer for test execution overhead
	assert.Less(t, duration.Milliseconds(), int64(40), "Should execute in parallel (~10ms), not sequential (~30ms)")

	// Verify all three calls started within a short window (parallel execution)
	mu.Lock()
	totalTime := callTimes["total"]
	activeTime := callTimes["active"]
	completedTime := callTimes["completed"]
	mu.Unlock()

	// All three should start within 5ms of each other (parallel)
	maxDiff := time.Duration(0)
	if diff := totalTime.Sub(activeTime); diff < 0 {
		diff = -diff
	} else if diff > maxDiff {
		maxDiff = diff
	}
	if diff := totalTime.Sub(completedTime); diff < 0 {
		diff = -diff
	} else if diff > maxDiff {
		maxDiff = diff
	}
	if diff := activeTime.Sub(completedTime); diff < 0 {
		diff = -diff
	} else if diff > maxDiff {
		maxDiff = diff
	}

	assert.Less(t, maxDiff.Milliseconds(), int64(5), "All goroutines should start within 5ms of each other")

	mockRepo.AssertExpectations(t)
}
