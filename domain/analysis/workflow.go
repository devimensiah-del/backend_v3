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

// RunAnalysis executes the "Strategic Cascade" for a specific challenge.
//
// EXECUTION MODE: Direct Analysis Mode
// - Maximum parallel execution respecting layer dependencies
// - Frameworks run in parallel WITHIN each layer
// - Layers execute sequentially (Layer 1 → Layer 2 → Layer 3 → Layer 3.5 → Layer 4 → Synthesis)
//
// For human-in-the-loop workflows, use wizard mode (WizardService):
// - Fully sequential step-by-step execution with human approval at each framework
// - "Add context → regenerate" refinement pattern
// - Full audit trail via versioning
//
// Parameters:
// - submissionID: Historical reference to the originating submission
// - companyID: The company being analyzed
// - challengeID: REQUIRED - The specific business challenge this analysis addresses
func (s *Service) RunAnalysis(ctx context.Context, submissionID, companyID string, challengeID uuid.UUID) (*Analysis, error) {
	startTime := time.Now()
	s.logger.Info().
		Str("sub_id", submissionID).
		Str("challenge_id", challengeID.String()).
		Str("execution_mode", "DIRECT_ANALYSIS").
		Msg("Starting Strategic Cascade Analysis - Direct Analysis Mode (Parallel within layers)")

	// Validate challenge_id is not nil
	if challengeID == uuid.Nil {
		return nil, fmt.Errorf("challenge_id is required - cannot run analysis without a challenge")
	}

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

	// 2. FETCH COMPANY DATA (use companyID directly, not submission lookup)
	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		s.logger.Error().Err(err).Str("company_id", companyID).Msg("Invalid company ID format")
		return nil, err
	}

	companyData, err := s.companyService.GetByID(ctx, companyUUID)
	if err != nil {
		s.logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to fetch company data")
		return nil, err
	}
	if companyData == nil {
		s.logger.Error().Str("company_id", companyID).Msg("Company not found")
		return nil, fmt.Errorf("company not found: %s", companyID)
	}

	s.logger.Info().
		Str("company_name", submission.CompanyName).
		Str("challenge_id", challengeID.String()).
		Msg("Submission and company data loaded successfully")

	// 3. SETUP CONTEXT
	knowledge := &ContextContainer{
		SubmissionData: submissionData,
		CompanyData:    companyDataToMap(companyData),
	}

	companyIDPtr := &companyID

	analysis, err := s.startAnalysisRecord(ctx, challengeID, companyIDPtr)
	if err != nil {
		return nil, err
	}

	// ========================================================================
	// LAYER 1: THE ENVIRONMENT (Macro + Industry)
	// Fully parallel execution - no dependencies between frameworks
	// ========================================================================
	s.runLayer("Layer 1: Environment", func(wg *sync.WaitGroup) {
		s.exec(wg, func() {
			frameworkStart := time.Now()
			s.logger.Info().Str("framework", "PESTEL").Msg("⚡ PESTEL started")
			var err error
			knowledge.PESTEL, err = s.runPESTEL(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ PESTEL failed")
			} else {
				s.logger.Info().
					Str("framework", "PESTEL").
					Int64("duration_ms", time.Since(frameworkStart).Milliseconds()).
					Msg("✅ PESTEL completed")
			}
		})
		s.exec(wg, func() {
			frameworkStart := time.Now()
			s.logger.Info().Str("framework", "Porter").Msg("⚡ Porter started")
			var err error
			knowledge.Porter, err = s.runPorter(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ Porter failed")
			} else {
				s.logger.Info().
					Str("framework", "Porter").
					Int64("duration_ms", time.Since(frameworkStart).Milliseconds()).
					Msg("✅ Porter completed")
			}
		})
		s.exec(wg, func() {
			frameworkStart := time.Now()
			s.logger.Info().Str("framework", "TAM-SAM-SOM").Msg("⚡ TAM-SAM-SOM started")
			var err error
			knowledge.TamSamSom, err = s.runTamSamSom(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ TAM-SAM-SOM failed")
			} else {
				s.logger.Info().
					Str("framework", "TAM-SAM-SOM").
					Int64("duration_ms", time.Since(frameworkStart).Milliseconds()).
					Msg("✅ TAM-SAM-SOM completed")
			}
		})
	})
	s.logger.Debug().Str("analysis_id", analysis.ID).Msg("Layer 1 checkpoint: Saving progress")
	s.saveCheckpoint(ctx, analysis, knowledge, string(StatusPending))

	// ========================================================================
	// LAYER 2: POSITIONING (Internal Fit)
	// Parallel within layer - both frameworks can run simultaneously
	// Dependencies: SWOT needs Layer 1 (PESTEL + Porter), Benchmarking needs Layer 1 (TAM-SAM-SOM)
	// ========================================================================
	s.runLayer("Layer 2: Positioning", func(wg *sync.WaitGroup) {
		s.exec(wg, func() {
			frameworkStart := time.Now()
			s.logger.Info().Str("framework", "SWOT").Msg("⚡ SWOT started")
			var err error
			knowledge.SWOT, err = s.runSWOT(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ SWOT failed")
			} else {
				s.logger.Info().
					Str("framework", "SWOT").
					Int64("duration_ms", time.Since(frameworkStart).Milliseconds()).
					Msg("✅ SWOT completed")
			}
		})
		s.exec(wg, func() {
			frameworkStart := time.Now()
			s.logger.Info().Str("framework", "Benchmarking").Msg("⚡ Benchmarking started")
			var err error
			knowledge.Benchmarking, err = s.runBenchmarking(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ Benchmarking failed")
			} else {
				s.logger.Info().
					Str("framework", "Benchmarking").
					Int64("duration_ms", time.Since(frameworkStart).Milliseconds()).
					Msg("✅ Benchmarking completed")
			}
		})
	})
	s.logger.Debug().Str("analysis_id", analysis.ID).Msg("Layer 2 checkpoint: Saving progress")
	s.saveCheckpoint(ctx, analysis, knowledge, string(StatusPending))

	// ========================================================================
	// LAYER 3: STRATEGY (Direction)
	// Parallel within layer - all three frameworks can run simultaneously
	// Dependencies: All need Layer 1 outputs
	// NOTE: Could potentially optimize by starting Scenarios and BlueOcean immediately after Layer 1
	// ========================================================================
	s.runLayer("Layer 3: Strategy", func(wg *sync.WaitGroup) {
		s.exec(wg, func() {
			frameworkStart := time.Now()
			s.logger.Info().Str("framework", "BlueOcean").Msg("⚡ BlueOcean started")
			var err error
			knowledge.BlueOcean, err = s.runBlueOcean(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ BlueOcean failed")
			} else {
				s.logger.Info().
					Str("framework", "BlueOcean").
					Int64("duration_ms", time.Since(frameworkStart).Milliseconds()).
					Msg("✅ BlueOcean completed")
			}
		})
		s.exec(wg, func() {
			frameworkStart := time.Now()
			s.logger.Info().Str("framework", "GrowthHacking").Msg("⚡ GrowthHacking started")
			var err error
			knowledge.GrowthHacking, err = s.runGrowthHacking(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ GrowthHacking failed")
			} else {
				s.logger.Info().
					Str("framework", "GrowthHacking").
					Int64("duration_ms", time.Since(frameworkStart).Milliseconds()).
					Msg("✅ GrowthHacking completed")
			}
		})
		s.exec(wg, func() {
			frameworkStart := time.Now()
			s.logger.Info().Str("framework", "Scenarios").Msg("⚡ Scenarios started")
			var err error
			knowledge.Scenarios, err = s.runScenarios(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ Scenarios failed")
			} else {
				s.logger.Info().
					Str("framework", "Scenarios").
					Int64("duration_ms", time.Since(frameworkStart).Milliseconds()).
					Msg("✅ Scenarios completed")
			}
		})
	})
	s.logger.Debug().Str("analysis_id", analysis.ID).Msg("Layer 3 checkpoint: Saving progress")
	s.saveCheckpoint(ctx, analysis, knowledge, string(StatusPending))

	// ========================================================================
	// LAYER 3.5: DECISION MAKING (Priority Recommendations)
	// CRITICAL: Decision Matrix MUST run before OKRs so OKRs can align with recommendations
	// Single framework - sequential execution
	// Dependencies: Needs Layer 3 (Scenarios)
	// ========================================================================
	s.runLayer("Layer 3.5: Decision Making", func(wg *sync.WaitGroup) {
		s.exec(wg, func() {
			frameworkStart := time.Now()
			s.logger.Info().Str("framework", "DecisionMatrix").Msg("⚡ DecisionMatrix started")
			var err error
			knowledge.DecisionMatrix, err = s.runDecisionMatrix(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ DecisionMatrix failed")
			} else {
				s.logger.Info().
					Str("framework", "DecisionMatrix").
					Int64("duration_ms", time.Since(frameworkStart).Milliseconds()).
					Msg("✅ DecisionMatrix completed")
			}
		})
	})
	s.logger.Debug().Str("analysis_id", analysis.ID).Msg("Layer 3.5 checkpoint: Saving progress")
	s.saveCheckpoint(ctx, analysis, knowledge, string(StatusPending))

	// ========================================================================
	// LAYER 4: EXECUTION (Roadmap)
	// Parallel within layer - both frameworks can run simultaneously
	// OKRs now have access to Decision Matrix recommendations for alignment
	// Dependencies: Both need Layer 3.5 (DecisionMatrix)
	// ========================================================================
	s.runLayer("Layer 4: Execution", func(wg *sync.WaitGroup) {
		s.exec(wg, func() {
			frameworkStart := time.Now()
			s.logger.Info().Str("framework", "OKRs").Msg("⚡ OKRs started")
			var err error
			knowledge.OKRs, err = s.runOKRs(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ OKRs failed")
			} else {
				s.logger.Info().
					Str("framework", "OKRs").
					Int64("duration_ms", time.Since(frameworkStart).Milliseconds()).
					Msg("✅ OKRs completed")
			}
		})
		s.exec(wg, func() {
			frameworkStart := time.Now()
			s.logger.Info().Str("framework", "BSC").Msg("⚡ BSC started")
			var err error
			knowledge.BSC, err = s.runBSC(ctx, knowledge)
			if err != nil {
				s.logger.Error().Err(err).Msg("❌ BSC failed")
			} else {
				s.logger.Info().
					Str("framework", "BSC").
					Int64("duration_ms", time.Since(frameworkStart).Milliseconds()).
					Msg("✅ BSC completed")
			}
		})
	})
	s.logger.Debug().Str("analysis_id", analysis.ID).Msg("Layer 4 checkpoint: Saving progress")
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
	// Uses the Premium Model (s.synthesisModel)
	// Dependencies: Requires all framework summaries
	// ========================================================================
	synthesisStart := time.Now()
	s.logger.Info().Str("framework", "Synthesis").Msg("⚡ Synthesis started - using premium model")
	synthesis, err := s.runSynthesis(ctx, knowledge)
	if err != nil {
		s.logger.Error().Err(err).Msg("❌ Synthesis failed")
	} else {
		s.logger.Info().
			Str("framework", "Synthesis").
			Int64("duration_ms", time.Since(synthesisStart).Milliseconds()).
			Msg("✅ Synthesis completed")
	}
	if err := analysis.SetFramework(FrameworkSynthesis, synthesis); err != nil {
		s.logger.Error().Err(err).Msg("Failed to set synthesis framework")
	}

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
		s.repo.Update(ctx, analysis)
		return analysis, fmt.Errorf("analysis incomplete: %s", errorMsg)
	}

	// FINISH
	s.markAsComplete(ctx, analysis, startTime)
	return analysis, nil
}

// =================================================================================
// MECHANICS - Parallel Execution Within Layers
// =================================================================================

// runLayer executes all frameworks in a layer in parallel
// Uses sync.WaitGroup to ensure all parallel tasks complete before proceeding to next layer
func (s *Service) runLayer(name string, tasks func(*sync.WaitGroup)) {
	layerStart := time.Now()
	s.logger.Info().
		Str("layer", name).
		Msg("🚀 Layer started - frameworks will run in PARALLEL")

	var wg sync.WaitGroup
	tasks(&wg)
	wg.Wait()

	layerDuration := time.Since(layerStart)
	s.logger.Info().
		Str("layer", name).
		Int64("duration_ms", layerDuration.Milliseconds()).
		Float64("duration_seconds", layerDuration.Seconds()).
		Msg("✅ Layer completed - all parallel frameworks finished")
}

// exec spawns a single framework task in a goroutine
// Increments WaitGroup counter before launching, decrements when done
func (s *Service) exec(wg *sync.WaitGroup, task func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		task()
	}()
}

func (s *Service) startAnalysisRecord(ctx context.Context, challengeID uuid.UUID, companyID *string) (*Analysis, error) {
	s.logger.Info().
		Str("challenge_id", challengeID.String()).
		Interface("company_id", companyID).
		Msg("startAnalysisRecord: BEGIN")

	existing, err := s.repo.GetByChallengeID(ctx, challengeID)
	s.logger.Info().
		Str("challenge_id", challengeID.String()).
		Bool("found", err == nil && existing != nil).
		Interface("error", err).
		Msg("startAnalysisRecord: GetByChallengeID result")

	if err == nil && existing != nil {
		s.logger.Info().
			Str("challenge_id", challengeID.String()).
			Str("existing_analysis_id", existing.ID).
			Str("existing_status", existing.Status).
			Msg("startAnalysisRecord: Found existing analysis record")

		switch existing.Status {
		case string(StatusCompleted):
			// Allow re-running analysis by resetting the existing record
			s.logger.Info().
				Str("challenge_id", challengeID.String()).
				Str("analysis_id", existing.ID).
				Str("old_status", existing.Status).
				Msg("startAnalysisRecord: Resetting existing analysis to pending for re-run")

			existing.Status = string(StatusPending)
			existing.CompanyID = companyID
			existing.UpdatedAt = time.Now()
			existing.CompletedAt = nil

			// Clear previous analysis results for fresh run
			existing.FrameworkResults = make(map[string]json.RawMessage)

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
		s.logger.Error().Err(err).Str("challenge_id", challengeID.String()).Msg("startAnalysisRecord: Error fetching existing analysis (not 'not found')")
		return nil, err
	}

	s.logger.Info().Str("challenge_id", challengeID.String()).Msg("startAnalysisRecord: No existing analysis found, creating new record")

	// If not found or other retrieval error, create a new record
	a := &Analysis{
		ID:          uuid.New().String(),
		CompanyID:   companyID,
		ChallengeID: challengeID,
		Status:      string(StatusPending),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	s.logger.Info().
		Str("analysis_id", a.ID).
		Str("challenge_id", challengeID.String()).
		Interface("company_id", companyID).
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

	// Store frameworks using SetFramework helper
	if k.PESTEL != nil {
		a.SetFramework(FrameworkPESTEL, *k.PESTEL)
	}
	if k.Porter != nil {
		a.SetFramework(FrameworkPorter, *k.Porter)
	}
	if k.TamSamSom != nil {
		a.SetFramework(FrameworkTAMSAMSOM, *k.TamSamSom)
	}
	if k.SWOT != nil {
		a.SetFramework(FrameworkSWOT, *k.SWOT)
	}
	if k.Benchmarking != nil {
		a.SetFramework(FrameworkBenchmarking, *k.Benchmarking)
	}
	if k.BlueOcean != nil {
		a.SetFramework(FrameworkBlueOcean, *k.BlueOcean)
	}
	if k.GrowthHacking != nil {
		a.SetFramework(FrameworkGrowthHacking, *k.GrowthHacking)
	}
	if k.Scenarios != nil {
		a.SetFramework(FrameworkScenarios, *k.Scenarios)
	}
	if k.OKRs != nil {
		a.SetFramework(FrameworkOKRs, *k.OKRs)
	}
	if k.BSC != nil {
		a.SetFramework(FrameworkBSC, *k.BSC)
	}
	if k.DecisionMatrix != nil {
		a.SetFramework(FrameworkDecisionMatrix, *k.DecisionMatrix)
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
	now := time.Now()
	totalDuration := time.Since(startTime)
	s.logger.Info().
		Int64("total_processing_time_ms", totalDuration.Milliseconds()).
		Float64("total_processing_time_seconds", totalDuration.Seconds()).
		Float64("total_processing_time_minutes", totalDuration.Minutes()).
		Msg("🎉 Analysis processing COMPLETED - all frameworks finished")
	a.CompletedAt = &now
	s.repo.Update(ctx, a)

	// Auto-generate access code for public sharing
	if a.AccessCode == nil || *a.AccessCode == "" {
		code, err := s.GenerateAccessCode(ctx, a.ID)
		if err != nil {
			s.logger.Warn().Err(err).Str("analysis_id", a.ID).Msg("Failed to auto-generate access code")
		} else {
			s.logger.Info().Str("analysis_id", a.ID).Str("access_code", code).Msg("Access code auto-generated")
		}
	}

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
		"company_data":  k.CompanyData,
		"macro_context": s.extractMacroContext(),
	}
	opts := llm.NewGenerationOptions(s.frameworks[FrameworkPESTEL])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, withDataPriority(llm.FrameworkPESTELPrompt), data, &res)
	return &res, err
}

func (s *Service) runPorter(ctx context.Context, k *ContextContainer) (*PorterAnalysis, error) {
	var res PorterAnalysis
	data := map[string]interface{}{
		"company_data":  k.CompanyData,
		"macro_context": s.extractMacroContext(),
	}
	opts := llm.NewGenerationOptions(s.frameworks[FrameworkPorter])
	err := s.llm.GenerateStructuredWithOptions(ctx, opts, withDataPriority(llm.FrameworkPorterPrompt), data, &res)
	return &res, err
}

func (s *Service) runTamSamSom(ctx context.Context, k *ContextContainer) (*TamSamSomAnalysis, error) {
	var res TamSamSomAnalysis
	data := map[string]interface{}{
		"company_data":  k.CompanyData,
		"macro_context": s.extractMacroContext(),
	}
	opts := llm.NewGenerationOptions(s.frameworks[FrameworkTAMSAMSOM])
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
		"company_data":    k.CompanyData,
		"pestel_insights": pestelSummary,
		"porter_insights": porterSummary,
	}
	opts := llm.NewGenerationOptions(s.frameworks[FrameworkSWOT])
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
		"company_data": k.CompanyData,
		"market_scale": marketScale,
	}
	opts := llm.NewGenerationOptions(s.frameworks[FrameworkBenchmarking])
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
		"company_data":    k.CompanyData,
		"porter_insights": porterSummary,
	}
	opts := llm.NewGenerationOptions(s.frameworks[FrameworkBlueOcean])
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
		"company_data":       k.CompanyData,
		"swot_summary":       swotSummary,
		"swot_weaknesses":    swotWeaknesses,
		"swot_opportunities": swotOpportunities,
		"market_scale":       marketScale,
	}
	opts := llm.NewGenerationOptions(s.frameworks[FrameworkGrowthHacking])
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
		"company_data":    k.CompanyData,
		"pestel_insights": pestelSummary,
		"macro_context":   s.extractMacroContext(),
	}
	opts := llm.NewGenerationOptions(s.frameworks[FrameworkScenarios])
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
		"company_data":                    k.CompanyData,
		"blue_ocean_insights":             blueOceanSummary,
		"swot_weaknesses":                 swotWeaknesses,
		"decision_matrix_recommendations": decisionMatrixRecommendations,
	}
	opts := llm.NewGenerationOptions(s.frameworks[FrameworkOKRs])
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
		"company_data":        k.CompanyData,
		"blue_ocean_insights": blueOceanSummary,
	}
	opts := llm.NewGenerationOptions(s.frameworks[FrameworkBSC])
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
		"company_data":      k.CompanyData,
		"scenario_insights": scenarioSummary,
	}
	opts := llm.NewGenerationOptions(s.frameworks[FrameworkDecisionMatrix])
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
		FrameworkPESTEL:        safeGetSummary(k.PESTEL),
		FrameworkPorter:        safeGetSummary(k.Porter),
		FrameworkSWOT:          safeGetSummary(k.SWOT),
		FrameworkBlueOcean:     safeGetSummary(k.BlueOcean),
		FrameworkOKRs:          safeGetSummary(k.OKRs),
		FrameworkScenarios:     safeGetSummary(k.Scenarios),
		FrameworkGrowthHacking: safeGetSummary(k.GrowthHacking),
	}
	data := map[string]interface{}{
		"company_data":            k.CompanyData,
		"all_framework_summaries": summaries,
	}
	// NEW: Uses framework-specific synthesis config (Claude 3.5 Sonnet with T=0.4)
	opts := llm.NewGenerationOptions(s.frameworks[FrameworkSynthesis])
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

// companyDataToMap converts AnalysisCompanyData to a map for template injection
func companyDataToMap(c *AnalysisCompanyData) map[string]interface{} {
	result := map[string]interface{}{
		"name": c.Name,
	}

	// Add all optional fields if present
	if c.CNPJ != nil {
		result["cnpj"] = *c.CNPJ
	}
	if c.Website != nil {
		result["website"] = *c.Website
	}
	if c.Industry != nil {
		result["industry"] = *c.Industry
	}
	if c.Sector != nil {
		result["sector"] = *c.Sector
	}
	if c.CompanySize != nil {
		result["company_size"] = *c.CompanySize
	}
	if c.Location != nil {
		result["location"] = *c.Location
	}
	if c.TargetMarket != nil {
		result["target_market"] = *c.TargetMarket
	}
	if c.FundingStage != nil {
		result["funding_stage"] = *c.FundingStage
	}
	if c.FoundationYear != nil {
		result["foundation_year"] = *c.FoundationYear
	}
	if c.Headquarters != nil {
		result["headquarters"] = *c.Headquarters
	}
	if c.TargetAudience != nil {
		result["target_audience"] = *c.TargetAudience
	}
	if c.ValueProposition != nil {
		result["value_proposition"] = *c.ValueProposition
	}
	if c.EmployeesRange != nil {
		result["employees_range"] = *c.EmployeesRange
	}
	if c.RevenueEstimate != nil {
		result["revenue_estimate"] = *c.RevenueEstimate
	}
	if c.BusinessModel != nil {
		result["business_model"] = *c.BusinessModel
	}
	if len(c.Competitors) > 0 {
		result["competitors"] = c.Competitors
	}
	if c.MarketShareStatus != nil {
		result["market_share_status"] = *c.MarketShareStatus
	}
	if c.DigitalMaturity != nil {
		result["digital_maturity"] = *c.DigitalMaturity
	}
	if len(c.Strengths) > 0 {
		result["strengths"] = c.Strengths
	}
	if len(c.Weaknesses) > 0 {
		result["weaknesses"] = c.Weaknesses
	}
	if c.MacroContext != nil {
		result["macro_context"] = c.MacroContext
	}

	return result
}

// extractMacroContext returns hardcoded Brazilian economic indicators for MVP
// These values are updated periodically and reflect the most recent available data
// TODO: Re-enable dynamic fetch from database when scaling beyond MVP
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

// =================================================================================
// FRAMEWORK VALIDATION
// =================================================================================

// validateFrameworkCompleteness checks all frameworks in the context container and returns names of empty ones
func (s *Service) validateFrameworkCompleteness(k *ContextContainer) []string {
	var empty []string

	if k.PESTEL == nil || k.PESTEL.Summary == "" {
		empty = append(empty, FrameworkPESTEL)
	}
	if k.Porter == nil || k.Porter.Summary == "" {
		empty = append(empty, FrameworkPorter)
	}
	if k.TamSamSom == nil || k.TamSamSom.Summary == "" {
		empty = append(empty, FrameworkTAMSAMSOM)
	}
	if k.SWOT == nil || k.SWOT.Summary == "" {
		empty = append(empty, FrameworkSWOT)
	}
	if k.Benchmarking == nil || k.Benchmarking.Summary == "" {
		empty = append(empty, FrameworkBenchmarking)
	}
	if k.BlueOcean == nil || k.BlueOcean.Summary == "" {
		empty = append(empty, FrameworkBlueOcean)
	}
	if k.GrowthHacking == nil || k.GrowthHacking.Summary == "" {
		empty = append(empty, FrameworkGrowthHacking)
	}
	if k.Scenarios == nil || k.Scenarios.Summary == "" {
		empty = append(empty, FrameworkScenarios)
	}
	if k.OKRs == nil || k.OKRs.Summary == "" {
		empty = append(empty, FrameworkOKRs)
	}
	if k.BSC == nil || k.BSC.Summary == "" {
		empty = append(empty, FrameworkBSC)
	}
	if k.DecisionMatrix == nil || k.DecisionMatrix.Summary == "" {
		empty = append(empty, FrameworkDecisionMatrix)
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
	var pestel PESTELAnalysis
	var porter PorterAnalysis
	pestelEmpty := a.GetFramework(FrameworkPESTEL, &pestel) != nil || pestel.Summary == ""
	porterEmpty := a.GetFramework(FrameworkPorter, &porter) != nil || porter.Summary == ""
	if pestelEmpty && porterEmpty {
		missing = append(missing, "environment_layer (pestel+porter both empty)")
	}

	// Layer 2 - Positioning (SWOT is critical)
	var swot SWOTAnalysis
	if a.GetFramework(FrameworkSWOT, &swot) != nil || (swot.Summary == "" && len(swot.Strengths) == 0 && len(swot.Weaknesses) == 0) {
		missing = append(missing, FrameworkSWOT)
	}

	// Layer 3 - Strategy (at least one strategy framework)
	var blueOcean BlueOceanAnalysis
	var growthHacking GrowthHackingAnalysis
	var scenarios ScenarioAnalysis
	blueOceanEmpty := a.GetFramework(FrameworkBlueOcean, &blueOcean) != nil || blueOcean.Summary == ""
	growthHackingEmpty := a.GetFramework(FrameworkGrowthHacking, &growthHacking) != nil || growthHacking.Summary == ""
	scenariosEmpty := a.GetFramework(FrameworkScenarios, &scenarios) != nil || scenarios.Summary == ""
	if blueOceanEmpty && growthHackingEmpty && scenariosEmpty {
		missing = append(missing, "strategy_layer (blue_ocean+growth_hacking+scenarios all empty)")
	}

	// Layer 4 - Execution (OKRs or BSC)
	var okrs OKRAnalysis
	var bsc BalancedScorecardAnalysis
	okrsEmpty := a.GetFramework(FrameworkOKRs, &okrs) != nil || okrs.Summary == ""
	bscEmpty := a.GetFramework(FrameworkBSC, &bsc) != nil || bsc.Summary == ""
	if okrsEmpty && bscEmpty {
		missing = append(missing, "execution_layer (okrs+bsc both empty)")
	}

	// Synthesis must exist
	var synthesis AnalysisSynthesis
	if a.GetFramework(FrameworkSynthesis, &synthesis) != nil || (synthesis.ExecutiveSummary == "" && synthesis.CentralChallenge == "") {
		missing = append(missing, FrameworkSynthesis)
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
