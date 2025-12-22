package framework

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository is defined in mock.go and shared across tests

// =============================================================================
// Service Tests
// =============================================================================

func TestServiceBuildExecutionPlan(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	svc := NewService(mockRepo, logger)
	ctx := context.Background()

	// Create test frameworks in different layers
	frameworks := []*Framework{
		{ID: uuid.New(), Code: "pestel", Name: "PESTEL", Layer: 1, IsActive: true},
		{ID: uuid.New(), Code: "porter", Name: "Porter", Layer: 1, IsActive: true},
		{ID: uuid.New(), Code: "swot", Name: "SWOT", Layer: 2, IsActive: true},
		{ID: uuid.New(), Code: "synthesis", Name: "Synthesis", Layer: 6, IsActive: true},
	}

	mockRepo.On("GetActive", ctx).Return(frameworks, nil)

	plan, err := svc.BuildExecutionPlan(ctx)
	require.NoError(t, err)
	require.NotNil(t, plan)

	// Should have 3 layers (1, 2, 6)
	assert.Len(t, plan.Layers, 3)

	// Layer 1 should have 2 frameworks (pestel, porter)
	assert.Equal(t, 1, plan.Layers[0].LayerNumber)
	assert.Len(t, plan.Layers[0].Frameworks, 2)

	// Layer 2 should have 1 framework (swot)
	assert.Equal(t, 2, plan.Layers[1].LayerNumber)
	assert.Len(t, plan.Layers[1].Frameworks, 1)

	// Layer 6 should have 1 framework (synthesis)
	assert.Equal(t, 6, plan.Layers[2].LayerNumber)
	assert.Len(t, plan.Layers[2].Frameworks, 1)

	mockRepo.AssertExpectations(t)
}

func TestServiceBuildExecutionPlanEmpty(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	svc := NewService(mockRepo, logger)
	ctx := context.Background()

	mockRepo.On("GetActive", ctx).Return([]*Framework{}, nil)

	plan, err := svc.BuildExecutionPlan(ctx)
	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Len(t, plan.Layers, 0)

	mockRepo.AssertExpectations(t)
}

func TestServiceGetDependencyContext(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	svc := NewService(mockRepo, logger)
	ctx := context.Background()

	frameworkID := uuid.New()
	companyID := uuid.New()
	pestelID := uuid.New()

	// Setup dependencies
	deps := []*FrameworkDependency{
		{ID: uuid.New(), FrameworkID: frameworkID, DependsOnID: pestelID, DependencyType: DependencyRequired},
	}

	// Setup PESTEL result
	pestelResult := &CompanyFrameworkResult{
		ID:          uuid.New(),
		CompanyID:   companyID,
		FrameworkID: pestelID,
		Status:      StatusCompleted,
		Result:      json.RawMessage(`{"political": ["Factor 1"]}`),
	}

	// Setup PESTEL framework
	pestelFramework := &Framework{ID: pestelID, Code: "pestel", Name: "PESTEL"}

	mockRepo.On("GetDependencies", ctx, frameworkID).Return(deps, nil)
	mockRepo.On("GetCurrentResult", ctx, companyID, pestelID, (*uuid.UUID)(nil)).Return(pestelResult, nil)
	mockRepo.On("GetByID", ctx, pestelID).Return(pestelFramework, nil)

	depContext, err := svc.GetDependencyContext(ctx, frameworkID, companyID, nil)
	require.NoError(t, err)

	// Should have PESTEL in context
	assert.Contains(t, depContext, "pestel")
	assert.JSONEq(t, `{"political": ["Factor 1"]}`, string(depContext["pestel"]))

	mockRepo.AssertExpectations(t)
}

func TestServiceCheckDependenciesMet(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	svc := NewService(mockRepo, logger)
	ctx := context.Background()

	frameworkID := uuid.New()
	companyID := uuid.New()
	pestelID := uuid.New()
	porterID := uuid.New()

	t.Run("all dependencies met", func(t *testing.T) {
		deps := []*FrameworkDependency{
			{FrameworkID: frameworkID, DependsOnID: pestelID, DependencyType: DependencyRequired},
		}

		completedResult := &CompanyFrameworkResult{
			Status: StatusCompleted,
		}

		mockRepo.On("GetDependencies", ctx, frameworkID).Return(deps, nil).Once()
		mockRepo.On("GetCurrentResult", ctx, companyID, pestelID, (*uuid.UUID)(nil)).Return(completedResult, nil).Once()

		met, missing, err := svc.CheckDependenciesMet(ctx, frameworkID, companyID, nil)
		require.NoError(t, err)
		assert.True(t, met)
		assert.Empty(t, missing)
	})

	t.Run("dependency missing", func(t *testing.T) {
		deps := []*FrameworkDependency{
			{FrameworkID: frameworkID, DependsOnID: porterID, DependencyType: DependencyRequired},
		}

		porterFramework := &Framework{ID: porterID, Code: "porter"}

		mockRepo.On("GetDependencies", ctx, frameworkID).Return(deps, nil).Once()
		mockRepo.On("GetCurrentResult", ctx, companyID, porterID, (*uuid.UUID)(nil)).Return(nil, nil).Once()
		mockRepo.On("GetByID", ctx, porterID).Return(porterFramework, nil).Once()

		met, missing, err := svc.CheckDependenciesMet(ctx, frameworkID, companyID, nil)
		require.NoError(t, err)
		assert.False(t, met)
		assert.Contains(t, missing, "porter")
	})

	mockRepo.AssertExpectations(t)
}

func TestServiceComputeContextHash(t *testing.T) {
	logger := zerolog.Nop()
	svc := NewService(nil, logger)

	companyData := map[string]string{"name": "Test Corp"}
	depContext := map[string]json.RawMessage{
		"pestel": json.RawMessage(`{"summary": "test"}`),
	}

	hash1 := svc.ComputeContextHash(companyData, depContext)
	hash2 := svc.ComputeContextHash(companyData, depContext)

	// Same input should produce same hash
	assert.Equal(t, hash1, hash2)
	assert.Len(t, hash1, 64) // SHA256 hex = 64 chars

	// Different input should produce different hash
	companyData["name"] = "Different Corp"
	hash3 := svc.ComputeContextHash(companyData, depContext)
	assert.NotEqual(t, hash1, hash3)
}

func TestServiceCreatePendingResult(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	svc := NewService(mockRepo, logger)
	ctx := context.Background()

	companyID := uuid.New()
	frameworkID := uuid.New()

	mockRepo.On("InvalidateResults", ctx, companyID, frameworkID).Return(nil)
	mockRepo.On("CreateResult", ctx, mock.AnythingOfType("*framework.CompanyFrameworkResult")).Return(nil)

	result, err := svc.CreatePendingResult(ctx, companyID, frameworkID, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, StatusPending, result.Status)
	assert.Equal(t, companyID, result.CompanyID)
	assert.Equal(t, frameworkID, result.FrameworkID)
	assert.True(t, result.IsCurrent)

	mockRepo.AssertExpectations(t)
}

func TestServiceMarkCompleted(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	svc := NewService(mockRepo, logger)
	ctx := context.Background()

	resultID := uuid.New()
	resultData := json.RawMessage(`{"summary": "completed"}`)

	mockRepo.On("UpdateResultCompleted", ctx, resultID, []byte(resultData), mock.AnythingOfType("time.Time")).Return(nil)

	err := svc.MarkCompleted(ctx, resultID, resultData)
	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestServiceMarkFailed(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	svc := NewService(mockRepo, logger)
	ctx := context.Background()

	resultID := uuid.New()
	errorMsg := "LLM timeout"

	mockRepo.On("UpdateResultStatus", ctx, resultID, StatusFailed, &errorMsg).Return(nil)

	err := svc.MarkFailed(ctx, resultID, errorMsg)
	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestServiceUpdateResult(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	svc := NewService(mockRepo, logger)
	ctx := context.Background()

	resultID := uuid.New()
	newResult := json.RawMessage(`{"updated": true, "summary": "edited by user"}`)

	t.Run("successful update", func(t *testing.T) {
		mockRepo.On("UpdateResultData", ctx, resultID, newResult).Return(nil).Once()

		err := svc.UpdateResult(ctx, resultID, newResult)
		require.NoError(t, err)
	})

	t.Run("result not found", func(t *testing.T) {
		notFoundID := uuid.New()
		mockRepo.On("UpdateResultData", ctx, notFoundID, newResult).Return(ErrResultNotFound).Once()

		err := svc.UpdateResult(ctx, notFoundID, newResult)
		assert.ErrorIs(t, err, ErrResultNotFound)
	})

	mockRepo.AssertExpectations(t)
}
