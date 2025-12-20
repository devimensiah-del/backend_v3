package company

import (
	"context"
	"fmt"

	"backend_v3/domain/enrichment"

	"github.com/google/uuid"
)

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
