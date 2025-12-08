package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"backend_v3/llm"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Hardcoded models for enrichment pipeline (via OpenRouter)
// Validated via SELIC test: Only Perplexity has real web search
// Claude :online suffix returns empty responses (Dec 2024)
const (
	// Stage 1: Perplexity for web search (ONLY model with real-time data)
	// SELIC test confirmed: Returns current 15% rate
	ModelStage1Search   = "perplexity/sonar-pro"
	ModelStage1Fallback = "perplexity/sonar"

	// Stage 2: Claude Sonnet 4.5 for reasoning and synthesis
	// NO :online suffix - use Perplexity for web data, Sonnet for reasoning
	// Cost: ~$0.01 per call vs $0.16 for :online
	ModelStage2Synthesis = "anthropic/claude-sonnet-4.5"
	ModelStage2Fallback  = "anthropic/claude-sonnet-4"
)

// Service handles stateless enrichment operations using a two-stage approach:
// Stage 1: Perplexity sonar-pro for web search and raw data gathering
// Stage 2: Claude Sonnet 4.5 for strategic reasoning and synthesis
// The caller is responsible for persisting the enriched data
type Service struct {
	llmClient *llm.Client
}

// NewService creates a new stateless enrichment service
func NewService(llmClient *llm.Client) *Service {
	return &Service{
		llmClient: llmClient,
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

// EnrichCompany uses two-stage enrichment:
// Stage 1: Perplexity searches the web for raw company data
// Stage 2: Claude Opus 4.5 validates and synthesizes with reasoning
// Returns enriched data - caller is responsible for saving to company table
func (s *Service) EnrichCompany(ctx context.Context, company *CompanyInput) (*EnrichedCompanyData, error) {
	log.Info().
		Str("company_id", company.ID.String()).
		Str("company_name", company.Name).
		Str("stage1_model", ModelStage1Search).
		Str("stage2_model", ModelStage2Synthesis).
		Msg("Starting two-stage company enrichment")

	// ==================== STAGE 1: Perplexity Web Search ====================
	log.Debug().Msg("Stage 1: Perplexity web search for raw data")

	searchPrompt := s.buildSearchPrompt(company)
	searchReq := llm.Request{
		Model: ModelStage1Search,
		SystemPrompt: `Você é um pesquisador de inteligência de mercado. Sua tarefa é encontrar informações factuais sobre empresas brasileiras.

INSTRUÇÕES:
1. Busque informações públicas e verificáveis
2. Inclua dados de: site oficial, LinkedIn, notícias recentes, Glassdoor, Reclame Aqui
3. Liste TODOS os concorrentes que encontrar
4. Inclua notícias dos últimos 12 meses
5. Retorne os dados de forma organizada (pode ser texto estruturado)
6. SEMPRE cite as fontes com URLs`,
		Messages:    []llm.Message{{Role: "user", Content: searchPrompt}},
		Temperature: 0.2, // Very low for factual accuracy
		MaxTokens:   3000,
	}

	searchResp, err := s.llmClient.CallWithFallback(ctx, &searchReq, ModelStage1Fallback)
	if err != nil {
		log.Error().
			Err(err).
			Str("company_id", company.ID.String()).
			Msg("Stage 1 (Perplexity search) failed")
		return nil, fmt.Errorf("perplexity search failed: %w", err)
	}

	rawData := searchResp.Content
	log.Debug().
		Int("raw_data_length", len(rawData)).
		Msg("Stage 1 complete: raw data gathered")

	// ==================== STAGE 2: Claude Sonnet 4.5 Synthesis ====================
	log.Debug().Msg("Stage 2: Claude Sonnet 4.5 data synthesis")

	synthesisPrompt := s.buildSynthesisPrompt(company, rawData)
	synthesisReq := llm.Request{
		Model: ModelStage2Synthesis,
		SystemPrompt: `Você é um extrator de dados. Sua ÚNICA tarefa é extrair dados da pesquisa e retornar JSON.

REGRA CRÍTICA DE INTEGRIDADE:
- Use APENAS dados do texto de pesquisa fornecido
- NUNCA use seu conhecimento prévio para "corrigir" ou substituir valores
- Se a pesquisa diz "SELIC = 15%", retorne 15%, mesmo que pareça diferente do que você "sabe"

REGRAS DE FORMATO:
1. Retorne APENAS JSON válido, SEM texto antes ou depois
2. JSON deve ser PLANO (sem objetos aninhados)
3. Arrays devem conter APENAS strings, NUNCA objetos
4. SEMPRE preencha: competitors, strengths, weaknesses (mínimo 3 itens cada)
5. Se dado não existir na pesquisa, use "N/A" ou infira do setor

FORMATO DOS ARRAYS (OBRIGATÓRIO):
- ✅ CORRETO: "competitors": ["Empresa A", "Empresa B"]
- ❌ ERRADO: "competitors": [{"nome": "Empresa A"}]
- ✅ CORRETO: "recent_news": ["Jan/2024: Notícia X", "Fev/2024: Notícia Y"]
- ❌ ERRADO: "recent_news": [{"data": "Jan/2024", "titulo": "Notícia X"}]`,
		Messages:    []llm.Message{{Role: "user", Content: synthesisPrompt}},
		Temperature: 0.1, // Very low for consistent JSON and data preservation
		MaxTokens:   4000,
	}

	synthesisResp, err := s.llmClient.CallWithFallback(ctx, &synthesisReq, ModelStage2Fallback)
	if err != nil {
		log.Error().
			Err(err).
			Str("company_id", company.ID.String()).
			Msg("Stage 2 (Opus synthesis) failed")
		return nil, fmt.Errorf("opus synthesis failed: %w", err)
	}

	// Parse synthesized response
	enrichedData, err := s.parseEnrichmentResponse(synthesisResp.Content)
	if err != nil {
		log.Error().
			Err(err).
			Str("company_id", company.ID.String()).
			Str("raw_response", synthesisResp.Content).
			Msg("Failed to parse synthesis response")
		return nil, fmt.Errorf("failed to parse synthesis response: %w", err)
	}

	log.Info().
		Str("company_id", company.ID.String()).
		Float64("confidence_score", enrichedData.ConfidenceScore).
		Int("sources_count", len(enrichedData.Sources)).
		Int("competitors_count", len(enrichedData.Competitors)).
		Int("strengths_count", len(enrichedData.Strengths)).
		Msg("Two-stage enrichment completed successfully")

	return enrichedData, nil
}

// buildSearchPrompt creates the Stage 1 prompt for Perplexity web search
func (s *Service) buildSearchPrompt(company *CompanyInput) string {
	prompt := fmt.Sprintf(`Pesquise informações sobre a empresa brasileira: %s`, company.Name)

	if company.Website != nil && *company.Website != "" {
		prompt += fmt.Sprintf("\nWebsite: %s", *company.Website)
	}
	if company.Industry != nil && *company.Industry != "" {
		prompt += fmt.Sprintf("\nSetor: %s", *company.Industry)
	}
	if company.Location != nil && *company.Location != "" {
		prompt += fmt.Sprintf("\nLocalização: %s", *company.Location)
	}

	prompt += `

ENCONTRE E LISTE:
1. **Dados Básicos**: CNPJ, razão social, fundação, sede, funcionários
2. **Negócio**: Produtos/serviços, modelo de negócio, público-alvo, proposta de valor
3. **Concorrentes**: Liste TODOS os concorrentes diretos e indiretos com breve descrição
4. **Mercado**: Tamanho do mercado, crescimento, tendências, regulamentações
5. **Notícias Recentes**: Eventos dos últimos 12 meses (lançamentos, funding, expansões)
6. **Reputação**: Notas no Glassdoor, Reclame Aqui, avaliações
7. **Executivos**: CEO, fundadores, liderança
8. **Forças e Fraquezas**: O que a empresa faz bem? Onde falha?
9. **Redes Sociais**: LinkedIn, Twitter/X

Seja específico e factual. Cite as fontes.`

	return prompt
}

// buildSynthesisPrompt creates the Stage 2 prompt for Gemini synthesis
func (s *Service) buildSynthesisPrompt(company *CompanyInput, rawData string) string {
	prompt := fmt.Sprintf(`# TAREFA: Sintetizar Dados em Análise Estratégica

## EMPRESA
Nome: %s`, company.Name)

	if company.Industry != nil && *company.Industry != "" {
		prompt += fmt.Sprintf("\nSetor: %s", *company.Industry)
	}

	prompt += fmt.Sprintf(`

## DADOS BRUTOS DA PESQUISA (Perplexity)
%s

## INSTRUÇÕES

Analise os dados acima e retorne um JSON estruturado com análise estratégica.

IMPORTANTE:
- Extraia TODOS os concorrentes mencionados
- Identifique forças e fraquezas baseado nos dados
- Infira oportunidades e ameaças do contexto de mercado
- Se algum dado não existir, use seu conhecimento do setor para inferir

## FORMATO DE RESPOSTA (JSON ESTRITO)

Retorne APENAS o JSON abaixo, preenchido com dados reais:

{
  "cnpj": "XX.XXX.XXX/XXXX-XX",
  "website": "https://...",
  "industry": "Setor",
  "company_size": "Micro/Pequena/Média/Grande",
  "location": "Cidade, Estado",
  "target_market": "B2B/B2C/B2B2C",
  "funding_stage": "Bootstrap/Seed/Series A/B/C",
  "foundation_year": "YYYY",
  "legal_name": "Razão Social",
  "headquarters": "Cidade, Estado",
  "sector": "Subsetor",
  "target_audience": "Público-alvo",
  "value_proposition": "Proposta de valor",
  "employees_range": "50-100",
  "revenue_estimate": "R$ XM - YM/ano",
  "business_model": "SaaS/Marketplace/etc",
  "market_share_status": "Líder/Desafiador/Nicho",
  "digital_maturity": 7,

  "main_products": ["Produto 1", "Produto 2"],
  "service_areas": ["Brasil", "LATAM"],
  "key_partnerships": ["Parceiro 1"],
  "recent_news": ["Dez/2024: Notícia 1", "Nov/2024: Notícia 2"],
  "key_executives": ["Nome - Cargo"],
  "company_history": "Breve história",
  "customer_segments": ["Segmento 1", "Segmento 2"],
  "pricing_model": "Assinatura/Uso/Freemium",
  "market_position": "Posicionamento",
  "unique_selling_points": ["USP 1", "USP 2"],

  "competitors": ["Concorrente 1", "Concorrente 2", "Concorrente 3"],
  "competitor_details": ["Conc 1: descrição", "Conc 2: descrição"],
  "competitive_advantage": "Vantagem principal",
  "market_share": "X%% estimado",

  "strengths": ["Força 1", "Força 2", "Força 3"],
  "weaknesses": ["Fraqueza 1", "Fraqueza 2"],
  "opportunities": ["Oportunidade 1", "Oportunidade 2"],
  "threats": ["Ameaça 1", "Ameaça 2"],
  "strategic_challenges": ["Desafio 1"],

  "industry_growth_rate": "+X%% CAGR",
  "industry_trends": ["Tendência 1", "Tendência 2"],
  "regulatory_context": "Marco regulatório",
  "market_concentration": "Fragmentado/Concentrado",

  "tam_estimate": "R$ XXB",
  "sam_estimate": "R$ XXB",
  "som_estimate": "R$ XXM",

  "linkedin_url": "https://linkedin.com/company/...",
  "twitter_handle": "@empresa",

  "confidence_score": 75,
  "sources": ["fonte1.com", "fonte2.com"]
}`, rawData)

	return prompt
}

// parseEnrichmentResponse parses the LLM response into EnrichedCompanyData
// Handles flexible JSON structures where models may return objects instead of strings
func (s *Service) parseEnrichmentResponse(content string) (*EnrichedCompanyData, error) {
	// Clean and extract JSON
	cleanJSON := strings.TrimSpace(content)

	// Remove markdown code blocks if present
	if strings.HasPrefix(cleanJSON, "```json") {
		cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
	}
	if strings.HasPrefix(cleanJSON, "```") {
		cleanJSON = strings.TrimPrefix(cleanJSON, "```")
	}
	if strings.HasSuffix(cleanJSON, "```") {
		cleanJSON = strings.TrimSuffix(cleanJSON, "```")
	}
	cleanJSON = strings.TrimSpace(cleanJSON)

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

	// First try strict parsing
	var result EnrichedCompanyData
	if err := json.Unmarshal([]byte(cleanJSON), &result); err != nil {
		// Fallback: try flexible parsing with intermediate struct
		result, flexErr := parseFlexibleJSON(cleanJSON)
		if flexErr != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w (content: %.500s)", err, cleanJSON)
		}
		return result, nil
	}

	// Validate confidence score
	if result.ConfidenceScore < 0 || result.ConfidenceScore > 100 {
		result.ConfidenceScore = 50 // Default to medium confidence
	}

	return &result, nil
}

// parseFlexibleJSON handles JSON where arrays may contain objects instead of strings
// Also handles nested structures where data might be under "empresa", "produtos", etc.
func parseFlexibleJSON(jsonStr string) (*EnrichedCompanyData, error) {
	// Parse into generic map first
	var rawMap map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &rawMap); err != nil {
		return nil, err
	}

	// Flatten nested structures (e.g., "empresa": {...} -> top level)
	flatMap := flattenNestedStructure(rawMap)

	// Convert back to JSON with normalized arrays
	normalized := normalizeArrayFields(flatMap)

	// Marshal and unmarshal into final struct
	normalizedJSON, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}

	var result EnrichedCompanyData
	if err := json.Unmarshal(normalizedJSON, &result); err != nil {
		return nil, err
	}

	// Validate confidence score
	if result.ConfidenceScore < 0 || result.ConfidenceScore > 100 {
		result.ConfidenceScore = 50
	}

	return &result, nil
}

// flattenNestedStructure extracts data from nested objects like "empresa", "sede"
func flattenNestedStructure(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Field mappings from Portuguese nested keys to our flat structure
	fieldMappings := map[string]string{
		"nome":             "legal_name",
		"razao_social":     "legal_name",
		"ano_fundacao":     "foundation_year",
		"fundacao":         "foundation_year",
		"numero_funcionarios": "employees_range",
		"funcionarios":     "employees_range",
		"cidade":           "location",
		"sede":             "headquarters",
		"setor":            "industry",
		"descricao":        "business_model",
		"modelo_negocio":   "business_model",
		"produtos":         "main_products",
		"produtos_servicos": "main_products",
		"noticias":         "recent_news",
		"noticias_recentes": "recent_news",
		"noticias_2024":    "recent_news",
		"executivos":       "key_executives",
		"executivos_chave": "key_executives",
		"concorrentes":     "competitors",
		"forcas":           "strengths",
		"fraquezas":        "weaknesses",
		"oportunidades":    "opportunities",
		"ameacas":          "threats",
		"fontes":           "sources",
	}

	// Process each key
	for key, value := range m {
		// Check if it's a nested object we should flatten
		if nested, ok := value.(map[string]interface{}); ok {
			// Special handling for known nested containers
			if key == "empresa" || key == "company" || key == "dados" || key == "data" {
				// Merge nested fields into result
				for nestedKey, nestedValue := range nested {
					targetKey := nestedKey
					if mapped, ok := fieldMappings[nestedKey]; ok {
						targetKey = mapped
					}
					// Handle nested sede object
					if nestedKey == "sede" {
						if sedeObj, ok := nestedValue.(map[string]interface{}); ok {
							if cidade, ok := sedeObj["cidade"].(string); ok {
								if estado, ok := sedeObj["estado"].(string); ok {
									result["headquarters"] = fmt.Sprintf("%s, %s", cidade, estado)
								} else {
									result["headquarters"] = cidade
								}
							}
							continue
						}
					}
					result[targetKey] = nestedValue
				}
				continue
			}
		}

		// Map Portuguese field names to English
		targetKey := key
		if mapped, ok := fieldMappings[key]; ok {
			targetKey = mapped
		}
		result[targetKey] = value
	}

	return result
}

// normalizeArrayFields converts object arrays to string arrays
func normalizeArrayFields(m map[string]interface{}) map[string]interface{} {
	arrayFields := []string{
		"main_products", "service_areas", "tech_stack", "certifications",
		"key_partnerships", "recent_news", "key_executives", "customer_segments",
		"unique_selling_points", "competitors", "competitor_details",
		"strengths", "weaknesses", "opportunities", "threats",
		"strategic_challenges", "industry_trends", "sources",
	}

	for _, field := range arrayFields {
		if arr, ok := m[field].([]interface{}); ok {
			stringArr := make([]string, 0, len(arr))
			for _, item := range arr {
				stringArr = append(stringArr, objectToString(item))
			}
			m[field] = stringArr
		}
	}

	return m
}

// objectToString converts any value to a string representation
func objectToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case map[string]interface{}:
		// Try common field patterns for news/product objects
		if title, ok := val["titulo"].(string); ok {
			if date, ok := val["data"].(string); ok {
				return fmt.Sprintf("%s: %s", date, title)
			}
			return title
		}
		if name, ok := val["nome"].(string); ok {
			if desc, ok := val["descricao"].(string); ok {
				return fmt.Sprintf("%s - %s", name, desc)
			}
			return name
		}
		if name, ok := val["name"].(string); ok {
			if desc, ok := val["description"].(string); ok {
				return fmt.Sprintf("%s - %s", name, desc)
			}
			return name
		}
		// Generic: join all string values
		var parts []string
		for _, v := range val {
			if s, ok := v.(string); ok {
				parts = append(parts, s)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, " - ")
		}
		// Last resort: marshal to JSON
		b, _ := json.Marshal(val)
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}
