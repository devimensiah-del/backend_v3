package framework

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for framework data access
type Repository interface {
	// Framework CRUD
	Create(ctx context.Context, f *Framework) error
	GetByID(ctx context.Context, id uuid.UUID) (*Framework, error)
	GetByCode(ctx context.Context, code string) (*Framework, error)
	GetActive(ctx context.Context) ([]*Framework, error)
	GetBaseFrameworks(ctx context.Context) ([]*Framework, error)
	GetByLayer(ctx context.Context, layer int) ([]*Framework, error)
	Update(ctx context.Context, f *Framework) error

	// Dependencies
	GetDependencies(ctx context.Context, frameworkID uuid.UUID) ([]*FrameworkDependency, error)
	GetAllDependencies(ctx context.Context) ([]*FrameworkDependency, error)
	AddDependency(ctx context.Context, dep *FrameworkDependency) error
	RemoveDependency(ctx context.Context, frameworkID, dependsOnID uuid.UUID) error

	// Company Framework Results
	CreateResult(ctx context.Context, result *CompanyFrameworkResult) error
	GetResultByID(ctx context.Context, id uuid.UUID) (*CompanyFrameworkResult, error)
	GetCurrentResult(ctx context.Context, companyID, frameworkID uuid.UUID, challengeID *uuid.UUID) (*CompanyFrameworkResult, error)
	GetCompanyResults(ctx context.Context, companyID uuid.UUID, challengeID *uuid.UUID) ([]*CompanyFrameworkResult, error)
	GetCompanyResultsByStatus(ctx context.Context, companyID uuid.UUID, status string) ([]*CompanyFrameworkResult, error)
	UpdateResultStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error
	UpdateResultCompleted(ctx context.Context, id uuid.UUID, result []byte, generatedAt time.Time) error
	UpdateResultData(ctx context.Context, id uuid.UUID, result json.RawMessage) error
	InvalidateResults(ctx context.Context, companyID uuid.UUID, frameworkID uuid.UUID) error

	// Transaction support
	WithTx(ctx context.Context, fn func(Repository) error) error
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
	tx *sqlx.Tx
}

// NewRepository creates a new PostgreSQL-backed repository
func NewRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// querier returns either the transaction or the database connection
func (r *PostgresRepository) querier() sqlx.ExtContext {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// WithTx executes a function within a transaction
func (r *PostgresRepository) WithTx(ctx context.Context, fn func(Repository) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	txRepo := &PostgresRepository{db: r.db, tx: tx}

	if err := fn(txRepo); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	return nil
}

// =============================================================================
// Framework CRUD
// =============================================================================

// Create inserts a new framework
func (r *PostgresRepository) Create(ctx context.Context, f *Framework) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	now := time.Now()
	f.CreatedAt = now
	f.UpdatedAt = now

	query := `
		INSERT INTO frameworks (
			id, code, name, description, layer, is_base, is_active,
			prompt_system, prompt_user, prompt_json_template,
			model_config, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10,
			$11, $12, $13
		)`

	_, err := r.querier().ExecContext(ctx, query,
		f.ID, f.Code, f.Name, f.Description, f.Layer, f.IsBase, f.IsActive,
		f.PromptSystem, f.PromptUser, f.PromptJSONTemplate,
		f.ModelConfig, f.CreatedAt, f.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create framework: %w", err)
	}
	return nil
}

// GetByID retrieves a framework by ID
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Framework, error) {
	query := `SELECT * FROM frameworks WHERE id = $1`
	var f Framework
	if err := sqlx.GetContext(ctx, r.querier(), &f, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get framework by id: %w", err)
	}
	return &f, nil
}

// GetByCode retrieves a framework by code
func (r *PostgresRepository) GetByCode(ctx context.Context, code string) (*Framework, error) {
	query := `SELECT * FROM frameworks WHERE code = $1`
	var f Framework
	if err := sqlx.GetContext(ctx, r.querier(), &f, query, code); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get framework by code: %w", err)
	}
	return &f, nil
}

// GetActive retrieves all active frameworks ordered by layer
func (r *PostgresRepository) GetActive(ctx context.Context) ([]*Framework, error) {
	query := `SELECT * FROM frameworks WHERE is_active = true ORDER BY layer, code`
	var frameworks []*Framework
	if err := sqlx.SelectContext(ctx, r.querier(), &frameworks, query); err != nil {
		return nil, fmt.Errorf("failed to get active frameworks: %w", err)
	}
	return frameworks, nil
}

// GetBaseFrameworks retrieves all base (required) frameworks
func (r *PostgresRepository) GetBaseFrameworks(ctx context.Context) ([]*Framework, error) {
	query := `SELECT * FROM frameworks WHERE is_base = true AND is_active = true ORDER BY layer, code`
	var frameworks []*Framework
	if err := sqlx.SelectContext(ctx, r.querier(), &frameworks, query); err != nil {
		return nil, fmt.Errorf("failed to get base frameworks: %w", err)
	}
	return frameworks, nil
}

// GetByLayer retrieves frameworks by layer
func (r *PostgresRepository) GetByLayer(ctx context.Context, layer int) ([]*Framework, error) {
	query := `SELECT * FROM frameworks WHERE layer = $1 AND is_active = true ORDER BY code`
	var frameworks []*Framework
	if err := sqlx.SelectContext(ctx, r.querier(), &frameworks, query, layer); err != nil {
		return nil, fmt.Errorf("failed to get frameworks by layer: %w", err)
	}
	return frameworks, nil
}

// Update updates an existing framework
func (r *PostgresRepository) Update(ctx context.Context, f *Framework) error {
	f.UpdatedAt = time.Now()

	query := `
		UPDATE frameworks SET
			name = $2, description = $3, layer = $4, is_base = $5, is_active = $6,
			prompt_system = $7, prompt_user = $8, prompt_json_template = $9,
			model_config = $10, updated_at = $11
		WHERE id = $1`

	result, err := r.querier().ExecContext(ctx, query,
		f.ID, f.Name, f.Description, f.Layer, f.IsBase, f.IsActive,
		f.PromptSystem, f.PromptUser, f.PromptJSONTemplate,
		f.ModelConfig, f.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update framework: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("framework not found")
	}
	return nil
}

// =============================================================================
// Dependencies
// =============================================================================

// GetDependencies retrieves direct dependencies for a framework
func (r *PostgresRepository) GetDependencies(ctx context.Context, frameworkID uuid.UUID) ([]*FrameworkDependency, error) {
	query := `SELECT * FROM framework_dependencies WHERE framework_id = $1`
	var deps []*FrameworkDependency
	if err := sqlx.SelectContext(ctx, r.querier(), &deps, query, frameworkID); err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}
	return deps, nil
}

// GetAllDependencies retrieves all dependencies
func (r *PostgresRepository) GetAllDependencies(ctx context.Context) ([]*FrameworkDependency, error) {
	query := `SELECT * FROM framework_dependencies`
	var deps []*FrameworkDependency
	if err := sqlx.SelectContext(ctx, r.querier(), &deps, query); err != nil {
		return nil, fmt.Errorf("failed to get all dependencies: %w", err)
	}
	return deps, nil
}

// AddDependency adds a dependency between frameworks
func (r *PostgresRepository) AddDependency(ctx context.Context, dep *FrameworkDependency) error {
	if dep.ID == uuid.Nil {
		dep.ID = uuid.New()
	}
	dep.CreatedAt = time.Now()

	query := `
		INSERT INTO framework_dependencies (id, framework_id, depends_on_id, dependency_type, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (framework_id, depends_on_id) DO NOTHING`

	_, err := r.querier().ExecContext(ctx, query,
		dep.ID, dep.FrameworkID, dep.DependsOnID, dep.DependencyType, dep.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to add dependency: %w", err)
	}
	return nil
}

// RemoveDependency removes a dependency between frameworks
func (r *PostgresRepository) RemoveDependency(ctx context.Context, frameworkID, dependsOnID uuid.UUID) error {
	query := `DELETE FROM framework_dependencies WHERE framework_id = $1 AND depends_on_id = $2`
	_, err := r.querier().ExecContext(ctx, query, frameworkID, dependsOnID)
	if err != nil {
		return fmt.Errorf("failed to remove dependency: %w", err)
	}
	return nil
}

// =============================================================================
// Company Framework Results
// =============================================================================

// CreateResult creates a new company framework result
func (r *PostgresRepository) CreateResult(ctx context.Context, result *CompanyFrameworkResult) error {
	if result.ID == uuid.Nil {
		result.ID = uuid.New()
	}
	now := time.Now()
	result.CreatedAt = now
	result.UpdatedAt = now

	query := `
		INSERT INTO company_framework_results (
			id, company_id, framework_id, challenge_id,
			result, status, error_message,
			version, is_current, context_hash,
			generated_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9, $10,
			$11, $12, $13
		)`

	_, err := r.querier().ExecContext(ctx, query,
		result.ID, result.CompanyID, result.FrameworkID, result.ChallengeID,
		result.Result, result.Status, result.ErrorMessage,
		result.Version, result.IsCurrent, result.ContextHash,
		result.GeneratedAt, result.CreatedAt, result.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create result: %w", err)
	}
	return nil
}

// GetResultByID retrieves a result by ID
func (r *PostgresRepository) GetResultByID(ctx context.Context, id uuid.UUID) (*CompanyFrameworkResult, error) {
	query := `SELECT * FROM company_framework_results WHERE id = $1`
	var result CompanyFrameworkResult
	if err := sqlx.GetContext(ctx, r.querier(), &result, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get result by id: %w", err)
	}
	return &result, nil
}

// GetCurrentResult retrieves the current result for a company+framework+challenge combo
func (r *PostgresRepository) GetCurrentResult(ctx context.Context, companyID, frameworkID uuid.UUID, challengeID *uuid.UUID) (*CompanyFrameworkResult, error) {
	var query string
	var args []interface{}

	if challengeID != nil {
		query = `
			SELECT * FROM company_framework_results
			WHERE company_id = $1 AND framework_id = $2 AND challenge_id = $3 AND is_current = true`
		args = []interface{}{companyID, frameworkID, *challengeID}
	} else {
		query = `
			SELECT * FROM company_framework_results
			WHERE company_id = $1 AND framework_id = $2 AND challenge_id IS NULL AND is_current = true`
		args = []interface{}{companyID, frameworkID}
	}

	var result CompanyFrameworkResult
	if err := sqlx.GetContext(ctx, r.querier(), &result, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get current result: %w", err)
	}
	return &result, nil
}

// GetCompanyResults retrieves all current results for a company
func (r *PostgresRepository) GetCompanyResults(ctx context.Context, companyID uuid.UUID, challengeID *uuid.UUID) ([]*CompanyFrameworkResult, error) {
	var query string
	var args []interface{}

	if challengeID != nil {
		query = `
			SELECT * FROM company_framework_results
			WHERE company_id = $1 AND (challenge_id = $2 OR challenge_id IS NULL) AND is_current = true
			ORDER BY created_at`
		args = []interface{}{companyID, *challengeID}
	} else {
		query = `
			SELECT * FROM company_framework_results
			WHERE company_id = $1 AND is_current = true
			ORDER BY created_at`
		args = []interface{}{companyID}
	}

	var results []*CompanyFrameworkResult
	if err := sqlx.SelectContext(ctx, r.querier(), &results, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get company results: %w", err)
	}
	return results, nil
}

// GetCompanyResultsByStatus retrieves results by status
func (r *PostgresRepository) GetCompanyResultsByStatus(ctx context.Context, companyID uuid.UUID, status string) ([]*CompanyFrameworkResult, error) {
	query := `
		SELECT * FROM company_framework_results
		WHERE company_id = $1 AND status = $2 AND is_current = true
		ORDER BY created_at`

	var results []*CompanyFrameworkResult
	if err := sqlx.SelectContext(ctx, r.querier(), &results, query, companyID, status); err != nil {
		return nil, fmt.Errorf("failed to get results by status: %w", err)
	}
	return results, nil
}

// UpdateResultStatus updates the status of a result
func (r *PostgresRepository) UpdateResultStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error {
	query := `
		UPDATE company_framework_results
		SET status = $2, error_message = $3, updated_at = $4
		WHERE id = $1`

	_, err := r.querier().ExecContext(ctx, query, id, status, errorMsg, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update result status: %w", err)
	}
	return nil
}

// UpdateResultCompleted marks a result as completed with data
func (r *PostgresRepository) UpdateResultCompleted(ctx context.Context, id uuid.UUID, result []byte, generatedAt time.Time) error {
	query := `
		UPDATE company_framework_results
		SET status = $2, result = $3, generated_at = $4, updated_at = $5, error_message = NULL
		WHERE id = $1`

	_, err := r.querier().ExecContext(ctx, query, id, StatusCompleted, result, generatedAt, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update result completed: %w", err)
	}
	return nil
}

// InvalidateResults marks all current results for a company+framework as not current
func (r *PostgresRepository) InvalidateResults(ctx context.Context, companyID uuid.UUID, frameworkID uuid.UUID) error {
	query := `
		UPDATE company_framework_results
		SET is_current = false, updated_at = $3
		WHERE company_id = $1 AND framework_id = $2 AND is_current = true`

	_, err := r.querier().ExecContext(ctx, query, companyID, frameworkID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to invalidate results: %w", err)
	}
	return nil
}

// UpdateResultData updates the result JSON data for user editing
func (r *PostgresRepository) UpdateResultData(ctx context.Context, id uuid.UUID, result json.RawMessage) error {
	query := `
		UPDATE company_framework_results
		SET result = $2, updated_at = $3
		WHERE id = $1`

	res, err := r.querier().ExecContext(ctx, query, id, result, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update result data: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrResultNotFound
	}
	return nil
}
