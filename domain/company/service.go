package company

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service handles business logic for companies
type Service struct {
	repo   Repository
	logger zerolog.Logger
}

// NewService creates a new company service
func NewService(repo Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With().Str("service", "company").Logger(),
	}
}

// CreateFromSubmission creates a new company record when a submission is created
// Returns the created company and records initial history
func (s *Service) CreateFromSubmission(ctx context.Context, input CreateFromSubmissionInput) (*Company, error) {
	s.logger.Info().
		Str("submission_id", input.SubmissionID.String()).
		Str("company_name", input.CompanyName).
		Msg("Creating company from submission")

	// Create the company
	company := NewCompany(input)

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

		// 3. Record initial history for all non-null fields
		sourceID := input.SubmissionID
		if err := s.recordInitialHistory(ctx, txRepo, company, &sourceID); err != nil {
			return fmt.Errorf("failed to record history: %w", err)
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

	return company, nil
}

// recordInitialHistory records the initial values from submission
func (s *Service) recordInitialHistory(ctx context.Context, repo Repository, company *Company, sourceID *uuid.UUID) error {
	// Helper to record a field
	recordField := func(fieldName string, value *string) error {
		if value == nil || *value == "" {
			return nil
		}
		return repo.RecordHistory(ctx, &DataHistoryEntry{
			CompanyID: company.ID,
			FieldName: fieldName,
			NewValue:  value,
			Source:    SourceSubmission,
			SourceID:  sourceID,
		})
	}

	// Record all submission fields
	name := company.Name
	if err := recordField("name", &name); err != nil {
		return err
	}
	if err := recordField("cnpj", company.CNPJ); err != nil {
		return err
	}
	if err := recordField("website", company.Website); err != nil {
		return err
	}
	if err := recordField("industry", company.Industry); err != nil {
		return err
	}
	if err := recordField("company_size", company.CompanySize); err != nil {
		return err
	}
	if err := recordField("location", company.Location); err != nil {
		return err
	}
	if err := recordField("target_market", company.TargetMarket); err != nil {
		return err
	}
	if err := recordField("funding_stage", company.FundingStage); err != nil {
		return err
	}
	if err := recordField("linkedin_url", company.LinkedInURL); err != nil {
		return err
	}
	if err := recordField("twitter_handle", company.TwitterHandle); err != nil {
		return err
	}

	return nil
}

// UpdateFromEnrichment updates company with enriched data
// Only updates fields that are not already set (COALESCE behavior)
func (s *Service) UpdateFromEnrichment(ctx context.Context, companyID uuid.UUID, input UpdateFromEnrichmentInput) error {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("enrichment_id", input.EnrichmentID.String()).
		Msg("Updating company from enrichment")

	// Get current company
	company, err := s.repo.GetByID(ctx, companyID)
	if err != nil {
		return fmt.Errorf("failed to get company: %w", err)
	}
	if company == nil {
		return fmt.Errorf("company not found: %s", companyID)
	}

	// Track changes for history
	type change struct {
		field    string
		oldValue *string
		newValue *string
	}
	var changes []change

	// Helper to update a string field with COALESCE logic
	updateField := func(fieldName string, current **string, newVal *string) {
		if newVal == nil || *newVal == "" {
			return
		}
		old := *current
		if old == nil || *old == "" {
			// Only update if current is empty
			*current = newVal
			oldStr := ""
			if old != nil {
				oldStr = *old
			}
			changes = append(changes, change{fieldName, &oldStr, newVal})
		}
	}

	// Update enriched fields (only if not already set)
	updateField("foundation_year", &company.FoundationYear, input.FoundationYear)
	updateField("legal_name", &company.LegalName, input.LegalName)
	updateField("headquarters", &company.Headquarters, input.Headquarters)
	updateField("sector", &company.Sector, input.Sector)
	updateField("target_audience", &company.TargetAudience, input.TargetAudience)
	updateField("value_proposition", &company.ValueProposition, input.ValueProposition)
	updateField("employees_range", &company.EmployeesRange, input.EmployeesRange)
	updateField("revenue_estimate", &company.RevenueEstimate, input.RevenueEstimate)
	updateField("business_model", &company.BusinessModel, input.BusinessModel)
	updateField("market_share_status", &company.MarketShareStatus, input.MarketShareStatus)

	// Update discovered basic fields (COALESCE)
	updateField("cnpj", &company.CNPJ, input.CNPJ)
	updateField("website", &company.Website, input.Website)
	updateField("linkedin_url", &company.LinkedInURL, input.LinkedInURL)
	updateField("twitter_handle", &company.TwitterHandle, input.TwitterHandle)

	// Handle arrays
	if len(input.Competitors) > 0 && len(company.Competitors) == 0 {
		company.Competitors = input.Competitors
	}
	if len(input.Strengths) > 0 && len(company.Strengths) == 0 {
		company.Strengths = input.Strengths
	}
	if len(input.Weaknesses) > 0 && len(company.Weaknesses) == 0 {
		company.Weaknesses = input.Weaknesses
	}

	// Handle digital maturity
	if input.DigitalMaturity != nil && company.DigitalMaturity == nil {
		company.DigitalMaturity = input.DigitalMaturity
	}

	// Nothing to update
	if len(changes) == 0 {
		s.logger.Debug().Msg("No new enrichment data to update")
		return nil
	}

	// Use transaction
	err = s.repo.WithTx(ctx, func(txRepo Repository) error {
		// Update company
		if err := txRepo.Update(ctx, company); err != nil {
			return fmt.Errorf("failed to update company: %w", err)
		}

		// Record history for each change
		for _, c := range changes {
			entry := &DataHistoryEntry{
				CompanyID: companyID,
				FieldName: c.field,
				OldValue:  c.oldValue,
				NewValue:  c.newValue,
				Source:    SourceEnrichment,
				SourceID:  &input.EnrichmentID,
			}
			if err := txRepo.RecordHistory(ctx, entry); err != nil {
				return fmt.Errorf("failed to record history for %s: %w", c.field, err)
			}
		}

		return nil
	})

	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update company from enrichment")
		return err
	}

	s.logger.Info().
		Int("fields_updated", len(changes)).
		Msg("Company updated from enrichment")

	return nil
}

// UpdateManually updates company with manual admin changes
func (s *Service) UpdateManually(ctx context.Context, companyID uuid.UUID, input ManualUpdateInput) error {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("user_id", input.UserID.String()).
		Msg("Manual company update")

	company, err := s.repo.GetByID(ctx, companyID)
	if err != nil {
		return fmt.Errorf("failed to get company: %w", err)
	}
	if company == nil {
		return fmt.Errorf("company not found: %s", companyID)
	}

	// Track changes
	type change struct {
		field    string
		oldValue *string
		newValue *string
	}
	var changes []change

	// Helper to update and track changes
	updateAndTrack := func(fieldName string, current **string, newVal *string) {
		if newVal == nil {
			return // Not being updated
		}
		old := ""
		if *current != nil {
			old = **current
		}
		*current = newVal
		changes = append(changes, change{fieldName, &old, newVal})
	}

	// Apply manual updates
	if input.Name != nil && *input.Name != company.Name {
		old := company.Name
		company.Name = *input.Name
		changes = append(changes, change{"name", &old, input.Name})
	}
	updateAndTrack("cnpj", &company.CNPJ, input.CNPJ)
	updateAndTrack("website", &company.Website, input.Website)
	updateAndTrack("industry", &company.Industry, input.Industry)
	updateAndTrack("company_size", &company.CompanySize, input.CompanySize)
	updateAndTrack("location", &company.Location, input.Location)
	updateAndTrack("target_market", &company.TargetMarket, input.TargetMarket)
	updateAndTrack("funding_stage", &company.FundingStage, input.FundingStage)
	updateAndTrack("linkedin_url", &company.LinkedInURL, input.LinkedInURL)
	updateAndTrack("twitter_handle", &company.TwitterHandle, input.TwitterHandle)

	if len(changes) == 0 {
		s.logger.Debug().Msg("No manual changes to apply")
		return nil
	}

	// Use transaction
	err = s.repo.WithTx(ctx, func(txRepo Repository) error {
		if err := txRepo.Update(ctx, company); err != nil {
			return fmt.Errorf("failed to update company: %w", err)
		}

		for _, c := range changes {
			entry := &DataHistoryEntry{
				CompanyID: companyID,
				FieldName: c.field,
				OldValue:  c.oldValue,
				NewValue:  c.newValue,
				Source:    SourceManual,
				ChangedBy: &input.UserID,
			}
			if err := txRepo.RecordHistory(ctx, entry); err != nil {
				return fmt.Errorf("failed to record history: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		s.logger.Error().Err(err).Msg("Failed manual company update")
		return err
	}

	s.logger.Info().
		Int("fields_updated", len(changes)).
		Msg("Manual company update completed")

	return nil
}

// GetByID retrieves a company by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Company, error) {
	return s.repo.GetByID(ctx, id)
}

// IsVerifiedCNPJExists checks if a verified company already has this CNPJ
// Used to block new submissions with the same CNPJ
func (s *Service) IsVerifiedCNPJExists(ctx context.Context, cnpj string) (bool, error) {
	if cnpj == "" {
		return false, nil
	}
	return s.repo.IsVerifiedCNPJExists(ctx, cnpj)
}

// GetVerifiedByCNPJ retrieves a verified company by CNPJ
func (s *Service) GetVerifiedByCNPJ(ctx context.Context, cnpj string) (*Company, error) {
	if cnpj == "" {
		return nil, nil
	}
	return s.repo.GetVerifiedByCNPJ(ctx, cnpj)
}

// GetBySubmissionID retrieves the company linked to a submission
func (s *Service) GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*Company, error) {
	return s.repo.GetBySubmissionID(ctx, submissionID)
}

// GetWithHistory retrieves a company with its full data history
func (s *Service) GetWithHistory(ctx context.Context, id uuid.UUID) (*CompanyWithHistory, error) {
	company, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, nil
	}

	history, err := s.repo.GetHistory(ctx, id)
	if err != nil {
		return nil, err
	}

	return &CompanyWithHistory{
		Company: company,
		History: history,
	}, nil
}

// GetWithSubmissions retrieves a company with all linked submissions
func (s *Service) GetWithSubmissions(ctx context.Context, id uuid.UUID) (*CompanyWithSubmissions, error) {
	company, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, nil
	}

	submissions, err := s.repo.GetSubmissions(ctx, id)
	if err != nil {
		return nil, err
	}

	return &CompanyWithSubmissions{
		Company:     company,
		Submissions: submissions,
	}, nil
}

// GetLastSubmission retrieves the most recent submission linked to this company
func (s *Service) GetLastSubmission(ctx context.Context, companyID uuid.UUID) (*LastSubmissionInfo, error) {
	return s.repo.GetLastSubmissionForCompany(ctx, companyID)
}

// GetLastCompletedEnrichment retrieves the most recent completed enrichment for this company
func (s *Service) GetLastCompletedEnrichment(ctx context.Context, companyID uuid.UUID) (*LastEnrichmentInfo, error) {
	return s.repo.GetLastCompletedEnrichmentForCompany(ctx, companyID)
}

// GetLatestEnrichmentStatus retrieves the most recent enrichment status for admin dashboard
func (s *Service) GetLatestEnrichmentStatus(ctx context.Context, companyID uuid.UUID) (*EnrichmentStatus, error) {
	return s.repo.GetLatestEnrichmentStatus(ctx, companyID)
}

// GetEnrichmentsHistory retrieves all enrichments for a company (for admin dashboard history)
func (s *Service) GetEnrichmentsHistory(ctx context.Context, companyID uuid.UUID) ([]*EnrichmentStatus, error) {
	return s.repo.GetEnrichmentsHistory(ctx, companyID)
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

// AddUserToCompany adds a user to the company's allowed_users array
// Only owner or admin can call this
func (s *Service) AddUserToCompany(ctx context.Context, companyID, userID uuid.UUID) error {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("user_id", userID.String()).
		Msg("Adding user to company")

	company, err := s.repo.GetByID(ctx, companyID)
	if err != nil {
		return fmt.Errorf("failed to get company: %w", err)
	}
	if company == nil {
		return fmt.Errorf("company not found: %s", companyID)
	}

	// Check if user is already in allowed_users
	if company.IsUserAllowed(userID) {
		s.logger.Debug().Msg("User already has access to company")
		return nil // Already has access
	}

	// Add user to allowed_users
	company.AllowedUsers = append(company.AllowedUsers, userID)

	if err := s.repo.Update(ctx, company); err != nil {
		s.logger.Error().Err(err).Msg("Failed to add user to company")
		return fmt.Errorf("failed to add user to company: %w", err)
	}

	s.logger.Info().Msg("User added to company successfully")
	return nil
}

// RemoveUserFromCompany removes a user from the company's allowed_users array
// Only owner or admin can call this. Cannot remove the owner.
func (s *Service) RemoveUserFromCompany(ctx context.Context, companyID, userID uuid.UUID) error {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("user_id", userID.String()).
		Msg("Removing user from company")

	company, err := s.repo.GetByID(ctx, companyID)
	if err != nil {
		return fmt.Errorf("failed to get company: %w", err)
	}
	if company == nil {
		return fmt.Errorf("company not found: %s", companyID)
	}

	// Cannot remove owner
	if company.IsOwner(userID) {
		return fmt.Errorf("cannot remove owner from company")
	}

	// Remove user from allowed_users
	newAllowedUsers := make(UUIDSlice, 0, len(company.AllowedUsers))
	for _, u := range company.AllowedUsers {
		if u != userID {
			newAllowedUsers = append(newAllowedUsers, u)
		}
	}
	company.AllowedUsers = newAllowedUsers

	if err := s.repo.Update(ctx, company); err != nil {
		s.logger.Error().Err(err).Msg("Failed to remove user from company")
		return fmt.Errorf("failed to remove user from company: %w", err)
	}

	s.logger.Info().Msg("User removed from company successfully")
	return nil
}

// VerifyCompany sets is_verified=true, which locks the CNPJ via unique constraint
func (s *Service) VerifyCompany(ctx context.Context, companyID uuid.UUID) (*Company, error) {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Msg("Verifying company")

	company, err := s.repo.GetByID(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get company: %w", err)
	}
	if company == nil {
		return nil, fmt.Errorf("company not found: %s", companyID)
	}

	if company.IsVerified {
		s.logger.Debug().Msg("Company is already verified")
		return company, nil
	}

	company.IsVerified = true

	if err := s.repo.Update(ctx, company); err != nil {
		s.logger.Error().Err(err).Msg("Failed to verify company")
		return nil, fmt.Errorf("failed to verify company: %w", err)
	}

	s.logger.Info().Msg("Company verified successfully")
	return company, nil
}

// UnverifyCompany sets is_verified=false (admin only, for corrections)
func (s *Service) UnverifyCompany(ctx context.Context, companyID uuid.UUID) (*Company, error) {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Msg("Unverifying company")

	company, err := s.repo.GetByID(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get company: %w", err)
	}
	if company == nil {
		return nil, fmt.Errorf("company not found: %s", companyID)
	}

	if !company.IsVerified {
		return company, nil
	}

	company.IsVerified = false

	if err := s.repo.Update(ctx, company); err != nil {
		return nil, fmt.Errorf("failed to unverify company: %w", err)
	}

	s.logger.Info().Msg("Company unverified successfully")
	return company, nil
}

// TransferOwnership transfers company ownership to another user (admin only)
func (s *Service) TransferOwnership(ctx context.Context, companyID, newOwnerID uuid.UUID) (*Company, error) {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("new_owner_id", newOwnerID.String()).
		Msg("Transferring company ownership")

	company, err := s.repo.GetByID(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get company: %w", err)
	}
	if company == nil {
		return nil, fmt.Errorf("company not found: %s", companyID)
	}

	company.OwnerID = &newOwnerID

	// Ensure new owner is in allowed_users
	if !company.AllowedUsers.Contains(newOwnerID) {
		company.AllowedUsers = append(company.AllowedUsers, newOwnerID)
	}

	if err := s.repo.Update(ctx, company); err != nil {
		return nil, fmt.Errorf("failed to transfer ownership: %w", err)
	}

	s.logger.Info().Msg("Company ownership transferred successfully")
	return company, nil
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

// GetHistory retrieves all history entries for a company
func (s *Service) GetHistory(ctx context.Context, companyID uuid.UUID) ([]*DataHistoryEntry, error) {
	return s.repo.GetHistory(ctx, companyID)
}

// GetSubmissions retrieves all submissions linked to a company
func (s *Service) GetSubmissions(ctx context.Context, companyID uuid.UUID) ([]*CompanySubmission, error) {
	return s.repo.GetSubmissions(ctx, companyID)
}

// ==================== Smart Merge (Field Verification) ====================

// UpdateFromEnrichmentSmartMerge updates company with enriched data using smart merge logic
// Only updates fields that are NOT verified (verified fields are protected from re-enrichment)
func (s *Service) UpdateFromEnrichmentSmartMerge(ctx context.Context, companyID uuid.UUID, input UpdateFromEnrichmentInput) error {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("enrichment_id", input.EnrichmentID.String()).
		Msg("Smart merge: updating company from enrichment")

	// Get current company
	company, err := s.repo.GetByID(ctx, companyID)
	if err != nil {
		return fmt.Errorf("failed to get company: %w", err)
	}
	if company == nil {
		return fmt.Errorf("company not found: %s", companyID)
	}

	// Get verified fields to protect them
	verifiedFields, err := s.repo.GetVerifiedFieldNames(ctx, companyID)
	if err != nil {
		return fmt.Errorf("failed to get verified fields: %w", err)
	}
	verifiedSet := make(map[string]bool, len(verifiedFields))
	for _, f := range verifiedFields {
		verifiedSet[f] = true
	}

	s.logger.Debug().
		Int("verified_field_count", len(verifiedFields)).
		Interface("verified_fields", verifiedFields).
		Msg("Protected fields that will not be overwritten")

	// Track changes for history
	type change struct {
		field    string
		oldValue *string
		newValue *string
	}
	var changes []change

	// Helper to update a string field with smart merge logic:
	// - Skip if field is verified (protected)
	// - Overwrite if field is NOT verified (regardless of current value)
	updateFieldSmart := func(fieldName string, current **string, newVal *string) {
		if newVal == nil || *newVal == "" {
			return // No new value to apply
		}
		if verifiedSet[fieldName] {
			s.logger.Debug().
				Str("field", fieldName).
				Msg("Field is verified - skipping update")
			return // Field is protected
		}
		old := *current
		if old != nil && *old == *newVal {
			return // Same value, no change needed
		}
		oldStr := ""
		if old != nil {
			oldStr = *old
		}
		*current = newVal
		changes = append(changes, change{fieldName, &oldStr, newVal})
	}

	// Apply enriched fields with smart merge
	updateFieldSmart("foundation_year", &company.FoundationYear, input.FoundationYear)
	updateFieldSmart("legal_name", &company.LegalName, input.LegalName)
	updateFieldSmart("headquarters", &company.Headquarters, input.Headquarters)
	updateFieldSmart("sector", &company.Sector, input.Sector)
	updateFieldSmart("target_audience", &company.TargetAudience, input.TargetAudience)
	updateFieldSmart("value_proposition", &company.ValueProposition, input.ValueProposition)
	updateFieldSmart("employees_range", &company.EmployeesRange, input.EmployeesRange)
	updateFieldSmart("revenue_estimate", &company.RevenueEstimate, input.RevenueEstimate)
	updateFieldSmart("business_model", &company.BusinessModel, input.BusinessModel)
	updateFieldSmart("market_share_status", &company.MarketShareStatus, input.MarketShareStatus)

	// Discovered basic fields (also smart merge)
	updateFieldSmart("cnpj", &company.CNPJ, input.CNPJ)
	updateFieldSmart("website", &company.Website, input.Website)
	updateFieldSmart("linkedin_url", &company.LinkedInURL, input.LinkedInURL)
	updateFieldSmart("twitter_handle", &company.TwitterHandle, input.TwitterHandle)

	// Handle arrays (only if not verified)
	if !verifiedSet["competitors"] && len(input.Competitors) > 0 {
		company.Competitors = input.Competitors
	}
	if !verifiedSet["strengths"] && len(input.Strengths) > 0 {
		company.Strengths = input.Strengths
	}
	if !verifiedSet["weaknesses"] && len(input.Weaknesses) > 0 {
		company.Weaknesses = input.Weaknesses
	}

	// Handle digital maturity (only if not verified)
	if !verifiedSet["digital_maturity"] && input.DigitalMaturity != nil {
		company.DigitalMaturity = input.DigitalMaturity
	}

	// Nothing to update
	if len(changes) == 0 {
		s.logger.Debug().Msg("No fields updated (all changes blocked by verified fields or no new data)")
		return nil
	}

	// Use transaction
	err = s.repo.WithTx(ctx, func(txRepo Repository) error {
		// Update company
		if err := txRepo.Update(ctx, company); err != nil {
			return fmt.Errorf("failed to update company: %w", err)
		}

		// Record history for each change
		for _, c := range changes {
			entry := &DataHistoryEntry{
				CompanyID: companyID,
				FieldName: c.field,
				OldValue:  c.oldValue,
				NewValue:  c.newValue,
				Source:    SourceEnrichment,
				SourceID:  &input.EnrichmentID,
			}
			if err := txRepo.RecordHistory(ctx, entry); err != nil {
				return fmt.Errorf("failed to record history for %s: %w", c.field, err)
			}
		}

		return nil
	})

	if err != nil {
		s.logger.Error().Err(err).Msg("Failed smart merge update from enrichment")
		return err
	}

	s.logger.Info().
		Int("fields_updated", len(changes)).
		Int("fields_protected", len(verifiedFields)).
		Msg("Smart merge completed successfully")

	return nil
}

// ==================== Field Verification Service Methods ====================

// GetFieldVerifications retrieves all field verifications for a company
func (s *Service) GetFieldVerifications(ctx context.Context, companyID uuid.UUID) ([]*FieldVerification, error) {
	return s.repo.GetFieldVerifications(ctx, companyID)
}

// GetWithVerifications retrieves a company with its field verification status
func (s *Service) GetWithVerifications(ctx context.Context, id uuid.UUID) (*CompanyWithVerifications, error) {
	company, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, nil
	}

	verifications, err := s.repo.GetFieldVerifications(ctx, id)
	if err != nil {
		return nil, err
	}

	return &CompanyWithVerifications{
		Company:       company,
		Verifications: verifications,
	}, nil
}

// VerifyField marks a specific field as verified (protected from re-enrichment)
func (s *Service) VerifyField(ctx context.Context, companyID uuid.UUID, fieldName string, verifiedBy *uuid.UUID) error {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("field_name", fieldName).
		Msg("Verifying field")

	// Validate field name is in verifiable list
	valid := false
	for _, f := range VerifiableFields {
		if f == fieldName {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("field '%s' is not a verifiable field", fieldName)
	}

	if err := s.repo.VerifyField(ctx, companyID, fieldName, verifiedBy); err != nil {
		s.logger.Error().Err(err).Msg("Failed to verify field")
		return err
	}

	s.logger.Info().Msg("Field verified successfully")
	return nil
}

// UnverifyField removes verification from a field (allows future re-enrichment)
func (s *Service) UnverifyField(ctx context.Context, companyID uuid.UUID, fieldName string) error {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("field_name", fieldName).
		Msg("Unverifying field")

	if err := s.repo.UnverifyField(ctx, companyID, fieldName); err != nil {
		s.logger.Error().Err(err).Msg("Failed to unverify field")
		return err
	}

	s.logger.Info().Msg("Field unverified successfully")
	return nil
}

// VerifyFields verifies multiple fields at once
func (s *Service) VerifyFields(ctx context.Context, companyID uuid.UUID, fieldNames []string, verifiedBy *uuid.UUID) error {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Int("field_count", len(fieldNames)).
		Msg("Verifying multiple fields")

	return s.repo.WithTx(ctx, func(txRepo Repository) error {
		for _, fieldName := range fieldNames {
			// Validate field name
			valid := false
			for _, f := range VerifiableFields {
				if f == fieldName {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("field '%s' is not a verifiable field", fieldName)
			}

			if err := txRepo.VerifyField(ctx, companyID, fieldName, verifiedBy); err != nil {
				return fmt.Errorf("failed to verify field %s: %w", fieldName, err)
			}
		}
		return nil
	})
}

// VerifyAllFields marks all fields as verified for a company
func (s *Service) VerifyAllFields(ctx context.Context, companyID uuid.UUID, verifiedBy *uuid.UUID) error {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Msg("Verifying all fields")

	if err := s.repo.VerifyAllFields(ctx, companyID, verifiedBy); err != nil {
		s.logger.Error().Err(err).Msg("Failed to verify all fields")
		return err
	}

	s.logger.Info().Msg("All fields verified successfully")
	return nil
}

// UnverifyAllFields removes all field verifications for a company
func (s *Service) UnverifyAllFields(ctx context.Context, companyID uuid.UUID) error {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Msg("Unverifying all fields")

	if err := s.repo.UnverifyAllFields(ctx, companyID); err != nil {
		s.logger.Error().Err(err).Msg("Failed to unverify all fields")
		return err
	}

	s.logger.Info().Msg("All fields unverified successfully")
	return nil
}

// GetVerifiedFieldNames returns just the names of verified fields for quick checks
func (s *Service) GetVerifiedFieldNames(ctx context.Context, companyID uuid.UUID) ([]string, error) {
	return s.repo.GetVerifiedFieldNames(ctx, companyID)
}

// ==================== Field Deprecation ====================

// ExpireFieldVerifications removes expired field verifications based on deprecation config
// If companyID is nil, checks all companies (for cron job)
func (s *Service) ExpireFieldVerifications(ctx context.Context, companyID *uuid.UUID) (int, error) {
	s.logger.Info().
		Interface("company_id", companyID).
		Msg("Expiring field verifications")

	count, err := s.repo.ExpireFieldVerifications(ctx, companyID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to expire field verifications")
		return 0, fmt.Errorf("failed to expire field verifications: %w", err)
	}

	if count > 0 {
		s.logger.Info().
			Int("expired_count", count).
			Msg("Field verifications expired successfully")
	}

	return count, nil
}

// UpdateFieldsWithAutoVerification updates company fields and auto-verifies them
// This is for admin manual edits that should become verified
func (s *Service) UpdateFieldsWithAutoVerification(ctx context.Context, companyID uuid.UUID, fields map[string]interface{}, userID uuid.UUID) (*Company, error) {
	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("user_id", userID.String()).
		Int("field_count", len(fields)).
		Msg("Updating company fields with auto-verification")

	// Get current company
	company, err := s.repo.GetByID(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get company: %w", err)
	}
	if company == nil {
		return nil, fmt.Errorf("company not found: %s", companyID)
	}

	// Track changes for history
	type change struct {
		field    string
		oldValue *string
		newValue *string
	}
	var changes []change
	var fieldsToVerify []string

	// Helper to get string pointer from interface
	toString := func(v interface{}) *string {
		if v == nil {
			return nil
		}
		if s, ok := v.(string); ok && s != "" {
			return &s
		}
		return nil
	}

	// Process each field
	for fieldName, newVal := range fields {
		newStrVal := toString(newVal)
		if newStrVal == nil {
			continue // Skip empty values
		}

		var oldVal *string
		switch fieldName {
		case "name":
			if company.Name != *newStrVal {
				oldVal = &company.Name
				company.Name = *newStrVal
			}
		case "cnpj":
			oldVal = company.CNPJ
			company.CNPJ = newStrVal
		case "website":
			oldVal = company.Website
			company.Website = newStrVal
		case "industry":
			oldVal = company.Industry
			company.Industry = newStrVal
		case "sector":
			oldVal = company.Sector
			company.Sector = newStrVal
		case "company_size":
			oldVal = company.CompanySize
			company.CompanySize = newStrVal
		case "location":
			oldVal = company.Location
			company.Location = newStrVal
		case "target_market":
			oldVal = company.TargetMarket
			company.TargetMarket = newStrVal
		case "funding_stage":
			oldVal = company.FundingStage
			company.FundingStage = newStrVal
		case "linkedin_url":
			oldVal = company.LinkedInURL
			company.LinkedInURL = newStrVal
		case "twitter_handle":
			oldVal = company.TwitterHandle
			company.TwitterHandle = newStrVal
		case "foundation_year":
			oldVal = company.FoundationYear
			company.FoundationYear = newStrVal
		case "legal_name":
			oldVal = company.LegalName
			company.LegalName = newStrVal
		case "headquarters":
			oldVal = company.Headquarters
			company.Headquarters = newStrVal
		case "target_audience":
			oldVal = company.TargetAudience
			company.TargetAudience = newStrVal
		case "value_proposition":
			oldVal = company.ValueProposition
			company.ValueProposition = newStrVal
		case "employees_range":
			oldVal = company.EmployeesRange
			company.EmployeesRange = newStrVal
		case "revenue_estimate":
			oldVal = company.RevenueEstimate
			company.RevenueEstimate = newStrVal
		case "business_model":
			oldVal = company.BusinessModel
			company.BusinessModel = newStrVal
		case "market_share_status":
			oldVal = company.MarketShareStatus
			company.MarketShareStatus = newStrVal
		default:
			s.logger.Debug().Str("field", fieldName).Msg("Unknown field, skipping")
			continue
		}

		// Check if value actually changed
		oldStr := ""
		if oldVal != nil {
			oldStr = *oldVal
		}
		if oldStr != *newStrVal {
			changes = append(changes, change{fieldName, &oldStr, newStrVal})
			fieldsToVerify = append(fieldsToVerify, fieldName)
		}
	}

	if len(changes) == 0 {
		s.logger.Debug().Msg("No changes to apply")
		return company, nil
	}

	// Use transaction
	err = s.repo.WithTx(ctx, func(txRepo Repository) error {
		// 1. Update company
		if err := txRepo.Update(ctx, company); err != nil {
			return fmt.Errorf("failed to update company: %w", err)
		}

		// 2. Record history for each change
		for _, c := range changes {
			entry := &DataHistoryEntry{
				CompanyID: companyID,
				FieldName: c.field,
				OldValue:  c.oldValue,
				NewValue:  c.newValue,
				Source:    SourceManual,
				ChangedBy: &userID,
			}
			if err := txRepo.RecordHistory(ctx, entry); err != nil {
				return fmt.Errorf("failed to record history for %s: %w", c.field, err)
			}
		}

		// 3. Auto-verify all updated fields
		for _, fieldName := range fieldsToVerify {
			// Check if field is in verifiable list
			valid := false
			for _, f := range VerifiableFields {
				if f == fieldName {
					valid = true
					break
				}
			}
			if valid {
				if err := txRepo.VerifyField(ctx, companyID, fieldName, &userID); err != nil {
					// Log warning but don't fail - verification is a bonus
					s.logger.Warn().
						Err(err).
						Str("field", fieldName).
						Msg("Failed to auto-verify field")
				}
			}
		}

		return nil
	})

	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update company fields")
		return nil, err
	}

	// Reload company to get fresh data
	company, _ = s.repo.GetByID(ctx, companyID)

	s.logger.Info().
		Int("fields_updated", len(changes)).
		Int("fields_verified", len(fieldsToVerify)).
		Msg("Company fields updated and verified successfully")

	return company, nil
}
