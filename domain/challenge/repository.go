package challenge

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// Repository defines the interface for challenge data access
type Repository interface {
	Create(ctx context.Context, c *Challenge) error
	GetByID(ctx context.Context, id uuid.UUID) (*Challenge, error)
	ListByCompanyID(ctx context.Context, companyID uuid.UUID) ([]*Challenge, error)
	Update(ctx context.Context, c *Challenge) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new PostgreSQL repository
func NewRepository(db *sqlx.DB) Repository {
	return &PostgresRepository{db: db}
}

// Create inserts a new challenge into the database
func (r *PostgresRepository) Create(ctx context.Context, c *Challenge) error {
	query := `
		INSERT INTO challenges (
			id, company_id, challenge_category, challenge_type, business_challenge,
			created_at, updated_at
		) VALUES (
			:id, :company_id, :challenge_category, :challenge_type, :business_challenge,
			:created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, c)
	if err != nil {
		// Log the actual SQL error for debugging
		log.Error().
			Err(err).
			Str("id", c.ID.String()).
			Str("company_id", c.CompanyID.String()).
			Str("category", string(c.ChallengeCategory)).
			Str("type", string(c.ChallengeType)).
			Str("business_challenge", c.BusinessChallenge[:min(100, len(c.BusinessChallenge))]).
			Msg("SQL ERROR: Failed to insert challenge")
		return NewRepositoryError("create", err)
	}

	return nil
}

// GetByID retrieves a challenge by ID
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Challenge, error) {
	query := `
		SELECT
			id, company_id, challenge_category, challenge_type, business_challenge,
			created_at, updated_at, deleted_at
		FROM challenges
		WHERE id = $1 AND deleted_at IS NULL
	`

	var challenge Challenge
	if err := r.db.GetContext(ctx, &challenge, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, NewRepositoryError("get_by_id", err)
	}

	return &challenge, nil
}

// ListByCompanyID retrieves all challenges for a company
func (r *PostgresRepository) ListByCompanyID(ctx context.Context, companyID uuid.UUID) ([]*Challenge, error) {
	query := `
		SELECT
			id, company_id, challenge_category, challenge_type, business_challenge,
			created_at, updated_at, deleted_at
		FROM challenges
		WHERE company_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	var challenges []*Challenge
	if err := r.db.SelectContext(ctx, &challenges, query, companyID); err != nil {
		return nil, NewRepositoryError("list_by_company", err)
	}

	return challenges, nil
}

// Update updates a challenge
func (r *PostgresRepository) Update(ctx context.Context, c *Challenge) error {
	c.UpdatedAt = time.Now()

	query := `
		UPDATE challenges SET
			challenge_category = :challenge_category,
			challenge_type = :challenge_type,
			business_challenge = :business_challenge,
			updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL
	`

	result, err := r.db.NamedExecContext(ctx, query, c)
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

// Delete soft-deletes a challenge
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE challenges SET deleted_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, now, id)
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
