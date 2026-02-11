package framework

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestDependencyService_GetDependencyCodes(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	service := NewDependencyService(mockRepo, logger)

	ctx := context.Background()

	// Setup test data
	swotID := uuid.New()
	mercadoID := uuid.New()
	pestelID := uuid.New()

	swotFramework := &Framework{
		ID:    swotID,
		Code:  "SWOT",
		Name:  "Análise SWOT",
		Layer: 2,
	}

	mercadoFramework := &Framework{
		ID:    mercadoID,
		Code:  "MERCADO",
		Name:  "Mercado",
		Layer: 1,
	}

	pestelFramework := &Framework{
		ID:    pestelID,
		Code:  "PESTEL",
		Name:  "Análise PESTEL",
		Layer: 1,
	}

	dependencies := []*FrameworkDependency{
		{ID: uuid.New(), FrameworkID: swotID, DependsOnID: mercadoID, DependencyType: DependencyRequired},
		{ID: uuid.New(), FrameworkID: swotID, DependsOnID: pestelID, DependencyType: DependencyRequired},
	}

	// Setup expectations
	mockRepo.On("GetByCode", ctx, "SWOT").Return(swotFramework, nil)
	mockRepo.On("GetDependencies", ctx, swotID).Return(dependencies, nil)
	mockRepo.On("GetByID", ctx, mercadoID).Return(mercadoFramework, nil)
	mockRepo.On("GetByID", ctx, pestelID).Return(pestelFramework, nil)

	// Execute
	codes, err := service.GetDependencyCodes(ctx, "SWOT")

	// Assert
	assert.NoError(t, err)
	assert.Len(t, codes, 2)
	assert.Contains(t, codes, "MERCADO")
	assert.Contains(t, codes, "PESTEL")

	mockRepo.AssertExpectations(t)
}

func TestDependencyService_GetDependentCodes(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	service := NewDependencyService(mockRepo, logger)

	ctx := context.Background()

	// Setup test data - MERCADO is depended on by SWOT and PESTEL
	mercadoID := uuid.New()
	swotID := uuid.New()
	porterID := uuid.New()

	mercadoFramework := &Framework{
		ID:    mercadoID,
		Code:  "MERCADO",
		Name:  "Mercado",
		Layer: 1,
	}

	swotFramework := &Framework{
		ID:    swotID,
		Code:  "SWOT",
		Name:  "Análise SWOT",
		Layer: 2,
	}

	porterFramework := &Framework{
		ID:    porterID,
		Code:  "PORTER",
		Name:  "Forças de Porter",
		Layer: 1,
	}

	dependents := []*FrameworkDependency{
		{ID: uuid.New(), FrameworkID: swotID, DependsOnID: mercadoID, DependencyType: DependencyRequired},
		{ID: uuid.New(), FrameworkID: porterID, DependsOnID: mercadoID, DependencyType: DependencyRequired},
	}

	// Setup expectations
	mockRepo.On("GetByCode", ctx, "MERCADO").Return(mercadoFramework, nil)
	mockRepo.On("GetDependents", ctx, mercadoID).Return(dependents, nil)
	mockRepo.On("GetByID", ctx, swotID).Return(swotFramework, nil)
	mockRepo.On("GetByID", ctx, porterID).Return(porterFramework, nil)

	// Execute
	codes, err := service.GetDependentCodes(ctx, "MERCADO")

	// Assert
	assert.NoError(t, err)
	assert.Len(t, codes, 2)
	assert.Contains(t, codes, "SWOT")
	assert.Contains(t, codes, "PORTER")

	mockRepo.AssertExpectations(t)
}

func TestDependencyService_CheckReadyToRun_Layer0AlwaysReady(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	service := NewDependencyService(mockRepo, logger)

	ctx := context.Background()
	companyID := uuid.New()

	// Layer 0 frameworks are always ready (they display enriched data)
	empresaFramework := &Framework{
		ID:    uuid.New(),
		Code:  "EMPRESA",
		Name:  "Dados da Empresa",
		Layer: 0,
	}

	mockRepo.On("GetByCode", ctx, "EMPRESA").Return(empresaFramework, nil)

	// Execute
	status, err := service.CheckReadyToRun(ctx, companyID, "EMPRESA", nil)

	// Assert
	assert.NoError(t, err)
	assert.True(t, status.Ready)
	assert.Empty(t, status.MissingDependencies)

	mockRepo.AssertExpectations(t)
}

func TestDependencyService_CheckReadyToRun_WithMissingDependencies(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	service := NewDependencyService(mockRepo, logger)

	ctx := context.Background()
	companyID := uuid.New()

	// Setup test data
	swotID := uuid.New()
	mercadoID := uuid.New()
	pestelID := uuid.New()

	swotFramework := &Framework{
		ID:    swotID,
		Code:  "SWOT",
		Name:  "Análise SWOT",
		Layer: 2,
	}

	// MERCADO is Layer 0 (data-only) - should be skipped in dependency check
	mercadoFramework := &Framework{
		ID:    mercadoID,
		Code:  "MERCADO",
		Name:  "Mercado",
		Layer: 0, // Layer 0 = data-only, always ready
	}

	pestelFramework := &Framework{
		ID:    pestelID,
		Code:  "PESTEL",
		Name:  "Análise PESTEL",
		Layer: 1,
	}

	dependencies := []*FrameworkDependency{
		{ID: uuid.New(), FrameworkID: swotID, DependsOnID: mercadoID, DependencyType: DependencyRequired},
		{ID: uuid.New(), FrameworkID: swotID, DependsOnID: pestelID, DependencyType: DependencyRequired},
	}

	// PESTEL not completed (MERCADO is Layer 0 so no result check needed)
	pestelResult := &CompanyFrameworkResult{
		ID:          uuid.New(),
		CompanyID:   companyID,
		FrameworkID: pestelID,
		Status:      StatusPending, // Not completed
	}

	// Setup expectations
	mockRepo.On("GetByCode", ctx, "SWOT").Return(swotFramework, nil)
	mockRepo.On("GetDependencies", ctx, swotID).Return(dependencies, nil)
	// GetByID is called first for each dependency to check its layer
	mockRepo.On("GetByID", ctx, mercadoID).Return(mercadoFramework, nil)
	mockRepo.On("GetByID", ctx, pestelID).Return(pestelFramework, nil)
	// Only PESTEL needs result check (MERCADO is Layer 0, skipped)
	mockRepo.On("GetCurrentResult", ctx, companyID, pestelID, (*uuid.UUID)(nil)).Return(pestelResult, nil)

	// Execute
	status, err := service.CheckReadyToRun(ctx, companyID, "SWOT", nil)

	// Assert
	assert.NoError(t, err)
	assert.False(t, status.Ready)
	assert.Len(t, status.MissingDependencies, 1)
	assert.Contains(t, status.MissingDependencies, "PESTEL")

	mockRepo.AssertExpectations(t)
}

func TestDependencyService_CheckReadyToRun_AllDependenciesCompleted(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	service := NewDependencyService(mockRepo, logger)

	ctx := context.Background()
	companyID := uuid.New()

	// Setup test data
	swotID := uuid.New()
	porterID := uuid.New()

	swotFramework := &Framework{
		ID:    swotID,
		Code:  "SWOT",
		Name:  "Análise SWOT",
		Layer: 2,
	}

	// PORTER is Layer 1 (AI framework) and will be checked for completion
	porterFramework := &Framework{
		ID:    porterID,
		Code:  "PORTER",
		Name:  "Porter 5 Forces",
		Layer: 1,
	}

	dependencies := []*FrameworkDependency{
		{ID: uuid.New(), FrameworkID: swotID, DependsOnID: porterID, DependencyType: DependencyRequired},
	}

	porterResult := &CompanyFrameworkResult{
		ID:          uuid.New(),
		CompanyID:   companyID,
		FrameworkID: porterID,
		Status:      StatusCompleted,
		Result:      json.RawMessage(`{"test": true}`),
	}

	// Setup expectations
	mockRepo.On("GetByCode", ctx, "SWOT").Return(swotFramework, nil)
	mockRepo.On("GetDependencies", ctx, swotID).Return(dependencies, nil)
	// GetByID is called first to check layer
	mockRepo.On("GetByID", ctx, porterID).Return(porterFramework, nil)
	// PORTER is Layer 1, so result check is needed
	mockRepo.On("GetCurrentResult", ctx, companyID, porterID, (*uuid.UUID)(nil)).Return(porterResult, nil)

	// Execute
	status, err := service.CheckReadyToRun(ctx, companyID, "SWOT", nil)

	// Assert
	assert.NoError(t, err)
	assert.True(t, status.Ready)
	assert.Empty(t, status.MissingDependencies)

	mockRepo.AssertExpectations(t)
}

func TestDependencyService_CheckReadyToRun_Layer0DependencySkipped(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	service := NewDependencyService(mockRepo, logger)

	ctx := context.Background()
	companyID := uuid.New()

	// Setup test data: PESTEL depends on MERCADO (Layer 0)
	pestelID := uuid.New()
	mercadoID := uuid.New()

	pestelFramework := &Framework{
		ID:    pestelID,
		Code:  "PESTEL",
		Name:  "Análise PESTEL",
		Layer: 1,
	}

	// MERCADO is Layer 0 (data-only) - should be skipped, no result check needed
	mercadoFramework := &Framework{
		ID:    mercadoID,
		Code:  "MERCADO",
		Name:  "Mercado",
		Layer: 0, // Data-only, always ready
	}

	dependencies := []*FrameworkDependency{
		{ID: uuid.New(), FrameworkID: pestelID, DependsOnID: mercadoID, DependencyType: DependencyRequired},
	}

	// Setup expectations - NO GetCurrentResult call for MERCADO since it's Layer 0
	mockRepo.On("GetByCode", ctx, "PESTEL").Return(pestelFramework, nil)
	mockRepo.On("GetDependencies", ctx, pestelID).Return(dependencies, nil)
	mockRepo.On("GetByID", ctx, mercadoID).Return(mercadoFramework, nil)
	// Note: GetCurrentResult is NOT called for MERCADO because it's Layer 0

	// Execute
	status, err := service.CheckReadyToRun(ctx, companyID, "PESTEL", nil)

	// Assert - PESTEL should be ready because MERCADO (Layer 0) is auto-complete
	assert.NoError(t, err)
	assert.True(t, status.Ready)
	assert.Empty(t, status.MissingDependencies)

	mockRepo.AssertExpectations(t)
}

func TestDependencyService_MarkDownstreamAsStale(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	service := NewDependencyService(mockRepo, logger)

	ctx := context.Background()
	companyID := uuid.New()

	// Setup: MERCADO -> SWOT -> SWOT_CROSS chain
	mercadoID := uuid.New()
	swotID := uuid.New()
	swotCrossID := uuid.New()

	mercadoFramework := &Framework{
		ID:    mercadoID,
		Code:  "MERCADO",
		Name:  "Mercado",
		Layer: 1,
	}

	swotFramework := &Framework{
		ID:    swotID,
		Code:  "SWOT",
		Name:  "Análise SWOT",
		Layer: 2,
	}

	swotCrossFramework := &Framework{
		ID:    swotCrossID,
		Code:  "SWOT_CROSS",
		Name:  "SWOT Cruzado",
		Layer: 2,
	}

	// SWOT depends on MERCADO
	mercadoDependents := []*FrameworkDependency{
		{ID: uuid.New(), FrameworkID: swotID, DependsOnID: mercadoID, DependencyType: DependencyRequired},
	}

	// SWOT_CROSS depends on SWOT
	swotDependents := []*FrameworkDependency{
		{ID: uuid.New(), FrameworkID: swotCrossID, DependsOnID: swotID, DependencyType: DependencyRequired},
	}

	// Results
	swotResult := &CompanyFrameworkResult{
		ID:          uuid.New(),
		CompanyID:   companyID,
		FrameworkID: swotID,
		Status:      StatusCompleted,
		Result:      json.RawMessage(`{"strengths": ["test"]}`),
		IsCurrent:   true,
	}

	swotCrossResult := &CompanyFrameworkResult{
		ID:          uuid.New(),
		CompanyID:   companyID,
		FrameworkID: swotCrossID,
		Status:      StatusCompleted,
		Result:      json.RawMessage(`{"so": ["strategy"]}`),
		IsCurrent:   true,
	}

	// Setup expectations
	mockRepo.On("GetByCode", ctx, "MERCADO").Return(mercadoFramework, nil)
	mockRepo.On("GetDependents", ctx, mercadoID).Return(mercadoDependents, nil)
	mockRepo.On("GetCurrentResult", ctx, companyID, swotID, (*uuid.UUID)(nil)).Return(swotResult, nil)
	mockRepo.On("MarkAsStale", ctx, swotResult.ID, "MERCADO foi atualizado").Return(nil)
	mockRepo.On("GetByID", ctx, swotID).Return(swotFramework, nil)
	mockRepo.On("GetDependents", ctx, swotID).Return(swotDependents, nil)
	mockRepo.On("GetCurrentResult", ctx, companyID, swotCrossID, (*uuid.UUID)(nil)).Return(swotCrossResult, nil)
	mockRepo.On("MarkAsStale", ctx, swotCrossResult.ID, "SWOT foi atualizado").Return(nil)
	mockRepo.On("GetByID", ctx, swotCrossID).Return(swotCrossFramework, nil)
	mockRepo.On("GetDependents", ctx, swotCrossID).Return([]*FrameworkDependency{}, nil)

	// Execute
	count, err := service.MarkDownstreamAsStale(ctx, companyID, "MERCADO")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 2, count) // SWOT and SWOT_CROSS should be marked stale

	mockRepo.AssertExpectations(t)
}

func TestDependencyService_GetStaleFrameworks(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	service := NewDependencyService(mockRepo, logger)

	ctx := context.Background()
	companyID := uuid.New()

	// Setup test data
	swotID := uuid.New()
	detectedAt := time.Now()

	swotFramework := &Framework{
		ID:    swotID,
		Code:  "SWOT",
		Name:  "Análise SWOT",
		Layer: 2,
	}

	staleReason := "MERCADO foi atualizado"
	staleResults := []*CompanyFrameworkResult{
		{
			ID:              uuid.New(),
			CompanyID:       companyID,
			FrameworkID:     swotID,
			Status:          StatusCompleted,
			IsStale:         true,
			StaleReason:     &staleReason,
			StaleDetectedAt: &detectedAt,
		},
	}

	// Setup expectations
	mockRepo.On("GetStaleResults", ctx, companyID).Return(staleResults, nil)
	mockRepo.On("GetByID", ctx, swotID).Return(swotFramework, nil)

	// Execute
	stale, err := service.GetStaleFrameworks(ctx, companyID)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, stale, 1)
	assert.Equal(t, "SWOT", stale[0].FrameworkCode)
	assert.Equal(t, "Análise SWOT", stale[0].FrameworkName)
	assert.Equal(t, "MERCADO foi atualizado", stale[0].StaleReason)

	mockRepo.AssertExpectations(t)
}

func TestDependencyService_AcknowledgeStale(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	service := NewDependencyService(mockRepo, logger)

	ctx := context.Background()
	resultID := uuid.New()
	userID := uuid.New()

	mockRepo.On("AcknowledgeStale", ctx, resultID, userID).Return(nil)

	// Execute
	err := service.AcknowledgeStale(ctx, resultID, userID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestDependencyService_ClearStaleStatus(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	service := NewDependencyService(mockRepo, logger)

	ctx := context.Background()
	resultID := uuid.New()

	mockRepo.On("ClearStaleStatus", ctx, resultID).Return(nil)

	// Execute
	err := service.ClearStaleStatus(ctx, resultID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestDependencyService_ComputeUpstreamHash(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	service := NewDependencyService(mockRepo, logger)

	ctx := context.Background()
	companyID := uuid.New()

	// Setup: SWOT depends on MERCADO
	swotID := uuid.New()
	mercadoID := uuid.New()

	swotFramework := &Framework{
		ID:    swotID,
		Code:  "SWOT",
		Name:  "Análise SWOT",
		Layer: 2,
	}

	mercadoFramework := &Framework{
		ID:    mercadoID,
		Code:  "MERCADO",
		Name:  "Mercado",
		Layer: 1,
	}

	dependencies := []*FrameworkDependency{
		{ID: uuid.New(), FrameworkID: swotID, DependsOnID: mercadoID, DependencyType: DependencyRequired},
	}

	mercadoResult := &CompanyFrameworkResult{
		ID:          uuid.New(),
		CompanyID:   companyID,
		FrameworkID: mercadoID,
		Status:      StatusCompleted,
		Result:      json.RawMessage(`{"market_data": "test"}`),
	}

	// Setup expectations for GetAllTransitiveDependencies
	mockRepo.On("GetByCode", ctx, "SWOT").Return(swotFramework, nil).Once()
	mockRepo.On("GetDependencies", ctx, swotID).Return(dependencies, nil)
	mockRepo.On("GetByID", ctx, mercadoID).Return(mercadoFramework, nil).Once()
	mockRepo.On("GetByCode", ctx, "MERCADO").Return(mercadoFramework, nil).Once()
	mockRepo.On("GetDependencies", ctx, mercadoID).Return([]*FrameworkDependency{}, nil)

	// Setup expectations for ComputeUpstreamHash
	mockRepo.On("GetByCode", ctx, "MERCADO").Return(mercadoFramework, nil).Once()
	mockRepo.On("GetCurrentResult", ctx, companyID, mercadoID, (*uuid.UUID)(nil)).Return(mercadoResult, nil)

	// Execute
	hash, err := service.ComputeUpstreamHash(ctx, companyID, "SWOT", nil)

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 32) // 16 bytes = 32 hex chars

	mockRepo.AssertExpectations(t)
}

func TestDependencyService_GetDependencyContext(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	service := NewDependencyService(mockRepo, logger)

	ctx := context.Background()
	companyID := uuid.New()

	// Setup: SWOT depends on MERCADO and PESTEL
	swotID := uuid.New()
	mercadoID := uuid.New()
	pestelID := uuid.New()

	swotFramework := &Framework{
		ID:    swotID,
		Code:  "SWOT",
		Name:  "Análise SWOT",
		Layer: 2,
	}

	mercadoFramework := &Framework{
		ID:    mercadoID,
		Code:  "MERCADO",
		Name:  "Mercado",
		Layer: 1,
	}

	pestelFramework := &Framework{
		ID:    pestelID,
		Code:  "PESTEL",
		Name:  "Análise PESTEL",
		Layer: 1,
	}

	swotDependencies := []*FrameworkDependency{
		{ID: uuid.New(), FrameworkID: swotID, DependsOnID: mercadoID, DependencyType: DependencyRequired},
		{ID: uuid.New(), FrameworkID: swotID, DependsOnID: pestelID, DependencyType: DependencyRequired},
	}

	mercadoResult := &CompanyFrameworkResult{
		ID:          uuid.New(),
		CompanyID:   companyID,
		FrameworkID: mercadoID,
		Status:      StatusCompleted,
		Result:      json.RawMessage(`{"market": "data"}`),
	}

	pestelResult := &CompanyFrameworkResult{
		ID:          uuid.New(),
		CompanyID:   companyID,
		FrameworkID: pestelID,
		Status:      StatusCompleted,
		Result:      json.RawMessage(`{"political": "factors"}`),
	}

	// Setup expectations for GetAllTransitiveDependencies
	mockRepo.On("GetByCode", ctx, "SWOT").Return(swotFramework, nil).Once()
	mockRepo.On("GetDependencies", ctx, swotID).Return(swotDependencies, nil)
	mockRepo.On("GetByID", ctx, mercadoID).Return(mercadoFramework, nil).Once()
	mockRepo.On("GetByCode", ctx, "MERCADO").Return(mercadoFramework, nil).Once()
	mockRepo.On("GetDependencies", ctx, mercadoID).Return([]*FrameworkDependency{}, nil)
	mockRepo.On("GetByID", ctx, pestelID).Return(pestelFramework, nil).Once()
	mockRepo.On("GetByCode", ctx, "PESTEL").Return(pestelFramework, nil).Once()
	mockRepo.On("GetDependencies", ctx, pestelID).Return([]*FrameworkDependency{}, nil)

	// Setup expectations for GetDependencyContext
	mockRepo.On("GetByCode", ctx, "MERCADO").Return(mercadoFramework, nil).Once()
	mockRepo.On("GetCurrentResult", ctx, companyID, mercadoID, (*uuid.UUID)(nil)).Return(mercadoResult, nil)
	mockRepo.On("GetByCode", ctx, "PESTEL").Return(pestelFramework, nil).Once()
	mockRepo.On("GetCurrentResult", ctx, companyID, pestelID, (*uuid.UUID)(nil)).Return(pestelResult, nil)

	// Execute
	depContext, err := service.GetDependencyContext(ctx, companyID, "SWOT", nil)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, depContext, 2)
	assert.Contains(t, depContext, "MERCADO")
	assert.Contains(t, depContext, "PESTEL")
	assert.JSONEq(t, `{"market": "data"}`, string(depContext["MERCADO"]))
	assert.JSONEq(t, `{"political": "factors"}`, string(depContext["PESTEL"]))

	mockRepo.AssertExpectations(t)
}

func TestDependencyService_GetAnalysisState(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	service := NewDependencyService(mockRepo, logger)

	ctx := context.Background()
	companyID := uuid.New()

	// Setup test data
	step1 := 1
	stepName1 := "Empresa"
	step3 := 3
	stepName3 := "Ambiente"

	empresaID := uuid.New()
	mercadoID := uuid.New()

	frameworks := []*Framework{
		{
			ID:             empresaID,
			Code:           "EMPRESA",
			Name:           "Dados da Empresa",
			Layer:          0,
			StepNumber:     &step1,
			StepName:       &stepName1,
			ExecutionOrder: 1,
			IsActive:       true,
		},
		{
			ID:             mercadoID,
			Code:           "MERCADO",
			Name:           "Mercado",
			Layer:          1,
			StepNumber:     &step3,
			StepName:       &stepName3,
			ExecutionOrder: 1,
			IsActive:       true,
		},
	}

	// EMPRESA completed, MERCADO pending
	results := []*CompanyFrameworkResult{
		{
			ID:          uuid.New(),
			CompanyID:   companyID,
			FrameworkID: empresaID,
			Status:      StatusCompleted,
			Result:      json.RawMessage(`{"test": true}`),
			IsCurrent:   true,
			Version:     1,
		},
	}

	// Setup expectations
	mockRepo.On("GetActive", ctx).Return(frameworks, nil)
	mockRepo.On("GetCompanyResults", ctx, companyID, (*uuid.UUID)(nil)).Return(results, nil)

	// For CheckReadyToRun calls
	empresaFramework := frameworks[0]
	mercadoFramework := frameworks[1]
	mockRepo.On("GetByCode", ctx, "EMPRESA").Return(empresaFramework, nil)
	mockRepo.On("GetByCode", ctx, "MERCADO").Return(mercadoFramework, nil)
	mockRepo.On("GetDependencies", ctx, mercadoID).Return([]*FrameworkDependency{}, nil)

	// Execute
	state, err := service.GetAnalysisState(ctx, companyID, nil)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, companyID, state.CompanyID)
	assert.Equal(t, 50, state.OverallProgress) // 1 out of 2 completed = 50%
	assert.Len(t, state.Frameworks, 2)

	// Check EMPRESA status
	empresaStatus := state.Frameworks["EMPRESA"]
	assert.Equal(t, StatusCompleted, empresaStatus.Status)
	assert.True(t, empresaStatus.HasResult)
	assert.Equal(t, 1, empresaStatus.Version)

	// Check MERCADO status
	mercadoStatus := state.Frameworks["MERCADO"]
	assert.Equal(t, StatusPending, mercadoStatus.Status)
	assert.False(t, mercadoStatus.HasResult)
	assert.True(t, mercadoStatus.CanExecute) // No dependencies

	mockRepo.AssertExpectations(t)
}

func TestDependencyService_GetAllTransitiveDependencies(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	service := NewDependencyService(mockRepo, logger)

	ctx := context.Background()

	// Setup: SWOT_CROSS -> SWOT -> MERCADO chain
	swotCrossID := uuid.New()
	swotID := uuid.New()
	mercadoID := uuid.New()

	swotCrossFramework := &Framework{ID: swotCrossID, Code: "SWOT_CROSS", Layer: 2}
	swotFramework := &Framework{ID: swotID, Code: "SWOT", Layer: 2}
	mercadoFramework := &Framework{ID: mercadoID, Code: "MERCADO", Layer: 1}

	swotCrossDeps := []*FrameworkDependency{
		{FrameworkID: swotCrossID, DependsOnID: swotID, DependencyType: DependencyRequired},
	}

	swotDeps := []*FrameworkDependency{
		{FrameworkID: swotID, DependsOnID: mercadoID, DependencyType: DependencyRequired},
	}

	// Setup expectations
	mockRepo.On("GetByCode", ctx, "SWOT_CROSS").Return(swotCrossFramework, nil)
	mockRepo.On("GetDependencies", ctx, swotCrossID).Return(swotCrossDeps, nil)
	mockRepo.On("GetByID", ctx, swotID).Return(swotFramework, nil)
	mockRepo.On("GetByCode", ctx, "SWOT").Return(swotFramework, nil)
	mockRepo.On("GetDependencies", ctx, swotID).Return(swotDeps, nil)
	mockRepo.On("GetByID", ctx, mercadoID).Return(mercadoFramework, nil)
	mockRepo.On("GetByCode", ctx, "MERCADO").Return(mercadoFramework, nil)
	mockRepo.On("GetDependencies", ctx, mercadoID).Return([]*FrameworkDependency{}, nil)

	// Execute
	deps, err := service.GetAllTransitiveDependencies(ctx, "SWOT_CROSS")

	// Assert
	assert.NoError(t, err)
	assert.Len(t, deps, 2)
	assert.Contains(t, deps, "SWOT")
	assert.Contains(t, deps, "MERCADO")

	mockRepo.AssertExpectations(t)
}

func TestDependencyService_CheckUpstreamChanged(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	service := NewDependencyService(mockRepo, logger)

	ctx := context.Background()
	companyID := uuid.New()
	resultID := uuid.New()

	// Setup
	swotID := uuid.New()
	mercadoID := uuid.New()
	storedHash := "abc123def456" // Different from computed

	swotFramework := &Framework{ID: swotID, Code: "SWOT", Layer: 2}
	mercadoFramework := &Framework{ID: mercadoID, Code: "MERCADO", Layer: 1}

	swotResult := &CompanyFrameworkResult{
		ID:               resultID,
		CompanyID:        companyID,
		FrameworkID:      swotID,
		Status:           StatusCompleted,
		UpstreamDataHash: &storedHash,
	}

	swotDeps := []*FrameworkDependency{
		{FrameworkID: swotID, DependsOnID: mercadoID, DependencyType: DependencyRequired},
	}

	mercadoResult := &CompanyFrameworkResult{
		ID:          uuid.New(),
		CompanyID:   companyID,
		FrameworkID: mercadoID,
		Status:      StatusCompleted,
		Result:      json.RawMessage(`{"market": "new_data"}`), // Changed data
	}

	// Setup expectations for GetResultByID
	mockRepo.On("GetResultByID", ctx, resultID).Return(swotResult, nil)
	mockRepo.On("GetByID", ctx, swotID).Return(swotFramework, nil)

	// For GetAllTransitiveDependencies
	mockRepo.On("GetByCode", ctx, "SWOT").Return(swotFramework, nil)
	mockRepo.On("GetDependencies", ctx, swotID).Return(swotDeps, nil)
	mockRepo.On("GetByID", ctx, mercadoID).Return(mercadoFramework, nil)
	mockRepo.On("GetByCode", ctx, "MERCADO").Return(mercadoFramework, nil)
	mockRepo.On("GetDependencies", ctx, mercadoID).Return([]*FrameworkDependency{}, nil)

	// For ComputeUpstreamHash
	mockRepo.On("GetCurrentResult", ctx, companyID, mercadoID, (*uuid.UUID)(nil)).Return(mercadoResult, nil)

	// Execute
	changed, err := service.CheckUpstreamChanged(ctx, resultID)

	// Assert
	assert.NoError(t, err)
	assert.True(t, changed) // Hash is different, so upstream changed

	mockRepo.AssertExpectations(t)
}
