package analysis

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	return sqlxDB, mock
}

func createTestAnalysisForRepo() *Analysis {
	now := time.Now()
	analysisID := uuid.New().String()
	companyID := uuid.New().String()
	challengeID := uuid.New()

	analysis := &Analysis{
		ID:               analysisID,
		CompanyID:        &companyID,
		ChallengeID:      challengeID,
		Status:           string(StatusCompleted),
		CreatedAt:        now,
		UpdatedAt:        now,
		CompletedAt:      &now,
		FrameworkResults: make(map[string]json.RawMessage),
	}

	// Use SetFramework to populate framework data
	analysis.SetFramework("pestel", &PESTELAnalysis{
		Political:     []string{"Political factor 1"},
		Economic:      []string{"Economic factor 1"},
		Social:        []string{"Social factor 1"},
		Technological: []string{"Tech factor 1"},
		Environmental: []string{"Env factor 1"},
		Legal:         []string{"Legal factor 1"},
		Summary:       "PESTEL summary",
	})

	analysis.SetFramework("porter", &PorterAnalysis{
		CompetitiveRivalry:    "High",
		SupplierPower:         "Medium",
		BuyerPower:            "High",
		ThreatNewEntrants:     "Low",
		ThreatSubstitutes:     "Medium",
		OverallAttractiveness: "Medium",
		Summary:               "Porter summary",
	})

	analysis.SetFramework("tam_sam_som", &TamSamSomAnalysis{
		TAM:         "R$ 10B",
		SAM:         "R$ 2B",
		SOM:         "R$ 200M",
		CAGR:        "8% CAGR",
		Assumptions: []string{"Market assumption 1"},
		Summary:     "TAM summary",
	})

	analysis.SetFramework("swot", &SWOTAnalysis{
		Strengths:     []SWOTItem{{Content: "Strong brand", Confidence: "Alta", Source: "fato"}},
		Weaknesses:    []SWOTItem{{Content: "Low tech", Confidence: "Média", Source: "análise de mercado"}},
		Opportunities: []SWOTItem{{Content: "Market expansion", Confidence: "Alta", Source: "fato"}},
		Threats:       []SWOTItem{{Content: "New competitors", Confidence: "Média", Source: "estimativa"}},
		Summary:       "SWOT summary",
	})

	analysis.SetFramework("benchmarking", &BenchmarkingAnalysis{
		Competitors:     []string{"Competitor 1", "Competitor 2"},
		PerformanceGaps: []string{"Gap 1"},
		BestPractices:   []string{"Practice 1"},
		Summary:         "Benchmarking summary",
	})

	analysis.SetFramework("blue_ocean", &BlueOceanAnalysis{
		Eliminate:     []string{"Factor to eliminate"},
		Reduce:        []string{"Factor to reduce"},
		Raise:         []string{"Factor to raise"},
		Create:        []string{"Factor to create"},
		NewValueCurve: "New value proposition",
		Summary:       "Blue Ocean summary",
	})

	analysis.SetFramework("growth_hacking", &GrowthHackingAnalysis{
		LeapLoop: GrowthLoop{
			Name:       "LEAP Loop",
			Type:       "acquisition",
			Steps:      []string{"Land", "Engage", "Activate", "Propagate"},
			Metrics:    []string{"CAC", "Conversion"},
			Bottleneck: "Engagement",
		},
		ScaleLoop: GrowthLoop{
			Name:       "SCALE Loop",
			Type:       "monetization",
			Steps:      []string{"Satisfy", "Convert", "Loop Back", "Expand"},
			Metrics:    []string{"LTV", "Revenue"},
			Bottleneck: "Conversion",
		},
		Summary: "Growth Hacking summary",
	})

	analysis.SetFramework("scenarios", &ScenarioAnalysis{
		Optimistic: Scenario{
			Name:            "Cenário Otimista",
			Probability:     20,
			Description:     "Best case scenario",
			RequiredActions: []string{"Action 1"},
		},
		Realist: Scenario{
			Name:            "Cenário Realista",
			Probability:     60,
			Description:     "Most likely scenario",
			RequiredActions: []string{"Action 2"},
		},
		Pessimistic: Scenario{
			Name:            "Cenário Pessimista",
			Probability:     20,
			Description:     "Worst case scenario",
			RequiredActions: []string{"Action 3"},
		},
		MitigationTactics:   []string{"Tactic 1"},
		EarlyWarningSignals: []string{"Signal 1"},
		Summary:             "Scenarios summary",
	})

	analysis.SetFramework("okrs", &OKRAnalysis{
		Quarters: []QuarterlyOKR{
			{
				Quarter:    "Q1 2025",
				Objective:  "Launch new product",
				KeyResults: []string{"KR1", "KR2", "KR3"},
				Investment: "R$ 100k",
				Timeline:   "3 months",
			},
		},
		Summary: "OKRs summary",
	})

	analysis.SetFramework("bsc", &BalancedScorecardAnalysis{
		Financial:      []string{"Financial metric 1"},
		Customer:       []string{"Customer metric 1"},
		Internal:       []string{"Internal metric 1"},
		LearningGrowth: []string{"Learning metric 1"},
		Summary:        "BSC summary",
	})

	analysis.SetFramework("decision_matrix", &DecisionMatrixAnalysis{
		Alternatives:        []string{"Alt 1", "Alt 2"},
		Criteria:            []string{"Criteria 1"},
		FinalRecommendation: "Choose Alt 1",
		RecommendedOption:   "Alt 1",
		Score:               "8.5/10",
		ScoreComparison:     "+15% above Alt 2",
		PriorityRecommendations: []PriorityRecommendation{
			{
				Priority:    1,
				Title:       "Recommendation 1",
				Description: "Detailed description",
				Timeline:    "6 months",
				Budget:      "R$ 150k",
			},
		},
		ReviewCycle: ReviewCycle{
			Frequency:             "Trimestral",
			ExtraordinaryTriggers: []string{"Market shift"},
		},
		MonitoringMetrics: []string{"Metric 1"},
		Summary:           "Decision Matrix summary",
	})

	analysis.SetFramework("synthesis", &AnalysisSynthesis{
		ExecutiveSummary:      "Executive summary text",
		CentralChallenge:      "Main challenge",
		MainFindings:          []string{"Finding 1", "Finding 2"},
		ImportantNotes:        []string{"Note 1"},
		KeyFindings:           []string{"Key finding 1"},
		StrategicPriorities:   []string{"Priority 1"},
		Roadmap:               []string{"Milestone 1"},
		OverallRecommendation: "Overall recommendation",
	})

	return analysis
}

// =============================================================================
// TESTS: Create
// =============================================================================

func TestRepository_Create_WithAll11Frameworks(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	testAnalysis := createTestAnalysisForRepo()

	// Expect INSERT with framework_results as single JSONB column (v2_013 schema)
	mock.ExpectExec(`INSERT INTO analyses`).
		WithArgs(
			testAnalysis.ID,
			testAnalysis.CompanyID,
			testAnalysis.ChallengeID, // Challenge ID (v2)
			sqlmock.AnyArg(), // framework_results (JSONB)
			testAnalysis.Status,
			testAnalysis.ErrorMessage,
			testAnalysis.IsVisibleToUser,
			testAnalysis.IsPublic,
			testAnalysis.AccessCode,
			testAnalysis.DeletedAt,
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
			testAnalysis.CompletedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), testAnalysis)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Create_JSONBSerialization(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	testAnalysis := createTestAnalysisForRepo()

	mock.ExpectExec(`INSERT INTO analyses`).
		WillReturnResult(sqlmock.NewResult(1, 1)).
		WillDelayFor(0)

	err := repo.Create(context.Background(), testAnalysis)

	assert.NoError(t, err)

	// Verify JSONB can be serialized from FrameworkResults
	var swotData SWOTAnalysis
	err = testAnalysis.GetFramework("swot", &swotData)
	assert.NoError(t, err)

	var pestelData PESTELAnalysis
	err = testAnalysis.GetFramework("pestel", &pestelData)
	assert.NoError(t, err)

	var porterData PorterAnalysis
	err = testAnalysis.GetFramework("porter", &porterData)
	assert.NoError(t, err)
}

// =============================================================================
// TESTS: Update vs UpdateWithTx
// =============================================================================

func TestRepository_Update_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	testAnalysis := createTestAnalysisForRepo()

	mock.ExpectExec(`UPDATE analyses SET`).
		WithArgs(
			sqlmock.AnyArg(), // framework_results (JSONB) - v2_013 schema
			testAnalysis.Status,
			testAnalysis.ErrorMessage,
			testAnalysis.IsVisibleToUser,
			testAnalysis.IsPublic,
			testAnalysis.AccessCode,
			testAnalysis.DeletedAt,
			sqlmock.AnyArg(), // updated_at
			testAnalysis.CompletedAt,
			testAnalysis.CurrentStep,   // wizard field
			testAnalysis.WizardMode,    // wizard field
			sqlmock.AnyArg(),           // steps_completed (JSONB array)
			testAnalysis.ID,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(context.Background(), testAnalysis)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Update_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	testAnalysis := createTestAnalysisForRepo()

	mock.ExpectExec(`UPDATE analyses SET`).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

	err := repo.Update(context.Background(), testAnalysis)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateWithTx_TransactionHandling(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	testAnalysis := createTestAnalysisForRepo()

	// Begin transaction
	mock.ExpectBegin()

	tx, err := repo.BeginTx(context.Background())
	require.NoError(t, err)

	// Expect update within transaction
	mock.ExpectExec(`UPDATE analyses SET`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateWithTx(context.Background(), tx, testAnalysis)
	assert.NoError(t, err)

	// Commit transaction
	mock.ExpectCommit()
	err = tx.Commit()
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BeginTx_CommitRollback(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)

	t.Run("Commit Transaction", func(t *testing.T) {
		mock.ExpectBegin()
		tx, err := repo.BeginTx(context.Background())
		require.NoError(t, err)

		mock.ExpectCommit()
		err = tx.Commit()
		assert.NoError(t, err)
	})

	t.Run("Rollback Transaction", func(t *testing.T) {
		mock.ExpectBegin()
		tx, err := repo.BeginTx(context.Background())
		require.NoError(t, err)

		mock.ExpectRollback()
		err = tx.Rollback()
		assert.NoError(t, err)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

// =============================================================================
// TESTS: GetByID
// =============================================================================

func TestRepository_GetByID_Success(t *testing.T) {
	t.Skip("Skipping due to sqlmock/sqlx complex type scanning issue. Integration tests verify this functionality.")

	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	testAnalysis := createTestAnalysisForRepo()

	// Serialize framework_results to JSON - pass as []byte for sqlx scanning
	frameworkResultsJSON, _ := json.Marshal(testAnalysis.FrameworkResults)

	rows := sqlmock.NewRows([]string{
		"id", "company_id", "challenge_id",
		"framework_results", "status", "error_message",
		"is_visible_to_user", "is_public", "access_code", "deleted_at",
		"created_at", "updated_at", "completed_at",
	}).AddRow(
		testAnalysis.ID, testAnalysis.CompanyID, testAnalysis.ChallengeID,
		frameworkResultsJSON, testAnalysis.Status, testAnalysis.ErrorMessage,
		testAnalysis.IsVisibleToUser,
		testAnalysis.IsPublic, testAnalysis.AccessCode, testAnalysis.DeletedAt,
		testAnalysis.CreatedAt, testAnalysis.UpdatedAt, testAnalysis.CompletedAt,
	)

	mock.ExpectQuery(`SELECT (.+) FROM analyses WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(testAnalysis.ID).
		WillReturnRows(rows)

	result, err := repo.GetByID(context.Background(), testAnalysis.ID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testAnalysis.ID, result.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	analysisID := uuid.New().String()

	mock.ExpectQuery(`SELECT (.+) FROM analyses WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(analysisID).
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByID(context.Background(), analysisID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// =============================================================================
// TESTS: Delete (Soft Delete)
// =============================================================================

func TestRepository_Delete_SoftDeleteSuccess(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	analysisID := uuid.New().String()

	mock.ExpectExec(`UPDATE analyses SET deleted_at = NOW\(\) WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(analysisID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(context.Background(), analysisID)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Delete_AlreadyDeleted(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	analysisID := uuid.New().String()

	mock.ExpectExec(`UPDATE analyses SET deleted_at = NOW\(\) WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(analysisID).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

	err := repo.Delete(context.Background(), analysisID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found or already deleted")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// =============================================================================
// TESTS: List with Pagination
// =============================================================================

func TestRepository_List_WithPagination(t *testing.T) {
	t.Skip("Skipping due to sqlmock/sqlx complex type scanning issue. Integration tests verify this functionality.")

	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)

	a1 := createTestAnalysisForRepo()
	a2 := createTestAnalysisForRepo()

	// Serialize framework_results to JSON - pass as []byte for sqlx scanning
	a1JSON, _ := json.Marshal(a1.FrameworkResults)
	a2JSON, _ := json.Marshal(a2.FrameworkResults)

	rows := sqlmock.NewRows([]string{
		"id", "company_id", "challenge_id",
		"framework_results", "status", "error_message",
		"is_visible_to_user", "is_public", "access_code", "deleted_at",
		"created_at", "updated_at", "completed_at",
	}).
		AddRow(
			a1.ID, a1.CompanyID, a1.ChallengeID,
			a1JSON, a1.Status, a1.ErrorMessage,
			a1.IsVisibleToUser, a1.IsPublic,
			a1.AccessCode, a1.DeletedAt,
			a1.CreatedAt, a1.UpdatedAt, a1.CompletedAt,
		).
		AddRow(
			a2.ID, a2.CompanyID, a2.ChallengeID,
			a2JSON, a2.Status, a2.ErrorMessage,
			a2.IsVisibleToUser, a2.IsPublic,
			a2.AccessCode, a2.DeletedAt,
			a2.CreatedAt, a2.UpdatedAt, a2.CompletedAt,
		)

	mock.ExpectQuery(`SELECT (.+) FROM analyses WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT \$1 OFFSET \$2`).
		WithArgs(10, 0).
		WillReturnRows(rows)

	results, err := repo.List(context.Background(), 10, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func marshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

// =============================================================================
// TESTS: JSONB Column Verification
// =============================================================================

func TestRepository_VerifyAll11FrameworksInJSONB(t *testing.T) {
	testAnalysis := createTestAnalysisForRepo()

	// Verify all 11 frameworks can be retrieved from FrameworkResults
	frameworks := []struct {
		name   string
		code   string
		schema interface{}
	}{
		{"PESTEL", "pestel", &PESTELAnalysis{}},
		{"Porter", "porter", &PorterAnalysis{}},
		{"TamSamSom", "tam_sam_som", &TamSamSomAnalysis{}},
		{"SWOT", "swot", &SWOTAnalysis{}},
		{"Benchmarking", "benchmarking", &BenchmarkingAnalysis{}},
		{"BlueOcean", "blue_ocean", &BlueOceanAnalysis{}},
		{"GrowthHacking", "growth_hacking", &GrowthHackingAnalysis{}},
		{"Scenarios", "scenarios", &ScenarioAnalysis{}},
		{"OKRs", "okrs", &OKRAnalysis{}},
		{"BSC", "bsc", &BalancedScorecardAnalysis{}},
		{"DecisionMatrix", "decision_matrix", &DecisionMatrixAnalysis{}},
	}

	for _, fw := range frameworks {
		t.Run(fw.name, func(t *testing.T) {
			err := testAnalysis.GetFramework(fw.code, fw.schema)
			assert.NoError(t, err, "Framework %s should be retrievable", fw.name)

			// Verify the raw JSON exists
			rawJSON, exists := testAnalysis.FrameworkResults[fw.code]
			assert.True(t, exists, "Framework %s should exist in FrameworkResults", fw.name)
			assert.NotEmpty(t, rawJSON, "Framework %s JSON should not be empty", fw.name)
		})
	}
}

// =============================================================================
// TESTS: Transaction Error Handling
// =============================================================================

func TestRepository_BeginTx_Error(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)

	mock.ExpectBegin().WillReturnError(errors.New("database connection lost"))

	tx, err := repo.BeginTx(context.Background())

	assert.Error(t, err)
	assert.Nil(t, tx)
	assert.Contains(t, err.Error(), "failed to begin transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}
