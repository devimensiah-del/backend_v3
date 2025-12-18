//go:build integration

package analysisbysteps_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"backend_v3/config"
	"backend_v3/domain/analysisbysteps"
	"backend_v3/llm"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TEST SETUP HELPERS
// =============================================================================

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

func getMockConfig() *config.Config {
	return &config.Config{
		LLMMockMode:   true, // Enable mock mode for tests
		PrimaryModel:  "test-model",
		AITemperature: 0.5,
		Frameworks:    make(map[string]config.FrameworkConfig),
	}
}

func withTxRollback(t *testing.T, db *sqlx.DB, fn func(tx *sqlx.Tx)) {
	t.Helper()
	tx, err := db.Beginx()
	require.NoError(t, err)
	defer tx.Rollback()
	fn(tx)
}

// createTestCompany creates a company record for FK constraints
func createTestCompany(t *testing.T, tx *sqlx.Tx) uuid.UUID {
	t.Helper()
	companyID := uuid.New()
	_, err := tx.Exec(`
		INSERT INTO companies (id, name, enrichment_status, created_at, updated_at)
		VALUES ($1, 'Test Company for AnalysisBySteps', 'completed', NOW(), NOW())
	`, companyID)
	require.NoError(t, err)
	return companyID
}

// createTestChallenge creates a challenge record for FK constraints
func createTestChallenge(t *testing.T, tx *sqlx.Tx, companyID uuid.UUID) uuid.UUID {
	t.Helper()
	challengeID := uuid.New()
	_, err := tx.Exec(`
		INSERT INTO challenges (id, company_id, challenge_category, challenge_type, business_challenge, created_at, updated_at)
		VALUES ($1, $2, 'growth', 'growth_organic', 'Test business challenge for integration testing', NOW(), NOW())
	`, challengeID, companyID)
	require.NoError(t, err)
	return challengeID
}

// createTestAnalysis creates an analysis record for FK constraints
func createTestAnalysis(t *testing.T, tx *sqlx.Tx, companyID, challengeID uuid.UUID) string {
	t.Helper()
	analysisID := uuid.New().String()
	_, err := tx.Exec(`
		INSERT INTO analyses (id, company_id, challenge_id, status, created_at, updated_at, version, current_step)
		VALUES ($1, $2, $3, 'pending', NOW(), NOW(), 1, 0)
	`, analysisID, companyID, challengeID)
	require.NoError(t, err)
	return analysisID
}

// =============================================================================
// INTEGRATION TESTS: Repository Layer
// =============================================================================

func TestIntegration_Repository_CreateAndGetStep(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()

		// Setup FK dependencies
		companyID := createTestCompany(t, tx)
		challengeID := createTestChallenge(t, tx, companyID)
		analysisID := createTestAnalysis(t, tx, companyID, challengeID)

		// Create step via direct SQL (repository uses db, we use tx)
		stepID := uuid.New().String()
		_, err := tx.ExecContext(ctx, `
			INSERT INTO analysis_steps_v2 (id, analysis_id, framework_code, step_number, status, visible, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		`, stepID, analysisID, "challenge_refinement", 0, "pending", true)
		require.NoError(t, err)

		// Verify step exists
		var step analysisbysteps.AnalysisStep
		err = tx.GetContext(ctx, &step, `
			SELECT id, analysis_id, framework_code, step_number, status, visible, ai_output, human_edited, generated_at, approved_at
			FROM analysis_steps_v2 WHERE id = $1
		`, stepID)
		require.NoError(t, err)

		assert.Equal(t, stepID, step.ID)
		assert.Equal(t, analysisID, step.AnalysisID)
		assert.Equal(t, "challenge_refinement", step.FrameworkCode)
		assert.Equal(t, 0, step.StepNumber)
		assert.Equal(t, analysisbysteps.StatusPending, step.Status)
		assert.True(t, step.Visible)
		assert.Nil(t, step.AIOutput)
		assert.Nil(t, step.HumanEdited)
	})
}

func TestIntegration_Repository_UpdateStepWithAIOutput(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()

		// Setup
		companyID := createTestCompany(t, tx)
		challengeID := createTestChallenge(t, tx, companyID)
		analysisID := createTestAnalysis(t, tx, companyID, challengeID)

		// Create step
		stepID := uuid.New().String()
		_, err := tx.ExecContext(ctx, `
			INSERT INTO analysis_steps_v2 (id, analysis_id, framework_code, step_number, status, visible, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		`, stepID, analysisID, "pestel", 1, "pending", true)
		require.NoError(t, err)

		// Simulate LLM output
		aiOutput := `{"political": ["Factor 1", "Factor 2"], "summary": "PESTEL analysis complete"}`

		// Update with AI output
		_, err = tx.ExecContext(ctx, `
			UPDATE analysis_steps_v2
			SET ai_output = $1, status = $2, generated_at = NOW(), updated_at = NOW()
			WHERE id = $3
		`, aiOutput, "generated", stepID)
		require.NoError(t, err)

		// Verify update
		var step analysisbysteps.AnalysisStep
		err = tx.GetContext(ctx, &step, `
			SELECT id, ai_output, status, generated_at FROM analysis_steps_v2 WHERE id = $1
		`, stepID)
		require.NoError(t, err)

		assert.NotNil(t, step.AIOutput)
		assert.Equal(t, aiOutput, *step.AIOutput)
		assert.Equal(t, analysisbysteps.StatusGenerated, step.Status)
		assert.NotNil(t, step.GeneratedAt)
	})
}

func TestIntegration_Repository_SetHumanEdited(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()

		// Setup
		companyID := createTestCompany(t, tx)
		challengeID := createTestChallenge(t, tx, companyID)
		analysisID := createTestAnalysis(t, tx, companyID, challengeID)

		// Create step with AI output
		stepID := uuid.New().String()
		aiOutput := `{"original": "ai content"}`
		_, err := tx.ExecContext(ctx, `
			INSERT INTO analysis_steps_v2 (id, analysis_id, framework_code, step_number, status, ai_output, visible, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		`, stepID, analysisID, "swot", 4, "generated", aiOutput, true)
		require.NoError(t, err)

		// Set human edit
		humanEdited := `{"edited": "human content", "improved": true}`
		_, err = tx.ExecContext(ctx, `
			UPDATE analysis_steps_v2
			SET human_edited = $1, updated_at = NOW()
			WHERE id = $2
		`, humanEdited, stepID)
		require.NoError(t, err)

		// Verify - both ai_output and human_edited should exist
		var step analysisbysteps.AnalysisStep
		err = tx.GetContext(ctx, &step, `
			SELECT ai_output, human_edited, status FROM analysis_steps_v2 WHERE id = $1
		`, stepID)
		require.NoError(t, err)

		assert.NotNil(t, step.AIOutput)
		assert.Equal(t, aiOutput, *step.AIOutput)
		assert.NotNil(t, step.HumanEdited)
		assert.Equal(t, humanEdited, *step.HumanEdited)
		// Status should remain "generated" (not changed by human edit)
		assert.Equal(t, analysisbysteps.StatusGenerated, step.Status)
	})
}

func TestIntegration_Repository_ApproveStep(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()

		// Setup
		companyID := createTestCompany(t, tx)
		challengeID := createTestChallenge(t, tx, companyID)
		analysisID := createTestAnalysis(t, tx, companyID, challengeID)

		// Create step with content
		stepID := uuid.New().String()
		aiOutput := `{"content": "ready for approval"}`
		_, err := tx.ExecContext(ctx, `
			INSERT INTO analysis_steps_v2 (id, analysis_id, framework_code, step_number, status, ai_output, visible, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		`, stepID, analysisID, "porter", 2, "generated", aiOutput, true)
		require.NoError(t, err)

		// Approve
		_, err = tx.ExecContext(ctx, `
			UPDATE analysis_steps_v2
			SET status = $1, approved_at = NOW(), updated_at = NOW()
			WHERE id = $2
		`, "approved", stepID)
		require.NoError(t, err)

		// Verify
		var step analysisbysteps.AnalysisStep
		err = tx.GetContext(ctx, &step, `
			SELECT status, approved_at FROM analysis_steps_v2 WHERE id = $1
		`, stepID)
		require.NoError(t, err)

		assert.Equal(t, analysisbysteps.StatusApproved, step.Status)
		assert.NotNil(t, step.ApprovedAt)
	})
}

func TestIntegration_Repository_GetByAnalysisID_OrderedByStepNumber(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()

		// Setup
		companyID := createTestCompany(t, tx)
		challengeID := createTestChallenge(t, tx, companyID)
		analysisID := createTestAnalysis(t, tx, companyID, challengeID)

		// Create 5 steps in random order
		stepsToCreate := []struct {
			code       string
			stepNumber int
		}{
			{"swot", 4},
			{"challenge_refinement", 0},
			{"porter", 2},
			{"pestel", 1},
			{"benchmarking", 3},
		}

		for _, s := range stepsToCreate {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO analysis_steps_v2 (id, analysis_id, framework_code, step_number, status, visible, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
			`, uuid.New().String(), analysisID, s.code, s.stepNumber, "pending", true)
			require.NoError(t, err)
		}

		// Query all steps ordered by step_number
		var steps []analysisbysteps.AnalysisStep
		err := tx.SelectContext(ctx, &steps, `
			SELECT id, analysis_id, framework_code, step_number, status
			FROM analysis_steps_v2
			WHERE analysis_id = $1
			ORDER BY step_number ASC
		`, analysisID)
		require.NoError(t, err)

		// Verify ordering
		require.Len(t, steps, 5)
		assert.Equal(t, 0, steps[0].StepNumber)
		assert.Equal(t, "challenge_refinement", steps[0].FrameworkCode)
		assert.Equal(t, 1, steps[1].StepNumber)
		assert.Equal(t, "pestel", steps[1].FrameworkCode)
		assert.Equal(t, 2, steps[2].StepNumber)
		assert.Equal(t, "porter", steps[2].FrameworkCode)
		assert.Equal(t, 3, steps[3].StepNumber)
		assert.Equal(t, "benchmarking", steps[3].FrameworkCode)
		assert.Equal(t, 4, steps[4].StepNumber)
		assert.Equal(t, "swot", steps[4].FrameworkCode)
	})
}

// =============================================================================
// INTEGRATION TESTS: Mock LLM Responses
// =============================================================================

func TestIntegration_MockLLM_ReturnsValidJSON(t *testing.T) {
	// Test all 14 framework mock responses
	frameworks := []string{
		"challenge_refinement", "pestel", "porter", "benchmarking",
		"swot", "swotcross", "tam_sam_som", "blue_ocean",
		"growth_hacking", "scenarios", "decision_matrix",
		"okrs", "bsc", "synthesis",
	}

	for _, framework := range frameworks {
		t.Run(framework, func(t *testing.T) {
			response, err := llm.GetMockResponse(framework)
			require.NoError(t, err, "Mock response should exist for %s", framework)
			require.NotEmpty(t, response, "Mock response should not be empty for %s", framework)

			// Verify it's valid JSON
			var parsed map[string]interface{}
			err = json.Unmarshal([]byte(response), &parsed)
			require.NoError(t, err, "Mock response should be valid JSON for %s", framework)

			// Verify it has expected structure
			assert.NotEmpty(t, parsed, "Parsed JSON should have content for %s", framework)
		})
	}
}

func TestIntegration_MockLLM_ChallengeRefinement_HasRequiredFields(t *testing.T) {
	var result map[string]interface{}
	err := llm.GetMockResponseStruct("challenge_refinement", &result)
	require.NoError(t, err)

	// Check expected structure
	assert.Contains(t, result, "challenge_analysis")
	assert.Contains(t, result, "measurement")
	assert.Contains(t, result, "summary")
	assert.Contains(t, result, "confidence_score")

	// Check nested challenge_analysis
	challengeAnalysis, ok := result["challenge_analysis"].(map[string]interface{})
	require.True(t, ok, "challenge_analysis should be a map")
	assert.Contains(t, challengeAnalysis, "declared_challenge")
	assert.Contains(t, challengeAnalysis, "refined_challenge")
}

func TestIntegration_MockLLM_SWOT_HasRequiredFields(t *testing.T) {
	var result map[string]interface{}
	err := llm.GetMockResponseStruct("swot", &result)
	require.NoError(t, err)

	// Check SWOT quadrants
	assert.Contains(t, result, "strengths")
	assert.Contains(t, result, "weaknesses")
	assert.Contains(t, result, "opportunities")
	assert.Contains(t, result, "threats")
	assert.Contains(t, result, "summary")

	// Verify strengths is an array with confidence/source
	strengths, ok := result["strengths"].([]interface{})
	require.True(t, ok, "strengths should be an array")
	require.NotEmpty(t, strengths)

	firstStrength, ok := strengths[0].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, firstStrength, "content")
	assert.Contains(t, firstStrength, "confidence")
	assert.Contains(t, firstStrength, "source")
}

func TestIntegration_MockLLM_Synthesis_HasRequiredFields(t *testing.T) {
	var result map[string]interface{}
	err := llm.GetMockResponseStruct("synthesis", &result)
	require.NoError(t, err)

	// Check synthesis fields
	assert.Contains(t, result, "executive_summary")
	assert.Contains(t, result, "central_challenge")
	assert.Contains(t, result, "key_insights")
	assert.Contains(t, result, "strategic_priorities")
	assert.Contains(t, result, "recommended_actions")
	assert.Contains(t, result, "expected_outcomes")
	assert.Contains(t, result, "next_steps")
}

// =============================================================================
// INTEGRATION TESTS: Full Workflow with Mock LLM
// =============================================================================

func TestIntegration_FullWorkflow_CreateAllSteps(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()

		// Setup FK dependencies
		companyID := createTestCompany(t, tx)
		challengeID := createTestChallenge(t, tx, companyID)
		analysisID := createTestAnalysis(t, tx, companyID, challengeID)

		// Create all 14 steps (simulating StartAnalysisBySteps)
		for i, meta := range analysisbysteps.FrameworkOrder {
			stepID := uuid.New().String()
			_, err := tx.ExecContext(ctx, `
				INSERT INTO analysis_steps_v2 (id, analysis_id, framework_code, step_number, status, visible, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
			`, stepID, analysisID, meta.Code, i, "pending", true)
			require.NoError(t, err, "Failed to create step %d (%s)", i, meta.Code)
		}

		// Verify all 14 steps exist
		var count int
		err := tx.GetContext(ctx, &count, `SELECT COUNT(*) FROM analysis_steps_v2 WHERE analysis_id = $1`, analysisID)
		require.NoError(t, err)
		assert.Equal(t, 14, count)

		// Verify first step is challenge_refinement
		var firstStep analysisbysteps.AnalysisStep
		err = tx.GetContext(ctx, &firstStep, `
			SELECT framework_code, step_number FROM analysis_steps_v2
			WHERE analysis_id = $1 ORDER BY step_number ASC LIMIT 1
		`, analysisID)
		require.NoError(t, err)
		assert.Equal(t, "challenge_refinement", firstStep.FrameworkCode)
		assert.Equal(t, 0, firstStep.StepNumber)

		// Verify last step is synthesis
		var lastStep analysisbysteps.AnalysisStep
		err = tx.GetContext(ctx, &lastStep, `
			SELECT framework_code, step_number FROM analysis_steps_v2
			WHERE analysis_id = $1 ORDER BY step_number DESC LIMIT 1
		`, analysisID)
		require.NoError(t, err)
		assert.Equal(t, "synthesis", lastStep.FrameworkCode)
		assert.Equal(t, 13, lastStep.StepNumber)
	})
}

func TestIntegration_FullWorkflow_GenerateEditApprove(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()

		// Setup
		companyID := createTestCompany(t, tx)
		challengeID := createTestChallenge(t, tx, companyID)
		analysisID := createTestAnalysis(t, tx, companyID, challengeID)

		// Create step 0
		stepID := uuid.New().String()
		_, err := tx.ExecContext(ctx, `
			INSERT INTO analysis_steps_v2 (id, analysis_id, framework_code, step_number, status, visible, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		`, stepID, analysisID, "challenge_refinement", 0, "pending", true)
		require.NoError(t, err)

		// Step 1: Generate (simulate with mock response)
		mockResponse, err := llm.GetMockResponse("challenge_refinement")
		require.NoError(t, err)

		_, err = tx.ExecContext(ctx, `
			UPDATE analysis_steps_v2
			SET status = 'generating', updated_at = NOW()
			WHERE id = $1
		`, stepID)
		require.NoError(t, err)

		// Verify status is generating
		var step analysisbysteps.AnalysisStep
		err = tx.GetContext(ctx, &step, `SELECT status FROM analysis_steps_v2 WHERE id = $1`, stepID)
		require.NoError(t, err)
		assert.Equal(t, analysisbysteps.StatusGenerating, step.Status)

		// Complete generation
		_, err = tx.ExecContext(ctx, `
			UPDATE analysis_steps_v2
			SET ai_output = $1, status = 'generated', generated_at = NOW(), updated_at = NOW()
			WHERE id = $2
		`, mockResponse, stepID)
		require.NoError(t, err)

		// Verify status is generated
		err = tx.GetContext(ctx, &step, `SELECT status, ai_output, generated_at FROM analysis_steps_v2 WHERE id = $1`, stepID)
		require.NoError(t, err)
		assert.Equal(t, analysisbysteps.StatusGenerated, step.Status)
		assert.NotNil(t, step.AIOutput)
		assert.NotNil(t, step.GeneratedAt)

		// Step 2: Human edit
		humanEdit := `{"edited": true, "challenge_analysis": {"refined_challenge": "Human-refined challenge"}}`
		_, err = tx.ExecContext(ctx, `
			UPDATE analysis_steps_v2
			SET human_edited = $1, updated_at = NOW()
			WHERE id = $2
		`, humanEdit, stepID)
		require.NoError(t, err)

		// Verify human edit saved (status unchanged)
		err = tx.GetContext(ctx, &step, `SELECT status, human_edited FROM analysis_steps_v2 WHERE id = $1`, stepID)
		require.NoError(t, err)
		assert.Equal(t, analysisbysteps.StatusGenerated, step.Status) // Still generated
		assert.NotNil(t, step.HumanEdited)

		// Step 3: Approve
		_, err = tx.ExecContext(ctx, `
			UPDATE analysis_steps_v2
			SET status = 'approved', approved_at = NOW(), updated_at = NOW()
			WHERE id = $1
		`, stepID)
		require.NoError(t, err)

		// Verify approved
		err = tx.GetContext(ctx, &step, `SELECT status, approved_at FROM analysis_steps_v2 WHERE id = $1`, stepID)
		require.NoError(t, err)
		assert.Equal(t, analysisbysteps.StatusApproved, step.Status)
		assert.NotNil(t, step.ApprovedAt)
	})
}

func TestIntegration_FullWorkflow_MultipleStepsSequential(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()

		// Setup
		companyID := createTestCompany(t, tx)
		challengeID := createTestChallenge(t, tx, companyID)
		analysisID := createTestAnalysis(t, tx, companyID, challengeID)

		// Create first 3 steps
		frameworks := []string{"challenge_refinement", "pestel", "porter"}
		stepIDs := make([]string, 3)

		for i, framework := range frameworks {
			stepIDs[i] = uuid.New().String()
			_, err := tx.ExecContext(ctx, `
				INSERT INTO analysis_steps_v2 (id, analysis_id, framework_code, step_number, status, visible, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
			`, stepIDs[i], analysisID, framework, i, "pending", true)
			require.NoError(t, err)
		}

		// Process each step: generate → approve
		for i, framework := range frameworks {
			// Generate
			mockResponse, _ := llm.GetMockResponse(framework)
			_, err := tx.ExecContext(ctx, `
				UPDATE analysis_steps_v2
				SET ai_output = $1, status = 'generated', generated_at = NOW(), updated_at = NOW()
				WHERE id = $2
			`, mockResponse, stepIDs[i])
			require.NoError(t, err)

			// Approve
			_, err = tx.ExecContext(ctx, `
				UPDATE analysis_steps_v2
				SET status = 'approved', approved_at = NOW(), updated_at = NOW()
				WHERE id = $1
			`, stepIDs[i])
			require.NoError(t, err)
		}

		// Verify all 3 steps are approved
		var steps []analysisbysteps.AnalysisStep
		err := tx.SelectContext(ctx, &steps, `
			SELECT status FROM analysis_steps_v2
			WHERE analysis_id = $1 ORDER BY step_number ASC
		`, analysisID)
		require.NoError(t, err)
		require.Len(t, steps, 3)

		for i, step := range steps {
			assert.Equal(t, analysisbysteps.StatusApproved, step.Status, "Step %d should be approved", i)
		}
	})
}

// =============================================================================
// INTEGRATION TESTS: Edge Cases
// =============================================================================

func TestIntegration_GetEffectiveOutput_PrioritizesHumanEdit(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()

		// Setup
		companyID := createTestCompany(t, tx)
		challengeID := createTestChallenge(t, tx, companyID)
		analysisID := createTestAnalysis(t, tx, companyID, challengeID)

		// Create step with both ai_output and human_edited
		stepID := uuid.New().String()
		aiOutput := `{"source": "ai", "content": "AI generated"}`
		humanEdited := `{"source": "human", "content": "Human edited"}`

		_, err := tx.ExecContext(ctx, `
			INSERT INTO analysis_steps_v2 (id, analysis_id, framework_code, step_number, status, ai_output, human_edited, visible, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		`, stepID, analysisID, "swot", 4, "generated", aiOutput, humanEdited, true)
		require.NoError(t, err)

		// Fetch and check GetEffectiveOutput
		var step analysisbysteps.AnalysisStep
		err = tx.GetContext(ctx, &step, `
			SELECT ai_output, human_edited FROM analysis_steps_v2 WHERE id = $1
		`, stepID)
		require.NoError(t, err)

		// GetEffectiveOutput should return human_edited
		effective := step.GetEffectiveOutput()
		require.NotNil(t, effective)
		assert.Equal(t, humanEdited, *effective)
		assert.NotEqual(t, aiOutput, *effective)
	})
}

func TestIntegration_StepStatusTransitions_AllValid(t *testing.T) {
	statuses := []analysisbysteps.StepStatus{
		analysisbysteps.StatusPending,
		analysisbysteps.StatusGenerating,
		analysisbysteps.StatusGenerated,
		analysisbysteps.StatusApproved,
		analysisbysteps.StatusFailed,
	}

	db := getTestDB(t)
	defer db.Close()

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			withTxRollback(t, db, func(tx *sqlx.Tx) {
				ctx := context.Background()

				// Setup
				companyID := createTestCompany(t, tx)
				challengeID := createTestChallenge(t, tx, companyID)
				analysisID := createTestAnalysis(t, tx, companyID, challengeID)

				// Create step with this status
				stepID := uuid.New().String()
				_, err := tx.ExecContext(ctx, `
					INSERT INTO analysis_steps_v2 (id, analysis_id, framework_code, step_number, status, visible, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
				`, stepID, analysisID, "pestel", 1, status, true)
				require.NoError(t, err)

				// Verify it was saved correctly
				var savedStatus string
				err = tx.GetContext(ctx, &savedStatus, `SELECT status FROM analysis_steps_v2 WHERE id = $1`, stepID)
				require.NoError(t, err)
				assert.Equal(t, string(status), savedStatus)
			})
		})
	}
}

// =============================================================================
// BENCHMARK: Mock vs Real LLM (for reference)
// =============================================================================

func BenchmarkMockLLMResponse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := llm.GetMockResponse("synthesis")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func init() {
	// Setup zerolog for tests
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}
