package analysis

import (
	"context"

	"backend_v3/config"
	"backend_v3/llm"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// LLMClient defines the contract for the AI service.
type LLMClient interface {
	GenerateStructured(ctx context.Context, model string, prompt string, data interface{}, targetSchema interface{}) error
	GenerateStructuredWithOptions(ctx context.Context, opts llm.GenerationOptions, prompt string, data interface{}, targetSchema interface{}) error
}

// Service handles all business analysis operations
type Service struct {
	repo   Repository
	llm    LLMClient
	logger zerolog.Logger

	// Deprecated fields (kept for backward compatibility)
	analystModel   string
	synthesisModel string

	// New: Framework-specific configurations for heterogeneous model routing
	frameworks map[string]config.FrameworkConfig
}

// NewService creates a new analysis service instance
func NewService(
	repo Repository,
	llm LLMClient,
	logger zerolog.Logger,
	analystModel string, // DEPRECATED: use frameworks map
	synthesisModel string, // DEPRECATED: use frameworks map
) *Service {
	return &Service{
		repo:           repo,
		llm:            llm,
		logger:         logger.With().Str("service", "analysis").Logger(),
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

	// Save the new version
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
		swot.Strengths = interfaceSliceToStringSlice(strengths)
	}
	if weaknesses, ok := edits["weaknesses"].([]interface{}); ok {
		swot.Weaknesses = interfaceSliceToStringSlice(weaknesses)
	}
	if opportunities, ok := edits["opportunities"].([]interface{}); ok {
		swot.Opportunities = interfaceSliceToStringSlice(opportunities)
	}
	if threats, ok := edits["threats"].([]interface{}); ok {
		swot.Threats = interfaceSliceToStringSlice(threats)
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

// generateAnalysisID generates a new UUID for analysis
func generateAnalysisID() string {
	return uuid.New().String()
}
