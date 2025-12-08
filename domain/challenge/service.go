package challenge

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Service handles challenge business logic
type Service struct {
	repo Repository
}

// NewService creates a new challenge service
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// Create creates a new challenge
func (s *Service) Create(ctx context.Context, c *Challenge) (*Challenge, error) {
	// Initialize ID and timestamps if not set
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = time.Now()
	}

	// Validate
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create in repository
	if err := s.repo.Create(ctx, c); err != nil {
		// Log internal error if it's a RepositoryError
		if repoErr, ok := err.(*RepositoryError); ok && repoErr.Internal() != nil {
			log.Error().
				Err(repoErr.Internal()).
				Str("company_id", c.CompanyID.String()).
				Str("category", string(c.ChallengeCategory)).
				Str("type", string(c.ChallengeType)).
				Msg("Database error during challenge creation")
		}
		return nil, fmt.Errorf("failed to create challenge: %w", err)
	}

	log.Info().
		Str("id", c.ID.String()).
		Str("company_id", c.CompanyID.String()).
		Str("category", string(c.ChallengeCategory)).
		Str("type", string(c.ChallengeType)).
		Msg("challenge created")

	return c, nil
}

// GetByID retrieves a challenge by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Challenge, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByCompany retrieves all challenges for a company
func (s *Service) ListByCompany(ctx context.Context, companyID uuid.UUID) ([]*Challenge, error) {
	return s.repo.ListByCompanyID(ctx, companyID)
}

// Update updates a challenge
func (s *Service) Update(ctx context.Context, c *Challenge) error {
	// Validate
	if err := c.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Update in repository
	if err := s.repo.Update(ctx, c); err != nil {
		return fmt.Errorf("failed to update challenge: %w", err)
	}

	log.Info().
		Str("id", c.ID.String()).
		Str("company_id", c.CompanyID.String()).
		Msg("challenge updated")

	return nil
}

// Delete soft-deletes a challenge
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete challenge: %w", err)
	}

	log.Info().Str("id", id.String()).Msg("challenge deleted")

	return nil
}

// =============================================================================
// VALIDATION HELPERS (for use by other domains via interface)
// =============================================================================

// ValidateCategory checks if a category string is valid
func (s *Service) ValidateCategory(category string) bool {
	return IsValidCategory(category)
}

// ValidateType checks if a type string is valid for the given category
func (s *Service) ValidateType(category, challengeType string) bool {
	return IsValidType(category, challengeType)
}
