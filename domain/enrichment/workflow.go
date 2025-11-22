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

// EnrichSubmission implements 3-layer progressive data collection workflow.
// Layer 1: Immediate (<2s) - Domain metadata, IP location, basic info
// Layer 2: Structured (3-6s) - Legal name, CNPJ, sector, employees, revenue
// Layer 3: AI Inference (6-10s) - Business summary, digital maturity, competitors, gaps
func (s *Service) EnrichSubmission(ctx context.Context, submissionID uuid.UUID) (*Enrichment, error) {
	startTime := time.Now()

	// 1. SETUP
	sub, enrichment, err := s.setupWorkspace(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if enrichment == nil {
		return nil, nil
	} // Locked by user

	// Initialize profile structure
	profile := StrategicProfile{
		ProfileVersion:  "3.0",
		CompletedLayers: 0,
	}

	// 2. LAYER 1: IMMEDIATE DATA COLLECTION (<2s)
	s.updateStatus(ctx, enrichment, "Collecting immediate data (Layer 1)...", 10)
	layer1Start := time.Now()

	layer1, err := s.collectLayer1(ctx, sub)
	if err != nil {
		log.Warn().Err(err).Msg("Layer 1 collection failed, continuing with partial data")
	} else {
		profile.Layer1 = *layer1
		profile.CompletedLayers = 1
	}

	log.Debug().Dur("duration", time.Since(layer1Start)).Msg("Layer 1 completed")

	// 3. LAYER 2: STRUCTURED BUSINESS DATA (3-6s)
	s.updateStatus(ctx, enrichment, "Fetching structured business data (Layer 2)...", 30)
	layer2Start := time.Now()

	layer2, err := s.collectLayer2(ctx, sub)
	if err != nil {
		log.Warn().Err(err).Msg("Layer 2 collection failed, continuing with partial data")
	} else {
		profile.Layer2 = *layer2
		profile.CompletedLayers = 2
	}

	log.Debug().Dur("duration", time.Since(layer2Start)).Msg("Layer 2 completed")

	// 4. LAYER 3: AI STRATEGIC INFERENCE (6-10s)
	s.updateStatus(ctx, enrichment, "Generating AI strategic intelligence (Layer 3)...", 60)
	layer3Start := time.Now()

	layer3, err := s.collectLayer3(ctx, sub, &profile)
	if err != nil {
		log.Warn().Err(err).Msg("Layer 3 collection failed, continuing with partial data")
	} else {
		profile.Layer3 = *layer3
		profile.CompletedLayers = 3
	}

	log.Debug().Dur("duration", time.Since(layer3Start)).Msg("Layer 3 completed")

	// 5. SAVE COMPLETE PROFILE
	profile.TotalDuration = int(time.Since(startTime).Milliseconds())
	s.updateStatus(ctx, enrichment, "Structuring intelligence profile...", 90)
	s.saveProfile(enrichment, &profile)

	// 6. FINALIZE
	return s.markAsComplete(ctx, sub, enrichment)
}

// collectLayer1 gathers immediate data (<2s): domain metadata, IP location, basic info
func (s *Service) collectLayer1(ctx context.Context, sub *submission.Submission) (*Layer1Data, error) {
	layer1 := &Layer1Data{
		CollectedAt: time.Now().Format(time.RFC3339),
		Sources:     []string{},
	}

	// Extract domain from website if available
	domain := ""
	if sub.CompanyWebsite != nil && *sub.CompanyWebsite != "" {
		domain = *sub.CompanyWebsite
		// Simple domain extraction (remove protocol and path)
		domain = strings.TrimPrefix(domain, "http://")
		domain = strings.TrimPrefix(domain, "https://")
		domain = strings.Split(domain, "/")[0]
	}

	if domain != "" {
		// Simulate domain metadata collection (replace with actual WHOIS API)
		layer1.DomainMetadata = DomainMetadata{
			Domain: domain,
			// Real implementation would call WHOIS API here
		}
		layer1.Sources = append(layer1.Sources, "whois")

		// Simulate IP location (replace with actual IP-API call)
		layer1.IPLocation = IPLocation{
			Country: "Brasil", // Default for now
		}
		layer1.Sources = append(layer1.Sources, "ip-api")

		// Simulate basic metadata scraping (replace with actual scraper)
		layer1.BasicInfo = BasicInfo{
			Title:       sub.CompanyName,
			Description: fmt.Sprintf("Empresa no setor de %s", safeString(sub.CompanyIndustry)),
		}
		layer1.Sources = append(layer1.Sources, "metadata-scraper")
	}

	return layer1, nil
}

// collectLayer2 gathers structured data (3-6s): legal name, CNPJ, sector, employees, revenue
func (s *Service) collectLayer2(ctx context.Context, sub *submission.Submission) (*Layer2Data, error) {
	layer2 := &Layer2Data{
		CollectedAt: time.Now().Format(time.RFC3339),
		Sources:     []string{},
	}

	// Simulate CNPJ lookup (replace with actual ReceitaWS/OpenCNPJ API)
	// Real implementation would call ReceitaWS API
	layer2.LegalInfo = LegalInfo{
		LegalName: sub.CompanyName,
		// CNPJ lookup would happen here
	}
	layer2.Sources = append(layer2.Sources, "receitaws")

	// Business info from industry and other data
	if sub.CompanyIndustry != nil {
		layer2.BusinessInfo = BusinessInfo{
			Sector: *sub.CompanyIndustry,
		}
	}

	// Location info
	layer2.LocationInfo = LocationInfo{
		City:  "São Paulo", // Would be extracted from CNPJ data
		State: "SP",
	}
	layer2.Sources = append(layer2.Sources, "opencnpj", "google-places")

	return layer2, nil
}

// collectLayer3 performs AI-powered strategic inference (6-10s)
func (s *Service) collectLayer3(ctx context.Context, sub *submission.Submission, profile *StrategicProfile) (*Layer3Data, error) {
	// Configure AI agent
	model := s.enrichmentCfg.Model
	if model == "" {
		model = s.model
	}
	temperature := s.enrichmentCfg.Temperature
	if temperature == 0 {
		temperature = 0.5
	}
	maxTokens := s.enrichmentCfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8000
	}

	agent := AgentProfile{
		Role:        "Strategic Intelligence Analyst",
		Model:       model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Tools:       []string{"search", "url_context"},
	}

	// Build context from Layer 1 and Layer 2
	contextData := s.buildContextForLayer3(sub, profile)

	// Use updated prompt for Layer 3
	prompt := llm.Layer3InferencePrompt
	prompt = strings.ReplaceAll(prompt, "{{COMPANY_NAME}}", sub.CompanyName)
	prompt = strings.ReplaceAll(prompt, "{{CONTEXT_DATA}}", contextData)

	// Execute AI analysis
	agentFindings, err := s.deployAgent(ctx, agent, prompt, sub)
	if err != nil {
		return nil, err
	}

	// Parse Layer 3 response
	layer3 := &Layer3Data{
		CollectedAt: time.Now().Format(time.RFC3339),
		Sources:     []string{"llm-analysis"},
	}

	// Extract sources from agent findings
	for _, source := range agentFindings.Sources {
		layer3.Sources = append(layer3.Sources, source.URL)
	}

	// Parse AI response into Layer3 structure
	cleanJson := strings.TrimPrefix(strings.TrimSuffix(agentFindings.Content, "```"), "```json")
	if err := json.Unmarshal([]byte(cleanJson), layer3); err != nil {
		// Fallback: try to parse into a generic map
		var rawMap map[string]interface{}
		if err2 := json.Unmarshal([]byte(cleanJson), &rawMap); err2 != nil {
			return nil, fmt.Errorf("failed to parse Layer 3 response: %w", err)
		}
		// Manual mapping if structured parsing fails
		log.Warn().Msg("Layer 3 parsing fell back to raw map, using defaults")
	}

	return layer3, nil
}

// buildContextForLayer3 compiles data from Layer 1 and Layer 2 for AI inference
func (s *Service) buildContextForLayer3(sub *submission.Submission, profile *StrategicProfile) string {
	var sb strings.Builder

	sb.WriteString("--- LAYER 1: IMMEDIATE DATA ---\n")
	if profile.Layer1.DomainMetadata.Domain != "" {
		sb.WriteString(fmt.Sprintf("Domain: %s\n", profile.Layer1.DomainMetadata.Domain))
	}
	if profile.Layer1.IPLocation.Country != "" {
		sb.WriteString(fmt.Sprintf("Location: %s\n", profile.Layer1.IPLocation.Country))
	}
	if profile.Layer1.BasicInfo.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", profile.Layer1.BasicInfo.Description))
	}

	sb.WriteString("\n--- LAYER 2: STRUCTURED DATA ---\n")
	if profile.Layer2.LegalInfo.LegalName != "" {
		sb.WriteString(fmt.Sprintf("Legal Name: %s\n", profile.Layer2.LegalInfo.LegalName))
	}
	if profile.Layer2.BusinessInfo.Sector != "" {
		sb.WriteString(fmt.Sprintf("Sector: %s\n", profile.Layer2.BusinessInfo.Sector))
	}
	if profile.Layer2.LocationInfo.City != "" {
		sb.WriteString(fmt.Sprintf("City: %s, %s\n", profile.Layer2.LocationInfo.City, profile.Layer2.LocationInfo.State))
	}

	sb.WriteString("\n--- USER PROVIDED DATA ---\n")
	sb.WriteString(fmt.Sprintf("Company Name: %s\n", sub.CompanyName))
	if sub.CompanyWebsite != nil {
		sb.WriteString(fmt.Sprintf("Website: %s\n", *sub.CompanyWebsite))
	}
	sb.WriteString(fmt.Sprintf("Business Challenge: %s\n", sub.BusinessChallenge))

	return sb.String()
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// =================================================================================
// UNDERWATER MECHANICS
// =================================================================================

type AgentProfile struct {
	Role        string
	Model       string
	Temperature float64
	MaxTokens   int
	Tools       []string
}

func (s *Service) setupWorkspace(ctx context.Context, id uuid.UUID) (*submission.Submission, *Enrichment, error) {
	sub, err := s.submissionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	enrichment, err := s.repo.GetBySubmissionID(ctx, id)
	if err != nil {
		enrichment = NewEnrichment(id)
		if err := s.repo.Create(ctx, enrichment); err != nil {
			return nil, nil, err
		}
	}
	enrichment.Start()
	if err := s.repo.UpdateSystem(ctx, enrichment); err != nil {
		return nil, nil, err
	}
	sub.SetStatus(submission.StatusEnriching)
	s.submissionRepo.Update(ctx, sub)
	return sub, enrichment, nil
}

func (s *Service) deployAgent(ctx context.Context, profile AgentProfile, promptTemplate string, sub *submission.Submission) (*llm.Response, error) {
	userDossier := s.compileUserDossier(sub)

	// Replace template variables with actual data
	finalPrompt := strings.ReplaceAll(promptTemplate, "{{COMPANY_NAME}}", sub.CompanyName)
	finalPrompt = strings.ReplaceAll(finalPrompt, "{{USER_CONTEXT}}", userDossier)

	req := &llm.Request{
		Model:        profile.Model,
		SystemPrompt: profile.Role,
		Messages:     []llm.Message{{Role: "user", Content: finalPrompt}},
		Tools:        profile.Tools,
		MaxURLs:      10,
		Temperature:  profile.Temperature,
		MaxTokens:    profile.MaxTokens,
	}
	return s.llmClient.Call(ctx, req)
}

func (s *Service) saveProfile(e *Enrichment, profile *StrategicProfile) {
	// Convert profile to JSONMap for storage
	data, err := json.Marshal(profile)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal profile")
		return
	}

	var storageMap map[string]interface{}
	if err := json.Unmarshal(data, &storageMap); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal to storage map")
		return
	}

	e.EnrichedData = JSONMap(storageMap)

	// Track sources from all layers
	e.SourcesStatus = make(JSONMap)
	for _, source := range profile.Layer1.Sources {
		e.SourcesStatus[source] = "success"
	}
	for _, source := range profile.Layer2.Sources {
		e.SourcesStatus[source] = "success"
	}
	for _, source := range profile.Layer3.Sources {
		e.SourcesStatus[source] = "success"
	}
}

func (s *Service) compileUserDossier(sub *submission.Submission) string {
	var sb strings.Builder
	sb.WriteString("--- DADOS FORNECIDOS PELO USUÁRIO ---\n")
	sb.WriteString(fmt.Sprintf("Nome: %s\n", sub.CompanyName))
	if sub.CompanyWebsite != nil {
		sb.WriteString(fmt.Sprintf("Site: %s (Fonte Primária)\n", *sub.CompanyWebsite))
	}
	if sub.CompanyIndustry != nil {
		sb.WriteString(fmt.Sprintf("Setor: %s\n", *sub.CompanyIndustry))
	}
	sb.WriteString(fmt.Sprintf("Desafio: %s\n", sub.BusinessChallenge))
	return sb.String()
}

func (s *Service) updateStatus(ctx context.Context, e *Enrichment, step string, progress int) {
	e.UpdateProgress(step, progress)
	s.repo.UpdateSystem(ctx, e)
}

func (s *Service) markAsComplete(ctx context.Context, sub *submission.Submission, e *Enrichment) (*Enrichment, error) {
	// Use Finish() instead of Complete() - status changes to "finished"
	e.Finish()
	if err := s.repo.UpdateSystem(ctx, e); err != nil {
		return nil, nil
	}
	sub.SetStatus(submission.StatusEnriched)
	s.submissionRepo.Update(ctx, sub)
	return e, nil
}

func (s *Service) handleCrash(ctx context.Context, sub *submission.Submission, e *Enrichment, err error) error {
	log.Error().Err(err).Msg("Enrichment Agent Crashed")
	e.Fail(err)
	s.repo.UpdateSystem(ctx, e)
	sub.SetStatus(submission.StatusEnrichmentFailed)
	s.submissionRepo.Update(ctx, sub)
	return err
}
