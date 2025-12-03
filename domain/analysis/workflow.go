package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"backend_v3/llm"

	"github.com/google/uuid"
)

// RunAnalysis executes the "Strategic Cascade".
func (s *Service) RunAnalysis(ctx context.Context, submissionID, enrichmentID string, enrichmentData map[string]interface{}) (*Analysis, error) {
	startTime := time.Now()
	s.logger.Info().Str("sub_id", submissionID).Msg("Starting Strategic Cascade Analysis")

	// 1. FETCH SUBMISSION DATA
	submissionUUID, err := uuid.Parse(submissionID)
	if err != nil {
		s.logger.Error().Err(err).Str("sub_id", submissionID).Msg("Invalid submission ID format")
		return nil, err
	}

	submission, err := s.submissionRepo.GetByID(ctx, submissionUUID)
	if err != nil {
		s.logger.Error().Err(err).Str("sub_id", submissionID).Msg("Failed to fetch submission data")
		return nil, err
	}

	// Convert submission to map for template injection
	submissionData := submissionToMap(submission)

	s.logger.Info().
		Str("company_name", submission.CompanyName).
		Str("business_challenge", submission.BusinessChallenge).
		Msg("Submission data loaded successfully")

	// 2. SETUP CONTEXT
	knowledge := &ContextContainer{
		SubmissionID:   submissionID,
		SubmissionData: submissionData,
		EnrichmentData: enrichmentData,
	}
	analysis, err := s.startAnalysisRecord(ctx, submissionID, enrichmentID)
	if err != nil {
		return nil, err
	}

	// ========================================================================
	// LAYER 1: THE ENVIRONMENT (Macro + Industry)
	// ========================================================================
	s.runLayer("Layer 1: Environment", func(wg *sync.WaitGroup) {
		s.exec(wg, func() {
			var err error
			knowledge.PESTEL, err = s.runPESTEL(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ PESTEL failed")
			}
		})
		s.exec(wg, func() {
			var err error
			knowledge.Porter, err = s.runPorter(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ Porter failed")
			}
		})
		s.exec(wg, func() {
			var err error
			knowledge.TamSamSom, err = s.runTamSamSom(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ TAM-SAM-SOM failed")
			}
		})
	})
	s.logger.Debug().Str("analysis_id", analysis.ID).Msg("Layer 2: Starting Positioning analysis")
	s.saveCheckpoint(ctx, analysis, knowledge, string(StatusPending))

	// ========================================================================
	// LAYER 2: POSITIONING (Internal Fit)
	// ========================================================================
	s.runLayer("Layer 2: Positioning", func(wg *sync.WaitGroup) {
		s.exec(wg, func() { knowledge.SWOT, _ = s.runSWOT(ctx, knowledge) })
		s.exec(wg, func() { knowledge.Benchmarking, _ = s.runBenchmarking(ctx, knowledge) })
	})
	s.logger.Debug().Str("analysis_id", analysis.ID).Msg("Layer 3: Starting Strategy analysis")
	s.saveCheckpoint(ctx, analysis, knowledge, string(StatusPending))

	// ========================================================================
	// LAYER 3: STRATEGY (Direction)
	// ========================================================================
	s.runLayer("Layer 3: Strategy", func(wg *sync.WaitGroup) {
		s.exec(wg, func() { knowledge.BlueOcean, _ = s.runBlueOcean(ctx, knowledge) })
		s.exec(wg, func() { knowledge.GrowthHacking, _ = s.runGrowthHacking(ctx, knowledge) })
		s.exec(wg, func() { knowledge.Scenarios, _ = s.runScenarios(ctx, knowledge) })
	})
	s.logger.Debug().Str("analysis_id", analysis.ID).Msg("Layer 3.5: Starting Decision Making analysis")
	s.saveCheckpoint(ctx, analysis, knowledge, string(StatusPending))

	// ========================================================================
	// LAYER 3.5: DECISION MAKING (Priority Recommendations)
	// CRITICAL: Decision Matrix MUST run before OKRs so OKRs can align with recommendations
	// ========================================================================
	s.runLayer("Layer 3.5: Decision Making", func(wg *sync.WaitGroup) {
		s.exec(wg, func() {
			var err error
			knowledge.DecisionMatrix, err = s.runDecisionMatrix(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ DecisionMatrix failed")
			}
		})
	})
	s.logger.Debug().Str("analysis_id", analysis.ID).Msg("Layer 4: Starting Execution analysis")
	s.saveCheckpoint(ctx, analysis, knowledge, string(StatusPending))

	// ========================================================================
	// LAYER 4: EXECUTION (Roadmap)
	// OKRs now have access to Decision Matrix recommendations for alignment
	// ========================================================================
	s.runLayer("Layer 4: Execution", func(wg *sync.WaitGroup) {
		s.exec(wg, func() {
			var err error
			knowledge.OKRs, err = s.runOKRs(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ OKRs failed")
			}
		})
		s.exec(wg, func() {
			var err error
			knowledge.BSC, err = s.runBSC(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ BSC failed")
			}
		})
	})
	s.logger.Debug().Str("analysis_id", analysis.ID).Msg("Starting final synthesis")
	s.saveCheckpoint(ctx, analysis, knowledge, string(StatusPending)) // Still pending until completed

	// ========================================================================
	// CONTENT VALIDATION (Enforce PDF Layout Constraints)
	// ========================================================================
	// Trim arrays to max counts, ensuring uniform page sizing
	validator := NewContentValidator(s.logger)
	validator.ValidateAndNormalize(analysis)

	// ========================================================================
	// FRAMEWORK COMPLETENESS VALIDATION
	// ========================================================================
	// Check if critical frameworks have data before proceeding to synthesis
	emptyFrameworks := s.validateFrameworkCompleteness(knowledge)
	if len(emptyFrameworks) > 0 {
		s.logger.Warn().
			Strs("empty_frameworks", emptyFrameworks).
			Int("empty_count", len(emptyFrameworks)).
			Msg("⚠️ Some frameworks returned empty data - analysis may be incomplete")
	}

	// ========================================================================
	// FINAL SYNTHESIS (The Senior Partner)
	// ========================================================================
	// Uses the Premium Model (s.synthesisModel)
	analysis.Synthesis, _ = s.runSynthesis(ctx, knowledge)

	// ========================================================================
	// FINAL VALIDATION BEFORE MARKING COMPLETE
	// ========================================================================
	// Verify we have minimum required data for a valid report
	criticalMissing := s.validateCriticalFrameworks(analysis)
	if len(criticalMissing) > 0 {
		errorMsg := fmt.Sprintf("critical frameworks failed: %v", criticalMissing)
		s.logger.Error().
			Strs("critical_missing", criticalMissing).
			Str("analysis_id", analysis.ID).
			Msg("❌ Analysis incomplete - marking as failed")
		analysis.Status = string(StatusFailed)
		analysis.ErrorMessage = &errorMsg
		analysis.ProcessingTimeMs = time.Since(startTime).Milliseconds()
		s.repo.Update(ctx, analysis)
		return analysis, fmt.Errorf("analysis incomplete: %s", errorMsg)
	}

	// FINISH
	s.markAsComplete(ctx, analysis, startTime)
	return analysis, nil
}

// =================================================================================
// MECHANICS
// =================================================================================

func (s *Service) runLayer(name string, tasks func(*sync.WaitGroup)) {
	s.logger.Info().Msg("Starting " + name)
	var wg sync.WaitGroup
	tasks(&wg)
	wg.Wait()
}

func (s *Service) exec(wg *sync.WaitGroup, task func()) {
	wg.Add(1)
	go func() { defer wg.Done(); task() }()
}

func (s *Service) startAnalysisRecord(ctx context.Context, subID, enrichID string) (*Analysis, error) {
	s.logger.Info().Str("submission_id", subID).Str("enrichment_id", enrichID).Msg("startAnalysisRecord: BEGIN")

	existing, err := s.repo.GetBySubmissionID(ctx, subID)
	s.logger.Info().
		Str("submission_id", subID).
		Bool("found", err == nil && existing != nil).
		Interface("error", err).
		Msg("startAnalysisRecord: GetBySubmissionID result")

	if err == nil && existing != nil {
		s.logger.Info().
			Str("submission_id", subID).
			Str("existing_analysis_id", existing.ID).
			Str("existing_status", existing.Status).
			Msg("startAnalysisRecord: Found existing analysis record")

		switch existing.Status {
		case string(StatusCompleted):
			// Allow re-running analysis by resetting the existing record
			s.logger.Info().
				Str("submission_id", subID).
				Str("analysis_id", existing.ID).
				Str("old_status", existing.Status).
				Msg("startAnalysisRecord: Resetting existing analysis to pending for re-run")

			existing.Status = string(StatusPending)
			existing.EnrichmentID = enrichID
			existing.UpdatedAt = time.Now()
			existing.CompletedAt = nil
			existing.ProcessingTimeMs = 0

			// Clear previous analysis results for fresh run
			existing.PESTEL = PESTELAnalysis{}
			existing.Porter = PorterAnalysis{}
			existing.TamSamSom = TamSamSomAnalysis{}
			existing.SWOT = SWOTAnalysis{}
			existing.Benchmarking = BenchmarkingAnalysis{}
			existing.BlueOcean = BlueOceanAnalysis{}
			existing.GrowthHacking = GrowthHackingAnalysis{}
			existing.Scenarios = ScenarioAnalysis{}
			existing.OKRs = OKRAnalysis{}
			existing.BSC = BalancedScorecardAnalysis{}
			existing.DecisionMatrix = DecisionMatrixAnalysis{}
			existing.Synthesis = AnalysisSynthesis{}

			if err := s.repo.Update(ctx, existing); err != nil {
				s.logger.Error().Err(err).Str("analysis_id", existing.ID).Msg("Failed to reset analysis for re-run")
				return nil, fmt.Errorf("failed to reset analysis for re-run: %w", err)
			}

			s.logger.Info().Str("analysis_id", existing.ID).Msg("Analysis reset successfully, starting fresh run")
			return existing, nil
		default:
			// Pending state: reuse record (worker will continue/restart processing)
			// If there was an error previously, it will be overwritten
			s.logger.Debug().Str("analysis_id", existing.ID).Msg("Reusing existing pending analysis record")
			return existing, nil
		}
	} else if err != nil && !strings.Contains(err.Error(), "not found") {
		s.logger.Error().Err(err).Str("submission_id", subID).Msg("startAnalysisRecord: Error fetching existing analysis (not 'not found')")
		return nil, err
	}

	s.logger.Info().Str("submission_id", subID).Msg("startAnalysisRecord: No existing analysis found, creating new record")

	// If not found or other retrieval error, create a new record
	a := &Analysis{
		ID:           uuid.New().String(),
		SubmissionID: subID,
		EnrichmentID: enrichID,
		Status:       string(StatusPending),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	s.logger.Info().
		Str("analysis_id", a.ID).
		Str("submission_id", subID).
		Str("enrichment_id", enrichID).
		Msg("startAnalysisRecord: Calling repo.Create")

	if err := s.repo.Create(ctx, a); err != nil {
		s.logger.Error().Err(err).Str("analysis_id", a.ID).Msg("startAnalysisRecord: Create FAILED")
		return nil, fmt.Errorf("failed to create analysis: %w", err)
	}

	s.logger.Info().Str("analysis_id", a.ID).Msg("startAnalysisRecord: Create SUCCESS - analysis record created")
	return a, nil
}

func (s *Service) saveCheckpoint(ctx context.Context, a *Analysis, k *ContextContainer, nextStatus string) {
	// CRITICAL FIX: Wrap checkpoint saves in database transaction
	// Prevents partial updates and race conditions during concurrent modifications
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		s.logger.Error().Err(err).Str("analysis_id", a.ID).Msg("Failed to begin checkpoint transaction")
		return
	}
	defer tx.Rollback() // Rollback if commit not called

	if k.PESTEL != nil {
		a.PESTEL = *k.PESTEL
	}
	if k.Porter != nil {
		a.Porter = *k.Porter
	}
	if k.TamSamSom != nil {
		a.TamSamSom = *k.TamSamSom
	}
	if k.SWOT != nil {
		a.SWOT = *k.SWOT
	}
	if k.Benchmarking != nil {
		a.Benchmarking = *k.Benchmarking
	}
	if k.BlueOcean != nil {
		a.BlueOcean = *k.BlueOcean
	}
	if k.GrowthHacking != nil {
		a.GrowthHacking = *k.GrowthHacking
	}
	if k.Scenarios != nil {
		a.Scenarios = *k.Scenarios
	}
	if k.OKRs != nil {
		a.OKRs = *k.OKRs
	}
	if k.BSC != nil {
		a.BSC = *k.BSC
	}
	if k.DecisionMatrix != nil {
		a.DecisionMatrix = *k.DecisionMatrix
	}

	a.Status = nextStatus

	// Update within transaction
	if err := s.repo.UpdateWithTx(ctx, tx, a); err != nil {
		s.logger.Error().Err(err).Str("analysis_id", a.ID).Msg("Failed to update analysis in transaction")
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error().Err(err).Str("analysis_id", a.ID).Msg("Failed to commit checkpoint transaction")
		return
	}

	s.logger.Info().
		Str("analysis_id", a.ID).
		Str("status", nextStatus).
		Msg("Checkpoint saved successfully")
}

func (s *Service) markAsComplete(ctx context.Context, a *Analysis, startTime time.Time) {
	a.Status = string(StatusCompleted)
	a.ProcessingTimeMs = time.Since(startTime).Milliseconds()
	now := time.Now()
	a.CompletedAt = &now
	s.repo.Update(ctx, a)
	s.logger.Info().Msg("Analysis Workflow Completed")
}

// --- Context-Aware Runners ---
// NEW: Uses framework-specific models with heterogeneous routing

// withDataPriority prepends the data priority instruction to any framework prompt
// This ensures all frameworks follow the user-data > AI-data priority rule
func withDataPriority(prompt string) string {
	return llm.DataPriorityInstruction + "\n" + prompt
}

func (s *Service) runPESTEL(ctx context.Context, k *ContextContainer) (*PESTELAnalysis, error) {
	var res PESTELAnalysis
	data := map[string]interface{}{
		"company_data":    k.SubmissionData,
		"enrichment_data": k.EnrichmentData,
		"macro_context":   s.extractMacroContext(k.EnrichmentData),
	}
	opts := llm.NewGenerationOptions(s.frameworks["pestel"])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, withDataPriority(llm.FrameworkPESTELPrompt), data, &res)
	return &res, err
}

func (s *Service) runPorter(ctx context.Context, k *ContextContainer) (*PorterAnalysis, error) {
	var res PorterAnalysis
	data := map[string]interface{}{
		"company_data":    k.SubmissionData,
		"enrichment_data": k.EnrichmentData,
		"macro_context":   s.extractMacroContext(k.EnrichmentData),
	}
	opts := llm.NewGenerationOptions(s.frameworks["porter"])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, withDataPriority(llm.FrameworkPorterPrompt), data, &res)
	return &res, err
}

func (s *Service) runTamSamSom(ctx context.Context, k *ContextContainer) (*TamSamSomAnalysis, error) {
	var res TamSamSomAnalysis
	data := map[string]interface{}{
		"company_data":    k.SubmissionData,
		"enrichment_data": k.EnrichmentData,
		"macro_context":   s.extractMacroContext(k.EnrichmentData),
	}
	opts := llm.NewGenerationOptions(s.frameworks["tam_sam_som"])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, withDataPriority(llm.FrameworkTamSamSomPrompt), data, &res)
	return &res, err
}

func (s *Service) runSWOT(ctx context.Context, k *ContextContainer) (*SWOTAnalysis, error) {
	var res SWOTAnalysis

	// SAFETY: Nil checks prevent panic if Layer 1 failed
	pestelSummary := ""
	if k.PESTEL != nil {
		pestelSummary = k.PESTEL.Summary
	}
	porterSummary := ""
	if k.Porter != nil {
		porterSummary = k.Porter.Summary
	}

	data := map[string]interface{}{
		"company_data":    k.SubmissionData,
		"enrichment_data": k.EnrichmentData,
		"pestel_insights": pestelSummary,
		"porter_insights": porterSummary,
	}
	opts := llm.NewGenerationOptions(s.frameworks["swot"])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, withDataPriority(llm.FrameworkSWOTPrompt), data, &res)
	return &res, err
}

func (s *Service) runBenchmarking(ctx context.Context, k *ContextContainer) (*BenchmarkingAnalysis, error) {
	var res BenchmarkingAnalysis

	// SAFETY: Nil check prevents panic if Layer 1 failed
	marketScale := ""
	if k.TamSamSom != nil {
		marketScale = k.TamSamSom.Summary
	}

	data := map[string]interface{}{
		"company_data":    k.SubmissionData,
		"enrichment_data": k.EnrichmentData,
		"market_scale":    marketScale,
	}
	opts := llm.NewGenerationOptions(s.frameworks["benchmarking"])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, withDataPriority(llm.FrameworkBenchmarkingPrompt), data, &res)
	return &res, err
}

func (s *Service) runBlueOcean(ctx context.Context, k *ContextContainer) (*BlueOceanAnalysis, error) {
	var res BlueOceanAnalysis

	// SAFETY: Nil check prevents panic if Layer 1 failed
	porterSummary := ""
	if k.Porter != nil {
		porterSummary = k.Porter.Summary
	}

	data := map[string]interface{}{
		"company_data":    k.SubmissionData,
		"enrichment_data": k.EnrichmentData,
		"porter_insights": porterSummary,
	}
	opts := llm.NewGenerationOptions(s.frameworks["blue_ocean"])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, withDataPriority(llm.FrameworkBlueOceanPrompt), data, &res)
	return &res, err
}

func (s *Service) runGrowthHacking(ctx context.Context, k *ContextContainer) (*GrowthHackingAnalysis, error) {
	var res GrowthHackingAnalysis

	// SAFETY: Nil checks for dependencies from Layer 1 and Layer 2
	// Growth Hacking needs SWOT insights for targeted experiments
	swotSummary := ""
	var swotWeaknesses []SWOTItem
	var swotOpportunities []SWOTItem
	if k.SWOT != nil {
		swotSummary = k.SWOT.Summary
		swotWeaknesses = k.SWOT.Weaknesses
		swotOpportunities = k.SWOT.Opportunities
	}

	// TAM-SAM-SOM for market scale targeting
	marketScale := ""
	if k.TamSamSom != nil {
		marketScale = k.TamSamSom.Summary
	}

	// DEBUG: Log what we're passing to GrowthHacking
	s.logger.Debug().
		Str("swot_summary", swotSummary).
		Int("weaknesses_count", len(swotWeaknesses)).
		Int("opportunities_count", len(swotOpportunities)).
		Str("market_scale", marketScale).
		Msg("🔍 DEBUG GrowthHacking Input Data")

	data := map[string]interface{}{
		"company_data":       k.SubmissionData,
		"enrichment_data":    k.EnrichmentData,
		"swot_summary":       swotSummary,
		"swot_weaknesses":    swotWeaknesses,
		"swot_opportunities": swotOpportunities,
		"market_scale":       marketScale,
	}
	opts := llm.NewGenerationOptions(s.frameworks["growth_hacking"])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, withDataPriority(llm.FrameworkGrowthHackingPrompt), data, &res)
	return &res, err
}

func (s *Service) runScenarios(ctx context.Context, k *ContextContainer) (*ScenarioAnalysis, error) {
	var res ScenarioAnalysis

	// SAFETY: Nil check prevents panic if Layer 1 failed
	pestelSummary := ""
	if k.PESTEL != nil {
		pestelSummary = k.PESTEL.Summary
	}

	data := map[string]interface{}{
		"company_data":    k.SubmissionData,
		"enrichment_data": k.EnrichmentData,
		"pestel_insights": pestelSummary,
		"macro_context":   s.extractMacroContext(k.EnrichmentData),
	}
	opts := llm.NewGenerationOptions(s.frameworks["scenarios"])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, withDataPriority(llm.FrameworkScenariosPrompt), data, &res)
	return &res, err
}

func (s *Service) runOKRs(ctx context.Context, k *ContextContainer) (*OKRAnalysis, error) {
	var res OKRAnalysis

	// SAFETY: Nil checks prevent panic if previous layers failed
	blueOceanSummary := ""
	if k.BlueOcean != nil {
		blueOceanSummary = k.BlueOcean.Summary
	}
	var swotWeaknesses []SWOTItem
	if k.SWOT != nil {
		swotWeaknesses = k.SWOT.Weaknesses
	}

	// NEW: Extract Decision Matrix recommendations for OKR alignment
	// OKRs should directly implement the priority recommendations from Decision Matrix
	var decisionMatrixRecommendations []PriorityRecommendation
	if k.DecisionMatrix != nil && len(k.DecisionMatrix.PriorityRecommendations) > 0 {
		decisionMatrixRecommendations = k.DecisionMatrix.PriorityRecommendations
	}

	// DEBUG: Log what we're passing to OKRs
	s.logger.Debug().
		Str("blue_ocean_summary", blueOceanSummary).
		Int("swot_weaknesses_count", len(swotWeaknesses)).
		Int("decision_matrix_recommendations_count", len(decisionMatrixRecommendations)).
		Msg("🔍 DEBUG OKRs Input Data")

	data := map[string]interface{}{
		"company_data":                    k.SubmissionData,
		"enrichment_data":                 k.EnrichmentData,
		"blue_ocean_insights":             blueOceanSummary,
		"swot_weaknesses":                 swotWeaknesses,
		"decision_matrix_recommendations": decisionMatrixRecommendations,
	}
	opts := llm.NewGenerationOptions(s.frameworks["okrs"])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, withDataPriority(llm.FrameworkOKRsPrompt), data, &res)

	// DEBUG: Log what we got back
	s.logger.Debug().
		Int("plan_90_days_count", len(res.Plan90Days)).
		Int("quarters_count_legacy", len(res.Quarters)).
		Str("total_investment", res.TotalInvestment).
		Str("summary", res.Summary).
		Msg("🔍 DEBUG OKRs Output Data")

	// FALLBACK: Generate summary from plan_90_days if LLM returned empty summary
	if res.Summary == "" && len(res.Plan90Days) > 0 {
		s.logger.Warn().Msg("⚠️ OKRs summary empty - generating fallback from plan_90_days")
		res.Summary = s.generateOKRsSummaryFallback(&res)
	}

	return &res, err
}

func (s *Service) runBSC(ctx context.Context, k *ContextContainer) (*BalancedScorecardAnalysis, error) {
	var res BalancedScorecardAnalysis

	// SAFETY: Nil check prevents panic if previous layer failed
	blueOceanSummary := ""
	if k.BlueOcean != nil {
		blueOceanSummary = k.BlueOcean.Summary
	}

	// DEBUG: Log what we're passing to BSC
	s.logger.Debug().
		Str("blue_ocean_summary", blueOceanSummary).
		Msg("🔍 DEBUG BSC Input Data")

	data := map[string]interface{}{
		"company_data":        k.SubmissionData,
		"enrichment_data":     k.EnrichmentData,
		"blue_ocean_insights": blueOceanSummary,
	}
	opts := llm.NewGenerationOptions(s.frameworks["bsc"])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, withDataPriority(llm.FrameworkBSCPrompt), data, &res)

	// DEBUG: Log what we got back
	s.logger.Debug().
		Int("financial_count", len(res.Financial)).
		Int("customer_count", len(res.Customer)).
		Int("internal_count", len(res.Internal)).
		Int("learning_count", len(res.LearningGrowth)).
		Str("summary", res.Summary).
		Msg("🔍 DEBUG BSC Output Data")

	return &res, err
}

func (s *Service) runDecisionMatrix(ctx context.Context, k *ContextContainer) (*DecisionMatrixAnalysis, error) {
	var res DecisionMatrixAnalysis

	// SAFETY: Nil check prevents panic if previous layer failed
	scenarioSummary := ""
	if k.Scenarios != nil {
		scenarioSummary = k.Scenarios.Summary
	}

	// DEBUG: Log what we're passing to DecisionMatrix
	s.logger.Debug().
		Str("scenario_summary", scenarioSummary).
		Msg("🔍 DEBUG DecisionMatrix Input Data")

	data := map[string]interface{}{
		"company_data":      k.SubmissionData,
		"enrichment_data":   k.EnrichmentData,
		"scenario_insights": scenarioSummary,
	}
	opts := llm.NewGenerationOptions(s.frameworks["decision_matrix"])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, withDataPriority(llm.FrameworkDecisionMatrixPrompt), data, &res)

	// DEBUG: Log what we got back
	s.logger.Debug().
		Int("alternatives_count", len(res.Alternatives)).
		Int("criteria_count", len(res.Criteria)).
		Str("recommendation", res.FinalRecommendation).
		Str("summary", res.Summary).
		Msg("🔍 DEBUG DecisionMatrix Output Data")

	return &res, err
}

func (s *Service) runSynthesis(ctx context.Context, k *ContextContainer) (AnalysisSynthesis, error) {
	var res AnalysisSynthesis

	// SAFETY: Build summaries map with nil checks to prevent panic if any framework failed
	summaries := map[string]string{
		"pestel":     safeGetSummary(k.PESTEL),
		"porter":     safeGetSummary(k.Porter),
		"swot":       safeGetSummary(k.SWOT),
		"blue_ocean": safeGetSummary(k.BlueOcean),
		"okrs":       safeGetSummary(k.OKRs),
		"scenarios":  safeGetSummary(k.Scenarios),
		"growth":     safeGetSummary(k.GrowthHacking),
	}
	data := map[string]interface{}{
		"company_data":            k.SubmissionData,
		"enrichment_data":         k.EnrichmentData,
		"all_framework_summaries": summaries,
	}
	// NEW: Uses framework-specific synthesis config (Claude 3.5 Sonnet with T=0.4)
	opts := llm.NewGenerationOptions(s.frameworks["synthesis"])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, llm.SynthesisPrompt, data, &res)
	return res, err
}

// Summarizable interface for type-safe nil checking
type Summarizable interface {
	GetSummary() string
}

// safeGetSummary extracts summary from any framework result, returning empty string if nil
func safeGetSummary(s interface{}) string {
	if s == nil {
		return ""
	}
	// Use type switch to handle all framework types
	switch v := s.(type) {
	case *PESTELAnalysis:
		if v == nil {
			return ""
		}
		return v.Summary
	case *PorterAnalysis:
		if v == nil {
			return ""
		}
		return v.Summary
	case *SWOTAnalysis:
		if v == nil {
			return ""
		}
		return v.Summary
	case *BlueOceanAnalysis:
		if v == nil {
			return ""
		}
		return v.Summary
	case *OKRAnalysis:
		if v == nil {
			return ""
		}
		return v.Summary
	case *ScenarioAnalysis:
		if v == nil {
			return ""
		}
		return v.Summary
	case *GrowthHackingAnalysis:
		if v == nil {
			return ""
		}
		return v.Summary
	case *TamSamSomAnalysis:
		if v == nil {
			return ""
		}
		return v.Summary
	case *BenchmarkingAnalysis:
		if v == nil {
			return ""
		}
		return v.Summary
	case *BalancedScorecardAnalysis:
		if v == nil {
			return ""
		}
		return v.Summary
	case *DecisionMatrixAnalysis:
		if v == nil {
			return ""
		}
		return v.Summary
	default:
		return ""
	}
}

// submissionToMap converts a SubmissionData struct to a map for template injection
func submissionToMap(s *SubmissionData) map[string]interface{} {
	result := map[string]interface{}{
		"company_name":       s.CompanyName,
		"business_challenge": s.BusinessChallenge,
	}

	// Add optional fields only if present
	if s.CompanyWebsite != nil {
		result["company_website"] = *s.CompanyWebsite
	}
	if s.CompanyIndustry != nil {
		result["company_industry"] = *s.CompanyIndustry
	}
	if s.CompanySize != nil {
		result["company_size"] = *s.CompanySize
	}
	if s.CompanyLocation != nil {
		result["company_location"] = *s.CompanyLocation
	}
	if s.TargetMarket != nil {
		result["target_market"] = *s.TargetMarket
	}
	if s.AnnualRevenueMin != nil && s.AnnualRevenueMax != nil {
		result["annual_revenue_range"] = map[string]float64{
			"min": *s.AnnualRevenueMin,
			"max": *s.AnnualRevenueMax,
		}
	}
	if s.FundingStage != nil {
		result["funding_stage"] = *s.FundingStage
	}

	return result
}

// extractMacroContext fetches macro data directly from the macroeconomics database
// IMPORTANT: This now fetches fresh data from DB, NOT from enrichment output
// This ensures analysis always has the most current macro data
// Falls back to enrichment data for backwards compatibility with old enrichments
func (s *Service) extractMacroContext(enrichmentData map[string]interface{}) map[string]interface{} {
	ctx := context.Background()

	// PRIMARY: Fetch macro data directly from macroeconomics service (DB)
	if s.macroService != nil {
		snapshot, err := s.macroService.GetLatestSnapshot(ctx)
		if err == nil && snapshot.HasData() {
			s.logger.Debug().
				Int("indicator_count", len(snapshot.Indicators)).
				Msg("Using macro data directly from DB (preferred)")

			// Convert snapshot to map for prompt injection
			result := map[string]interface{}{
				"economic_indicators": s.formatSnapshotForPrompt(snapshot),
			}
			return result
		}
		if err != nil {
			s.logger.Warn().Err(err).Msg("Failed to fetch macro data from DB, falling back to enrichment data")
		} else {
			s.logger.Warn().Msg("No macro data in DB, falling back to enrichment data")
		}
	} else {
		s.logger.Debug().Msg("MacroService not configured, falling back to enrichment data")
	}

	// FALLBACK: Try to extract from enrichment data (backwards compatibility)
	if enrichmentData == nil {
		s.logger.Debug().Msg("No enrichment data provided, returning empty macro_context")
		return map[string]interface{}{}
	}

	// Check if macro_context exists in enrichment data
	if macroCtx, ok := enrichmentData["macro_context"]; ok && macroCtx != nil {
		if macroMap, ok := macroCtx.(map[string]interface{}); ok {
			s.logger.Debug().Msg("Macro-context extracted from enrichment data (legacy)")
			return macroMap
		}
		s.logger.Warn().Msg("macro_context exists but is not a map, returning empty")
	}

	// Return empty map for backward compatibility
	return map[string]interface{}{}
}

// formatSnapshotForPrompt converts macro snapshot to a map suitable for prompt injection
func (s *Service) formatSnapshotForPrompt(snapshot *MacroSnapshot) map[string]interface{} {
	result := map[string]interface{}{}

	for code, ind := range snapshot.Indicators {
		if ind == nil {
			continue
		}
		// Format based on indicator type
		switch code {
		case "selic", "selic_meta":
			result["interest_rate"] = fmt.Sprintf("%.2f%% a.a.", ind.Value)
		case "ipca", "ipca_12m":
			result["inflation_rate"] = fmt.Sprintf("%.2f%% (12 meses)", ind.Value)
		case "usd_brl":
			result["exchange_rate"] = fmt.Sprintf("R$ %.2f/USD", ind.Value)
		case "pib":
			result["gdp_growth"] = fmt.Sprintf("%.2f%%", ind.Value)
		default:
			// Include other indicators by code
			result[code] = fmt.Sprintf("%.2f %s", ind.Value, ind.Unit)
		}
	}

	return result
}

// =================================================================================
// FRAMEWORK VALIDATION
// =================================================================================

// validateFrameworkCompleteness checks all frameworks in the context container and returns names of empty ones
func (s *Service) validateFrameworkCompleteness(k *ContextContainer) []string {
	var empty []string

	if k.PESTEL == nil || k.PESTEL.Summary == "" {
		empty = append(empty, "pestel")
	}
	if k.Porter == nil || k.Porter.Summary == "" {
		empty = append(empty, "porter")
	}
	if k.TamSamSom == nil || k.TamSamSom.Summary == "" {
		empty = append(empty, "tam_sam_som")
	}
	if k.SWOT == nil || k.SWOT.Summary == "" {
		empty = append(empty, "swot")
	}
	if k.Benchmarking == nil || k.Benchmarking.Summary == "" {
		empty = append(empty, "benchmarking")
	}
	if k.BlueOcean == nil || k.BlueOcean.Summary == "" {
		empty = append(empty, "blue_ocean")
	}
	if k.GrowthHacking == nil || k.GrowthHacking.Summary == "" {
		empty = append(empty, "growth_hacking")
	}
	if k.Scenarios == nil || k.Scenarios.Summary == "" {
		empty = append(empty, "scenarios")
	}
	if k.OKRs == nil || k.OKRs.Summary == "" {
		empty = append(empty, "okrs")
	}
	if k.BSC == nil || k.BSC.Summary == "" {
		empty = append(empty, "bsc")
	}
	if k.DecisionMatrix == nil || k.DecisionMatrix.Summary == "" {
		empty = append(empty, "decision_matrix")
	}

	return empty
}

// validateCriticalFrameworks checks the final Analysis struct for critical missing data
// Critical frameworks are those required for a meaningful report
// Returns list of missing critical frameworks
func (s *Service) validateCriticalFrameworks(a *Analysis) []string {
	var missing []string

	// CRITICAL: These must have data for a valid report
	// Layer 1 - Environment (at least PESTEL or Porter must work)
	if a.PESTEL.Summary == "" && a.Porter.Summary == "" {
		missing = append(missing, "environment_layer (pestel+porter both empty)")
	}

	// Layer 2 - Positioning (SWOT is critical)
	if a.SWOT.Summary == "" && len(a.SWOT.Strengths) == 0 && len(a.SWOT.Weaknesses) == 0 {
		missing = append(missing, "swot")
	}

	// Layer 3 - Strategy (at least one strategy framework)
	if a.BlueOcean.Summary == "" && a.GrowthHacking.Summary == "" && a.Scenarios.Summary == "" {
		missing = append(missing, "strategy_layer (blue_ocean+growth_hacking+scenarios all empty)")
	}

	// Layer 4 - Execution (OKRs or BSC)
	if a.OKRs.Summary == "" && a.BSC.Summary == "" {
		missing = append(missing, "execution_layer (okrs+bsc both empty)")
	}

	// Synthesis must exist
	if a.Synthesis.ExecutiveSummary == "" && a.Synthesis.CentralChallenge == "" {
		missing = append(missing, "synthesis")
	}

	return missing
}

// generateOKRsSummaryFallback creates a summary from plan_90_days when LLM returns empty summary
func (s *Service) generateOKRsSummaryFallback(okrs *OKRAnalysis) string {
	if len(okrs.Plan90Days) == 0 {
		return ""
	}

	// Build summary from monthly objectives
	var objectives []string
	for _, month := range okrs.Plan90Days {
		if month.Objective != "" {
			objectives = append(objectives, month.Objective)
		}
	}

	if len(objectives) == 0 {
		return "Plano de 90 dias com marcos mensais para execução estratégica."
	}

	// Create a concise summary (max ~200 chars as per prompt)
	summary := fmt.Sprintf("90 dias: %s", objectives[0])
	if len(summary) > 180 {
		summary = summary[:177] + "..."
	}

	s.logger.Info().
		Str("fallback_summary", summary).
		Int("objectives_used", len(objectives)).
		Msg("✅ OKRs fallback summary generated")

	return summary
}

// =================================================================================
// DYNAMIC FRAMEWORK EXECUTION
// =================================================================================

// RunAnalysisDynamic executes frameworks dynamically based on database configuration
// This is the new flexible approach that will replace hardcoded layer execution
func (s *Service) RunAnalysisDynamic(
	ctx context.Context,
	analysisID string,
	enrichmentData map[string]interface{},
	requestedCodes []string, // Optional: nil or empty means all active frameworks
) error {
	if s.frameworkService == nil {
		return fmt.Errorf("framework service not configured")
	}

	// Determine which frameworks to run
	var codes []string
	if len(requestedCodes) > 0 {
		codes = requestedCodes
	} else {
		// Get all active framework codes
		activeFrameworks, err := s.frameworkService.ListActive(ctx)
		if err != nil {
			return fmt.Errorf("failed to get active frameworks: %w", err)
		}
		for _, fw := range activeFrameworks {
			codes = append(codes, fw.Code)
		}
	}

	// Get execution plan (resolves dependencies, orders correctly)
	frameworks, err := s.frameworkService.GetExecutionPlan(ctx, codes)
	if err != nil {
		return fmt.Errorf("failed to get execution plan: %w", err)
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Int("framework_count", len(frameworks)).
		Strs("codes", codes).
		Msg("Starting dynamic analysis execution")

	results := make(map[string]json.RawMessage)

	// Execute each framework in dependency order
	for _, fw := range frameworks {
		s.logger.Info().
			Str("analysis_id", analysisID).
			Str("framework", fw.Code).
			Msg("Executing framework")

		result, err := s.executeFrameworkDynamic(ctx, fw, enrichmentData, results)
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("analysis_id", analysisID).
				Str("framework", fw.Code).
				Msg("Framework execution failed")
			return fmt.Errorf("framework %s failed: %w", fw.Code, err)
		}

		results[fw.Code] = result

		// Save incrementally to database
		if err := s.repo.SetFrameworkResult(ctx, analysisID, fw.Code, result); err != nil {
			s.logger.Warn().
				Err(err).
				Str("analysis_id", analysisID).
				Str("framework", fw.Code).
				Msg("Failed to save framework result incrementally")
			// Continue anyway - we'll save everything at the end
		}
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Int("completed", len(results)).
		Msg("Dynamic analysis execution completed")

	return nil
}

// executeFrameworkDynamic executes a single framework using database configuration
func (s *Service) executeFrameworkDynamic(
	ctx context.Context,
	fw *Framework,
	enrichmentData map[string]interface{},
	previousResults map[string]json.RawMessage,
) (json.RawMessage, error) {
	// Execute using existing LLM infrastructure
	// Create a context container for compatibility with existing methods
	// We need to populate it with previous results for dependent frameworks
	k := &ContextContainer{
		EnrichmentData: enrichmentData,
	}

	// Populate context container with previous results if needed
	// This allows dependent frameworks to access results of their dependencies
	if err := s.populateContextFromResults(k, previousResults); err != nil {
		s.logger.Warn().Err(err).Str("framework", fw.Code).Msg("Failed to populate context from previous results")
	}

	// Map framework code to existing execution methods
	// This bridges the dynamic system with existing hardcoded implementations
	var result interface{}
	var err error

	switch fw.Code {
	case "pestel":
		result, err = s.runPESTEL(ctx, k)
	case "porter":
		result, err = s.runPorter(ctx, k)
	case "tam_sam_som":
		result, err = s.runTamSamSom(ctx, k)
	case "swot":
		result, err = s.runSWOT(ctx, k)
	case "benchmarking":
		result, err = s.runBenchmarking(ctx, k)
	case "blue_ocean":
		result, err = s.runBlueOcean(ctx, k)
	case "growth_hacking":
		result, err = s.runGrowthHacking(ctx, k)
	case "scenarios":
		result, err = s.runScenarios(ctx, k)
	case "okrs":
		result, err = s.runOKRs(ctx, k)
	case "bsc":
		result, err = s.runBSC(ctx, k)
	case "decision_matrix":
		result, err = s.runDecisionMatrix(ctx, k)
	case "synthesis":
		result, err = s.runSynthesis(ctx, k)
	default:
		return nil, fmt.Errorf("unknown framework code: %s", fw.Code)
	}

	if err != nil {
		return nil, fmt.Errorf("LLM call failed for %s: %w", fw.Code, err)
	}

	// Convert result to JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result for %s: %w", fw.Code, err)
	}

	return resultJSON, nil
}

// populateContextFromResults populates the ContextContainer with results from previously executed frameworks
// This enables dependent frameworks to access the outputs of their dependencies
func (s *Service) populateContextFromResults(k *ContextContainer, results map[string]json.RawMessage) error {
	// Unmarshal each result into its appropriate field in the context container
	for code, resultJSON := range results {
		switch code {
		case "pestel":
			var pestel PESTELAnalysis
			if err := json.Unmarshal(resultJSON, &pestel); err == nil {
				k.PESTEL = &pestel
			}
		case "porter":
			var porter PorterAnalysis
			if err := json.Unmarshal(resultJSON, &porter); err == nil {
				k.Porter = &porter
			}
		case "tam_sam_som":
			var tamSamSom TamSamSomAnalysis
			if err := json.Unmarshal(resultJSON, &tamSamSom); err == nil {
				k.TamSamSom = &tamSamSom
			}
		case "swot":
			var swot SWOTAnalysis
			if err := json.Unmarshal(resultJSON, &swot); err == nil {
				k.SWOT = &swot
			}
		case "benchmarking":
			var bench BenchmarkingAnalysis
			if err := json.Unmarshal(resultJSON, &bench); err == nil {
				k.Benchmarking = &bench
			}
		case "blue_ocean":
			var blueOcean BlueOceanAnalysis
			if err := json.Unmarshal(resultJSON, &blueOcean); err == nil {
				k.BlueOcean = &blueOcean
			}
		case "growth_hacking":
			var growth GrowthHackingAnalysis
			if err := json.Unmarshal(resultJSON, &growth); err == nil {
				k.GrowthHacking = &growth
			}
		case "scenarios":
			var scenarios ScenarioAnalysis
			if err := json.Unmarshal(resultJSON, &scenarios); err == nil {
				k.Scenarios = &scenarios
			}
		case "okrs":
			var okrs OKRAnalysis
			if err := json.Unmarshal(resultJSON, &okrs); err == nil {
				k.OKRs = &okrs
			}
		case "bsc":
			var bsc BalancedScorecardAnalysis
			if err := json.Unmarshal(resultJSON, &bsc); err == nil {
				k.BSC = &bsc
			}
		case "decision_matrix":
			var dm DecisionMatrixAnalysis
			if err := json.Unmarshal(resultJSON, &dm); err == nil {
				k.DecisionMatrix = &dm
			}
		}
	}
	return nil
}
