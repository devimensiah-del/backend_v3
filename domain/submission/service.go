package submission

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// SERVICE
// =============================================================================

// Service provides submission business logic.
type Service struct {
	repo             Repository
	companyService   CompanyServiceInterface
	challengeService ChallengeServiceInterface
}

// NewService creates a new submission service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SetCompanyService injects the company service (optional, for graceful degradation).
func (s *Service) SetCompanyService(companySvc CompanyServiceInterface) {
	s.companyService = companySvc
}

// SetChallengeService injects the challenge service.
func (s *Service) SetChallengeService(challengeSvc ChallengeServiceInterface) {
	s.challengeService = challengeSvc
}

// =============================================================================
// MAIN WORKFLOW: SubmitForm
// =============================================================================

// SubmitForm handles the complete public submission workflow
func (s *Service) SubmitForm(ctx context.Context, req *SubmitRequest) (*SubmitFormResponse, error) {
	// Normalize input first (handles URL schemes, case normalization)
	s.normalizeRequest(req)

	// Validate input
	if err := s.validateSubmitRequest(req); err != nil {
		return nil, err
	}

	// Ensure services are available
	if s.companyService == nil {
		return nil, NewWorkflowError("setup", fmt.Errorf("company service not configured"))
	}
	if s.challengeService == nil {
		return nil, NewWorkflowError("setup", fmt.Errorf("challenge service not configured"))
	}

	// Track created entities for saga rollback
	var submissionID, companyID, challengeID uuid.UUID
	var rollbackNeeded bool

	defer func() {
		if rollbackNeeded {
			s.rollbackSaga(ctx, submissionID, companyID, challengeID)
		}
	}()

	// Step 1: Create submission
	submission := s.buildSubmissionFromRequest(req)
	if err := submission.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, submission); err != nil {
		return nil, NewWorkflowError("submission", err)
	}
	submissionID = submission.ID
	rollbackNeeded = true // Enable rollback from this point

	log.Info().
		Str("id", submission.ID.String()).
		Str("company", submission.CompanyName).
		Msg("Submission created")

	// Step 2: Create company (triggers enrichment)
	companyInput := CompanyCreateInput{
		SubmissionID:     submission.ID,
		CompanyName:      submission.CompanyName,
		CNPJ:             submission.CNPJ,
		Website:          submission.CompanyWebsite,
		Industry:         submission.CompanyIndustry,
		CompanySize:      submission.CompanySize,
		Location:         submission.CompanyLocation,
		TargetMarket:     submission.TargetMarket,
		FundingStage:     submission.FundingStage,
		AnnualRevenueMin: submission.AnnualRevenueMin,
		AnnualRevenueMax: submission.AnnualRevenueMax,
		LinkedInURL:      submission.LinkedInURL,
		TwitterHandle:    submission.TwitterHandle,
		OwnerID:          req.UserID,
	}

	companyResult, err := s.companyService.CreateFromSubmission(ctx, companyInput)
	if err != nil {
		log.Error().Err(err).Str("submission_id", submission.ID.String()).Msg("Company creation failed")
		return nil, NewWorkflowError("company", err)
	}
	companyID = companyResult.ID

	log.Info().
		Str("submission_id", submission.ID.String()).
		Str("company_id", companyResult.ID.String()).
		Msg("Company created")

	// Step 3: Create challenge
	challengeInput := ChallengeCreateInput{
		CompanyID:         companyResult.ID,
		ChallengeCategory: req.ChallengeCategory,
		ChallengeType:     req.ChallengeType,
		BusinessChallenge: req.BusinessChallenge,
	}

	challengeID, err = s.challengeService.CreateFromInput(ctx, challengeInput)
	if err != nil {
		log.Error().Err(err).Str("submission_id", submission.ID.String()).Msg("Challenge creation failed")
		return nil, NewWorkflowError("challenge", err)
	}

	log.Info().
		Str("submission_id", submission.ID.String()).
		Str("company_id", companyResult.ID.String()).
		Str("challenge_id", challengeID.String()).
		Str("category", req.ChallengeCategory).
		Msg("Challenge created")

	// All steps succeeded - disable rollback
	rollbackNeeded = false

	return &SubmitFormResponse{
		SubmissionID: submission.ID,
		CompanyID:    companyResult.ID,
		ChallengeID:  challengeID,
	}, nil
}

// rollbackSaga performs compensating transactions to clean up partial saga state.
// It's best-effort - failures are logged but don't propagate.
// Rollback order is reverse of creation: Challenge → Company → Submission.
func (s *Service) rollbackSaga(ctx context.Context, submissionID, companyID, challengeID uuid.UUID) {
	logger := log.With().Str("saga", "rollback").Logger()

	// Rollback in reverse order (challenge → company → submission)
	if challengeID != uuid.Nil && s.challengeService != nil {
		if err := s.challengeService.DeleteChallenge(ctx, challengeID); err != nil {
			logger.Error().
				Err(err).
				Str("challenge_id", challengeID.String()).
				Msg("Failed to rollback challenge")
		} else {
			logger.Info().
				Str("challenge_id", challengeID.String()).
				Msg("Rolled back challenge")
		}
	}

	if companyID != uuid.Nil && s.companyService != nil {
		if err := s.companyService.DeleteCompany(ctx, companyID); err != nil {
			logger.Error().
				Err(err).
				Str("company_id", companyID.String()).
				Msg("Failed to rollback company")
		} else {
			logger.Info().
				Str("company_id", companyID.String()).
				Msg("Rolled back company")
		}
	}

	if submissionID != uuid.Nil {
		if err := s.repo.Delete(ctx, submissionID); err != nil {
			logger.Error().
				Err(err).
				Str("submission_id", submissionID.String()).
				Msg("Failed to rollback submission")
		} else {
			logger.Info().
				Str("submission_id", submissionID.String()).
				Msg("Rolled back submission")
		}
	}
}

// =============================================================================
// CRUD OPERATIONS
// =============================================================================

// Create creates a new submission (low-level, prefer SubmitForm for full workflow).
func (s *Service) Create(ctx context.Context, sub *Submission) (*Submission, error) {
	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = time.Now()
	}
	if sub.UpdatedAt.IsZero() {
		sub.UpdatedAt = time.Now()
	}

	if err := sub.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, err
	}

	log.Info().Str("id", sub.ID.String()).Str("company", sub.CompanyName).Msg("Submission created")
	return sub, nil
}

// GetByID retrieves a submission by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Submission, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByEmail retrieves all submissions for an email address.
func (s *Service) GetByEmail(ctx context.Context, email string) ([]*Submission, error) {
	return s.repo.GetByEmail(ctx, email)
}

// List retrieves submissions with pagination and filtering.
func (s *Service) List(ctx context.Context, opts *ListOptions) ([]*Submission, int, error) {
	return s.repo.List(ctx, opts)
}

// Delete soft-deletes a submission.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// =============================================================================
// ADMIN WORKFLOW: CreateFromCompany
// =============================================================================

// CreateFromCompany creates a submission from existing company data.
// Used by admin re-enrich/re-analyze workflows. Does NOT create company.
func (s *Service) CreateFromCompany(ctx context.Context, input CreateFromCompanyInput) (*Submission, error) {
	now := time.Now()

	sub := &Submission{
		ID:               uuid.New(),
		CompanyName:      input.CompanyName,
		CNPJ:             input.CNPJ,
		CompanyWebsite:   input.Website,
		CompanyIndustry:  input.Industry,
		CompanySize:      input.CompanySize,
		CompanyLocation:  input.Location,
		ContactName:      input.ContactName,
		ContactEmail:     input.ContactEmail,
		TargetMarket:     input.TargetMarket,
		AnnualRevenueMin: input.AnnualRevenueMin,
		AnnualRevenueMax: input.AnnualRevenueMax,
		FundingStage:     input.FundingStage,
		LinkedInURL:      input.LinkedInURL,
		TwitterHandle:    input.TwitterHandle,
		UserID:           input.UserID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := sub.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, err
	}

	log.Info().
		Str("id", sub.ID.String()).
		Str("company", sub.CompanyName).
		Str("company_id", input.CompanyID.String()).
		Msg("Submission created from company data")

	return sub, nil
}

// =============================================================================
// ANONYMOUS LINKING
// =============================================================================

// LinkAnonymousToUser links all anonymous submissions with matching email to a user.
// Called after signup to connect pre-existing submissions.
func (s *Service) LinkAnonymousToUser(ctx context.Context, userID uuid.UUID, email string) error {
	submissions, err := s.repo.GetAnonymousByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to find anonymous submissions: %w", err)
	}
	if len(submissions) == 0 {
		log.Debug().Str("email", email).Msg("No anonymous submissions to link")
		return nil
	}

	log.Info().
		Str("email", email).
		Str("user_id", userID.String()).
		Int("count", len(submissions)).
		Msg("Linking anonymous submissions to new user")

	linkedCount := 0
	for _, sub := range submissions {
		if err := s.repo.UpdateUserID(ctx, sub.ID, userID); err != nil {
			log.Error().Err(err).Str("submission_id", sub.ID.String()).Msg("Failed to link submission")
			continue
		}
		linkedCount++

		// Update company ownership if service available
		if s.companyService != nil {
			if err := s.companyService.SetOwnerFromSubmission(ctx, sub.ID, userID); err != nil {
				log.Warn().Err(err).Str("submission_id", sub.ID.String()).Msg("Failed to update company owner")
			}
		}
	}

	log.Info().
		Str("email", email).
		Int("linked_count", linkedCount).
		Int("total_found", len(submissions)).
		Msg("Anonymous submission linking completed")

	return nil
}

// =============================================================================
// VALIDATION HELPERS
// =============================================================================

// validateSubmitRequest validates all fields of a SubmitRequest.
// Note: Call normalizeRequest() before this to ensure fields are normalized.
func (s *Service) validateSubmitRequest(req *SubmitRequest) error {
	// Challenge fields (required)
	if req.ChallengeCategory == "" {
		return NewValidationError("challenge_category", "is required")
	}
	if req.ChallengeType == "" {
		return NewValidationError("challenge_type", "is required")
	}
	if req.BusinessChallenge == "" {
		return NewValidationError("business_challenge", "is required")
	}

	// Validate using challenge service (delegates to challenge domain)
	if s.challengeService != nil {
		if !s.challengeService.ValidateCategory(req.ChallengeCategory) {
			return NewValidationError("challenge_category", fmt.Sprintf("invalid value '%s'", req.ChallengeCategory))
		}
		if !s.challengeService.ValidateType(req.ChallengeCategory, req.ChallengeType) {
			return NewValidationError("challenge_type", fmt.Sprintf("'%s' not valid for category '%s'", req.ChallengeType, req.ChallengeCategory))
		}
	}

	// URL validation (does not mutate)
	if err := s.validateURLs(req); err != nil {
		return err
	}

	// String length validation
	if err := s.validateLengths(req); err != nil {
		return err
	}

	// Revenue range validation
	if req.AnnualRevenueMin != nil && req.AnnualRevenueMax != nil {
		if *req.AnnualRevenueMin > *req.AnnualRevenueMax {
			return NewValidationError("annual_revenue", "min cannot be greater than max")
		}
	}

	return nil
}

// normalizeRequest normalizes fields before validation.
// This is separate from validation to maintain single responsibility.
func (s *Service) normalizeRequest(req *SubmitRequest) {
	// Normalize category and type to lowercase
	req.ChallengeCategory = strings.ToLower(strings.TrimSpace(req.ChallengeCategory))
	req.ChallengeType = strings.ToLower(strings.TrimSpace(req.ChallengeType))

	// Normalize URLs - add https:// if missing scheme
	if req.CompanyWebsite != nil && *req.CompanyWebsite != "" {
		website := strings.TrimSpace(*req.CompanyWebsite)
		if !strings.HasPrefix(website, "http://") && !strings.HasPrefix(website, "https://") {
			website = "https://" + website
		}
		req.CompanyWebsite = &website
	}

	if req.LinkedInURL != nil && *req.LinkedInURL != "" {
		linkedIn := strings.TrimSpace(*req.LinkedInURL)
		if !strings.HasPrefix(linkedIn, "http://") && !strings.HasPrefix(linkedIn, "https://") {
			linkedIn = "https://" + linkedIn
		}
		req.LinkedInURL = &linkedIn
	}
}

// validateURLs validates URL fields (does not mutate - use normalizeRequest first).
func (s *Service) validateURLs(req *SubmitRequest) error {
	if req.CompanyWebsite != nil && *req.CompanyWebsite != "" {
		if _, err := url.ParseRequestURI(*req.CompanyWebsite); err != nil {
			return NewValidationError("company_website", "invalid URL format")
		}
	}

	if req.LinkedInURL != nil && *req.LinkedInURL != "" {
		if _, err := url.ParseRequestURI(*req.LinkedInURL); err != nil {
			return NewValidationError("linkedin_url", "invalid URL format")
		}
		// Must be a LinkedIn URL
		if !strings.Contains(strings.ToLower(*req.LinkedInURL), "linkedin.com") {
			return NewValidationError("linkedin_url", "must be a LinkedIn URL")
		}
	}

	return nil
}

// validateLengths validates string lengths.
func (s *Service) validateLengths(req *SubmitRequest) error {
	if len(req.CompanyName) > MaxCompanyNameLength {
		return NewValidationError("company_name", fmt.Sprintf("exceeds %d characters", MaxCompanyNameLength))
	}
	if len(req.ContactName) > MaxContactNameLength {
		return NewValidationError("contact_name", fmt.Sprintf("exceeds %d characters", MaxContactNameLength))
	}
	if len(req.ContactEmail) > MaxEmailLength {
		return NewValidationError("contact_email", fmt.Sprintf("exceeds %d characters", MaxEmailLength))
	}
	if req.ContactPhone != nil && len(*req.ContactPhone) > MaxPhoneLength {
		return NewValidationError("contact_phone", fmt.Sprintf("exceeds %d characters", MaxPhoneLength))
	}
	if len(req.BusinessChallenge) > MaxBusinessChallengeLength {
		return NewValidationError("business_challenge", fmt.Sprintf("exceeds %d characters", MaxBusinessChallengeLength))
	}
	if req.AdditionalNotes != nil && len(*req.AdditionalNotes) > MaxAdditionalNotesLength {
		return NewValidationError("additional_notes", fmt.Sprintf("exceeds %d characters", MaxAdditionalNotesLength))
	}
	if req.CompanyWebsite != nil && len(*req.CompanyWebsite) > MaxURLLength {
		return NewValidationError("company_website", fmt.Sprintf("exceeds %d characters", MaxURLLength))
	}
	if req.LinkedInURL != nil && len(*req.LinkedInURL) > MaxURLLength {
		return NewValidationError("linkedin_url", fmt.Sprintf("exceeds %d characters", MaxURLLength))
	}
	return nil
}

// buildSubmissionFromRequest creates a Submission entity from a SubmitRequest.
func (s *Service) buildSubmissionFromRequest(req *SubmitRequest) *Submission {
	now := time.Now()
	return &Submission{
		ID:               uuid.New(),
		CompanyName:      req.CompanyName,
		CNPJ:             req.CNPJ,
		CompanyWebsite:   req.CompanyWebsite,
		CompanyIndustry:  req.CompanyIndustry,
		CompanySize:      req.CompanySize,
		CompanyLocation:  req.CompanyLocation,
		ContactName:      req.ContactName,
		ContactEmail:     req.ContactEmail,
		ContactPhone:     req.ContactPhone,
		ContactPosition:  req.ContactPosition,
		TargetMarket:     req.TargetMarket,
		AnnualRevenueMin: req.AnnualRevenueMin,
		AnnualRevenueMax: req.AnnualRevenueMax,
		FundingStage:     req.FundingStage,
		AdditionalNotes:  req.AdditionalNotes,
		LinkedInURL:      req.LinkedInURL,
		TwitterHandle:    req.TwitterHandle,
		UserID:           req.UserID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}
