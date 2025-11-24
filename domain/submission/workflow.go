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
// Flow: 1. Prepare Data -> 2. Save to Safe -> 3. Wake up the AI Robot
func (s *Service) SubmitForm(ctx context.Context, req *SubmitRequest) (*Submission, error) {

	// Step 1: Prepare the data (Map the form input to our internal folder structure)
	submission := s.createSubmissionEntity(req)

	// Step 2: Validation (Check if they forgot their name or email)
	if err := submission.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Step 3: Save to Database (Put the file in the cabinet so we never lose it)
	if err := s.repo.Create(ctx, submission); err != nil {
		return nil, fmt.Errorf("failed to save submission: %w", err)
	}

	// Step 4: Trigger the AI Agent (Send a signal to the background worker to start researching)
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
		statusRow, statusErr := s.repo.GetEnrichmentStatus(ctx, sub.ID)
		if statusErr == nil {
			return fmt.Errorf("enrichment already exists with status %s", statusRow.Status)
		}
		if !errors.Is(statusErr, sql.ErrNoRows) {
			return fmt.Errorf("failed to verify enrichment status: %w", statusErr)
		}
		return fmt.Errorf("enrichment already exists")
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
