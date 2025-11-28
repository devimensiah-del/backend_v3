package submission

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

// CompanyServiceInterface defines the contract for company service
// Using an interface avoids import cycles
type CompanyServiceInterface interface {
	CreateFromSubmission(ctx context.Context, input CompanyCreateInput) error
	IsVerifiedCNPJExists(ctx context.Context, cnpj string) (bool, error)
}

// CompanyCreateInput contains the data needed to create a company from a submission
type CompanyCreateInput struct {
	SubmissionID     uuid.UUID
	CompanyName      string
	CNPJ             *string
	Website          *string
	Industry         *string
	CompanySize      *string
	Location         *string
	TargetMarket     *string
	FundingStage     *string
	AnnualRevenueMin *float64
	AnnualRevenueMax *float64
	LinkedInURL      *string
	TwitterHandle    *string
}

// CreateOptions contains optional parameters for submission creation
type CreateOptions struct {
	IsAdmin bool // If true, bypasses verified CNPJ check
}

// ErrVerifiedCNPJExists indicates a submission was blocked due to a locked CNPJ
var ErrVerifiedCNPJExists = fmt.Errorf("this CNPJ is registered to a verified company and cannot be used for new submissions")

// Service holds the tools we need: a Database Repository and a Queue Client
type Service struct {
	repo                 Repository
	queueClient          queueClient
	revenuePerSubmission float64
	companyService       CompanyServiceInterface
}

type queueClient interface {
	Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
	Close() error
}

// NewService is the constructor (starts the engine)
func NewService(repo Repository, queueClient queueClient) *Service {
	return &Service{
		repo:                 repo,
		queueClient:          queueClient,
		revenuePerSubmission: loadRevenuePerSubmission(),
	}
}

// SetCompanyService injects the company service (optional, for graceful degradation)
func (s *Service) SetCompanyService(companySvc CompanyServiceInterface) {
	s.companyService = companySvc
}

// Create creates a new submission
// opts is optional; pass nil for default behavior (non-admin)
func (s *Service) Create(ctx context.Context, sub *Submission, opts *CreateOptions) (*Submission, error) {
	// Initialize ID and timestamps if not set
	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = time.Now()
	}
	if sub.UpdatedAt.IsZero() {
		sub.UpdatedAt = time.Now()
	}
	if sub.Status == "" {
		sub.Status = StatusReceived // Submission status always "received"
	}

	// Check if CNPJ is locked by a verified company (unless admin)
	isAdmin := opts != nil && opts.IsAdmin
	if !isAdmin && s.companyService != nil && sub.CNPJ != nil && *sub.CNPJ != "" {
		exists, err := s.companyService.IsVerifiedCNPJExists(ctx, *sub.CNPJ)
		if err != nil {
			log.Warn().Err(err).Str("cnpj", *sub.CNPJ).Msg("Failed to check verified CNPJ - allowing submission")
			// Graceful degradation: allow submission if check fails
		} else if exists {
			log.Info().Str("cnpj", *sub.CNPJ).Msg("Submission blocked: CNPJ belongs to verified company")
			return nil, ErrVerifiedCNPJExists
		}
	}

	// Validate
	if err := sub.Validate(); err != nil {
		return nil, err
	}

	// Create in repository
	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, err
	}

	log.Info().Str("id", sub.ID.String()).Str("company", sub.CompanyName).Msg("submission created")

	// Create company record (non-blocking, graceful degradation)
	if s.companyService != nil {
		go func() {
			companyInput := CompanyCreateInput{
				SubmissionID:     sub.ID,
				CompanyName:      sub.CompanyName,
				CNPJ:             sub.CNPJ,
				Website:          sub.CompanyWebsite,
				Industry:         sub.CompanyIndustry,
				CompanySize:      sub.CompanySize,
				Location:         sub.CompanyLocation,
				TargetMarket:     sub.TargetMarket,
				FundingStage:     sub.FundingStage,
				AnnualRevenueMin: sub.AnnualRevenueMin,
				AnnualRevenueMax: sub.AnnualRevenueMax,
				LinkedInURL:      sub.LinkedInURL,
				TwitterHandle:    sub.TwitterHandle,
			}
			if err := s.companyService.CreateFromSubmission(context.Background(), companyInput); err != nil {
				log.Warn().
					Err(err).
					Str("submission_id", sub.ID.String()).
					Str("company_name", sub.CompanyName).
					Msg("Failed to create company record - submission continues without company")
			} else {
				log.Info().
					Str("submission_id", sub.ID.String()).
					Str("company_name", sub.CompanyName).
					Msg("Company record created from submission")
			}
		}()
	}

	return sub, nil
}

// GetByID finds a single submission using its ID (accepts string or UUID)
func (s *Service) GetByID(ctx context.Context, id interface{}) (*Submission, error) {
	var uid uuid.UUID
	var err error

	switch v := id.(type) {
	case string:
		uid, err = uuid.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("invalid UUID format: %w", err)
		}
	case uuid.UUID:
		uid = v
	default:
		return nil, fmt.Errorf("id must be string or uuid.UUID, got %T", id)
	}

	return s.repo.GetByID(ctx, uid)
}

// GetByEmail finds all submissions made by a specific person
func (s *Service) GetByEmail(ctx context.Context, email string) ([]*Submission, error) {
	return s.repo.GetByEmail(ctx, email)
}

// ListAll gives the admin a list of everything, with pages and filters
func (s *Service) ListAll(ctx context.Context, opts *ListOptions) ([]*Submission, int, error) {
	return s.repo.List(ctx, opts)
}

// Delete moves a submission to the trash can (soft delete)
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}

// ==================== ANALYTICS ====================

// AnalyticsData holds the metrics for admin analytics dashboard
type AnalyticsData struct {
	TotalSubmissions     int
	ActiveSubmissions    int
	CompletedSubmissions int
	Revenue              float64
}

// GetAnalytics retrieves analytics data for the admin dashboard
func (s *Service) GetAnalytics(ctx context.Context) (*AnalyticsData, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		total, active, completed int
		wg                       sync.WaitGroup
		once                     sync.Once
		errChan                  = make(chan error, 1)
	)

	wg.Add(3)

	go func() {
		defer wg.Done()
		count, err := s.repo.GetTotalCount(ctx)
		if err != nil {
			once.Do(func() {
				errChan <- err
				cancel()
			})
			return
		}
		total = count
	}()

	go func() {
		defer wg.Done()
		count, err := s.repo.GetActiveCount(ctx)
		if err != nil {
			once.Do(func() {
				errChan <- err
				cancel()
			})
			return
		}
		active = count
	}()

	go func() {
		defer wg.Done()
		count, err := s.repo.GetCompletedCount(ctx)
		if err != nil {
			once.Do(func() {
				errChan <- err
				cancel()
			})
			return
		}
		completed = count
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case err := <-errChan:
		return nil, fmt.Errorf("failed to get analytics: %w", err)
	case <-done:
	}

	// Calculate revenue (configurable for pricing changes)
	revenue := float64(completed) * s.revenuePerSubmission

	return &AnalyticsData{
		TotalSubmissions:     total,
		ActiveSubmissions:    active,
		CompletedSubmissions: completed,
		Revenue:              revenue,
	}, nil
}

// =================================================================================
// DATA STRUCTURES (Request Inputs)
// =================================================================================

type SubmitRequest struct {
	CompanyName     string  `json:"companyName"`
	CNPJ            *string `json:"cnpj,omitempty"`
	CompanyWebsite  *string `json:"companyWebsite,omitempty"`
	CompanyIndustry *string `json:"companyIndustry,omitempty"`
	CompanySize     *string `json:"companySize,omitempty"`
	CompanyLocation *string `json:"companyLocation,omitempty"`

	ContactName     string  `json:"contactName"`
	ContactEmail    string  `json:"contactEmail"`
	ContactPhone    *string `json:"contactPhone,omitempty"`
	ContactPosition *string `json:"contactPosition,omitempty"`

	TargetMarket     *string  `json:"targetMarket,omitempty"`
	AnnualRevenueMin *float64 `json:"annualRevenueMin,omitempty"`
	AnnualRevenueMax *float64 `json:"annualRevenueMax,omitempty"`
	FundingStage     *string  `json:"fundingStage,omitempty"`

	BusinessChallenge string  `json:"businessChallenge"`
	AdditionalNotes   *string `json:"additionalNotes,omitempty"`
	LinkedInURL       *string `json:"linkedinUrl,omitempty"`
	TwitterHandle     *string `json:"twitterHandle,omitempty"`

	UserID *uuid.UUID `json:"-"`
}

type ListOptions struct {
	Limit   int
	Offset  int
	Status  *Status
	Email   *string
	UserID  *uuid.UUID
	OrderBy string
	Order   string
}

// loadRevenuePerSubmission reads pricing for analytics from environment with a safe default.
func loadRevenuePerSubmission() float64 {
	const fallback = 500.0
	val := os.Getenv("ANALYTICS_REVENUE_PER_SUBMISSION")
	if val == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

// CreateFromCompanyInput contains the data needed to create a submission from company data
// Used by admin re-enrich/re-analyze workflows
type CreateFromCompanyInput struct {
	CompanyID         uuid.UUID
	CompanyName       string
	CNPJ              *string
	Website           *string
	Industry          *string
	CompanySize       *string
	Location          *string
	TargetMarket      *string
	FundingStage      *string
	AnnualRevenueMin  *float64
	AnnualRevenueMax  *float64
	LinkedInURL       *string
	TwitterHandle     *string
	BusinessChallenge string // From last submission
	ContactName       string // From requesting user
	ContactEmail      string // From requesting user
	UserID            *uuid.UUID
}

// CreateFromCompany creates a submission populated from company data
// Does NOT trigger enrichment automatically (caller controls workflow)
// Does NOT create company record (company already exists)
func (s *Service) CreateFromCompany(ctx context.Context, input CreateFromCompanyInput) (*Submission, error) {
	now := time.Now()

	sub := &Submission{
		ID:                uuid.New(),
		CompanyName:       input.CompanyName,
		CNPJ:              input.CNPJ,
		CompanyWebsite:    input.Website,
		CompanyIndustry:   input.Industry,
		CompanySize:       input.CompanySize,
		CompanyLocation:   input.Location,
		ContactName:       input.ContactName,
		ContactEmail:      input.ContactEmail,
		TargetMarket:      input.TargetMarket,
		AnnualRevenueMin:  input.AnnualRevenueMin,
		AnnualRevenueMax:  input.AnnualRevenueMax,
		FundingStage:      input.FundingStage,
		BusinessChallenge: input.BusinessChallenge,
		LinkedInURL:       input.LinkedInURL,
		TwitterHandle:     input.TwitterHandle,
		UserID:            input.UserID,
		Status:            StatusReceived,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Validate
	if err := sub.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create in repository (bypass company creation since company already exists)
	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to save submission: %w", err)
	}

	log.Info().
		Str("id", sub.ID.String()).
		Str("company", sub.CompanyName).
		Str("company_id", input.CompanyID.String()).
		Msg("Submission created from company data (re-enrich/re-analyze workflow)")

	return sub, nil
}
