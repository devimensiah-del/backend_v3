package analysis

import (
	"context"

	"github.com/rs/zerolog"
)

// LLMClient defines the contract for the AI service.
type LLMClient interface {
	GenerateStructured(ctx context.Context, model string, prompt string, data interface{}, targetSchema interface{}) error
}

// Service handles all business analysis operations
type Service struct {
	repo   Repository
	llm    LLMClient
	logger zerolog.Logger

	// Configuration injected at startup
	analystModel   string
	synthesisModel string
}

// NewService creates a new analysis service instance
func NewService(
	repo Repository,
	llm LLMClient,
	logger zerolog.Logger,
	analystModel string, // e.g. "google/gemini-2.5-pro"
	synthesisModel string, // e.g. "anthropic/claude-sonnet-4.5"
) *Service {
	return &Service{
		repo:           repo,
		llm:            llm,
		logger:         logger.With().Str("service", "analysis").Logger(),
		analystModel:   analystModel,
		synthesisModel: synthesisModel,
	}
}

// GetByID retrieves an analysis by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Analysis, error) {
	return s.repo.GetByID(ctx, id)
}

// GetBySubmissionID retrieves an analysis by submission ID
func (s *Service) GetBySubmissionID(ctx context.Context, submissionID string) (*Analysis, error) {
	return s.repo.GetBySubmissionID(ctx, submissionID)
}

// ListAll retrieves all analyses with pagination
func (s *Service) ListAll(ctx context.Context, limit, offset int) ([]*Analysis, error) {
	return s.repo.List(ctx, limit, offset)
}

// Delete removes an analysis record
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
