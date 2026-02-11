package framework

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service provides business logic for framework operations
type Service struct {
	repo   Repository
	logger zerolog.Logger
}

// NewService creates a new framework service
func NewService(repo Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With().Str("service", "framework").Logger(),
	}
}

// GetRepo returns the repository (for creating dependent services)
func (s *Service) GetRepo() Repository {
	return s.repo
}

// GetLogger returns the logger (for creating dependent services)
func (s *Service) GetLogger() zerolog.Logger {
	return s.logger
}

// =============================================================================
// Framework Operations
// =============================================================================

// GetByCode retrieves a framework by its code
func (s *Service) GetByCode(ctx context.Context, code string) (*Framework, error) {
	return s.repo.GetByCode(ctx, code)
}

// GetByID retrieves a framework by its ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Framework, error) {
	return s.repo.GetByID(ctx, id)
}

// GetActive retrieves all active frameworks
func (s *Service) GetActive(ctx context.Context) ([]*Framework, error) {
	return s.repo.GetActive(ctx)
}

// GetBaseFrameworks retrieves all base (required) frameworks
func (s *Service) GetBaseFrameworks(ctx context.Context) ([]*Framework, error) {
	return s.repo.GetBaseFrameworks(ctx)
}

// Update updates a framework
func (s *Service) Update(ctx context.Context, f *Framework) error {
	if err := f.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	return s.repo.Update(ctx, f)
}

// =============================================================================
// Execution Plan
// =============================================================================

// BuildExecutionPlan creates an execution plan respecting dependencies
// Returns frameworks grouped by layer for parallel execution
func (s *Service) BuildExecutionPlan(ctx context.Context) (*ExecutionPlan, error) {
	frameworks, err := s.repo.GetActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get frameworks: %w", err)
	}

	if len(frameworks) == 0 {
		return &ExecutionPlan{Layers: []ExecutionLayer{}}, nil
	}

	// Group by layer (layer already encodes execution order)
	layerMap := make(map[int][]*Framework)
	for _, f := range frameworks {
		layerMap[f.Layer] = append(layerMap[f.Layer], f)
	}

	// Sort layers
	var layers []int
	for l := range layerMap {
		layers = append(layers, l)
	}
	sort.Ints(layers)

	// Build plan
	plan := &ExecutionPlan{}
	for _, l := range layers {
		plan.Layers = append(plan.Layers, ExecutionLayer{
			LayerNumber: l,
			Frameworks:  layerMap[l],
		})
	}

	s.logger.Debug().
		Int("total_frameworks", len(frameworks)).
		Int("total_layers", len(plan.Layers)).
		Msg("Built execution plan")

	return plan, nil
}

// BuildExecutionPlanForFrameworks creates an execution plan for specific frameworks
// Automatically includes dependencies
func (s *Service) BuildExecutionPlanForFrameworks(ctx context.Context, codes []string) (*ExecutionPlan, error) {
	// Get all frameworks and dependencies
	allFrameworks, err := s.repo.GetActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get frameworks: %w", err)
	}

	allDeps, err := s.repo.GetAllDependencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}

	// Build lookup maps
	codeToFramework := make(map[string]*Framework)
	idToFramework := make(map[uuid.UUID]*Framework)
	for _, f := range allFrameworks {
		codeToFramework[f.Code] = f
		idToFramework[f.ID] = f
	}

	// Build dependency map (framework_id -> []depends_on_id)
	depMap := make(map[uuid.UUID][]uuid.UUID)
	for _, d := range allDeps {
		depMap[d.FrameworkID] = append(depMap[d.FrameworkID], d.DependsOnID)
	}

	// Collect required frameworks including dependencies (recursive)
	required := make(map[uuid.UUID]*Framework)
	var collectDeps func(id uuid.UUID)
	collectDeps = func(id uuid.UUID) {
		if _, exists := required[id]; exists {
			return
		}
		if f, ok := idToFramework[id]; ok {
			required[id] = f
			for _, depID := range depMap[id] {
				collectDeps(depID)
			}
		}
	}

	// Start with requested frameworks
	for _, code := range codes {
		if f, ok := codeToFramework[code]; ok {
			collectDeps(f.ID)
		}
	}

	// Convert to slice and group by layer
	layerMap := make(map[int][]*Framework)
	for _, f := range required {
		layerMap[f.Layer] = append(layerMap[f.Layer], f)
	}

	// Sort layers
	var layers []int
	for l := range layerMap {
		layers = append(layers, l)
	}
	sort.Ints(layers)

	// Build plan
	plan := &ExecutionPlan{}
	for _, l := range layers {
		plan.Layers = append(plan.Layers, ExecutionLayer{
			LayerNumber: l,
			Frameworks:  layerMap[l],
		})
	}

	return plan, nil
}

// =============================================================================
// Dependency Resolution
// =============================================================================

// GetDependencyContext retrieves completed results from dependency frameworks
func (s *Service) GetDependencyContext(ctx context.Context, frameworkID, companyID uuid.UUID, challengeID *uuid.UUID) (map[string]json.RawMessage, error) {
	deps, err := s.repo.GetDependencies(ctx, frameworkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}

	context := make(map[string]json.RawMessage)
	for _, dep := range deps {
		result, err := s.repo.GetCurrentResult(ctx, companyID, dep.DependsOnID, challengeID)
		if err != nil {
			s.logger.Warn().Err(err).
				Str("dep_id", dep.DependsOnID.String()).
				Msg("Failed to get dependency result")
			continue
		}
		if result != nil && result.IsCompleted() {
			// Get framework code for the dependency
			fw, err := s.repo.GetByID(ctx, dep.DependsOnID)
			if err == nil && fw != nil {
				context[fw.Code] = result.Result
			}
		}
	}

	return context, nil
}

// ResolveDependencies returns all transitive dependencies for a framework
func (s *Service) ResolveDependencies(ctx context.Context, frameworkID uuid.UUID) ([]*Framework, error) {
	allDeps, err := s.repo.GetAllDependencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all dependencies: %w", err)
	}

	// Build dependency map
	depMap := make(map[uuid.UUID][]uuid.UUID)
	for _, d := range allDeps {
		depMap[d.FrameworkID] = append(depMap[d.FrameworkID], d.DependsOnID)
	}

	// Collect all transitive dependencies
	visited := make(map[uuid.UUID]bool)
	var collect func(id uuid.UUID)
	collect = func(id uuid.UUID) {
		if visited[id] {
			return
		}
		visited[id] = true
		for _, depID := range depMap[id] {
			collect(depID)
		}
	}

	// Start from the given framework
	for _, depID := range depMap[frameworkID] {
		collect(depID)
	}

	// Fetch framework details
	var frameworks []*Framework
	for id := range visited {
		f, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get framework %s: %w", id, err)
		}
		if f != nil {
			frameworks = append(frameworks, f)
		}
	}

	// Sort by layer
	sort.Slice(frameworks, func(i, j int) bool {
		return frameworks[i].Layer < frameworks[j].Layer
	})

	return frameworks, nil
}

// CheckDependenciesMet verifies all dependencies for a framework are completed
func (s *Service) CheckDependenciesMet(ctx context.Context, frameworkID, companyID uuid.UUID, challengeID *uuid.UUID) (bool, []string, error) {
	deps, err := s.repo.GetDependencies(ctx, frameworkID)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get dependencies: %w", err)
	}

	var missing []string
	for _, dep := range deps {
		if dep.DependencyType != DependencyRequired {
			continue // Only check required dependencies
		}

		result, err := s.repo.GetCurrentResult(ctx, companyID, dep.DependsOnID, challengeID)
		if err != nil {
			return false, nil, fmt.Errorf("failed to check dependency result: %w", err)
		}

		if result == nil || !result.IsCompleted() {
			fw, _ := s.repo.GetByID(ctx, dep.DependsOnID)
			if fw != nil {
				missing = append(missing, fw.Code)
			} else {
				missing = append(missing, dep.DependsOnID.String())
			}
		}
	}

	return len(missing) == 0, missing, nil
}

// =============================================================================
// Company Framework Results
// =============================================================================

// CreatePendingResult creates a new pending result for a company+framework
func (s *Service) CreatePendingResult(ctx context.Context, companyID, frameworkID uuid.UUID, challengeID *uuid.UUID) (*CompanyFrameworkResult, error) {
	// First, invalidate any existing current results
	if err := s.repo.InvalidateResults(ctx, companyID, frameworkID); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to invalidate existing results")
	}

	result := &CompanyFrameworkResult{
		ID:          uuid.New(),
		CompanyID:   companyID,
		FrameworkID: frameworkID,
		ChallengeID: challengeID,
		Result:      json.RawMessage(`{}`), // Initialize with empty JSON object for JSONB column
		Status:      StatusPending,
		Version:     1, // Will be incremented if there are previous versions
		IsCurrent:   true,
	}

	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := s.repo.CreateResult(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to create result: %w", err)
	}

	s.logger.Info().
		Str("result_id", result.ID.String()).
		Str("company_id", companyID.String()).
		Str("framework_id", frameworkID.String()).
		Msg("Created pending framework result")

	return result, nil
}

// MarkProcessing marks a result as processing
func (s *Service) MarkProcessing(ctx context.Context, resultID uuid.UUID) error {
	return s.repo.UpdateResultStatus(ctx, resultID, StatusProcessing, nil)
}

// MarkCompleted marks a result as completed with data
func (s *Service) MarkCompleted(ctx context.Context, resultID uuid.UUID, result json.RawMessage) error {
	return s.repo.UpdateResultCompleted(ctx, resultID, result, time.Now())
}

// MarkFailed marks a result as failed with error message
func (s *Service) MarkFailed(ctx context.Context, resultID uuid.UUID, errorMsg string) error {
	return s.repo.UpdateResultStatus(ctx, resultID, StatusFailed, &errorMsg)
}

// UpdateResult updates the result JSON data (for user editing)
func (s *Service) UpdateResult(ctx context.Context, resultID uuid.UUID, result json.RawMessage) error {
	return s.repo.UpdateResultData(ctx, resultID, result)
}

// GetResultByID retrieves a result by ID
func (s *Service) GetResultByID(ctx context.Context, id uuid.UUID) (*CompanyFrameworkResult, error) {
	return s.repo.GetResultByID(ctx, id)
}

// GetCompanyResults retrieves all current results for a company
func (s *Service) GetCompanyResults(ctx context.Context, companyID uuid.UUID, challengeID *uuid.UUID) ([]*CompanyFrameworkResult, error) {
	return s.repo.GetCompanyResults(ctx, companyID, challengeID)
}

// GetCompanyResultsWithFramework retrieves results with framework details
func (s *Service) GetCompanyResultsWithFramework(ctx context.Context, companyID uuid.UUID, challengeID *uuid.UUID) ([]*CompanyFrameworkResultWithDetails, error) {
	results, err := s.repo.GetCompanyResults(ctx, companyID, challengeID)
	if err != nil {
		return nil, err
	}

	var detailed []*CompanyFrameworkResultWithDetails
	for _, r := range results {
		fw, err := s.repo.GetByID(ctx, r.FrameworkID)
		if err != nil {
			continue
		}
		detailed = append(detailed, &CompanyFrameworkResultWithDetails{
			Result:    r,
			Framework: fw,
		})
	}

	return detailed, nil
}

// =============================================================================
// Context Hash
// =============================================================================

// ComputeContextHash generates a hash for cache invalidation
func (s *Service) ComputeContextHash(companyData interface{}, dependencyContext map[string]json.RawMessage) string {
	combined := map[string]interface{}{
		"company":      companyData,
		"dependencies": dependencyContext,
	}
	b, _ := json.Marshal(combined)
	hash := sha256.Sum256(b)
	return fmt.Sprintf("%x", hash)
}

// =============================================================================
// Helper Types
// =============================================================================

// CompanyFrameworkResultWithDetails includes framework details
type CompanyFrameworkResultWithDetails struct {
	Result    *CompanyFrameworkResult `json:"result"`
	Framework *Framework              `json:"framework"`
}
