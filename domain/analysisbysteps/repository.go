package analysisbysteps

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository handles database operations for analysis steps
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new Repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new analysis step
func (r *Repository) Create(ctx context.Context, step *AnalysisStep) error {
	query := `
		INSERT INTO analysis_steps_v2 (
			analysis_id, framework_code, step_number, ai_output, human_edited,
			visible, status, generated_at, approved_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW()
		)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowxContext(ctx, query,
		step.AnalysisID, step.FrameworkCode, step.StepNumber, step.AIOutput, step.HumanEdited,
		step.Visible, step.Status, step.GeneratedAt, step.ApprovedAt,
	).Scan(&step.ID, &step.CreatedAt, &step.UpdatedAt)
}

// GetByID retrieves an analysis step by ID
func (r *Repository) GetByID(ctx context.Context, id string) (*AnalysisStep, error) {
	var step AnalysisStep
	query := `SELECT * FROM analysis_steps_v2 WHERE id = $1`
	err := r.db.GetContext(ctx, &step, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis step by id: %w", err)
	}
	return &step, nil
}

// GetByAnalysisID retrieves all steps for an analysis, ordered by step number
func (r *Repository) GetByAnalysisID(ctx context.Context, analysisID string) ([]AnalysisStep, error) {
	var steps []AnalysisStep
	query := `SELECT * FROM analysis_steps_v2 WHERE analysis_id = $1 ORDER BY step_number`
	err := r.db.SelectContext(ctx, &steps, query, analysisID)
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis steps: %w", err)
	}
	return steps, nil
}

// GetByAnalysisAndFramework retrieves a step by analysis ID and framework code
func (r *Repository) GetByAnalysisAndFramework(ctx context.Context, analysisID, frameworkCode string) (*AnalysisStep, error) {
	var step AnalysisStep
	query := `SELECT * FROM analysis_steps_v2 WHERE analysis_id = $1 AND framework_code = $2`
	err := r.db.GetContext(ctx, &step, query, analysisID, frameworkCode)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis step: %w", err)
	}
	return &step, nil
}

// Update updates an existing analysis step
func (r *Repository) Update(ctx context.Context, step *AnalysisStep) error {
	query := `
		UPDATE analysis_steps_v2 SET
			ai_output = $2,
			human_edited = $3,
			visible = $4,
			status = $5,
			generated_at = $6,
			approved_at = $7,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`

	return r.db.QueryRowxContext(ctx, query,
		step.ID, step.AIOutput, step.HumanEdited, step.Visible,
		step.Status, step.GeneratedAt, step.ApprovedAt,
	).Scan(&step.UpdatedAt)
}

// Upsert creates or updates an analysis step based on analysis_id + framework_code
func (r *Repository) Upsert(ctx context.Context, step *AnalysisStep) error {
	query := `
		INSERT INTO analysis_steps_v2 (
			analysis_id, framework_code, step_number, ai_output, human_edited,
			visible, status, generated_at, approved_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW()
		)
		ON CONFLICT (analysis_id, framework_code) DO UPDATE SET
			ai_output = EXCLUDED.ai_output,
			human_edited = EXCLUDED.human_edited,
			visible = EXCLUDED.visible,
			status = EXCLUDED.status,
			generated_at = EXCLUDED.generated_at,
			approved_at = EXCLUDED.approved_at,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowxContext(ctx, query,
		step.AnalysisID, step.FrameworkCode, step.StepNumber, step.AIOutput, step.HumanEdited,
		step.Visible, step.Status, step.GeneratedAt, step.ApprovedAt,
	).Scan(&step.ID, &step.CreatedAt, &step.UpdatedAt)
}

// SetAIOutput updates the AI output and marks step as generated
func (r *Repository) SetAIOutput(ctx context.Context, id string, output string) error {
	now := time.Now()
	query := `
		UPDATE analysis_steps_v2 SET
			ai_output = $2,
			status = $3,
			generated_at = $4,
			updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, output, StatusGenerated, now)
	return err
}

// SetHumanEdited updates the human-edited content
func (r *Repository) SetHumanEdited(ctx context.Context, id string, edited string) error {
	query := `
		UPDATE analysis_steps_v2 SET
			human_edited = $2,
			updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, edited)
	return err
}

// Approve marks a step as approved
func (r *Repository) Approve(ctx context.Context, id string) error {
	now := time.Now()
	query := `
		UPDATE analysis_steps_v2 SET
			status = $2,
			approved_at = $3,
			updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, StatusApproved, now)
	return err
}

// SetVisibility updates the visibility of a step
func (r *Repository) SetVisibility(ctx context.Context, id string, visible bool) error {
	query := `
		UPDATE analysis_steps_v2 SET
			visible = $2,
			updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, visible)
	return err
}
