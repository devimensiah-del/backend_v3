package enrichment

import (
	"backend_v3/domain/submission"
	"backend_v3/llm"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// EnrichSubmission is the workflow for the "Virtual Consultant".
// It sets up the agent, gives it a mission (Prompt), and executes the search.
// EnrichSubmission is the "Virtual Consultant" workflow.
func (s *Service) EnrichSubmission(ctx context.Context, submissionID uuid.UUID) (*Enrichment, error) {

	// 1. SETUP
	sub, enrichment, err := s.setupWorkspace(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if enrichment == nil {
		return nil, nil
	} // Locked by user

	// 2. DEFINIÇÃO DO AGENTE
	// NEW: Use framework-specific enrichment config
	// Fallback to s.model if enrichmentCfg not set (backward compatibility)
	model := s.enrichmentCfg.Model
	if model == "" {
		model = s.model
	}
	temperature := s.enrichmentCfg.Temperature
	if temperature == 0 {
		temperature = 0.5 // Default fallback
	}
	maxTokens := s.enrichmentCfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8000
	}

	agent := AgentProfile{
		Role:        "Strategic Data Enrichment Agent",
		Model:       model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Tools:       []string{"search", "url_context"},
	}

	// 3. MISSÃO
	missionInstructions := llm.StrategicEnrichmentPrompt

	// 4. EXECUÇÃO
	s.updateStatus(ctx, enrichment, "Agent scanning digital footprint...", 20)

	agentFindings, err := s.deployAgent(ctx, agent, missionInstructions, sub)
	if err != nil {
		return nil, s.handleCrash(ctx, sub, enrichment, err)
	}

	// 5. SALVAR
	s.updateStatus(ctx, enrichment, "Structuring intelligence profile...", 90)
	s.processAndSaveFindings(enrichment, agentFindings)

	// 6. FINALIZAR
	return s.markAsComplete(ctx, sub, enrichment)
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

func (s *Service) processAndSaveFindings(e *Enrichment, resp *llm.Response) {
	cleanJson := strings.TrimPrefix(strings.TrimSuffix(resp.Content, "```"), "```json")
	var profile StrategicProfile
	if err := json.Unmarshal([]byte(cleanJson), &profile); err != nil {
		var rawMap map[string]interface{}
		_ = json.Unmarshal([]byte(cleanJson), &rawMap)
		e.EnrichedData = JSONMap(rawMap)
	} else {
		data, _ := json.Marshal(profile)
		var storageMap map[string]interface{}
		_ = json.Unmarshal(data, &storageMap)
		e.EnrichedData = JSONMap(storageMap)
	}
	e.SourcesStatus = make(JSONMap)
	for _, source := range resp.Sources {
		e.SourcesStatus[source.URL] = "success"
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
	e.Complete()
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
