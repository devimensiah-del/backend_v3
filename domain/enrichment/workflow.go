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

	// 3. DETECT GAPS
	missingFields := s.detectMissingFields(sub)

	// 4. AGENT EXECUTION
	s.updateStatus(ctx, enrichment, "Agent is synthesizing Intelligence Profile...", 40)

	prompt := llm.UnifiedEnrichmentPrompt
	prompt = strings.ReplaceAll(prompt, "{{COMPANY_NAME}}", sub.CompanyName)
	prompt = strings.ReplaceAll(prompt, "{{USER_CONTEXT}}", s.compileUserDossier(sub))
	prompt = strings.ReplaceAll(prompt, "{{TECHNICAL_CONTEXT}}", techContextString)
	prompt = strings.ReplaceAll(prompt, "{{MISSING_FIELDS}}", missingFields)

	agentReq := llm.Request{
		Model:        s.enrichmentCfg.Model,
		SystemPrompt: "You are a JSON-only Corporate Intelligence Agent.",
		Messages:     []llm.Message{{Role: "user", Content: prompt}},
		// IMPORTANT: Ensure your client.go maps "search" to the provider's specific tool (e.g., google_search)
		Tools:       []string{"search"},
		Temperature: s.enrichmentCfg.Temperature, // Use config temp
		MaxTokens:   s.enrichmentCfg.MaxTokens,   // Use config tokens
	}

	// Call LLM
	resp, err := s.llmClient.Call(ctx, &agentReq)
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

	enrichment.EnrichedData = JSONMap(finalProfile)
	enrichment.SourcesStatus = s.combineSources(technicalData.Sources, resp.Sources)

	// Save final state
	s.saveProfile(enrichment, nil)

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
	e.Finish()
	if err := s.repo.UpdateSystem(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *Service) saveProfile(e *Enrichment, err error) {
	if err != nil {
		e.Fail(err)
	}
	_ = context.Background()
}

func (s *Service) compileUserDossier(sub *submission.Submission) string {
	dossier := fmt.Sprintf("Nome: %s\n", sub.CompanyName)
	if sub.CompanyWebsite != nil {
		dossier += fmt.Sprintf("Site: %s\n", *sub.CompanyWebsite)
	}
	if sub.BusinessChallenge != "" {
		dossier += fmt.Sprintf("Desafio: %s\n", sub.BusinessChallenge)
	}
	return dossier
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
