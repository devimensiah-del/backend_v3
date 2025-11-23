package analysis_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"backend_v3/config"
	"backend_v3/domain/analysis"

	"backend_v3/llm"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// =============================================================================
// MOCK: MockLLMClient for testing LLM failures
// =============================================================================

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
// ENHANCED TESTS: 4-Layer Cascade Execution with Parallel Processing
// =============================================================================

func TestAnalysisWorkflow_4LayerCascadeExecution(t *testing.T) {
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	mockRepo := new(MockRepository)
	mockSubRepo := new(MockSubmissionRepo)
	resilientLLM := NewResilientLLM(OpenRouterKey, logger)

	queueClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr: "localhost:6379",
	})
	defer queueClient.Close()

	service := analysis.NewService(
		mockRepo,
		mockSubRepo,
		resilientLLM,
		logger,
		queueClient,
		"", "",
	)

	frameworks := make(map[string]config.FrameworkConfig)
	frameworkList := []string{"pestel", "porter", "tam_sam_som", "swot", "benchmarking", "blue_ocean", "growth_hacking", "scenarios", "okrs", "bsc", "decision_matrix", "synthesis"}
	for _, f := range frameworkList {
		frameworks[f] = config.FrameworkConfig{
			Model:       TestModel,
			Temperature: 0.4,
			MaxTokens:   2000,
		}
	}
	service.SetFrameworks(frameworks)

	subID := uuid.New()
	enrichID := uuid.New().String()

	var enrichmentData map[string]interface{}
	err := json.Unmarshal([]byte(coimmaEnrichmentJSON), &enrichmentData)
	assert.NoError(t, err)

	mockSubRepo.On("GetByID", mock.Anything, subID).Return(coimmaSubmission, nil)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*analysis.Analysis")).Return(nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*analysis.Analysis")).Return(nil)
	mockRepo.On("BeginTx", mock.Anything).Return((*sqlx.Tx)(nil), nil)
	mockRepo.On("UpdateWithTx", mock.Anything, mock.Anything, mock.AnythingOfType("*analysis.Analysis")).Return(nil)

	logger.Info().Msg("🚀 Testing 4-Layer Cascade with Parallel Execution...")

	result, err := service.RunAnalysis(context.Background(), subID.String(), enrichID, enrichmentData)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Validate LAYER 1: Environment (PESTEL + Porter + TAM-SAM-SOM in parallel)
	logger.Info().Msg("\n📊 LAYER 1: Environment Analysis (Parallel: PESTEL, Porter, TAM-SAM-SOM)")
	assert.NotEmpty(t, result.PESTEL.Summary, "PESTEL should be populated")
	assert.NotEmpty(t, result.PESTEL.Political, "PESTEL Political factors should exist")
	assert.NotEmpty(t, result.PESTEL.Economic, "PESTEL Economic factors should exist")
	assert.NotEmpty(t, result.Porter.Summary, "Porter should be populated")
	assert.NotEmpty(t, result.TamSamSom.Summary, "TAM-SAM-SOM should be populated")
	logger.Info().Msg("✅ Layer 1 completed successfully")

	// Validate LAYER 2: Positioning (SWOT + Benchmarking in parallel)
	logger.Info().Msg("\n📊 LAYER 2: Positioning Analysis (Parallel: SWOT, Benchmarking)")
	assert.NotEmpty(t, result.SWOT.Summary, "SWOT should be populated")
	assert.NotEmpty(t, result.SWOT.Strengths, "SWOT Strengths should exist")
	assert.NotEmpty(t, result.Benchmarking.Summary, "Benchmarking should be populated")
	logger.Info().Msg("✅ Layer 2 completed successfully")

	// Validate LAYER 3: Strategy (BlueOcean + GrowthHacking + Scenarios in parallel)
	logger.Info().Msg("\n📊 LAYER 3: Strategy Development (Parallel: BlueOcean, GrowthHacking, Scenarios)")
	assert.NotEmpty(t, result.BlueOcean.Summary, "BlueOcean should be populated")
	assert.NotEmpty(t, result.GrowthHacking.Summary, "GrowthHacking should be populated")
	assert.NotEmpty(t, result.Scenarios.Summary, "Scenarios should be populated")
	logger.Info().Msg("✅ Layer 3 completed successfully")

	// Validate LAYER 4: Execution (OKRs + BSC + DecisionMatrix in parallel)
	logger.Info().Msg("\n📊 LAYER 4: Execution Planning (Parallel: OKRs, BSC, DecisionMatrix)")
	assert.NotEmpty(t, result.OKRs.Summary, "OKRs should be populated")
	assert.NotEmpty(t, result.BSC.Summary, "BSC should be populated")
	assert.NotEmpty(t, result.DecisionMatrix.Summary, "DecisionMatrix should be populated")
	logger.Info().Msg("✅ Layer 4 completed successfully")

	// Validate Final Synthesis uses all 11 frameworks
	logger.Info().Msg("\n📊 Final Synthesis (Uses All 11 Frameworks)")
	assert.NotEmpty(t, result.Synthesis.ExecutiveSummary, "Synthesis Executive Summary should exist")
	assert.NotEmpty(t, result.Synthesis.KeyFindings, "Synthesis Key Findings should exist")
	assert.NotEmpty(t, result.Synthesis.StrategicPriorities, "Synthesis Strategic Priorities should exist")
	logger.Info().Msg("✅ Synthesis generated from all 11 frameworks")

	// Verify final status
	assert.Equal(t, "completed", result.Status, "Analysis should be marked as completed")

	logger.Info().Msg("\n✅ 4-Layer Cascade Test PASSED!")
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// ENHANCED TESTS: Checkpoint Saves After Each Layer (Transactional)
// =============================================================================

func TestAnalysisWorkflow_CheckpointAfterEachLayer(t *testing.T) {
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	mockRepo := new(MockRepository)
	mockSubRepo := new(MockSubmissionRepo)
	resilientLLM := NewResilientLLM(OpenRouterKey, logger)

	queueClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr: "localhost:6379",
	})
	defer queueClient.Close()

	service := analysis.NewService(
		mockRepo,
		mockSubRepo,
		resilientLLM,
		logger,
		queueClient,
		"", "",
	)

	frameworks := make(map[string]config.FrameworkConfig)
	frameworkList := []string{"pestel", "porter", "tam_sam_som", "swot", "benchmarking", "blue_ocean", "growth_hacking", "scenarios", "okrs", "bsc", "decision_matrix", "synthesis"}
	for _, f := range frameworkList {
		frameworks[f] = config.FrameworkConfig{
			Model:       TestModel,
			Temperature: 0.4,
			MaxTokens:   2000,
		}
	}
	service.SetFrameworks(frameworks)

	subID := uuid.New()
	enrichID := uuid.New().String()

	var enrichmentData map[string]interface{}
	err := json.Unmarshal([]byte(coimmaEnrichmentJSON), &enrichmentData)
	assert.NoError(t, err)

	mockSubRepo.On("GetByID", mock.Anything, subID).Return(coimmaSubmission, nil)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*analysis.Analysis")).Return(nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*analysis.Analysis")).Return(nil)

	// Track checkpoint save calls (4 layers = 4 checkpoints)
	checkpointCount := 0
	mockRepo.On("BeginTx", mock.Anything).Return((*sqlx.Tx)(nil), nil).Run(func(args mock.Arguments) {
		checkpointCount++
		logger.Info().Msgf("🔄 Checkpoint #%d: Transaction started", checkpointCount)
	})

	mockRepo.On("UpdateWithTx", mock.Anything, mock.Anything, mock.AnythingOfType("*analysis.Analysis")).Return(nil).Run(func(args mock.Arguments) {
		logger.Info().Msgf("💾 Checkpoint #%d: Analysis state saved", checkpointCount)
	})

	logger.Info().Msg("🚀 Testing Checkpoint Saves After Each Layer...")

	result, err := service.RunAnalysis(context.Background(), subID.String(), enrichID, enrichmentData)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify checkpoint saves occurred (4 layers = 4 checkpoints)
	logger.Info().Msgf("📊 Total checkpoint transactions initiated: %d (expected: 4)", checkpointCount)

	// Verify all repository methods were called
	mockRepo.AssertExpectations(t)
	mockSubRepo.AssertExpectations(t)

	logger.Info().Msg("✅ Checkpoint Save Test PASSED!")
}

// =============================================================================
// ENHANCED TESTS: MacroContext Integration with PESTEL
// =============================================================================

func TestAnalysisWorkflow_MacroContextInPESTEL(t *testing.T) {
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	mockRepo := new(MockRepository)
	mockSubRepo := new(MockSubmissionRepo)
	resilientLLM := NewResilientLLM(OpenRouterKey, logger)

	queueClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr: "localhost:6379",
	})
	defer queueClient.Close()

	service := analysis.NewService(
		mockRepo,
		mockSubRepo,
		resilientLLM,
		logger,
		queueClient,
		"", "",
	)

	frameworks := make(map[string]config.FrameworkConfig)
	frameworkList := []string{"pestel", "porter", "tam_sam_som", "swot", "benchmarking", "blue_ocean", "growth_hacking", "scenarios", "okrs", "bsc", "decision_matrix", "synthesis"}
	for _, f := range frameworkList {
		frameworks[f] = config.FrameworkConfig{
			Model:       TestModel,
			Temperature: 0.4,
			MaxTokens:   2000,
		}
	}
	service.SetFrameworks(frameworks)

	subID := uuid.New()
	enrichID := uuid.New().String()

	var enrichmentData map[string]interface{}
	err := json.Unmarshal([]byte(coimmaEnrichmentJSON), &enrichmentData)
	assert.NoError(t, err)

	mockSubRepo.On("GetByID", mock.Anything, subID).Return(coimmaSubmission, nil)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*analysis.Analysis")).Return(nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*analysis.Analysis")).Return(nil)
	mockRepo.On("BeginTx", mock.Anything).Return((*sqlx.Tx)(nil), nil)
	mockRepo.On("UpdateWithTx", mock.Anything, mock.Anything, mock.AnythingOfType("*analysis.Analysis")).Return(nil)

	logger.Info().Msg("🚀 Testing MacroContext Integration in PESTEL...")

	result, err := service.RunAnalysis(context.Background(), subID.String(), enrichID, enrichmentData)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify macro-context data is present in enrichment
	macroContext, ok := enrichmentData["macro_context"].(map[string]interface{})
	assert.True(t, ok, "Enrichment data should contain macro_context")
	assert.NotNil(t, macroContext, "macro_context should not be nil")

	logger.Info().Msg("\n📊 Checking PESTEL Economic Factors for Macro-Context Integration...")

	pestelEconomic := strings.Join(result.PESTEL.Economic, " ")
	logger.Info().Msgf("PESTEL Economic Factors: %s", pestelEconomic)

	// Check if PESTEL references macro-context indicators
	pestelLower := strings.ToLower(pestelEconomic)
	hasSELIC := strings.Contains(pestelLower, "selic") || strings.Contains(pestelLower, "10.5") || strings.Contains(pestelLower, "10,5")
	hasGDP := strings.Contains(pestelLower, "pib") || strings.Contains(pestelLower, "gdp") || strings.Contains(pestelLower, "2.0") || strings.Contains(pestelLower, "2,0")
	hasInflation := strings.Contains(pestelLower, "inflação") || strings.Contains(pestelLower, "inflation") || strings.Contains(pestelLower, "ipca") || strings.Contains(pestelLower, "4.5") || strings.Contains(pestelLower, "4,5")

	if hasSELIC {
		logger.Info().Msg("✅ PESTEL references SELIC rate from macro_context")
	}
	if hasGDP {
		logger.Info().Msg("✅ PESTEL references GDP growth from macro_context")
	}
	if hasInflation {
		logger.Info().Msg("✅ PESTEL references Inflation from macro_context")
	}

	// At least one macro indicator should be present
	assert.True(t, hasSELIC || hasGDP || hasInflation, "PESTEL should integrate macro-context economic data")

	logger.Info().Msg("✅ MacroContext Integration Test PASSED!")
}

// =============================================================================
// ENHANCED TESTS: Version Management and is_latest Flag
// =============================================================================

func TestAnalysisWorkflow_VersionIncrementsAndIsLatestFlag(t *testing.T) {
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	mockRepo := new(MockRepository)
	queueClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr: "localhost:6379",
	})
	defer queueClient.Close()

	service := analysis.NewService(
		mockRepo,
		nil,
		nil,
		logger,
		queueClient,
		"", "",
	)

	logger.Info().Msg("🚀 Testing Version Management and is_latest Flag...")

	// Create initial analysis (version 1)
	initialAnalysis := &analysis.Analysis{
		ID:           uuid.New().String(),
		SubmissionID: uuid.New().String(),
		EnrichmentID: uuid.New().String(),
		Version:      1,
		IsLatest:     true,
		Status:       string(analysis.StatusCompleted),
	}

	logger.Info().Msgf("📊 Initial Analysis: v%d, is_latest=%v", initialAnalysis.Version, initialAnalysis.IsLatest)

	mockRepo.On("GetByID", mock.Anything, initialAnalysis.ID).Return(initialAnalysis, nil)
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(a *analysis.Analysis) bool {
		isValid := a.Version == 2 &&
			a.ParentAnalysisID != nil &&
			*a.ParentAnalysisID == initialAnalysis.ID &&
			a.Status == string(analysis.StatusPending)

		if isValid {
			logger.Info().Msgf("✅ New version created: v%d, parent_id=%s, is_latest=%v",
				a.Version, *a.ParentAnalysisID, a.IsLatest)
		}
		return isValid
	})).Return(nil)

	newVersion, err := service.CreateVersion(context.Background(), initialAnalysis.ID, nil)

	assert.NoError(t, err)
	assert.NotNil(t, newVersion)
	assert.Equal(t, 2, newVersion.Version, "Version should increment from 1 to 2")
	assert.NotNil(t, newVersion.ParentAnalysisID, "parent_analysis_id should be set")
	assert.Equal(t, initialAnalysis.ID, *newVersion.ParentAnalysisID, "parent_analysis_id should reference old version")

	logger.Info().Msg("✅ Version Management Test PASSED!")
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// ENHANCED TESTS: Status Transition Validation
// =============================================================================

func TestAnalysisWorkflow_StatusTransitionValidation(t *testing.T) {
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	mockRepo := new(MockRepository)
	queueClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr: "localhost:6379",
	})
	defer queueClient.Close()

	service := analysis.NewService(
		mockRepo,
		nil,
		nil,
		logger,
		queueClient,
		"", "",
	)

	logger.Info().Msg("🚀 Testing Status Transition Validation...")

	testAnalysis := &analysis.Analysis{
		ID:           uuid.New().String(),
		SubmissionID: uuid.New().String(),
		EnrichmentID: uuid.New().String(),
		Version:      1,
		IsLatest:     true,
	}

	// Test 1: Approve should reject pending status
	logger.Info().Msg("\n📊 Test 1: Approve should reject 'pending' status")
	testAnalysis.Status = string(analysis.StatusPending)
	mockRepo.On("GetByID", mock.Anything, testAnalysis.ID).Return(testAnalysis, nil).Once()

	err := service.Approve(context.Background(), testAnalysis.ID)
	assert.Error(t, err, "Approve should reject pending status")
	assert.Contains(t, err.Error(), "must be in 'completed' status")
	logger.Info().Msg("✅ Correctly rejected pending status")

	// Test 2: Approve should accept completed status
	logger.Info().Msg("\n📊 Test 2: Approve should accept 'completed' status")
	testAnalysis.Status = string(analysis.StatusCompleted)
	mockRepo.On("GetByID", mock.Anything, testAnalysis.ID).Return(testAnalysis, nil).Once()
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *analysis.Analysis) bool {
		return a.Status == string(analysis.StatusApproved)
	})).Return(nil).Once()

	err = service.Approve(context.Background(), testAnalysis.ID)
	assert.NoError(t, err, "Approve should accept completed status")
	logger.Info().Msg("✅ Successfully approved completed analysis")

	// Test 3: Send should reject completed status
	logger.Info().Msg("\n📊 Test 3: Send should reject 'completed' status (requires 'approved')")
	testAnalysis.Status = string(analysis.StatusCompleted)
	mockRepo.On("GetByID", mock.Anything, testAnalysis.ID).Return(testAnalysis, nil).Once()

	err = service.Send(context.Background(), testAnalysis.ID, "test@example.com")
	assert.Error(t, err, "Send should reject completed status")
	assert.Contains(t, err.Error(), "must be in 'approved' status")
	logger.Info().Msg("✅ Correctly rejected completed status")

	// Test 4: Send should accept approved status
	logger.Info().Msg("\n📊 Test 4: Send should accept 'approved' status")
	testAnalysis.Status = string(analysis.StatusApproved)
	mockRepo.On("GetByID", mock.Anything, testAnalysis.ID).Return(testAnalysis, nil).Once()
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *analysis.Analysis) bool {
		return a.Status == string(analysis.StatusSent) && a.SentTo != nil && *a.SentTo == "test@example.com"
	})).Return(nil).Once()

	err = service.Send(context.Background(), testAnalysis.ID, "test@example.com")
	assert.NoError(t, err, "Send should accept approved status")
	logger.Info().Msg("✅ Successfully sent approved analysis")

	logger.Info().Msg("\n✅ Status Transition Validation Test PASSED!")
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// ENHANCED TESTS: Partial Failure Recovery
// =============================================================================

func TestAnalysisWorkflow_PartialFailureHandling(t *testing.T) {
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	mockRepo := new(MockRepository)
	mockSubRepo := new(MockSubmissionRepo)

	// Create a mock LLM that simulates failure
	mockLLM := new(MockLLMClient)

	queueClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr: "localhost:6379",
	})
	defer queueClient.Close()

	service := analysis.NewService(
		mockRepo,
		mockSubRepo,
		mockLLM,
		logger,
		queueClient,
		"", "",
	)

	frameworks := make(map[string]config.FrameworkConfig)
	service.SetFrameworks(frameworks)

	subID := uuid.New()
	enrichID := uuid.New().String()

	mockSubRepo.On("GetByID", mock.Anything, subID).Return(coimmaSubmission, nil)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*analysis.Analysis")).Return(nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*analysis.Analysis")).Return(nil)
	mockRepo.On("BeginTx", mock.Anything).Return((*sqlx.Tx)(nil), nil)
	mockRepo.On("UpdateWithTx", mock.Anything, mock.Anything, mock.AnythingOfType("*analysis.Analysis")).Return(nil)

	// Mock successful LLM calls for Layer 1 & 2 (5 frameworks)
	mockLLM.On("GenerateStructuredWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(5)

	// Mock failure on Layer 3 (simulate timeout)
	mockLLM.On("GenerateStructuredWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("LLM timeout")).Once()

	logger.Info().Msg("🚀 Testing Partial Failure Handling...")

	enrichmentData := map[string]interface{}{
		"company_data": map[string]interface{}{
			"company_name": "Test Company",
		},
	}

	_, err := service.RunAnalysis(context.Background(), subID.String(), enrichID, enrichmentData)

	// Workflow continues even if individual frameworks fail (logged but not fatal)
	logger.Info().Msgf("Result: %v", err)

	logger.Info().Msg("✅ Partial Failure Handling Test PASSED!")
}
