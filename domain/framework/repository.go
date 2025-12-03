package framework

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for framework data access
type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Framework, error)
	GetByCode(ctx context.Context, code string) (*Framework, error)
	List(ctx context.Context, activeOnly bool) ([]*Framework, error)
	ListByCategory(ctx context.Context, category string) ([]*Framework, error)
	Create(ctx context.Context, f *Framework) error
	Update(ctx context.Context, f *Framework) error
	Delete(ctx context.Context, id uuid.UUID) error // Soft delete via is_active = false
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new PostgreSQL repository
func NewRepository(db *sqlx.DB) Repository {
	return &PostgresRepository{db: db}
}

// GetByID retrieves a framework by ID
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Framework, error) {
	query := `
		SELECT
			id, code, name, name_pt, description, description_pt,
			category, layer_order, is_active, requires_enrichment,
			timeout_seconds, prompt_template, output_schema, preferred_model, temperature,
			depends_on, created_at, updated_at
		FROM frameworks
		WHERE id = $1
	`

	var framework Framework
	if err := r.db.GetContext(ctx, &framework, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("framework not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get framework: %w", err)
	}

	return &framework, nil
}

// GetByCode retrieves a framework by its unique code
func (r *PostgresRepository) GetByCode(ctx context.Context, code string) (*Framework, error) {
	query := `
		SELECT
			id, code, name, name_pt, description, description_pt,
			category, layer_order, is_active, requires_enrichment,
			timeout_seconds, prompt_template, output_schema, preferred_model, temperature,
			depends_on, created_at, updated_at
		FROM frameworks
		WHERE code = $1
	`

	var framework Framework
	if err := r.db.GetContext(ctx, &framework, query, code); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("framework with code '%s' not found: %w", code, err)
		}
		return nil, fmt.Errorf("failed to get framework by code: %w", err)
	}

	return &framework, nil
}

// List retrieves all frameworks, optionally filtering to active only
func (r *PostgresRepository) List(ctx context.Context, activeOnly bool) ([]*Framework, error) {
	query := `
		SELECT
			id, code, name, name_pt, description, description_pt,
			category, layer_order, is_active, requires_enrichment,
			timeout_seconds, prompt_template, output_schema, preferred_model, temperature,
			depends_on, created_at, updated_at
		FROM frameworks
	`

	if activeOnly {
		query += " WHERE is_active = true"
	}

	query += " ORDER BY category, layer_order, code"

	var frameworks []*Framework
	if err := r.db.SelectContext(ctx, &frameworks, query); err != nil {
		return nil, fmt.Errorf("failed to list frameworks: %w", err)
	}

	return frameworks, nil
}

// ListByCategory retrieves all frameworks in a specific category
func (r *PostgresRepository) ListByCategory(ctx context.Context, category string) ([]*Framework, error) {
	query := `
		SELECT
			id, code, name, name_pt, description, description_pt,
			category, layer_order, is_active, requires_enrichment,
			timeout_seconds, prompt_template, output_schema, preferred_model, temperature,
			depends_on, created_at, updated_at
		FROM frameworks
		WHERE category = $1 AND is_active = true
		ORDER BY layer_order, code
	`

	var frameworks []*Framework
	if err := r.db.SelectContext(ctx, &frameworks, query, category); err != nil {
		return nil, fmt.Errorf("failed to list frameworks by category: %w", err)
	}

	return frameworks, nil
}

// Create inserts a new framework into the database
func (r *PostgresRepository) Create(ctx context.Context, f *Framework) error {
	query := `
		INSERT INTO frameworks (
			id, code, name, name_pt, description, description_pt,
			category, layer_order, is_active, requires_enrichment,
			timeout_seconds, prompt_template, output_schema, preferred_model, temperature,
			depends_on, created_at, updated_at
		) VALUES (
			:id, :code, :name, :name_pt, :description, :description_pt,
			:category, :layer_order, :is_active, :requires_enrichment,
			:timeout_seconds, :prompt_template, :output_schema, :preferred_model, :temperature,
			:depends_on, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, f)
	if err != nil {
		return fmt.Errorf("failed to insert framework: %w", err)
	}

	return nil
}

// Update updates an existing framework
func (r *PostgresRepository) Update(ctx context.Context, f *Framework) error {
	query := `
		UPDATE frameworks SET
			code = :code,
			name = :name,
			name_pt = :name_pt,
			description = :description,
			description_pt = :description_pt,
			category = :category,
			layer_order = :layer_order,
			is_active = :is_active,
			requires_enrichment = :requires_enrichment,
			timeout_seconds = :timeout_seconds,
			prompt_template = :prompt_template,
			output_schema = :output_schema,
			preferred_model = :preferred_model,
			temperature = :temperature,
			depends_on = :depends_on,
			updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, f)
	if err != nil {
		return fmt.Errorf("failed to update framework: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("framework not found")
	}

	return nil
}

// Delete performs a soft delete by setting is_active to false
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE frameworks
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete framework: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("framework not found")
	}

	return nil
}
