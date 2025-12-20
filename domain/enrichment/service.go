package enrichment

import (
	"context"
	"fmt"

	"backend_v3/llm"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// MODEL CONFIGURATION
// =============================================================================

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

// =============================================================================
// SERVICE
// =============================================================================

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

// =============================================================================
// MAIN ORCHESTRATION
// =============================================================================

// EnrichCompany uses two-stage enrichment:
// Stage 1: Perplexity searches the web for raw company data
// Stage 2: Claude Sonnet 4.5 validates and synthesizes with reasoning
// Returns enriched data - caller is responsible for saving to company table
func (s *Service) EnrichCompany(ctx context.Context, company *CompanyInput) (*EnrichedCompanyData, error) {
	log.Info().
		Str("company_id", company.ID.String()).
		Str("company_name", company.Name).
		Str("stage1_model", ModelStage1Search).
		Str("stage2_model", ModelStage2Synthesis).
		Msg("Starting two-stage company enrichment")

	// Stage 1: Perplexity Web Search
	rawData, err := s.executeSearch(ctx, company)
	if err != nil {
		return nil, err
	}

	// Stage 2: Claude Synthesis
	enrichedData, err := s.executeSynthesis(ctx, company, rawData)
	if err != nil {
		return nil, err
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

// =============================================================================
// STAGE EXECUTION
// =============================================================================

// executeSearch runs Stage 1: Perplexity web search for raw data
func (s *Service) executeSearch(ctx context.Context, company *CompanyInput) (string, error) {
	log.Debug().Msg("Stage 1: Perplexity web search for raw data")

	searchReq := llm.Request{
		Model:        ModelStage1Search,
		SystemPrompt: SearchSystemPrompt,
		Messages:     []llm.Message{{Role: "user", Content: BuildSearchPrompt(company)}},
		Temperature:  0.2, // Very low for factual accuracy
		MaxTokens:    3000,
	}

	searchResp, err := s.llmClient.CallWithFallback(ctx, &searchReq, ModelStage1Fallback)
	if err != nil {
		log.Error().
			Err(err).
			Str("company_id", company.ID.String()).
			Msg("Stage 1 (Perplexity search) failed")
		return "", fmt.Errorf("perplexity search failed: %w", err)
	}

	rawData := searchResp.Content
	log.Debug().
		Int("raw_data_length", len(rawData)).
		Msg("Stage 1 complete: raw data gathered")

	return rawData, nil
}

// executeSynthesis runs Stage 2: Claude synthesis of raw data into structured output
func (s *Service) executeSynthesis(ctx context.Context, company *CompanyInput, rawData string) (*EnrichedCompanyData, error) {
	log.Debug().Msg("Stage 2: Claude Sonnet 4.5 data synthesis")

	synthesisReq := llm.Request{
		Model:        ModelStage2Synthesis,
		SystemPrompt: SynthesisSystemPrompt,
		Messages:     []llm.Message{{Role: "user", Content: BuildSynthesisPrompt(company, rawData)}},
		Temperature:  0.1, // Very low for consistent JSON and data preservation
		MaxTokens:    4000,
	}

	synthesisResp, err := s.llmClient.CallWithFallback(ctx, &synthesisReq, ModelStage2Fallback)
	if err != nil {
		log.Error().
			Err(err).
			Str("company_id", company.ID.String()).
			Msg("Stage 2 (Claude synthesis) failed")
		return nil, fmt.Errorf("claude synthesis failed: %w", err)
	}

	// Parse synthesized response
	enrichedData, err := ParseEnrichmentResponse(synthesisResp.Content)
	if err != nil {
		log.Error().
			Err(err).
			Str("company_id", company.ID.String()).
			Str("raw_response", synthesisResp.Content).
			Msg("Failed to parse synthesis response")
		return nil, fmt.Errorf("failed to parse synthesis response: %w", err)
	}

	return enrichedData, nil
}
