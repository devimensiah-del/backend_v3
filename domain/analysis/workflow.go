package analysis

import (
	"context"
	"sync"
	"time"

	"backend_v3/llm"

	"github.com/google/uuid"
)

// RunAnalysis executes the "Strategic Cascade".
func (s *Service) RunAnalysis(ctx context.Context, submissionID, enrichmentID string, enrichmentData map[string]interface{}) (*Analysis, error) {
	startTime := time.Now()
	s.logger.Info().Str("sub_id", submissionID).Msg("Starting Strategic Cascade Analysis")

	// 1. SETUP
	// We use the models injected into the service (safe for testing)
	knowledge := &ContextContainer{SubmissionID: submissionID, EnrichmentData: enrichmentData}
	analysis := s.createAnalysisRecord(ctx, submissionID, enrichmentID)

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
	s.saveCheckpoint(ctx, analysis, knowledge, "processing_layer_2")

	// ========================================================================
	// LAYER 2: POSITIONING (Internal Fit)
	// ========================================================================
	s.runLayer("Layer 2: Positioning", func(wg *sync.WaitGroup) {
		s.exec(wg, func() { knowledge.SWOT, _ = s.runSWOT(ctx, knowledge) })
		s.exec(wg, func() { knowledge.Benchmarking, _ = s.runBenchmarking(ctx, knowledge) })
	})
	s.saveCheckpoint(ctx, analysis, knowledge, "processing_layer_3")

	// ========================================================================
	// LAYER 3: STRATEGY (Direction)
	// ========================================================================
	s.runLayer("Layer 3: Strategy", func(wg *sync.WaitGroup) {
		s.exec(wg, func() { knowledge.BlueOcean, _ = s.runBlueOcean(ctx, knowledge) })
		s.exec(wg, func() { knowledge.GrowthHacking, _ = s.runGrowthHacking(ctx, knowledge) })
		s.exec(wg, func() { knowledge.Scenarios, _ = s.runScenarios(ctx, knowledge) })
	})
	s.saveCheckpoint(ctx, analysis, knowledge, "processing_layer_4")

	// ========================================================================
	// LAYER 4: EXECUTION (Roadmap)
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
		s.exec(wg, func() {
			var err error
			knowledge.DecisionMatrix, err = s.runDecisionMatrix(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ DecisionMatrix failed")
			}
		})
	})
	s.saveCheckpoint(ctx, analysis, knowledge, "processing_synthesis")

	// ========================================================================
	// CONTENT VALIDATION (Enforce PDF Layout Constraints)
	// ========================================================================
	// Trim arrays to max counts, ensuring uniform page sizing
	validator := NewContentValidator(s.logger)
	validator.ValidateAndNormalize(analysis)

	// ========================================================================
	// FINAL SYNTHESIS (The Senior Partner)
	// ========================================================================
	// Uses the Premium Model (s.synthesisModel)
	analysis.Synthesis, _ = s.runSynthesis(ctx, knowledge)

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

func (s *Service) createAnalysisRecord(ctx context.Context, subID, enrichID string) *Analysis {
	a := &Analysis{
		ID:           uuid.New().String(),
		SubmissionID: subID,
		EnrichmentID: enrichID,
		Status:       "processing_layer_1",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	s.repo.Create(ctx, a)
	return a
}

func (s *Service) saveCheckpoint(ctx context.Context, a *Analysis, k *ContextContainer, nextStatus string) {
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
	s.repo.Update(ctx, a)
}

func (s *Service) markAsComplete(ctx context.Context, a *Analysis, startTime time.Time) {
	a.Status = "completed"
	a.ProcessingTimeMs = time.Since(startTime).Milliseconds()
	now := time.Now()
	a.CompletedAt = &now
	s.repo.Update(ctx, a)
	s.logger.Info().Msg("Analysis Workflow Completed")
}

// --- Context-Aware Runners ---
// Uses s.analystModel for all frameworks

func (s *Service) runPESTEL(ctx context.Context, k *ContextContainer) (*PESTELAnalysis, error) {
	var res PESTELAnalysis
	data := map[string]interface{}{"enrichment_data": k.EnrichmentData}
	err := s.llm.GenerateStructured(ctx, s.analystModel, llm.FrameworkPESTELPrompt, data, &res)
	return &res, err
}

func (s *Service) runPorter(ctx context.Context, k *ContextContainer) (*PorterAnalysis, error) {
	var res PorterAnalysis
	data := map[string]interface{}{"enrichment_data": k.EnrichmentData}
	err := s.llm.GenerateStructured(ctx, s.analystModel, llm.FrameworkPorterPrompt, data, &res)
	return &res, err
}

func (s *Service) runTamSamSom(ctx context.Context, k *ContextContainer) (*TamSamSomAnalysis, error) {
	var res TamSamSomAnalysis
	data := map[string]interface{}{"enrichment_data": k.EnrichmentData}
	err := s.llm.GenerateStructured(ctx, s.analystModel, llm.FrameworkTamSamSomPrompt, data, &res)
	return &res, err
}

func (s *Service) runSWOT(ctx context.Context, k *ContextContainer) (*SWOTAnalysis, error) {
	var res SWOTAnalysis
	data := map[string]interface{}{
		"enrichment_data": k.EnrichmentData,
		"pestel_insights": k.PESTEL.Summary,
		"porter_insights": k.Porter.Summary,
	}
	err := s.llm.GenerateStructured(ctx, s.analystModel, llm.FrameworkSWOTPrompt, data, &res)
	return &res, err
}

func (s *Service) runBenchmarking(ctx context.Context, k *ContextContainer) (*BenchmarkingAnalysis, error) {
	var res BenchmarkingAnalysis
	data := map[string]interface{}{"enrichment_data": k.EnrichmentData, "market_scale": k.TamSamSom.Summary}
	err := s.llm.GenerateStructured(ctx, s.analystModel, llm.FrameworkBenchmarkingPrompt, data, &res)
	return &res, err
}

func (s *Service) runBlueOcean(ctx context.Context, k *ContextContainer) (*BlueOceanAnalysis, error) {
	var res BlueOceanAnalysis
	data := map[string]interface{}{"enrichment_data": k.EnrichmentData, "porter_insights": k.Porter.Summary}
	err := s.llm.GenerateStructured(ctx, s.analystModel, llm.FrameworkBlueOceanPrompt, data, &res)
	return &res, err
}

func (s *Service) runGrowthHacking(ctx context.Context, k *ContextContainer) (*GrowthHackingAnalysis, error) {
	var res GrowthHackingAnalysis
	data := map[string]interface{}{"enrichment_data": k.EnrichmentData}
	err := s.llm.GenerateStructured(ctx, s.analystModel, llm.FrameworkGrowthHackingPrompt, data, &res)
	return &res, err
}

func (s *Service) runScenarios(ctx context.Context, k *ContextContainer) (*ScenarioAnalysis, error) {
	var res ScenarioAnalysis
	data := map[string]interface{}{"enrichment_data": k.EnrichmentData, "pestel_insights": k.PESTEL.Summary}
	err := s.llm.GenerateStructured(ctx, s.analystModel, llm.FrameworkScenariosPrompt, data, &res)
	return &res, err
}

func (s *Service) runOKRs(ctx context.Context, k *ContextContainer) (*OKRAnalysis, error) {
	var res OKRAnalysis

	// DEBUG: Log what we're passing to OKRs
	s.logger.Debug().
		Str("blue_ocean_summary", k.BlueOcean.Summary).
		Int("swot_weaknesses_count", len(k.SWOT.Weaknesses)).
		Msg("🔍 DEBUG OKRs Input Data")

	data := map[string]interface{}{
		"enrichment_data":     k.EnrichmentData,
		"blue_ocean_insights": k.BlueOcean.Summary,
		"swot_weaknesses":     k.SWOT.Weaknesses,
	}
	err := s.llm.GenerateStructured(ctx, s.analystModel, llm.FrameworkOKRsPrompt, data, &res)

	// DEBUG: Log what we got back
	s.logger.Debug().
		Int("objectives_count", len(res.Objectives)).
		Str("summary", res.Summary).
		Msg("🔍 DEBUG OKRs Output Data")

	return &res, err
}

func (s *Service) runBSC(ctx context.Context, k *ContextContainer) (*BalancedScorecardAnalysis, error) {
	var res BalancedScorecardAnalysis

	// DEBUG: Log what we're passing to BSC
	s.logger.Debug().
		Str("blue_ocean_summary", k.BlueOcean.Summary).
		Msg("🔍 DEBUG BSC Input Data")

	data := map[string]interface{}{"enrichment_data": k.EnrichmentData, "blue_ocean_insights": k.BlueOcean.Summary}
	err := s.llm.GenerateStructured(ctx, s.analystModel, llm.FrameworkBSCPrompt, data, &res)

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

	// DEBUG: Log what we're passing to DecisionMatrix
	s.logger.Debug().
		Str("scenario_summary", k.Scenarios.Summary).
		Msg("🔍 DEBUG DecisionMatrix Input Data")

	data := map[string]interface{}{"enrichment_data": k.EnrichmentData, "scenario_insights": k.Scenarios.Summary}
	err := s.llm.GenerateStructured(ctx, s.analystModel, llm.FrameworkDecisionMatrixPrompt, data, &res)

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
	summaries := map[string]string{
		"pestel":     k.PESTEL.Summary,
		"porter":     k.Porter.Summary,
		"swot":       k.SWOT.Summary,
		"blue_ocean": k.BlueOcean.Summary,
		"okrs":       k.OKRs.Summary,
		"scenarios":  k.Scenarios.Summary,
		"growth":     k.GrowthHacking.Summary,
	}
	data := map[string]interface{}{
		"enrichment_data":         k.EnrichmentData,
		"all_framework_summaries": summaries,
	}
	// Uses s.synthesisModel (Sonnet 4.5)
	err := s.llm.GenerateStructured(ctx, s.synthesisModel, llm.SynthesisPrompt, data, &res)
	return res, err
}
