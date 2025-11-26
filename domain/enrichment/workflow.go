package enrichment

import (
	"backend_v3/adapter/dns"
	"backend_v3/adapter/scraper"
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

// EnrichSubmission - The "Transient Data" Pipeline
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

	// 2. GATHER TRANSIENT DATA (Memory Only - No DB Save)
	s.updateStatus(ctx, enrichment, "Scanning digital footprint (Transient)...", 10)
	technicalData := s.gatherTransientData(ctx, sub)

	// Convert technical data to JSON string for the Prompt Context
	techContextBytes, _ := json.Marshal(technicalData)
	techContextString := string(techContextBytes)

	// NEW: Fetch real-time Brazilian macro-economic data
	s.updateStatus(ctx, enrichment, "Fetching real-time macro-economic data...", 25)
	macro, err := s.macroProvider.FetchLatestMacroData(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("failed to fetch macro data (continuing with partial data)")
		// Graceful degradation - continue without macro data
	}

	// 3. DETECT GAPS
	missingFields := s.detectMissingFields(sub)

	// 4. AGENT EXECUTION
	s.updateStatus(ctx, enrichment, "Agent is synthesizing Intelligence Profile...", 40)

	prompt := llm.UnifiedEnrichmentPrompt
	prompt = strings.ReplaceAll(prompt, "{{COMPANY_NAME}}", sub.CompanyName)
	prompt = strings.ReplaceAll(prompt, "{{USER_CONTEXT}}", s.compileUserDossier(sub))
	prompt = strings.ReplaceAll(prompt, "{{TECHNICAL_CONTEXT}}", techContextString)
	prompt = strings.ReplaceAll(prompt, "{{MISSING_FIELDS}}", missingFields)

	// NEW: Inject real-time macro data if available
	if macro != nil {
		macroJSON, _ := json.Marshal(macro)
		macroContextStr := string(macroJSON)
		prompt = strings.ReplaceAll(prompt, "{{REAL_TIME_MACRO_DATA}}", macroContextStr)
	} else {
		// Graceful fallback if macro data unavailable
		prompt = strings.ReplaceAll(prompt, "{{REAL_TIME_MACRO_DATA}}", "(Macro data unavailable - use LLM search for current economic context)")
	}

	agentReq := llm.Request{
		Model:        s.enrichmentCfg.Model,
		SystemPrompt: "You are a JSON-only Corporate Intelligence Agent.",
		Messages:     []llm.Message{{Role: "user", Content: prompt}},
		// IMPORTANT: Ensure your client.go maps "search" to the provider's specific tool (e.g., google_search)
		Tools:       []string{"search"},
		Temperature: s.enrichmentCfg.Temperature, // Use config temp
		MaxTokens:   s.enrichmentCfg.MaxTokens,   // Use config tokens
	}

	// Call LLM with automatic fallback on failure (rate limit, timeout, 5xx errors)
	log.Info().
		Str("primary_model", s.enrichmentCfg.Model).
		Str("fallback_model", s.enrichmentCfg.FallbackModel).
		Int("max_tokens", s.enrichmentCfg.MaxTokens).
		Msg("Starting LLM call for enrichment")
	resp, err := s.llmClient.CallWithFallback(ctx, &agentReq, s.enrichmentCfg.FallbackModel)
	if err != nil {
		return nil, s.handleCrash(ctx, sub, enrichment, err)
	}

	// 5. PARSE & SAVE
	s.updateStatus(ctx, enrichment, "Finalizing profile...", 90)

	var finalProfile map[string]interface{}
	cleanJson := s.cleanJsonBlock(resp.Content)

	// CRITICAL FIX: Fail explicitly on JSON parse errors instead of creating error placeholders
	// Silent failures lead to corrupt data being passed to analysis, resulting in garbage PDFs
	if err := json.Unmarshal([]byte(cleanJson), &finalProfile); err != nil {
		log.Error().
			Err(err).
			Str("content", resp.Content).
			Str("submission_id", submissionID.String()).
			Msg("CRITICAL: LLM returned invalid JSON - failing enrichment job")

		// Return error to job handler for retry logic
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w (content: %s)", err, cleanJson)
	}

	// INJECT submitted_data section from the original submission form
	// This ensures all user-provided fields are preserved and visible in the dashboard
	finalProfile["submitted_data"] = s.buildSubmittedData(sub)

	enrichment.EnrichedData = JSONMap(finalProfile)
	enrichment.SourcesStatus = s.combineSources(technicalData.Sources, resp.Sources)

	// Save final state
	s.saveProfile(ctx, enrichment, nil)

	log.Info().Dur("duration", time.Since(startTime)).Msg("Enrichment Pipeline Success")
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

// --- TRANSIENT DATA COLLECTOR ---

type DomainMetadata struct {
	Domain      string   `json:"domain"`
	NameServers []string `json:"name_servers"`
}

type IPLocation struct {
	Country string `json:"country"`
	City    string `json:"city"`
}

// TransientData holds the temporary technical data before AI synthesis
type TransientData struct {
	DomainInfo DomainMetadata   `json:"domain_info"`
	MetaTags   scraper.MetaData `json:"meta_tags"` // Use the scraper type directly
	IPLocation IPLocation       `json:"ip_location"`
	Sources    []string         `json:"sources_used"`
}

func (s *Service) gatherTransientData(ctx context.Context, sub *submission.Submission) TransientData {
	data := TransientData{
		Sources: []string{},
	}

	// 1. Domain Analysis
	if sub.CompanyWebsite != nil && *sub.CompanyWebsite != "" {
		domain := *sub.CompanyWebsite
		dnsInfo := dns.Analyze(ctx, domain)

		data.DomainInfo = DomainMetadata{
			Domain:      domain,
			NameServers: dnsInfo.NameServers,
		}
		if dnsInfo.HasMX {
			data.Sources = append(data.Sources, "dns_validation_active")
		} else {
			data.Sources = append(data.Sources, "dns_validation_no_email")
		}

		// 2. Metadata Scraping
		scrapeCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
		defer cancel()

		meta, err := s.scraper.Scrape(scrapeCtx, domain)
		if err == nil {
			data.MetaTags = meta
			data.Sources = append(data.Sources, "website_scraper")
		} else {
			log.Warn().Err(err).Str("domain", domain).Msg("Scraping failed, proceeding with AI only")
			data.Sources = append(data.Sources, "scraper_failed")
		}
	}

	// 3. Location
	if sub.CompanyLocation != nil && *sub.CompanyLocation != "" {
		data.IPLocation = IPLocation{
			Country: *sub.CompanyLocation,
			City:    "User Provided",
		}
		data.Sources = append(data.Sources, "user_input")
	}

	return data
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

func (s *Service) combineSources(technical []string, ai []llm.Source) JSONMap {
	combined := make(JSONMap)
	for _, src := range technical {
		combined[src] = "success"
	}
	for _, src := range ai {
		combined[src.URL] = "ai_search_result"
	}
	return combined
}

func (s *Service) cleanJsonBlock(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return content
}
