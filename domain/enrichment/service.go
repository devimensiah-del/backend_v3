package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"backend_v3/config"
	"backend_v3/domain/submission"
	"backend_v3/llm"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

// Service handles business logic for enrichment
// AI-Only Pipeline: Perplexity (pre-search) + Gemini (synthesis)
// External adapters (DNS, scraper, macrodata APIs) have been REMOVED
// Perplexity handles all data gathering including company identification, macro data, and technical context
type Service struct {
	repo           Repository
	submissionRepo submission.Repository
	llmClient      *llm.Client
	queueClient    *asynq.Client          // For job orchestration
	enrichmentCfg  config.FrameworkConfig // Gemini config for synthesis
	preSearchCfg   config.FrameworkConfig // Perplexity config for pre-search
}

// NewService creates a new enrichment service
// AI-Only pipeline: Perplexity pre-search + Gemini synthesis
// No external adapters (DNS, scraper, macro APIs removed - Perplexity handles everything)
func NewService(repo Repository, submissionRepo submission.Repository, llmClient *llm.Client, queueClient *asynq.Client, enrichmentCfg config.FrameworkConfig, preSearchCfg config.FrameworkConfig) *Service {
	return &Service{
		repo:           repo,
		submissionRepo: submissionRepo,
		llmClient:      llmClient,
		queueClient:    queueClient,
		enrichmentCfg:  enrichmentCfg, // Gemini with Google Search for synthesis
		preSearchCfg:   preSearchCfg,  // Perplexity for company ID + macro data + technical context
	}
}

// GetByID retrieves enrichment by its own ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Enrichment, error) {
	return s.repo.GetByID(ctx, id)
}

// GetBySubmissionID retrieves enrichment linked to a submission
func (s *Service) GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*Enrichment, error) {
	return s.repo.GetBySubmissionID(ctx, submissionID)
}

// Create creates a new enrichment record
func (s *Service) Create(ctx context.Context, e *Enrichment) error {
	return s.repo.Create(ctx, e)
}

// UpdateEnrichmentData is called by the User Interface (API).
func (s *Service) UpdateEnrichmentData(ctx context.Context, id uuid.UUID, data map[string]interface{}) error {
	enrichment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Convert generic map to our special JSONMap type
	enrichment.EnrichedData = JSONMap(data)

	// Force the lock and update
	return s.repo.UpdateUser(ctx, enrichment)
}

// UpdateProgress updates the progress and current step of an enrichment
func (s *Service) UpdateProgress(ctx context.Context, id uuid.UUID, progress int, step string) error {
	enrichment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	enrichment.UpdateProgress(step, progress)

	// Update system (bypassing user locks if any, though progress is system-owned)
	return s.repo.UpdateSystem(ctx, enrichment)
}

// UpdateFields updates enrichment fields (admin edit)
// Status remains unchanged (stays "completed")
// Performs deep merge for nested objects to preserve existing fields
// IMPORTANT: Cannot edit after status is "approved"
func (s *Service) UpdateFields(ctx context.Context, id uuid.UUID, updateData map[string]interface{}) (*Enrichment, error) {
	// Get current enrichment
	enrichment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate: Cannot edit after approval
	if enrichment.Status == StatusApproved {
		return nil, fmt.Errorf("cannot edit enrichment after it has been approved")
	}

	// Deep merge update data into existing enriched_data
	enrichment.EnrichedData = deepMerge(enrichment.EnrichedData, updateData)

	// Update via repository
	if err := s.repo.UpdateUser(ctx, enrichment); err != nil {
		return nil, err
	}

	return enrichment, nil
}

// UpdateUnlock manually unlocks an enrichment (recovery endpoint)
// Used when an enrichment gets stuck due to locking issues
func (s *Service) UpdateUnlock(ctx context.Context, e *Enrichment) error {
	e.IsLocked = false
	e.UpdatedAt = time.Now()

	// Use ForceUpdateAndUnlock to bypass any existing locks
	if err := s.repo.ForceUpdateAndUnlock(ctx, e); err != nil {
		return fmt.Errorf("failed to unlock enrichment: %w", err)
	}

	return nil
}

// deepMerge recursively merges source into destination
// For objects: merges keys (doesn't replace entire object)
// For primitives/arrays: replaces value
func deepMerge(dest map[string]interface{}, src map[string]interface{}) map[string]interface{} {
	if dest == nil {
		dest = make(map[string]interface{})
	}

	for key, srcVal := range src {
		if destVal, exists := dest[key]; exists {
			// If both are maps, merge recursively
			if destMap, destIsMap := destVal.(map[string]interface{}); destIsMap {
				if srcMap, srcIsMap := srcVal.(map[string]interface{}); srcIsMap {
					dest[key] = deepMerge(destMap, srcMap)
					continue
				}
			}
		}
		// Otherwise, replace with source value
		dest[key] = srcVal
	}

	return dest
}

// Approve changes status from "completed" → "approved" and triggers analysis job creation
func (s *Service) Approve(ctx context.Context, id uuid.UUID) error {
	// Get enrichment
	enrichment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Validate status is "completed"
	if enrichment.Status != StatusCompleted {
		return fmt.Errorf("enrichment must be in 'completed' status to approve, current status: %s", enrichment.Status)
	}

	// Update status to approved and force unlock (admin action overrides user lock)
	enrichment.Status = StatusApproved
	if err := s.repo.ForceUpdateAndUnlock(ctx, enrichment); err != nil {
		return fmt.Errorf("failed to update enrichment status: %w", err)
	}

	log.Info().
		Str("enrichment_id", id.String()).
		Str("submission_id", enrichment.SubmissionID.String()).
		Msg("Enrichment approved, triggering analysis job")

	// Enqueue analysis job
	payload := map[string]string{
		"submission_id": enrichment.SubmissionID.String(),
		"enrichment_id": id.String(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal analysis job payload")
		return fmt.Errorf("failed to create analysis job: %w", err)
	}

	task := asynq.NewTask("analysis_job", payloadBytes)
	if _, err := s.queueClient.Enqueue(task); err != nil {
		log.Error().Err(err).Msg("Failed to enqueue analysis job")
		return fmt.Errorf("failed to enqueue analysis job: %w", err)
	}

	log.Info().
		Str("enrichment_id", id.String()).
		Str("submission_id", enrichment.SubmissionID.String()).
		Msg("Analysis job enqueued successfully")

	return nil
}

// ReopenForEditing reverts an approved enrichment back to completed status
// This allows admin to make further edits before re-approving
// Note: This does NOT delete or reset any analysis data - it simply unlocks enrichment for editing
func (s *Service) ReopenForEditing(ctx context.Context, id uuid.UUID) (*Enrichment, error) {
	// Get enrichment
	enrichment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Only approved enrichments can be reopened
	if enrichment.Status != StatusApproved {
		return nil, fmt.Errorf("only approved enrichments can be reopened for editing, current status: %s", enrichment.Status)
	}

	// Revert to completed status
	enrichment.Status = StatusCompleted

	// Update via repository (force update to bypass any locks)
	if err := s.repo.ForceUpdateAndUnlock(ctx, enrichment); err != nil {
		return nil, fmt.Errorf("failed to reopen enrichment: %w", err)
	}

	log.Info().
		Str("enrichment_id", id.String()).
		Str("submission_id", enrichment.SubmissionID.String()).
		Msg("Enrichment reopened for editing (approved → completed)")

	return enrichment, nil
}

// MarkAsFailed updates enrichment with error message
// Called by worker ErrorHandler after Asynq exhausts max retries
// Status remains "pending" with error_message populated
func (s *Service) MarkAsFailed(ctx context.Context, submissionID uuid.UUID, errorMsg string) error {
	enrichment, err := s.repo.GetBySubmissionID(ctx, submissionID)
	if err != nil {
		return fmt.Errorf("failed to get enrichment: %w", err)
	}

	// Fail() already sets ErrorMessage, but we set it explicitly for clarity
	enrichment.Fail(fmt.Errorf(errorMsg))

	if err := s.repo.UpdateSystem(ctx, enrichment); err != nil {
		return fmt.Errorf("failed to update enrichment error message: %w", err)
	}

	return nil
}
