package analysis

import (
	"context"
	"encoding/json"
	"fmt"

	"backend_v3/config"
	"backend_v3/llm"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

// LLMClient defines the contract for the AI service.
type LLMClient interface {
	GenerateStructured(ctx context.Context, model string, prompt string, data interface{}, targetSchema interface{}) error
	GenerateStructuredWithOptions(ctx context.Context, opts llm.GenerationOptions, prompt string, data interface{}, targetSchema interface{}) error
}

// SubmissionRepository defines the interface for accessing submission data
// We only need GetByID for fetching submission context
type SubmissionRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*SubmissionData, error)
}

// SubmissionData represents the essential submission fields needed for analysis
type SubmissionData struct {
	CompanyName       string
	CompanyWebsite    *string
	CompanyIndustry   *string
	CompanySize       *string
	CompanyLocation   *string
	BusinessChallenge string
	TargetMarket      *string
	AnnualRevenueMin  *float64
	AnnualRevenueMax  *float64
	FundingStage      *string
}

// Service handles all business analysis operations
type Service struct {
	repo           Repository
	submissionRepo SubmissionRepository
	llm            LLMClient
	logger         zerolog.Logger
	queueClient    *asynq.Client // For job orchestration

	// Deprecated fields (kept for backward compatibility)
	analystModel   string
	synthesisModel string

	// New: Framework-specific configurations for heterogeneous model routing
	frameworks map[string]config.FrameworkConfig
}

// NewService creates a new analysis service instance
func NewService(
	repo Repository,
	submissionRepo SubmissionRepository,
	llm LLMClient,
	logger zerolog.Logger,
	queueClient *asynq.Client,
	analystModel string, // DEPRECATED: use frameworks map
	synthesisModel string, // DEPRECATED: use frameworks map
) *Service {
	return &Service{
		repo:           repo,
		submissionRepo: submissionRepo,
		llm:            llm,
		logger:         logger.With().Str("service", "analysis").Logger(),
		queueClient:    queueClient,
		analystModel:   analystModel,
		synthesisModel: synthesisModel,
		frameworks:     make(map[string]config.FrameworkConfig), // Will be populated by main.go
	}
}

// SetFrameworks updates the framework configurations (called by main.go)
func (s *Service) SetFrameworks(frameworks map[string]config.FrameworkConfig) {
	s.frameworks = frameworks
}

// GetByID retrieves an analysis by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Analysis, error) {
	return s.repo.GetByID(ctx, id)
}

// GetBySubmissionID retrieves an analysis by submission ID
func (s *Service) GetBySubmissionID(ctx context.Context, submissionID string) (*Analysis, error) {
	return s.repo.GetBySubmissionID(ctx, submissionID)
}

// ListAll retrieves all analyses with pagination
func (s *Service) ListAll(ctx context.Context, limit, offset int) ([]*Analysis, error) {
	return s.repo.List(ctx, limit, offset)
}

// Delete removes an analysis record
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// CreateVersion creates a new version of an existing analysis with optional edits
// This is typically called after admin review when changes need to be made
func (s *Service) CreateVersion(ctx context.Context, analysisID string, edits map[string]interface{}) (*Analysis, error) {
	// Get the current analysis
	currentAnalysis, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return nil, err
	}

	// Create a new version based on the current one
	newVersion := currentAnalysis.CreateNewVersion()

	// Apply edits if provided
	if edits != nil {
		s.applyEditsToAnalysis(newVersion, edits)
	}

	// Generate new ID for the new version
	newVersion.ID = generateAnalysisID()

	// Clear previous latest and save the new version as latest
	if err := s.repo.ClearLatest(ctx, currentAnalysis.SubmissionID); err != nil {
		return nil, fmt.Errorf("failed to clear latest flag: %w", err)
	}
	err = s.repo.Create(ctx, newVersion)
	if err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("original_id", analysisID).
		Str("new_id", newVersion.ID).
		Int("version", newVersion.Version).
		Msg("Created new analysis version")

	return newVersion, nil
}

// GetLatestVersion retrieves the latest version of an analysis for a submission
func (s *Service) GetLatestVersion(ctx context.Context, submissionID string) (*Analysis, error) {
	return s.repo.GetLatestVersionBySubmissionID(ctx, submissionID)
}

// GetAllVersions retrieves all versions of analyses for a submission
func (s *Service) GetAllVersions(ctx context.Context, submissionID string) ([]*Analysis, error) {
	return s.repo.GetAllVersionsBySubmissionID(ctx, submissionID)
}

// applyEditsToAnalysis applies edits from a map to the analysis structure
func (s *Service) applyEditsToAnalysis(analysis *Analysis, edits map[string]interface{}) {
	// This is a helper method to apply edits to specific frameworks
	// You can expand this based on your needs

	if pestelEdits, ok := edits["pestel"].(map[string]interface{}); ok {
		s.applyPESTELEdits(&analysis.PESTEL, pestelEdits)
	}

	if porterEdits, ok := edits["porter"].(map[string]interface{}); ok {
		s.applyPorterEdits(&analysis.Porter, porterEdits)
	}

	if swotEdits, ok := edits["swot"].(map[string]interface{}); ok {
		s.applySWOTEdits(&analysis.SWOT, swotEdits)
	}

	// Add more framework edits as needed...
}

// Helper methods to apply edits to specific frameworks
func (s *Service) applyPESTELEdits(pestel *PESTELAnalysis, edits map[string]interface{}) {
	if political, ok := edits["political"].([]interface{}); ok {
		pestel.Political = interfaceSliceToStringSlice(political)
	}
	if economic, ok := edits["economic"].([]interface{}); ok {
		pestel.Economic = interfaceSliceToStringSlice(economic)
	}
	if social, ok := edits["social"].([]interface{}); ok {
		pestel.Social = interfaceSliceToStringSlice(social)
	}
	if technological, ok := edits["technological"].([]interface{}); ok {
		pestel.Technological = interfaceSliceToStringSlice(technological)
	}
	if environmental, ok := edits["environmental"].([]interface{}); ok {
		pestel.Environmental = interfaceSliceToStringSlice(environmental)
	}
	if legal, ok := edits["legal"].([]interface{}); ok {
		pestel.Legal = interfaceSliceToStringSlice(legal)
	}
	if summary, ok := edits["summary"].(string); ok {
		pestel.Summary = summary
	}
}

func (s *Service) applyPorterEdits(porter *PorterAnalysis, edits map[string]interface{}) {
	if competitiveRivalry, ok := edits["competitive_rivalry"].(string); ok {
		porter.CompetitiveRivalry = competitiveRivalry
	}
	if supplierPower, ok := edits["supplier_power"].(string); ok {
		porter.SupplierPower = supplierPower
	}
	if buyerPower, ok := edits["buyer_power"].(string); ok {
		porter.BuyerPower = buyerPower
	}
	if threatNewEntrants, ok := edits["threat_new_entrants"].(string); ok {
		porter.ThreatNewEntrants = threatNewEntrants
	}
	if threatSubstitutes, ok := edits["threat_substitutes"].(string); ok {
		porter.ThreatSubstitutes = threatSubstitutes
	}
	if overallAttractiveness, ok := edits["overall_attractiveness"].(string); ok {
		porter.OverallAttractiveness = overallAttractiveness
	}
	if summary, ok := edits["summary"].(string); ok {
		porter.Summary = summary
	}
}

func (s *Service) applySWOTEdits(swot *SWOTAnalysis, edits map[string]interface{}) {
	if strengths, ok := edits["strengths"].([]interface{}); ok {
		swot.Strengths = interfaceSliceToSWOTItemSlice(strengths)
	}
	if weaknesses, ok := edits["weaknesses"].([]interface{}); ok {
		swot.Weaknesses = interfaceSliceToSWOTItemSlice(weaknesses)
	}
	if opportunities, ok := edits["opportunities"].([]interface{}); ok {
		swot.Opportunities = interfaceSliceToSWOTItemSlice(opportunities)
	}
	if threats, ok := edits["threats"].([]interface{}); ok {
		swot.Threats = interfaceSliceToSWOTItemSlice(threats)
	}
	if summary, ok := edits["summary"].(string); ok {
		swot.Summary = summary
	}
}

// Helper function to convert []interface{} to []string
func interfaceSliceToStringSlice(slice []interface{}) []string {
	result := make([]string, len(slice))
	for i, v := range slice {
		if str, ok := v.(string); ok {
			result[i] = str
		}
	}
	return result
}

// Helper function to convert []interface{} to []SWOTItem
// Handles both old format ([]string) and new format ([]SWOTItem with confidence/source)
func interfaceSliceToSWOTItemSlice(slice []interface{}) []SWOTItem {
	result := make([]SWOTItem, 0, len(slice))
	for _, v := range slice {
		// Check if it's the new format (map with content, confidence, source)
		if itemMap, ok := v.(map[string]interface{}); ok {
			item := SWOTItem{}
			if content, ok := itemMap["content"].(string); ok {
				item.Content = content
			}
			if confidence, ok := itemMap["confidence"].(string); ok {
				item.Confidence = confidence
			}
			if source, ok := itemMap["source"].(string); ok {
				item.Source = source
			}
			result = append(result, item)
		} else if str, ok := v.(string); ok {
			// Backward compatibility: old format was just strings
			result = append(result, SWOTItem{
				Content:    str,
				Confidence: "Média",              // Default confidence for legacy data
				Source:     "análise de mercado", // Default source
			})
		}
	}
	return result
}

// generateAnalysisID generates a new UUID for analysis
func generateAnalysisID() string {
	return uuid.New().String()
}

// UpdateFields updates analysis framework fields (admin edit)
// Status remains unchanged
func (s *Service) UpdateFields(ctx context.Context, analysisID string, updateData map[string]interface{}) (*Analysis, error) {
	// Get current analysis
	currentAnalysis, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return nil, err
	}

	// Apply edits to analysis
	s.applyEditsToAnalysis(currentAnalysis, updateData)

	// Update via repository
	if err := s.repo.Update(ctx, currentAnalysis); err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Msg("Updated analysis fields")

	return currentAnalysis, nil
}

// Approve changes status from "completed" → "approved" and triggers PDF generation
func (s *Service) Approve(ctx context.Context, analysisID string) error {
	// Get analysis
	analysis, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return err
	}

	// Validate status is "completed"
	if analysis.Status != string(StatusCompleted) {
		return fmt.Errorf("analysis must be in 'completed' status to approve, current status: %s", analysis.Status)
	}

	// Update status to approved
	analysis.Status = string(StatusApproved)

	// Update via repository
	if err := s.repo.Update(ctx, analysis); err != nil {
		return fmt.Errorf("failed to update analysis status: %w", err)
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Msg("Analysis approved, triggering PDF generation")

	// Enqueue PDF generation job
	payload := map[string]string{
		"submission_id": analysis.SubmissionID,
		"analysis_id":   analysisID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to marshal PDF job payload")
		return fmt.Errorf("failed to create PDF job: %w", err)
	}

	task := asynq.NewTask("report", payloadBytes)
	if _, err := s.queueClient.Enqueue(task); err != nil {
		s.logger.Error().Err(err).Msg("Failed to enqueue PDF generation job")
		return fmt.Errorf("failed to enqueue PDF job: %w", err)
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Str("submission_id", analysis.SubmissionID).
		Msg("PDF generation job enqueued successfully")

	return nil
}

// Send changes status from "approved" → "sent", records notification details, and triggers user notification
func (s *Service) Send(ctx context.Context, analysisID string, userEmail string) error {
	// Get analysis
	analysis, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return err
	}

	// Validate status is "approved"
	if analysis.Status != string(StatusApproved) {
		return fmt.Errorf("analysis must be in 'approved' status to send, current status: %s", analysis.Status)
	}

	// Validate report exists with a PDF (best-effort guard)
	if s.reportService != nil {
		rep, repErr := s.reportService.GetBySubmissionID(ctx, analysis.SubmissionID)
		if repErr != nil {
			return fmt.Errorf("analysis approved but report not found: %w", repErr)
		}
		if rep.PDFURL == "" {
			return fmt.Errorf("analysis approved but PDF não está disponível ainda")
		}
	}

	// Update status to sent
	analysis.Status = string(StatusSent)
	sentTo := userEmail
	analysis.SentTo = &sentTo

	// Update via repository
	if err := s.repo.Update(ctx, analysis); err != nil {
		return fmt.Errorf("failed to update analysis status: %w", err)
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Str("sent_to", userEmail).
		Msg("Analysis marked as sent, triggering user notification")

	// Enqueue user notification job
	notificationPayload := map[string]string{
		"submission_id": analysis.SubmissionID,
		"analysis_id":   analysisID,
		"email":         userEmail,
		"type":          "analysis_ready",
	}

	notificationBytes, err := json.Marshal(notificationPayload)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to marshal notification payload")
		// Don't fail the whole operation if notification fails - analysis is already marked as sent
		return nil
	}

	notificationTask := asynq.NewTask("notification", notificationBytes)
	if _, err := s.queueClient.Enqueue(notificationTask); err != nil {
		s.logger.Error().Err(err).Msg("Failed to enqueue notification job")
		// Don't fail the whole operation - analysis is already marked as sent
	} else {
		s.logger.Info().
			Str("analysis_id", analysisID).
			Str("email", userEmail).
			Msg("User notification job enqueued successfully")
	}

	return nil
}

// MarkAsFailed updates analysis with error message
// Called by worker ErrorHandler after Asynq exhausts max retries
// Status remains "pending" with error_message populated
func (s *Service) MarkAsFailed(ctx context.Context, submissionID string, errorMsg string) error {
	analysis, err := s.repo.GetBySubmissionID(ctx, submissionID)
	if err != nil {
		return fmt.Errorf("failed to get analysis: %w", err)
	}

	// Set error message (status stays "pending")
	analysis.ErrorMessage = &errorMsg

	if err := s.repo.Update(ctx, analysis); err != nil {
		return fmt.Errorf("failed to update analysis error message: %w", err)
	}

	s.logger.Error().
		Str("analysis_id", analysis.ID).
		Str("submission_id", submissionID).
		Str("error", errorMsg).
		Msg("Analysis marked with error message")

	return nil
}
