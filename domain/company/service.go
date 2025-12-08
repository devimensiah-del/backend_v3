package company

import (
	"context"
	"fmt"
	"time"

	"backend_v3/domain/enrichment"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service handles business logic for companies
type Service struct {
	repo              Repository
	logger            zerolog.Logger
	enrichmentService *enrichment.Service
	enrichmentTimeout time.Duration
}

// NewService creates a new company service
// enrichmentTimeoutSeconds defaults to 300 (5 minutes) if 0 or negative
func NewService(repo Repository, logger zerolog.Logger, enrichmentTimeoutSeconds int) *Service {
	timeout := time.Duration(enrichmentTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute // default
	}
	return &Service{
		repo:              repo,
		logger:            logger.With().Str("service", "company").Logger(),
		enrichmentTimeout: timeout,
	}
}

// SetEnrichmentService injects the enrichment service (optional)
func (s *Service) SetEnrichmentService(enrichSvc *enrichment.Service) {
	s.enrichmentService = enrichSvc
}

// CreateDirect creates a company directly (without a submission)
// Used by POST /companies endpoint for authenticated users
// Automatically triggers enrichment asynchronously
func (s *Service) CreateDirect(ctx context.Context, input CreateFromSubmissionInput) (*Company, error) {
	s.logger.Info().
		Str("company_name", input.CompanyName).
		Msg("Creating company directly (no submission)")

	// Create the company
	company := NewCompany(input)

	// Validate before persisting
	if err := company.Validate(); err != nil {
		return nil, err
	}

	// Just create the company, no submission link needed
	if err := s.repo.Create(ctx, company); err != nil {
		s.logger.Error().Err(err).Msg("Failed to create company directly")
		return nil, fmt.Errorf("failed to create company: %w", err)
	}

	s.logger.Info().
		Str("company_id", company.ID.String()).
		Str("company_name", company.Name).
		Msg("Company created directly")

	// Trigger enrichment asynchronously (fire-and-forget)
	// Use a fresh context with timeout since this runs independently of the request
	if s.enrichmentService != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error().
						Interface("panic", r).
						Str("company_id", company.ID.String()).
						Msg("Enrichment goroutine panicked - recovered")
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), s.enrichmentTimeout)
			defer cancel()
			s.runEnrichment(ctx, company.ID)
		}()
	} else {
		s.logger.Warn().Msg("Enrichment service not configured - skipping enrichment")
	}

	return company, nil
}

// CreateFromSubmission creates a new company record when a submission is created
// Links company to submission via company_submissions table
// Automatically triggers enrichment asynchronously
func (s *Service) CreateFromSubmission(ctx context.Context, input CreateFromSubmissionInput) (*Company, error) {
	s.logger.Info().
		Str("submission_id", input.SubmissionID.String()).
		Str("company_name", input.CompanyName).
		Msg("Creating company from submission")

	// Create the company
	company := NewCompany(input)

	// Validate before persisting
	if err := company.Validate(); err != nil {
		return nil, err
	}

	// Use transaction to ensure atomicity
	err := s.repo.WithTx(ctx, func(txRepo Repository) error {
		// 1. Insert company
		if err := txRepo.Create(ctx, company); err != nil {
			return fmt.Errorf("failed to create company: %w", err)
		}

		// 2. Link to submission
		link := &CompanySubmission{
			CompanyID:    company.ID,
			SubmissionID: input.SubmissionID,
			IsPrimary:    true,
		}
		if err := txRepo.LinkSubmission(ctx, link); err != nil {
			return fmt.Errorf("failed to link submission: %w", err)
		}

		return nil
	})

	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create company from submission")
		return nil, err
	}

	s.logger.Info().
		Str("company_id", company.ID.String()).
		Str("company_name", company.Name).
		Msg("Company created successfully")

	// Trigger enrichment asynchronously (fire-and-forget)
	// Use a fresh context with timeout since this runs independently of the request
	if s.enrichmentService != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error().
						Interface("panic", r).
						Str("company_id", company.ID.String()).
						Msg("Enrichment goroutine panicked - recovered")
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), s.enrichmentTimeout)
			defer cancel()
			s.runEnrichment(ctx, company.ID)
		}()
	} else {
		s.logger.Warn().Msg("Enrichment service not configured - skipping enrichment")
	}

	return company, nil
}

// GetByID retrieves a company by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Company, error) {
	return s.repo.GetByID(ctx, id)
}

// Delete removes a company (hard delete for saga rollback)
// This should only be used during saga rollback - not for normal operations
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	s.logger.Info().
		Str("company_id", id.String()).
		Msg("Deleting company (saga rollback)")

	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error().
			Err(err).
			Str("company_id", id.String()).
			Msg("Failed to delete company")
		return fmt.Errorf("failed to delete company: %w", err)
	}

	s.logger.Info().
		Str("company_id", id.String()).
		Msg("Company deleted successfully")

	return nil
}

// GetBySubmissionID retrieves the company linked to a submission
func (s *Service) GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*Company, error) {
	return s.repo.GetBySubmissionID(ctx, submissionID)
}

// LinkSubmission creates a link between company and submission
func (s *Service) LinkSubmission(ctx context.Context, companyID, submissionID uuid.UUID, isPrimary bool, linkedBy *uuid.UUID) error {
	link := &CompanySubmission{
		CompanyID:    companyID,
		SubmissionID: submissionID,
		IsPrimary:    isPrimary,
		LinkedBy:     linkedBy,
	}
	return s.repo.LinkSubmission(ctx, link)
}

// ==================== User Companies API ====================

// GetUserCompanies returns all companies where the user is owner or in allowed_users
func (s *Service) GetUserCompanies(ctx context.Context, userID uuid.UUID) ([]*Company, error) {
	s.logger.Debug().
		Str("user_id", userID.String()).
		Msg("Getting companies for user")

	companies, err := s.repo.GetUserCompanies(ctx, userID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get user companies")
		return nil, fmt.Errorf("failed to get user companies: %w", err)
	}

	s.logger.Debug().
		Int("count", len(companies)).
		Msg("Found user companies")

	return companies, nil
}

// ListAll returns all companies (admin)
func (s *Service) ListAll(ctx context.Context, limit, offset int) ([]*Company, int, error) {
	return s.repo.ListAll(ctx, limit, offset)
}

// GetAnalysesHistory returns all analyses for a company with their associated challenges
func (s *Service) GetAnalysesHistory(ctx context.Context, companyID uuid.UUID) ([]*AnalysisHistoryItem, error) {
	return s.repo.GetAnalysesHistory(ctx, companyID)
}

// SetOwnerFromSubmission sets owner_id and allowed_users for company linked to submission
// Used when linking anonymous submissions to newly signed-up users
// Only sets owner if company doesn't already have one (idempotent)
func (s *Service) SetOwnerFromSubmission(ctx context.Context, submissionID, userID uuid.UUID) error {
	s.logger.Debug().
		Str("submission_id", submissionID.String()).
		Str("user_id", userID.String()).
		Msg("Setting owner from submission")

	// Find company linked to this submission
	company, err := s.repo.GetBySubmissionID(ctx, submissionID)
	if err != nil {
		return fmt.Errorf("failed to get company for submission: %w", err)
	}
	if company == nil {
		s.logger.Debug().
			Str("submission_id", submissionID.String()).
			Msg("No company linked to submission - skipping owner assignment")
		return nil // No company to update, not an error
	}

	// Skip if already has an owner
	if company.OwnerID != nil {
		s.logger.Debug().
			Str("company_id", company.ID.String()).
			Str("existing_owner", company.OwnerID.String()).
			Msg("Company already has owner - skipping")
		return nil
	}

	// Set owner and add to allowed_users
	company.OwnerID = &userID
	if !company.AllowedUsers.Contains(userID) {
		company.AllowedUsers = append(company.AllowedUsers, userID)
	}

	if err := s.repo.Update(ctx, company); err != nil {
		return fmt.Errorf("failed to update company owner: %w", err)
	}

	s.logger.Info().
		Str("company_id", company.ID.String()).
		Str("company_name", company.Name).
		Str("new_owner", userID.String()).
		Msg("Company owner set from submission")

	return nil
}

// ==================== ENRICHMENT INTEGRATION ====================

// runEnrichment executes the enrichment process for a company (async)
func (s *Service) runEnrichment(ctx context.Context, companyID uuid.UUID) {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Msg("Starting company enrichment")

	// 1. Set status to processing
	if err := s.repo.SetEnrichmentProcessing(ctx, companyID); err != nil {
		s.logger.Error().
			Err(err).
			Str("company_id", companyID.String()).
			Msg("Failed to set enrichment processing status")
		return
	}

	// 2. Load company
	company, err := s.repo.GetByID(ctx, companyID)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("company_id", companyID.String()).
			Msg("Failed to load company for enrichment")
		s.repo.SetEnrichmentFailed(ctx, companyID, "Failed to load company data")
		return
	}
	if company == nil {
		s.logger.Error().
			Str("company_id", companyID.String()).
			Msg("Company not found for enrichment")
		s.repo.SetEnrichmentFailed(ctx, companyID, "Company not found")
		return
	}

	// 3. Call enrichment service
	companyInput := &enrichment.CompanyInput{
		ID:       company.ID,
		Name:     company.Name,
		CNPJ:     company.CNPJ,
		Website:  company.Website,
		Industry: company.Industry,
		Location: company.Location,
	}

	enrichedData, err := s.enrichmentService.EnrichCompany(ctx, companyInput)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("company_id", companyID.String()).
			Msg("Enrichment failed")
		s.repo.SetEnrichmentFailed(ctx, companyID, err.Error())
		return
	}

	// 4. Update company with enriched data
	if err := s.repo.SetEnrichmentCompleted(ctx, companyID, enrichedData); err != nil {
		s.logger.Error().
			Err(err).
			Str("company_id", companyID.String()).
			Msg("Failed to save enriched data")
		s.repo.SetEnrichmentFailed(ctx, companyID, "Failed to save enriched data")
		return
	}

	s.logger.Info().
		Str("company_id", companyID.String()).
		Float64("confidence_score", enrichedData.ConfidenceScore).
		Msg("Company enrichment completed successfully")
}

// RetryEnrichment re-runs enrichment for a company using "fill gaps only" logic
// Only fills in NULL/empty fields - preserves any manually edited values
// Returns error if enrichment service not configured
func (s *Service) RetryEnrichment(ctx context.Context, companyID uuid.UUID) error {
	if s.enrichmentService == nil {
		return fmt.Errorf("enrichment service not configured")
	}

	// Check company exists
	company, err := s.repo.GetByID(ctx, companyID)
	if err != nil {
		return fmt.Errorf("failed to get company: %w", err)
	}
	if company == nil {
		return fmt.Errorf("company not found: %s", companyID)
	}

	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("company_name", company.Name).
		Str("current_status", company.EnrichmentStatus).
		Msg("Retrying enrichment (fill gaps only)")

	// Run enrichment synchronously (caller can wrap in goroutine if needed)
	s.runEnrichment(ctx, companyID)

	return nil
}

// NOTE: UpdateFieldsWithAutoVerification removed - edit company in Supabase directly
