package report_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"backend_v3/domain/report"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----------------------------------------------------------------------------
// TEST: Create Report with All 16 HTML Pages
// ----------------------------------------------------------------------------

func TestPostgresRepository_Create(t *testing.T) {
	// Setup mock database
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := report.NewPostgresRepository(sqlxDB)

	// Create test report with all 24 HTML pages
	now := time.Now()
	testReport := &report.Report{
		ID:           uuid.New().String(),
		SubmissionID: uuid.New().String(),
		AnalysisID:   uuid.New().String(),
		Status:       "completed",

		// All HTML pages (v2 structure)
		CoverPage:                "<html>Cover</html>",
		ExecutiveSummary:         "<html>Executive Summary</html>",
		TableOfContents:          "<html>TOC</html>",
		DividerPart1Page:         "<html>Divider1</html>",
		PESTELPesPage:            "<html>PESTEL PES</html>",
		PESTELTelPage:            "<html>PESTEL TEL</html>",
		PorterPage:               "<html>Porter</html>",
		SWOTPage:                 "<html>SWOT</html>",
		DividerPart2Page:         "<html>Divider2</html>",
		TamSamSomPage:            "<html>TAM-SAM-SOM</html>",
		BlueOceanPage:            "<html>Blue Ocean</html>",
		DividerPart3Page:         "<html>Divider3</html>",
		OKRPage:                  "<html>OKRs</html>",
		GrowthLoopsPage:          "<html>Growth Loops</html>",
		DividerPart4Page:         "<html>Divider4</html>",
		ScenariosPage:            "<html>Scenarios</html>",
		RecommendationsPage:      "<html>Recommendations</html>",
		BSCPage:                  "<html>BSC</html>",
		BenchmarkingPage:         "<html>Benchmarking</html>",
		FinancialProjectionsPage: "<html>Financial Projections</html>",
		GrowthHackingPage:        "<html>Growth Hacking</html>",
		RiskAssessmentPage:       "<html>Risk Assessment</html>",
		RoadmapPage:              "<html>Roadmap</html>",
		AppendixPage:             "<html>Appendix</html>",

		// PDF info
		PDFURL:              "https://storage.example.com/report.pdf",
		PDFGeneratedAt:      &now,
		PDFGenerationStatus: "completed",

		// Metadata
		TotalPages:       24,
		GenerationTimeMs: 5000,
		CreatedAt:        now,
		UpdatedAt:        now,
		CompletedAt:      &now,
	}

	// Expect INSERT query
	mock.ExpectExec(`INSERT INTO reports`).WillReturnResult(sqlmock.NewResult(1, 1))

	// Execute
	err = repo.Create(context.Background(), testReport)

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ----------------------------------------------------------------------------
// TEST: GetBySubmissionID - Single Report Per Submission
// ----------------------------------------------------------------------------

func TestPostgresRepository_GetBySubmissionID_Found(t *testing.T) {
	// Setup mock database
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := report.NewPostgresRepository(sqlxDB)

	// Test data
	submissionID := uuid.New().String()
	reportID := uuid.New().String()
	now := time.Now()

	// Mock rows with all columns
	rows := sqlmock.NewRows([]string{
		"id", "submission_id", "analysis_id",
		"cover_page", "executive_summary", "table_of_contents",
		"divider_part1_page", "pestel_pes_page", "pestel_tel_page", "porter_page", "swot_page",
		"divider_part2_page", "tam_sam_som_page", "blue_ocean_page",
		"divider_part3_page", "okr_page", "growth_loops_page",
		"divider_part4_page", "scenarios_page", "recommendations_page",
		"bsc_page", "benchmarking_page", "financial_projections_page", "growth_hacking_page",
		"risk_assessment_page", "roadmap_page", "appendix_page",
		"pdf_url", "pdf_generated_at", "pdf_generation_status",
		"status", "error_message", "generation_time_ms", "total_pages",
		"created_at", "updated_at", "completed_at",
	}).AddRow(
		reportID, submissionID, uuid.New().String(),
		"<html>Cover</html>", "<html>Exec</html>", "<html>TOC</html>",
		"<html>Div1</html>", "<html>PES</html>", "<html>TEL</html>", "<html>Porter</html>", "<html>SWOT</html>",
		"<html>Div2</html>", "<html>TAM</html>", "<html>Ocean</html>",
		"<html>Div3</html>", "<html>OKR</html>", "<html>GrowthLoops</html>",
		"<html>Div4</html>", "<html>Scenarios</html>", "<html>Recommendations</html>",
		"<html>BSC</html>", "<html>Bench</html>", "<html>Financial</html>", "<html>Growth</html>",
		"<html>Risk</html>", "<html>Roadmap</html>", "<html>Appendix</html>",
		"https://storage.example.com/report.pdf", now, "completed",
		"completed", "", 5000, 24,
		now, now, now,
	)

	// Expect SELECT query
	mock.ExpectQuery(`SELECT .+ FROM reports WHERE submission_id`).
		WithArgs(submissionID).
		WillReturnRows(rows)

	// Execute
	result, err := repo.GetBySubmissionID(context.Background(), submissionID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, reportID, result.ID)
	assert.Equal(t, submissionID, result.SubmissionID)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, 24, result.TotalPages)

	// Verify all HTML pages are populated
	assert.NotEmpty(t, result.CoverPage)
	assert.NotEmpty(t, result.ExecutiveSummary)
	assert.NotEmpty(t, result.TableOfContents)
	assert.NotEmpty(t, result.PESTELPesPage)
	assert.NotEmpty(t, result.PESTELTelPage)
	assert.NotEmpty(t, result.PorterPage)
	assert.NotEmpty(t, result.SWOTPage)
	assert.NotEmpty(t, result.TamSamSomPage)
	assert.NotEmpty(t, result.BlueOceanPage)
	assert.NotEmpty(t, result.OKRPage)
	assert.NotEmpty(t, result.GrowthLoopsPage)
	assert.NotEmpty(t, result.BSCPage)
	assert.NotEmpty(t, result.BenchmarkingPage)
	assert.NotEmpty(t, result.FinancialProjectionsPage)
	assert.NotEmpty(t, result.GrowthHackingPage)
	assert.NotEmpty(t, result.RiskAssessmentPage)
	assert.NotEmpty(t, result.RoadmapPage)
	assert.NotEmpty(t, result.AppendixPage)

	// Verify PDF metadata
	assert.Equal(t, "https://storage.example.com/report.pdf", result.PDFURL)
	assert.Equal(t, "completed", result.PDFGenerationStatus)
	assert.NotNil(t, result.PDFGeneratedAt)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_GetBySubmissionID_NotFound(t *testing.T) {
	// Setup mock database
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := report.NewPostgresRepository(sqlxDB)

	submissionID := uuid.New().String()

	// Expect SELECT query with no results
	mock.ExpectQuery(`SELECT .+ FROM reports WHERE submission_id`).
		WithArgs(submissionID).
		WillReturnError(sql.ErrNoRows)

	// Execute
	result, err := repo.GetBySubmissionID(context.Background(), submissionID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "report not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// ----------------------------------------------------------------------------
// TEST: GetByID
// ----------------------------------------------------------------------------

func TestPostgresRepository_GetByID_Found(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := report.NewPostgresRepository(sqlxDB)

	reportID := uuid.New().String()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "submission_id", "analysis_id",
		"cover_page", "executive_summary", "table_of_contents",
		"divider_part1_page", "pestel_pes_page", "pestel_tel_page", "porter_page", "swot_page",
		"divider_part2_page", "tam_sam_som_page", "blue_ocean_page",
		"divider_part3_page", "okr_page", "growth_loops_page",
		"divider_part4_page", "scenarios_page", "recommendations_page",
		"bsc_page", "benchmarking_page", "financial_projections_page", "growth_hacking_page",
		"risk_assessment_page", "roadmap_page", "appendix_page",
		"pdf_url", "pdf_generated_at", "pdf_generation_status",
		"status", "error_message", "generation_time_ms", "total_pages",
		"created_at", "updated_at", "completed_at",
	}).AddRow(
		reportID, uuid.New().String(), uuid.New().String(),
		"<html>Cover</html>", "<html>Exec</html>", "<html>TOC</html>",
		"<html>Div1</html>", "<html>PES</html>", "<html>TEL</html>", "<html>Porter</html>", "<html>SWOT</html>",
		"<html>Div2</html>", "<html>TAM</html>", "<html>Ocean</html>",
		"<html>Div3</html>", "<html>OKR</html>", "<html>GrowthLoops</html>",
		"<html>Div4</html>", "<html>Scenarios</html>", "<html>Recommendations</html>",
		"<html>BSC</html>", "<html>Bench</html>", "<html>Financial</html>", "<html>Growth</html>",
		"<html>Risk</html>", "<html>Roadmap</html>", "<html>Appendix</html>",
		"https://storage.example.com/report.pdf", now, "completed",
		"completed", "", 5000, 24,
		now, now, now,
	)

	mock.ExpectQuery(`SELECT .+ FROM reports WHERE id`).
		WithArgs(reportID).
		WillReturnRows(rows)

	result, err := repo.GetByID(context.Background(), reportID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, reportID, result.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := report.NewPostgresRepository(sqlxDB)

	reportID := uuid.New().String()

	mock.ExpectQuery(`SELECT .+ FROM reports WHERE id`).
		WithArgs(reportID).
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByID(context.Background(), reportID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "report not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ----------------------------------------------------------------------------
// TEST: Update Report
// ----------------------------------------------------------------------------

func TestPostgresRepository_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := report.NewPostgresRepository(sqlxDB)

	now := time.Now()
	testReport := &report.Report{
		ID:           uuid.New().String(),
		SubmissionID: uuid.New().String(),
		AnalysisID:   uuid.New().String(),
		Status:       "completed",

		CoverPage:                "<html>Updated Cover</html>",
		ExecutiveSummary:         "<html>Updated Exec</html>",
		TableOfContents:          "<html>Updated TOC</html>",
		DividerPart1Page:         "<html>Updated Div1</html>",
		PESTELPesPage:            "<html>Updated PES</html>",
		PESTELTelPage:            "<html>Updated TEL</html>",
		PorterPage:               "<html>Updated Porter</html>",
		SWOTPage:                 "<html>Updated SWOT</html>",
		DividerPart2Page:         "<html>Updated Div2</html>",
		TamSamSomPage:            "<html>Updated TAM</html>",
		BlueOceanPage:            "<html>Updated Ocean</html>",
		DividerPart3Page:         "<html>Updated Div3</html>",
		OKRPage:                  "<html>Updated OKR</html>",
		GrowthLoopsPage:          "<html>Updated Loops</html>",
		DividerPart4Page:         "<html>Updated Div4</html>",
		ScenariosPage:            "<html>Updated Scenarios</html>",
		RecommendationsPage:      "<html>Updated Recs</html>",
		BSCPage:                  "<html>Updated BSC</html>",
		BenchmarkingPage:         "<html>Updated Bench</html>",
		FinancialProjectionsPage: "<html>Updated Financial</html>",
		GrowthHackingPage:        "<html>Updated Growth</html>",
		RiskAssessmentPage:       "<html>Updated Risk</html>",
		RoadmapPage:              "<html>Updated Roadmap</html>",
		AppendixPage:             "<html>Updated Appendix</html>",

		PDFURL:              "https://storage.example.com/updated.pdf",
		PDFGeneratedAt:      &now,
		PDFGenerationStatus: "completed",

		TotalPages:       24,
		GenerationTimeMs: 6000,
		UpdatedAt:        now,
		CompletedAt:      &now,
	}

	mock.ExpectExec(`UPDATE reports SET`).WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Update(context.Background(), testReport)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_Update_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := report.NewPostgresRepository(sqlxDB)

	testReport := &report.Report{
		ID: uuid.New().String(),
	}

	mock.ExpectExec(`UPDATE reports SET`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Update(context.Background(), testReport)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "report not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ----------------------------------------------------------------------------
// TEST: List Reports with Pagination
// ----------------------------------------------------------------------------

func TestPostgresRepository_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := report.NewPostgresRepository(sqlxDB)

	now := time.Now()

	// List now returns lightweight ReportSummary (excludes 24 HTML page columns for performance)
	rows := sqlmock.NewRows([]string{
		"id", "submission_id", "analysis_id",
		"pdf_url", "pdf_generated_at", "pdf_generation_status",
		"status", "error_message", "generation_time_ms", "total_pages",
		"created_at", "updated_at", "completed_at",
	}).
		AddRow(uuid.New().String(), uuid.New().String(), uuid.New().String(),
			"https://storage.example.com/1.pdf", now, "completed",
			"completed", "", 5000, 24,
			now, now, now).
		AddRow(uuid.New().String(), uuid.New().String(), uuid.New().String(),
			"https://storage.example.com/2.pdf", now, "completed",
			"completed", "", 5000, 24,
			now, now, now)

	mock.ExpectQuery(`SELECT .+ FROM reports ORDER BY created_at DESC`).
		WithArgs(10, 0).
		WillReturnRows(rows)

	results, err := repo.List(context.Background(), 10, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "https://storage.example.com/1.pdf", results[0].PDFURL)
	assert.Equal(t, "completed", results[0].Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ----------------------------------------------------------------------------
// TEST: Delete Report
// ----------------------------------------------------------------------------

func TestPostgresRepository_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := report.NewPostgresRepository(sqlxDB)

	reportID := uuid.New().String()

	mock.ExpectExec(`DELETE FROM reports WHERE id`).
		WithArgs(reportID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Delete(context.Background(), reportID)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_Delete_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := report.NewPostgresRepository(sqlxDB)

	reportID := uuid.New().String()

	mock.ExpectExec(`DELETE FROM reports WHERE id`).
		WithArgs(reportID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Delete(context.Background(), reportID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "report not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ----------------------------------------------------------------------------
// TEST: Database Error Handling
// ----------------------------------------------------------------------------

func TestPostgresRepository_GetBySubmissionID_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := report.NewPostgresRepository(sqlxDB)

	submissionID := uuid.New().String()

	// Simulate database connection error
	mock.ExpectQuery(`SELECT .+ FROM reports WHERE submission_id`).
		WithArgs(submissionID).
		WillReturnError(sql.ErrConnDone)

	result, err := repo.GetBySubmissionID(context.Background(), submissionID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}
