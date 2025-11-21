package enrichment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for enrichment data access
type Repository interface {
	Create(ctx context.Context, e *Enrichment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Enrichment, error)
	GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*Enrichment, error)
	// UpdateSystem updates enrichment ONLY if it is NOT locked by a user (used by Worker)
	UpdateSystem(ctx context.Context, e *Enrichment) error
	// UpdateUser updates enrichment and LOCKS it (used by API)
	UpdateUser(ctx context.Context, e *Enrichment) error
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new PostgreSQL repository
func NewRepository(db *sqlx.DB) Repository {
	return &PostgresRepository{db: db}
}

// Create inserts a new enrichment
func (r *PostgresRepository) Create(ctx context.Context, e *Enrichment) error {
	// sqlx automatically calls .Value() on JSONMap fields
	query := `
		INSERT INTO enrichments (
			id, submission_id, status, progress, current_step, is_locked,
			sources_status, enriched_data, started_at, completed_at,
			error_message, retry_count, max_retries, created_at, updated_at
		) VALUES (
			:id, :submission_id, :status, :progress, :current_step, :is_locked,
			:sources_status, :enriched_data, :started_at, :completed_at,
			:error_message, :retry_count, :max_retries, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, e)
	if err != nil {
		return fmt.Errorf("failed to insert enrichment: %w", err)
	}

	return nil
}

// GetByID retrieves an enrichment by ID
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Enrichment, error) {
	query := `SELECT * FROM enrichments WHERE id = $1`

	var e Enrichment
	// sqlx automatically calls .Scan() on JSONMap fields
	if err := r.db.GetContext(ctx, &e, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("enrichment not found")
		}
		return nil, fmt.Errorf("failed to get enrichment: %w", err)
	}

	return &e, nil
}

// GetBySubmissionID retrieves enrichment for a submission
func (r *PostgresRepository) GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*Enrichment, error) {
	query := `
		SELECT * FROM enrichments
		WHERE submission_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var e Enrichment
	if err := r.db.GetContext(ctx, &e, query, submissionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("enrichment not found")
		}
		return nil, fmt.Errorf("failed to get enrichment: %w", err)
	}

	return &e, nil
}

// UpdateSystem updates enrichment ONLY if it is NOT locked by a user
// This is used by the Background Worker
func (r *PostgresRepository) UpdateSystem(ctx context.Context, e *Enrichment) error {
	e.UpdatedAt = time.Now()

	// We explicitly add "AND is_locked = false" to the WHERE clause
	query := `
		UPDATE enrichments SET
			status = :status, 
			progress = :progress, 
			current_step = :current_step,
			sources_status = :sources_status, 
			enriched_data = :enriched_data, 
			started_at = :started_at, 
			completed_at = :completed_at,
			error_message = :error_message, 
			retry_count = :retry_count, 
			updated_at = :updated_at
		WHERE id = :id AND is_locked = FALSE
	`

	result, err := r.db.NamedExecContext(ctx, query, e)
	if err != nil {
		return fmt.Errorf("failed to update enrichment: %w", err)
	}

	// If 0 rows affected, it means the user locked it.
	// We intentionally DO NOT return an error here, because the worker should just "skip" saving.
	// The user's data is now the source of truth.
	_, _ = result.RowsAffected()

	return nil
}

// UpdateUser updates enrichment and LOCKS it
// This is used by the API (User Dashboard)
func (r *PostgresRepository) UpdateUser(ctx context.Context, e *Enrichment) error {
	e.UpdatedAt = time.Now()
	e.IsLocked = true // Force lock

	query := `
		UPDATE enrichments SET
			enriched_data = :enriched_data, 
			is_locked = TRUE,
			updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, e)
	if err != nil {
		return fmt.Errorf("failed to update enrichment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("enrichment not found")
	}

	return nil
}
