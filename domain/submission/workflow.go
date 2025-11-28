package submission

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	jobtypes "backend_v3/jobs/types"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

// SubmitForm is the "Main Event". It tells the story of what happens when a user clicks "Submit".
// Flow: 1. Prepare Data -> 2. Check CNPJ -> 3. Save to Safe -> 4. Create Company -> 5. Wake up the AI Robot
// opts is optional; pass nil for default behavior (non-admin)
func (s *Service) SubmitForm(ctx context.Context, req *SubmitRequest, opts *CreateOptions) (*Submission, error) {

	// Step 1: Prepare the data (Map the form input to our internal folder structure)
	submission := s.createSubmissionEntity(req)

	// Step 2: Check if CNPJ is locked by a verified company (unless admin)
	isAdmin := opts != nil && opts.IsAdmin
	if !isAdmin && s.companyService != nil && submission.CNPJ != nil && *submission.CNPJ != "" {
		exists, err := s.companyService.IsVerifiedCNPJExists(ctx, *submission.CNPJ)
		if err != nil {
			log.Warn().Err(err).Str("cnpj", *submission.CNPJ).Msg("Failed to check verified CNPJ - allowing submission")
			// Graceful degradation: allow submission if check fails
		} else if exists {
			log.Info().Str("cnpj", *submission.CNPJ).Msg("Submission blocked: CNPJ belongs to verified company")
			return nil, ErrVerifiedCNPJExists
		}
	}

	// Step 3: Validation (Check if they forgot their name or email)
	if err := submission.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Step 4: Save to Database (Put the file in the cabinet so we never lose it)
	if err := s.repo.Create(ctx, submission); err != nil {
		return nil, fmt.Errorf("failed to save submission: %w", err)
	}

	log.Info().Str("id", submission.ID.String()).Str("company", submission.CompanyName).Msg("submission created")

	// Step 5: Create company record (SYNCHRONOUS - must complete before enrichment)
	// This ensures the company exists when enrichment tries to link/update it
	if s.companyService != nil {
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
		}
		if err := s.companyService.CreateFromSubmission(ctx, companyInput); err != nil {
			// Log error but continue - enrichment can still proceed without company
			log.Error().
				Err(err).
				Str("submission_id", submission.ID.String()).
				Str("company_name", submission.CompanyName).
				Msg("Company creation failed - enrichment will proceed without company link")
		} else {
			log.Info().
				Str("submission_id", submission.ID.String()).
				Str("company_name", submission.CompanyName).
				Msg("Company record created from submission")
		}
	}

	// Step 6: Trigger the AI Agent (Send a signal to the background worker to start researching)
	if err := s.TriggerEnrichmentProcess(ctx, submission); err != nil {
		return nil, fmt.Errorf("failed to start enrichment process: %w", err)
	}

	log.Info().Str("id", submission.ID.String()).Msg("submission process started successfully")
	return submission, nil
}

// =================================================================================
// INTERNAL HELPER FUNCTIONS ( The Technical Details )
// =================================================================================

// createSubmissionEntity takes the raw web request and turns it into a Submission object
func (s *Service) createSubmissionEntity(req *SubmitRequest) *Submission {
	sub := NewSubmission(
		req.CompanyName,
		req.ContactName,
		req.ContactEmail,
		req.BusinessChallenge,
		req.UserID,
	)

	// Copy optional details
	sub.CNPJ = req.CNPJ
	sub.CompanyWebsite = req.CompanyWebsite
	sub.CompanyIndustry = req.CompanyIndustry
	sub.CompanySize = req.CompanySize
	sub.CompanyLocation = req.CompanyLocation
	sub.ContactPhone = req.ContactPhone
	sub.ContactPosition = req.ContactPosition
	sub.TargetMarket = req.TargetMarket
	sub.AnnualRevenueMin = req.AnnualRevenueMin
	sub.AnnualRevenueMax = req.AnnualRevenueMax
	sub.FundingStage = req.FundingStage
	sub.AdditionalNotes = req.AdditionalNotes
	sub.LinkedInURL = req.LinkedInURL
	sub.TwitterHandle = req.TwitterHandle

	return sub
}

// TriggerEnrichmentProcess handles the technical messaging to the Redis Queue
func (s *Service) TriggerEnrichmentProcess(ctx context.Context, sub *Submission) error {
	if s.queueClient == nil {
		return fmt.Errorf("queue client not configured")
	}

	// Pre-queue idempotency: reserve a row in the DB so concurrent requests cannot enqueue twice.
	reserved, err := s.repo.ReserveEnrichment(ctx, sub.ID)
	if err != nil {
		return fmt.Errorf("failed to reserve enrichment: %w", err)
	}
	if !reserved {
		// Enrichment already exists - check if we can still proceed
		statusRow, statusErr := s.repo.GetEnrichmentStatus(ctx, sub.ID)
		if statusErr != nil {
			if errors.Is(statusErr, sql.ErrNoRows) {
				// This shouldn't happen - INSERT ON CONFLICT returned no rows but no enrichment found
				// This could be a RLS issue. Log and continue with enqueue attempt.
				log.Warn().
					Str("submission_id", sub.ID.String()).
					Msg("ReserveEnrichment indicated conflict but no enrichment found - possible RLS issue, proceeding with enqueue")
			} else {
				return fmt.Errorf("failed to verify enrichment status: %w", statusErr)
			}
		} else {
			// Enrichment exists - check if it's in a state where we should skip
			if statusRow.Status == "processing" || statusRow.Status == "completed" || statusRow.Status == "approved" {
				log.Info().
					Str("submission_id", sub.ID.String()).
					Str("status", statusRow.Status).
					Msg("Enrichment already in progress or completed, skipping enqueue")
				return nil // Not an error - enrichment is already handled
			}
			// If pending or failed, we can re-enqueue
			log.Info().
				Str("submission_id", sub.ID.String()).
				Str("status", statusRow.Status).
				Msg("Found existing enrichment in retryable state, proceeding with enqueue")
		}
	}

	// Pack the data into a small parcel (JSON)
	// NOTE: Task type must match jobs.TypeEnrichment ("enrichment_job")
	payload, err := json.Marshal(map[string]interface{}{
		"submission_id": sub.ID.String(),
	})
	if err != nil {
		return err
	}

	// Create the task ticket with the correct type name
	task := asynq.NewTask(jobtypes.TypeEnrichment, payload)

	// Send the ticket to the queue (Retry up to 3 times if the robot is asleep)
	_, err = s.queueClient.Enqueue(task, asynq.MaxRetry(3))
	return err
}
