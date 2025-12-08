package wizard

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"backend_v3/config"
	"backend_v3/domain/analysis"
	"backend_v3/llm"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service handles the human-in-the-loop framework wizard
type Service struct {
	repo       analysis.Repository
	llm        *llm.Client
	frameworks map[string]config.FrameworkConfig
	companySvc CompanyServiceInterface
	logger     zerolog.Logger
}

// CompanyServiceInterface defines what wizard needs from company service
// Uses the same AnalysisCompanyData type as the main analysis workflow
type CompanyServiceInterface interface {
	GetByID(ctx context.Context, companyID uuid.UUID) (*analysis.AnalysisCompanyData, error)
}

// NewService creates a new wizard service
func NewService(repo analysis.Repository, llmClient *llm.Client, frameworks map[string]config.FrameworkConfig, logger zerolog.Logger) *Service {
	return &Service{
		repo:       repo,
		llm:        llmClient,
		frameworks: frameworks,
		logger:     logger.With().Str("service", "wizard").Logger(),
	}
}

// SetCompanyService injects the company service for data access
func (s *Service) SetCompanyService(svc CompanyServiceInterface) {
	s.companySvc = svc
}

// State represents the current state of the wizard
type State struct {
	AnalysisID     string            `json:"analysis_id"`
	CurrentStep    int               `json:"current_step"`
	TotalSteps     int               `json:"total_steps"`
	Framework      *FrameworkStep    `json:"framework"`
	StepStatus     string            `json:"step_status"` // pending, generating, generated, approved
	Output         json.RawMessage   `json:"output,omitempty"`
	HumanContext   string            `json:"human_context,omitempty"`
	HumanAnswers   map[string]string `json:"human_answers,omitempty"`
	PreviousSteps  []StepSummary     `json:"previous_steps"`
	IterationCount int               `json:"iteration_count"`
	ErrorMessage   string            `json:"error_message,omitempty"`
}

// StepSummary is a condensed view of a completed step
type StepSummary struct {
	Step          int    `json:"step"`
	FrameworkCode string `json:"framework_code"`
	FrameworkName string `json:"framework_name"`
	Status        string `json:"status"`
	ApprovedAt    string `json:"approved_at,omitempty"`
}

// StartWizard initializes or returns existing wizard state for a company + challenge
func (s *Service) StartWizard(ctx context.Context, companyID string, challengeID string) (*State, error) {
	s.logger.Info().
		Str("company_id", companyID).
		Str("challenge_id", challengeID).
		Msg("Starting wizard")

	// Parse challenge ID
	challengeUUID, err := uuid.Parse(challengeID)
	if err != nil {
		return nil, fmt.Errorf("invalid challenge_id: %w", err)
	}

	// Check if analysis already exists for this challenge
	a, err := s.repo.GetByChallengeID(ctx, challengeUUID)
	if err != nil {
		// Create new analysis with company + challenge
		a = &analysis.Analysis{
			ID:          uuid.New().String(),
			CompanyID:   &companyID,
			ChallengeID: challengeUUID,
			Status:      string(analysis.StatusPending),
			CurrentStep: 0,
			WizardMode:  true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := s.repo.Create(ctx, a); err != nil {
			return nil, fmt.Errorf("failed to create analysis: %w", err)
		}
	}

	return s.GetCurrentState(ctx, a.ID)
}

// GetCurrentState returns the current wizard state
func (s *Service) GetCurrentState(ctx context.Context, analysisID string) (*State, error) {
	a, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return nil, fmt.Errorf("analysis not found: %w", err)
	}

	// Get current step info
	currentStep := a.CurrentStep
	if currentStep >= TotalSteps {
		currentStep = TotalSteps - 1
	}
	framework := GetFrameworkStep(currentStep)

	// Get step data if exists
	step, _ := s.repo.GetAnalysisStep(ctx, analysisID, framework.Code)

	state := &State{
		AnalysisID:  analysisID,
		CurrentStep: currentStep,
		TotalSteps:  TotalSteps,
		Framework:   framework,
		StepStatus:  "pending",
	}

	if step != nil {
		state.StepStatus = step.Status
		state.Output = step.Output
		state.HumanContext = step.HumanContext
		state.IterationCount = step.IterationCount
		if step.HumanAnswers != nil {
			json.Unmarshal(step.HumanAnswers, &state.HumanAnswers)
		}
		if step.ErrorMessage != nil {
			state.ErrorMessage = *step.ErrorMessage
		}
	}

	// Get previous steps summary
	state.PreviousSteps = s.getPreviousStepsSummary(ctx, analysisID, currentStep)

	return state, nil
}

// GenerateStep generates the framework output for the current step
func (s *Service) GenerateStep(ctx context.Context, analysisID string, humanContext string, answers map[string]string) (*State, error) {
	a, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return nil, fmt.Errorf("analysis not found: %w", err)
	}

	currentStep := a.CurrentStep
	framework := GetFrameworkStep(currentStep)
	if framework == nil {
		return nil, fmt.Errorf("invalid step: %d", currentStep)
	}

	// RULE: Cannot go back to previous steps - wizard only moves forward
	if currentStep >= TotalSteps {
		return nil, fmt.Errorf("all steps already completed - wizard is finished")
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Int("step", currentStep).
		Str("framework", framework.Code).
		Msg("Generating framework")

	// Get or create step record
	step, _ := s.repo.GetAnalysisStep(ctx, analysisID, framework.Code)
	if step == nil {
		step = &analysis.AnalysisStep{
			ID:            uuid.New().String(),
			AnalysisID:    analysisID,
			StepNumber:    currentStep,
			FrameworkCode: framework.Code,
			Status:        "generating",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
	} else {
		step.Status = "generating"
		step.IterationCount++
	}

	// Store human context and answers
	step.HumanContext = humanContext
	if answers != nil {
		answersJSON, _ := json.Marshal(answers)
		step.HumanAnswers = answersJSON
	}

	// Save step as generating
	if err := s.repo.SaveAnalysisStep(ctx, step); err != nil {
		return nil, fmt.Errorf("failed to save step: %w", err)
	}

	// Build context from previous steps
	previousContext := s.buildPreviousContext(ctx, analysisID)

	// Generate framework output
	startTime := time.Now()
	output, err := s.runFramework(ctx, a, framework.Code, previousContext, humanContext)
	processingTime := int(time.Since(startTime).Milliseconds())

	if err != nil {
		errorMsg := err.Error()
		step.Status = "failed"
		step.ErrorMessage = &errorMsg
		s.repo.SaveAnalysisStep(ctx, step)
		return nil, fmt.Errorf("framework generation failed: %w", err)
	}

	// Update step with output
	step.Status = "generated"
	step.Output = output
	step.ProcessingTimeMs = processingTime
	now := time.Now()
	step.GeneratedAt = &now
	step.UpdatedAt = now

	if err := s.repo.SaveAnalysisStep(ctx, step); err != nil {
		return nil, fmt.Errorf("failed to save step output: %w", err)
	}

	return s.GetCurrentState(ctx, analysisID)
}

// ApproveStep approves the current step and advances to the next
func (s *Service) ApproveStep(ctx context.Context, analysisID string, userID *string) (*State, error) {
	a, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return nil, fmt.Errorf("analysis not found: %w", err)
	}

	currentStep := a.CurrentStep
	framework := GetFrameworkStep(currentStep)

	// Get step
	step, err := s.repo.GetAnalysisStep(ctx, analysisID, framework.Code)
	if err != nil || step == nil {
		return nil, fmt.Errorf("step not found - generate first")
	}

	if step.Status != "generated" {
		return nil, fmt.Errorf("step must be generated before approval (current: %s)", step.Status)
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Int("step", currentStep).
		Str("framework", framework.Code).
		Msg("Approving step")

	// Mark step as approved
	step.Status = "approved"
	now := time.Now()
	step.ApprovedAt = &now
	if userID != nil {
		uid, err := uuid.Parse(*userID)
		if err != nil {
			s.logger.Warn().Err(err).Str("user_id", *userID).Msg("Invalid user ID format for ApprovedBy - skipping")
		} else {
			step.ApprovedBy = &uid
		}
	}
	step.UpdatedAt = now

	if err := s.repo.SaveAnalysisStep(ctx, step); err != nil {
		return nil, fmt.Errorf("failed to approve step: %w", err)
	}

	// Copy approved output to analysis framework_results
	if a.FrameworkResults == nil {
		a.FrameworkResults = make(map[string]json.RawMessage)
	}
	a.FrameworkResults[framework.Code] = step.Output

	// Advance to next step
	a.CurrentStep++
	a.StepsCompleted = append(a.StepsCompleted, framework.Code)
	a.UpdatedAt = now

	// Check if wizard is complete
	if a.CurrentStep >= TotalSteps {
		s.logger.Info().Str("analysis_id", analysisID).Msg("All steps complete, generating synthesis")
		synthesis, err := s.generateSynthesis(ctx, a)
		if err != nil {
			s.logger.Warn().Err(err).Msg("Synthesis generation failed")
		} else {
			a.FrameworkResults["synthesis"] = synthesis
		}
		a.Status = string(analysis.StatusCompleted)
		a.CompletedAt = &now
	}

	if err := s.repo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("failed to advance analysis: %w", err)
	}

	return s.GetCurrentState(ctx, analysisID)
}

// RefineStep regenerates the current step using the refinement prompt with previous output + feedback
func (s *Service) RefineStep(ctx context.Context, analysisID string, additionalContext string, notes string) (*State, error) {
	a, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return nil, fmt.Errorf("analysis not found: %w", err)
	}

	currentStep := a.CurrentStep
	framework := GetFrameworkStep(currentStep)
	if framework == nil {
		return nil, fmt.Errorf("invalid step: %d", currentStep)
	}

	// Get step - must have been generated first
	step, _ := s.repo.GetAnalysisStep(ctx, analysisID, framework.Code)
	if step == nil {
		return nil, fmt.Errorf("step not found - generate first")
	}

	// Must have previous output to refine
	if step.Output == nil || len(step.Output) == 0 {
		return nil, fmt.Errorf("no previous output to refine - generate first")
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Int("step", currentStep).
		Str("framework", framework.Code).
		Int("iteration", step.IterationCount+1).
		Msg("Refining step with previous output + feedback")

	// Store previous output in history before refinement
	step.PreviousOutputs = appendPreviousOutput(step.PreviousOutputs, step.Output)

	// Mark step as generating
	step.Status = "generating"
	step.IterationCount++
	step.UpdatedAt = time.Now()

	// Store the feedback
	step.HumanContext = additionalContext
	if notes != "" {
		step.RefinementNotes = notes
	}

	if err := s.repo.SaveAnalysisStep(ctx, step); err != nil {
		return nil, fmt.Errorf("failed to save step: %w", err)
	}

	// Run refinement with previous output and feedback
	startTime := time.Now()
	output, err := s.runRefinement(ctx, a, framework.Code, step.Output, additionalContext)
	processingTime := int(time.Since(startTime).Milliseconds())

	if err != nil {
		errorMsg := err.Error()
		step.Status = "failed"
		step.ErrorMessage = &errorMsg
		s.repo.SaveAnalysisStep(ctx, step)
		return nil, fmt.Errorf("refinement failed: %w", err)
	}

	// Update step with refined output
	step.Status = "generated"
	step.Output = output
	step.ProcessingTimeMs = processingTime
	now := time.Now()
	step.GeneratedAt = &now
	step.UpdatedAt = now

	if err := s.repo.SaveAnalysisStep(ctx, step); err != nil {
		return nil, fmt.Errorf("failed to save refined output: %w", err)
	}

	return s.GetCurrentState(ctx, analysisID)
}

// GetWizardSummary returns a summary of all approved frameworks
func (s *Service) GetWizardSummary(ctx context.Context, analysisID string) (map[string]json.RawMessage, error) {
	a, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return nil, fmt.Errorf("analysis not found: %w", err)
	}

	return a.FrameworkResults, nil
}

// GenerateAllProgress tracks progress of bulk generation
type GenerateAllProgress struct {
	CurrentStep    int               `json:"current_step"`
	TotalSteps     int               `json:"total_steps"`
	CompletedSteps []string          `json:"completed_steps"`
	FailedSteps    []string          `json:"failed_steps,omitempty"`
	Status         string            `json:"status"` // processing, completed, failed
	ErrorMessage   string            `json:"error_message,omitempty"`
	ProcessingTime int64             `json:"processing_time_ms"`
	Results        map[string]string `json:"results,omitempty"` // framework_code -> status
}

// GenerateAll generates all frameworks sequentially and auto-approves each step
// This is an admin-only function that bypasses the wizard human-in-the-loop
func (s *Service) GenerateAll(ctx context.Context, analysisID string) (*GenerateAllProgress, error) {
	startTime := time.Now()

	a, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return nil, fmt.Errorf("analysis not found: %w", err)
	}

	// Initialize progress
	progress := &GenerateAllProgress{
		CurrentStep:    a.CurrentStep,
		TotalSteps:     TotalSteps,
		CompletedSteps: []string{},
		FailedSteps:    []string{},
		Status:         "processing",
		Results:        make(map[string]string),
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Int("starting_step", a.CurrentStep).
		Int("total_steps", TotalSteps).
		Msg("Starting bulk generation of all frameworks")

	// Initialize framework results if nil
	if a.FrameworkResults == nil {
		a.FrameworkResults = make(map[string]json.RawMessage)
	}

	// Process each remaining step
	for step := a.CurrentStep; step < TotalSteps; step++ {
		framework := GetFrameworkStep(step)
		if framework == nil {
			continue
		}

		progress.CurrentStep = step

		s.logger.Info().
			Str("analysis_id", analysisID).
			Int("step", step).
			Str("framework", framework.Code).
			Msg("Generating framework")

		// Build context from previous steps
		previousContext := s.buildPreviousContext(ctx, analysisID)

		// Generate framework output
		stepStartTime := time.Now()
		output, err := s.runFramework(ctx, a, framework.Code, previousContext, "")
		stepProcessingTime := int(time.Since(stepStartTime).Milliseconds())

		if err != nil {
			s.logger.Error().
				Err(err).
				Str("framework", framework.Code).
				Msg("Framework generation failed")

			progress.FailedSteps = append(progress.FailedSteps, framework.Code)
			progress.Results[framework.Code] = "failed: " + err.Error()

			// Continue to next step instead of failing completely
			continue
		}

		// Create/update step record
		now := time.Now()
		step_record := &analysis.AnalysisStep{
			ID:               uuid.New().String(),
			AnalysisID:       analysisID,
			StepNumber:       step,
			FrameworkCode:    framework.Code,
			Status:           "approved",
			Output:           output,
			ProcessingTimeMs: stepProcessingTime,
			GeneratedAt:      &now,
			ApprovedAt:       &now,
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		if err := s.repo.SaveAnalysisStep(ctx, step_record); err != nil {
			s.logger.Error().
				Err(err).
				Str("framework", framework.Code).
				Msg("Failed to save step record")
		}

		// Update analysis with approved output
		a.FrameworkResults[framework.Code] = output
		a.CurrentStep = step + 1
		a.StepsCompleted = append(a.StepsCompleted, framework.Code)
		a.UpdatedAt = now

		// Save progress after each step
		if err := s.repo.Update(ctx, a); err != nil {
			s.logger.Error().
				Err(err).
				Str("framework", framework.Code).
				Msg("Failed to update analysis")
		}

		progress.CompletedSteps = append(progress.CompletedSteps, framework.Code)
		progress.Results[framework.Code] = "completed"

		s.logger.Info().
			Str("framework", framework.Code).
			Int("processing_time_ms", stepProcessingTime).
			Msg("Framework generated and approved")
	}

	// Generate synthesis if all steps completed
	if a.CurrentStep >= TotalSteps && len(progress.FailedSteps) == 0 {
		s.logger.Info().Str("analysis_id", analysisID).Msg("All steps complete, generating synthesis")

		synthesis, err := s.generateSynthesis(ctx, a)
		if err != nil {
			s.logger.Warn().Err(err).Msg("Synthesis generation failed")
			progress.Results["synthesis"] = "failed: " + err.Error()
		} else {
			a.FrameworkResults["synthesis"] = synthesis
			progress.Results["synthesis"] = "completed"
			progress.CompletedSteps = append(progress.CompletedSteps, "synthesis")
		}

		// Mark analysis as completed
		now := time.Now()
		a.Status = string(analysis.StatusCompleted)
		a.CompletedAt = &now
		a.UpdatedAt = now

		if err := s.repo.Update(ctx, a); err != nil {
			s.logger.Error().Err(err).Msg("Failed to mark analysis as completed")
		}

		progress.Status = "completed"
	} else if len(progress.FailedSteps) > 0 {
		progress.Status = "completed_with_errors"
		progress.ErrorMessage = fmt.Sprintf("%d frameworks failed", len(progress.FailedSteps))
	} else {
		progress.Status = "completed"
	}

	progress.ProcessingTime = time.Since(startTime).Milliseconds()

	s.logger.Info().
		Str("analysis_id", analysisID).
		Int("completed", len(progress.CompletedSteps)).
		Int("failed", len(progress.FailedSteps)).
		Int64("total_time_ms", progress.ProcessingTime).
		Msg("Bulk generation complete")

	return progress, nil
}

// --- Helper methods ---

func (s *Service) getPreviousStepsSummary(ctx context.Context, analysisID string, currentStep int) []StepSummary {
	var summaries []StepSummary

	for i := 0; i < currentStep; i++ {
		fw := GetFrameworkStep(i)
		if fw == nil {
			continue
		}

		summary := StepSummary{
			Step:          i,
			FrameworkCode: fw.Code,
			FrameworkName: fw.Name,
			Status:        "pending",
		}

		step, _ := s.repo.GetAnalysisStep(ctx, analysisID, fw.Code)
		if step != nil {
			summary.Status = step.Status
			if step.ApprovedAt != nil {
				summary.ApprovedAt = step.ApprovedAt.Format(time.RFC3339)
			}
		}

		summaries = append(summaries, summary)
	}

	return summaries
}

func (s *Service) buildPreviousContext(ctx context.Context, analysisID string) map[string]interface{} {
	context := make(map[string]interface{})

	a, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return context
	}

	// Include all approved framework results as context for next steps
	for code, output := range a.FrameworkResults {
		var data interface{}
		if err := json.Unmarshal(output, &data); err == nil {
			context[code] = data
		}
	}

	return context
}

func (s *Service) runFramework(ctx context.Context, a *analysis.Analysis, frameworkCode string, previousContext map[string]interface{}, humanContext string) (json.RawMessage, error) {
	// Get company data for the analysis
	var companyData *analysis.AnalysisCompanyData
	if s.companySvc != nil && a.CompanyID != nil {
		companyUUID, err := uuid.Parse(*a.CompanyID)
		if err == nil {
			companyData, _ = s.companySvc.GetByID(ctx, companyUUID)
		}
	}

	// Hardcoded macro context for MVP (updated 2024-12)
	macroContext := map[string]interface{}{
		"selic": map[string]interface{}{
			"value": "11.25",
			"unit":  "% a.a.",
			"name":  "Taxa SELIC",
		},
		"ipca": map[string]interface{}{
			"value": "4.76",
			"unit":  "% (12 meses)",
			"name":  "IPCA",
		},
		"usd_brl": map[string]interface{}{
			"value": "6.05",
			"unit":  "BRL",
			"name":  "USD/BRL",
		},
	}

	// Build data for prompt
	data := map[string]interface{}{
		"previous_frameworks": previousContext,
		"human_context":       humanContext,
	}

	// Add company data if available
	if companyData != nil {
		data["COMPANY_DATA"] = companyData
	}

	// Add hardcoded macro context
	data["MACRO_CONTEXT"] = macroContext

	// For frameworks that need prior results, inject them
	s.injectFrameworkDependencies(frameworkCode, previousContext, data)

	// Get framework config
	fwConfig, ok := s.frameworks[frameworkCode]
	if !ok {
		fwConfig = s.frameworks["primary"]
	}

	// Get prompt for framework
	prompt := s.getFrameworkPrompt(frameworkCode)

	s.logger.Debug().
		Str("framework", frameworkCode).
		Interface("data_keys", getMapKeys(data)).
		Msg("Running framework with data")

	// Generate structured output
	var result interface{}
	opts := llm.NewGenerationOptions(fwConfig)
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, prompt, data, &result)
	if err != nil {
		return nil, err
	}

	// Convert to JSON
	output, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return output, nil
}

func (s *Service) runRefinement(ctx context.Context, a *analysis.Analysis, frameworkCode string, previousOutput json.RawMessage, humanFeedback string) (json.RawMessage, error) {
	// Get company data
	var companyData *analysis.AnalysisCompanyData
	if s.companySvc != nil && a.CompanyID != nil {
		companyUUID, err := uuid.Parse(*a.CompanyID)
		if err == nil {
			companyData, _ = s.companySvc.GetByID(ctx, companyUUID)
		}
	}

	// Build context from previous approved steps
	previousContext := s.buildPreviousContext(ctx, a.ID)

	// Build data for refinement prompt
	data := map[string]interface{}{
		"PREVIOUS_OUTPUT": string(previousOutput),
		"HUMAN_FEEDBACK":  humanFeedback,
	}

	if companyData != nil {
		data["COMPANY_DATA"] = companyData
	}

	// Add framework-specific context based on dependencies
	frameworkContext := s.buildFrameworkSpecificContext(frameworkCode, previousContext)
	if len(frameworkContext) > 0 {
		data["FRAMEWORK_SPECIFIC_CONTEXT"] = frameworkContext
	}

	// Get framework config
	fwConfig, ok := s.frameworks[frameworkCode]
	if !ok {
		fwConfig = s.frameworks["primary"]
	}

	s.logger.Debug().
		Str("framework", frameworkCode).
		Int("previous_output_len", len(previousOutput)).
		Int("feedback_len", len(humanFeedback)).
		Msg("Running refinement prompt")

	// Generate refined output
	var result interface{}
	opts := llm.NewGenerationOptions(fwConfig)
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, llm.FrameworkRefinementPrompt, data, &result)
	if err != nil {
		return nil, err
	}

	output, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal refined output: %w", err)
	}

	return output, nil
}

func (s *Service) injectFrameworkDependencies(frameworkCode string, previousContext map[string]interface{}, data map[string]interface{}) {
	switch frameworkCode {
	case "swot":
		if pestel, ok := previousContext["pestel"]; ok {
			data["PESTEL_INSIGHTS"] = pestel
		}
		if porter, ok := previousContext["porter"]; ok {
			data["PORTER_INSIGHTS"] = porter
		}
	case "blue_ocean":
		if porter, ok := previousContext["porter"]; ok {
			data["PORTER_INSIGHTS"] = porter
		}
	case "growth_hacking":
		if swot, ok := previousContext["swot"]; ok {
			data["SWOT_SUMMARY"] = swot
			if swotMap, ok := swot.(map[string]interface{}); ok {
				data["SWOT_WEAKNESSES"] = swotMap["weaknesses"]
				data["SWOT_OPPORTUNITIES"] = swotMap["opportunities"]
			}
		}
		if tamSamSom, ok := previousContext["tam_sam_som"]; ok {
			data["MARKET_SCALE"] = tamSamSom
		}
	case "scenarios":
		if pestel, ok := previousContext["pestel"]; ok {
			data["PESTEL_INSIGHTS"] = pestel
		}
	case "decision_matrix":
		if scenarios, ok := previousContext["scenarios"]; ok {
			data["SCENARIO_INSIGHTS"] = scenarios
		}
	case "okrs":
		if blueOcean, ok := previousContext["blue_ocean"]; ok {
			data["BLUE_OCEAN_INSIGHTS"] = blueOcean
		}
		if decisionMatrix, ok := previousContext["decision_matrix"]; ok {
			data["DECISION_MATRIX_RECOMMENDATIONS"] = decisionMatrix
		}
	case "bsc":
		if blueOcean, ok := previousContext["blue_ocean"]; ok {
			data["BLUE_OCEAN_INSIGHTS"] = blueOcean
		}
	}
}

func (s *Service) buildFrameworkSpecificContext(frameworkCode string, previousContext map[string]interface{}) map[string]interface{} {
	context := make(map[string]interface{})

	switch frameworkCode {
	case "swot":
		if pestel, ok := previousContext["pestel"]; ok {
			context["PESTEL_INSIGHTS"] = pestel
		}
		if porter, ok := previousContext["porter"]; ok {
			context["PORTER_INSIGHTS"] = porter
		}
	case "blue_ocean":
		if porter, ok := previousContext["porter"]; ok {
			context["PORTER_INSIGHTS"] = porter
		}
	case "growth_hacking":
		if swot, ok := previousContext["swot"]; ok {
			context["SWOT_SUMMARY"] = swot
		}
		if tamSamSom, ok := previousContext["tam_sam_som"]; ok {
			context["MARKET_SCALE"] = tamSamSom
		}
	case "scenarios":
		if pestel, ok := previousContext["pestel"]; ok {
			context["PESTEL_INSIGHTS"] = pestel
		}
	case "decision_matrix":
		if scenarios, ok := previousContext["scenarios"]; ok {
			context["SCENARIO_INSIGHTS"] = scenarios
		}
	case "okrs":
		if blueOcean, ok := previousContext["blue_ocean"]; ok {
			context["BLUE_OCEAN_INSIGHTS"] = blueOcean
		}
		if decisionMatrix, ok := previousContext["decision_matrix"]; ok {
			context["DECISION_MATRIX_RECOMMENDATIONS"] = decisionMatrix
		}
	case "bsc":
		if blueOcean, ok := previousContext["blue_ocean"]; ok {
			context["BLUE_OCEAN_INSIGHTS"] = blueOcean
		}
	}

	return context
}

func (s *Service) getFrameworkPrompt(code string) string {
	switch code {
	case "challenge_refinement":
		return llm.ChallengeRefinementPrompt
	case "pestel":
		return llm.FrameworkPESTELPrompt
	case "porter":
		return llm.FrameworkPorterPrompt
	case "benchmarking":
		return llm.FrameworkBenchmarkingPrompt
	case "swot":
		return llm.FrameworkSWOTPrompt
	case "tam_sam_som":
		return llm.FrameworkTamSamSomPrompt
	case "blue_ocean":
		return llm.FrameworkBlueOceanPrompt
	case "growth_hacking":
		return llm.FrameworkGrowthHackingPrompt
	case "scenarios":
		return llm.FrameworkScenariosPrompt
	case "decision_matrix":
		return llm.FrameworkDecisionMatrixPrompt
	case "okrs":
		return llm.FrameworkOKRsPrompt
	case "bsc":
		return llm.FrameworkBSCPrompt
	default:
		return ""
	}
}

func (s *Service) generateSynthesis(ctx context.Context, a *analysis.Analysis) (json.RawMessage, error) {
	// Build summaries from all frameworks
	summaries := make(map[string]string)
	for code, output := range a.FrameworkResults {
		var data map[string]interface{}
		if err := json.Unmarshal(output, &data); err == nil {
			if summary, ok := data["summary"].(string); ok {
				summaries[code] = summary
			}
		}
	}

	data := map[string]interface{}{
		"all_framework_summaries": summaries,
	}

	var synthesis analysis.AnalysisSynthesis
	opts := llm.NewGenerationOptions(s.frameworks["synthesis"])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, llm.SynthesisPrompt, data, &synthesis)
	if err != nil {
		return nil, err
	}

	return json.Marshal(synthesis)
}

// appendPreviousOutput adds the current output to the history array
func appendPreviousOutput(history json.RawMessage, output json.RawMessage) json.RawMessage {
	var outputs []json.RawMessage
	if history != nil && len(history) > 0 {
		json.Unmarshal(history, &outputs)
	}
	outputs = append(outputs, output)

	// Keep last 5 iterations to prevent unbounded growth
	if len(outputs) > 5 {
		outputs = outputs[len(outputs)-5:]
	}

	result, _ := json.Marshal(outputs)
	return result
}

// getMapKeys returns the keys of a map for debugging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
