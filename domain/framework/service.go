package framework

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service handles business logic for framework management
type Service struct {
	repo   Repository
	logger zerolog.Logger
}

// NewService creates a new framework service
func NewService(repo Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// GetByCode retrieves a framework by its unique code
func (s *Service) GetByCode(ctx context.Context, code string) (*Framework, error) {
	return s.repo.GetByCode(ctx, code)
}

// List retrieves all frameworks
func (s *Service) List(ctx context.Context) ([]*Framework, error) {
	return s.repo.List(ctx, false)
}

// ListActive retrieves only active frameworks
func (s *Service) ListActive(ctx context.Context) ([]*Framework, error) {
	return s.repo.List(ctx, true)
}

// Create creates a new framework after validation
func (s *Service) Create(ctx context.Context, f *Framework) error {
	if err := f.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := s.repo.Create(ctx, f); err != nil {
		return fmt.Errorf("failed to create framework: %w", err)
	}

	s.logger.Info().
		Str("code", f.Code).
		Str("name", f.Name).
		Str("category", f.Category).
		Msg("Framework created")

	return nil
}

// Update updates an existing framework after validation
func (s *Service) Update(ctx context.Context, f *Framework) error {
	if err := f.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := s.repo.Update(ctx, f); err != nil {
		return fmt.Errorf("failed to update framework: %w", err)
	}

	s.logger.Info().
		Str("id", f.ID.String()).
		Str("code", f.Code).
		Msg("Framework updated")

	return nil
}

// Deactivate soft-deletes a framework by setting is_active to false
func (s *Service) Deactivate(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to deactivate framework: %w", err)
	}

	s.logger.Info().
		Str("id", id.String()).
		Msg("Framework deactivated")

	return nil
}

// GetExecutionPlan resolves dependencies and returns frameworks in correct execution order
// Uses topological sort to handle dependencies (DependsOn field)
// Validates that all requested codes exist and are active
func (s *Service) GetExecutionPlan(ctx context.Context, requestedCodes []string) ([]*Framework, error) {
	// Step 1: Fetch all active frameworks
	allFrameworks, err := s.repo.List(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("failed to list frameworks: %w", err)
	}

	// Step 2: Build a lookup map by code
	frameworkMap := make(map[string]*Framework)
	for _, fw := range allFrameworks {
		frameworkMap[fw.Code] = fw
	}

	// Step 3: Validate all requested codes exist and are active
	for _, code := range requestedCodes {
		if _, exists := frameworkMap[code]; !exists {
			return nil, fmt.Errorf("requested framework '%s' not found or not active", code)
		}
	}

	// Step 4: Collect all frameworks needed (requested + dependencies)
	needed := make(map[string]bool)
	var collectDependencies func(code string) error
	collectDependencies = func(code string) error {
		if needed[code] {
			return nil // Already processed
		}
		fw, exists := frameworkMap[code]
		if !exists {
			return fmt.Errorf("dependency '%s' not found or not active", code)
		}
		needed[code] = true
		for _, dep := range fw.DependsOn {
			if err := collectDependencies(dep); err != nil {
				return err
			}
		}
		return nil
	}

	for _, code := range requestedCodes {
		if err := collectDependencies(code); err != nil {
			return nil, err
		}
	}

	// Step 5: Topological sort (Kahn's algorithm)
	// Build in-degree map for needed frameworks
	inDegree := make(map[string]int)
	adjacency := make(map[string][]string)

	for code := range needed {
		inDegree[code] = 0
		adjacency[code] = []string{}
	}

	for code := range needed {
		fw := frameworkMap[code]
		for _, dep := range fw.DependsOn {
			if needed[dep] { // Only count dependencies that are in needed set
				adjacency[dep] = append(adjacency[dep], code)
				inDegree[code]++
			}
		}
	}

	// Find all nodes with in-degree 0 (no dependencies)
	queue := []string{}
	for code := range needed {
		if inDegree[code] == 0 {
			queue = append(queue, code)
		}
	}

	// Process queue
	var sorted []string
	for len(queue) > 0 {
		// Pop first element
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		// Reduce in-degree for all neighbors
		for _, neighbor := range adjacency[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check for cycles
	if len(sorted) != len(needed) {
		return nil, fmt.Errorf("circular dependency detected in framework dependencies")
	}

	// Step 6: Convert sorted codes to framework list
	result := make([]*Framework, 0, len(sorted))
	for _, code := range sorted {
		result = append(result, frameworkMap[code])
	}

	s.logger.Debug().
		Strs("requested", requestedCodes).
		Strs("execution_order", sorted).
		Msg("Execution plan generated")

	return result, nil
}
