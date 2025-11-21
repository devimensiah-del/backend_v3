package submission

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

// Service holds the tools we need: a Database Repository and a Queue Client
type Service struct {
	repo        Repository
	queueClient *asynq.Client
}

// NewService is the constructor (starts the engine)
func NewService(repo Repository, queueClient *asynq.Client) *Service {
	return &Service{
		repo:        repo,
		queueClient: queueClient,
	}
}

// Create creates a new submission
func (s *Service) Create(ctx context.Context, sub *Submission) (*Submission, error) {
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
		sub.Status = StatusPending
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

// UpdateStatus changes the label on the file (e.g., "Pending" -> "Enriched")
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	submission, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	submission.SetStatus(status)

	if err := s.repo.Update(ctx, submission); err != nil {
		return err
	}

	log.Info().Str("id", id.String()).Str("status", string(status)).Msg("status updated")
	return nil
}

// Delete moves a submission to the trash can (soft delete)
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}

// =================================================================================
// DATA STRUCTURES (Request Inputs)
// =================================================================================

type SubmitRequest struct {
	CompanyName     string  `json:"companyName"`
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
