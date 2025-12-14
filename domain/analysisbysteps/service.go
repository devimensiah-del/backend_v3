package analysisbysteps

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"backend_v3/config"
	"backend_v3/domain/analysis"
	"backend_v3/domain/challenge"
	"backend_v3/domain/company"
	"backend_v3/llm"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service handles business logic for analysis steps with human-in-the-loop editing
type Service struct {
	repo           *Repository
	analysisRepo   analysis.Repository
	companyService *company.Service
	challengeRepo  challenge.Repository
	llmClient      *llm.Client
	frameworks     map[string]config.FrameworkConfig
	logger         zerolog.Logger
}

// NewService creates a new Service with all required dependencies
func NewService(
	repo *Repository,
	analysisRepo analysis.Repository,
	companyService *company.Service,
	challengeRepo challenge.Repository,
	llmClient *llm.Client,
	frameworks map[string]config.FrameworkConfig,
	logger zerolog.Logger,
) *Service {
	return &Service{
		repo:           repo,
		analysisRepo:   analysisRepo,
		companyService: companyService,
		challengeRepo:  challengeRepo,
		llmClient:      llmClient,
		frameworks:     frameworks,
		logger:         logger.With().Str("service", "analysisbysteps").Logger(),
	}
}

// StartAnalysisBySteps creates a new step-by-step analysis with status='draft', current_step=0
// Creates all 14 AnalysisStep records in pending status
func (s *Service) StartAnalysisBySteps(ctx context.Context, challengeID uuid.UUID) (*StartResponse, error) {
	s.logger.Info().
		Str("challenge_id", challengeID.String()).
		Msg("Starting step-by-step analysis")

	// 1. Fetch the challenge to get company_id
	chal, err := s.challengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch challenge: %w", err)
	}
	if chal == nil {
		return nil, fmt.Errorf("challenge not found: %s", challengeID.String())
	}

	// 2. Check if an analysis already exists for this challenge
	existingAnalysis, err := s.analysisRepo.GetByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing analysis: %w", err)
	}

	var analysisID string
	if existingAnalysis != nil {
		// Reuse existing analysis
		analysisID = existingAnalysis.ID
		s.logger.Info().
			Str("analysis_id", analysisID).
			Msg("Reusing existing analysis for challenge")
	} else {
		// 3. Create a new Analysis record
		companyIDStr := chal.CompanyID.String()
		newAnalysis := &analysis.Analysis{
			ID:               uuid.New().String(),
			CompanyID:        &companyIDStr,
			ChallengeID:      challengeID,
			Status:           string(analysis.StatusPending),
			IsVisibleToUser:  false,
			IsPublic:         false,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			Version:          1,
			CurrentStep:      0,
			WizardMode:       false,
			FrameworkResults: make(analysis.FrameworkResultsJSONB),
		}

		if err := s.analysisRepo.Create(ctx, newAnalysis); err != nil {
			return nil, fmt.Errorf("failed to create analysis: %w", err)
		}

		analysisID = newAnalysis.ID
		s.logger.Info().
			Str("analysis_id", analysisID).
			Msg("Created new analysis record")
	}

	// 4. Check if steps already exist
	existingSteps, err := s.repo.GetByAnalysisID(ctx, analysisID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing steps: %w", err)
	}

	if len(existingSteps) > 0 {
		// Steps already exist, return them
		s.logger.Info().
			Str("analysis_id", analysisID).
			Int("step_count", len(existingSteps)).
			Msg("Steps already exist, returning existing data")

		return &StartResponse{
			AnalysisID:  analysisID,
			ChallengeID: challengeID.String(),
			TotalSteps:  TotalSteps(),
			CurrentStep: 0,
			Steps:       existingSteps,
		}, nil
	}

	// 5. Create all 14 AnalysisStep records in pending status
	steps := make([]AnalysisStep, 0, TotalSteps())
	for i, meta := range FrameworkOrder {
		step := AnalysisStep{
			AnalysisID:    analysisID,
			FrameworkCode: meta.Code,
			StepNumber:    i,
			Status:        StatusPending,
			Visible:       true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := s.repo.Create(ctx, &step); err != nil {
			return nil, fmt.Errorf("failed to create step %d (%s): %w", i, meta.Code, err)
		}

		steps = append(steps, step)
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Int("steps_created", len(steps)).
		Msg("Successfully created all analysis steps")

	return &StartResponse{
		AnalysisID:  analysisID,
		ChallengeID: challengeID.String(),
		TotalSteps:  TotalSteps(),
		CurrentStep: 0,
		Steps:       steps,
	}, nil
}

// GenerateStep calls LLM for current step with context from all previous approved steps
// Preserves existing human_edited if present (doesn't overwrite)
// Sets status to 'generating' before LLM call, then 'generated' on success
func (s *Service) GenerateStep(ctx context.Context, analysisID string, stepNumber int) (*AnalysisStep, error) {
	s.logger.Info().
		Str("analysis_id", analysisID).
		Int("step_number", stepNumber).
		Msg("Generating step with LLM")

	// 1. Validate step number
	if stepNumber < 0 || stepNumber >= TotalSteps() {
		return nil, fmt.Errorf("invalid step number: %d (must be 0-%d)", stepNumber, TotalSteps()-1)
	}

	// 2. Get the analysis record
	analysisRecord, err := s.analysisRepo.GetByID(ctx, analysisID)
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis: %w", err)
	}
	if analysisRecord == nil {
		return nil, fmt.Errorf("analysis not found: %s", analysisID)
	}

	// 3. Get all steps for this analysis
	allSteps, err := s.repo.GetByAnalysisID(ctx, analysisID)
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis steps: %w", err)
	}

	// Find the current step
	var currentStep *AnalysisStep
	for i := range allSteps {
		if allSteps[i].StepNumber == stepNumber {
			currentStep = &allSteps[i]
			break
		}
	}

	if currentStep == nil {
		return nil, fmt.Errorf("step %d not found for analysis %s", stepNumber, analysisID)
	}

	// 4. Validate all previous steps are approved (can't skip steps)
	for i := 0; i < stepNumber; i++ {
		previousStep := &allSteps[i]
		if !previousStep.IsApproved() {
			return nil, fmt.Errorf("cannot generate step %d: previous step %d (%s) is not approved yet",
				stepNumber, i, previousStep.FrameworkCode)
		}
	}

	// 5. Set status to 'generating' before LLM call
	currentStep.Status = StatusGenerating
	currentStep.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, currentStep); err != nil {
		return nil, fmt.Errorf("failed to update step status to generating: %w", err)
	}

	// 6. Build context with all approved previous steps
	previousSteps := allSteps[:stepNumber]
	stepContext, err := s.buildStepContext(ctx, analysisRecord, previousSteps)
	if err != nil {
		// Set status to failed
		currentStep.Status = StatusFailed
		currentStep.UpdatedAt = time.Now()
		_ = s.repo.Update(ctx, currentStep)
		return nil, fmt.Errorf("failed to build step context: %w", err)
	}

	// 7. Get framework configuration
	frameworkMeta := GetFrameworkMeta(currentStep.FrameworkCode)
	if frameworkMeta == nil {
		currentStep.Status = StatusFailed
		currentStep.UpdatedAt = time.Now()
		_ = s.repo.Update(ctx, currentStep)
		return nil, fmt.Errorf("framework metadata not found for code: %s", currentStep.FrameworkCode)
	}

	frameworkConfig, ok := s.frameworks[currentStep.FrameworkCode]
	if !ok {
		// Use default config if not found
		frameworkConfig = config.FrameworkConfig{
			Model:         "google/gemini-2.5-flash",
			Temperature:   0.5,
			MaxTokens:     8000,
			FallbackModel: "openai/gpt-4.1-mini",
		}
		s.logger.Warn().
			Str("framework_code", currentStep.FrameworkCode).
			Msg("Using default framework config - config not found")
	}

	// 8. Call LLM with framework-specific prompt and config
	opts := llm.NewGenerationOptions(frameworkConfig)
	prompt := getPromptForFramework(currentStep.FrameworkCode)

	var result map[string]interface{}
	err = s.llmClient.GenerateStructuredWithOptions(ctx, opts, prompt, stepContext, &result)
	if err != nil {
		// Set status to failed
		currentStep.Status = StatusFailed
		currentStep.UpdatedAt = time.Now()
		_ = s.repo.Update(ctx, currentStep)

		s.logger.Error().
			Err(err).
			Str("analysis_id", analysisID).
			Int("step_number", stepNumber).
			Str("framework_code", currentStep.FrameworkCode).
			Msg("LLM generation failed")

		return nil, fmt.Errorf("LLM generation failed for %s: %w", currentStep.FrameworkCode, err)
	}

	// 9. On success: set ai_output, status=generated, generated_at
	outputJSON, err := json.Marshal(result)
	if err != nil {
		currentStep.Status = StatusFailed
		currentStep.UpdatedAt = time.Now()
		_ = s.repo.Update(ctx, currentStep)
		return nil, fmt.Errorf("failed to marshal LLM output: %w", err)
	}

	outputStr := string(outputJSON)
	now := time.Now()
	currentStep.AIOutput = &outputStr
	currentStep.Status = StatusGenerated
	currentStep.GeneratedAt = &now
	currentStep.UpdatedAt = now

	// Note: We preserve human_edited if it exists (don't clear it)
	if err := s.repo.Update(ctx, currentStep); err != nil {
		return nil, fmt.Errorf("failed to update step with LLM output: %w", err)
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Int("step_number", stepNumber).
		Str("framework_code", currentStep.FrameworkCode).
		Msg("Successfully generated step with LLM")

	return currentStep, nil
}

// SaveHumanEdit saves human edit without calling LLM
// Only updates human_edited field, does NOT change status
func (s *Service) SaveHumanEdit(ctx context.Context, stepID string, editedJSON string) (*AnalysisStep, error) {
	s.logger.Info().
		Str("step_id", stepID).
		Msg("Saving human edit")

	// 1. Get the step
	step, err := s.repo.GetByID(ctx, stepID)
	if err != nil {
		return nil, fmt.Errorf("failed to get step: %w", err)
	}
	if step == nil {
		return nil, fmt.Errorf("step not found: %s", stepID)
	}

	// 2. Validate JSON
	var testJSON map[string]interface{}
	if err := json.Unmarshal([]byte(editedJSON), &testJSON); err != nil {
		return nil, fmt.Errorf("invalid JSON in human edit: %w", err)
	}

	// 3. Update human_edited using repository method (preserves status)
	if err := s.repo.SetHumanEdited(ctx, stepID, editedJSON); err != nil {
		return nil, fmt.Errorf("failed to save human edit: %w", err)
	}

	// 4. Fetch updated step
	updatedStep, err := s.repo.GetByID(ctx, stepID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated step: %w", err)
	}

	s.logger.Info().
		Str("step_id", stepID).
		Str("framework_code", updatedStep.FrameworkCode).
		Msg("Successfully saved human edit")

	return updatedStep, nil
}

// ApproveAndAdvance validates output, sets approved_at, status=approved
// Returns the approved step and next step info
func (s *Service) ApproveAndAdvance(ctx context.Context, stepID string) (*ApproveResponse, error) {
	s.logger.Info().
		Str("step_id", stepID).
		Msg("Approving step and advancing")

	// 1. Get the step
	step, err := s.repo.GetByID(ctx, stepID)
	if err != nil {
		return nil, fmt.Errorf("failed to get step: %w", err)
	}
	if step == nil {
		return nil, fmt.Errorf("step not found: %s", stepID)
	}

	// 2. Validate step has content (ai_output or human_edited)
	if step.AIOutput == nil && step.HumanEdited == nil {
		return nil, fmt.Errorf("cannot approve step %d (%s): no content available (generate AI output first)",
			step.StepNumber, step.FrameworkCode)
	}

	// 3. Approve the step
	if err := s.repo.Approve(ctx, stepID); err != nil {
		return nil, fmt.Errorf("failed to approve step: %w", err)
	}

	// 4. Fetch updated step
	approvedStep, err := s.repo.GetByID(ctx, stepID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch approved step: %w", err)
	}

	// 5. Determine if this is the last step
	isComplete := approvedStep.StepNumber == (TotalSteps() - 1)

	// 6. If not last step, fetch next step info
	var nextStep *AnalysisStep
	currentStepNum := approvedStep.StepNumber

	if !isComplete {
		allSteps, err := s.repo.GetByAnalysisID(ctx, approvedStep.AnalysisID)
		if err != nil {
			return nil, fmt.Errorf("failed to get all steps: %w", err)
		}

		for i := range allSteps {
			if allSteps[i].StepNumber == currentStepNum+1 {
				nextStep = &allSteps[i]
				break
			}
		}
	}

	// 7. If last step (synthesis), mark analysis as complete
	if isComplete {
		analysisRecord, err := s.analysisRepo.GetByID(ctx, approvedStep.AnalysisID)
		if err == nil && analysisRecord != nil {
			now := time.Now()
			analysisRecord.Status = string(analysis.StatusCompleted)
			analysisRecord.CompletedAt = &now
			analysisRecord.UpdatedAt = now
			_ = s.analysisRepo.Update(ctx, analysisRecord)
		}
	}

	s.logger.Info().
		Str("step_id", stepID).
		Int("step_number", approvedStep.StepNumber).
		Bool("is_complete", isComplete).
		Msg("Successfully approved step")

	return &ApproveResponse{
		ApprovedStep: approvedStep,
		NextStep:     nextStep,
		IsComplete:   isComplete,
		CurrentStep:  currentStepNum + 1,
	}, nil
}

// GetStepState returns current step + all previous steps as read-only context
func (s *Service) GetStepState(ctx context.Context, analysisID string) (*StepStateResponse, error) {
	s.logger.Debug().
		Str("analysis_id", analysisID).
		Msg("Getting step state")

	// 1. Get all steps for this analysis
	allSteps, err := s.repo.GetByAnalysisID(ctx, analysisID)
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis steps: %w", err)
	}

	if len(allSteps) == 0 {
		return nil, fmt.Errorf("no steps found for analysis: %s", analysisID)
	}

	// 2. Find current step (first non-approved, or last if all approved)
	var currentStep *AnalysisStep
	currentStepIndex := 0

	for i := range allSteps {
		if !allSteps[i].IsApproved() {
			currentStep = &allSteps[i]
			currentStepIndex = i
			break
		}
	}

	// If all approved, current step is the last one
	if currentStep == nil {
		currentStepIndex = len(allSteps) - 1
		currentStep = &allSteps[currentStepIndex]
	}

	// 3. Get framework metadata
	frameworkMeta := GetFrameworkMeta(currentStep.FrameworkCode)

	// 4. Get previous steps (read-only context)
	var previousSteps []AnalysisStep
	if currentStepIndex > 0 {
		previousSteps = allSteps[:currentStepIndex]
	} else {
		previousSteps = []AnalysisStep{}
	}

	return &StepStateResponse{
		AnalysisID:      analysisID,
		CurrentStep:     currentStepIndex,
		TotalSteps:      TotalSteps(),
		CurrentStepData: currentStep,
		PreviousSteps:   previousSteps,
		FrameworkMeta:   frameworkMeta,
	}, nil
}

// GetAnalysisSteps returns all steps for an analysis ordered by step_number
func (s *Service) GetAnalysisSteps(ctx context.Context, analysisID string) ([]AnalysisStep, error) {
	s.logger.Debug().
		Str("analysis_id", analysisID).
		Msg("Getting all analysis steps")

	steps, err := s.repo.GetByAnalysisID(ctx, analysisID)
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis steps: %w", err)
	}

	return steps, nil
}

// buildStepContext builds LLM context for a framework step
// Includes: company_data, challenge_data, macro_context, and all approved previous steps
func (s *Service) buildStepContext(ctx context.Context, analysisRecord *analysis.Analysis, previousSteps []AnalysisStep) (map[string]interface{}, error) {
	s.logger.Debug().
		Str("analysis_id", analysisRecord.ID).
		Int("previous_steps_count", len(previousSteps)).
		Msg("Building step context")

	// 1. Get company data
	var companyID uuid.UUID
	if analysisRecord.CompanyID != nil {
		parsed, err := uuid.Parse(*analysisRecord.CompanyID)
		if err != nil {
			return nil, fmt.Errorf("invalid company_id: %w", err)
		}
		companyID = parsed
	} else {
		return nil, fmt.Errorf("analysis has no company_id")
	}

	companyData, err := s.companyService.GetByID(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get company data: %w", err)
	}
	if companyData == nil {
		return nil, fmt.Errorf("company not found: %s", companyID.String())
	}

	// 2. Get challenge data
	chal, err := s.challengeRepo.GetByID(ctx, analysisRecord.ChallengeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenge: %w", err)
	}
	if chal == nil {
		return nil, fmt.Errorf("challenge not found: %s", analysisRecord.ChallengeID.String())
	}

	// 3. Get macro context (hardcoded)
	macroContext := s.extractMacroContext()

	// 4. Build previous frameworks map (frameworkCode → effectiveOutput)
	previousFrameworks := make(map[string]interface{})
	for _, step := range previousSteps {
		if step.IsApproved() {
			effectiveOutput := step.GetEffectiveOutput()
			if effectiveOutput != nil {
				// Try to parse as JSON for structured data
				var outputData map[string]interface{}
				if err := json.Unmarshal([]byte(*effectiveOutput), &outputData); err == nil {
					previousFrameworks[step.FrameworkCode] = outputData
				} else {
					// If not valid JSON, store as string
					previousFrameworks[step.FrameworkCode] = *effectiveOutput
				}
			}
		}
	}

	// 5. Build final context
	stepContext := map[string]interface{}{
		"company_data":        companyData,
		"challenge_context":   chal.BusinessChallenge,
		"challenge_type":      string(chal.ChallengeType),
		"challenge_category":  string(chal.ChallengeCategory),
		"macro_context":       macroContext,
		"previous_frameworks": previousFrameworks,
	}

	return stepContext, nil
}

// extractMacroContext returns hardcoded Brazilian economic indicators for MVP
// Copied from analysis/workflow.go:1171-1182
func (s *Service) extractMacroContext() map[string]interface{} {
	// Hardcoded Brazilian economic indicators for MVP
	// Last updated: December 2025
	return map[string]interface{}{
		"economic_indicators": map[string]interface{}{
			"interest_rate":  "15.00% a.a.",      // SELIC (BCB) - 05/12/2025
			"inflation_rate": "4.68% (12 meses)", // IPCA (IBGE) - 05/12/2025
			"exchange_rate":  "R$ 5,44/USD",      // Dólar Comercial - 05/12/2025
			"as_of":          "2025-12",          // Reference date
		},
	}
}
