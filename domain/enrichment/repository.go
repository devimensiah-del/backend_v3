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
	// ForceUpdateAndUnlock updates enrichment and forces unlock (used by Worker on completion)
	ForceUpdateAndUnlock(ctx context.Context, e *Enrichment) error
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
			sources_status, sources_used, data, started_at, completed_at,
			error_message, retry_count, max_retries, created_at, updated_at
		) VALUES (
			:id, :submission_id, :status, :progress, :current_step, :is_locked,
			:sources_status, :sources_used, :data, :started_at, :completed_at,
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

	fmt.Printf("[DEBUG] GetBySubmissionID: querying for submission_id=%s\n", submissionID.String())

	var e Enrichment
	if err := r.db.GetContext(ctx, &e, query, submissionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Printf("[DEBUG] GetBySubmissionID: no rows found for submission_id=%s\n", submissionID.String())
			return nil, fmt.Errorf("enrichment not found")
		}
		fmt.Printf("[DEBUG] GetBySubmissionID: query failed with error: %v\n", err)
		return nil, fmt.Errorf("failed to get enrichment: %w", err)
	}

	fmt.Printf("[DEBUG] GetBySubmissionID: found enrichment id=%s, status=%s, data_size=%d\n",
		e.ID.String(), e.Status, len(e.EnrichedData))

	return &e, nil
}

// UpdateSystem updates enrichment ONLY if it is NOT locked by a user
// This is used by the Background Worker
// CRITICAL: The AND is_locked = FALSE clause prevents the worker from overwriting user edits
func (r *PostgresRepository) UpdateSystem(ctx context.Context, e *Enrichment) error {
	e.UpdatedAt = time.Now()

	fmt.Printf("[DEBUG] UpdateSystem: id=%s, status=%s, progress=%d, is_locked=%v, data_size=%d\n",
		e.ID.String(), e.Status, e.Progress, e.IsLocked, len(e.EnrichedData))

	query := `
		UPDATE enrichments SET
			status = :status,
			progress = :progress,
			current_step = :current_step,
			sources_status = :sources_status,
			sources_used = :sources_used,
			data = :data,
			started_at = :started_at,
			completed_at = :completed_at,
			error_message = :error_message,
			retry_count = :retry_count,
			updated_at = :updated_at
		WHERE id = :id
		AND is_locked = FALSE
	`

	result, err := r.db.NamedExecContext(ctx, query, e)
	if err != nil {
		fmt.Printf("[DEBUG] UpdateSystem: SQL error: %v\n", err)
		return fmt.Errorf("failed to update enrichment: %w", err)
	}

	// Check if any rows were affected
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	fmt.Printf("[DEBUG] UpdateSystem: rows affected=%d for id=%s\n", rows, e.ID.String())

	// If 0 rows affected, enrichment is locked by user OR not found
	if rows == 0 {
		// Check actual state in DB for debugging
		var dbLocked bool
		var dbID string
		checkQuery := `SELECT id, is_locked FROM enrichments WHERE id = $1`
		checkErr := r.db.QueryRowContext(ctx, checkQuery, e.ID).Scan(&dbID, &dbLocked)
		if checkErr != nil {
			fmt.Printf("[DEBUG] UpdateSystem: check query failed: %v\n", checkErr)
		} else {
			fmt.Printf("[DEBUG] UpdateSystem: DB state - id=%s, is_locked=%v\n", dbID, dbLocked)
		}
		return fmt.Errorf("enrichment is locked by user or not found - cannot update (progress: %d%%)", e.Progress)
	}

	return nil
}

// UpdateUser updates enrichment and LOCKS it
// This is used by the API (User Dashboard)
func (r *PostgresRepository) UpdateUser(ctx context.Context, e *Enrichment) error {
	e.UpdatedAt = time.Now()
	e.IsLocked = true // Force lock

	query := `
		UPDATE enrichments SET
			data = :data,
			sources_used = :sources_used,
			is_locked = TRUE,
			updated_at = :updated_at
		WHERE id = :id
		AND status != 'approved'
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

// ForceUpdateAndUnlock updates enrichment and forces unlock
// This is used by the worker to complete enrichment even if user locked it
func (r *PostgresRepository) ForceUpdateAndUnlock(ctx context.Context, e *Enrichment) error {
	e.UpdatedAt = time.Now()
	e.IsLocked = false // Force unlock

	fmt.Printf("[DEBUG] ForceUpdateAndUnlock: id=%s, status=%s, progress=%d, data_size=%d\n",
		e.ID.String(), e.Status, e.Progress, len(e.EnrichedData))

	query := `
		UPDATE enrichments SET
			status = :status,
			progress = :progress,
			current_step = :current_step,
			sources_status = :sources_status,
			sources_used = :sources_used,
			data = :data,
			started_at = :started_at,
			completed_at = :completed_at,
			error_message = :error_message,
			retry_count = :retry_count,
			is_locked = FALSE,
			updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, e)
	if err != nil {
		fmt.Printf("[DEBUG] ForceUpdateAndUnlock: SQL error: %v\n", err)
		return fmt.Errorf("failed to force update enrichment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	fmt.Printf("[DEBUG] ForceUpdateAndUnlock: rows affected=%d for id=%s\n", rows, e.ID.String())

	if rows == 0 {
		// Check if enrichment exists at all
		var exists bool
		checkErr := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM enrichments WHERE id = $1)`, e.ID).Scan(&exists)
		if checkErr != nil {
			fmt.Printf("[DEBUG] ForceUpdateAndUnlock: existence check failed: %v\n", checkErr)
		} else {
			fmt.Printf("[DEBUG] ForceUpdateAndUnlock: enrichment exists=%v for id=%s\n", exists, e.ID.String())
		}
		return fmt.Errorf("enrichment not found")
	}

	return nil
}
