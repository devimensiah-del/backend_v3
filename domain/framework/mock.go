package framework

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of Repository for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, f *Framework) error {
	args := m.Called(ctx, f)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*Framework, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Framework), args.Error(1)
}

func (m *MockRepository) GetByCode(ctx context.Context, code string) (*Framework, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Framework), args.Error(1)
}

func (m *MockRepository) GetActive(ctx context.Context) ([]*Framework, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Framework), args.Error(1)
}

func (m *MockRepository) GetBaseFrameworks(ctx context.Context) ([]*Framework, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Framework), args.Error(1)
}

func (m *MockRepository) GetByLayer(ctx context.Context, layer int) ([]*Framework, error) {
	args := m.Called(ctx, layer)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Framework), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, f *Framework) error {
	args := m.Called(ctx, f)
	return args.Error(0)
}

func (m *MockRepository) GetDependencies(ctx context.Context, frameworkID uuid.UUID) ([]*FrameworkDependency, error) {
	args := m.Called(ctx, frameworkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*FrameworkDependency), args.Error(1)
}

func (m *MockRepository) GetAllDependencies(ctx context.Context) ([]*FrameworkDependency, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*FrameworkDependency), args.Error(1)
}

func (m *MockRepository) AddDependency(ctx context.Context, dep *FrameworkDependency) error {
	args := m.Called(ctx, dep)
	return args.Error(0)
}

func (m *MockRepository) RemoveDependency(ctx context.Context, frameworkID, dependsOnID uuid.UUID) error {
	args := m.Called(ctx, frameworkID, dependsOnID)
	return args.Error(0)
}

func (m *MockRepository) CreateResult(ctx context.Context, result *CompanyFrameworkResult) error {
	args := m.Called(ctx, result)
	return args.Error(0)
}

func (m *MockRepository) GetResultByID(ctx context.Context, id uuid.UUID) (*CompanyFrameworkResult, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CompanyFrameworkResult), args.Error(1)
}

func (m *MockRepository) GetCurrentResult(ctx context.Context, companyID, frameworkID uuid.UUID, challengeID *uuid.UUID) (*CompanyFrameworkResult, error) {
	args := m.Called(ctx, companyID, frameworkID, challengeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CompanyFrameworkResult), args.Error(1)
}

func (m *MockRepository) GetCompanyResults(ctx context.Context, companyID uuid.UUID, challengeID *uuid.UUID) ([]*CompanyFrameworkResult, error) {
	args := m.Called(ctx, companyID, challengeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CompanyFrameworkResult), args.Error(1)
}

func (m *MockRepository) GetCompanyResultsByStatus(ctx context.Context, companyID uuid.UUID, status string) ([]*CompanyFrameworkResult, error) {
	args := m.Called(ctx, companyID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CompanyFrameworkResult), args.Error(1)
}

func (m *MockRepository) UpdateResultStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error {
	args := m.Called(ctx, id, status, errorMsg)
	return args.Error(0)
}

func (m *MockRepository) UpdateResultCompleted(ctx context.Context, id uuid.UUID, result []byte, generatedAt time.Time) error {
	args := m.Called(ctx, id, result, generatedAt)
	return args.Error(0)
}

func (m *MockRepository) InvalidateResults(ctx context.Context, companyID uuid.UUID, frameworkID uuid.UUID) error {
	args := m.Called(ctx, companyID, frameworkID)
	return args.Error(0)
}

func (m *MockRepository) UpdateResultData(ctx context.Context, id uuid.UUID, result json.RawMessage) error {
	args := m.Called(ctx, id, result)
	return args.Error(0)
}

func (m *MockRepository) WithTx(ctx context.Context, fn func(Repository) error) error {
	return fn(m)
}
