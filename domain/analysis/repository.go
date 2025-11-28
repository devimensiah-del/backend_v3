package analysis

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repository defines data access methods for analysis
type Repository interface {
	Create(ctx context.Context, analysis *Analysis) error
	Update(ctx context.Context, analysis *Analysis) error
	UpdateWithTx(ctx context.Context, tx *sqlx.Tx, analysis *Analysis) error // Transactional update
	GetByID(ctx context.Context, id string) (*Analysis, error)
	GetBySubmissionID(ctx context.Context, submissionID string) (*Analysis, error)
	List(ctx context.Context, limit, offset int) ([]*Analysis, error)
	Delete(ctx context.Context, id string) error
	BeginTx(ctx context.Context) (*sqlx.Tx, error)                          // Begin transaction
	SetVisibility(ctx context.Context, id string, visible bool) error       // Toggle user visibility
	SetBlurStatus(ctx context.Context, id string, blurred bool) error      // Toggle blur status for premium frameworks
	GetByAccessCode(ctx context.Context, code string) (*Analysis, error)    // Get by public access code
	SetAccessCode(ctx context.Context, id string, code string) error        // Set access code
	AccessCodeExists(ctx context.Context, code string) (bool, error)        // Check if code exists (for collision handling)
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
			id, submission_id, enrichment_id,
			swot, pestel, porter, okrs, tam_sam_som, benchmarking, blue_ocean, growth_hacking, scenarios, bsc, decision_matrix,
			synthesis, status, error_message, processing_time_ms,
			is_visible_to_user, is_blurred, access_code, access_code_created_at, deleted_at,
			created_at, updated_at, completed_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18,
			$19, $20, $21, $22, $23,
			$24, $25, $26
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		analysis.ID, analysis.SubmissionID, analysis.EnrichmentID,
		analysis.SWOT, analysis.PESTEL, analysis.Porter, analysis.OKRs, analysis.TamSamSom, analysis.Benchmarking, analysis.BlueOcean, analysis.GrowthHacking, analysis.Scenarios, analysis.BSC, analysis.DecisionMatrix,
		analysis.Synthesis, analysis.Status, analysis.ErrorMessage, analysis.ProcessingTimeMs,
		analysis.IsVisibleToUser, analysis.IsBlurred, analysis.AccessCode, analysis.AccessCodeCreatedAt, analysis.DeletedAt,
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
			swot = $1,
			pestel = $2,
			porter = $3,
			okrs = $4,
			tam_sam_som = $5,
			benchmarking = $6,
			blue_ocean = $7,
			growth_hacking = $8,
			scenarios = $9,
			bsc = $10,
			decision_matrix = $11,
			synthesis = $12,
			status = $13,
			error_message = $14,
			processing_time_ms = $15,
			is_visible_to_user = $16,
			is_blurred = $17,
			access_code = $18,
			access_code_created_at = $19,
			deleted_at = $20,
			updated_at = $21,
			completed_at = $22
		WHERE id = $23
	`

	result, err := r.db.ExecContext(ctx, query,
		analysis.SWOT, analysis.PESTEL, analysis.Porter, analysis.OKRs, analysis.TamSamSom,
		analysis.Benchmarking, analysis.BlueOcean, analysis.GrowthHacking, analysis.Scenarios, analysis.BSC,
		analysis.DecisionMatrix, analysis.Synthesis, analysis.Status, analysis.ErrorMessage, analysis.ProcessingTimeMs,
		analysis.IsVisibleToUser, analysis.IsBlurred, analysis.AccessCode, analysis.AccessCodeCreatedAt,
		analysis.DeletedAt, analysis.UpdatedAt, analysis.CompletedAt, analysis.ID,
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
			swot = $1,
			pestel = $2,
			porter = $3,
			okrs = $4,
			tam_sam_som = $5,
			benchmarking = $6,
			blue_ocean = $7,
			growth_hacking = $8,
			scenarios = $9,
			bsc = $10,
			decision_matrix = $11,
			synthesis = $12,
			status = $13,
			error_message = $14,
			processing_time_ms = $15,
			is_visible_to_user = $16,
			is_blurred = $17,
			access_code = $18,
			access_code_created_at = $19,
			deleted_at = $20,
			updated_at = $21,
			completed_at = $22
		WHERE id = $23
	`

	result, err := tx.ExecContext(ctx, query,
		analysis.SWOT, analysis.PESTEL, analysis.Porter, analysis.OKRs, analysis.TamSamSom,
		analysis.Benchmarking, analysis.BlueOcean, analysis.GrowthHacking, analysis.Scenarios, analysis.BSC,
		analysis.DecisionMatrix, analysis.Synthesis, analysis.Status, analysis.ErrorMessage, analysis.ProcessingTimeMs,
		analysis.IsVisibleToUser, analysis.IsBlurred, analysis.AccessCode, analysis.AccessCodeCreatedAt,
		analysis.DeletedAt, analysis.UpdatedAt, analysis.CompletedAt, analysis.ID,
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
			id, submission_id, enrichment_id,
			swot, pestel, porter, okrs, tam_sam_som, benchmarking, blue_ocean, growth_hacking, scenarios, bsc, decision_matrix,
			synthesis, status, error_message, processing_time_ms,
			is_visible_to_user, is_blurred, access_code, access_code_created_at, deleted_at,
			created_at, updated_at, completed_at
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

// GetBySubmissionID retrieves an analysis by submission ID
func (r *PostgresRepository) GetBySubmissionID(ctx context.Context, submissionID string) (*Analysis, error) {
	query := `
		SELECT
			id, submission_id, enrichment_id,
			swot, pestel, porter, okrs, tam_sam_som, benchmarking, blue_ocean, growth_hacking, scenarios, bsc, decision_matrix,
			synthesis, status, error_message, processing_time_ms,
			is_visible_to_user, is_blurred, access_code, access_code_created_at, deleted_at,
			created_at, updated_at, completed_at
		FROM analyses
		WHERE submission_id = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	var analysis Analysis
	err := r.db.GetContext(ctx, &analysis, query, submissionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("analysis not found for submission: %s", submissionID)
		}
		return nil, fmt.Errorf("failed to get analysis: %w", err)
	}

	return &analysis, nil
}

// List retrieves all analyses with pagination
func (r *PostgresRepository) List(ctx context.Context, limit, offset int) ([]*Analysis, error) {
	query := `
		SELECT
			id, submission_id, enrichment_id,
			swot, pestel, porter, okrs, tam_sam_som, benchmarking, blue_ocean, growth_hacking, scenarios, bsc, decision_matrix,
			synthesis, status, error_message, processing_time_ms,
			is_visible_to_user, is_blurred, access_code, access_code_created_at, deleted_at,
			created_at, updated_at, completed_at
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

// SetBlurStatus toggles the is_blurred flag for an analysis
// This controls whether premium frameworks are blurred for users (paywall)
// Independent of is_visible_to_user which controls access
func (r *PostgresRepository) SetBlurStatus(ctx context.Context, id string, blurred bool) error {
	query := `UPDATE analyses SET is_blurred = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, blurred, id)
	if err != nil {
		return fmt.Errorf("failed to update blur status: %w", err)
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
			id, submission_id, enrichment_id,
			swot, pestel, porter, okrs, tam_sam_som, benchmarking, blue_ocean, growth_hacking, scenarios, bsc, decision_matrix,
			synthesis, status, error_message, processing_time_ms,
			is_visible_to_user, is_blurred, access_code, access_code_created_at, deleted_at,
			created_at, updated_at, completed_at
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
	query := `UPDATE analyses SET access_code = $1, access_code_created_at = NOW(), updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`

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
