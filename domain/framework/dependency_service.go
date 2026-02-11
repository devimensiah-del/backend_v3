package framework

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DependencyService manages framework dependencies and stale cascade
type DependencyService struct {
	repo   Repository
	logger zerolog.Logger
}

// NewDependencyService creates a new dependency service
func NewDependencyService(repo Repository, logger zerolog.Logger) *DependencyService {
	return &DependencyService{
		repo:   repo,
		logger: logger.With().Str("service", "dependency").Logger(),
	}
}

// =============================================================================
// Dependency Resolution
// =============================================================================

// GetDependencyCodes returns all framework codes that the given framework depends on
func (s *DependencyService) GetDependencyCodes(ctx context.Context, frameworkCode string) ([]string, error) {
	fw, err := s.repo.GetByCode(ctx, frameworkCode)
	if err != nil || fw == nil {
		return nil, fmt.Errorf("framework not found: %s", frameworkCode)
	}

	deps, err := s.repo.GetDependencies(ctx, fw.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}

	var codes []string
	for _, dep := range deps {
		depFw, err := s.repo.GetByID(ctx, dep.DependsOnID)
		if err != nil || depFw == nil {
			continue
		}
		codes = append(codes, depFw.Code)
	}

	return codes, nil
}

// GetDependentCodes returns all framework codes that depend on the given framework
func (s *DependencyService) GetDependentCodes(ctx context.Context, frameworkCode string) ([]string, error) {
	fw, err := s.repo.GetByCode(ctx, frameworkCode)
	if err != nil || fw == nil {
		return nil, fmt.Errorf("framework not found: %s", frameworkCode)
	}

	deps, err := s.repo.GetDependents(ctx, fw.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependents: %w", err)
	}

	var codes []string
	for _, dep := range deps {
		depFw, err := s.repo.GetByID(ctx, dep.FrameworkID)
		if err != nil || depFw == nil {
			continue
		}
		codes = append(codes, depFw.Code)
	}

	return codes, nil
}

// GetAllTransitiveDependencies returns all transitive dependencies for a framework
func (s *DependencyService) GetAllTransitiveDependencies(ctx context.Context, frameworkCode string) ([]string, error) {
	visited := make(map[string]bool)
	var result []string

	var collect func(code string) error
	collect = func(code string) error {
		if visited[code] {
			return nil
		}
		visited[code] = true

		deps, err := s.GetDependencyCodes(ctx, code)
		if err != nil {
			return err
		}

		for _, dep := range deps {
			if err := collect(dep); err != nil {
				return err
			}
			if dep != frameworkCode {
				result = append(result, dep)
			}
		}
		return nil
	}

	if err := collect(frameworkCode); err != nil {
		return nil, err
	}

	return result, nil
}

// CheckReadyToRun verifies all dependencies are completed
func (s *DependencyService) CheckReadyToRun(ctx context.Context, companyID uuid.UUID, frameworkCode string, challengeID *uuid.UUID) (*ReadinessStatus, error) {
	fw, err := s.repo.GetByCode(ctx, frameworkCode)
	if err != nil || fw == nil {
		return nil, fmt.Errorf("framework not found: %s", frameworkCode)
	}

	// Layer 0 frameworks (EMPRESA, NEGOCIO) are always ready (they display enriched data)
	if fw.Layer == 0 {
		return &ReadinessStatus{Ready: true}, nil
	}

	deps, err := s.repo.GetDependencies(ctx, fw.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}

	var missing []string
	for _, dep := range deps {
		if dep.DependencyType != DependencyRequired {
			continue
		}

		// Get the dependency framework to check its layer
		depFw, err := s.repo.GetByID(ctx, dep.DependsOnID)
		if err != nil || depFw == nil {
			continue
		}

		// Layer 0 frameworks (data-only: EMPRESA, NEGOCIO, MERCADO) are always ready
		// They display enriched data from company, not AI-generated framework results
		if depFw.Layer == 0 {
			continue
		}

		result, err := s.repo.GetCurrentResult(ctx, companyID, dep.DependsOnID, challengeID)
		if err != nil {
			continue
		}

		if result == nil || !result.IsCompleted() {
			missing = append(missing, depFw.Code)
		}
	}

	return &ReadinessStatus{
		Ready:               len(missing) == 0,
		MissingDependencies: missing,
	}, nil
}

// =============================================================================
// Stale Cascade
// =============================================================================

// MarkDownstreamAsStale marks all dependent frameworks as stale when an upstream framework is updated
func (s *DependencyService) MarkDownstreamAsStale(ctx context.Context, companyID uuid.UUID, changedFrameworkCode string) (int, error) {
	fw, err := s.repo.GetByCode(ctx, changedFrameworkCode)
	if err != nil || fw == nil {
		return 0, fmt.Errorf("framework not found: %s", changedFrameworkCode)
	}

	affectedCount := 0
	visited := make(map[uuid.UUID]bool)

	var markStale func(frameworkID uuid.UUID, reason string) error
	markStale = func(frameworkID uuid.UUID, reason string) error {
		if visited[frameworkID] {
			return nil
		}
		visited[frameworkID] = true

		// Get dependents (frameworks that depend on this one)
		deps, err := s.repo.GetDependents(ctx, frameworkID)
		if err != nil {
			return err
		}

		for _, dep := range deps {
			// Get the current result for this dependent framework
			result, err := s.repo.GetCurrentResult(ctx, companyID, dep.FrameworkID, nil)
			if err != nil {
				s.logger.Warn().Err(err).Str("framework_id", dep.FrameworkID.String()).Msg("Failed to get result")
				continue
			}

			if result != nil && result.IsCompleted() {
				// Mark as stale
				if err := s.repo.MarkAsStale(ctx, result.ID, reason); err != nil {
					s.logger.Warn().Err(err).Str("result_id", result.ID.String()).Msg("Failed to mark as stale")
					continue
				}
				affectedCount++

				// Get dependent framework name for cascading reason
				depFw, _ := s.repo.GetByID(ctx, dep.FrameworkID)
				if depFw != nil {
					s.logger.Info().
						Str("company_id", companyID.String()).
						Str("framework", depFw.Code).
						Str("reason", reason).
						Msg("Marked framework as stale")

					// Recursively mark its dependents
					if err := markStale(dep.FrameworkID, depFw.Code+" foi atualizado"); err != nil {
						return err
					}
				}
			}
		}

		return nil
	}

	reason := changedFrameworkCode + " foi atualizado"
	if err := markStale(fw.ID, reason); err != nil {
		return affectedCount, fmt.Errorf("failed to mark downstream as stale: %w", err)
	}

	s.logger.Info().
		Str("company_id", companyID.String()).
		Str("changed_framework", changedFrameworkCode).
		Int("affected_count", affectedCount).
		Msg("Marked downstream frameworks as stale")

	return affectedCount, nil
}

// GetStaleFrameworks returns all stale frameworks for a company
func (s *DependencyService) GetStaleFrameworks(ctx context.Context, companyID uuid.UUID) ([]*StaleFramework, error) {
	results, err := s.repo.GetStaleResults(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stale results: %w", err)
	}

	var stale []*StaleFramework
	for _, r := range results {
		fw, err := s.repo.GetByID(ctx, r.FrameworkID)
		if err != nil || fw == nil {
			continue
		}

		sf := &StaleFramework{
			FrameworkCode:       fw.Code,
			FrameworkName:       fw.Name,
			StaleReason:         "",
			ResultID:            r.ID,
			StaleAcknowledgedAt: r.StaleAcknowledgedAt,
		}

		if r.StaleReason != nil {
			sf.StaleReason = *r.StaleReason
		}
		if r.StaleDetectedAt != nil {
			sf.StaleDetectedAt = *r.StaleDetectedAt
		}

		stale = append(stale, sf)
	}

	return stale, nil
}

// AcknowledgeStale marks a stale result as acknowledged
func (s *DependencyService) AcknowledgeStale(ctx context.Context, resultID uuid.UUID, userID uuid.UUID) error {
	return s.repo.AcknowledgeStale(ctx, resultID, userID)
}

// ClearStaleStatus clears the stale status after re-generation
func (s *DependencyService) ClearStaleStatus(ctx context.Context, resultID uuid.UUID) error {
	return s.repo.ClearStaleStatus(ctx, resultID)
}

// =============================================================================
// Context Hash
// =============================================================================

// ComputeUpstreamHash computes a hash of all upstream dependency results
func (s *DependencyService) ComputeUpstreamHash(ctx context.Context, companyID uuid.UUID, frameworkCode string, challengeID *uuid.UUID) (string, error) {
	deps, err := s.GetAllTransitiveDependencies(ctx, frameworkCode)
	if err != nil {
		return "", fmt.Errorf("failed to get dependencies: %w", err)
	}

	var data []byte
	for _, depCode := range deps {
		fw, err := s.repo.GetByCode(ctx, depCode)
		if err != nil || fw == nil {
			continue
		}

		result, err := s.repo.GetCurrentResult(ctx, companyID, fw.ID, challengeID)
		if err != nil || result == nil {
			continue
		}

		if result.Result != nil {
			data = append(data, result.Result...)
		}
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:16]), nil
}

// CheckUpstreamChanged checks if upstream data has changed since the result was generated
func (s *DependencyService) CheckUpstreamChanged(ctx context.Context, resultID uuid.UUID) (bool, error) {
	result, err := s.repo.GetResultByID(ctx, resultID)
	if err != nil || result == nil {
		return false, fmt.Errorf("result not found")
	}

	// Get framework code
	fw, err := s.repo.GetByID(ctx, result.FrameworkID)
	if err != nil || fw == nil {
		return false, fmt.Errorf("framework not found")
	}

	// Compute current hash
	currentHash, err := s.ComputeUpstreamHash(ctx, result.CompanyID, fw.Code, result.ChallengeID)
	if err != nil {
		return false, err
	}

	// Compare with stored hash
	if result.UpstreamDataHash == nil {
		return true, nil // No hash stored, assume changed
	}

	return *result.UpstreamDataHash != currentHash, nil
}

// =============================================================================
// Analysis State
// =============================================================================

// GetAnalysisState returns the complete analysis state for a company
func (s *DependencyService) GetAnalysisState(ctx context.Context, companyID uuid.UUID, challengeID *uuid.UUID) (*AnalysisState, error) {
	// Get all active frameworks
	frameworks, err := s.repo.GetActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get frameworks: %w", err)
	}

	// Get all results for this company
	results, err := s.repo.GetCompanyResults(ctx, companyID, challengeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get results: %w", err)
	}

	// Build result map
	resultMap := make(map[uuid.UUID]*CompanyFrameworkResult)
	for _, r := range results {
		resultMap[r.FrameworkID] = r
	}

	// Build framework status map
	frameworkStatus := make(map[string]*FrameworkStatus)
	stepMap := make(map[int]*StepDefinition)

	for _, fw := range frameworks {
		if fw.StepNumber == nil {
			continue
		}

		// Check readiness
		readiness, _ := s.CheckReadyToRun(ctx, companyID, fw.Code, challengeID)

		status := &FrameworkStatus{
			Code:           fw.Code,
			Name:           fw.Name,
			Status:         StatusPending,
			IsStale:        false,
			Version:        0,
			HasResult:      false,
			CanExecute:     readiness != nil && readiness.Ready,
			SubtabOf:       fw.SubtabOf,
			ExecutionOrder: fw.ExecutionOrder,
		}

		if readiness != nil && len(readiness.MissingDependencies) > 0 {
			status.MissingDependencies = readiness.MissingDependencies
		}

		// Check if there's a result
		if result, ok := resultMap[fw.ID]; ok {
			status.Status = result.Status
			status.Version = result.Version
			status.HasResult = result.Result != nil && len(result.Result) > 0
			status.IsStale = result.IsStale

			if result.StaleReason != nil {
				status.StaleReason = result.StaleReason
			}
			if result.StaleAcknowledgedAt != nil {
				t := result.StaleAcknowledgedAt.Format("2006-01-02T15:04:05Z07:00")
				status.StaleAcknowledgedAt = &t
			}
		}

		frameworkStatus[fw.Code] = status

		// Add to step
		stepNum := *fw.StepNumber
		if _, ok := stepMap[stepNum]; !ok {
			stepName := "Step " + fmt.Sprintf("%d", stepNum)
			if fw.StepName != nil {
				stepName = *fw.StepName
			}
			stepMap[stepNum] = &StepDefinition{
				Number:     stepNum,
				Name:       stepName,
				Frameworks: []*FrameworkStatus{},
			}
		}
		stepMap[stepNum].Frameworks = append(stepMap[stepNum].Frameworks, status)
	}

	// Build ordered steps slice
	var steps []*StepDefinition
	for i := 1; i <= 5; i++ {
		if step, ok := stepMap[i]; ok {
			steps = append(steps, step)
		}
	}

	// Calculate progress and stale count
	totalFrameworks := len(frameworks)
	completedFrameworks := 0
	staleCount := 0

	for _, r := range results {
		if r.IsCompleted() {
			completedFrameworks++
		}
		if r.IsStale {
			staleCount++
		}
	}

	progress := 0
	if totalFrameworks > 0 {
		progress = (completedFrameworks * 100) / totalFrameworks
	}

	return &AnalysisState{
		CompanyID:       companyID,
		Steps:           steps,
		Frameworks:      frameworkStatus,
		OverallProgress: progress,
		StaleCount:      staleCount,
	}, nil
}

// =============================================================================
// Dependency Context for Prompts
// =============================================================================

// GetDependencyContext returns all completed dependency results for prompt injection
func (s *DependencyService) GetDependencyContext(ctx context.Context, companyID uuid.UUID, frameworkCode string, challengeID *uuid.UUID) (map[string]json.RawMessage, error) {
	deps, err := s.GetAllTransitiveDependencies(ctx, frameworkCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}

	context := make(map[string]json.RawMessage)
	for _, depCode := range deps {
		fw, err := s.repo.GetByCode(ctx, depCode)
		if err != nil || fw == nil {
			continue
		}

		result, err := s.repo.GetCurrentResult(ctx, companyID, fw.ID, challengeID)
		if err != nil || result == nil {
			continue
		}

		if result.IsCompleted() && result.Result != nil {
			context[depCode] = result.Result
		}
	}

	return context, nil
}
