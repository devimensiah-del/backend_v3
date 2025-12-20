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

// CompanyInput represents the input data for enrichment
type CompanyInput struct {
	ID       uuid.UUID
	Name     string
	CNPJ     *string
	Website  *string
	Industry *string
	Location *string
}

// =============================================================================
// STEP 1: BASIC INFO
// =============================================================================

// ExecuteStep1 runs Step 1 enrichment: Basic Info
// Fields: CNPJ, razão social, fundação, sede, funcionários, website, redes sociais, executivos
// If CNPJ is provided, fetches official data from CNPJ registry (casadosdados) first
func (s *Service) ExecuteStep1(ctx context.Context, company *CompanyInput) (*Step1BasicInfo, error) {
	log.Info().
		Str("company_id", company.ID.String()).
		Str("company_name", company.Name).
		Str("step", "1-basic-info").
		Msg("Starting Step 1 enrichment")

	// Stage 0: Try to fetch CNPJ registry data if CNPJ is provided
	var cnpjData *jina.CNPJData
	if company.CNPJ != nil && *company.CNPJ != "" && s.jinaClient != nil {
		log.Info().
			Str("company_id", company.ID.String()).
			Str("cnpj", *company.CNPJ).
			Msg("Fetching CNPJ registry data from casadosdados")

		cnpjData = s.jinaClient.FetchCNPJDataSafe(ctx, *company.CNPJ)

		if cnpjData != nil {
			log.Info().
				Str("company_id", company.ID.String()).
				Str("legal_name", cnpjData.LegalName).
				Int("partners_count", len(cnpjData.Partners)).
				Msg("CNPJ registry data fetched successfully")
		}
	}

	// Stage 1: Perplexity search
	rawData, err := s.executeSearch(ctx, Step1SearchSystemPrompt, BuildStep1SearchPrompt(company))
	if err != nil {
		log.Error().Err(err).Str("company_id", company.ID.String()).Msg("Step 1 search failed")
		return nil, fmt.Errorf("step 1 search failed: %w", err)
	}

	log.Debug().
		Int("raw_data_length", len(rawData)).
		Bool("has_cnpj_data", cnpjData != nil).
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

	// Stage 3: Merge CNPJ registry data (official data takes priority)
	if cnpjData != nil {
		result = mergeCNPJData(result, cnpjData)
		result.CNPJVerified = true
		log.Info().
			Str("company_id", company.ID.String()).
			Msg("Merged CNPJ registry data into Step 1 result")
	}

	log.Info().
		Str("company_id", company.ID.String()).
		Float64("confidence", result.ConfidenceScore).
		Int("sources_count", len(result.Sources)).
		Bool("cnpj_verified", result.CNPJVerified).
		Msg("Step 1 enrichment completed")

	return result, nil
}

// mergeCNPJData merges official CNPJ registry data into Step1BasicInfo
// Registry data takes priority for official fields (legal name, address, partners, etc.)
// Perplexity data fills gaps (website, social links, etc.)
func mergeCNPJData(result *Step1BasicInfo, cnpjData *jina.CNPJData) *Step1BasicInfo {
	// Official data - always use if available
	if cnpjData.CNPJ != "" {
		result.CNPJ = &cnpjData.CNPJ
	}
	if cnpjData.LegalName != "" {
		result.LegalName = &cnpjData.LegalName
	}
	if cnpjData.TradeName != "" {
		result.TradeName = &cnpjData.TradeName
	}
	if cnpjData.FoundationYear != "" {
		result.FoundationYear = FlexibleString(cnpjData.FoundationYear)
	}

	// Address - build full address
	if cnpjData.Address != "" || cnpjData.City != "" {
		address := cnpjData.Address
		if cnpjData.City != "" && cnpjData.State != "" {
			if address != "" {
				address += " | "
			}
			address += cnpjData.City + ", " + cnpjData.State
		}
		if address != "" {
			result.Headquarters = &address
		}
	}

	// Contact info
	if cnpjData.Phone != "" {
		result.Phone = &cnpjData.Phone
	}
	if cnpjData.Email != "" {
		result.Email = &cnpjData.Email
	}

	// CNAE
	if cnpjData.CNAEPrimary != "" {
		result.CNAEPrimary = &cnpjData.CNAEPrimary
	}
	if len(cnpjData.CNAECodes) > 0 {
		result.CNAECodes = cnpjData.CNAECodes
	}

	// Financial
	if cnpjData.CapitalSocial != "" {
		result.CapitalSocial = &cnpjData.CapitalSocial
	}

	// Partners - use registry data as more accurate than key_executives guess
	if len(cnpjData.Partners) > 0 {
		result.Partners = cnpjData.Partners
	}

	// Company type to employees range mapping
	if cnpjData.CompanyType != "" && (result.EmployeesRange == nil || *result.EmployeesRange == "") {
		switch cnpjData.CompanyType {
		case "Micro Empresa":
			me := "1-9 funcionários (Micro Empresa)"
			result.EmployeesRange = &me
		case "Empresa de Pequeno Porte":
			epp := "10-49 funcionários (Pequeno Porte)"
			result.EmployeesRange = &epp
		case "Empresa de Médio Porte":
			emp := "50-99 funcionários (Médio Porte)"
			result.EmployeesRange = &emp
		}
	}

	// Add casadosdados as source if not already present
	hasCasadosdados := false
	for _, source := range result.Sources {
		if source == "casadosdados.com.br" {
			hasCasadosdados = true
			break
		}
	}
	if !hasCasadosdados {
		result.Sources = append(result.Sources, "casadosdados.com.br")
	}

	return result
}

// =============================================================================
// STEP 2: BUSINESS MODEL
// =============================================================================

// BackfillCNPJData fetches and merges CNPJ registry data into existing Step 1 data
// Used when CNPJ was discovered during Step 1 enrichment (not provided upfront)
// Returns nil if no backfill is needed or if the fetch fails (non-blocking)
func (s *Service) BackfillCNPJData(ctx context.Context, step1Data *Step1BasicInfo) *Step1BasicInfo {
	// Skip if already verified or no CNPJ available
	if step1Data == nil || step1Data.CNPJVerified {
		return nil
	}
	if step1Data.CNPJ == nil || *step1Data.CNPJ == "" {
		return nil
	}
	if s.jinaClient == nil {
		return nil
	}

	log.Info().
		Str("cnpj", *step1Data.CNPJ).
		Msg("Backfilling CNPJ registry data (discovered during Step 1)")

	cnpjData := s.jinaClient.FetchCNPJDataSafe(ctx, *step1Data.CNPJ)
	if cnpjData == nil {
		log.Warn().
			Str("cnpj", *step1Data.CNPJ).
			Msg("CNPJ backfill failed, continuing without registry data")
		return nil
	}

	// Merge and mark as verified
	updatedStep1 := mergeCNPJData(step1Data, cnpjData)
	updatedStep1.CNPJVerified = true

	log.Info().
		Str("cnpj", *step1Data.CNPJ).
		Str("legal_name", cnpjData.LegalName).
		Int("partners_count", len(cnpjData.Partners)).
		Msg("CNPJ registry data backfilled successfully")

	return updatedStep1
}

// ExecuteStep2 runs Step 2 enrichment: Business Model
// Fields: modelo de negócio, produtos/serviços, público alvo, proposta de valor, região geográfica
// Requires Step 1 data for context
// Optionally crawls company website via Jina Reader for richer context
func (s *Service) ExecuteStep2(ctx context.Context, company *CompanyInput, step1Data *Step1BasicInfo) (*Step2BusinessModel, error) {
	log.Info().
		Str("company_id", company.ID.String()).
		Str("company_name", company.Name).
		Str("step", "2-business-model").
		Msg("Starting Step 2 enrichment")

	// Stage 0: Try to crawl company website via Jina Reader (non-blocking)
	var websiteContent string
	websiteURL := ""
	if company.Website != nil && *company.Website != "" {
		websiteURL = *company.Website
	} else if step1Data != nil && step1Data.Website != nil && *step1Data.Website != "" {
		websiteURL = *step1Data.Website
	}

	if websiteURL != "" && s.jinaClient != nil {
		log.Info().
			Str("company_id", company.ID.String()).
			Str("website", websiteURL).
			Msg("Crawling company website via Jina Reader")

		websiteContent = s.jinaClient.ReadPageSafe(ctx, websiteURL)

		if websiteContent != "" {
			log.Info().
				Str("company_id", company.ID.String()).
				Int("content_length", len(websiteContent)).
				Msg("Website content crawled successfully")
		}
	}

	// Stage 1: Perplexity search with Step 1 context + website content
	rawData, err := s.executeSearch(ctx, Step2SearchSystemPrompt, BuildStep2SearchPrompt(company, step1Data, websiteContent))
	if err != nil {
		log.Error().Err(err).Str("company_id", company.ID.String()).Msg("Step 2 search failed")
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
		log.Error().Err(err).Str("company_id", company.ID.String()).Msg("Step 2 format failed")
		return nil, fmt.Errorf("step 2 format failed: %w", err)
	}

	// Parse response
	result, err := ParseStep2Response(formattedJSON)
	if err != nil {
		log.Error().Err(err).Str("company_id", company.ID.String()).Str("raw", formattedJSON).Msg("Step 2 parse failed")
		return nil, fmt.Errorf("step 2 parse failed: %w", err)
	}

	log.Info().
		Str("company_id", company.ID.String()).
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
// Requires Step 1 and Step 2 data for context
func (s *Service) ExecuteStep3(ctx context.Context, company *CompanyInput, step1Data *Step1BasicInfo, step2Data *Step2BusinessModel) (*Step3CompetitiveIntel, error) {
	log.Info().
		Str("company_id", company.ID.String()).
		Str("company_name", company.Name).
		Str("step", "3-competitive-intel").
		Msg("Starting Step 3 enrichment")

	// Stage 1: Perplexity search with Step 1+2 context
	rawData, err := s.executeSearch(ctx, Step3SearchSystemPrompt, BuildStep3SearchPrompt(company, step1Data, step2Data))
	if err != nil {
		log.Error().Err(err).Str("company_id", company.ID.String()).Msg("Step 3 search failed")
		return nil, fmt.Errorf("step 3 search failed: %w", err)
	}

	log.Debug().
		Int("raw_data_length", len(rawData)).
		Msg("Step 3 search complete")

	// Stage 2: Gemini JSON formatting
	formatPrompt := BuildFormatPrompt("Inteligência Competitiva", rawData, Step3JSONTemplate)
	formattedJSON, err := s.executeFormat(ctx, Step3FormatSystemPrompt, formatPrompt)
	if err != nil {
		log.Error().Err(err).Str("company_id", company.ID.String()).Msg("Step 3 format failed")
		return nil, fmt.Errorf("step 3 format failed: %w", err)
	}

	// Parse response
	result, err := ParseStep3Response(formattedJSON)
	if err != nil {
		log.Error().Err(err).Str("company_id", company.ID.String()).Str("raw", formattedJSON).Msg("Step 3 parse failed")
		return nil, fmt.Errorf("step 3 parse failed: %w", err)
	}

	log.Info().
		Str("company_id", company.ID.String()).
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
