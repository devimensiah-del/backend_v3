//go:build integration

package analysis_test

import (
	"context"
	"os"
	"testing"
	"time"

	"backend_v3/domain/analysis"

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
	// Note: business_challenge and status columns were dropped by v2_009
	// Challenge data now lives in challenges table
	_, err := tx.Exec(`
		INSERT INTO submissions (id, company_name, contact_name, contact_email, created_at, updated_at)
		VALUES ($1, 'Test Company', 'Test User', 'test@test.com', NOW(), NOW())
	`, subID)
	require.NoError(t, err)
	return subID
}

// createTestCompany creates a company for FK constraint
func createTestCompany(t *testing.T, tx *sqlx.Tx) string {
	t.Helper()
	companyID := uuid.New().String()
	_, err := tx.Exec(`
		INSERT INTO companies (id, name, enrichment_status, created_at, updated_at)
		VALUES ($1, 'Test Company', 'completed', NOW(), NOW())
	`, companyID)
	require.NoError(t, err)
	return companyID
}

// txRepository wraps transaction for Repository interface
type txRepository struct {
	tx *sqlx.Tx
}

func (r *txRepository) Create(ctx context.Context, a *analysis.Analysis) error {
	query := `
		INSERT INTO analyses (
			id, submission_id, company_id,
			swot, pestel, porter, okrs, tam_sam_som, benchmarking, blue_ocean, growth_hacking, scenarios, bsc, decision_matrix,
			synthesis, status, error_message, processing_time_ms,
			approved_at, approved_by, sent_at, sent_to, deleted_at,
			created_at, updated_at, completed_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18,
			$19, $20, $21, $22, $23,
			$24, $25, $26
		)
	`

	// Get JSONB data from FrameworkResults
	swot := a.FrameworkResults["swot"]
	pestel := a.FrameworkResults["pestel"]
	porter := a.FrameworkResults["porter"]
	okrs := a.FrameworkResults["okrs"]
	tamSamSom := a.FrameworkResults["tam_sam_som"]
	benchmarking := a.FrameworkResults["benchmarking"]
	blueOcean := a.FrameworkResults["blue_ocean"]
	growthHacking := a.FrameworkResults["growth_hacking"]
	scenarios := a.FrameworkResults["scenarios"]
	bsc := a.FrameworkResults["bsc"]
	decisionMatrix := a.FrameworkResults["decision_matrix"]
	synthesis := a.FrameworkResults["synthesis"]

	_, err := r.tx.ExecContext(ctx, query,
		a.ID, a.SubmissionID, a.CompanyID,
		swot, pestel, porter, okrs, tamSamSom, benchmarking, blueOcean, growthHacking, scenarios, bsc, decisionMatrix,
		synthesis, a.Status, a.ErrorMessage, a.ProcessingTimeMs,
		a.ApprovedAt, a.ApprovedBy, a.SentAt, a.SentTo, a.DeletedAt,
		a.CreatedAt, a.UpdatedAt, a.CompletedAt,
	)
	return err
}

func (r *txRepository) GetByID(ctx context.Context, id string) (*analysis.Analysis, error) {
	var a analysis.Analysis
	err := r.tx.GetContext(ctx, &a, `
		SELECT
			id, submission_id, company_id,
			swot, pestel, porter, okrs, tam_sam_som, benchmarking, blue_ocean, growth_hacking, scenarios, bsc, decision_matrix,
			synthesis, status, error_message, processing_time_ms,
			approved_at, approved_by, sent_at, sent_to, deleted_at,
			created_at, updated_at, completed_at
		FROM analyses WHERE id = $1 AND deleted_at IS NULL
	`, id)
	return &a, err
}

func (r *txRepository) GetBySubmissionID(ctx context.Context, submissionID string) (*analysis.Analysis, error) {
	var a analysis.Analysis
	err := r.tx.GetContext(ctx, &a, `
		SELECT
			id, submission_id, company_id,
			swot, pestel, porter, okrs, tam_sam_som, benchmarking, blue_ocean, growth_hacking, scenarios, bsc, decision_matrix,
			synthesis, status, error_message, processing_time_ms,
			approved_at, approved_by, sent_at, sent_to, deleted_at,
			created_at, updated_at, completed_at
		FROM analyses WHERE submission_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC LIMIT 1
	`, submissionID)
	return &a, err
}

func (r *txRepository) Update(ctx context.Context, a *analysis.Analysis) error {
	// Get JSONB data from FrameworkResults
	swot := a.FrameworkResults["swot"]
	pestel := a.FrameworkResults["pestel"]
	porter := a.FrameworkResults["porter"]
	okrs := a.FrameworkResults["okrs"]
	tamSamSom := a.FrameworkResults["tam_sam_som"]
	benchmarking := a.FrameworkResults["benchmarking"]
	blueOcean := a.FrameworkResults["blue_ocean"]
	growthHacking := a.FrameworkResults["growth_hacking"]
	scenarios := a.FrameworkResults["scenarios"]
	bsc := a.FrameworkResults["bsc"]
	decisionMatrix := a.FrameworkResults["decision_matrix"]
	synthesis := a.FrameworkResults["synthesis"]

	result, err := r.tx.ExecContext(ctx, `
		UPDATE analyses SET
			swot = $1, pestel = $2, porter = $3, okrs = $4, tam_sam_som = $5,
			benchmarking = $6, blue_ocean = $7, growth_hacking = $8, scenarios = $9, bsc = $10,
			decision_matrix = $11, synthesis = $12, status = $13, error_message = $14,
			processing_time_ms = $15, approved_at = $16, approved_by = $17, sent_at = $18,
			sent_to = $19, updated_at = $20, completed_at = $21
		WHERE id = $22 AND deleted_at IS NULL
	`,
		swot, pestel, porter, okrs, tamSamSom,
		benchmarking, blueOcean, growthHacking, scenarios, bsc,
		decisionMatrix, synthesis, a.Status, a.ErrorMessage,
		a.ProcessingTimeMs, a.ApprovedAt, a.ApprovedBy, a.SentAt,
		a.SentTo, time.Now(), a.CompletedAt, a.ID,
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

// =============================================================================
// INTEGRATION TESTS
// =============================================================================

func TestIntegration_CreateAnalysis_SavesToDB(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		// Create submission and company first (FK constraints)
		subID := createTestSubmission(t, tx)
		companyID := createTestCompany(t, tx)

		// Create analysis
		now := time.Now()
		a := &analysis.Analysis{
			ID:           uuid.New().String(),
			SubmissionID: subID,
			CompanyID:    &companyID,
			Status:       string(analysis.StatusPending),
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		err := repo.Create(ctx, a)
		require.NoError(t, err, "Create should succeed")

		// Verify it's in DB
		saved, err := repo.GetByID(ctx, a.ID)
		require.NoError(t, err)

		assert.Equal(t, a.ID, saved.ID)
		assert.Equal(t, subID, saved.SubmissionID)
		assert.NotNil(t, saved.CompanyID)
		assert.Equal(t, companyID, *saved.CompanyID)
		assert.Equal(t, string(analysis.StatusPending), saved.Status)
	})
}

func TestIntegration_GetBySubmissionID_ReturnsAnalysis(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		companyID := createTestCompany(t, tx)

		now := time.Now()
		a := &analysis.Analysis{
			ID:           uuid.New().String(),
			SubmissionID: subID,
			CompanyID:    &companyID,
			Status:       string(analysis.StatusPending),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		require.NoError(t, repo.Create(ctx, a))

		// Query by submission ID
		found, err := repo.GetBySubmissionID(ctx, subID)
		require.NoError(t, err)
		assert.Equal(t, a.ID, found.ID)
	})
}

func TestIntegration_StatusTransitions(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		companyID := createTestCompany(t, tx)

		now := time.Now()
		a := &analysis.Analysis{
			ID:           uuid.New().String(),
			SubmissionID: subID,
			CompanyID:    &companyID,
			Status:       string(analysis.StatusPending),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		require.NoError(t, repo.Create(ctx, a))

		// Initial state: pending
		assert.Equal(t, string(analysis.StatusPending), a.Status)

		// Complete
		a.Complete()
		require.NoError(t, repo.Update(ctx, a))

		saved, _ := repo.GetByID(ctx, a.ID)
		assert.Equal(t, string(analysis.StatusCompleted), saved.Status)
		assert.NotNil(t, saved.CompletedAt)

		// Approve
		approver := "admin-user"
		a.Approve(&approver)
		require.NoError(t, repo.Update(ctx, a))

		saved, _ = repo.GetByID(ctx, a.ID)
		assert.Equal(t, string(analysis.StatusApproved), saved.Status)
		assert.NotNil(t, saved.ApprovedAt)
		assert.Equal(t, "admin-user", *saved.ApprovedBy)

		// Send
		email := "user@example.com"
		a.Send(&email)
		require.NoError(t, repo.Update(ctx, a))

		saved, _ = repo.GetByID(ctx, a.ID)
		assert.Equal(t, string(analysis.StatusSent), saved.Status)
		assert.NotNil(t, saved.SentAt)
		assert.Equal(t, "user@example.com", *saved.SentTo)
	})
}

func TestIntegration_JSONB_SWOTSavesAndLoads(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		companyID := createTestCompany(t, tx)

		now := time.Now()
		a := &analysis.Analysis{
			ID:              uuid.New().String(),
			SubmissionID:    subID,
			CompanyID:       &companyID,
			Status:          string(analysis.StatusPending),
			FrameworkResults: make(map[string]json.RawMessage),
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// Use SetFramework to add SWOT data
		a.SetFramework("swot", &analysis.SWOTAnalysis{
			Strengths: []analysis.SWOTItem{
				{Content: "Strong brand", Confidence: "Alta", Source: "fato"},
				{Content: "Good team", Confidence: "Média", Source: "análise de mercado"},
			},
			Weaknesses: []analysis.SWOTItem{
				{Content: "Limited resources", Confidence: "Alta", Source: "fato"},
			},
			Opportunities: []analysis.SWOTItem{
				{Content: "Market growth", Confidence: "Média", Source: "estimativa"},
			},
			Threats: []analysis.SWOTItem{
				{Content: "Competition", Confidence: "Alta", Source: "análise de mercado"},
			},
			Summary: "Test SWOT summary",
		})

		require.NoError(t, repo.Create(ctx, a))

		// Retrieve and verify JSONB deserialization
		saved, err := repo.GetByID(ctx, a.ID)
		require.NoError(t, err)

		// Get SWOT data via GetFramework
		var swotResult analysis.SWOTAnalysis
		err = saved.GetFramework("swot", &swotResult)
		require.NoError(t, err)

		assert.Len(t, swotResult.Strengths, 2)
		assert.Equal(t, "Strong brand", swotResult.Strengths[0].Content)
		assert.Equal(t, "Alta", swotResult.Strengths[0].Confidence)
		assert.Equal(t, "fato", swotResult.Strengths[0].Source)
		assert.Equal(t, "Test SWOT summary", swotResult.Summary)
	})
}

func TestIntegration_JSONB_PorterSavesAndLoads(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		companyID := createTestCompany(t, tx)

		now := time.Now()
		a := &analysis.Analysis{
			ID:              uuid.New().String(),
			SubmissionID:    subID,
			CompanyID:       &companyID,
			Status:          string(analysis.StatusPending),
			FrameworkResults: make(map[string]json.RawMessage),
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// Use SetFramework to add Porter data
		a.SetFramework("porter", &analysis.PorterAnalysis{
			CompetitiveRivalry:                   "High competition in the market",
			SupplierPower:                        "Low supplier power",
			BuyerPower:                           "Moderate buyer power",
			ThreatNewEntrants:                    "High barriers to entry",
			ThreatSubstitutes:                    "Low substitute threat",
			PowerPartnershipsEcosystems:          "Strong ecosystem presence",
			DisruptionAIData:                     "High AI disruption potential",
			CompetitiveRivalryIntensity:          "Alta",
			SupplierPowerIntensity:               "Baixa",
			BuyerPowerIntensity:                  "Média",
			ThreatNewEntrantsIntensity:           "Baixa",
			ThreatSubstitutesIntensity:           "Baixa",
			PowerPartnershipsEcosystemsIntensity: "Alta",
			DisruptionAIDataIntensity:            "Alta",
			StrategicImplications:                []string{"Focus on AI", "Build partnerships"},
			OverallAttractiveness:                "Moderately attractive",
			Summary:                              "Porter 7 Forces analysis summary",
		})

		require.NoError(t, repo.Create(ctx, a))

		// Retrieve and verify
		saved, err := repo.GetByID(ctx, a.ID)
		require.NoError(t, err)

		// Get Porter data via GetFramework
		var porterResult analysis.PorterAnalysis
		err = saved.GetFramework("porter", &porterResult)
		require.NoError(t, err)

		assert.Equal(t, "High competition in the market", porterResult.CompetitiveRivalry)
		assert.Equal(t, "Alta", porterResult.CompetitiveRivalryIntensity)
		assert.Len(t, porterResult.StrategicImplications, 2)
		assert.Equal(t, "Porter 7 Forces analysis summary", porterResult.Summary)
	})
}

func TestIntegration_JSONB_OKRsSavesAndLoads(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		companyID := createTestCompany(t, tx)

		now := time.Now()
		a := &analysis.Analysis{
			ID:              uuid.New().String(),
			SubmissionID:    subID,
			CompanyID:       &companyID,
			Status:          string(analysis.StatusPending),
			FrameworkResults: make(map[string]json.RawMessage),
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// Use SetFramework to add OKRs data
		a.SetFramework("okrs", &analysis.OKRAnalysis{
			Quarters: []analysis.QuarterlyOKR{
				{
					Quarter:    "Q1 2025",
					Objective:  "Launch MVP",
					KeyResults: []string{"KR1: 100 users", "KR2: 90% uptime", "KR3: NPS > 50"},
					Investment: "R$ 25 mil",
					Timeline:   "3-4 meses",
				},
				{
					Quarter:    "Q2 2025",
					Objective:  "Scale operations",
					KeyResults: []string{"KR1: 1000 users", "KR2: Revenue R$50k", "KR3: Team of 10"},
					Investment: "R$ 50 mil",
					Timeline:   "3 meses",
				},
			},
			Summary: "Quarterly OKRs aligned with strategic goals",
		})

		require.NoError(t, repo.Create(ctx, a))

		// Retrieve and verify
		saved, err := repo.GetByID(ctx, a.ID)
		require.NoError(t, err)

		// Get OKRs data via GetFramework
		var okrsResult analysis.OKRAnalysis
		err = saved.GetFramework("okrs", &okrsResult)
		require.NoError(t, err)

		assert.Len(t, okrsResult.Quarters, 2)
		assert.Equal(t, "Q1 2025", okrsResult.Quarters[0].Quarter)
		assert.Equal(t, "Launch MVP", okrsResult.Quarters[0].Objective)
		assert.Len(t, okrsResult.Quarters[0].KeyResults, 3)
		assert.Equal(t, "R$ 25 mil", okrsResult.Quarters[0].Investment)
	})
}

func TestIntegration_JSONB_Plan90DaysSavesAndLoads(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		companyID := createTestCompany(t, tx)

		now := time.Now()
		a := &analysis.Analysis{
			ID:              uuid.New().String(),
			SubmissionID:    subID,
			CompanyID:       &companyID,
			Status:          string(analysis.StatusPending),
			FrameworkResults: make(map[string]json.RawMessage),
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// Use SetFramework to add Plan90Days data
		a.SetFramework("okrs", &analysis.OKRAnalysis{
			Plan90Days: []analysis.MonthlyOKR{
				{
					Month:                 "Mês 1",
					Focus:                 "Fundação",
					Objective:             "Validar produto com early adopters",
					KeyResults:            []string{"KR1: 50 usuarios ativos", "KR2: NPS > 40", "KR3: Retenção > 60%"},
					Investment:            "R$ 15 mil",
					AlignedRecommendation: "Foco em validação de mercado",
				},
				{
					Month:                 "Mês 2",
					Focus:                 "Crescimento",
					Objective:             "Escalar base de usuarios",
					KeyResults:            []string{"KR1: 200 usuarios", "KR2: CAC < R$50", "KR3: Receita R$10k"},
					Investment:            "R$ 25 mil",
					AlignedRecommendation: "Implementar growth loops",
				},
				{
					Month:                 "Mês 3",
					Focus:                 "Consolidação",
					Objective:             "Otimizar unit economics",
					KeyResults:            []string{"KR1: LTV/CAC > 3", "KR2: Churn < 5%", "KR3: MRR R$30k"},
					Investment:            "R$ 35 mil",
					AlignedRecommendation: "Foco em retenção e monetização",
				},
			},
			TotalInvestment: "R$ 75 mil",
			SuccessMetrics:  []string{"MRR > R$30k", "500+ usuarios ativos", "NPS > 50"},
			Summary:         "Plano 90 dias alinhado com matriz de decisão",
		})

		require.NoError(t, repo.Create(ctx, a))

		// Retrieve and verify V2 Plan90Days format
		saved, err := repo.GetByID(ctx, a.ID)
		require.NoError(t, err)

		// Get OKRs data via GetFramework
		var okrsResult analysis.OKRAnalysis
		err = saved.GetFramework("okrs", &okrsResult)
		require.NoError(t, err)

		assert.Len(t, okrsResult.Plan90Days, 3)
		assert.Equal(t, "Mês 1", okrsResult.Plan90Days[0].Month)
		assert.Equal(t, "Fundação", okrsResult.Plan90Days[0].Focus)
		assert.Equal(t, "Validar produto com early adopters", okrsResult.Plan90Days[0].Objective)
		assert.Len(t, okrsResult.Plan90Days[0].KeyResults, 3)
		assert.Equal(t, "R$ 15 mil", okrsResult.Plan90Days[0].Investment)
		assert.Equal(t, "Foco em validação de mercado", okrsResult.Plan90Days[0].AlignedRecommendation)
		assert.Equal(t, "R$ 75 mil", okrsResult.TotalInvestment)
		assert.Len(t, okrsResult.SuccessMetrics, 3)

		// Verify legacy Quarters is empty (new format doesn't populate it)
		assert.Len(t, okrsResult.Quarters, 0)
	})
}

func TestIntegration_ErrorMessage_SavesOnFailure(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		companyID := createTestCompany(t, tx)

		now := time.Now()
		a := &analysis.Analysis{
			ID:           uuid.New().String(),
			SubmissionID: subID,
			CompanyID:    &companyID,
			Status:       string(analysis.StatusPending),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		require.NoError(t, repo.Create(ctx, a))

		// Simulate failure (status stays "pending", error message is saved)
		a.Fail("LLM API timeout")
		require.NoError(t, repo.Update(ctx, a))

		saved, _ := repo.GetByID(ctx, a.ID)
		assert.Equal(t, string(analysis.StatusPending), saved.Status) // Status stays pending
		assert.NotNil(t, saved.ErrorMessage)
		assert.Equal(t, "LLM API timeout", *saved.ErrorMessage)
	})
}

func TestIntegration_SoftDelete(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		companyID := createTestCompany(t, tx)

		now := time.Now()
		a := &analysis.Analysis{
			ID:           uuid.New().String(),
			SubmissionID: subID,
			CompanyID:    &companyID,
			Status:       string(analysis.StatusPending),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		require.NoError(t, repo.Create(ctx, a))

		// Soft delete
		_, err := tx.Exec(`UPDATE analyses SET deleted_at = NOW() WHERE id = $1`, a.ID)
		require.NoError(t, err)

		// Should not be found
		_, err = repo.GetByID(ctx, a.ID)
		assert.Error(t, err, "Should not find soft-deleted analysis")
	})
}
