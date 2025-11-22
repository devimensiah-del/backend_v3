package enrichment

import (
	"context"

	"backend_v3/config"
	"backend_v3/domain/submission"
	"backend_v3/llm"

	"github.com/google/uuid"
)

// Service handles business logic for enrichment
type Service struct {
	repo           Repository
	submissionRepo submission.Repository
	llmClient      *llm.Client
	model          string                // DEPRECATED: kept for backward compatibility
	enrichmentCfg  config.FrameworkConfig // NEW: Framework-specific config
}

// NewService creates a new enrichment service
func NewService(repo Repository, submissionRepo submission.Repository, llmClient *llm.Client, model string) *Service {
	return &Service{
		repo:           repo,
		submissionRepo: submissionRepo,
		llmClient:      llmClient,
		model:          model,
		enrichmentCfg:  config.FrameworkConfig{}, // Will be set by main.go
	}
}

// SetEnrichmentConfig updates the enrichment configuration (called by main.go)
func (s *Service) SetEnrichmentConfig(cfg config.FrameworkConfig) {
	s.enrichmentCfg = cfg
}

// GetByID retrieves enrichment by its own ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Enrichment, error) {
	return s.repo.GetByID(ctx, id)
}

// GetBySubmissionID retrieves enrichment linked to a submission
func (s *Service) GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*Enrichment, error) {
	return s.repo.GetBySubmissionID(ctx, submissionID)
}

// UpdateEnrichmentData is called by the User Interface (API).
// IMPORTANT: This uses UpdateUser, which forces a 'Lock' on the file.
// Once called, the AI background worker will stop overwriting this data.
func (s *Service) UpdateEnrichmentData(ctx context.Context, id uuid.UUID, data map[string]interface{}) error {
	enrichment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Convert generic map to our special JSONMap type
	enrichment.EnrichedData = JSONMap(data)

	// Force the lock and update
	return s.repo.UpdateUser(ctx, enrichment)
}
