package analysis

import (
	"context"

	"backend_v3/config"
	"backend_v3/llm"

	"github.com/rs/zerolog"
)

// LLMClient defines the contract for the AI service.
type LLMClient interface {
	GenerateStructured(ctx context.Context, model string, prompt string, data interface{}, targetSchema interface{}) error
	GenerateStructuredWithOptions(ctx context.Context, opts llm.GenerationOptions, prompt string, data interface{}, targetSchema interface{}) error
}

// Service handles all business analysis operations
type Service struct {
	repo   Repository
	llm    LLMClient
	logger zerolog.Logger

	// Deprecated fields (kept for backward compatibility)
	analystModel   string
	synthesisModel string

	// New: Framework-specific configurations for heterogeneous model routing
	frameworks map[string]config.FrameworkConfig
}

// NewService creates a new analysis service instance
func NewService(
	repo Repository,
	llm LLMClient,
	logger zerolog.Logger,
	analystModel string, // DEPRECATED: use frameworks map
	synthesisModel string, // DEPRECATED: use frameworks map
) *Service {
	return &Service{
		repo:           repo,
		llm:            llm,
		logger:         logger.With().Str("service", "analysis").Logger(),
		analystModel:   analystModel,
		synthesisModel: synthesisModel,
		frameworks:     make(map[string]config.FrameworkConfig), // Will be populated by main.go
	}
}

// SetFrameworks updates the framework configurations (called by main.go)
func (s *Service) SetFrameworks(frameworks map[string]config.FrameworkConfig) {
	s.frameworks = frameworks
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
