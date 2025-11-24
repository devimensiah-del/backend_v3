package submission

import (
	"context"
	"encoding/json"
	"fmt"

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
	// MOVED: Triggering is now handled by the API handler to ensure Enrichment record is created first
	// This avoids circular dependencies and ensures the UI has a record to poll immediately.

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

// triggerEnrichmentProcess handles the technical messaging to the Redis Queue
func (s *Service) triggerEnrichmentProcess(sub *Submission) error {
	// Pack the data into a small parcel (JSON)
	// NOTE: Task type must match jobs.TypeEnrichment ("enrichment_job")
	payload, err := json.Marshal(map[string]interface{}{
		"submission_id": sub.ID.String(),
	})
	if err != nil {
		return err
	}

	// Create the task ticket with the correct type name
	task := asynq.NewTask("enrichment_job", payload)

	// Send the ticket to the queue (Retry up to 3 times if the robot is asleep)
	_, err = s.queueClient.Enqueue(task, asynq.MaxRetry(3))
	return err
}
