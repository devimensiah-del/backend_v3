package submission_test

import (
	"context"
	"errors"
	"testing"

	"backend_v3/domain/submission"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// =============================================================================
// MOCKS
// =============================================================================

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, s *submission.Submission) error {
	return m.Called(ctx, s).Error(0)
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
	return m.Called(ctx, s).Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockRepository) GetByCompanyID(ctx context.Context, companyID uuid.UUID) (*submission.Submission, error) {
	args := m.Called(ctx, companyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*submission.Submission), args.Error(1)
}

func (m *MockRepository) GetAnonymousByEmail(ctx context.Context, email string) ([]*submission.Submission, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*submission.Submission), args.Error(1)
}

func (m *MockRepository) UpdateUserID(ctx context.Context, submissionID, userID uuid.UUID) error {
	return m.Called(ctx, submissionID, userID).Error(0)
}

func (m *MockRepository) BeginTx(ctx context.Context) (submission.Repository, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(submission.Repository), args.Error(1)
}

func (m *MockRepository) Commit() error {
	return m.Called().Error(0)
}

func (m *MockRepository) Rollback() error {
	return m.Called().Error(0)
}

type MockCompanyService struct {
	mock.Mock
}

func (m *MockCompanyService) CreateFromSubmission(ctx context.Context, input submission.CompanyCreateInput) (submission.CompanyResult, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(submission.CompanyResult), args.Error(1)
}

func (m *MockCompanyService) SetOwnerFromSubmission(ctx context.Context, submissionID, userID uuid.UUID) error {
	return m.Called(ctx, submissionID, userID).Error(0)
}

func (m *MockCompanyService) DeleteCompany(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

type MockChallengeService struct {
	mock.Mock
}

func (m *MockChallengeService) CreateFromInput(ctx context.Context, input submission.ChallengeCreateInput) (uuid.UUID, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockChallengeService) DeleteChallenge(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockChallengeService) ValidateCategory(category string) bool {
	args := m.Called(category)
	return args.Bool(0)
}

func (m *MockChallengeService) ValidateType(category, challengeType string) bool {
	args := m.Called(category, challengeType)
	return args.Bool(0)
}

// =============================================================================
// HELPERS
// =============================================================================

func ptr[T any](v T) *T { return &v }

func validSubmitRequest() *submission.SubmitRequest {
	return &submission.SubmitRequest{
		CompanyName:       "Test Corp",
		ContactName:       "John Doe",
		ContactEmail:      "john@test.com",
		ChallengeCategory: "growth",
		ChallengeType:     "growth_organic",
		BusinessChallenge: "Need to scale user acquisition",
	}
}

// =============================================================================
// TEST: Create (CRUD)
// =============================================================================

func TestService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := new(MockRepository)
		repo.On("Create", mock.Anything, mock.AnythingOfType("*submission.Submission")).Return(nil)

		svc := submission.NewService(repo)
		result, err := svc.Create(context.Background(), &submission.Submission{
			CompanyName:  "Test Corp",
			ContactName:  "John Doe",
			ContactEmail: "john@test.com",
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.ID)
		repo.AssertExpectations(t)
	})

	t.Run("validation error - missing company name", func(t *testing.T) {
		repo := new(MockRepository)
		svc := submission.NewService(repo)

		_, err := svc.Create(context.Background(), &submission.Submission{
			ContactName:  "John",
			ContactEmail: "john@test.com",
		})

		assert.Error(t, err)
		assert.True(t, submission.IsValidation(err))
	})

	t.Run("validation error - invalid email", func(t *testing.T) {
		repo := new(MockRepository)
		svc := submission.NewService(repo)

		_, err := svc.Create(context.Background(), &submission.Submission{
			CompanyName:  "Test",
			ContactName:  "John",
			ContactEmail: "not-an-email",
		})

		assert.Error(t, err)
		assert.True(t, submission.IsValidation(err))
	})

	t.Run("validation error - revenue min > max", func(t *testing.T) {
		repo := new(MockRepository)
		svc := submission.NewService(repo)

		_, err := svc.Create(context.Background(), &submission.Submission{
			CompanyName:      "Test",
			ContactName:      "John",
			ContactEmail:     "john@test.com",
			AnnualRevenueMin: ptr(1000000.0),
			AnnualRevenueMax: ptr(500000.0),
		})

		assert.Error(t, err)
		assert.True(t, submission.IsValidation(err))
	})

	t.Run("db error", func(t *testing.T) {
		repo := new(MockRepository)
		repo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error"))

		svc := submission.NewService(repo)
		_, err := svc.Create(context.Background(), &submission.Submission{
			CompanyName:  "Test",
			ContactName:  "John",
			ContactEmail: "john@test.com",
		})

		assert.Error(t, err)
	})
}

// =============================================================================
// TEST: GetByID
// =============================================================================

func TestService_GetByID(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		repo := new(MockRepository)
		expected := &submission.Submission{ID: uuid.New(), CompanyName: "Test"}
		repo.On("GetByID", mock.Anything, expected.ID).Return(expected, nil)

		svc := submission.NewService(repo)
		result, err := svc.GetByID(context.Background(), expected.ID)

		assert.NoError(t, err)
		assert.Equal(t, expected.ID, result.ID)
	})

	t.Run("not found", func(t *testing.T) {
		repo := new(MockRepository)
		repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, submission.ErrNotFound)

		svc := submission.NewService(repo)
		_, err := svc.GetByID(context.Background(), uuid.New())

		assert.Error(t, err)
		assert.True(t, submission.IsNotFound(err))
	})
}

// =============================================================================
// TEST: Delete
// =============================================================================

func TestService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := new(MockRepository)
		repo.On("Delete", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil)

		svc := submission.NewService(repo)
		err := svc.Delete(context.Background(), uuid.New())

		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		repo := new(MockRepository)
		repo.On("Delete", mock.Anything, mock.Anything).Return(submission.ErrNotFound)

		svc := submission.NewService(repo)
		err := svc.Delete(context.Background(), uuid.New())

		assert.True(t, submission.IsNotFound(err))
	})
}

// =============================================================================
// TEST: SubmitForm (Main Workflow)
// =============================================================================

func TestService_SubmitForm(t *testing.T) {
	companyID := uuid.New()
	challengeID := uuid.New()

	t.Run("success - full workflow", func(t *testing.T) {
		repo := new(MockRepository)
		companySvc := new(MockCompanyService)
		challengeSvc := new(MockChallengeService)

		// Repository calls (no transaction in current implementation)
		repo.On("Create", mock.Anything, mock.AnythingOfType("*submission.Submission")).Return(nil)
		repo.On("Delete", mock.Anything, mock.Anything).Return(nil) // For potential rollback

		// Validation calls
		challengeSvc.On("ValidateCategory", "growth").Return(true)
		challengeSvc.On("ValidateType", "growth", "growth_organic").Return(true)

		// Service calls
		companySvc.On("CreateFromSubmission", mock.Anything, mock.Anything).
			Return(submission.CompanyResult{ID: companyID, Name: "Test Corp"}, nil)
		companySvc.On("DeleteCompany", mock.Anything, mock.Anything).Return(nil) // For potential rollback
		challengeSvc.On("CreateFromInput", mock.Anything, mock.Anything).Return(challengeID, nil)
		challengeSvc.On("DeleteChallenge", mock.Anything, mock.Anything).Return(nil) // For potential rollback

		svc := submission.NewService(repo)
		svc.SetCompanyService(companySvc)
		svc.SetChallengeService(challengeSvc)

		result, err := svc.SubmitForm(context.Background(), validSubmitRequest())

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.SubmissionID)
		assert.Equal(t, companyID, result.CompanyID)
		assert.Equal(t, challengeID, result.ChallengeID)
	})

	t.Run("validation error - missing challenge category", func(t *testing.T) {
		repo := new(MockRepository)
		svc := submission.NewService(repo)
		svc.SetCompanyService(new(MockCompanyService))
		svc.SetChallengeService(new(MockChallengeService))

		req := validSubmitRequest()
		req.ChallengeCategory = ""

		_, err := svc.SubmitForm(context.Background(), req)

		assert.Error(t, err)
		assert.True(t, submission.IsValidation(err))
	})

	t.Run("validation error - invalid challenge type for category", func(t *testing.T) {
		repo := new(MockRepository)
		challengeSvc := new(MockChallengeService)

		// Validation: category valid, but type invalid for that category
		challengeSvc.On("ValidateCategory", "growth").Return(true)
		challengeSvc.On("ValidateType", "growth", "transform_digital").Return(false)

		svc := submission.NewService(repo)
		svc.SetCompanyService(new(MockCompanyService))
		svc.SetChallengeService(challengeSvc)

		req := validSubmitRequest()
		req.ChallengeCategory = "growth"
		req.ChallengeType = "transform_digital" // Wrong category

		_, err := svc.SubmitForm(context.Background(), req)

		assert.Error(t, err)
		assert.True(t, submission.IsValidation(err))
	})

	t.Run("validation error - revenue min > max", func(t *testing.T) {
		repo := new(MockRepository)
		challengeSvc := new(MockChallengeService)

		// Validation passes for challenge fields
		challengeSvc.On("ValidateCategory", "growth").Return(true)
		challengeSvc.On("ValidateType", "growth", "growth_organic").Return(true)

		svc := submission.NewService(repo)
		svc.SetCompanyService(new(MockCompanyService))
		svc.SetChallengeService(challengeSvc)

		req := validSubmitRequest()
		req.AnnualRevenueMin = ptr(1000000.0)
		req.AnnualRevenueMax = ptr(500000.0)

		_, err := svc.SubmitForm(context.Background(), req)

		assert.Error(t, err)
		assert.True(t, submission.IsValidation(err))
	})

	t.Run("error - company service not configured", func(t *testing.T) {
		repo := new(MockRepository)
		svc := submission.NewService(repo)
		// No company service set

		_, err := svc.SubmitForm(context.Background(), validSubmitRequest())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "setup failed")
	})

	t.Run("error - company creation fails", func(t *testing.T) {
		repo := new(MockRepository)
		companySvc := new(MockCompanyService)
		challengeSvc := new(MockChallengeService)

		repo.On("Create", mock.Anything, mock.Anything).Return(nil)
		repo.On("Delete", mock.Anything, mock.Anything).Return(nil)

		// Validation passes
		challengeSvc.On("ValidateCategory", "growth").Return(true)
		challengeSvc.On("ValidateType", "growth", "growth_organic").Return(true)

		companySvc.On("CreateFromSubmission", mock.Anything, mock.Anything).
			Return(submission.CompanyResult{}, errors.New("enrichment failed"))

		svc := submission.NewService(repo)
		svc.SetCompanyService(companySvc)
		svc.SetChallengeService(challengeSvc)

		_, err := svc.SubmitForm(context.Background(), validSubmitRequest())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "company")
	})

	t.Run("error - challenge creation fails", func(t *testing.T) {
		repo := new(MockRepository)
		companySvc := new(MockCompanyService)
		challengeSvc := new(MockChallengeService)

		repo.On("Create", mock.Anything, mock.Anything).Return(nil)
		repo.On("Delete", mock.Anything, mock.Anything).Return(nil)

		// Validation passes
		challengeSvc.On("ValidateCategory", "growth").Return(true)
		challengeSvc.On("ValidateType", "growth", "growth_organic").Return(true)

		companySvc.On("CreateFromSubmission", mock.Anything, mock.Anything).
			Return(submission.CompanyResult{ID: companyID, Name: "Test"}, nil)
		companySvc.On("DeleteCompany", mock.Anything, companyID).Return(nil)
		challengeSvc.On("CreateFromInput", mock.Anything, mock.Anything).
			Return(uuid.Nil, errors.New("challenge failed"))

		svc := submission.NewService(repo)
		svc.SetCompanyService(companySvc)
		svc.SetChallengeService(challengeSvc)

		_, err := svc.SubmitForm(context.Background(), validSubmitRequest())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "challenge")
	})
}

// =============================================================================
// TEST: Challenge Validation (Category/Type combinations)
// =============================================================================

func TestService_SubmitForm_ChallengeValidation(t *testing.T) {
	validCombinations := []struct {
		category string
		typ      string
	}{
		{"growth", "growth_organic"},
		{"growth", "growth_geographic"},
		{"transform", "transform_digital"},
		{"transform", "transform_model"},
		{"transition", "transition_succession"},
		{"compete", "compete_differentiate"},
		{"funding", "funding_raise"},
	}

	for _, tc := range validCombinations {
		t.Run(tc.category+"/"+tc.typ, func(t *testing.T) {
			repo := new(MockRepository)
			companySvc := new(MockCompanyService)
			challengeSvc := new(MockChallengeService)

			repo.On("Create", mock.Anything, mock.Anything).Return(nil)
			repo.On("Delete", mock.Anything, mock.Anything).Return(nil)

			// Validation passes for this combination
			challengeSvc.On("ValidateCategory", tc.category).Return(true)
			challengeSvc.On("ValidateType", tc.category, tc.typ).Return(true)

			companySvc.On("CreateFromSubmission", mock.Anything, mock.Anything).
				Return(submission.CompanyResult{ID: uuid.New()}, nil)
			companySvc.On("DeleteCompany", mock.Anything, mock.Anything).Return(nil)
			challengeSvc.On("CreateFromInput", mock.Anything, mock.Anything).
				Return(uuid.New(), nil)
			challengeSvc.On("DeleteChallenge", mock.Anything, mock.Anything).Return(nil)

			svc := submission.NewService(repo)
			svc.SetCompanyService(companySvc)
			svc.SetChallengeService(challengeSvc)

			req := validSubmitRequest()
			req.ChallengeCategory = tc.category
			req.ChallengeType = tc.typ

			_, err := svc.SubmitForm(context.Background(), req)
			assert.NoError(t, err)
		})
	}

	invalidCombinations := []struct {
		category string
		typ      string
	}{
		{"growth", "transform_digital"},
		{"transform", "funding_raise"},
		{"invalid", "growth_organic"},
		{"growth", "invalid_type"},
	}

	for _, tc := range invalidCombinations {
		t.Run("invalid_"+tc.category+"/"+tc.typ, func(t *testing.T) {
			repo := new(MockRepository)
			challengeSvc := new(MockChallengeService)

			// Mock validation to return false for invalid combinations
			// Category "invalid" returns false, valid categories return true
			if tc.category == "invalid" {
				challengeSvc.On("ValidateCategory", tc.category).Return(false)
			} else {
				challengeSvc.On("ValidateCategory", tc.category).Return(true)
				// Type returns false for invalid type or wrong category combo
				challengeSvc.On("ValidateType", tc.category, tc.typ).Return(false)
			}

			svc := submission.NewService(repo)
			svc.SetCompanyService(new(MockCompanyService))
			svc.SetChallengeService(challengeSvc)

			req := validSubmitRequest()
			req.ChallengeCategory = tc.category
			req.ChallengeType = tc.typ

			_, err := svc.SubmitForm(context.Background(), req)
			assert.Error(t, err)
			assert.True(t, submission.IsValidation(err))
		})
	}
}

// =============================================================================
// TEST: LinkAnonymousToUser
// =============================================================================

func TestService_LinkAnonymousToUser(t *testing.T) {
	t.Run("links submissions successfully", func(t *testing.T) {
		repo := new(MockRepository)
		companySvc := new(MockCompanyService)

		userID := uuid.New()
		sub1 := &submission.Submission{ID: uuid.New()}
		sub2 := &submission.Submission{ID: uuid.New()}

		repo.On("GetAnonymousByEmail", mock.Anything, "john@test.com").
			Return([]*submission.Submission{sub1, sub2}, nil)
		repo.On("UpdateUserID", mock.Anything, sub1.ID, userID).Return(nil)
		repo.On("UpdateUserID", mock.Anything, sub2.ID, userID).Return(nil)

		companySvc.On("SetOwnerFromSubmission", mock.Anything, mock.Anything, userID).Return(nil)

		svc := submission.NewService(repo)
		svc.SetCompanyService(companySvc)

		err := svc.LinkAnonymousToUser(context.Background(), userID, "john@test.com")

		assert.NoError(t, err)
		repo.AssertNumberOfCalls(t, "UpdateUserID", 2)
	})

	t.Run("no submissions to link", func(t *testing.T) {
		repo := new(MockRepository)
		repo.On("GetAnonymousByEmail", mock.Anything, "new@test.com").
			Return([]*submission.Submission{}, nil)

		svc := submission.NewService(repo)
		err := svc.LinkAnonymousToUser(context.Background(), uuid.New(), "new@test.com")

		assert.NoError(t, err)
		repo.AssertNotCalled(t, "UpdateUserID")
	})
}
