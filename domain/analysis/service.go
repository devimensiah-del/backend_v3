package analysis

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"

	"backend_v3/config"
	"backend_v3/domain/submission"
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


// AnalysisCompanyServiceInterface defines what analysis needs from company service
// This interface allows analysis to read company data directly instead of from enrichment
type AnalysisCompanyServiceInterface interface {
	GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*AnalysisCompanyData, error)
	GetByID(ctx context.Context, companyID uuid.UUID) (*AnalysisCompanyData, error)
}

// AnalysisCompanyData represents company data for analysis prompts
type AnalysisCompanyData struct {
	Name              string                 `json:"name"`
	CNPJ              *string                `json:"cnpj,omitempty"`
	Website           *string                `json:"website,omitempty"`
	Industry          *string                `json:"industry,omitempty"`
	Sector            *string                `json:"sector,omitempty"`
	CompanySize       *string                `json:"company_size,omitempty"`
	Location          *string                `json:"location,omitempty"`
	TargetMarket      *string                `json:"target_market,omitempty"`
	FundingStage      *string                `json:"funding_stage,omitempty"`
	FoundationYear    *string                `json:"foundation_year,omitempty"`
	Headquarters      *string                `json:"headquarters,omitempty"`
	TargetAudience    *string                `json:"target_audience,omitempty"`
	ValueProposition  *string                `json:"value_proposition,omitempty"`
	EmployeesRange    *string                `json:"employees_range,omitempty"`
	RevenueEstimate   *string                `json:"revenue_estimate,omitempty"`
	BusinessModel     *string                `json:"business_model,omitempty"`
	Competitors       []string               `json:"competitors,omitempty"`
	MarketShareStatus *string                `json:"market_share_status,omitempty"`
	DigitalMaturity   *int                   `json:"digital_maturity,omitempty"`
	Strengths         []string               `json:"strengths,omitempty"`
	Weaknesses        []string               `json:"weaknesses,omitempty"`
	MacroContext      map[string]interface{} `json:"macro_context,omitempty"`
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
	repo             Repository
	submissionRepo   SubmissionRepository
	llm              LLMClient
	logger           zerolog.Logger
	queueClient      *asynq.Client                     // For job orchestration
	companyService   AnalysisCompanyServiceInterface   // Optional: used to fetch company data directly

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

// SetCompanyService wires a company service dependency (used for fetching company data directly)
// This allows analysis to read from company table instead of enrichment
func (s *Service) SetCompanyService(svc AnalysisCompanyServiceInterface) {
	s.companyService = svc
}

// GetByID retrieves an analysis by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Analysis, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByChallengeID retrieves an analysis by challenge ID
func (s *Service) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) (*Analysis, error) {
	return s.repo.GetByChallengeID(ctx, challengeID)
}

// ListAll retrieves all analyses with pagination
func (s *Service) ListAll(ctx context.Context, limit, offset int) ([]*Analysis, error) {
	return s.repo.List(ctx, limit, offset)
}

// Delete removes an analysis record
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// applyEditsToAnalysis applies edits from a map to the analysis structure.
// Uses JSON merge to apply partial updates to framework structs.
func (s *Service) applyEditsToAnalysis(analysis *Analysis, edits map[string]interface{}) {
	s.logger.Debug().
		Interface("edits_keys", getMapKeys(edits)).
		Msg("applyEditsToAnalysis received")

	// Iterate over all known framework codes
	for _, code := range knownFrameworkCodes {
		frameworkEdits, ok := edits[code].(map[string]interface{})
		if !ok {
			// Check if key exists but has wrong type
			if _, exists := edits[code]; exists {
				s.logger.Warn().
					Str("framework", code).
					Str("actual_type", fmt.Sprintf("%T", edits[code])).
					Msg("Framework edits type assertion failed")
			}
			continue
		}

		s.logger.Debug().Str("framework", code).Msg("Applying framework edits")

		// Create new framework instance
		framework := frameworkFactory(code)
		if framework == nil {
			s.logger.Warn().Str("framework", code).Msg("Unknown framework type")
			continue
		}

		// Load existing data (ignore errors if not found)
		_ = analysis.GetFramework(code, framework)

		// Apply edits using JSON merge
		if err := mergeEdits(framework, frameworkEdits); err != nil {
			s.logger.Error().Err(err).Str("framework", code).Msg("Failed to merge edits")
			continue
		}

		// Store back
		if err := analysis.SetFramework(code, framework); err != nil {
			s.logger.Error().Err(err).Str("framework", code).Msg("Failed to set framework")
		}
	}
}

// Helper to get map keys for debugging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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
// Status is set to "failed" with error_message populated
func (s *Service) MarkAsFailed(ctx context.Context, analysisID string, errorMsg string) error {
	analysis, err := s.repo.GetByID(ctx, analysisID)
	if err != nil {
		return fmt.Errorf("failed to get analysis: %w", err)
	}

	// Set status to failed and populate error message
	analysis.Fail(errorMsg)

	if err := s.repo.Update(ctx, analysis); err != nil {
		return fmt.Errorf("failed to update analysis error message: %w", err)
	}

	s.logger.Error().
		Str("analysis_id", analysis.ID).
		Str("error", errorMsg).
		Msg("Analysis marked as failed")

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

// --- Submission Repository Adapter ---

// SubmissionRepositoryAdapter adapts submission.Repository to analysis.SubmissionRepository
type SubmissionRepositoryAdapter struct {
	repo submission.Repository
}

// NewSubmissionRepositoryAdapter creates a new adapter
func NewSubmissionRepositoryAdapter(repo submission.Repository) SubmissionRepository {
	return &SubmissionRepositoryAdapter{repo: repo}
}

// GetByID fetches submission and converts it to SubmissionData needed by analysis
func (a *SubmissionRepositoryAdapter) GetByID(ctx context.Context, id uuid.UUID) (*SubmissionData, error) {
	sub, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Convert submission.Submission to analysis.SubmissionData
	// NOTE: BusinessChallenge field was removed from Submission model
	// Analysis now works with Challenge entity directly via challenge_id
	return &SubmissionData{
		CompanyName:       sub.CompanyName,
		CompanyWebsite:    sub.CompanyWebsite,
		CompanyIndustry:   sub.CompanyIndustry,
		CompanySize:       sub.CompanySize,
		CompanyLocation:   sub.CompanyLocation,
		BusinessChallenge: "", // Challenge data comes from Challenge entity, not Submission
		TargetMarket:      sub.TargetMarket,
		AnnualRevenueMin:  sub.AnnualRevenueMin,
		AnnualRevenueMax:  sub.AnnualRevenueMax,
		FundingStage:      sub.FundingStage,
	}, nil
}

// --- Edit Helpers (merged from edit_helpers.go) ---

// mergeEdits applies partial edits to a target struct using JSON merge.
// This replaces 12+ manual apply*Edits methods with a single generic function.
func mergeEdits(target interface{}, edits map[string]interface{}) error {
	current, err := json.Marshal(target)
	if err != nil {
		return err
	}

	var currentMap map[string]interface{}
	if err := json.Unmarshal(current, &currentMap); err != nil {
		return err
	}

	merged := deepMergeJSON(currentMap, edits)

	mergedBytes, err := json.Marshal(merged)
	if err != nil {
		return err
	}

	return json.Unmarshal(mergedBytes, target)
}

// deepMergeJSON recursively merges src into dst.
func deepMergeJSON(dst, src map[string]interface{}) map[string]interface{} {
	if dst == nil {
		dst = make(map[string]interface{})
	}

	for key, srcVal := range src {
		if dstVal, ok := dst[key]; ok {
			if dstMap, dstIsMap := dstVal.(map[string]interface{}); dstIsMap {
				if srcMap, srcIsMap := srcVal.(map[string]interface{}); srcIsMap {
					dst[key] = deepMergeJSON(dstMap, srcMap)
					continue
				}
			}
		}
		dst[key] = srcVal
	}

	return dst
}

// frameworkFactory returns a new instance of the framework struct for a given code.
func frameworkFactory(code string) interface{} {
	switch code {
	case "pestel":
		return new(PESTELAnalysis)
	case "porter":
		return new(PorterAnalysis)
	case "swot":
		return new(SWOTAnalysis)
	case "okrs":
		return new(OKRAnalysis)
	case "tam_sam_som":
		return new(TamSamSomAnalysis)
	case "benchmarking":
		return new(BenchmarkingAnalysis)
	case "blue_ocean":
		return new(BlueOceanAnalysis)
	case "growth_hacking":
		return new(GrowthHackingAnalysis)
	case "scenarios":
		return new(ScenarioAnalysis)
	case "bsc":
		return new(BalancedScorecardAnalysis)
	case "decision_matrix":
		return new(DecisionMatrixAnalysis)
	case "synthesis":
		return new(AnalysisSynthesis)
	default:
		return nil
	}
}

// knownFrameworkCodes lists all supported framework codes
var knownFrameworkCodes = []string{
	"pestel", "porter", "swot", "okrs", "tam_sam_som",
	"benchmarking", "blue_ocean", "growth_hacking", "scenarios",
	"bsc", "decision_matrix", "synthesis",
}
