package framework

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestPromptBuilder_BuildPrompt_InitialMode(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	depService := NewDependencyService(mockRepo, logger)
	builder := NewPromptBuilder(mockRepo, depService, logger)

	ctx := context.Background()
	companyID := uuid.New()

	// Setup framework
	swotID := uuid.New()
	systemPrompt := "Você é um especialista em SWOT."
	swotFramework := &Framework{
		ID:           swotID,
		Code:         "SWOT",
		Name:         "Análise SWOT",
		Layer:        2,
		PromptSystem: &systemPrompt,
		PromptUser:   "Analise os dados da empresa: {{EMPRESA}}\nDesafio: {{DESAFIO}}",
	}

	// Setup company and challenge data
	companyData := json.RawMessage(`{"name": "Empresa Teste", "sector": "Tecnologia"}`)
	challengeData := json.RawMessage(`{"description": "Expandir para novo mercado"}`)

	// Setup expectations
	mockRepo.On("GetByCode", ctx, "SWOT").Return(swotFramework, nil).Once()
	// For GetDependencyContext -> GetAllTransitiveDependencies
	mockRepo.On("GetByCode", ctx, "SWOT").Return(swotFramework, nil).Once()
	mockRepo.On("GetDependencies", ctx, swotID).Return([]*FrameworkDependency{}, nil)

	// Execute
	result, err := builder.BuildPrompt(ctx, &PromptContext{
		CompanyID:     companyID,
		FrameworkCode: "SWOT",
		Mode:          PromptModeInitial,
		CompanyData:   companyData,
		ChallengeData: challengeData,
	})

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, PromptModeInitial, result.Mode)
	assert.Contains(t, result.SystemPrompt, "especialista em SWOT")
	assert.Contains(t, result.SystemPrompt, "MODO INICIAL")
	assert.Contains(t, result.UserPrompt, "Empresa Teste")
	assert.Contains(t, result.UserPrompt, "Expandir para novo mercado")

	mockRepo.AssertExpectations(t)
}

func TestPromptBuilder_BuildPrompt_UpdateMode(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	depService := NewDependencyService(mockRepo, logger)
	builder := NewPromptBuilder(mockRepo, depService, logger)

	ctx := context.Background()
	companyID := uuid.New()

	// Setup framework with update prompt
	swotID := uuid.New()
	updatePrompt := "Atualize a análise SWOT considerando as mudanças."
	swotFramework := &Framework{
		ID:           swotID,
		Code:         "SWOT",
		Name:         "Análise SWOT",
		Layer:        2,
		PromptUser:   "Gere análise SWOT inicial",
		PromptUpdate: &updatePrompt,
	}

	currentResult := json.RawMessage(`{"strengths": ["Equipe experiente"], "weaknesses": ["Falta de recursos"]}`)
	refinement := "Adicione mais detalhes sobre as oportunidades"

	// Setup expectations
	mockRepo.On("GetByCode", ctx, "SWOT").Return(swotFramework, nil).Once()
	mockRepo.On("GetByCode", ctx, "SWOT").Return(swotFramework, nil).Once()
	mockRepo.On("GetDependencies", ctx, swotID).Return([]*FrameworkDependency{}, nil)

	// Execute
	result, err := builder.BuildPrompt(ctx, &PromptContext{
		CompanyID:      companyID,
		FrameworkCode:  "SWOT",
		Mode:           PromptModeUpdate,
		CurrentResult:  currentResult,
		UserRefinement: &refinement,
	})

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, PromptModeUpdate, result.Mode)
	assert.Contains(t, result.SystemPrompt, "MODO DE ATUALIZAÇÃO")
	assert.Contains(t, result.SystemPrompt, "PRESERVE informações válidas")
	assert.Contains(t, result.UserPrompt, "ANÁLISE ATUAL")
	assert.Contains(t, result.UserPrompt, "Equipe experiente")
	assert.Contains(t, result.UserPrompt, "FEEDBACK DO USUÁRIO")
	assert.Contains(t, result.UserPrompt, "Adicione mais detalhes sobre as oportunidades")

	mockRepo.AssertExpectations(t)
}

func TestPromptBuilder_BuildPrompt_WithDependencyContext(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	depService := NewDependencyService(mockRepo, logger)
	builder := NewPromptBuilder(mockRepo, depService, logger)

	ctx := context.Background()
	companyID := uuid.New()

	// Setup frameworks
	swotID := uuid.New()
	mercadoID := uuid.New()

	swotFramework := &Framework{
		ID:         swotID,
		Code:       "SWOT",
		Name:       "Análise SWOT",
		Layer:      2,
		PromptUser: "Considere os dados de mercado: {{MERCADO}}\nGere SWOT.",
	}

	mercadoFramework := &Framework{
		ID:    mercadoID,
		Code:  "MERCADO",
		Name:  "Mercado",
		Layer: 1,
	}

	// SWOT depends on MERCADO
	swotDependencies := []*FrameworkDependency{
		{FrameworkID: swotID, DependsOnID: mercadoID, DependencyType: DependencyRequired},
	}

	mercadoResult := &CompanyFrameworkResult{
		ID:          uuid.New(),
		CompanyID:   companyID,
		FrameworkID: mercadoID,
		Status:      StatusCompleted,
		Result:      json.RawMessage(`{"tam": "R$50B", "growth": "15% a.a."}`),
	}

	// Setup expectations for BuildPrompt
	mockRepo.On("GetByCode", ctx, "SWOT").Return(swotFramework, nil).Once()

	// For GetDependencyContext -> GetAllTransitiveDependencies
	mockRepo.On("GetByCode", ctx, "SWOT").Return(swotFramework, nil).Once()
	mockRepo.On("GetDependencies", ctx, swotID).Return(swotDependencies, nil)
	mockRepo.On("GetByID", ctx, mercadoID).Return(mercadoFramework, nil)
	mockRepo.On("GetByCode", ctx, "MERCADO").Return(mercadoFramework, nil).Once()
	mockRepo.On("GetDependencies", ctx, mercadoID).Return([]*FrameworkDependency{}, nil)

	// For GetDependencyContext data retrieval
	mockRepo.On("GetByCode", ctx, "MERCADO").Return(mercadoFramework, nil).Once()
	mockRepo.On("GetCurrentResult", ctx, companyID, mercadoID, (*uuid.UUID)(nil)).Return(mercadoResult, nil)

	// Execute
	result, err := builder.BuildPrompt(ctx, &PromptContext{
		CompanyID:     companyID,
		FrameworkCode: "SWOT",
		Mode:          PromptModeInitial,
	})

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.UserPrompt, "R$50B")
	assert.Contains(t, result.UserPrompt, "15% a.a.")

	mockRepo.AssertExpectations(t)
}

func TestPromptBuilder_ReplacePlaceholders(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	depService := NewDependencyService(mockRepo, logger)
	builder := NewPromptBuilder(mockRepo, depService, logger)

	companyData := json.RawMessage(`{"name": "ACME Corp"}`)
	challengeData := json.RawMessage(`{"goal": "Crescer 50%"}`)
	depContext := map[string]json.RawMessage{
		"MERCADO": json.RawMessage(`{"tam": "R$100M"}`),
		"PESTEL":  json.RawMessage(`{"political": "estável"}`),
	}

	prompt := "Empresa: {{EMPRESA}}\nDesafio: {{DESAFIO}}\nMercado: {{MERCADO}}\nPESTEL: {{PESTEL}}"

	result := builder.replacePlaceholders(prompt, &PromptContext{
		CompanyData:   companyData,
		ChallengeData: challengeData,
	}, depContext)

	assert.Contains(t, result, "ACME Corp")
	assert.Contains(t, result, "Crescer 50%")
	assert.Contains(t, result, "R$100M")
	assert.Contains(t, result, "estável")
}

func TestExecutionService_PrepareExecution_InitialMode(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	depService := NewDependencyService(mockRepo, logger)
	promptBuilder := NewPromptBuilder(mockRepo, depService, logger)
	execService := NewExecutionService(mockRepo, depService, promptBuilder, logger)

	ctx := context.Background()
	companyID := uuid.New()

	// Setup
	swotID := uuid.New()
	swotFramework := &Framework{
		ID:         swotID,
		Code:       "SWOT",
		Name:       "Análise SWOT",
		Layer:      2,
		PromptUser: "Gere SWOT",
	}

	// CheckReadyToRun expectations
	mockRepo.On("GetByCode", ctx, "SWOT").Return(swotFramework, nil).Times(4)
	mockRepo.On("GetDependencies", ctx, swotID).Return([]*FrameworkDependency{}, nil).Times(2)

	// GetCurrentResult (no existing result)
	mockRepo.On("GetCurrentResult", ctx, companyID, swotID, (*uuid.UUID)(nil)).Return(nil, nil)

	// Execute
	req := &ExecuteRequest{
		CompanyID:     companyID,
		FrameworkCode: "SWOT",
	}

	result, err := execService.PrepareExecution(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, PromptModeInitial, result.Mode)

	mockRepo.AssertExpectations(t)
}

func TestExecutionService_PrepareExecution_UpdateMode(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	depService := NewDependencyService(mockRepo, logger)
	promptBuilder := NewPromptBuilder(mockRepo, depService, logger)
	execService := NewExecutionService(mockRepo, depService, promptBuilder, logger)

	ctx := context.Background()
	companyID := uuid.New()

	// Setup
	swotID := uuid.New()
	swotFramework := &Framework{
		ID:         swotID,
		Code:       "SWOT",
		Name:       "Análise SWOT",
		Layer:      2,
		PromptUser: "Gere SWOT",
	}

	existingResult := &CompanyFrameworkResult{
		ID:          uuid.New(),
		CompanyID:   companyID,
		FrameworkID: swotID,
		Status:      StatusCompleted,
		Result:      json.RawMessage(`{"strengths": ["test"]}`),
		Version:     1,
	}

	// CheckReadyToRun expectations
	mockRepo.On("GetByCode", ctx, "SWOT").Return(swotFramework, nil).Times(4)
	mockRepo.On("GetDependencies", ctx, swotID).Return([]*FrameworkDependency{}, nil).Times(2)

	// GetCurrentResult (existing completed result)
	mockRepo.On("GetCurrentResult", ctx, companyID, swotID, (*uuid.UUID)(nil)).Return(existingResult, nil)

	// Execute
	req := &ExecuteRequest{
		CompanyID:     companyID,
		FrameworkCode: "SWOT",
	}

	result, err := execService.PrepareExecution(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, PromptModeUpdate, result.Mode)
	assert.Contains(t, result.UserPrompt, "ANÁLISE ATUAL")

	mockRepo.AssertExpectations(t)
}

func TestExecutionService_CompareResults(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	depService := NewDependencyService(mockRepo, logger)
	promptBuilder := NewPromptBuilder(mockRepo, depService, logger)
	execService := NewExecutionService(mockRepo, depService, promptBuilder, logger)

	previousResult := json.RawMessage(`{
		"strengths": ["Equipe experiente", "Marca forte"],
		"weaknesses": ["Recursos limitados"],
		"opportunities": ["Novo mercado"]
	}`)

	newResult := json.RawMessage(`{
		"strengths": ["Equipe experiente", "Marca forte", "Nova tecnologia"],
		"weaknesses": ["Recursos limitados", "Concorrência"],
		"opportunities": ["Novo mercado"],
		"threats": ["Regulamentação"]
	}`)

	suggestion, err := execService.CompareResults(previousResult, newResult)

	assert.NoError(t, err)
	assert.NotNil(t, suggestion)
	assert.Contains(t, suggestion.ChangesSummary, "campos")
	assert.NotEmpty(t, suggestion.FieldChanges)
}

func TestPromptBuilder_InjectCurrentResult(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	depService := NewDependencyService(mockRepo, logger)
	builder := NewPromptBuilder(mockRepo, depService, logger)

	prompt := "Gere análise SWOT."
	currentResult := json.RawMessage(`{"strengths": ["Teste"]}`)
	refinement := "Foco em oportunidades de expansão"

	result := builder.injectCurrentResult(prompt, currentResult, &refinement)

	assert.Contains(t, result, "ANÁLISE ATUAL")
	assert.Contains(t, result, "Teste")
	assert.Contains(t, result, "FEEDBACK DO USUÁRIO")
	assert.Contains(t, result, "Foco em oportunidades de expansão")
}

func TestPromptBuilder_InjectCurrentResult_NoRefinement(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := zerolog.Nop()
	depService := NewDependencyService(mockRepo, logger)
	builder := NewPromptBuilder(mockRepo, depService, logger)

	prompt := "Gere análise SWOT."
	currentResult := json.RawMessage(`{"strengths": ["Teste"]}`)

	result := builder.injectCurrentResult(prompt, currentResult, nil)

	assert.Contains(t, result, "ANÁLISE ATUAL")
	assert.Contains(t, result, "Teste")
	assert.NotContains(t, result, "FEEDBACK DO USUÁRIO")
	assert.Contains(t, result, "Atualize esta análise")
}
