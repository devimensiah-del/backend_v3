package submission

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for submission data access
// This allows for easy testing and swapping implementations
type Repository interface {
	Create(ctx context.Context, s *Submission) error
	GetByID(ctx context.Context, id uuid.UUID) (*Submission, error)
	GetByEmail(ctx context.Context, email string) ([]*Submission, error)
	List(ctx context.Context, opts *ListOptions) ([]*Submission, int, error)
	Update(ctx context.Context, s *Submission) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Cross-table lookup used for idempotency checks before queueing enrichment jobs
	GetEnrichmentStatus(ctx context.Context, submissionID uuid.UUID) (*EnrichmentStatusRow, error)
	// ReserveEnrichment inserts an enrichment placeholder if one does not already exist (idempotent)
	ReserveEnrichment(ctx context.Context, submissionID uuid.UUID) (bool, error)

	// Analytics methods
	GetTotalCount(ctx context.Context) (int, error)
	GetActiveCount(ctx context.Context) (int, error)
	GetCompletedCount(ctx context.Context) (int, error)
}

// PostgresRepository implements Repository using PostgreSQL
// This is the production implementation
type PostgresRepository struct {
	db *sqlx.DB
}

// EnrichmentStatusRow mirrors the minimal columns needed for idempotency checks.
type EnrichmentStatusRow struct {
	Status      string     `db:"status"`
	StartedAt   *time.Time `db:"started_at"`
	CompletedAt *time.Time `db:"completed_at"`
}

// NewRepository creates a new PostgreSQL repository
func NewRepository(db *sqlx.DB) Repository {
	return &PostgresRepository{db: db}
}

// Create inserts a new submission into the database
func (r *PostgresRepository) Create(ctx context.Context, s *Submission) error {
	query := `
		INSERT INTO submissions (
			id, company_name, cnpj, company_website, company_industry, company_size, company_location,
			contact_name, contact_email, contact_phone, contact_position,
			target_market, annual_revenue_min, annual_revenue_max, funding_stage,
			business_challenge, additional_notes, linkedin_url, twitter_handle,
			status, user_id, created_at, updated_at
		) VALUES (
			:id, :company_name, :cnpj, :company_website, :company_industry, :company_size, :company_location,
			:contact_name, :contact_email, :contact_phone, :contact_position,
			:target_market, :annual_revenue_min, :annual_revenue_max, :funding_stage,
			:business_challenge, :additional_notes, :linkedin_url, :twitter_handle,
			:status, :user_id, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, s)
	if err != nil {
		return fmt.Errorf("failed to insert submission: %w", err)
	}

	return nil
}

// GetByID retrieves a submission by ID
// Uses explicit column list to ensure model/schema alignment
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Submission, error) {
	query := `
		SELECT
			id, company_name, cnpj, company_website, company_industry, company_size, company_location,
			contact_name, contact_email, contact_phone, contact_position,
			target_market, annual_revenue_min, annual_revenue_max, funding_stage,
			business_challenge, additional_notes, linkedin_url, twitter_handle,
			status, user_id, created_at, updated_at, deleted_at
		FROM submissions
		WHERE id = $1 AND deleted_at IS NULL
	`

	var submission Submission
	if err := r.db.GetContext(ctx, &submission, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("submission not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	return &submission, nil
}

// GetByEmail retrieves all submissions for an email address
// Uses explicit column list to ensure model/schema alignment
func (r *PostgresRepository) GetByEmail(ctx context.Context, email string) ([]*Submission, error) {
	query := `
		SELECT
			id, company_name, cnpj, company_website, company_industry, company_size, company_location,
			contact_name, contact_email, contact_phone, contact_position,
			target_market, annual_revenue_min, annual_revenue_max, funding_stage,
			business_challenge, additional_notes, linkedin_url, twitter_handle,
			status, user_id, created_at, updated_at, deleted_at
		FROM submissions
		WHERE contact_email = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	var submissions []*Submission
	if err := r.db.SelectContext(ctx, &submissions, query, email); err != nil {
		return nil, fmt.Errorf("failed to get submissions by email: %w", err)
	}

	return submissions, nil
}

// GetEnrichmentStatus returns the latest enrichment status for a submission (if any).
func (r *PostgresRepository) GetEnrichmentStatus(ctx context.Context, submissionID uuid.UUID) (*EnrichmentStatusRow, error) {
	query := `
		SELECT status, started_at, completed_at
		FROM enrichments
		WHERE submission_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var row EnrichmentStatusRow
	if err := r.db.GetContext(ctx, &row, query, submissionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to lookup enrichment status: %w", err)
	}

	return &row, nil
}

// ReserveEnrichment inserts a placeholder enrichment row to guarantee one-per-submission semantics.
// Returns true if a new row was created, false if it already existed.
func (r *PostgresRepository) ReserveEnrichment(ctx context.Context, submissionID uuid.UUID) (bool, error) {
	query := `
		INSERT INTO enrichments (submission_id)
		VALUES ($1)
		ON CONFLICT (submission_id) DO NOTHING
	`

	result, err := r.db.ExecContext(ctx, query, submissionID)
	if err != nil {
		return false, fmt.Errorf("failed to reserve enrichment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rows > 0, nil
}

// List retrieves submissions with pagination and filtering
func (r *PostgresRepository) List(ctx context.Context, opts *ListOptions) ([]*Submission, int, error) {
	// Set defaults
	if opts == nil {
		opts = &ListOptions{}
	}
	if opts.Limit == 0 {
		opts.Limit = 50
	}
	if opts.OrderBy == "" {
		opts.OrderBy = "created_at"
	}
	if opts.Order == "" {
		opts.Order = "DESC"
	}

	// SECURITY FIX: Whitelist allowed sort columns
	allowedSorts := map[string]bool{
		"created_at":   true,
		"updated_at":   true,
		"company_name": true,
		"status":       true,
	}

	orderBy := opts.OrderBy
	if !allowedSorts[orderBy] {
		return nil, 0, fmt.Errorf("invalid orderBy value: %s", orderBy)
	}

	orderDir := strings.ToUpper(opts.Order)
	if orderDir != "ASC" && orderDir != "DESC" {
		return nil, 0, fmt.Errorf("invalid order value: %s", opts.Order)
	}

	// Build query with explicit columns to ensure model/schema alignment
	selectCols := `id, company_name, cnpj, company_website, company_industry, company_size, company_location,
		contact_name, contact_email, contact_phone, contact_position,
		target_market, annual_revenue_min, annual_revenue_max, funding_stage,
		business_challenge, additional_notes, linkedin_url, twitter_handle,
		status, user_id, created_at, updated_at, deleted_at`
	query := "SELECT " + selectCols + " FROM submissions WHERE deleted_at IS NULL"
	countQuery := "SELECT COUNT(*) FROM submissions WHERE deleted_at IS NULL"
	args := []interface{}{}
	argPos := 1

	// Add filters
	if opts.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		countQuery += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *opts.Status)
		argPos++
	}

	if opts.Email != nil {
		query += fmt.Sprintf(" AND contact_email = $%d", argPos)
		countQuery += fmt.Sprintf(" AND contact_email = $%d", argPos)
		args = append(args, *opts.Email)
		argPos++
	}

	if opts.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argPos)
		countQuery += fmt.Sprintf(" AND user_id = $%d", argPos)
		args = append(args, *opts.UserID)
		argPos++
	}

	// Get total count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("failed to count submissions: %w", err)
	}

	// Add sorting and pagination safely
	// distinct string formatting helps prevent parameter sniffing issues in some drivers
	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d", orderBy, orderDir, argPos, argPos+1)
	args = append(args, opts.Limit, opts.Offset)

	// Execute query
	var submissions []*Submission
	if err := r.db.SelectContext(ctx, &submissions, query, args...); err != nil {
		return nil, 0, fmt.Errorf("failed to list submissions: %w", err)
	}

	return submissions, total, nil
}

// Update updates a submission
func (r *PostgresRepository) Update(ctx context.Context, s *Submission) error {
	s.UpdatedAt = time.Now()

	query := `
		UPDATE submissions SET
			company_name = :company_name,
			cnpj = :cnpj,
			company_website = :company_website,
			company_industry = :company_industry,
			company_size = :company_size,
			company_location = :company_location,
			contact_name = :contact_name,
			contact_email = :contact_email,
			contact_phone = :contact_phone,
			contact_position = :contact_position,
			target_market = :target_market,
			annual_revenue_min = :annual_revenue_min,
			annual_revenue_max = :annual_revenue_max,
			funding_stage = :funding_stage,
			business_challenge = :business_challenge,
			additional_notes = :additional_notes,
			linkedin_url = :linkedin_url,
			twitter_handle = :twitter_handle,
			updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL
	`

	result, err := r.db.NamedExecContext(ctx, query, s)
	if err != nil {
		return fmt.Errorf("failed to update submission: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("submission not found")
	}

	return nil
}

// Delete soft-deletes a submission
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE submissions SET deleted_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("failed to delete submission: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("submission not found")
	}

	return nil
}

// ==================== ANALYTICS METHODS ====================

// GetTotalCount returns the total number of non-deleted submissions
func (r *PostgresRepository) GetTotalCount(ctx context.Context) (int, error) {
	query := `
		SELECT COUNT(*) FROM submissions
		WHERE deleted_at IS NULL
	`

	var count int
	if err := r.db.GetContext(ctx, &count, query); err != nil {
		return 0, fmt.Errorf("failed to get total submission count: %w", err)
	}

	return count, nil
}

// GetActiveCount returns the number of submissions that are still being worked on
// Active = analysis not yet sent to user (analysis.status != 'sent' or no analysis exists)
func (r *PostgresRepository) GetActiveCount(ctx context.Context) (int, error) {
	query := `
		SELECT COUNT(DISTINCT s.id)
		FROM submissions s
		LEFT JOIN analyses a ON s.id = a.submission_id
		WHERE s.deleted_at IS NULL
		AND (a.id IS NULL OR a.status != 'sent')
	`

	var count int
	if err := r.db.GetContext(ctx, &count, query); err != nil {
		return 0, fmt.Errorf("failed to get active submission count: %w", err)
	}

	return count, nil
}

// GetCompletedCount returns the number of submissions with analysis sent to user
// Completed = analysis.status = 'sent'
func (r *PostgresRepository) GetCompletedCount(ctx context.Context) (int, error) {
	query := `
		SELECT COUNT(DISTINCT s.id)
		FROM submissions s
		INNER JOIN analyses a ON s.id = a.submission_id
		WHERE s.deleted_at IS NULL
		AND a.status = 'sent'
	`

	var count int
	if err := r.db.GetContext(ctx, &count, query); err != nil {
		return 0, fmt.Errorf("failed to get completed submission count: %w", err)
	}

	return count, nil
}
