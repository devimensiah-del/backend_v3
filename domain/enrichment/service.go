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

// EnrichCompany calls the enrichment model with web search to fill company fields
// Uses Gemini 3 Pro with :online suffix for Exa-powered web search
// Returns enriched data - caller is responsible for saving to company table
func (s *Service) EnrichCompany(ctx context.Context, company *CompanyInput) (*EnrichedCompanyData, error) {
	log.Info().
		Str("company_id", company.ID.String()).
		Str("company_name", company.Name).
		Str("model", s.preSearchCfg.Model).
		Msg("Starting company enrichment with web search")

	// 1. Identify missing fields
	missingFields := s.identifyMissingFields(company)
	log.Debug().
		Str("missing_fields", missingFields).
		Msg("Identified missing fields")

	// 2. Build enrichment prompt
	prompt := s.buildEnrichmentPrompt(company, missingFields)

	// 3. Call enrichment model (Gemini with web search)
	req := llm.Request{
		Model: s.preSearchCfg.Model,
		SystemPrompt: `Você é um analista de inteligência de mercado especializado em pesquisa corporativa.

REGRAS ABSOLUTAS:
1. Retorne APENAS JSON válido, sem texto antes ou depois
2. SEMPRE preencha competitors, strengths, weaknesses com dados reais (NUNCA arrays vazios)
3. Use a busca web para encontrar informações atualizadas
4. Se não encontrar dados específicos da empresa, use dados do SETOR como proxy
5. Inclua URLs das fontes consultadas`,
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		Temperature: 0.3, // Low temperature for consistent results
		MaxTokens:   4000, // Increased for comprehensive output
	}

	resp, err := s.llmClient.CallWithFallback(ctx, &req, s.preSearchCfg.FallbackModel)
	if err != nil {
		log.Error().
			Err(err).
			Str("company_id", company.ID.String()).
			Str("model", s.preSearchCfg.Model).
			Msg("Enrichment failed")
		return nil, fmt.Errorf("enrichment failed: %w", err)
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

// buildEnrichmentPrompt builds a comprehensive prompt for company enrichment
func (s *Service) buildEnrichmentPrompt(company *CompanyInput, missingFields string) string {
	prompt := fmt.Sprintf(`# TAREFA: Pesquisa Profunda de Inteligência Corporativa

Você é um analista de inteligência de mercado sênior. Sua tarefa é pesquisar EXAUSTIVAMENTE a empresa abaixo e retornar dados estruturados para análise estratégica.

## EMPRESA ALVO
**Nome:** %s`, company.Name)

	if company.CNPJ != nil && *company.CNPJ != "" {
		prompt += fmt.Sprintf("\n**CNPJ:** %s", *company.CNPJ)
	}
	if company.Website != nil && *company.Website != "" {
		prompt += fmt.Sprintf("\n**Website:** %s", *company.Website)
	}
	if company.Industry != nil && *company.Industry != "" {
		prompt += fmt.Sprintf("\n**Setor:** %s", *company.Industry)
	}
	if company.Location != nil && *company.Location != "" {
		prompt += fmt.Sprintf("\n**Localização:** %s", *company.Location)
	}

	prompt += fmt.Sprintf(`

## CAMPOS PENDENTES
%s

## INSTRUÇÕES DE PESQUISA

### OBRIGATÓRIO - Pesquise ATIVAMENTE:
1. **Concorrentes**: Liste 3-5 concorrentes REAIS com nomes específicos. Pesquise "concorrentes de [empresa]" ou "alternativas a [empresa]"
2. **Forças**: Liste 3-5 pontos fortes ESPECÍFICOS (não genéricos). Ex: "Maior market share no Brasil", "Patente única de X"
3. **Fraquezas**: Liste 2-4 fraquezas ou desafios REAIS. Ex: "Presença limitada no Nordeste", "Alta rotatividade reportada"
4. **Oportunidades**: Liste 2-3 oportunidades de mercado específicas
5. **Ameaças**: Liste 2-3 ameaças competitivas ou regulatórias reais
6. **Produtos/Serviços**: Liste os principais produtos ou serviços oferecidos
7. **Notícias Recentes**: Liste 2-3 notícias ou eventos recentes (últimos 12 meses)

### IMPORTANTE:
- NÃO retorne arrays vazios [] para competitors, strengths, weaknesses - sempre pesquise e preencha
- Use dados REAIS e VERIFICÁVEIS de fontes públicas
- Se a empresa for pequena/nova, pesquise o setor para inferir concorrentes e tendências
- Inclua URLs das fontes no campo "sources"

## FORMATO DE RESPOSTA (JSON ESTRITO)

Retorne APENAS o JSON abaixo, sem texto adicional:

{
  "cnpj": "XX.XXX.XXX/XXXX-XX",
  "website": "https://...",
  "industry": "Setor específico",
  "company_size": "Micro/Pequena/Média/Grande",
  "location": "Cidade, Estado",
  "target_market": "B2B/B2C/B2B2C com descrição",
  "funding_stage": "Bootstrap/Seed/Series A/B/C/IPO",
  "annual_revenue_min": 1000000,
  "annual_revenue_max": 5000000,
  "foundation_year": "YYYY",
  "legal_name": "Razão Social Ltda",
  "headquarters": "Cidade, Estado, Brasil",
  "sector": "Subsetor detalhado",
  "target_audience": "Descrição do público-alvo ideal",
  "value_proposition": "Proposta de valor única da empresa",
  "employees_range": "50-100",
  "revenue_estimate": "R$ 1M - 5M/ano",
  "business_model": "SaaS B2B / Marketplace / E-commerce / etc",
  "market_share_status": "Líder/Desafiador/Seguidor/Nicho",
  "digital_maturity": 7,

  "main_products": ["Produto A - descrição", "Serviço B - descrição"],
  "service_areas": ["Brasil", "América Latina"],
  "tech_stack": ["React", "AWS", "Python"],
  "certifications": ["ISO 9001", "SOC 2"],
  "key_partnerships": ["Parceiro estratégico A", "Integração com B"],
  "recent_news": ["Dez/2024: Lançou produto X", "Nov/2024: Expandiu para região Y"],
  "key_executives": ["João Silva - CEO", "Maria Santos - CTO"],
  "company_history": "Fundada em YYYY, a empresa começou como... cresceu para...",
  "culture_values": "Cultura de inovação focada em...",
  "esg_initiatives": "Iniciativas de sustentabilidade e responsabilidade social",
  "customer_segments": ["PMEs de tecnologia", "Grandes varejistas"],
  "pricing_model": "Assinatura mensal / Por uso / Freemium",
  "market_position": "Posicionamento como líder em X para o segmento Y",
  "unique_selling_points": ["Diferencial 1", "Diferencial 2"],

  "competitors": ["Concorrente Real 1", "Concorrente Real 2", "Concorrente Real 3"],
  "competitor_details": ["Conc. 1: líder em X, forte em Y", "Conc. 2: foco em PMEs, preço baixo"],
  "competitive_advantage": "Principal vantagem competitiva da empresa",
  "market_share": "X%% do mercado (ou estimativa qualitativa)",

  "strengths": ["Força específica 1", "Força específica 2", "Força específica 3"],
  "weaknesses": ["Fraqueza específica 1", "Fraqueza específica 2"],
  "opportunities": ["Oportunidade de mercado 1", "Oportunidade 2"],
  "threats": ["Ameaça competitiva 1", "Ameaça regulatória 2"],
  "strategic_challenges": ["Desafio estratégico principal"],

  "industry_growth_rate": "+X%% CAGR (20XX-20XX)",
  "industry_trends": ["Tendência 1", "Tendência 2", "Tendência 3"],
  "regulatory_context": "Marco regulatório relevante (LGPD, Anvisa, etc)",
  "market_concentration": "Fragmentado/Concentrado/Oligopólio",

  "tam_estimate": "R$ XXB (mercado total)",
  "sam_estimate": "R$ XXB (mercado endereçável)",
  "som_estimate": "R$ XXM (mercado obtível)",

  "linkedin_url": "https://linkedin.com/company/...",
  "twitter_handle": "@empresa",

  "confidence_score": 85,
  "sources": ["https://fonte1.com", "https://fonte2.com"]
}`, missingFields)

	return prompt
}

// parseEnrichmentResponse parses the LLM response into EnrichedCompanyData
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
