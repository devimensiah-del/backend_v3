package analysis

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repository defines data access methods for analysis
// Frontend developers: This handles all database operations for analyses
type Repository interface {
	Create(ctx context.Context, analysis *Analysis) error
	Update(ctx context.Context, analysis *Analysis) error
	GetByID(ctx context.Context, id string) (*Analysis, error)
	GetBySubmissionID(ctx context.Context, submissionID string) (*Analysis, error)
	List(ctx context.Context, limit, offset int) ([]*Analysis, error)
	Delete(ctx context.Context, id string) error
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
func (r *PostgresRepository) Create(ctx context.Context, analysis *Analysis) error {
	query := `
		INSERT INTO analyses (
			id, submission_id, enrichment_id,
			swot, pestel, porter, okrs, tam_sam_som, benchmarking, blue_ocean, growth_hacking, scenarios, bsc, decision_matrix,
			synthesis, status, error_message, processing_time_ms,
			created_at, updated_at, completed_at
		) VALUES (
			:id, :submission_id, :enrichment_id,
			:swot, :pestel, :porter, :okrs, :tam_sam_som, :benchmarking, :blue_ocean, :growth_hacking, :scenarios, :bsc, :decision_matrix,
			:synthesis, :status, :error_message, :processing_time_ms,
			:created_at, :updated_at, :completed_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, analysis)
	if err != nil {
		return fmt.Errorf("failed to create analysis: %w", err)
	}

	return nil
}

// Update modifies an existing analysis record
func (r *PostgresRepository) Update(ctx context.Context, analysis *Analysis) error {
	query := `
		UPDATE analyses SET
			swot = :swot,
			pestel = :pestel,
			porter = :porter,
			okrs = :okrs,
			tam_sam_som = :tam_sam_som,
			benchmarking = :benchmarking,
			blue_ocean = :blue_ocean,
			growth_hacking = :growth_hacking,
			scenarios = :scenarios,
			bsc = :bsc,
			decision_matrix = :decision_matrix,
			synthesis = :synthesis,
			status = :status,
			error_message = :error_message,
			processing_time_ms = :processing_time_ms,
			updated_at = :updated_at,
			completed_at = :completed_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, analysis)
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

// GetByID retrieves an analysis by its ID
func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*Analysis, error) {
	query := `
		SELECT
			id, submission_id, enrichment_id,
			swot, pestel, porter, okrs, tam_sam_som, benchmarking, blue_ocean, growth_hacking, scenarios, bsc, decision_matrix,
			synthesis, status, error_message, processing_time_ms,
			created_at, updated_at, completed_at
		FROM analyses
		WHERE id = $1
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
			created_at, updated_at, completed_at
		FROM analyses
		WHERE submission_id = $1
		ORDER BY created_at DESC
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
// Frontend developers: Use this for admin dashboard listing
func (r *PostgresRepository) List(ctx context.Context, limit, offset int) ([]*Analysis, error) {
	query := `
		SELECT
			id, submission_id, enrichment_id,
			swot, pestel, porter, okrs, tam_sam_som, benchmarking, blue_ocean, growth_hacking, scenarios, bsc, decision_matrix,
			synthesis, status, error_message, processing_time_ms,
			created_at, updated_at, completed_at
		FROM analyses
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

// Delete removes an analysis record
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM analyses WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete analysis: %w", err)
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
