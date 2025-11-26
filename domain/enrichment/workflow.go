package enrichment

import (
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

	// 2. PERPLEXITY PRE-SEARCH PHASE (Company ID + Data Gathering + Macro Data)
	// This single call replaces: DNS lookup, web scraping, macrodata APIs
	s.updateStatus(ctx, enrichment, "Perplexity is researching company and market data...", 15)
	preSearchContext := s.runPreSearch(ctx, sub)

	// 3. DETECT GAPS (for Gemini to focus on)
	missingFields := s.detectMissingFields(sub)

	// 4. GEMINI SYNTHESIS PHASE (Final Profile Generation)
	s.updateStatus(ctx, enrichment, "Gemini is synthesizing Intelligence Profile...", 50)

	prompt := llm.UnifiedEnrichmentPrompt
	prompt = strings.ReplaceAll(prompt, "{{COMPANY_NAME}}", sub.CompanyName)
	prompt = strings.ReplaceAll(prompt, "{{USER_CONTEXT}}", s.compileUserDossier(sub))
	prompt = strings.ReplaceAll(prompt, "{{MISSING_FIELDS}}", missingFields)

	// Inject Perplexity pre-search results (replaces technical context + macro data)
	prompt = strings.ReplaceAll(prompt, "{{PRE_SEARCH_CONTEXT}}", preSearchContext)

	// Remove old placeholders (no longer used)
	prompt = strings.ReplaceAll(prompt, "{{TECHNICAL_CONTEXT}}", "(Technical data gathered by Perplexity - see PRE_SEARCH_CONTEXT above)")
	prompt = strings.ReplaceAll(prompt, "{{REAL_TIME_MACRO_DATA}}", "(Macro data gathered by Perplexity - see PRE_SEARCH_CONTEXT above)")

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
	s.updateStatus(ctx, enrichment, "Finalizing profile...", 90)

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

	enrichment.EnrichedData = JSONMap(finalProfile)
	enrichment.SourcesStatus = s.combineAISources(resp.Sources)

	// Save final state
	s.saveProfile(ctx, enrichment, nil)

	log.Info().Dur("duration", time.Since(startTime)).Msg("Enrichment Pipeline Success (AI-Only)")
	return s.markAsComplete(ctx, sub, enrichment)
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

	// 3. Check Lock
	if enrichment.IsLocked || enrichment.Status == StatusApproved {
		return sub, nil, nil // Return nil enrichment to signal "skip"
	}

	// 4. Start processing
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

	return e, nil
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
		Temperature:  0.3, // Lower temperature for more consistent identification
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
