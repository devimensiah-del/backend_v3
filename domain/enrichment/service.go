package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"backend_v3/config"
	"backend_v3/llm"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Service handles stateless enrichment operations
// It calls Perplexity to gather company data and returns it to the caller
// The caller is responsible for persisting the enriched data
type Service struct {
	llmClient    *llm.Client
	preSearchCfg config.FrameworkConfig // Perplexity config
}

// NewService creates a new stateless enrichment service
func NewService(llmClient *llm.Client, preSearchCfg config.FrameworkConfig) *Service {
	return &Service{
		llmClient:    llmClient,
		preSearchCfg: preSearchCfg,
	}
}

// CompanyInput represents the input data for enrichment
type CompanyInput struct {
	ID       uuid.UUID
	Name     string
	CNPJ     *string
	Website  *string
	Industry *string
	Location *string
}

// EnrichCompany calls Perplexity to fill missing company fields
// Returns enriched data - caller is responsible for saving to company table
func (s *Service) EnrichCompany(ctx context.Context, company *CompanyInput) (*EnrichedCompanyData, error) {
	log.Info().
		Str("company_id", company.ID.String()).
		Str("company_name", company.Name).
		Msg("Starting Perplexity enrichment")

	// 1. Identify missing fields
	missingFields := s.identifyMissingFields(company)
	log.Debug().
		Str("missing_fields", missingFields).
		Msg("Identified missing fields")

	// 2. Build Perplexity prompt
	prompt := s.buildEnrichmentPrompt(company, missingFields)

	// 3. Call Perplexity (presearch model)
	req := llm.Request{
		Model:        s.preSearchCfg.Model,
		SystemPrompt: "You are a JSON-only Company Data Enrichment Expert. Always return valid JSON. Only fill fields where you have high confidence (>70%).",
		Messages:     []llm.Message{{Role: "user", Content: prompt}},
		Temperature:  0.3, // Low temperature for more consistent results
		MaxTokens:    2000,
	}

	resp, err := s.llmClient.CallWithFallback(ctx, &req, s.preSearchCfg.FallbackModel)
	if err != nil {
		log.Error().
			Err(err).
			Str("company_id", company.ID.String()).
			Msg("Perplexity enrichment failed")
		return nil, fmt.Errorf("perplexity enrichment failed: %w", err)
	}

	// 4. Parse response
	enrichedData, err := s.parseEnrichmentResponse(resp.Content)
	if err != nil {
		log.Error().
			Err(err).
			Str("company_id", company.ID.String()).
			Str("raw_response", resp.Content).
			Msg("Failed to parse enrichment response")
		return nil, fmt.Errorf("failed to parse enrichment response: %w", err)
	}

	log.Info().
		Str("company_id", company.ID.String()).
		Float64("confidence_score", enrichedData.ConfidenceScore).
		Int("sources_count", len(enrichedData.Sources)).
		Msg("Enrichment completed successfully")

	return enrichedData, nil
}

// identifyMissingFields returns a description of fields that need enrichment
func (s *Service) identifyMissingFields(company *CompanyInput) string {
	var missing []string

	if company.CNPJ == nil || *company.CNPJ == "" {
		missing = append(missing, "- CNPJ (Cadastro Nacional da Pessoa Jurídica)")
	}
	if company.Website == nil || *company.Website == "" {
		missing = append(missing, "- Website oficial")
	}
	if company.Industry == nil || *company.Industry == "" {
		missing = append(missing, "- Setor/Indústria (CNAE)")
	}
	if company.Location == nil || *company.Location == "" {
		missing = append(missing, "- Localização (Sede: cidade, estado)")
	}

	if len(missing) == 0 {
		return "Todos os campos principais estão preenchidos. Valide e aprofunde as informações existentes."
	}

	return "Campos faltantes (preencha apenas se tiver confiança >70%):\n" + strings.Join(missing, "\n")
}

// buildEnrichmentPrompt builds the prompt for Perplexity
func (s *Service) buildEnrichmentPrompt(company *CompanyInput, missingFields string) string {
	prompt := fmt.Sprintf(`Você é um Agente de Enriquecimento de Dados Corporativos.

EMPRESA:
Nome: %s`, company.Name)

	if company.CNPJ != nil && *company.CNPJ != "" {
		prompt += fmt.Sprintf("\nCNPJ: %s", *company.CNPJ)
	}
	if company.Website != nil && *company.Website != "" {
		prompt += fmt.Sprintf("\nWebsite: %s", *company.Website)
	}
	if company.Industry != nil && *company.Industry != "" {
		prompt += fmt.Sprintf("\nSetor: %s", *company.Industry)
	}
	if company.Location != nil && *company.Location != "" {
		prompt += fmt.Sprintf("\nLocalização: %s", *company.Location)
	}

	prompt += fmt.Sprintf(`

%s

INSTRUÇÕES:
1. Busque dados públicos confiáveis sobre esta empresa
2. Preencha APENAS os campos que faltam e onde você tem alta confiança (>70%%)
3. NÃO sobrescreva dados já fornecidos
4. Se não encontrar informação confiável, deixe o campo null
5. Liste as URLs das fontes usadas

Retorne JSON estrito (PT-BR):
{
  "cnpj": "XX.XXX.XXX/XXXX-XX ou null",
  "website": "https://... ou null",
  "industry": "Setor específico ou null",
  "company_size": "Micro/Pequena/Média/Grande ou null",
  "location": "Cidade, Estado ou null",
  "target_market": "B2B/B2C/descrição ou null",
  "funding_stage": "Estágio de funding ou null",
  "annual_revenue_min": 0.0,
  "annual_revenue_max": 0.0,
  "foundation_year": "YYYY ou null",
  "legal_name": "Razão Social ou null",
  "headquarters": "Cidade, Estado, País ou null",
  "sector": "Setor detalhado ou null",
  "target_audience": "Público-alvo ou null",
  "value_proposition": "Proposta de valor ou null",
  "employees_range": "Ex: 10-50 ou null",
  "revenue_estimate": "Ex: R$ 2M - 5M/ano ou null",
  "business_model": "Ex: B2B SaaS ou null",
  "market_share_status": "Líder/Desafiador/Nicho ou null",
  "digital_maturity": 7,
  "competitors": ["Concorrente 1", "Concorrente 2"],
  "strengths": ["Força 1", "Força 2"],
  "weaknesses": ["Fraqueza 1", "Fraqueza 2"],
  "industry_growth_rate": "+X%% CAGR ou null",
  "industry_trends": ["Tendência 1", "Tendência 2"],
  "regulatory_context": "Contexto regulatório relevante ou null",
  "market_concentration": "Fragmentado/Concentrado/Oligopólio ou null",
  "linkedin_url": "URL ou null",
  "twitter_handle": "@handle ou null",
  "confidence_score": 85,
  "sources": ["URL1", "URL2"]
}`, missingFields)

	return prompt
}

// parseEnrichmentResponse parses the Perplexity response into EnrichedCompanyData
func (s *Service) parseEnrichmentResponse(content string) (*EnrichedCompanyData, error) {
	// Clean and extract JSON
	cleanJSON := strings.TrimSpace(content)

	// Find JSON boundaries (handles conversational text before/after JSON)
	startObj := strings.Index(cleanJSON, "{")
	startArr := strings.Index(cleanJSON, "[")
	endObj := strings.LastIndex(cleanJSON, "}")
	endArr := strings.LastIndex(cleanJSON, "]")

	// Determine which structure we have (object vs array)
	start := -1
	end := -1
	if startObj != -1 && endObj != -1 {
		if startArr == -1 || startObj < startArr {
			start, end = startObj, endObj
		}
	}
	if startArr != -1 && endArr != -1 {
		if start == -1 || startArr < start {
			start, end = startArr, endArr
		}
	}

	if start != -1 && end != -1 && end > start {
		cleanJSON = cleanJSON[start : end+1]
	}

	var result EnrichedCompanyData
	if err := json.Unmarshal([]byte(cleanJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w (content: %s)", err, cleanJSON)
	}

	// Validate confidence score
	if result.ConfidenceScore < 0 || result.ConfidenceScore > 100 {
		result.ConfidenceScore = 50 // Default to medium confidence
	}

	return &result, nil
}
