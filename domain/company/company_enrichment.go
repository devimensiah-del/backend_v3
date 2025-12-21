package company

import (
	"context"
	"errors"
	"fmt"

	"backend_v3/domain/enrichment"

	"github.com/google/uuid"
)

// Enrichment errors
var (
	ErrEnrichmentNotConfigured = errors.New("enrichment service not configured")
	ErrStep1NotCompleted       = errors.New("step 1 enrichment must be completed first")
	ErrStep2NotCompleted       = errors.New("step 2 enrichment must be completed first")
	ErrStep2AlreadyRunning     = errors.New("step 2 enrichment already in progress or completed")
	ErrStep3AlreadyRunning     = errors.New("step 3 enrichment already in progress or completed")
)

// =============================================================================
// STEP 1: BASIC INFO (Automatic at company creation)
// =============================================================================

// runStep1Enrichment executes Step 1 enrichment for a company (async)
// Called automatically when a company is created
func (s *Service) runStep1Enrichment(ctx context.Context, companyID uuid.UUID) {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("step", "1-basic-info").
		Msg("Starting Step 1 enrichment")

	// Ensure enrichment record exists
	if s.enrichmentRepo != nil {
		if _, err := s.enrichmentRepo.Create(ctx, companyID); err != nil {
			s.logger.Error().Err(err).Msg("Failed to create enrichment record")
		}
	}

	// 1. Set status to processing (both in company table and enrichment table)
	if err := s.repo.SetEnrichmentProcessing(ctx, companyID); err != nil {
		s.logger.Error().
			Err(err).
			Str("company_id", companyID.String()).
			Msg("Failed to set enrichment processing status")
		return
	}
	if s.enrichmentRepo != nil {
		if err := s.enrichmentRepo.SetStep1Processing(ctx, companyID); err != nil {
			s.logger.Error().Err(err).Msg("Failed to set step1 processing status")
		}
	}

	// 2. Load company
	company, err := s.repo.GetByID(ctx, companyID)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("company_id", companyID.String()).
			Msg("Failed to load company for enrichment")
		s.repo.SetEnrichmentFailed(ctx, companyID, "Failed to load company data")
		if s.enrichmentRepo != nil {
			s.enrichmentRepo.SetStep1Failed(ctx, companyID, "Failed to load company data")
		}
		return
	}
	if company == nil {
		s.logger.Error().
			Str("company_id", companyID.String()).
			Msg("Company not found for enrichment")
		s.repo.SetEnrichmentFailed(ctx, companyID, "Company not found")
		if s.enrichmentRepo != nil {
			s.enrichmentRepo.SetStep1Failed(ctx, companyID, "Company not found")
		}
		return
	}

	// 3. Build company input
	companyInput := &enrichment.CompanyInput{
		ID:       company.ID,
		Name:     company.Name,
		CNPJ:     company.CNPJ,
		Website:  company.Website,
		Industry: company.Industry,
		Location: company.Location,
	}

	// 4. Execute Step 1
	step1Data, err := s.enrichmentService.ExecuteStep1(ctx, companyInput)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("company_id", companyID.String()).
			Msg("Step 1 enrichment failed")
		s.repo.SetEnrichmentFailed(ctx, companyID, err.Error())
		if s.enrichmentRepo != nil {
			s.enrichmentRepo.SetStep1Failed(ctx, companyID, err.Error())
		}
		return
	}

	// 5. Store Step 1 data in enrichment table for context building
	if s.enrichmentRepo != nil {
		if err := s.enrichmentRepo.SetStep1Completed(ctx, companyID, step1Data); err != nil {
			s.logger.Error().Err(err).Msg("Failed to save step1 data to enrichment table")
		}
	}

	// 6. Update company with enriched data (convert Step1 to legacy format for backward compatibility)
	enrichedData := step1ToLegacyData(step1Data)
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
		Float64("confidence_score", step1Data.ConfidenceScore).
		Msg("Step 1 enrichment completed successfully")
}

// buildEnrichmentContext creates an EnrichmentContext from a Company record
// Used by Step 2 and Step 3 to pass ALL company fields to enrichment prompts
func buildEnrichmentContext(company *Company) *enrichment.EnrichmentContext {
	return &enrichment.EnrichmentContext{
		ID:                  company.ID,
		Name:                company.Name,
		LegalName:           company.LegalName,
		TradeName:           company.TradeName,
		CNPJ:                company.CNPJ,
		Website:             company.Website,
		BusinessModel:       company.BusinessModel,
		Industry:            company.Industry,
		Sector:              company.Sector,
		MainProducts:        company.MainProducts,
		PricingModel:        company.PricingModel,
		TargetMarket:        company.TargetMarket,
		TargetAudience:      company.TargetAudience,
		ValueProposition:    company.ValueProposition,
		Location:            company.Location,
		Headquarters:        company.Headquarters,
		EmployeesRange:      company.EmployeesRange,
		GeographicRegions:   company.GeographicRegions,
		ServiceAreas:        company.ServiceAreas,
		CompanySize:         company.CompanySize,
		FundingStage:        company.FundingStage,
		FoundationYear:      company.FoundationYear,
		CustomerSegments:    company.CustomerSegments,
		UniqueSellingPoints: company.UniqueSellingPoints,
		KeyExecutives:       company.KeyExecutives,
		LinkedInURL:         company.LinkedInURL,
	}
}

// step1ToLegacyData converts Step1BasicInfo to EnrichedCompanyData for backward compatibility
func step1ToLegacyData(step1 *enrichment.Step1BasicInfo) *enrichment.EnrichedCompanyData {
	return &enrichment.EnrichedCompanyData{
		CNPJ:           step1.CNPJ,
		Website:        step1.Website,
		LegalName:      step1.LegalName,
		FoundationYear: step1.FoundationYear.ToStringPtr(),
		Headquarters:   step1.Headquarters,
		EmployeesRange: step1.EmployeesRange,
		// CNPJ Registry data
		TradeName:     step1.TradeName,
		Phone:         step1.Phone,
		Email:         step1.Email,
		CNAEPrimary:   step1.CNAEPrimary,
		CNAECodes:     step1.CNAECodes,
		CapitalSocial: step1.CapitalSocial,
		Partners:      step1.Partners,
		CNPJVerified:  step1.CNPJVerified,
		// Social links
		LinkedInURL:   step1.LinkedInURL,
		TwitterHandle: step1.TwitterHandle,
		InstagramURL:  step1.InstagramURL,
		FacebookURL:   step1.FacebookURL,
		// Other
		KeyExecutives:   step1.KeyExecutives,
		ConfidenceScore: step1.ConfidenceScore,
		Sources:         step1.Sources,
	}
}

// runEnrichment is an alias for runStep1Enrichment for backward compatibility
func (s *Service) runEnrichment(ctx context.Context, companyID uuid.UUID) {
	s.runStep1Enrichment(ctx, companyID)
}

// RetryStep1 re-runs Step 1 enrichment even if already completed
// Resets status and runs enrichment asynchronously
func (s *Service) RetryStep1(ctx context.Context, companyID uuid.UUID) error {
	if s.enrichmentService == nil {
		return ErrEnrichmentNotConfigured
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
		Str("step", "1-basic-info").
		Str("company_name", company.Name).
		Msg("Retrying Step 1 enrichment")

	// Reset Step 1 status if enrichment repo is available
	if s.enrichmentRepo != nil {
		if err := s.enrichmentRepo.ResetStep1(ctx, companyID); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to reset step1 status, continuing anyway")
		}
	}

	// Run asynchronously
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error().
					Interface("panic", r).
					Str("company_id", companyID.String()).
					Msg("Step 1 retry goroutine panicked")
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), s.enrichmentTimeout)
		defer cancel()
		s.runStep1Enrichment(ctx, companyID)
	}()

	return nil
}

// =============================================================================
// STEP 2: BUSINESS MODEL (Human-triggered)
// =============================================================================

// TriggerStep2 initiates Step 2 enrichment (business model)
// Requires Step 1 to be completed first
// Returns immediately, enrichment runs asynchronously
func (s *Service) TriggerStep2(ctx context.Context, companyID uuid.UUID) error {
	if s.enrichmentService == nil {
		return ErrEnrichmentNotConfigured
	}
	if s.enrichmentRepo == nil {
		return ErrEnrichmentNotConfigured
	}

	// Check enrichment status
	enrichmentRecord, err := s.enrichmentRepo.GetByCompanyID(ctx, companyID)
	if err != nil {
		if errors.Is(err, enrichment.ErrNotFound) {
			return ErrStep1NotCompleted
		}
		return fmt.Errorf("failed to get enrichment status: %w", err)
	}

	if !enrichmentRecord.CanTriggerStep2() {
		if enrichmentRecord.Step1Status != enrichment.StatusCompleted {
			return ErrStep1NotCompleted
		}
		return ErrStep2AlreadyRunning
	}

	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("step", "2-business-model").
		Msg("Triggering Step 2 enrichment")

	// Run asynchronously
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error().
					Interface("panic", r).
					Str("company_id", companyID.String()).
					Msg("Step 2 enrichment goroutine panicked")
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), s.enrichmentTimeout)
		defer cancel()
		s.runStep2Enrichment(ctx, companyID)
	}()

	return nil
}

// runStep2Enrichment executes Step 2 enrichment (async)
func (s *Service) runStep2Enrichment(ctx context.Context, companyID uuid.UUID) {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("step", "2-business-model").
		Msg("Starting Step 2 enrichment")

	// 1. Set status to processing
	if err := s.enrichmentRepo.SetStep2Processing(ctx, companyID); err != nil {
		s.logger.Error().Err(err).Msg("Failed to set step2 processing status")
		return
	}

	// 2. Load company (contains ALL fields including Step 1 enriched data)
	company, err := s.repo.GetByID(ctx, companyID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to load company for step2")
		s.enrichmentRepo.SetStep2Failed(ctx, companyID, "Failed to load company")
		return
	}

	// 3. Build enrichment context from company record (ALL fields)
	enrichCtx := buildEnrichmentContext(company)

	// 4. Execute Step 2
	step2Data, err := s.enrichmentService.ExecuteStep2(ctx, enrichCtx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Step 2 enrichment failed")
		s.enrichmentRepo.SetStep2Failed(ctx, companyID, err.Error())
		return
	}

	// 5. Store Step 2 data
	if err := s.enrichmentRepo.SetStep2Completed(ctx, companyID, step2Data); err != nil {
		s.logger.Error().Err(err).Msg("Failed to save step2 data")
		return
	}

	// 6. Update company with Step 2 data
	if err := s.updateCompanyWithStep2Data(ctx, companyID, step2Data); err != nil {
		s.logger.Error().Err(err).Msg("Failed to update company with step2 data")
		// Don't fail the enrichment - data is saved in enrichment table
	}

	s.logger.Info().
		Str("company_id", companyID.String()).
		Float64("confidence_score", step2Data.ConfidenceScore).
		Int("products_count", len(step2Data.MainProducts)).
		Msg("Step 2 enrichment completed successfully")
}

// updateCompanyWithStep2Data updates the company record with Step 2 enriched data
func (s *Service) updateCompanyWithStep2Data(ctx context.Context, companyID uuid.UUID, data *enrichment.Step2BusinessModel) error {
	company, err := s.repo.GetByID(ctx, companyID)
	if err != nil {
		return err
	}

	// Update fields from Step 2
	if data.BusinessModel != nil {
		company.BusinessModel = data.BusinessModel
	}
	if data.Industry != nil {
		company.Industry = data.Industry
	}
	if data.Sector != nil {
		company.Sector = data.Sector
	}
	if len(data.MainProducts) > 0 {
		company.MainProducts = data.MainProducts
	}
	if data.PricingModel != nil {
		company.PricingModel = data.PricingModel
	}
	if data.TargetMarket != nil {
		company.TargetMarket = data.TargetMarket
	}
	if data.TargetAudience != nil {
		company.TargetAudience = data.TargetAudience
	}
	if len(data.CustomerSegments) > 0 {
		company.CustomerSegments = data.CustomerSegments
	}
	if data.ValueProposition != nil {
		company.ValueProposition = data.ValueProposition
	}
	if len(data.UniqueSellingPoints) > 0 {
		company.UniqueSellingPoints = data.UniqueSellingPoints
	}
	if len(data.GeographicRegions) > 0 {
		company.GeographicRegions = data.GeographicRegions
	}
	if len(data.ServiceAreas) > 0 {
		company.ServiceAreas = data.ServiceAreas
	}

	return s.repo.Update(ctx, company)
}

// RetryStep2 re-runs Step 2 enrichment even if already completed
// Resets status and runs enrichment asynchronously
func (s *Service) RetryStep2(ctx context.Context, companyID uuid.UUID) error {
	if s.enrichmentService == nil {
		return ErrEnrichmentNotConfigured
	}
	if s.enrichmentRepo == nil {
		return ErrEnrichmentNotConfigured
	}

	// Check that Step 1 is completed
	enrichmentRecord, err := s.enrichmentRepo.GetByCompanyID(ctx, companyID)
	if err != nil {
		if errors.Is(err, enrichment.ErrNotFound) {
			return ErrStep1NotCompleted
		}
		return fmt.Errorf("failed to get enrichment status: %w", err)
	}

	if enrichmentRecord.Step1Status != enrichment.StatusCompleted {
		return ErrStep1NotCompleted
	}

	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("step", "2-business-model").
		Str("previous_status", string(enrichmentRecord.Step2Status)).
		Msg("Retrying Step 2 enrichment")

	// Reset Step 2 status
	if err := s.enrichmentRepo.ResetStep2(ctx, companyID); err != nil {
		return fmt.Errorf("failed to reset step 2: %w", err)
	}

	// Run asynchronously
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error().
					Interface("panic", r).
					Str("company_id", companyID.String()).
					Msg("Step 2 retry goroutine panicked")
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), s.enrichmentTimeout)
		defer cancel()
		s.runStep2Enrichment(ctx, companyID)
	}()

	return nil
}

// =============================================================================
// STEP 3: COMPETITIVE INTELLIGENCE (Human-triggered)
// =============================================================================

// TriggerStep3 initiates Step 3 enrichment (competitive intelligence)
// Requires Step 2 to be completed first
// Returns immediately, enrichment runs asynchronously
func (s *Service) TriggerStep3(ctx context.Context, companyID uuid.UUID) error {
	if s.enrichmentService == nil {
		return ErrEnrichmentNotConfigured
	}
	if s.enrichmentRepo == nil {
		return ErrEnrichmentNotConfigured
	}

	// Check enrichment status
	enrichmentRecord, err := s.enrichmentRepo.GetByCompanyID(ctx, companyID)
	if err != nil {
		if errors.Is(err, enrichment.ErrNotFound) {
			return ErrStep2NotCompleted
		}
		return fmt.Errorf("failed to get enrichment status: %w", err)
	}

	if !enrichmentRecord.CanTriggerStep3() {
		if enrichmentRecord.Step2Status != enrichment.StatusCompleted {
			return ErrStep2NotCompleted
		}
		return ErrStep3AlreadyRunning
	}

	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("step", "3-competitive-intel").
		Msg("Triggering Step 3 enrichment")

	// Run asynchronously
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error().
					Interface("panic", r).
					Str("company_id", companyID.String()).
					Msg("Step 3 enrichment goroutine panicked")
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), s.enrichmentTimeout)
		defer cancel()
		s.runStep3Enrichment(ctx, companyID)
	}()

	return nil
}

// runStep3Enrichment executes Step 3 enrichment (async)
func (s *Service) runStep3Enrichment(ctx context.Context, companyID uuid.UUID) {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("step", "3-competitive-intel").
		Msg("Starting Step 3 enrichment")

	// 1. Set status to processing
	if err := s.enrichmentRepo.SetStep3Processing(ctx, companyID); err != nil {
		s.logger.Error().Err(err).Msg("Failed to set step3 processing status")
		return
	}

	// 2. Load company (contains ALL fields including Step 1 and Step 2 enriched data)
	company, err := s.repo.GetByID(ctx, companyID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to load company for step3")
		s.enrichmentRepo.SetStep3Failed(ctx, companyID, "Failed to load company")
		return
	}

	// 3. Build enrichment context from company record (ALL fields)
	enrichCtx := buildEnrichmentContext(company)

	// 4. Execute Step 3
	step3Data, err := s.enrichmentService.ExecuteStep3(ctx, enrichCtx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Step 3 enrichment failed")
		s.enrichmentRepo.SetStep3Failed(ctx, companyID, err.Error())
		return
	}

	// 5. Store Step 3 data
	if err := s.enrichmentRepo.SetStep3Completed(ctx, companyID, step3Data); err != nil {
		s.logger.Error().Err(err).Msg("Failed to save step3 data")
		return
	}

	// 6. Update company with Step 3 data
	if err := s.updateCompanyWithStep3Data(ctx, companyID, step3Data); err != nil {
		s.logger.Error().Err(err).Msg("Failed to update company with step3 data")
		// Don't fail the enrichment - data is saved in enrichment table
	}

	s.logger.Info().
		Str("company_id", companyID.String()).
		Float64("confidence_score", step3Data.ConfidenceScore).
		Int("competitors_count", len(step3Data.Competitors)).
		Int("news_count", len(step3Data.RecentNews)).
		Msg("Step 3 enrichment completed successfully")
}

// RetryStep3 re-runs Step 3 enrichment even if already completed
// Resets status and runs enrichment asynchronously
func (s *Service) RetryStep3(ctx context.Context, companyID uuid.UUID) error {
	if s.enrichmentService == nil {
		return ErrEnrichmentNotConfigured
	}
	if s.enrichmentRepo == nil {
		return ErrEnrichmentNotConfigured
	}

	// Check that Step 2 is completed
	enrichmentRecord, err := s.enrichmentRepo.GetByCompanyID(ctx, companyID)
	if err != nil {
		if errors.Is(err, enrichment.ErrNotFound) {
			return ErrStep2NotCompleted
		}
		return fmt.Errorf("failed to get enrichment status: %w", err)
	}

	if enrichmentRecord.Step2Status != enrichment.StatusCompleted {
		return ErrStep2NotCompleted
	}

	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("step", "3-competitive-intel").
		Str("previous_status", string(enrichmentRecord.Step3Status)).
		Msg("Retrying Step 3 enrichment")

	// Reset Step 3 status
	if err := s.enrichmentRepo.ResetStep3(ctx, companyID); err != nil {
		return fmt.Errorf("failed to reset step 3: %w", err)
	}

	// Run asynchronously
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error().
					Interface("panic", r).
					Str("company_id", companyID.String()).
					Msg("Step 3 retry goroutine panicked")
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), s.enrichmentTimeout)
		defer cancel()
		s.runStep3Enrichment(ctx, companyID)
	}()

	return nil
}

// updateCompanyWithStep3Data updates the company record with Step 3 enriched data
func (s *Service) updateCompanyWithStep3Data(ctx context.Context, companyID uuid.UUID, data *enrichment.Step3CompetitiveIntel) error {
	company, err := s.repo.GetByID(ctx, companyID)
	if err != nil {
		return err
	}

	// Update fields from Step 3
	if len(data.Competitors) > 0 {
		company.Competitors = data.Competitors
	}
	if len(data.CompetitorDetails) > 0 {
		company.CompetitorDetails = data.CompetitorDetails
	}
	if data.IndustryGrowthRate != nil {
		company.IndustryGrowthRate = data.IndustryGrowthRate
	}
	if len(data.IndustryTrends) > 0 {
		company.IndustryTrends = data.IndustryTrends
	}
	if data.MarketConcentration != nil {
		company.MarketConcentration = data.MarketConcentration
	}
	if data.RegulatoryContext != nil {
		company.RegulatoryContext = data.RegulatoryContext
	}
	if data.MarketPosition != nil {
		company.MarketShareStatus = data.MarketPosition // Map to existing field
	}
	if len(data.RecentNews) > 0 {
		company.RecentNews = data.RecentNews
	}
	if len(data.Sources) > 0 {
		company.EnrichmentSources = data.Sources
	}

	return s.repo.Update(ctx, company)
}

// =============================================================================
// LEGACY SUPPORT
// =============================================================================

// RetryEnrichment re-runs Step 1 enrichment for a company
// Only fills in NULL/empty fields - preserves any manually edited values
// Returns error if enrichment service not configured
func (s *Service) RetryEnrichment(ctx context.Context, companyID uuid.UUID) error {
	if s.enrichmentService == nil {
		return ErrEnrichmentNotConfigured
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
		Msg("Retrying Step 1 enrichment")

	// Run enrichment synchronously (caller can wrap in goroutine if needed)
	s.runStep1Enrichment(ctx, companyID)

	return nil
}

// GetEnrichmentStatus returns the current enrichment status for a company
func (s *Service) GetEnrichmentStatus(ctx context.Context, companyID uuid.UUID) (*enrichment.CompanyEnrichment, error) {
	if s.enrichmentRepo == nil {
		return nil, ErrEnrichmentNotConfigured
	}
	return s.enrichmentRepo.GetByCompanyID(ctx, companyID)
}
