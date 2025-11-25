package analysis

import (
	"context"
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
	s.logger.Debug().Str("analysis_id", analysis.ID).Msg("Layer 4: Starting Execution analysis")
	s.saveCheckpoint(ctx, analysis, knowledge, string(StatusPending))

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
	s.logger.Debug().Str("analysis_id", analysis.ID).Msg("Starting final synthesis")
	s.saveCheckpoint(ctx, analysis, knowledge, string(StatusPending)) // Still pending until completed

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
		case string(StatusCompleted), string(StatusApproved), string(StatusSent):
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

	if err := s.repo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("failed to create analysis: %w", err)
	}

	s.logger.Debug().Str("analysis_id", a.ID).Msg("Layer 1: Starting Environment analysis")
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

func (s *Service) runPESTEL(ctx context.Context, k *ContextContainer) (*PESTELAnalysis, error) {
	var res PESTELAnalysis
	data := map[string]interface{}{
		"company_data":    k.SubmissionData,
		"enrichment_data": k.EnrichmentData,
		"macro_context":   s.extractMacroContext(k.EnrichmentData),
	}
	opts := llm.NewGenerationOptions(s.frameworks["pestel"])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, llm.FrameworkPESTELPrompt, data, &res)
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
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, llm.FrameworkPorterPrompt, data, &res)
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
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, llm.FrameworkTamSamSomPrompt, data, &res)
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
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, llm.FrameworkSWOTPrompt, data, &res)
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
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, llm.FrameworkBenchmarkingPrompt, data, &res)
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
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, llm.FrameworkBlueOceanPrompt, data, &res)
	return &res, err
}

func (s *Service) runGrowthHacking(ctx context.Context, k *ContextContainer) (*GrowthHackingAnalysis, error) {
	var res GrowthHackingAnalysis
	data := map[string]interface{}{
		"company_data":    k.SubmissionData,
		"enrichment_data": k.EnrichmentData,
	}
	opts := llm.NewGenerationOptions(s.frameworks["growth_hacking"])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, llm.FrameworkGrowthHackingPrompt, data, &res)
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
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, llm.FrameworkScenariosPrompt, data, &res)
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

	// DEBUG: Log what we're passing to OKRs
	s.logger.Debug().
		Str("blue_ocean_summary", blueOceanSummary).
		Int("swot_weaknesses_count", len(swotWeaknesses)).
		Msg("🔍 DEBUG OKRs Input Data")

	data := map[string]interface{}{
		"company_data":        k.SubmissionData,
		"enrichment_data":     k.EnrichmentData,
		"blue_ocean_insights": blueOceanSummary,
		"swot_weaknesses":     swotWeaknesses,
	}
	opts := llm.NewGenerationOptions(s.frameworks["okrs"])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, llm.FrameworkOKRsPrompt, data, &res)

	// DEBUG: Log what we got back
	s.logger.Debug().
		Int("quarters_count", len(res.Quarters)).
		Str("summary", res.Summary).
		Msg("🔍 DEBUG OKRs Output Data")

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
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, llm.FrameworkBSCPrompt, data, &res)

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
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, llm.FrameworkDecisionMatrixPrompt, data, &res)

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

// extractMacroContext safely extracts macro_context from enrichment data
// Returns the macro_context if present, or an empty map for backward compatibility
func (s *Service) extractMacroContext(enrichmentData map[string]interface{}) map[string]interface{} {
	if enrichmentData == nil {
		s.logger.Debug().Msg("No enrichment data provided, returning empty macro_context")
		return map[string]interface{}{}
	}

	// Check if macro_context exists in enrichment data
	if macroCtx, ok := enrichmentData["macro_context"]; ok && macroCtx != nil {
		// If it's already a map, return it
		if macroMap, ok := macroCtx.(map[string]interface{}); ok {
			s.logger.Debug().Msg("Macro-context found and extracted successfully")
			return macroMap
		}
		s.logger.Warn().Msg("macro_context exists but is not a map, returning empty")
	} else {
		s.logger.Debug().Msg("No macro_context in enrichment data (backward compatibility mode)")
	}

	// Return empty map for backward compatibility with old enrichments
	return map[string]interface{}{}
}
