//go:build integration

package report_test

import (
	"context"
	"os"
	"testing"
	"time"

	"backend_v3/domain/report"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	db, err := sqlx.Connect("postgres", dsn)
	require.NoError(t, err, "failed to connect to test database")
	return db
}

func withTxRollback(t *testing.T, db *sqlx.DB, fn func(tx *sqlx.Tx)) {
	t.Helper()
	tx, err := db.Beginx()
	require.NoError(t, err)
	defer tx.Rollback()
	fn(tx)
}

// createTestSubmission creates a submission for FK constraint
func createTestSubmission(t *testing.T, tx *sqlx.Tx) string {
	t.Helper()
	subID := uuid.New().String()
	_, err := tx.Exec(`
		INSERT INTO submissions (id, company_name, contact_name, contact_email, business_challenge, status, created_at, updated_at)
		VALUES ($1, 'Test Company', 'Test User', 'test@test.com', 'Test challenge', 'received', NOW(), NOW())
	`, subID)
	require.NoError(t, err)
	return subID
}

// createTestEnrichment creates an enrichment for FK constraint
func createTestEnrichment(t *testing.T, tx *sqlx.Tx, submissionID string) string {
	t.Helper()
	enrichID := uuid.New().String()
	_, err := tx.Exec(`
		INSERT INTO enrichments (id, submission_id, status, progress, current_step, is_locked, sources_status, sources_used, data, created_at, updated_at)
		VALUES ($1, $2, 'approved', 100, 'Done', FALSE, '{}', '{}', '{}', NOW(), NOW())
	`, enrichID, submissionID)
	require.NoError(t, err)
	return enrichID
}

// createTestAnalysis creates an analysis for FK constraint
func createTestAnalysis(t *testing.T, tx *sqlx.Tx, submissionID, enrichmentID string) string {
	t.Helper()
	analysisID := uuid.New().String()
	_, err := tx.Exec(`
		INSERT INTO analyses (id, submission_id, enrichment_id, status,
			swot, pestel, porter, okrs, tam_sam_som, benchmarking, blue_ocean, growth_hacking, scenarios, bsc, decision_matrix, synthesis,
			created_at, updated_at)
		VALUES ($1, $2, $3, 'approved',
			'{}', '{}', '{}', '{}', '{}', '{}', '{}', '{}', '{}', '{}', '{}', '{}',
			NOW(), NOW())
	`, analysisID, submissionID, enrichmentID)
	require.NoError(t, err)
	return analysisID
}

// txRepository wraps transaction for Repository interface
type txRepository struct {
	tx *sqlx.Tx
}

func (r *txRepository) Create(ctx context.Context, rep *report.Report) error {
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
			$1, $2, $3,
			$4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14,
			$15, $16, $17,
			$18, $19, $20,
			$21, $22, $23, $24,
			$25, $26, $27,
			$28, $29, $30,
			$31, $32, $33, $34,
			$35, $36, $37
		)
	`

	_, err := r.tx.ExecContext(ctx, query,
		rep.ID, rep.SubmissionID, rep.AnalysisID,
		rep.CoverPage, rep.ExecutiveSummary, rep.TableOfContents,
		rep.DividerPart1Page, rep.PESTELPesPage, rep.PESTELTelPage, rep.PorterPage, rep.SWOTPage,
		rep.DividerPart2Page, rep.TamSamSomPage, rep.BlueOceanPage,
		rep.DividerPart3Page, rep.OKRPage, rep.GrowthLoopsPage,
		rep.DividerPart4Page, rep.ScenariosPage, rep.RecommendationsPage,
		rep.BSCPage, rep.BenchmarkingPage, rep.FinancialProjectionsPage, rep.GrowthHackingPage,
		rep.RiskAssessmentPage, rep.RoadmapPage, rep.AppendixPage,
		rep.PDFURL, rep.PDFGeneratedAt, rep.PDFGenerationStatus,
		rep.Status, rep.ErrorMessage, rep.GenerationTimeMs, rep.TotalPages,
		rep.CreatedAt, rep.UpdatedAt, rep.CompletedAt,
	)
	return err
}

func (r *txRepository) GetByID(ctx context.Context, id string) (*report.Report, error) {
	var rep report.Report
	err := r.tx.GetContext(ctx, &rep, `
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
		FROM reports WHERE id = $1
	`, id)
	return &rep, err
}

func (r *txRepository) GetBySubmissionID(ctx context.Context, submissionID string) (*report.Report, error) {
	var rep report.Report
	err := r.tx.GetContext(ctx, &rep, `
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
		FROM reports WHERE submission_id = $1
		ORDER BY created_at DESC LIMIT 1
	`, submissionID)
	return &rep, err
}

func (r *txRepository) Update(ctx context.Context, rep *report.Report) error {
	result, err := r.tx.ExecContext(ctx, `
		UPDATE reports SET
			pdf_url = $1, pdf_generated_at = $2, pdf_generation_status = $3,
			status = $4, error_message = $5, generation_time_ms = $6,
			updated_at = $7, completed_at = $8
		WHERE id = $9
	`,
		rep.PDFURL, rep.PDFGeneratedAt, rep.PDFGenerationStatus,
		rep.Status, rep.ErrorMessage, rep.GenerationTimeMs,
		time.Now(), rep.CompletedAt, rep.ID,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return assert.AnError
	}
	return nil
}

func (r *txRepository) Upsert(ctx context.Context, rep *report.Report) error {
	// For tests, use ON CONFLICT DO UPDATE
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
			$1, $2, $3,
			$4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14,
			$15, $16, $17,
			$18, $19, $20,
			$21, $22, $23, $24,
			$25, $26, $27,
			$28, $29, $30,
			$31, $32, $33, $34,
			$35, $36, $37
		)
		ON CONFLICT (submission_id) DO UPDATE SET
			analysis_id = EXCLUDED.analysis_id,
			pdf_url = EXCLUDED.pdf_url,
			pdf_generated_at = EXCLUDED.pdf_generated_at,
			pdf_generation_status = EXCLUDED.pdf_generation_status,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at,
			completed_at = EXCLUDED.completed_at
	`

	_, err := r.tx.ExecContext(ctx, query,
		rep.ID, rep.SubmissionID, rep.AnalysisID,
		rep.CoverPage, rep.ExecutiveSummary, rep.TableOfContents,
		rep.DividerPart1Page, rep.PESTELPesPage, rep.PESTELTelPage, rep.PorterPage, rep.SWOTPage,
		rep.DividerPart2Page, rep.TamSamSomPage, rep.BlueOceanPage,
		rep.DividerPart3Page, rep.OKRPage, rep.GrowthLoopsPage,
		rep.DividerPart4Page, rep.ScenariosPage, rep.RecommendationsPage,
		rep.BSCPage, rep.BenchmarkingPage, rep.FinancialProjectionsPage, rep.GrowthHackingPage,
		rep.RiskAssessmentPage, rep.RoadmapPage, rep.AppendixPage,
		rep.PDFURL, rep.PDFGeneratedAt, rep.PDFGenerationStatus,
		rep.Status, rep.ErrorMessage, rep.GenerationTimeMs, rep.TotalPages,
		rep.CreatedAt, rep.UpdatedAt, rep.CompletedAt,
	)
	return err
}

// =============================================================================
// INTEGRATION TESTS
// =============================================================================

func TestIntegration_CreateReport_SavesToDB(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		// Create FK dependencies
		subID := createTestSubmission(t, tx)
		enrichID := createTestEnrichment(t, tx, subID)
		analysisID := createTestAnalysis(t, tx, subID, enrichID)

		// Create report
		now := time.Now()
		rep := &report.Report{
			ID:                  uuid.New().String(),
			SubmissionID:        subID,
			AnalysisID:          analysisID,
			Status:              "pending",
			PDFGenerationStatus: "pending",
			TotalPages:          24,
			CreatedAt:           now,
			UpdatedAt:           now,
		}

		err := repo.Create(ctx, rep)
		require.NoError(t, err, "Create should succeed")

		// Verify it's in DB
		saved, err := repo.GetByID(ctx, rep.ID)
		require.NoError(t, err)

		assert.Equal(t, rep.ID, saved.ID)
		assert.Equal(t, subID, saved.SubmissionID)
		assert.Equal(t, analysisID, saved.AnalysisID)
		assert.Equal(t, "pending", saved.Status)
		assert.Equal(t, 24, saved.TotalPages)
	})
}

func TestIntegration_GetBySubmissionID_ReturnsReport(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		enrichID := createTestEnrichment(t, tx, subID)
		analysisID := createTestAnalysis(t, tx, subID, enrichID)

		now := time.Now()
		rep := &report.Report{
			ID:                  uuid.New().String(),
			SubmissionID:        subID,
			AnalysisID:          analysisID,
			Status:              "pending",
			PDFGenerationStatus: "pending",
			TotalPages:          24,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		require.NoError(t, repo.Create(ctx, rep))

		// Query by submission ID
		found, err := repo.GetBySubmissionID(ctx, subID)
		require.NoError(t, err)
		assert.Equal(t, rep.ID, found.ID)
	})
}

func TestIntegration_HTMLPagesSaveAndLoad(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		enrichID := createTestEnrichment(t, tx, subID)
		analysisID := createTestAnalysis(t, tx, subID, enrichID)

		now := time.Now()
		rep := &report.Report{
			ID:                  uuid.New().String(),
			SubmissionID:        subID,
			AnalysisID:          analysisID,
			Status:              "completed",
			PDFGenerationStatus: "completed",
			TotalPages:          24,
			// Sample HTML content for each page
			CoverPage:                "<html><body>Cover Page</body></html>",
			ExecutiveSummary:         "<html><body>Executive Summary</body></html>",
			TableOfContents:          "<html><body>TOC</body></html>",
			DividerPart1Page:         "<html><body>Part 1</body></html>",
			PESTELPesPage:            "<html><body>PESTEL PES</body></html>",
			PESTELTelPage:            "<html><body>PESTEL TEL</body></html>",
			PorterPage:               "<html><body>Porter</body></html>",
			SWOTPage:                 "<html><body>SWOT</body></html>",
			DividerPart2Page:         "<html><body>Part 2</body></html>",
			TamSamSomPage:            "<html><body>TAM SAM SOM</body></html>",
			BlueOceanPage:            "<html><body>Blue Ocean</body></html>",
			DividerPart3Page:         "<html><body>Part 3</body></html>",
			OKRPage:                  "<html><body>OKRs</body></html>",
			GrowthLoopsPage:          "<html><body>Growth Loops</body></html>",
			DividerPart4Page:         "<html><body>Part 4</body></html>",
			ScenariosPage:            "<html><body>Scenarios</body></html>",
			RecommendationsPage:      "<html><body>Recommendations</body></html>",
			BSCPage:                  "<html><body>BSC</body></html>",
			BenchmarkingPage:         "<html><body>Benchmarking</body></html>",
			FinancialProjectionsPage: "<html><body>Financial Projections</body></html>",
			GrowthHackingPage:        "<html><body>Growth Hacking</body></html>",
			RiskAssessmentPage:       "<html><body>Risk Assessment</body></html>",
			RoadmapPage:              "<html><body>Roadmap</body></html>",
			AppendixPage:             "<html><body>Appendix</body></html>",
			CreatedAt:                now,
			UpdatedAt:                now,
		}
		require.NoError(t, repo.Create(ctx, rep))

		// Retrieve and verify HTML pages
		saved, err := repo.GetByID(ctx, rep.ID)
		require.NoError(t, err)

		assert.Equal(t, "<html><body>Cover Page</body></html>", saved.CoverPage)
		assert.Equal(t, "<html><body>Executive Summary</body></html>", saved.ExecutiveSummary)
		assert.Equal(t, "<html><body>SWOT</body></html>", saved.SWOTPage)
		assert.Equal(t, "<html><body>OKRs</body></html>", saved.OKRPage)
		assert.Equal(t, "<html><body>Appendix</body></html>", saved.AppendixPage)
	})
}

func TestIntegration_PDFGeneration_UpdatesStatus(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		enrichID := createTestEnrichment(t, tx, subID)
		analysisID := createTestAnalysis(t, tx, subID, enrichID)

		now := time.Now()
		rep := &report.Report{
			ID:                  uuid.New().String(),
			SubmissionID:        subID,
			AnalysisID:          analysisID,
			Status:              "pending",
			PDFGenerationStatus: "pending",
			TotalPages:          24,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		require.NoError(t, repo.Create(ctx, rep))

		// Simulate PDF generation completion
		pdfTime := time.Now()
		rep.PDFURL = "https://storage.example.com/reports/test-report.pdf"
		rep.PDFGeneratedAt = &pdfTime
		rep.PDFGenerationStatus = "completed"
		rep.Status = "completed"
		rep.GenerationTimeMs = 5000
		rep.CompletedAt = &pdfTime

		err := repo.Update(ctx, rep)
		require.NoError(t, err)

		// Verify updates
		saved, err := repo.GetByID(ctx, rep.ID)
		require.NoError(t, err)

		assert.Equal(t, "completed", saved.Status)
		assert.Equal(t, "completed", saved.PDFGenerationStatus)
		assert.Equal(t, "https://storage.example.com/reports/test-report.pdf", saved.PDFURL)
		assert.NotNil(t, saved.PDFGeneratedAt)
		assert.NotNil(t, saved.CompletedAt)
		assert.Equal(t, int64(5000), saved.GenerationTimeMs)
	})
}

func TestIntegration_Upsert_UpdatesExisting(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		enrichID := createTestEnrichment(t, tx, subID)
		analysisID := createTestAnalysis(t, tx, subID, enrichID)

		now := time.Now()
		rep := &report.Report{
			ID:                  uuid.New().String(),
			SubmissionID:        subID,
			AnalysisID:          analysisID,
			Status:              "pending",
			PDFGenerationStatus: "pending",
			TotalPages:          24,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		require.NoError(t, repo.Create(ctx, rep))

		// Upsert with updated data (same submission_id)
		rep2 := &report.Report{
			ID:                  uuid.New().String(), // Different ID
			SubmissionID:        subID,               // Same submission
			AnalysisID:          analysisID,
			Status:              "completed",
			PDFGenerationStatus: "completed",
			PDFURL:              "https://storage.example.com/reports/v2.pdf",
			TotalPages:          24,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		err := repo.Upsert(ctx, rep2)
		require.NoError(t, err)

		// Only one report should exist for this submission
		saved, err := repo.GetBySubmissionID(ctx, subID)
		require.NoError(t, err)

		assert.Equal(t, "completed", saved.Status)
		assert.Equal(t, "https://storage.example.com/reports/v2.pdf", saved.PDFURL)
	})
}

func TestIntegration_GetPDFURL_ReturnsCorrectValue(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		enrichID := createTestEnrichment(t, tx, subID)
		analysisID := createTestAnalysis(t, tx, subID, enrichID)

		now := time.Now()
		expectedURL := "https://storage.example.com/reports/test.pdf"
		rep := &report.Report{
			ID:                  uuid.New().String(),
			SubmissionID:        subID,
			AnalysisID:          analysisID,
			Status:              "completed",
			PDFGenerationStatus: "completed",
			PDFURL:              expectedURL,
			TotalPages:          24,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		require.NoError(t, repo.Create(ctx, rep))

		saved, err := repo.GetByID(ctx, rep.ID)
		require.NoError(t, err)

		// Test GetPDFURL method (implements analysis.ReportSummary interface)
		assert.Equal(t, expectedURL, saved.GetPDFURL())
	})
}
