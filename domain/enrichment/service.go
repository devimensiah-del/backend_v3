package enrichment

import (
	"context"
	"fmt"
	"time"

	"backend_v3/adapter/macrodata"
	"backend_v3/config"
	"backend_v3/domain/submission"
	"backend_v3/llm"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

// Service handles business logic for enrichment
// AI-Only Pipeline: Perplexity (pre-search) + Gemini (synthesis)
// MacroDataProvider re-integrated to fetch authoritative SELIC/IPCA/USD data from BCB/IBGE APIs
// MacroService provides DB-backed cached data from scheduled cron jobs
type Service struct {
	repo           Repository
	submissionRepo submission.Repository
	llmClient      *llm.Client
	queueClient    *asynq.Client                // For job orchestration
	enrichmentCfg  config.FrameworkConfig       // Gemini config for synthesis
	preSearchCfg   config.FrameworkConfig       // Perplexity config for pre-search
	macroProvider  *macrodata.MacroDataProvider // BCB/IBGE APIs for real-time economic data (fallback)
	macroService   MacroServiceInterface        // DB-backed macro data (primary, optional)
	companyService CompanyServiceInterface      // Company data persistence (optional)
}

// NewService creates a new enrichment service
// Two-phase pipeline: Perplexity pre-search + Gemini synthesis
// MacroDataProvider provides authoritative SELIC/IPCA/USD data from BCB/IBGE APIs
func NewService(repo Repository, submissionRepo submission.Repository, llmClient *llm.Client, queueClient *asynq.Client, enrichmentCfg config.FrameworkConfig, preSearchCfg config.FrameworkConfig, macroProvider *macrodata.MacroDataProvider) *Service {
	return &Service{
		repo:           repo,
		submissionRepo: submissionRepo,
		llmClient:      llmClient,
		queueClient:    queueClient,
		enrichmentCfg:  enrichmentCfg,  // Gemini with Google Search for synthesis
		preSearchCfg:   preSearchCfg,   // Perplexity for company ID + technical context
		macroProvider:  macroProvider,  // BCB/IBGE APIs for economic indicators (fallback)
		macroService:   nil,            // Optional: set via SetMacroService for DB-first
	}
}

// SetMacroService sets the macroeconomics service for DB-first macro data fetching
// This is optional - if not set, the service falls back to direct API calls via macroProvider
func (s *Service) SetMacroService(svc MacroServiceInterface) {
	s.macroService = svc
	log.Info().Msg("MacroService injected into enrichment service (DB-first enabled)")
}

// SetCompanyService sets the company service for data persistence
// This is optional - if not set, enrichment continues without updating company records
func (s *Service) SetCompanyService(svc CompanyServiceInterface) {
	s.companyService = svc
	log.Info().Msg("CompanyService injected into enrichment service (company updates enabled)")
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
// Note: If the enrichment is locked by a user, this update will silently fail
// (the user's edits take precedence over progress updates)
// NOTE: This method accepts SUBMISSION ID (not enrichment ID) because that's what the worker has
func (s *Service) UpdateProgress(ctx context.Context, submissionID uuid.UUID, progress int, step string) error {
	enrichment, err := s.repo.GetBySubmissionID(ctx, submissionID)
	if err != nil {
		return fmt.Errorf("failed to get enrichment: %w", err)
	}

	enrichment.UpdateProgress(step, progress)

	// UpdateSystem respects user locks - if locked, this update won't persist
	// This is intentional: user edits take precedence over worker progress
	return s.repo.UpdateSystem(ctx, enrichment)
}

// UpdateFields updates enrichment fields (admin edit)
// Status remains unchanged (stays "completed")
// Performs deep merge for nested objects to preserve existing fields
// Can edit completed enrichments (no approval workflow anymore)
func (s *Service) UpdateFields(ctx context.Context, id uuid.UUID, updateData map[string]interface{}) (*Enrichment, error) {
	// Get current enrichment
	enrichment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
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

// MarkAsFailed updates enrichment with error message and sets status to "failed"
// Called by worker ErrorHandler after Asynq exhausts max retries
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

// CopyEnrichment creates a new enrichment by copying data from an existing one
// Sets status to "completed" since data is already validated (used for re-analyze workflow)
// Returns the new enrichment ID
func (s *Service) CopyEnrichment(ctx context.Context, sourceEnrichmentID, newSubmissionID uuid.UUID) (*Enrichment, error) {
	// Get source enrichment
	source, err := s.repo.GetByID(ctx, sourceEnrichmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source enrichment: %w", err)
	}

	// Validate source is completed
	if source.Status != StatusCompleted {
		return nil, fmt.Errorf("source enrichment must be completed, current status: %s", source.Status)
	}

	// Create new enrichment with copied data
	now := time.Now()
	newEnrichment := &Enrichment{
		ID:            uuid.New(),
		SubmissionID:  newSubmissionID,
		CompanyID:     source.CompanyID, // Preserve company link
		Status:        StatusCompleted,  // Completed status (no more approved)
		Progress:      100,
		CurrentStep:   NullableString("Copied from previous enrichment"),
		IsLocked:      false,
		SourcesStatus: source.SourcesStatus,
		SourcesUsed:   source.SourcesUsed,
		EnrichedData:  source.EnrichedData, // Copy the enriched data
		StartedAt:     &now,
		CompletedAt:   &now,
		RetryCount:    0,
		MaxRetries:    3,
		AutoTriggerAnalysis: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.Create(ctx, newEnrichment); err != nil {
		return nil, fmt.Errorf("failed to create copied enrichment: %w", err)
	}

	log.Info().
		Str("source_enrichment_id", sourceEnrichmentID.String()).
		Str("new_enrichment_id", newEnrichment.ID.String()).
		Str("new_submission_id", newSubmissionID.String()).
		Msg("Enrichment data copied successfully")

	return newEnrichment, nil
}

// CreateWithAutoAnalyze creates a new enrichment with auto_trigger_analysis flag set
// Used for "enrich and analyze" workflow
func (s *Service) CreateWithAutoAnalyze(ctx context.Context, submissionID uuid.UUID) (*Enrichment, error) {
	e := NewEnrichment(submissionID)
	e.AutoTriggerAnalysis = true

	if err := s.repo.Create(ctx, e); err != nil {
		return nil, fmt.Errorf("failed to create enrichment with auto-analyze: %w", err)
	}

	log.Info().
		Str("enrichment_id", e.ID.String()).
		Str("submission_id", submissionID.String()).
		Bool("auto_trigger_analysis", true).
		Msg("Enrichment created with auto-analyze flag")

	return e, nil
}
