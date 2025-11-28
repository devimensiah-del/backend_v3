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
	GetByCompanyID(ctx context.Context, companyID uuid.UUID) (*Enrichment, error)
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
			id, submission_id, company_id, status, progress, current_step, is_locked,
			sources_status, sources_used, data, started_at, completed_at,
			error_message, retry_count, max_retries, auto_trigger_analysis, created_at, updated_at
		) VALUES (
			:id, :submission_id, :company_id, :status, :progress, :current_step, :is_locked,
			:sources_status, :sources_used, :data, :started_at, :completed_at,
			:error_message, :retry_count, :max_retries, :auto_trigger_analysis, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, e)
	if err != nil {
		return fmt.Errorf("failed to insert enrichment: %w", err)
	}

	return nil
}

// enrichmentColumns lists all columns the Enrichment struct expects
// This avoids SELECT * which fails if DB has extra columns
const enrichmentColumns = `
	id, submission_id, company_id, status, progress, current_step, is_locked,
	sources_status, sources_used, data, started_at, completed_at,
	error_message, retry_count, max_retries, auto_trigger_analysis, created_at, updated_at
`

// GetByID retrieves an enrichment by ID
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Enrichment, error) {
	query := `SELECT ` + enrichmentColumns + ` FROM enrichments WHERE id = $1`

	var e Enrichment
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
		SELECT ` + enrichmentColumns + ` FROM enrichments
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

// GetByCompanyID retrieves the latest enrichment for a company
// Returns the most recent enrichment linked directly to the company
func (r *PostgresRepository) GetByCompanyID(ctx context.Context, companyID uuid.UUID) (*Enrichment, error) {
	query := `
		SELECT ` + enrichmentColumns + ` FROM enrichments
		WHERE company_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var e Enrichment
	if err := r.db.GetContext(ctx, &e, query, companyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("enrichment not found for company")
		}
		return nil, fmt.Errorf("failed to get enrichment by company: %w", err)
	}

	return &e, nil
}

// UpdateSystem updates enrichment ONLY if it is NOT locked by a user
// This is used by the Background Worker
// CRITICAL: The AND is_locked = FALSE clause prevents the worker from overwriting user edits
func (r *PostgresRepository) UpdateSystem(ctx context.Context, e *Enrichment) error {
	e.UpdatedAt = time.Now()

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
		return fmt.Errorf("failed to update enrichment: %w", err)
	}

	// Check if any rows were affected
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// If 0 rows affected, enrichment is locked by user OR not found
	if rows == 0 {
		return fmt.Errorf("enrichment is locked by user or not found - cannot update (progress: %d%%)", e.Progress)
	}

	return nil
}

// UpdateUser updates enrichment and LOCKS it
// This is used by the API (User Dashboard)
// Note: Cannot edit completed enrichments via this method (use admin override if needed)
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
		AND status != 'completed'
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
		// Check if it exists but is completed
		var status string
		checkErr := r.db.QueryRowContext(ctx, `SELECT status FROM enrichments WHERE id = $1`, e.ID).Scan(&status)
		if checkErr == nil && status == "completed" {
			return fmt.Errorf("cannot edit completed enrichment - use admin endpoint to modify")
		}
		return fmt.Errorf("enrichment not found")
	}

	return nil
}

// ForceUpdateAndUnlock updates enrichment and forces unlock
// This is used by the worker to complete enrichment even if user locked it
func (r *PostgresRepository) ForceUpdateAndUnlock(ctx context.Context, e *Enrichment) error {
	e.UpdatedAt = time.Now()
	e.IsLocked = false // Force unlock

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
			auto_trigger_analysis = :auto_trigger_analysis,
			updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, e)
	if err != nil {
		return fmt.Errorf("failed to force update enrichment: %w", err)
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
