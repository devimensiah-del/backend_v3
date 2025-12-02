package analysis

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

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

// MacroServiceInterface defines the contract for fetching macroeconomic data
// This interface is implemented by macroeconomics.Service (via adapter in main.go)
// Using an interface avoids import cycles between analysis and macroeconomics domains
type MacroServiceInterface interface {
	// GetLatestSnapshot retrieves the most recent values for ALL active indicators
	// Returns nil error if DB is empty (graceful degradation)
	GetLatestSnapshot(ctx context.Context) (*MacroSnapshot, error)
}

// MacroSnapshot represents latest economic indicators from DB
type MacroSnapshot struct {
	Indicators map[string]*MacroIndicator `json:"indicators"`
}

// MacroIndicator represents a single indicator with full metadata
type MacroIndicator struct {
	Code  string  `json:"code"`
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// HasData returns true if at least one indicator has data
func (s *MacroSnapshot) HasData() bool {
	return s != nil && len(s.Indicators) > 0
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

// ReportLookup defines the minimal dependency needed to verify PDF existence
type ReportLookup interface {
	GetBySubmissionID(ctx context.Context, submissionID string) (ReportSummary, error)
}

// ReportSummary captures only the fields analysis needs to validate sending
type ReportSummary interface {
	GetPDFURL() string
}

// Service handles all business analysis operations
type Service struct {
	repo           Repository
	submissionRepo SubmissionRepository
	llm            LLMClient
	logger         zerolog.Logger
	queueClient    *asynq.Client // For job orchestration
	reportLookup   ReportLookup  // Optional: used to verify PDF before Send
	macroService   MacroServiceInterface // Optional: used to fetch macro data directly from DB

	// Framework configurations (4-model approach: presearch, enrichment, primary, synthesis)
	frameworks map[string]config.FrameworkConfig
}

// NewService creates a new analysis service instance
func NewService(
	repo Repository,
	submissionRepo SubmissionRepository,
	llm LLMClient,
	logger zerolog.Logger,
	queueClient *asynq.Client,
) *Service {
	return &Service{
		repo:           repo,
		submissionRepo: submissionRepo,
		llm:            llm,
		logger:         logger.With().Str("service", "analysis").Logger(),
		queueClient:    queueClient,
		frameworks:     make(map[string]config.FrameworkConfig), // Populated via SetFrameworks()
	}
}

// SetFrameworks updates the framework configurations (called by main.go)
func (s *Service) SetFrameworks(frameworks map[string]config.FrameworkConfig) {
	s.frameworks = frameworks
}

// SetReportLookup wires a report lookup dependency (used for PDF existence checks)
func (s *Service) SetReportLookup(lookup ReportLookup) {
	s.reportLookup = lookup
}

// SetMacroService wires a macro service dependency (used for fetching macro data directly from DB)
// This is preferred over extracting macro data from enrichment output
func (s *Service) SetMacroService(svc MacroServiceInterface) {
	s.macroService = svc
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
// IMPORTANT: Admin can only edit when status is "completed" (AI finished processing)
func (s *Service) UpdateFields(ctx context.Context, analysisID string, updateData map[string]interface{}) (*Analysis, error) {
	// Get current analysis
	currentAnalysis, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return nil, err
	}

	// Only allow edits when AI has finished (status = "completed")
	// Block edits during: pending (AI processing)
	if currentAnalysis.Status == string(StatusPending) || currentAnalysis.Status == string(StatusProcessing) {
		return nil, fmt.Errorf("cannot edit analysis while AI is still processing")
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

// SetVisibility toggles user visibility for an analysis
// Admin can make a completed analysis visible to the user
func (s *Service) SetVisibility(ctx context.Context, analysisID string, visible bool) error {
	// Get analysis first to validate state
	analysis, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return err
	}

	// Only allow visibility toggle on completed analyses
	if analysis.Status != string(StatusCompleted) {
		return fmt.Errorf("analysis must be completed before toggling visibility, current status: %s", analysis.Status)
	}

	if err := s.repo.SetVisibility(ctx, analysisID, visible); err != nil {
		return err
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Bool("visible", visible).
		Msg("Analysis visibility updated")

	return nil
}

// SetBlurStatus toggles the blur overlay for premium frameworks
// This is independent of visibility - an analysis can be visible but blurred (paywall)
// Admin can unblur to give full access to premium content
func (s *Service) SetBlurStatus(ctx context.Context, analysisID string, blurred bool) error {
	// Get analysis first to validate it exists
	_, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return err
	}

	// No status restriction - blur can be toggled at any time
	// (it's a display control, not a workflow state)

	if err := s.repo.SetBlurStatus(ctx, analysisID, blurred); err != nil {
		return err
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Bool("blurred", blurred).
		Msg("Analysis blur status updated")

	return nil
}

// SetPublicStatus toggles whether the analysis is accessible without login
// When true, anyone with the access code can view the report
// When false, the access code requires authentication
func (s *Service) SetPublicStatus(ctx context.Context, analysisID string, public bool) error {
	// Get analysis first to validate it exists
	_, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return err
	}

	// No status restriction - public flag can be toggled at any time
	// (it's an access control, not a workflow state)

	if err := s.repo.SetPublicStatus(ctx, analysisID, public); err != nil {
		return err
	}

	s.logger.Info().
		Str("analysis_id", analysisID).
		Bool("public", public).
		Msg("Analysis public status updated")

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

// GetByAccessCode retrieves an analysis by its public access code
// Returns nil, nil if not found (for 404 handling)
func (s *Service) GetByAccessCode(ctx context.Context, code string) (*Analysis, error) {
	return s.repo.GetByAccessCode(ctx, code)
}

// GenerateAccessCode creates a unique 8-character access code for public sharing
// Handles collisions by regenerating up to 5 times
func (s *Service) GenerateAccessCode(ctx context.Context, analysisID string) (string, error) {
	// Verify analysis exists and is completed
	analysis, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return "", err
	}

	// Only allow access code generation for completed analyses
	if analysis.Status != string(StatusCompleted) {
		return "", fmt.Errorf("analysis must be completed before generating access code, current status: %s", analysis.Status)
	}

	const maxRetries = 5
	for i := 0; i < maxRetries; i++ {
		code := generateSecureCode(8)
		exists, err := s.repo.AccessCodeExists(ctx, code)
		if err != nil {
			return "", fmt.Errorf("failed to check code existence: %w", err)
		}
		if !exists {
			err = s.repo.SetAccessCode(ctx, analysisID, code)
			if err != nil {
				return "", fmt.Errorf("failed to set access code: %w", err)
			}

			s.logger.Info().
				Str("analysis_id", analysisID).
				Str("access_code", code).
				Msg("Generated access code for analysis")

			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique code after %d attempts", maxRetries)
}

// generateSecureCode generates a cryptographically secure alphanumeric code
// Uses only uppercase letters and digits (36 characters) for URL-safety
func generateSecureCode(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[n.Int64()]
	}

	return string(result)
}
