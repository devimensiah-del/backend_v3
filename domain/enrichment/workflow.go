package enrichment

import (
	"backend_v3/adapter/macrodata"
	"backend_v3/domain/submission"
	"backend_v3/llm"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// EnrichSubmission - Simplified Two-Phase AI-Only Pipeline
// Phase 1: Perplexity pre-search for company identification + data gathering
// Phase 2: Gemini synthesis for final profile generation
// NO external adapters (DNS, scraper, macrodata APIs removed - Perplexity handles everything)
func (s *Service) EnrichSubmission(ctx context.Context, submissionID uuid.UUID) (*Enrichment, error) {
	startTime := time.Now()

	// VERSION MARKER - If you see this log, you're running the UPDATED code
	log.Info().Str("version", "2024-11-28-FIX-MACRO").Msg(">>> ENRICHMENT WORKFLOW v2 - Macro injection enabled <<<")

	// 1. SETUP WORKSPACE
	sub, enrichment, err := s.setupWorkspace(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	// If enrichment is nil, it means it's locked/already done, so we skip
	if enrichment == nil {
		log.Info().Str("sub_id", submissionID.String()).Msg("Enrichment skipped (Locked or Completed)")
		return nil, nil
	}

	// 2. PERPLEXITY PRE-SEARCH PHASE (Company ID + Data Gathering)
	s.updateStatus(ctx, enrichment, "Pesquisando dados da empresa e mercado...", 15)
	preSearchContext := s.runPreSearch(ctx, sub)

	// 3. FETCH MACRO DATA (BCB/IBGE APIs) - Returns null values if APIs fail
	s.updateStatus(ctx, enrichment, "Obtendo indicadores econômicos do BCB/IBGE...", 35)
	macroDataContext := s.fetchAndFormatMacroData(ctx)

	// 4. DETECT GAPS (for Gemini to focus on)
	missingFields := s.detectMissingFields(sub)

	// 5. GEMINI SYNTHESIS PHASE (Final Profile Generation)
	s.updateStatus(ctx, enrichment, "Sintetizando Perfil de Inteligência...", 50)

	prompt := llm.UnifiedEnrichmentPrompt
	prompt = strings.ReplaceAll(prompt, "{{COMPANY_NAME}}", sub.CompanyName)
	prompt = strings.ReplaceAll(prompt, "{{USER_CONTEXT}}", s.compileUserDossier(sub))
	prompt = strings.ReplaceAll(prompt, "{{MISSING_FIELDS}}", missingFields)

	// Inject Perplexity pre-search results
	prompt = strings.ReplaceAll(prompt, "{{PRE_SEARCH_CONTEXT}}", preSearchContext)

	// Inject authoritative macro data from BCB/IBGE APIs
	prompt = strings.ReplaceAll(prompt, "{{REAL_TIME_MACRO_DATA}}", macroDataContext)

	// Remove old placeholder (no longer used)
	prompt = strings.ReplaceAll(prompt, "{{TECHNICAL_CONTEXT}}", "(Technical data gathered by Perplexity - see PRE_SEARCH_CONTEXT above)")

	agentReq := llm.Request{
		Model:        s.enrichmentCfg.Model,
		SystemPrompt: "You are a JSON-only Corporate Intelligence Agent.",
		Messages:     []llm.Message{{Role: "user", Content: prompt}},
		Tools:        []string{"search"}, // Gemini can still search if needed
		Temperature:  s.enrichmentCfg.Temperature,
		MaxTokens:    s.enrichmentCfg.MaxTokens,
	}

	// Call LLM with automatic fallback on failure
	log.Info().
		Str("primary_model", s.enrichmentCfg.Model).
		Str("fallback_model", s.enrichmentCfg.FallbackModel).
		Int("max_tokens", s.enrichmentCfg.MaxTokens).
		Msg("Starting Gemini synthesis for enrichment")
	resp, err := s.llmClient.CallWithFallback(ctx, &agentReq, s.enrichmentCfg.FallbackModel)
	if err != nil {
		return nil, s.handleCrash(ctx, sub, enrichment, err)
	}

	// 5. PARSE & SAVE
	s.updateStatus(ctx, enrichment, "Finalizando perfil...", 90)

	var finalProfile map[string]interface{}
	cleanJson := s.cleanJsonBlock(resp.Content)

	// CRITICAL: Fail explicitly on JSON parse errors
	if err := json.Unmarshal([]byte(cleanJson), &finalProfile); err != nil {
		log.Error().
			Err(err).
			Str("content", resp.Content).
			Str("submission_id", submissionID.String()).
			Msg("CRITICAL: LLM returned invalid JSON - failing enrichment job")

		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w (content: %s)", err, cleanJson)
	}

	// INJECT submitted_data section from the original submission form
	finalProfile["submitted_data"] = s.buildSubmittedData(sub)

	// INJECT authoritative macro data directly (don't trust AI to copy it correctly)
	s.injectMacroDataIntoProfile(ctx, finalProfile)

	enrichment.EnrichedData = JSONMap(finalProfile)
	enrichment.SourcesStatus = s.combineAISources(resp.Sources)

	// Save final state
	s.saveProfile(ctx, enrichment, nil)

	log.Info().Dur("duration", time.Since(startTime)).Msg("Enrichment Pipeline Success (AI-Only)")
	return s.markAsComplete(ctx, sub, enrichment)
}

// EnrichCompany - Company-First Enrichment Pipeline
// Primary source: companies table (verified fields respected)
// Uses Perplexity pre-search + Gemini synthesis, same as EnrichSubmission
// but builds context from company record instead of submission form
func (s *Service) EnrichCompany(ctx context.Context, companyID uuid.UUID) (*Enrichment, error) {
	startTime := time.Now()

	// 1. VALIDATE COMPANY SERVICE
	if s.companyService == nil {
		return nil, fmt.Errorf("company service not configured - cannot enrich company")
	}

	// 2. LOAD COMPANY DATA
	company, err := s.companyService.GetCompanyDataByID(ctx, companyID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get company: %w", err)
	}
	if company == nil {
		return nil, fmt.Errorf("company not found: %s", companyID.String())
	}

	// 3. GET VERIFIED FIELDS (protected from re-enrichment)
	verifiedFields, err := s.companyService.GetVerifiedFieldNames(ctx, companyID.String())
	if err != nil {
		log.Warn().Err(err).Str("company_id", companyID.String()).Msg("Failed to get verified fields, proceeding without protection")
		verifiedFields = []string{}
	}

	// 4. GET OR CREATE ENRICHMENT RECORD
	enrichment, err := s.repo.GetByCompanyID(ctx, companyID)
	if err != nil {
		// Create new enrichment for this company
		// We need a submission ID - use a nil UUID placeholder or get from linked submission
		enrichment = NewEnrichmentForCompany(companyID, uuid.Nil)
		if err := s.repo.Create(ctx, enrichment); err != nil {
			return nil, fmt.Errorf("failed to create enrichment record: %w", err)
		}
	}

	// 5. CHECK LOCK
	if enrichment.IsLocked {
		log.Info().Str("company_id", companyID.String()).Msg("Enrichment skipped (Locked)")
		return nil, nil
	}

	// 6. START PROCESSING
	enrichment.Start()
	if err := s.repo.UpdateSystem(ctx, enrichment); err != nil {
		return nil, err
	}

	// 7. PERPLEXITY PRE-SEARCH PHASE
	s.updateStatus(ctx, enrichment, "Pesquisando dados da empresa e mercado...", 15)
	preSearchContext := s.runPreSearchForCompany(ctx, company)

	// 8. FETCH MACRO DATA (BCB/IBGE APIs)
	s.updateStatus(ctx, enrichment, "Obtendo indicadores econômicos do BCB/IBGE...", 35)
	macroDataContext := s.fetchAndFormatMacroData(ctx)

	// 9. DETECT GAPS (exclude verified fields)
	missingFields := s.detectMissingCompanyFields(company, verifiedFields)

	// 10. GEMINI SYNTHESIS PHASE
	s.updateStatus(ctx, enrichment, "Sintetizando Perfil de Inteligência...", 50)

	prompt := llm.UnifiedEnrichmentPrompt
	prompt = strings.ReplaceAll(prompt, "{{COMPANY_NAME}}", company.Name)
	prompt = strings.ReplaceAll(prompt, "{{USER_CONTEXT}}", s.compileCompanyDossier(company))
	prompt = strings.ReplaceAll(prompt, "{{MISSING_FIELDS}}", missingFields)
	prompt = strings.ReplaceAll(prompt, "{{PRE_SEARCH_CONTEXT}}", preSearchContext)
	prompt = strings.ReplaceAll(prompt, "{{REAL_TIME_MACRO_DATA}}", macroDataContext)
	prompt = strings.ReplaceAll(prompt, "{{TECHNICAL_CONTEXT}}", "(Technical data gathered by Perplexity - see PRE_SEARCH_CONTEXT above)")

	agentReq := llm.Request{
		Model:        s.enrichmentCfg.Model,
		SystemPrompt: "You are a JSON-only Corporate Intelligence Agent.",
		Messages:     []llm.Message{{Role: "user", Content: prompt}},
		Tools:        []string{"search"},
		Temperature:  s.enrichmentCfg.Temperature,
		MaxTokens:    s.enrichmentCfg.MaxTokens,
	}

	log.Info().
		Str("company_id", companyID.String()).
		Str("company_name", company.Name).
		Str("model", s.enrichmentCfg.Model).
		Int("verified_fields", len(verifiedFields)).
		Msg("Starting company-first enrichment")

	resp, err := s.llmClient.CallWithFallback(ctx, &agentReq, s.enrichmentCfg.FallbackModel)
	if err != nil {
		return nil, s.handleCrashCompany(ctx, companyID, enrichment, err)
	}

	// 11. PARSE & SAVE
	s.updateStatus(ctx, enrichment, "Finalizando perfil...", 90)

	var finalProfile map[string]interface{}
	cleanJson := s.cleanJsonBlock(resp.Content)

	if err := json.Unmarshal([]byte(cleanJson), &finalProfile); err != nil {
		log.Error().
			Err(err).
			Str("content", resp.Content).
			Str("company_id", companyID.String()).
			Msg("CRITICAL: LLM returned invalid JSON - failing enrichment job")
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w (content: %s)", err, cleanJson)
	}

	// Build company_data section (like submitted_data but from company record)
	finalProfile["company_data"] = s.buildCompanyDataSection(company)

	// INJECT authoritative macro data directly (don't trust AI to copy it correctly)
	s.injectMacroDataIntoProfile(ctx, finalProfile)

	enrichment.EnrichedData = JSONMap(finalProfile)
	enrichment.SourcesStatus = s.combineAISources(resp.Sources)

	s.saveProfile(ctx, enrichment, nil)

	log.Info().
		Dur("duration", time.Since(startTime)).
		Str("company_id", companyID.String()).
		Msg("Company Enrichment Pipeline Success")

	return s.markAsCompleteCompany(ctx, companyID, enrichment)
}

// handleCrashCompany handles enrichment failures for company-first pipeline
func (s *Service) handleCrashCompany(ctx context.Context, companyID uuid.UUID, e *Enrichment, err error) error {
	log.Error().Err(err).Str("company_id", companyID.String()).Msg("Company Enrichment Crashed")
	e.Fail(err)
	_ = s.repo.UpdateSystem(ctx, e)
	return err
}

// markAsCompleteCompany completes enrichment and updates company record
func (s *Service) markAsCompleteCompany(ctx context.Context, companyID uuid.UUID, e *Enrichment) (*Enrichment, error) {
	log.Info().
		Str("enrichment_id", e.ID.String()).
		Str("company_id", companyID.String()).
		Msg("Marking company enrichment as complete")

	e.Finish()
	e.IsLocked = false
	if err := s.repo.ForceUpdateAndUnlock(ctx, e); err != nil {
		log.Error().
			Err(err).
			Str("enrichment_id", e.ID.String()).
			Msg("Failed to mark enrichment as complete")
		return nil, err
	}

	// Update company record with enriched data (non-blocking)
	if s.companyService != nil {
		go func() {
			if err := s.updateCompanyFromEnrichmentByID(context.Background(), companyID, e); err != nil {
				log.Warn().
					Err(err).
					Str("company_id", companyID.String()).
					Msg("Failed to update company from enrichment")
			} else {
				log.Info().
					Str("company_id", companyID.String()).
					Msg("Company updated with enrichment data")
			}
		}()
	}

	return e, nil
}

// updateCompanyFromEnrichmentByID updates company record using company ID directly
func (s *Service) updateCompanyFromEnrichmentByID(ctx context.Context, companyID uuid.UUID, e *Enrichment) error {
	data := e.EnrichedData

	input := CompanyUpdateInput{
		EnrichmentID: e.ID.String(),
	}

	// Same extraction logic as updateCompanyFromEnrichment
	if overview, ok := data["profile_overview"].(map[string]interface{}); ok {
		if v, ok := overview["legal_name"].(string); ok && v != "" {
			input.LegalName = &v
		}
		if v, ok := overview["foundation_year"].(string); ok && v != "" {
			input.FoundationYear = &v
		}
		if v, ok := overview["headquarters"].(string); ok && v != "" {
			input.Headquarters = &v
		}
	}

	if market, ok := data["market_position"].(map[string]interface{}); ok {
		if v, ok := market["sector"].(string); ok && v != "" {
			input.Sector = &v
		}
		if v, ok := market["target_audience"].(string); ok && v != "" {
			input.TargetAudience = &v
		}
		if v, ok := market["value_proposition"].(string); ok && v != "" {
			input.ValueProposition = &v
		}
	}

	if financials, ok := data["financials"].(map[string]interface{}); ok {
		if v, ok := financials["employees_range"].(string); ok && v != "" {
			input.EmployeesRange = &v
		}
		if v, ok := financials["revenue_estimate"].(string); ok && v != "" {
			input.RevenueEstimate = &v
		}
		if v, ok := financials["business_model"].(string); ok && v != "" {
			input.BusinessModel = &v
		}
	}

	if competitive, ok := data["competitive_landscape"].(map[string]interface{}); ok {
		if v, ok := competitive["market_share_status"].(string); ok && v != "" {
			input.MarketShareStatus = &v
		}
		if comps, ok := competitive["competitors"].([]interface{}); ok {
			for _, c := range comps {
				if sVal, ok := c.(string); ok {
					input.Competitors = append(input.Competitors, sVal)
				}
			}
		}
	}

	if strategic, ok := data["strategic_assessment"].(map[string]interface{}); ok {
		if v, ok := strategic["digital_maturity"].(float64); ok {
			dm := int(v)
			input.DigitalMaturity = &dm
		}
		if strengths, ok := strategic["strengths"].([]interface{}); ok {
			for _, sVal := range strengths {
				if str, ok := sVal.(string); ok {
					input.Strengths = append(input.Strengths, str)
				}
			}
		}
		if weaknesses, ok := strategic["weaknesses"].([]interface{}); ok {
			for _, w := range weaknesses {
				if str, ok := w.(string); ok {
					input.Weaknesses = append(input.Weaknesses, str)
				}
			}
		}
	}

	if discovered, ok := data["discovered_data"].(map[string]interface{}); ok {
		if v, ok := discovered["cnpj"].(string); ok && v != "" {
			input.CNPJ = &v
		}
		if v, ok := discovered["website"].(string); ok && v != "" {
			input.Website = &v
		}
		if v, ok := discovered["linkedin_url"].(string); ok && v != "" {
			input.LinkedInURL = &v
		}
		if v, ok := discovered["twitter_handle"].(string); ok && v != "" {
			input.TwitterHandle = &v
		}
	}

	// Use smart merge with company ID (not submission ID)
	// For company-first enrichment, we pass company ID as the "submission" param
	// since the interface uses submissionID to look up company
	return s.companyService.UpdateFromEnrichmentSmartMerge(ctx, companyID.String(), input)
}

// runPreSearchForCompany runs Perplexity pre-search for company-first enrichment
func (s *Service) runPreSearchForCompany(ctx context.Context, company *CompanyData) string {
	if s.preSearchCfg.Model == "" {
		log.Warn().Msg("Pre-search model not configured, skipping pre-search phase")
		return "(Pre-search not configured - using existing company data directly)"
	}

	prompt := llm.PreSearchPrompt
	prompt = strings.ReplaceAll(prompt, "{{COMPANY_NAME}}", company.Name)
	prompt = strings.ReplaceAll(prompt, "{{USER_CONTEXT}}", s.compileCompanyDossier(company))

	preSearchReq := llm.Request{
		Model:        s.preSearchCfg.Model,
		SystemPrompt: "You are a JSON-only Company Identification Expert. Always return valid JSON.",
		Messages:     []llm.Message{{Role: "user", Content: prompt}},
		Temperature:  0.3,
		MaxTokens:    2000,
	}

	log.Info().
		Str("company_name", company.Name).
		Str("model", s.preSearchCfg.Model).
		Msg("Running pre-search phase for company identification")

	preSearchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := s.llmClient.CallWithFallback(preSearchCtx, &preSearchReq, s.preSearchCfg.FallbackModel)
	if err != nil {
		log.Warn().
			Err(err).
			Str("company_name", company.Name).
			Msg("Pre-search failed, continuing without pre-identification (graceful degradation)")
		return fmt.Sprintf("(Pre-search failed: %s - proceeding with existing company data)", err.Error())
	}

	cleanedJSON := s.cleanJsonBlock(resp.Content)
	var preSearchResult map[string]interface{}
	if err := json.Unmarshal([]byte(cleanedJSON), &preSearchResult); err != nil {
		log.Warn().
			Err(err).
			Str("raw_response", resp.Content).
			Msg("Pre-search returned invalid JSON, continuing with partial data")
		return fmt.Sprintf("(Pre-search raw data - may need manual validation: %s)", cleanedJSON)
	}

	confidenceScore := 0.0
	if score, ok := preSearchResult["confidence_score"].(float64); ok {
		confidenceScore = score
	}

	log.Info().
		Str("company_name", company.Name).
		Float64("confidence_score", confidenceScore).
		Msg("Pre-search completed successfully")

	return cleanedJSON
}

// buildCompanyDataSection creates the company_data section for company-first enrichment
func (s *Service) buildCompanyDataSection(company *CompanyData) map[string]interface{} {
	data := map[string]interface{}{
		"company_name": company.Name,
	}

	if company.CNPJ != nil && *company.CNPJ != "" {
		data["cnpj"] = *company.CNPJ
	}
	if company.Website != nil && *company.Website != "" {
		data["website"] = *company.Website
	}
	if company.Industry != nil && *company.Industry != "" {
		data["industry"] = *company.Industry
	}
	if company.CompanySize != nil && *company.CompanySize != "" {
		data["company_size"] = *company.CompanySize
	}
	if company.Location != nil && *company.Location != "" {
		data["location"] = *company.Location
	}
	if company.TargetMarket != nil && *company.TargetMarket != "" {
		data["target_market"] = *company.TargetMarket
	}
	if company.FundingStage != nil && *company.FundingStage != "" {
		data["funding_stage"] = *company.FundingStage
	}
	if company.AnnualRevenueMin != nil {
		data["annual_revenue_min"] = *company.AnnualRevenueMin
	}
	if company.AnnualRevenueMax != nil {
		data["annual_revenue_max"] = *company.AnnualRevenueMax
	}
	if company.LinkedInURL != nil && *company.LinkedInURL != "" {
		data["linkedin_url"] = *company.LinkedInURL
	}
	if company.TwitterHandle != nil && *company.TwitterHandle != "" {
		data["twitter_handle"] = *company.TwitterHandle
	}

	return data
}

// --- HELPER METHODS ---

func (s *Service) setupWorkspace(ctx context.Context, submissionID uuid.UUID) (*submission.Submission, *Enrichment, error) {
	// 1. Get Submission
	sub, err := s.submissionRepo.GetByID(ctx, submissionID)
	if err != nil {
		return nil, nil, fmt.Errorf("submission not found: %w", err)
	}

	// 2. Check if Enrichment exists
	enrichment, err := s.repo.GetBySubmissionID(ctx, submissionID)
	if err != nil {
		// Create new if not found
		enrichment = NewEnrichment(submissionID)
		if err := s.repo.Create(ctx, enrichment); err != nil {
			return nil, nil, fmt.Errorf("failed to create enrichment record: %w", err)
		}
	}

	// 3. Check Lock and Status
	// Block if: locked by user OR already being processed (prevents double-start)
	if enrichment.IsLocked {
		return sub, nil, nil // Return nil enrichment to signal "skip"
	}
	// Check if already processing (started but not completed) to prevent double-start
	if enrichment.StartedAt != nil && enrichment.Status == StatusPending && enrichment.Progress > 0 {
		log.Info().
			Str("sub_id", submissionID.String()).
			Int("progress", enrichment.Progress).
			Msg("Enrichment already processing - skipping to prevent double-start")
		return sub, nil, nil
	}

	// 4. Link enrichment to company (if not already linked)
	if enrichment.CompanyID == nil && s.companyService != nil {
		companyIDStr, _ := s.companyService.GetCompanyIDBySubmissionID(ctx, submissionID.String())
		if companyIDStr != nil {
			companyID, err := uuid.Parse(*companyIDStr)
			if err == nil {
				enrichment.CompanyID = &companyID
				log.Debug().
					Str("enrichment_id", enrichment.ID.String()).
					Str("company_id", companyID.String()).
					Msg("Linked enrichment to company")
			}
		}
	}

	// 5. Start processing
	enrichment.Start()
	if err := s.repo.UpdateSystem(ctx, enrichment); err != nil {
		return nil, nil, err
	}

	return sub, enrichment, nil
}

func (s *Service) updateStatus(ctx context.Context, e *Enrichment, msg string, pct int) {
	e.UpdateProgress(msg, pct)
	_ = s.repo.UpdateSystem(ctx, e) // Ignore error, not critical
}

func (s *Service) handleCrash(ctx context.Context, sub *submission.Submission, e *Enrichment, err error) error {
	log.Error().Err(err).Str("sub_id", sub.ID.String()).Msg("Enrichment Crashed")
	e.Fail(err)
	_ = s.repo.UpdateSystem(ctx, e)
	return err
}

func (s *Service) markAsComplete(ctx context.Context, sub *submission.Submission, e *Enrichment) (*Enrichment, error) {
	log.Info().
		Str("enrichment_id", e.ID.String()).
		Int("progress", e.Progress).
		Bool("was_locked", e.IsLocked).
		Msg("Marking enrichment as complete")

	e.Finish()
	// Force unlock on completion to prevent stuck enrichments (bypasses user lock)
	e.IsLocked = false
	if err := s.repo.ForceUpdateAndUnlock(ctx, e); err != nil {
		log.Error().
			Err(err).
			Str("enrichment_id", e.ID.String()).
			Msg("Failed to mark enrichment as complete")
		return nil, err
	}

	log.Info().
		Str("enrichment_id", e.ID.String()).
		Str("status", string(e.Status)).
		Msg("Enrichment marked as complete successfully")

	// Update company record with enriched data (non-blocking)
	if s.companyService != nil {
		go func() {
			if err := s.updateCompanyFromEnrichment(context.Background(), sub.ID, e); err != nil {
				log.Warn().
					Err(err).
					Str("submission_id", sub.ID.String()).
					Msg("Failed to update company from enrichment - company data may be incomplete")
			} else {
				log.Info().
					Str("submission_id", sub.ID.String()).
					Msg("Company updated with enrichment data")
			}
		}()
	}

	return e, nil
}

// updateCompanyFromEnrichment extracts enriched data and updates the company record
// Uses smart merge: only updates fields that are NOT verified (protected)
func (s *Service) updateCompanyFromEnrichment(ctx context.Context, submissionID uuid.UUID, e *Enrichment) error {
	log.Info().
		Str("submission_id", submissionID.String()).
		Interface("company_id", e.CompanyID).
		Msg("Starting company update from enrichment")

	// Extract relevant fields from enriched data
	data := e.EnrichedData

	input := CompanyUpdateInput{
		EnrichmentID: e.ID.String(),
	}

	// Extract profile_overview fields
	if overview, ok := data["profile_overview"].(map[string]interface{}); ok {
		if v, ok := overview["legal_name"].(string); ok && v != "" {
			input.LegalName = &v
		}
		if v, ok := overview["foundation_year"].(string); ok && v != "" {
			input.FoundationYear = &v
		}
		if v, ok := overview["headquarters"].(string); ok && v != "" {
			input.Headquarters = &v
		}
	}

	// Extract market_position fields
	if market, ok := data["market_position"].(map[string]interface{}); ok {
		if v, ok := market["sector"].(string); ok && v != "" {
			input.Sector = &v
		}
		if v, ok := market["target_audience"].(string); ok && v != "" {
			input.TargetAudience = &v
		}
		if v, ok := market["value_proposition"].(string); ok && v != "" {
			input.ValueProposition = &v
		}
	}

	// Extract financials fields
	if financials, ok := data["financials"].(map[string]interface{}); ok {
		if v, ok := financials["employees_range"].(string); ok && v != "" {
			input.EmployeesRange = &v
		}
		if v, ok := financials["revenue_estimate"].(string); ok && v != "" {
			input.RevenueEstimate = &v
		}
		if v, ok := financials["business_model"].(string); ok && v != "" {
			input.BusinessModel = &v
		}
	}

	// Extract competitive_landscape fields
	if competitive, ok := data["competitive_landscape"].(map[string]interface{}); ok {
		if v, ok := competitive["market_share_status"].(string); ok && v != "" {
			input.MarketShareStatus = &v
		}
		if comps, ok := competitive["competitors"].([]interface{}); ok {
			for _, c := range comps {
				if sVal, ok := c.(string); ok {
					input.Competitors = append(input.Competitors, sVal)
				}
			}
		}
	}

	// Extract strategic_assessment fields
	if strategic, ok := data["strategic_assessment"].(map[string]interface{}); ok {
		if v, ok := strategic["digital_maturity"].(float64); ok {
			dm := int(v)
			input.DigitalMaturity = &dm
		}
		if strengths, ok := strategic["strengths"].([]interface{}); ok {
			for _, sVal := range strengths {
				if str, ok := sVal.(string); ok {
					input.Strengths = append(input.Strengths, str)
				}
			}
		}
		if weaknesses, ok := strategic["weaknesses"].([]interface{}); ok {
			for _, w := range weaknesses {
				if str, ok := w.(string); ok {
					input.Weaknesses = append(input.Weaknesses, str)
				}
			}
		}
	}

	// Extract discovered_data (basic info discovered by AI)
	if discovered, ok := data["discovered_data"].(map[string]interface{}); ok {
		if v, ok := discovered["cnpj"].(string); ok && v != "" {
			input.CNPJ = &v
		}
		if v, ok := discovered["website"].(string); ok && v != "" {
			input.Website = &v
		}
		if v, ok := discovered["linkedin_url"].(string); ok && v != "" {
			input.LinkedInURL = &v
		}
		if v, ok := discovered["twitter_handle"].(string); ok && v != "" {
			input.TwitterHandle = &v
		}
	}

	// Use smart merge: respects verified fields, overwrites unverified ones
	// NOTE: The adapter expects submission ID, not company ID - it looks up company internally
	return s.companyService.UpdateFromEnrichmentSmartMerge(ctx, submissionID.String(), input)
}

func (s *Service) saveProfile(ctx context.Context, e *Enrichment, err error) {
	if err != nil {
		e.Fail(err)
	}

	log.Info().
		Str("enrichment_id", e.ID.String()).
		Int("progress", e.Progress).
		Bool("is_locked", e.IsLocked).
		Msg("Saving enriched profile to database")

	// Actually save the enriched data to database
	if updateErr := s.repo.UpdateSystem(ctx, e); updateErr != nil {
		log.Error().
			Err(updateErr).
			Str("enrichment_id", e.ID.String()).
			Int("progress", e.Progress).
			Bool("is_locked", e.IsLocked).
			Msg("Failed to save enriched profile to database - enrichment may be locked by user")
	} else {
		log.Info().
			Str("enrichment_id", e.ID.String()).
			Msg("Enriched profile saved successfully")
	}
}

func (s *Service) compileUserDossier(sub *submission.Submission) string {
	dossier := fmt.Sprintf("Nome da Empresa: %s\n", sub.CompanyName)

	// Company Information
	if sub.CNPJ != nil && *sub.CNPJ != "" {
		dossier += fmt.Sprintf("CNPJ: %s\n", *sub.CNPJ)
	}
	if sub.CompanyWebsite != nil && *sub.CompanyWebsite != "" {
		dossier += fmt.Sprintf("Website: %s\n", *sub.CompanyWebsite)
	}
	if sub.CompanyIndustry != nil && *sub.CompanyIndustry != "" {
		dossier += fmt.Sprintf("Setor/Indústria: %s\n", *sub.CompanyIndustry)
	}
	if sub.CompanySize != nil && *sub.CompanySize != "" {
		dossier += fmt.Sprintf("Tamanho da Empresa: %s\n", *sub.CompanySize)
	}
	if sub.CompanyLocation != nil && *sub.CompanyLocation != "" {
		dossier += fmt.Sprintf("Localização: %s\n", *sub.CompanyLocation)
	}

	// Contact Information
	dossier += fmt.Sprintf("Nome do Contato: %s\n", sub.ContactName)
	dossier += fmt.Sprintf("Email do Contato: %s\n", sub.ContactEmail)
	if sub.ContactPhone != nil && *sub.ContactPhone != "" {
		dossier += fmt.Sprintf("Telefone: %s\n", *sub.ContactPhone)
	}
	if sub.ContactPosition != nil && *sub.ContactPosition != "" {
		dossier += fmt.Sprintf("Cargo: %s\n", *sub.ContactPosition)
	}

	// Business Context
	if sub.TargetMarket != nil && *sub.TargetMarket != "" {
		dossier += fmt.Sprintf("Mercado Alvo: %s\n", *sub.TargetMarket)
	}
	if sub.FundingStage != nil && *sub.FundingStage != "" {
		dossier += fmt.Sprintf("Estágio de Financiamento: %s\n", *sub.FundingStage)
	}
	if sub.AnnualRevenueMin != nil {
		dossier += fmt.Sprintf("Receita Anual Mínima: R$ %.2f\n", *sub.AnnualRevenueMin)
	}
	if sub.AnnualRevenueMax != nil {
		dossier += fmt.Sprintf("Receita Anual Máxima: R$ %.2f\n", *sub.AnnualRevenueMax)
	}

	// Strategic Challenge
	if sub.BusinessChallenge != "" {
		dossier += fmt.Sprintf("Desafio Principal: %s\n", sub.BusinessChallenge)
	}
	if sub.AdditionalNotes != nil && *sub.AdditionalNotes != "" {
		dossier += fmt.Sprintf("Notas Adicionais: %s\n", *sub.AdditionalNotes)
	}

	// Social Links
	if sub.LinkedInURL != nil && *sub.LinkedInURL != "" {
		dossier += fmt.Sprintf("LinkedIn: %s\n", *sub.LinkedInURL)
	}
	if sub.TwitterHandle != nil && *sub.TwitterHandle != "" {
		dossier += fmt.Sprintf("Twitter/X: %s\n", *sub.TwitterHandle)
	}

	return dossier
}

// compileCompanyDossier builds the company context dossier from Company data
// Used by EnrichCompany for company-first enrichment (instead of submission-first)
func (s *Service) compileCompanyDossier(company *CompanyData) string {
	dossier := fmt.Sprintf("Nome da Empresa: %s\n", company.Name)

	// Core identifiers (already enriched/verified)
	if company.CNPJ != nil && *company.CNPJ != "" {
		dossier += fmt.Sprintf("CNPJ: %s\n", *company.CNPJ)
	}
	if company.Website != nil && *company.Website != "" {
		dossier += fmt.Sprintf("Website: %s\n", *company.Website)
	}
	if company.Industry != nil && *company.Industry != "" {
		dossier += fmt.Sprintf("Setor/Indústria: %s\n", *company.Industry)
	}
	if company.CompanySize != nil && *company.CompanySize != "" {
		dossier += fmt.Sprintf("Tamanho da Empresa: %s\n", *company.CompanySize)
	}
	if company.Location != nil && *company.Location != "" {
		dossier += fmt.Sprintf("Localização: %s\n", *company.Location)
	}
	if company.TargetMarket != nil && *company.TargetMarket != "" {
		dossier += fmt.Sprintf("Mercado Alvo: %s\n", *company.TargetMarket)
	}
	if company.FundingStage != nil && *company.FundingStage != "" {
		dossier += fmt.Sprintf("Estágio de Financiamento: %s\n", *company.FundingStage)
	}
	if company.AnnualRevenueMin != nil {
		dossier += fmt.Sprintf("Receita Anual Mínima: R$ %.2f\n", *company.AnnualRevenueMin)
	}
	if company.AnnualRevenueMax != nil {
		dossier += fmt.Sprintf("Receita Anual Máxima: R$ %.2f\n", *company.AnnualRevenueMax)
	}

	// Enriched data
	if company.FoundationYear != nil && *company.FoundationYear != "" {
		dossier += fmt.Sprintf("Ano de Fundação: %s\n", *company.FoundationYear)
	}
	if company.LegalName != nil && *company.LegalName != "" {
		dossier += fmt.Sprintf("Razão Social: %s\n", *company.LegalName)
	}
	if company.Headquarters != nil && *company.Headquarters != "" {
		dossier += fmt.Sprintf("Sede: %s\n", *company.Headquarters)
	}
	if company.Sector != nil && *company.Sector != "" {
		dossier += fmt.Sprintf("Setor: %s\n", *company.Sector)
	}

	// Social links
	if company.LinkedInURL != nil && *company.LinkedInURL != "" {
		dossier += fmt.Sprintf("LinkedIn: %s\n", *company.LinkedInURL)
	}
	if company.TwitterHandle != nil && *company.TwitterHandle != "" {
		dossier += fmt.Sprintf("Twitter/X: %s\n", *company.TwitterHandle)
	}

	return dossier
}

// detectMissingCompanyFields identifies fields that need to be enriched
// Excludes verified fields from the missing list (they're protected)
func (s *Service) detectMissingCompanyFields(company *CompanyData, verifiedFields []string) string {
	verifiedSet := make(map[string]bool)
	for _, f := range verifiedFields {
		verifiedSet[f] = true
	}

	var missing []string

	// Check each field - only add to missing if NOT verified
	if !verifiedSet["website"] && (company.Website == nil || *company.Website == "") {
		missing = append(missing, "- WEBSITE/DOMÍNIO (Crítico: Encontre o site oficial)")
	}
	if !verifiedSet["location"] && (company.Location == nil || *company.Location == "") {
		missing = append(missing, "- LOCALIZAÇÃO (Sede: Cidade/País)")
	}
	if !verifiedSet["industry"] && (company.Industry == nil || *company.Industry == "") {
		missing = append(missing, "- SETOR DE ATUAÇÃO (CNAE/Indústria)")
	}
	if !verifiedSet["annual_revenue_min"] && company.AnnualRevenueMin == nil {
		missing = append(missing, "- FATURAMENTO ANUAL (Estime via porte/setor)")
	}
	if !verifiedSet["target_market"] && (company.TargetMarket == nil || *company.TargetMarket == "") {
		missing = append(missing, "- PÚBLICO ALVO (B2B/B2C, Perfil de Cliente)")
	}

	if len(missing) == 0 {
		return "Nenhum dado crítico faltando. Foque em validar a veracidade e aprofundar a estratégia."
	}
	return strings.Join(missing, "\n")
}

// buildSubmittedData creates the submitted_data section from submission fields
func (s *Service) buildSubmittedData(sub *submission.Submission) map[string]interface{} {
	data := map[string]interface{}{
		"company_name":       sub.CompanyName,
		"contact_name":       sub.ContactName,
		"contact_email":      sub.ContactEmail,
		"business_challenge": sub.BusinessChallenge,
	}

	// Optional fields - only add if provided
	if sub.CNPJ != nil && *sub.CNPJ != "" {
		data["cnpj"] = *sub.CNPJ
	}
	if sub.CompanyWebsite != nil && *sub.CompanyWebsite != "" {
		data["website"] = *sub.CompanyWebsite
	}
	if sub.CompanyIndustry != nil && *sub.CompanyIndustry != "" {
		data["industry"] = *sub.CompanyIndustry
	}
	if sub.CompanySize != nil && *sub.CompanySize != "" {
		data["company_size"] = *sub.CompanySize
	}
	if sub.CompanyLocation != nil && *sub.CompanyLocation != "" {
		data["location"] = *sub.CompanyLocation
	}
	if sub.ContactPhone != nil && *sub.ContactPhone != "" {
		data["contact_phone"] = *sub.ContactPhone
	}
	if sub.ContactPosition != nil && *sub.ContactPosition != "" {
		data["contact_position"] = *sub.ContactPosition
	}
	if sub.TargetMarket != nil && *sub.TargetMarket != "" {
		data["target_market"] = *sub.TargetMarket
	}
	if sub.FundingStage != nil && *sub.FundingStage != "" {
		data["funding_stage"] = *sub.FundingStage
	}
	if sub.AnnualRevenueMin != nil {
		data["annual_revenue_min"] = *sub.AnnualRevenueMin
	}
	if sub.AnnualRevenueMax != nil {
		data["annual_revenue_max"] = *sub.AnnualRevenueMax
	}
	if sub.AdditionalNotes != nil && *sub.AdditionalNotes != "" {
		data["additional_notes"] = *sub.AdditionalNotes
	}
	if sub.LinkedInURL != nil && *sub.LinkedInURL != "" {
		data["linkedin_url"] = *sub.LinkedInURL
	}
	if sub.TwitterHandle != nil && *sub.TwitterHandle != "" {
		data["twitter_handle"] = *sub.TwitterHandle
	}

	return data
}

// combineAISources converts LLM sources to JSONMap for storage
func (s *Service) combineAISources(aiSources []llm.Source) JSONMap {
	combined := make(JSONMap)
	combined["perplexity_presearch"] = "success"
	combined["gemini_synthesis"] = "success"
	for _, src := range aiSources {
		if src.URL != "" {
			combined[src.URL] = "ai_search_result"
		}
	}
	return combined
}

func (s *Service) detectMissingFields(sub *submission.Submission) string {
	var missing []string

	if sub.CompanyWebsite == nil || *sub.CompanyWebsite == "" {
		missing = append(missing, "- WEBSITE/DOMÍNIO (Crítico: Encontre o site oficial)")
	}
	if sub.CompanyLocation == nil || *sub.CompanyLocation == "" {
		missing = append(missing, "- LOCALIZAÇÃO (Sede: Cidade/País)")
	}
	if sub.CompanyIndustry == nil || *sub.CompanyIndustry == "" {
		missing = append(missing, "- SETOR DE ATUAÇÃO (CNAE/Indústria)")
	}
	if sub.AnnualRevenueMin == nil {
		missing = append(missing, "- FATURAMENTO ANUAL (Estime via porte/setor)")
	}
	if sub.TargetMarket == nil || *sub.TargetMarket == "" {
		missing = append(missing, "- PÚBLICO ALVO (B2B/B2C, Perfil de Cliente)")
	}

	if len(missing) == 0 {
		return "Nenhum dado crítico omitido pelo usuário. Foque em validar a veracidade e aprofundar a estratégia."
	}
	return strings.Join(missing, "\n")
}

func (s *Service) cleanJsonBlock(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return content
}

// fetchAndFormatMacroData fetches authoritative economic data
// Priority: 1. DB (macroService) -> 2. APIs (macroProvider)
// and formats it for injection into the enrichment prompt
// If all sources fail, returns instructions to set economic indicator fields to null
func (s *Service) fetchAndFormatMacroData(ctx context.Context) string {
	// 1. PRIMARY: Try DB-first via macroService (if configured)
	if s.macroService != nil {
		snapshot, err := s.macroService.GetLatestSnapshot(ctx)
		if err == nil && snapshot.HasData() {
			log.Debug().
				Int("indicator_count", len(snapshot.Indicators)).
				Msg("Using cached macro data from DB")
			return s.formatMacroSnapshot(snapshot)
		}
		if err != nil {
			log.Warn().Err(err).Msg("DB macro data unavailable, falling back to API")
		} else {
			log.Warn().Msg("DB has no macro data, falling back to API")
		}
	}

	// 2. FALLBACK: Try direct API calls via macroProvider
	if s.macroProvider == nil {
		log.Warn().Msg("MacroDataProvider not configured")
		return "DADOS ECONÔMICOS: INDISPONÍVEIS\nINSTRUÇÃO: Defina todos os campos de indicadores econômicos como null no JSON."
	}

	// Fetch data from BCB/IBGE APIs (with 6h cache)
	macroCtx, err := s.macroProvider.FetchLatestMacroData(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch macro data from APIs")
		return "DADOS ECONÔMICOS: ERRO NA API\nINSTRUÇÃO: Defina todos os campos de indicadores econômicos como null no JSON."
	}

	// Check if we got any data at all
	hasData := macroCtx.EconomicIndicators.SELIC != nil ||
		macroCtx.EconomicIndicators.IPCA != nil ||
		macroCtx.ExchangeRates.USDtoBRL != nil

	if !hasData {
		log.Warn().Msg("No macro data returned from any API")
		return `DADOS ECONÔMICOS: NENHUM DADO RETORNADO
INSTRUÇÃO IMPORTANTE:
- NÃO invente dados macroeconômicos
- Defina os seguintes campos como null no JSON de macro_context.economic_indicators:
  - interest_rate: null
  - inflation_rate: null
  - exchange_rate: null
  - gdp_growth: null
- Você pode estimar impactos setoriais com base em tendências gerais, mas NÃO cite valores específicos de SELIC, IPCA ou USD/BRL`
	}

	// Format the API data for prompt injection
	return s.formatMacroContext(macroCtx)
}

// formatMacroSnapshot formats DB-backed macro snapshot for prompt injection
// Now formats ALL available indicators grouped by category
func (s *Service) formatMacroSnapshot(snapshot *MacroSnapshot) string {
	var sb strings.Builder
	sb.WriteString("# INDICADORES ECONÔMICOS BRASILEIROS ATUAIS\n\n")
	sb.WriteString("IMPORTANTE: Selecione apenas os indicadores relevantes para o setor e desafio da empresa analisada.\n")
	sb.WriteString("Para indicadores não disponíveis, mantenha null no JSON.\n\n")

	// Category display names
	categoryNames := map[string]string{
		"interest_rate": "TAXA DE JUROS",
		"inflation":     "INFLAÇÃO",
		"exchange_rate": "CÂMBIO",
		"gdp":           "PRODUTO INTERNO BRUTO (PIB)",
		"production":    "PRODUÇÃO E SERVIÇOS",
		"employment":    "EMPREGO",
		"construction":  "CONSTRUÇÃO",
	}

	// Category order for consistent output
	categoryOrder := []string{
		"interest_rate", "inflation", "exchange_rate",
		"gdp", "production", "employment", "construction",
	}

	// Group indicators by category
	byCategory := make(map[string][]*MacroIndicator)
	for _, ind := range snapshot.Indicators {
		byCategory[ind.Category] = append(byCategory[ind.Category], ind)
	}

	// Format each category
	for _, cat := range categoryOrder {
		indicators, exists := byCategory[cat]
		if !exists || len(indicators) == 0 {
			continue
		}

		categoryName, ok := categoryNames[cat]
		if !ok {
			categoryName = cat
		}

		sb.WriteString(fmt.Sprintf("## %s\n", categoryName))

		for _, ind := range indicators {
			// Format value with unit
			var valueStr string
			if ind.Unit == "BRL" {
				valueStr = fmt.Sprintf("R$ %.2f", ind.Value)
			} else if ind.Unit == "%" {
				valueStr = fmt.Sprintf("%.2f%%", ind.Value)
			} else {
				valueStr = fmt.Sprintf("%.2f %s", ind.Value, ind.Unit)
			}

			sb.WriteString(fmt.Sprintf("- **%s**: %s", ind.Name, valueStr))

			// Add reference period if available
			if ind.ReferencePeriod != nil && *ind.ReferencePeriod != "" {
				sb.WriteString(fmt.Sprintf(" (ref: %s)", *ind.ReferencePeriod))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("*Dados atualizados em: %s*\n", snapshot.AsOf.Format("02/01/2006 15:04")))
	sb.WriteString("\n=== FIM DOS DADOS ECONÔMICOS ===\n")

	log.Info().
		Int("indicator_count", len(snapshot.Indicators)).
		Str("source", "db_cache").
		Msg("Macro data formatted for prompt")

	return sb.String()
}

// formatMacroContext formats API-fetched macro context for prompt injection
func (s *Service) formatMacroContext(macroCtx *macrodata.MacroContext) string {
	var sb strings.Builder
	sb.WriteString("=== DADOS ECONÔMICOS OFICIAIS (APIs BCB/IBGE) ===\n")
	sb.WriteString("IMPORTANTE: USE APENAS os valores fornecidos abaixo. Para campos marcados como null, mantenha null no JSON.\n\n")

	// SELIC
	if macroCtx.EconomicIndicators.SELIC != nil {
		selic := macroCtx.EconomicIndicators.SELIC
		sb.WriteString(fmt.Sprintf("TAXA SELIC: %.2f%% a.a.\n", selic.Rate))
		sb.WriteString(fmt.Sprintf("  - Data: %s\n", selic.Date.Format("02/01/2006")))
		sb.WriteString(fmt.Sprintf("  - Fonte: %s\n\n", selic.Source))
	} else {
		sb.WriteString("TAXA SELIC: null (API indisponível)\n\n")
	}

	// IPCA
	if macroCtx.EconomicIndicators.IPCA != nil {
		ipca := macroCtx.EconomicIndicators.IPCA
		sb.WriteString(fmt.Sprintf("IPCA: %.2f%% (%s)\n", ipca.Rate, ipca.MonthYear))
		sb.WriteString(fmt.Sprintf("  - Fonte: %s\n\n", ipca.Source))
	} else {
		sb.WriteString("IPCA: null (API indisponível)\n\n")
	}

	// USD/BRL
	if macroCtx.ExchangeRates.USDtoBRL != nil {
		usd := macroCtx.ExchangeRates.USDtoBRL
		sb.WriteString(fmt.Sprintf("CÂMBIO USD/BRL: R$ %.4f\n", usd.Rate))
		sb.WriteString(fmt.Sprintf("  - Data: %s\n", usd.Date.Format("02/01/2006 15:04")))
		sb.WriteString(fmt.Sprintf("  - Fonte: %s\n\n", usd.Source))
	} else {
		sb.WriteString("CÂMBIO USD/BRL: null (API indisponível)\n\n")
	}

	sb.WriteString("=== FIM DOS DADOS ===\n")

	log.Info().
		Bool("has_selic", macroCtx.EconomicIndicators.SELIC != nil).
		Bool("has_ipca", macroCtx.EconomicIndicators.IPCA != nil).
		Bool("has_usd_brl", macroCtx.ExchangeRates.USDtoBRL != nil).
		Str("source", "api").
		Msg("Macro data formatted for prompt")

	return sb.String()
}

// injectMacroDataIntoProfile overwrites the AI-generated macro_context with real DB values
// This guarantees correct economic indicators regardless of what the AI hallucinated
func (s *Service) injectMacroDataIntoProfile(ctx context.Context, profile map[string]interface{}) {
	log.Info().Msg("=== MACRO INJECTION START ===")

	// Get or create macro_context section
	macroContext, ok := profile["macro_context"].(map[string]interface{})
	if !ok {
		log.Debug().Msg("Creating new macro_context section (not found in profile)")
		macroContext = make(map[string]interface{})
		profile["macro_context"] = macroContext
	}

	// Get or create economic_indicators section
	econIndicators, ok := macroContext["economic_indicators"].(map[string]interface{})
	if !ok {
		log.Debug().Msg("Creating new economic_indicators section (not found in macro_context)")
		econIndicators = make(map[string]interface{})
		macroContext["economic_indicators"] = econIndicators
	}

	// Log current AI-generated values before overwrite
	if currentRate, exists := econIndicators["interest_rate"]; exists {
		log.Debug().Interface("ai_interest_rate", currentRate).Msg("AI-generated interest_rate (will be overwritten)")
	}
	if currentExchange, exists := econIndicators["exchange_rate"]; exists {
		log.Debug().Interface("ai_exchange_rate", currentExchange).Msg("AI-generated exchange_rate (will be overwritten)")
	}

	// Always set country and injection timestamp for verification
	econIndicators["country"] = "Brasil"
	econIndicators["_macro_injected_at"] = time.Now().Format(time.RFC3339)

	// 1. Try DB first via macroService
	log.Debug().Bool("macroService_configured", s.macroService != nil).Msg("Checking macroService")
	if s.macroService != nil {
		snapshot, err := s.macroService.GetLatestSnapshot(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("DB macro data unavailable for injection - GetLatestSnapshot error")
		} else if snapshot == nil {
			log.Warn().Msg("DB macro data unavailable for injection - snapshot is nil")
		} else if !snapshot.HasData() {
			log.Warn().Msg("DB macro data unavailable for injection - snapshot has no data")
		} else {
			log.Info().
				Int("indicator_count", len(snapshot.Indicators)).
				Msg("Found macro data in DB, injecting...")
			// Log what we're about to inject
			for code, ind := range snapshot.Indicators {
				log.Debug().
					Str("code", code).
					Str("name", ind.Name).
					Float64("value", ind.Value).
					Str("unit", ind.Unit).
					Msg("DB indicator found")
			}
			s.injectFromSnapshot(econIndicators, snapshot)
			log.Info().
				Interface("injected_interest_rate", econIndicators["interest_rate"]).
				Interface("injected_exchange_rate", econIndicators["exchange_rate"]).
				Msg("Injected authoritative macro data from DB into enrichment")
			return
		}
	}

	// 2. Fallback to API
	log.Debug().Bool("macroProvider_configured", s.macroProvider != nil).Msg("Checking macroProvider for API fallback")
	if s.macroProvider != nil {
		macroCtx, err := s.macroProvider.FetchLatestMacroData(ctx)
		if err == nil {
			log.Debug().
				Bool("has_selic", macroCtx.EconomicIndicators.SELIC != nil).
				Bool("has_ipca", macroCtx.EconomicIndicators.IPCA != nil).
				Bool("has_usd_brl", macroCtx.ExchangeRates.USDtoBRL != nil).
				Msg("API macro data fetched")
			s.injectFromAPI(econIndicators, macroCtx)
			log.Info().
				Interface("injected_interest_rate", econIndicators["interest_rate"]).
				Interface("injected_exchange_rate", econIndicators["exchange_rate"]).
				Msg("Injected authoritative macro data from API into enrichment")
			return
		}
		log.Warn().Err(err).Msg("API macro data unavailable for injection")
	}

	log.Warn().Msg("=== MACRO INJECTION FAILED - No authoritative macro data available, AI-generated values will be used ===")
}

// injectFromSnapshot injects DB-backed macro data into economic_indicators
func (s *Service) injectFromSnapshot(econIndicators map[string]interface{}, snapshot *MacroSnapshot) {
	for _, ind := range snapshot.Indicators {
		switch ind.Code {
		case "selic", "selic_meta":
			econIndicators["interest_rate"] = fmt.Sprintf("SELIC %.2f%% a.a.", ind.Value)
		case "ipca", "ipca_12m":
			econIndicators["inflation_rate"] = fmt.Sprintf("IPCA %.2f%% (acum. 12 meses)", ind.Value)
		case "usd_brl":
			econIndicators["exchange_rate"] = fmt.Sprintf("USD/BRL %.4f", ind.Value)
		case "pib_var", "pib_growth":
			econIndicators["gdp_growth"] = fmt.Sprintf("%.1f%%", ind.Value)
		case "desemprego", "unemployment":
			econIndicators["unemployment_rate"] = fmt.Sprintf("%.1f%%", ind.Value)
		}
	}
}

// injectFromAPI injects API-fetched macro data into economic_indicators
func (s *Service) injectFromAPI(econIndicators map[string]interface{}, macroCtx *macrodata.MacroContext) {
	if macroCtx.EconomicIndicators.SELIC != nil {
		econIndicators["interest_rate"] = fmt.Sprintf("SELIC %.2f%% a.a.", macroCtx.EconomicIndicators.SELIC.Rate)
	}
	if macroCtx.EconomicIndicators.IPCA != nil {
		econIndicators["inflation_rate"] = fmt.Sprintf("IPCA %.2f%% (%s)", macroCtx.EconomicIndicators.IPCA.Rate, macroCtx.EconomicIndicators.IPCA.MonthYear)
	}
	if macroCtx.ExchangeRates.USDtoBRL != nil {
		econIndicators["exchange_rate"] = fmt.Sprintf("USD/BRL %.4f", macroCtx.ExchangeRates.USDtoBRL.Rate)
	}
}

// runPreSearch executes the Perplexity pre-search phase to identify the exact company
// This step runs BEFORE the main enrichment to solve company disambiguation
// Returns JSON string of pre-search results or graceful fallback message
func (s *Service) runPreSearch(ctx context.Context, sub *submission.Submission) string {
	// Skip if pre-search model is not configured
	if s.preSearchCfg.Model == "" {
		log.Warn().Msg("Pre-search model not configured, skipping pre-search phase")
		return "(Pre-search not configured - using user-provided company name directly)"
	}

	// Build pre-search prompt
	prompt := llm.PreSearchPrompt
	prompt = strings.ReplaceAll(prompt, "{{COMPANY_NAME}}", sub.CompanyName)
	prompt = strings.ReplaceAll(prompt, "{{USER_CONTEXT}}", s.compileUserDossier(sub))

	preSearchReq := llm.Request{
		Model:        s.preSearchCfg.Model,
		SystemPrompt: "You are a JSON-only Company Identification Expert. Always return valid JSON.",
		Messages:     []llm.Message{{Role: "user", Content: prompt}},
		Temperature:  0.3,  // Lower temperature for more consistent identification
		MaxTokens:    2000, // Smaller response for pre-search
	}

	log.Info().
		Str("company_name", sub.CompanyName).
		Str("model", s.preSearchCfg.Model).
		Msg("Running pre-search phase for company identification")

	// Execute with timeout (pre-search should be fast)
	preSearchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := s.llmClient.CallWithFallback(preSearchCtx, &preSearchReq, s.preSearchCfg.FallbackModel)
	if err != nil {
		log.Warn().
			Err(err).
			Str("company_name", sub.CompanyName).
			Msg("Pre-search failed, continuing without pre-identification (graceful degradation)")
		return fmt.Sprintf("(Pre-search failed: %s - proceeding with user-provided company name)", err.Error())
	}

	// Validate the response is valid JSON
	cleanedJSON := s.cleanJsonBlock(resp.Content)
	var preSearchResult map[string]interface{}
	if err := json.Unmarshal([]byte(cleanedJSON), &preSearchResult); err != nil {
		log.Warn().
			Err(err).
			Str("raw_response", resp.Content).
			Msg("Pre-search returned invalid JSON, continuing with partial data")
		return fmt.Sprintf("(Pre-search raw data - may need manual validation: %s)", cleanedJSON)
	}

	// Check confidence score - if too low, note it
	confidenceScore := 0.0
	if score, ok := preSearchResult["confidence_score"].(float64); ok {
		confidenceScore = score
	}

	log.Info().
		Str("company_name", sub.CompanyName).
		Float64("confidence_score", confidenceScore).
		Msg("Pre-search completed successfully")

	// Return the full JSON for injection into main enrichment prompt
	return cleanedJSON
}
