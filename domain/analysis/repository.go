package analysis

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Repository defines data access methods for analysis
type Repository interface {
	Create(ctx context.Context, analysis *Analysis) error
	Update(ctx context.Context, analysis *Analysis) error
	UpdateWithTx(ctx context.Context, tx *sqlx.Tx, analysis *Analysis) error // Transactional update
	GetByID(ctx context.Context, id string) (*Analysis, error)
	GetByChallengeID(ctx context.Context, challengeID uuid.UUID) (*Analysis, error)
	List(ctx context.Context, limit, offset int) ([]*Analysis, error)
	Delete(ctx context.Context, id string) error
	BeginTx(ctx context.Context) (*sqlx.Tx, error)                       // Begin transaction
	SetVisibility(ctx context.Context, id string, visible bool) error    // Toggle user visibility
	SetPublicStatus(ctx context.Context, id string, public bool) error   // Toggle public access (no login required)
	GetByAccessCode(ctx context.Context, code string) (*Analysis, error) // Get by public access code
	SetAccessCode(ctx context.Context, id string, code string) error     // Set access code
	AccessCodeExists(ctx context.Context, code string) (bool, error)     // Check if code exists (for collision handling)
	GetFrameworkResult(ctx context.Context, analysisID, frameworkCode string) (json.RawMessage, error)
	SetFrameworkResult(ctx context.Context, analysisID, frameworkCode string, result json.RawMessage) error

	// Wizard methods (analysis_steps table)
	GetAnalysisStep(ctx context.Context, analysisID, frameworkCode string) (*AnalysisStep, error)
	SaveAnalysisStep(ctx context.Context, step *AnalysisStep) error
	GetAllAnalysisSteps(ctx context.Context, analysisID string) ([]*AnalysisStep, error)

	// Versioning
	IncrementVersion(ctx context.Context, analysisID string) error
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create inserts a new analysis record
// Uses positional parameters ($1, $2, ...) instead of named parameters
// to avoid lib/pq driver issues with custom JSONB types implementing driver.Valuer
func (r *PostgresRepository) Create(ctx context.Context, analysis *Analysis) error {
	query := `
		INSERT INTO analyses (
			id, company_id, challenge_id,
			framework_results, status, error_message,
			is_visible_to_user, is_public, access_code, deleted_at,
			created_at, updated_at, completed_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13
		)
	`

	// Serialize framework_results to JSON
	frameworkResultsJSON, err := json.Marshal(analysis.FrameworkResults)
	if err != nil {
		return fmt.Errorf("failed to marshal framework_results: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		analysis.ID, analysis.CompanyID, analysis.ChallengeID,
		string(frameworkResultsJSON), analysis.Status, analysis.ErrorMessage,
		analysis.IsVisibleToUser, analysis.IsPublic, analysis.AccessCode, analysis.DeletedAt,
		analysis.CreatedAt, analysis.UpdatedAt, analysis.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create analysis: %w", err)
	}

	return nil
}

// Update modifies an existing analysis record
// Uses positional parameters to avoid lib/pq driver issues with JSONB types
func (r *PostgresRepository) Update(ctx context.Context, analysis *Analysis) error {
	query := `
		UPDATE analyses SET
			framework_results = $1,
			status = $2,
			error_message = $3,
			is_visible_to_user = $4,
			is_public = $5,
			access_code = $6,
			deleted_at = $7,
			updated_at = $8,
			completed_at = $9,
			current_step = $10,
			wizard_mode = $11,
			steps_completed = $12
		WHERE id = $13
	`

	// Serialize framework_results to JSON
	frameworkResultsJSON, err := json.Marshal(analysis.FrameworkResults)
	if err != nil {
		return fmt.Errorf("failed to marshal framework_results: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query,
		string(frameworkResultsJSON), analysis.Status, analysis.ErrorMessage,
		analysis.IsVisibleToUser, analysis.IsPublic, analysis.AccessCode,
		analysis.DeletedAt, analysis.UpdatedAt, analysis.CompletedAt,
		analysis.CurrentStep, analysis.WizardMode, pq.Array(analysis.StepsCompleted),
		analysis.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update analysis: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("analysis not found: %s", analysis.ID)
	}

	return nil
}

// UpdateWithTx modifies an existing analysis record within a transaction
// Uses positional parameters to avoid lib/pq driver issues with JSONB types
func (r *PostgresRepository) UpdateWithTx(ctx context.Context, tx *sqlx.Tx, analysis *Analysis) error {
	query := `
		UPDATE analyses SET
			framework_results = $1,
			status = $2,
			error_message = $3,
			is_visible_to_user = $4,
			is_public = $5,
			access_code = $6,
			deleted_at = $7,
			updated_at = $8,
			completed_at = $9,
			current_step = $10,
			wizard_mode = $11,
			steps_completed = $12
		WHERE id = $13
	`

	// Serialize framework_results to JSON
	frameworkResultsJSON, err := json.Marshal(analysis.FrameworkResults)
	if err != nil {
		return fmt.Errorf("failed to marshal framework_results: %w", err)
	}

	result, err := tx.ExecContext(ctx, query,
		string(frameworkResultsJSON), analysis.Status, analysis.ErrorMessage,
		analysis.IsVisibleToUser, analysis.IsPublic, analysis.AccessCode,
		analysis.DeletedAt, analysis.UpdatedAt, analysis.CompletedAt,
		analysis.CurrentStep, analysis.WizardMode, pq.Array(analysis.StepsCompleted),
		analysis.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update analysis in transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("analysis not found: %s", analysis.ID)
	}

	return nil
}

// BeginTx starts a new database transaction
func (r *PostgresRepository) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

// GetByID retrieves an analysis by its ID
func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*Analysis, error) {
	query := `
		SELECT
			id, company_id, challenge_id,
			framework_results, status, error_message,
			is_visible_to_user, is_public, access_code, deleted_at,
			created_at, updated_at, completed_at,
			version, current_step, wizard_mode, steps_completed
		FROM analyses
		WHERE id = $1 AND deleted_at IS NULL
	`

	var analysis Analysis
	err := r.db.GetContext(ctx, &analysis, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("analysis not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get analysis: %w", err)
	}

	return &analysis, nil
}

// GetByChallengeID retrieves an analysis by challenge ID
func (r *PostgresRepository) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) (*Analysis, error) {
	query := `
		SELECT
			id, company_id, challenge_id,
			framework_results, status, error_message,
			is_visible_to_user, is_public, access_code, deleted_at,
			created_at, updated_at, completed_at,
			version, current_step, wizard_mode, steps_completed
		FROM analyses
		WHERE challenge_id = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	var analysis Analysis
	err := r.db.GetContext(ctx, &analysis, query, challengeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("analysis not found for challenge: %s", challengeID)
		}
		return nil, fmt.Errorf("failed to get analysis: %w", err)
	}

	return &analysis, nil
}

// List retrieves all analyses with pagination
func (r *PostgresRepository) List(ctx context.Context, limit, offset int) ([]*Analysis, error) {
	query := `
		SELECT
			id, company_id, challenge_id,
			framework_results, status, error_message,
			is_visible_to_user, is_public, access_code, deleted_at,
			created_at, updated_at, completed_at,
			version, current_step, wizard_mode, steps_completed
		FROM analyses
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	var analyses []*Analysis
	err := r.db.SelectContext(ctx, &analyses, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list analyses: %w", err)
	}

	return analyses, nil
}

// Delete soft-deletes an analysis record by setting deleted_at timestamp
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE analyses SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete analysis: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("analysis not found or already deleted: %s", id)
	}

	return nil
}

// SetVisibility toggles the is_visible_to_user flag for an analysis
// This controls whether end users can see the analysis and download the PDF
func (r *PostgresRepository) SetVisibility(ctx context.Context, id string, visible bool) error {
	query := `UPDATE analyses SET is_visible_to_user = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, visible, id)
	if err != nil {
		return fmt.Errorf("failed to update visibility: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("analysis not found: %s", id)
	}

	return nil
}

// SetPublicStatus toggles the is_public flag for an analysis
// This controls whether the report is accessible without authentication
// When true, anyone with the access code can view the report
// When false, the access code requires authentication
func (r *PostgresRepository) SetPublicStatus(ctx context.Context, id string, public bool) error {
	query := `UPDATE analyses SET is_public = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, public, id)
	if err != nil {
		return fmt.Errorf("failed to update public status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("analysis not found: %s", id)
	}

	return nil
}

// GetByAccessCode retrieves an analysis by its public access code
// Returns nil, nil if code doesn't exist (for 404 handling)
func (r *PostgresRepository) GetByAccessCode(ctx context.Context, code string) (*Analysis, error) {
	query := `
		SELECT
			id, company_id, challenge_id,
			framework_results, status, error_message,
			is_visible_to_user, is_public, access_code, deleted_at,
			created_at, updated_at, completed_at,
			version, current_step, wizard_mode, steps_completed
		FROM analyses
		WHERE access_code = $1 AND deleted_at IS NULL
	`

	var analysis Analysis
	err := r.db.GetContext(ctx, &analysis, query, code)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found - return nil, nil for 404 handling
		}
		return nil, fmt.Errorf("failed to get analysis by access code: %w", err)
	}

	return &analysis, nil
}

// SetAccessCode sets or updates the access code for an analysis
func (r *PostgresRepository) SetAccessCode(ctx context.Context, id string, code string) error {
	query := `UPDATE analyses SET access_code = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, code, id)
	if err != nil {
		return fmt.Errorf("failed to set access code: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("analysis not found: %s", id)
	}

	return nil
}

// AccessCodeExists checks if an access code is already in use (for collision handling)
func (r *PostgresRepository) AccessCodeExists(ctx context.Context, code string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM analyses WHERE access_code = $1 AND deleted_at IS NULL)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, code)
	if err != nil {
		return false, fmt.Errorf("failed to check access code existence: %w", err)
	}

	return exists, nil
}

// GetFrameworkResult retrieves a single framework's result from the generic column
func (r *PostgresRepository) GetFrameworkResult(ctx context.Context, analysisID, frameworkCode string) (json.RawMessage, error) {
	var result json.RawMessage
	query := `
		SELECT framework_results->$2
		FROM analyses
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &result, query, analysisID, frameworkCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get framework result %s: %w", frameworkCode, err)
	}
	return result, nil
}

// SetFrameworkResult updates a single framework result in the generic column
func (r *PostgresRepository) SetFrameworkResult(ctx context.Context, analysisID, frameworkCode string, result json.RawMessage) error {
	query := `
		UPDATE analyses
		SET framework_results = jsonb_set(
			COALESCE(framework_results, '{}'),
			ARRAY[$2],
			$3::jsonb
		),
		updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query, analysisID, frameworkCode, result)
	if err != nil {
		return fmt.Errorf("failed to set framework result %s: %w", frameworkCode, err)
	}
	return nil
}

// --- Wizard Methods (analysis_steps table) ---

// GetAnalysisStep retrieves a specific step by analysis ID and framework code
func (r *PostgresRepository) GetAnalysisStep(ctx context.Context, analysisID, frameworkCode string) (*AnalysisStep, error) {
	query := `
		SELECT
			id, analysis_id, step_number, framework_code, status,
			COALESCE(output, 'null'::jsonb) as output,
			human_context, refinement_notes,
			COALESCE(clarifying_questions, 'null'::jsonb) as clarifying_questions,
			COALESCE(human_answers, 'null'::jsonb) as human_answers,
			iteration_count, error_message, processing_time_ms,
			generated_at, approved_at, approved_by,
			created_at, updated_at,
			COALESCE(previous_outputs, '[]'::jsonb) as previous_outputs
		FROM analysis_steps
		WHERE analysis_id = $1 AND framework_code = $2
	`

	var step AnalysisStep
	err := r.db.GetContext(ctx, &step, query, analysisID, frameworkCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get analysis step (analysisID=%s, code=%s): %w", analysisID, frameworkCode, err)
	}

	return &step, nil
}

// jsonOrNull converts nil json.RawMessage to SQL NULL, otherwise returns as-is
// This is needed because PostgreSQL JSONB columns reject nil/empty bytes as invalid JSON
func jsonOrNull(data json.RawMessage) interface{} {
	if len(data) == 0 {
		return nil
	}
	return data
}

// SaveAnalysisStep inserts or updates an analysis step
func (r *PostgresRepository) SaveAnalysisStep(ctx context.Context, step *AnalysisStep) error {
	query := `
		INSERT INTO analysis_steps (
			id, analysis_id, step_number, framework_code, status,
			output, human_context, refinement_notes,
			clarifying_questions, human_answers,
			iteration_count, error_message, processing_time_ms,
			generated_at, approved_at, approved_by,
			created_at, updated_at, previous_outputs
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10,
			$11, $12, $13,
			$14, $15, $16,
			$17, $18, $19
		)
		ON CONFLICT (analysis_id, framework_code) DO UPDATE SET
			status = EXCLUDED.status,
			output = EXCLUDED.output,
			human_context = EXCLUDED.human_context,
			refinement_notes = EXCLUDED.refinement_notes,
			clarifying_questions = EXCLUDED.clarifying_questions,
			human_answers = EXCLUDED.human_answers,
			iteration_count = EXCLUDED.iteration_count,
			error_message = EXCLUDED.error_message,
			processing_time_ms = EXCLUDED.processing_time_ms,
			generated_at = EXCLUDED.generated_at,
			approved_at = EXCLUDED.approved_at,
			approved_by = EXCLUDED.approved_by,
			updated_at = EXCLUDED.updated_at,
			previous_outputs = EXCLUDED.previous_outputs
		RETURNING id
	`

	var returnedID string
	err := r.db.QueryRowContext(ctx, query,
		step.ID, step.AnalysisID, step.StepNumber, step.FrameworkCode, step.Status,
		jsonOrNull(step.Output), step.HumanContext, step.RefinementNotes,
		jsonOrNull(step.ClarifyingQuestions), jsonOrNull(step.HumanAnswers),
		step.IterationCount, step.ErrorMessage, step.ProcessingTimeMs,
		step.GeneratedAt, step.ApprovedAt, step.ApprovedBy,
		step.CreatedAt, step.UpdatedAt, jsonOrNull(step.PreviousOutputs),
	).Scan(&returnedID)
	if err != nil {
		return fmt.Errorf("failed to save analysis step (analysisID=%s, code=%s): %w", step.AnalysisID, step.FrameworkCode, err)
	}

	return nil
}

// GetAllAnalysisSteps retrieves all steps for an analysis
func (r *PostgresRepository) GetAllAnalysisSteps(ctx context.Context, analysisID string) ([]*AnalysisStep, error) {
	query := `
		SELECT
			id, analysis_id, step_number, framework_code, status,
			output, human_context, refinement_notes,
			clarifying_questions, human_answers,
			iteration_count, error_message, processing_time_ms,
			generated_at, approved_at, approved_by,
			created_at, updated_at
		FROM analysis_steps
		WHERE analysis_id = $1
		ORDER BY step_number ASC
	`

	var steps []*AnalysisStep
	err := r.db.SelectContext(ctx, &steps, query, analysisID)
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis steps: %w", err)
	}

	return steps, nil
}

// --- Versioning Methods ---

// IncrementVersion increments the version number on an analysis
func (r *PostgresRepository) IncrementVersion(ctx context.Context, analysisID string) error {
	query := `
		UPDATE analyses
		SET version = version + 1,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, analysisID)
	if err != nil {
		return fmt.Errorf("failed to increment version: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("analysis not found: %s", analysisID)
	}

	return nil
}
