package enrichment

import (
	"context"
	"fmt"

	"backend_v3/jina"
	"backend_v3/llm"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// SERVICE
// =============================================================================

// Service handles enrichment operations with 3-step approach:
// Step 1: Basic Info (automatic at company creation)
// Step 2: Business Model (human-triggered)
// Step 3: Competitive Intel (human-triggered)
//
// Each step uses: Perplexity (search) → Gemini 3 Flash (JSON formatting)
// Step 2 optionally uses Jina Reader to crawl company website first
type Service struct {
	llmClient  *llm.Client
	jinaClient *jina.Client
}

// NewService creates a new enrichment service
func NewService(llmClient *llm.Client) *Service {
	return &Service{
		llmClient:  llmClient,
		jinaClient: jina.NewClient(),
	}
}

// NewServiceWithJina creates a new enrichment service with a custom Jina client (for testing)
func NewServiceWithJina(llmClient *llm.Client, jinaClient *jina.Client) *Service {
	return &Service{
		llmClient:  llmClient,
		jinaClient: jinaClient,
	}
}

// CompanyInput represents the input data for enrichment (used by Step 1)
type CompanyInput struct {
	ID       uuid.UUID
	Name     string
	CNPJ     *string
	Website  *string
	Industry *string
	Location *string
}

// EnrichmentContext contains all available company data for enrichment prompts
// Used by Step 2 and Step 3 - passes ALL company fields instead of separate step data
type EnrichmentContext struct {
	// Core identity
	ID        uuid.UUID
	Name      string
	LegalName *string
	TradeName *string
	CNPJ      *string
	Website   *string

	// Business context
	BusinessModel    *string
	Industry         *string
	Sector           *string
	MainProducts     []string
	PricingModel     *string
	TargetMarket     *string // B2B, B2C, B2B2C
	TargetAudience   *string
	ValueProposition *string

	// Location & size
	Location          *string
	Headquarters      *string
	EmployeesRange    *string
	GeographicRegions []string
	ServiceAreas      []string

	// Additional context
	CompanySize         *string
	FundingStage        *string
	FoundationYear      *string
	CustomerSegments    []string
	UniqueSellingPoints []string
	KeyExecutives       []string
	LinkedInURL         *string
}

// =============================================================================
// STEP 1: BASIC INFO
// =============================================================================

// ExecuteStep1 runs Step 1 enrichment: Basic Info
// Fields: razão social, fundação, sede, funcionários, website, redes sociais, executivos
// Note: CNPJ verification is done in Step 2, not Step 1 (admin reviews Step 1 first)
func (s *Service) ExecuteStep1(ctx context.Context, company *CompanyInput) (*Step1BasicInfo, error) {
	log.Info().
		Str("company_id", company.ID.String()).
		Str("company_name", company.Name).
		Str("step", "1-basic-info").
		Msg("Starting Step 1 enrichment")

	// Stage 1: Perplexity search for basic company info
	searchPrompt := BuildStep1SearchPrompt(company)

	rawData, err := s.executeSearch(ctx, Step1SearchSystemPrompt, searchPrompt)
	if err != nil {
		log.Error().Err(err).Str("company_id", company.ID.String()).Msg("Step 1 search failed")
		return nil, fmt.Errorf("step 1 search failed: %w", err)
	}

	log.Debug().
		Int("raw_data_length", len(rawData)).
		Msg("Step 1 search complete")

	// Stage 2: Gemini JSON formatting
	formatPrompt := BuildFormatPrompt("Dados Básicos", rawData, Step1JSONTemplate)
	formattedJSON, err := s.executeFormat(ctx, Step1FormatSystemPrompt, formatPrompt)
	if err != nil {
		log.Error().Err(err).Str("company_id", company.ID.String()).Msg("Step 1 format failed")
		return nil, fmt.Errorf("step 1 format failed: %w", err)
	}

	// Parse response
	result, err := ParseStep1Response(formattedJSON)
	if err != nil {
		log.Error().Err(err).Str("company_id", company.ID.String()).Str("raw", formattedJSON).Msg("Step 1 parse failed")
		return nil, fmt.Errorf("step 1 parse failed: %w", err)
	}

	log.Info().
		Str("company_id", company.ID.String()).
		Float64("confidence", result.ConfidenceScore).
		Int("sources_count", len(result.Sources)).
		Msg("Step 1 enrichment completed")

	return result, nil
}

// =============================================================================
// CNPJ VERIFICATION (called from Step 2)
// =============================================================================

// CNPJData contains parsed data from CNPJ registry
type CNPJData struct {
	TradeName     *string  `json:"trade_name,omitempty"`
	Phone         *string  `json:"phone,omitempty"`
	Email         *string  `json:"email,omitempty"`
	CNAEPrimary   *string  `json:"cnae_primary,omitempty"`
	CNAECodes     []string `json:"cnae_codes,omitempty"`
	CapitalSocial *string  `json:"capital_social,omitempty"`
	Partners      []string `json:"partners,omitempty"`
	// Also include basic fields that might be more accurate from registry
	LegalName      *string `json:"legal_name,omitempty"`
	FoundationYear *string `json:"foundation_year,omitempty"`
	Headquarters   *string `json:"headquarters,omitempty"`
}

// FetchAndParseCNPJ fetches CNPJ registry data and parses it using LLM
// Returns nil if CNPJ is not provided or fetch fails (non-blocking)
func (s *Service) FetchAndParseCNPJ(ctx context.Context, cnpj string) (*CNPJData, error) {
	if cnpj == "" || s.jinaClient == nil {
		return nil, nil
	}

	log.Info().
		Str("cnpj", cnpj).
		Msg("Fetching CNPJ registry data from casadosdados")

	// Fetch raw content from casadosdados via Jina
	cnpjContent := s.jinaClient.FetchCNPJDataSafe(ctx, cnpj)
	if cnpjContent == nil || cnpjContent.Content == "" {
		log.Warn().Str("cnpj", cnpj).Msg("Failed to fetch CNPJ registry content")
		return nil, nil
	}

	log.Info().
		Str("cnpj", cnpj).
		Int("content_length", len(cnpjContent.Content)).
		Msg("CNPJ registry content fetched, parsing with LLM")

	// Use LLM to extract structured data from raw content
	formatPrompt := fmt.Sprintf(`Extraia os dados do CNPJ abaixo para JSON:

%s

Retorne APENAS JSON válido com os campos:
{
  "legal_name": "Razão Social ou null",
  "trade_name": "Nome Fantasia ou null",
  "foundation_year": "YYYY (extrair da data de abertura) ou null",
  "headquarters": "Cidade, Estado ou null",
  "phone": "telefone ou null",
  "email": "email@empresa.com (em minúsculas) ou null",
  "cnae_primary": "código - descrição ou null",
  "cnae_codes": ["código - descrição", ...],
  "capital_social": "R$ X.XXX,XX ou null",
  "partners": ["Nome - Cargo - Data", ...]
}

IMPORTANTE:
- Converta nomes para Proper Case (ex: "JOAO SILVA" → "João Silva")
- Email sempre em minúsculas
- NÃO inclua cabeçalhos como "Sócios:" nos arrays`, cnpjContent.Content)

	formattedJSON, err := s.executeFormat(ctx, "Você é um extrator de dados JSON. Retorne APENAS JSON válido.", formatPrompt)
	if err != nil {
		log.Error().Err(err).Str("cnpj", cnpj).Msg("Failed to parse CNPJ data with LLM")
		return nil, nil // Non-blocking, return nil on error
	}

	// Parse the JSON response
	var result CNPJData
	if err := parseJSON(formattedJSON, &result); err != nil {
		log.Error().Err(err).Str("cnpj", cnpj).Str("raw", formattedJSON).Msg("Failed to unmarshal CNPJ data")
		return nil, nil
	}

	log.Info().
		Str("cnpj", cnpj).
		Bool("has_legal_name", result.LegalName != nil).
		Bool("has_partners", len(result.Partners) > 0).
		Int("cnae_count", len(result.CNAECodes)).
		Msg("CNPJ data parsed successfully")

	return &result, nil
}

// =============================================================================
// STEP 2: BUSINESS MODEL
// =============================================================================

// ExecuteStep2 runs Step 2 enrichment: Business Model
// Fields: modelo de negócio, produtos/serviços, público alvo, proposta de valor, região geográfica
// Uses EnrichmentContext with ALL company fields for richer context
// Optionally crawls company website via Jina Reader for richer context
func (s *Service) ExecuteStep2(ctx context.Context, enrichCtx *EnrichmentContext) (*Step2BusinessModel, error) {
	log.Info().
		Str("company_id", enrichCtx.ID.String()).
		Str("company_name", enrichCtx.Name).
		Str("step", "2-business-model").
		Msg("Starting Step 2 enrichment")

	// Stage 0: Try to crawl company website via Jina Reader (non-blocking)
	var websiteContent string
	websiteURL := ""
	if enrichCtx.Website != nil && *enrichCtx.Website != "" {
		websiteURL = *enrichCtx.Website
	}

	if websiteURL != "" && s.jinaClient != nil {
		log.Info().
			Str("company_id", enrichCtx.ID.String()).
			Str("website", websiteURL).
			Msg("Crawling company website via Jina Reader")

		websiteContent = s.jinaClient.ReadPageSafe(ctx, websiteURL)

		if websiteContent != "" {
			log.Info().
				Str("company_id", enrichCtx.ID.String()).
				Int("content_length", len(websiteContent)).
				Msg("Website content crawled successfully")
		}
	}

	// Stage 1: Perplexity search with full company context + website content
	rawData, err := s.executeSearch(ctx, Step2SearchSystemPrompt, BuildStep2SearchPrompt(enrichCtx, websiteContent))
	if err != nil {
		log.Error().Err(err).Str("company_id", enrichCtx.ID.String()).Msg("Step 2 search failed")
		return nil, fmt.Errorf("step 2 search failed: %w", err)
	}

	log.Debug().
		Int("raw_data_length", len(rawData)).
		Bool("had_website_content", websiteContent != "").
		Msg("Step 2 search complete")

	// Stage 2: Gemini JSON formatting
	formatPrompt := BuildFormatPrompt("Modelo de Negócio", rawData, Step2JSONTemplate)
	formattedJSON, err := s.executeFormat(ctx, Step2FormatSystemPrompt, formatPrompt)
	if err != nil {
		log.Error().Err(err).Str("company_id", enrichCtx.ID.String()).Msg("Step 2 format failed")
		return nil, fmt.Errorf("step 2 format failed: %w", err)
	}

	// Parse response
	result, err := ParseStep2Response(formattedJSON)
	if err != nil {
		log.Error().Err(err).Str("company_id", enrichCtx.ID.String()).Str("raw", formattedJSON).Msg("Step 2 parse failed")
		return nil, fmt.Errorf("step 2 parse failed: %w", err)
	}

	log.Info().
		Str("company_id", enrichCtx.ID.String()).
		Float64("confidence", result.ConfidenceScore).
		Int("sources_count", len(result.Sources)).
		Int("products_count", len(result.MainProducts)).
		Msg("Step 2 enrichment completed")

	return result, nil
}

// =============================================================================
// STEP 3: COMPETITIVE INTELLIGENCE
// =============================================================================

// ExecuteStep3 runs Step 3 enrichment: Competitive Intelligence
// Fields: concorrentes, informações do setor, reputação, notícias recentes
// Uses EnrichmentContext with ALL company fields for comprehensive competitor search
func (s *Service) ExecuteStep3(ctx context.Context, enrichCtx *EnrichmentContext) (*Step3CompetitiveIntel, error) {
	log.Info().
		Str("company_id", enrichCtx.ID.String()).
		Str("company_name", enrichCtx.Name).
		Str("step", "3-competitive-intel").
		Msg("Starting Step 3 enrichment")

	// Stage 1: Perplexity search with full company context
	rawData, err := s.executeSearch(ctx, Step3SearchSystemPrompt, BuildStep3SearchPrompt(enrichCtx))
	if err != nil {
		log.Error().Err(err).Str("company_id", enrichCtx.ID.String()).Msg("Step 3 search failed")
		return nil, fmt.Errorf("step 3 search failed: %w", err)
	}

	log.Debug().
		Int("raw_data_length", len(rawData)).
		Msg("Step 3 search complete")

	// Stage 2: Gemini JSON formatting
	formatPrompt := BuildFormatPrompt("Inteligência Competitiva", rawData, Step3JSONTemplate)
	formattedJSON, err := s.executeFormat(ctx, Step3FormatSystemPrompt, formatPrompt)
	if err != nil {
		log.Error().Err(err).Str("company_id", enrichCtx.ID.String()).Msg("Step 3 format failed")
		return nil, fmt.Errorf("step 3 format failed: %w", err)
	}

	// Parse response
	result, err := ParseStep3Response(formattedJSON)
	if err != nil {
		log.Error().Err(err).Str("company_id", enrichCtx.ID.String()).Str("raw", formattedJSON).Msg("Step 3 parse failed")
		return nil, fmt.Errorf("step 3 parse failed: %w", err)
	}

	log.Info().
		Str("company_id", enrichCtx.ID.String()).
		Float64("confidence", result.ConfidenceScore).
		Int("sources_count", len(result.Sources)).
		Int("competitors_count", len(result.Competitors)).
		Int("news_count", len(result.RecentNews)).
		Msg("Step 3 enrichment completed")

	return result, nil
}

// =============================================================================
// INTERNAL HELPERS
// =============================================================================

// executeSearch runs the Perplexity search stage
func (s *Service) executeSearch(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	req := llm.Request{
		Model:        ModelSearch,
		SystemPrompt: systemPrompt,
		Messages:     []llm.Message{{Role: "user", Content: userPrompt}},
		Temperature:  0.2, // Low for factual accuracy
		MaxTokens:    3000,
	}

	resp, err := s.llmClient.CallWithFallback(ctx, &req, ModelSearchFallback)
	if err != nil {
		return "", fmt.Errorf("perplexity search failed: %w", err)
	}

	return resp.Content, nil
}

// executeFormat runs the Gemini JSON formatting stage
func (s *Service) executeFormat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	req := llm.Request{
		Model:        ModelFormat,
		SystemPrompt: systemPrompt,
		Messages:     []llm.Message{{Role: "user", Content: userPrompt}},
		Temperature:  0.1, // Very low for consistent JSON
		MaxTokens:    4000,
	}

	resp, err := s.llmClient.CallWithFallback(ctx, &req, ModelFormatFallback)
	if err != nil {
		return "", fmt.Errorf("gemini format failed: %w", err)
	}

	return resp.Content, nil
}

// =============================================================================
// LEGACY SUPPORT
// =============================================================================

// EnrichCompany is DEPRECATED - use ExecuteStep1, ExecuteStep2, ExecuteStep3
// Kept for backward compatibility during migration
func (s *Service) EnrichCompany(ctx context.Context, company *CompanyInput) (*EnrichedCompanyData, error) {
	log.Warn().Msg("EnrichCompany is deprecated - use ExecuteStep1/2/3 instead")

	// Execute only Step 1 for backward compatibility
	step1, err := s.ExecuteStep1(ctx, company)
	if err != nil {
		return nil, err
	}

	// Convert Step1 data to legacy format
	return &EnrichedCompanyData{
		CNPJ:            step1.CNPJ,
		Website:         step1.Website,
		LegalName:       step1.LegalName,
		FoundationYear:  step1.FoundationYear.ToStringPtr(),
		Headquarters:    step1.Headquarters,
		EmployeesRange:  step1.EmployeesRange,
		LinkedInURL:     step1.LinkedInURL,
		TwitterHandle:   step1.TwitterHandle,
		KeyExecutives:   step1.KeyExecutives,
		ConfidenceScore: step1.ConfidenceScore,
		Sources:         step1.Sources,
	}, nil
}
