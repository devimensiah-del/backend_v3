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

// Repository defines the interface for submission data access.
// This allows for easy testing and swapping implementations.
type Repository interface {
	Create(ctx context.Context, s *Submission) error
	GetByID(ctx context.Context, id uuid.UUID) (*Submission, error)
	GetByEmail(ctx context.Context, email string) ([]*Submission, error)
	List(ctx context.Context, opts *ListOptions) ([]*Submission, int, error)
	Update(ctx context.Context, s *Submission) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Company lookup
	GetByCompanyID(ctx context.Context, companyID uuid.UUID) (*Submission, error)

	// Anonymous submission linking (for signup auto-link)
	GetAnonymousByEmail(ctx context.Context, email string) ([]*Submission, error)
	UpdateUserID(ctx context.Context, submissionID, userID uuid.UUID) error

	// Transaction support
	BeginTx(ctx context.Context) (Repository, error)
	Commit() error
	Rollback() error
}

// PostgresRepository implements Repository using PostgreSQL.
// This is the production implementation.
type PostgresRepository struct {
	db *sqlx.DB
	tx *sqlx.Tx // nil unless inside a transaction
}

// NewRepository creates a new PostgreSQL repository
func NewRepository(db *sqlx.DB) Repository {
	return &PostgresRepository{db: db}
}

// Create inserts a new submission into the database.
func (r *PostgresRepository) Create(ctx context.Context, s *Submission) error {
	query := `
		INSERT INTO submissions (
			id, company_name, cnpj, company_website, company_industry, company_size, company_location,
			contact_name, contact_email, contact_phone, contact_position,
			target_market, annual_revenue_min, annual_revenue_max, funding_stage,
			additional_notes, linkedin_url, twitter_handle,
			user_id, created_at, updated_at
		) VALUES (
			:id, :company_name, :cnpj, :company_website, :company_industry, :company_size, :company_location,
			:contact_name, :contact_email, :contact_phone, :contact_position,
			:target_market, :annual_revenue_min, :annual_revenue_max, :funding_stage,
			:additional_notes, :linkedin_url, :twitter_handle,
			:user_id, :created_at, :updated_at
		)
	`

	_, err := r.execer().NamedExecContext(ctx, query, s)
	if err != nil {
		return NewRepositoryError("create", err)
	}

	return nil
}

// GetByID retrieves a submission by ID.
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Submission, error) {
	query := `
		SELECT
			id, company_name, cnpj, company_website, company_industry, company_size, company_location,
			contact_name, contact_email, contact_phone, contact_position,
			target_market, annual_revenue_min, annual_revenue_max, funding_stage,
			additional_notes, linkedin_url, twitter_handle,
			user_id, created_at, updated_at, deleted_at
		FROM submissions
		WHERE id = $1 AND deleted_at IS NULL
	`

	var submission Submission
	if err := r.execer().GetContext(ctx, &submission, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, NewRepositoryError("get_by_id", err)
	}

	return &submission, nil
}

// GetByEmail retrieves all submissions for an email address (case-insensitive).
func (r *PostgresRepository) GetByEmail(ctx context.Context, email string) ([]*Submission, error) {
	query := `
		SELECT
			id, company_name, cnpj, company_website, company_industry, company_size, company_location,
			contact_name, contact_email, contact_phone, contact_position,
			target_market, annual_revenue_min, annual_revenue_max, funding_stage,
			additional_notes, linkedin_url, twitter_handle,
			user_id, created_at, updated_at, deleted_at
		FROM submissions
		WHERE LOWER(contact_email) = LOWER($1) AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	var submissions []*Submission
	if err := r.execer().SelectContext(ctx, &submissions, query, email); err != nil {
		return nil, NewRepositoryError("get_by_email", err)
	}

	return submissions, nil
}

// GetByCompanyID retrieves the submission for a company (via company_submissions).
func (r *PostgresRepository) GetByCompanyID(ctx context.Context, companyID uuid.UUID) (*Submission, error) {
	query := `
		SELECT
			s.id, s.company_name, s.cnpj, s.company_website, s.company_industry, s.company_size, s.company_location,
			s.contact_name, s.contact_email, s.contact_phone, s.contact_position,
			s.target_market, s.annual_revenue_min, s.annual_revenue_max, s.funding_stage,
			s.additional_notes, s.linkedin_url, s.twitter_handle,
			s.user_id, s.created_at, s.updated_at, s.deleted_at
		FROM submissions s
		INNER JOIN company_submissions cs ON s.id = cs.submission_id
		WHERE cs.company_id = $1 AND s.deleted_at IS NULL
		ORDER BY cs.linked_at DESC
		LIMIT 1
	`

	var submission Submission
	if err := r.execer().GetContext(ctx, &submission, query, companyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, NewRepositoryError("get_by_company_id", err)
	}

	return &submission, nil
}

// List retrieves submissions with pagination and filtering.
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

	// Whitelist allowed sort columns (security)
	allowedSorts := map[string]bool{
		"created_at":   true,
		"updated_at":   true,
		"company_name": true,
	}

	orderBy := opts.OrderBy
	if !allowedSorts[orderBy] {
		return nil, 0, NewValidationError("order_by", "invalid sort column")
	}

	orderDir := strings.ToUpper(opts.Order)
	if orderDir != "ASC" && orderDir != "DESC" {
		return nil, 0, NewValidationError("order", "must be ASC or DESC")
	}

	// Build query
	selectCols := `id, company_name, cnpj, company_website, company_industry, company_size, company_location,
		contact_name, contact_email, contact_phone, contact_position,
		target_market, annual_revenue_min, annual_revenue_max, funding_stage,
		additional_notes, linkedin_url, twitter_handle,
		user_id, created_at, updated_at, deleted_at`
	query := "SELECT " + selectCols + " FROM submissions WHERE deleted_at IS NULL"
	countQuery := "SELECT COUNT(*) FROM submissions WHERE deleted_at IS NULL"
	args := []interface{}{}
	argPos := 1

	// Add filters
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
	if err := r.execer().GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, NewRepositoryError("list_count", err)
	}

	// Add sorting and pagination
	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d", orderBy, orderDir, argPos, argPos+1)
	args = append(args, opts.Limit, opts.Offset)

	// Execute query
	var submissions []*Submission
	if err := r.execer().SelectContext(ctx, &submissions, query, args...); err != nil {
		return nil, 0, NewRepositoryError("list", err)
	}

	return submissions, total, nil
}

// Update updates a submission.
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
			additional_notes = :additional_notes,
			linkedin_url = :linkedin_url,
			twitter_handle = :twitter_handle,
			updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL
	`

	result, err := r.execer().NamedExecContext(ctx, query, s)
	if err != nil {
		return NewRepositoryError("update", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return NewRepositoryError("update", err)
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete soft-deletes a submission.
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE submissions SET deleted_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	result, err := r.execer().ExecContext(ctx, query, now, id)
	if err != nil {
		return NewRepositoryError("delete", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return NewRepositoryError("delete", err)
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// ==================== ANONYMOUS SUBMISSION LINKING ====================

// GetAnonymousByEmail returns submissions with matching email and NULL user_id.
// Used during signup to find submissions to link to the new user.
func (r *PostgresRepository) GetAnonymousByEmail(ctx context.Context, email string) ([]*Submission, error) {
	query := `
		SELECT id, company_name, cnpj, company_website, company_industry, company_size, company_location,
			contact_name, contact_email, contact_phone, contact_position,
			target_market, annual_revenue_min, annual_revenue_max, funding_stage,
			additional_notes, linkedin_url, twitter_handle,
			user_id, created_at, updated_at, deleted_at
		FROM submissions
		WHERE LOWER(contact_email) = LOWER($1)
		AND user_id IS NULL
		AND deleted_at IS NULL
	`

	var submissions []*Submission
	if err := r.execer().SelectContext(ctx, &submissions, query, email); err != nil {
		return nil, NewRepositoryError("get_anonymous_by_email", err)
	}

	return submissions, nil
}

// UpdateUserID sets the user_id on a submission
// Used during signup to link anonymous submissions to the new user
// Only updates if user_id is currently NULL to prevent race conditions
func (r *PostgresRepository) UpdateUserID(ctx context.Context, submissionID, userID uuid.UUID) error {
	query := `UPDATE submissions SET user_id = $1, updated_at = NOW() WHERE id = $2 AND user_id IS NULL AND deleted_at IS NULL`

	result, err := r.execer().ExecContext(ctx, query, userID, submissionID)
	if err != nil {
		return NewRepositoryError("update_user_id", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return NewRepositoryError("update_user_id", err)
	}

	if rows == 0 {
		// Could be not found OR already linked - either way, no action needed
		return nil
	}

	return nil
}

// =============================================================================
// TRANSACTION SUPPORT
// =============================================================================

// execer returns the appropriate executor (tx if in transaction, db otherwise)
func (r *PostgresRepository) execer() interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// BeginTx starts a new transaction and returns a repository bound to it.
func (r *PostgresRepository) BeginTx(ctx context.Context) (Repository, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, NewRepositoryError("begin_tx", err)
	}
	return &PostgresRepository{db: r.db, tx: tx}, nil
}

// Commit commits the current transaction.
func (r *PostgresRepository) Commit() error {
	if r.tx == nil {
		return nil // no-op if not in transaction
	}
	if err := r.tx.Commit(); err != nil {
		return NewRepositoryError("commit", err)
	}
	r.tx = nil
	return nil
}

// Rollback aborts the current transaction.
func (r *PostgresRepository) Rollback() error {
	if r.tx == nil {
		return nil // no-op if not in transaction
	}
	if err := r.tx.Rollback(); err != nil {
		return NewRepositoryError("rollback", err)
	}
	r.tx = nil
	return nil
}
