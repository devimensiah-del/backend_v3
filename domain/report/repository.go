package report

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repository defines data access methods for reports
type Repository interface {
	Create(ctx context.Context, report *Report) error
	Update(ctx context.Context, report *Report) error
	Upsert(ctx context.Context, report *Report) error
	GetByID(ctx context.Context, id string) (*Report, error)
	GetBySubmissionID(ctx context.Context, submissionID string) (*Report, error)
	// List returns lightweight summaries (excludes HTML pages for performance)
	List(ctx context.Context, limit, offset int) ([]*ReportSummary, error)
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

// Create inserts a new report record
func (r *PostgresRepository) Create(ctx context.Context, report *Report) error {
	query := `
		INSERT INTO reports (
			id, submission_id, analysis_id,
			cover_page, executive_summary, table_of_contents,
			divider_part1_page, pestel_pes_page, pestel_tel_page, porter_page, swot_page,
			divider_part2_page, tam_sam_som_page, blue_ocean_page,
			divider_part3_page, okr_page, growth_loops_page,
			divider_part4_page, scenarios_page, recommendations_page,
			bsc_page, benchmarking_page, financial_projections_page, growth_hacking_page,
			risk_assessment_page, roadmap_page, appendix_page,
			pdf_url, pdf_generated_at, pdf_generation_status,
			status, error_message, generation_time_ms, total_pages,
			created_at, updated_at, completed_at
		) VALUES (
			:id, :submission_id, :analysis_id,
			:cover_page, :executive_summary, :table_of_contents,
			:divider_part1_page, :pestel_pes_page, :pestel_tel_page, :porter_page, :swot_page,
			:divider_part2_page, :tam_sam_som_page, :blue_ocean_page,
			:divider_part3_page, :okr_page, :growth_loops_page,
			:divider_part4_page, :scenarios_page, :recommendations_page,
			:bsc_page, :benchmarking_page, :financial_projections_page, :growth_hacking_page,
			:risk_assessment_page, :roadmap_page, :appendix_page,
			:pdf_url, :pdf_generated_at, :pdf_generation_status,
			:status, :error_message, :generation_time_ms, :total_pages,
			:created_at, :updated_at, :completed_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, report)
	if err != nil {
		return fmt.Errorf("failed to create report: %w", err)
	}

	return nil
}

// Upsert creates or updates a report (idempotent operation)
// Uses submission_id as the conflict key to ensure one-to-one relationship
func (r *PostgresRepository) Upsert(ctx context.Context, report *Report) error {
	query := `
		INSERT INTO reports (
			id, submission_id, analysis_id,
			cover_page, executive_summary, table_of_contents,
			divider_part1_page, pestel_pes_page, pestel_tel_page, porter_page, swot_page,
			divider_part2_page, tam_sam_som_page, blue_ocean_page,
			divider_part3_page, okr_page, growth_loops_page,
			divider_part4_page, scenarios_page, recommendations_page,
			bsc_page, benchmarking_page, financial_projections_page, growth_hacking_page,
			risk_assessment_page, roadmap_page, appendix_page,
			pdf_url, pdf_generated_at, pdf_generation_status,
			status, error_message, generation_time_ms, total_pages,
			created_at, updated_at, completed_at
		) VALUES (
			:id, :submission_id, :analysis_id,
			:cover_page, :executive_summary, :table_of_contents,
			:divider_part1_page, :pestel_pes_page, :pestel_tel_page, :porter_page, :swot_page,
			:divider_part2_page, :tam_sam_som_page, :blue_ocean_page,
			:divider_part3_page, :okr_page, :growth_loops_page,
			:divider_part4_page, :scenarios_page, :recommendations_page,
			:bsc_page, :benchmarking_page, :financial_projections_page, :growth_hacking_page,
			:risk_assessment_page, :roadmap_page, :appendix_page,
			:pdf_url, :pdf_generated_at, :pdf_generation_status,
			:status, :error_message, :generation_time_ms, :total_pages,
			:created_at, :updated_at, :completed_at
		)
		ON CONFLICT (submission_id)
		DO UPDATE SET
			analysis_id = EXCLUDED.analysis_id,
			cover_page = EXCLUDED.cover_page,
			executive_summary = EXCLUDED.executive_summary,
			table_of_contents = EXCLUDED.table_of_contents,
			divider_part1_page = EXCLUDED.divider_part1_page,
			pestel_pes_page = EXCLUDED.pestel_pes_page,
			pestel_tel_page = EXCLUDED.pestel_tel_page,
			porter_page = EXCLUDED.porter_page,
			swot_page = EXCLUDED.swot_page,
			divider_part2_page = EXCLUDED.divider_part2_page,
			tam_sam_som_page = EXCLUDED.tam_sam_som_page,
			blue_ocean_page = EXCLUDED.blue_ocean_page,
			divider_part3_page = EXCLUDED.divider_part3_page,
			okr_page = EXCLUDED.okr_page,
			growth_loops_page = EXCLUDED.growth_loops_page,
			divider_part4_page = EXCLUDED.divider_part4_page,
			scenarios_page = EXCLUDED.scenarios_page,
			recommendations_page = EXCLUDED.recommendations_page,
			bsc_page = EXCLUDED.bsc_page,
			benchmarking_page = EXCLUDED.benchmarking_page,
			financial_projections_page = EXCLUDED.financial_projections_page,
			growth_hacking_page = EXCLUDED.growth_hacking_page,
			risk_assessment_page = EXCLUDED.risk_assessment_page,
			roadmap_page = EXCLUDED.roadmap_page,
			appendix_page = EXCLUDED.appendix_page,
			pdf_url = EXCLUDED.pdf_url,
			pdf_generated_at = EXCLUDED.pdf_generated_at,
			pdf_generation_status = EXCLUDED.pdf_generation_status,
			status = EXCLUDED.status,
			error_message = EXCLUDED.error_message,
			generation_time_ms = EXCLUDED.generation_time_ms,
			total_pages = EXCLUDED.total_pages,
			updated_at = EXCLUDED.updated_at,
			completed_at = EXCLUDED.completed_at
	`

	_, err := r.db.NamedExecContext(ctx, query, report)
	if err != nil {
		return fmt.Errorf("failed to upsert report: %w", err)
	}

	return nil
}

// Update modifies an existing report record
func (r *PostgresRepository) Update(ctx context.Context, report *Report) error {
	query := `
		UPDATE reports SET
			cover_page = :cover_page,
			executive_summary = :executive_summary,
			table_of_contents = :table_of_contents,
			divider_part1_page = :divider_part1_page,
			pestel_pes_page = :pestel_pes_page,
			pestel_tel_page = :pestel_tel_page,
			porter_page = :porter_page,
			swot_page = :swot_page,
			divider_part2_page = :divider_part2_page,
			tam_sam_som_page = :tam_sam_som_page,
			blue_ocean_page = :blue_ocean_page,
			divider_part3_page = :divider_part3_page,
			okr_page = :okr_page,
			growth_loops_page = :growth_loops_page,
			divider_part4_page = :divider_part4_page,
			scenarios_page = :scenarios_page,
			recommendations_page = :recommendations_page,
			bsc_page = :bsc_page,
			benchmarking_page = :benchmarking_page,
			financial_projections_page = :financial_projections_page,
			growth_hacking_page = :growth_hacking_page,
			risk_assessment_page = :risk_assessment_page,
			roadmap_page = :roadmap_page,
			appendix_page = :appendix_page,
			pdf_url = :pdf_url,
			pdf_generated_at = :pdf_generated_at,
			pdf_generation_status = :pdf_generation_status,
			status = :status,
			error_message = :error_message,
			generation_time_ms = :generation_time_ms,
			total_pages = :total_pages,
			updated_at = :updated_at,
			completed_at = :completed_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, report)
	if err != nil {
		return fmt.Errorf("failed to update report: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("report not found: %s", report.ID)
	}

	return nil
}

// GetByID retrieves a report by its ID
func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*Report, error) {
	query := `
		SELECT
			id, submission_id, analysis_id,
			cover_page, executive_summary, table_of_contents,
			divider_part1_page, pestel_pes_page, pestel_tel_page, porter_page, swot_page,
			divider_part2_page, tam_sam_som_page, blue_ocean_page,
			divider_part3_page, okr_page, growth_loops_page,
			divider_part4_page, scenarios_page, recommendations_page,
			bsc_page, benchmarking_page, financial_projections_page, growth_hacking_page,
			risk_assessment_page, roadmap_page, appendix_page,
			pdf_url, pdf_generated_at, pdf_generation_status,
			status, error_message, generation_time_ms, total_pages,
			created_at, updated_at, completed_at
		FROM reports
		WHERE id = $1
	`

	var report Report
	err := r.db.GetContext(ctx, &report, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("report not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get report: %w", err)
	}

	return &report, nil
}

// GetBySubmissionID retrieves a report by submission ID
func (r *PostgresRepository) GetBySubmissionID(ctx context.Context, submissionID string) (*Report, error) {
	query := `
		SELECT
			id, submission_id, analysis_id,
			cover_page, executive_summary, table_of_contents,
			divider_part1_page, pestel_pes_page, pestel_tel_page, porter_page, swot_page,
			divider_part2_page, tam_sam_som_page, blue_ocean_page,
			divider_part3_page, okr_page, growth_loops_page,
			divider_part4_page, scenarios_page, recommendations_page,
			bsc_page, benchmarking_page, financial_projections_page, growth_hacking_page,
			risk_assessment_page, roadmap_page, appendix_page,
			pdf_url, pdf_generated_at, pdf_generation_status,
			status, error_message, generation_time_ms, total_pages,
			created_at, updated_at, completed_at
		FROM reports
		WHERE submission_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var report Report
	err := r.db.GetContext(ctx, &report, query, submissionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("report not found for submission: %s", submissionID)
		}
		return nil, fmt.Errorf("failed to get report: %w", err)
	}

	return &report, nil
}

// List retrieves report summaries with pagination
// PERFORMANCE: Excludes 24 HTML page columns to avoid pulling ~100KB+ per report
// Use GetByID() when you need the full report with HTML pages
func (r *PostgresRepository) List(ctx context.Context, limit, offset int) ([]*ReportSummary, error) {
	query := `
		SELECT
			id, submission_id, analysis_id,
			pdf_url, pdf_generated_at, pdf_generation_status,
			status, error_message, generation_time_ms, total_pages,
			created_at, updated_at, completed_at
		FROM reports
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	var summaries []*ReportSummary
	err := r.db.SelectContext(ctx, &summaries, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list reports: %w", err)
	}

	return summaries, nil
}

// Delete removes a report record
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM reports WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete report: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("report not found: %s", id)
	}

	return nil
}
