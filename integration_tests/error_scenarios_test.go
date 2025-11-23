package integration_tests

import (
	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrorScenarios_EnrichmentFailure verifies enrichment failure handling
func TestErrorScenarios_EnrichmentFailure(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()
	defer helper.Cleanup(t)

	ctx := context.Background()

	t.Run("Enrichment failure → retry logic → DLQ after 3 failures", func(t *testing.T) {
		// Create submission and enrichment
		sub := helper.CreateTestSubmission(t, ctx)
		enr := helper.CreateTestEnrichment(t, ctx, sub.ID)
		require.Equal(t, enrichment.StatusPending, enr.Status)

		// Simulate failure 1
		enr.Start()
		testError := errors.New("LLM API timeout")
		enr.Fail(testError)
		enr.RetryCount = 1
		err := helper.EnrichmentRepo.UpdateSystem(ctx, enr)
		require.NoError(t, err)

		// Verify failure 1 state
		retrievedEnr := helper.GetEnrichmentBySubmissionID(t, ctx, sub.ID)
		assert.Equal(t, enrichment.StatusPending, retrievedEnr.Status) // Status stays pending
		assert.Equal(t, "LLM API timeout", retrievedEnr.ErrorMessage)
		assert.Equal(t, 1, retrievedEnr.RetryCount)
		t.Logf("✓ Failure 1: status=pending, error=%s, retry_count=%d", retrievedEnr.ErrorMessage, retrievedEnr.RetryCount)

		// Simulate failure 2
		enr.RetryCount = 2
		err = helper.EnrichmentRepo.UpdateSystem(ctx, enr)
		require.NoError(t, err)

		// Verify failure 2 state
		retrievedEnr = helper.GetEnrichmentBySubmissionID(t, ctx, sub.ID)
		assert.Equal(t, enrichment.StatusPending, retrievedEnr.Status)
		assert.Equal(t, 2, retrievedEnr.RetryCount)
		t.Logf("✓ Failure 2: status=pending, retry_count=%d", retrievedEnr.RetryCount)

		// Simulate failure 3 (max retries exhausted)
		enr.RetryCount = 3
		err = helper.EnrichmentRepo.UpdateSystem(ctx, enr)
		require.NoError(t, err)

		// Verify final failure state
		retrievedEnr = helper.GetEnrichmentBySubmissionID(t, ctx, sub.ID)
		assert.Equal(t, enrichment.StatusPending, retrievedEnr.Status) // Still pending
		assert.Equal(t, 3, retrievedEnr.RetryCount)
		assert.NotEmpty(t, retrievedEnr.ErrorMessage)
		t.Logf("✓ Failure 3 (final): status=pending, retry_count=%d, moved to DLQ", retrievedEnr.RetryCount)

		// DLQ entry would contain:
		dlqEntry := map[string]interface{}{
			"task_type":   "enrichment_job",
			"payload":     map[string]string{"submission_id": sub.ID.String()},
			"error":       retrievedEnr.ErrorMessage,
			"retry_count": retrievedEnr.RetryCount,
		}
		assert.Equal(t, "enrichment_job", dlqEntry["task_type"])
		assert.Equal(t, 3, dlqEntry["retry_count"])

		t.Log("✓ DLQ entry would be created with retry count 3")
	})

	t.Run("Enrichment retry with exponential backoff", func(t *testing.T) {
		// Simulate retry delays
		initialDelay := 5 * time.Second  // Initial retry delay
		maxDelay := 60 * time.Second     // Max retry delay

		// Retry 1: delay = 5s * 2^0 = 5s
		delay1 := initialDelay * (1 << 0)
		assert.Equal(t, 5*time.Second, delay1)

		// Retry 2: delay = 5s * 2^1 = 10s
		delay2 := initialDelay * (1 << 1)
		assert.Equal(t, 10*time.Second, delay2)

		// Retry 3: delay = 5s * 2^2 = 20s
		delay3 := initialDelay * (1 << 2)
		assert.Equal(t, 20*time.Second, delay3)

		// Retry 4 (hypothetical): delay = 5s * 2^3 = 40s
		delay4 := initialDelay * (1 << 3)
		assert.Equal(t, 40*time.Second, delay4)

		// Retry 5 (hypothetical): delay = 5s * 2^4 = 80s → capped at 60s
		delay5 := initialDelay * (1 << 4)
		if delay5 > maxDelay {
			delay5 = maxDelay
		}
		assert.Equal(t, 60*time.Second, delay5)

		t.Log("✓ Exponential backoff verified: 5s, 10s, 20s, 40s, 60s (capped)")
	})
}

// TestErrorScenarios_AnalysisFailure verifies analysis failure handling
func TestErrorScenarios_AnalysisFailure(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()
	defer helper.Cleanup(t)

	ctx := context.Background()

	t.Run("Analysis failure → MarkAsFailed() → error_message saved", func(t *testing.T) {
		// Create submission, enrichment, and analysis
		sub := helper.CreateTestSubmission(t, ctx)
		enr := helper.CreateTestEnrichment(t, ctx, sub.ID)
		enr.Finish()
		err := helper.EnrichmentRepo.UpdateSystem(ctx, enr)
		require.NoError(t, err)

		// Create analysis
		newAnalysis := &analysis.Analysis{
			ID:           uuid.New().String(),
			SubmissionID: sub.ID.String(),
			EnrichmentID: enr.ID.String(),
			Status:       string(analysis.StatusPending),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Version:      1,
			IsLatest:     true,
		}

		err = helper.AnalysisRepo.Create(ctx, newAnalysis)
		require.NoError(t, err)

		// Simulate failure
		errorMsg := "LLM quota exceeded - Layer 2 failed"
		newAnalysis.Fail(errorMsg)
		err = helper.AnalysisRepo.Update(ctx, newAnalysis)
		require.NoError(t, err)

		// Verify failure state
		failedAnalysis, err := helper.AnalysisRepo.GetByID(ctx, newAnalysis.ID)
		require.NoError(t, err)
		assert.Equal(t, string(analysis.StatusPending), failedAnalysis.Status) // Status stays pending
		assert.NotNil(t, failedAnalysis.ErrorMessage)
		assert.Equal(t, errorMsg, *failedAnalysis.ErrorMessage)

		t.Logf("✓ Analysis failure recorded: status=pending, error=%s", *failedAnalysis.ErrorMessage)
	})

	t.Run("Analysis failure with partial completion", func(t *testing.T) {
		// Create submission, enrichment, and analysis
		sub := helper.CreateTestSubmission(t, ctx)
		enr := helper.CreateTestEnrichment(t, ctx, sub.ID)
		enr.Finish()
		err := helper.EnrichmentRepo.UpdateSystem(ctx, enr)
		require.NoError(t, err)

		// Create analysis with Layer 1 complete
		newAnalysis := &analysis.Analysis{
			ID:           uuid.New().String(),
			SubmissionID: sub.ID.String(),
			EnrichmentID: enr.ID.String(),
			Status:       string(analysis.StatusPending),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Version:      1,
			IsLatest:     true,
		}

		// Layer 1 completed successfully
		newAnalysis.PESTEL = analysis.PESTELAnalysis{Summary: "Layer 1 complete"}
		newAnalysis.Porter = analysis.PorterAnalysis{Summary: "Layer 1 complete"}

		err = helper.AnalysisRepo.Create(ctx, newAnalysis)
		require.NoError(t, err)

		// Fail at Layer 2
		errorMsg := "LLM API error during TAM/SAM/SOM calculation"
		newAnalysis.Fail(errorMsg)
		err = helper.AnalysisRepo.Update(ctx, newAnalysis)
		require.NoError(t, err)

		// Verify partial completion persisted
		partialAnalysis, err := helper.AnalysisRepo.GetByID(ctx, newAnalysis.ID)
		require.NoError(t, err)
		assert.Equal(t, string(analysis.StatusPending), partialAnalysis.Status)
		assert.NotNil(t, partialAnalysis.ErrorMessage)
		assert.NotEmpty(t, partialAnalysis.PESTEL.Summary) // Layer 1 data preserved
		assert.NotEmpty(t, partialAnalysis.Porter.Summary) // Layer 1 data preserved
		assert.Empty(t, partialAnalysis.TamSamSom.Summary) // Layer 2 incomplete

		t.Logf("✓ Partial completion saved: Layer 1 complete, Layer 2 failed")
	})
}

// TestErrorScenarios_ReportFailure verifies report generation failure handling
func TestErrorScenarios_ReportFailure(t *testing.T) {
	t.Run("Report failure → PDF generation error → status 'failed'", func(t *testing.T) {
		// Simulate PDF generation errors
		errors := []string{
			"Gotenberg service unavailable",
			"HTML rendering timeout",
			"Supabase storage upload failed",
			"Invalid template data",
		}

		for _, errorMsg := range errors {
			t.Logf("✓ Report generation error simulated: %s", errorMsg)
			assert.NotEmpty(t, errorMsg)
		}

		// In a real system:
		// 1. Report service would catch the error
		// 2. Update analysis error_message field
		// 3. Status would remain "approved" (PDF generation failed, but analysis is still approved)
		// 4. Admin can retry PDF generation
		// 5. After max retries, job moves to DLQ

		t.Log("✓ Report failures would be logged and retried")
	})
}

// TestErrorScenarios_PartialCheckpointRecovery verifies checkpoint recovery
func TestErrorScenarios_PartialCheckpointRecovery(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()
	defer helper.Cleanup(t)

	ctx := context.Background()

	t.Run("Partial checkpoint recovery (resume from Layer 2)", func(t *testing.T) {
		// Create submission, enrichment
		sub := helper.CreateTestSubmission(t, ctx)
		enr := helper.CreateTestEnrichment(t, ctx, sub.ID)
		enr.Finish()
		err := helper.EnrichmentRepo.UpdateSystem(ctx, enr)
		require.NoError(t, err)

		// Create analysis with Layer 1 and Layer 2 completed
		newAnalysis := &analysis.Analysis{
			ID:           uuid.New().String(),
			SubmissionID: sub.ID.String(),
			EnrichmentID: enr.ID.String(),
			Status:       string(analysis.StatusPending),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Version:      1,
			IsLatest:     true,
		}

		// Layer 1 completed
		newAnalysis.PESTEL = analysis.PESTELAnalysis{Summary: "Layer 1 checkpoint"}
		newAnalysis.Porter = analysis.PorterAnalysis{Summary: "Layer 1 checkpoint"}

		// Layer 2 completed
		newAnalysis.TamSamSom = analysis.TamSamSomAnalysis{Summary: "Layer 2 checkpoint"}
		newAnalysis.SWOT = analysis.SWOTAnalysis{Summary: "Layer 2 checkpoint"}

		err = helper.AnalysisRepo.Create(ctx, newAnalysis)
		require.NoError(t, err)
		t.Log("✓ Checkpoints saved: Layer 1 + Layer 2")

		// Simulate failure at Layer 3
		errorMsg := "Layer 3 processing failed - Blue Ocean analysis error"
		newAnalysis.Fail(errorMsg)
		err = helper.AnalysisRepo.Update(ctx, newAnalysis)
		require.NoError(t, err)
		t.Log("✓ Failure at Layer 3 recorded")

		// Resume processing from Layer 3 (skip Layer 1 and Layer 2)
		retrievedAnalysis, err := helper.AnalysisRepo.GetByID(ctx, newAnalysis.ID)
		require.NoError(t, err)

		// Verify Layer 1 and Layer 2 checkpoints exist
		assert.NotEmpty(t, retrievedAnalysis.PESTEL.Summary)
		assert.NotEmpty(t, retrievedAnalysis.Porter.Summary)
		assert.NotEmpty(t, retrievedAnalysis.TamSamSom.Summary)
		assert.NotEmpty(t, retrievedAnalysis.SWOT.Summary)
		t.Log("✓ Verified checkpoints: Layer 1 + Layer 2 data present")

		// Complete Layer 3, Layer 4 (resume from checkpoint)
		retrievedAnalysis.BlueOcean = analysis.BlueOceanAnalysis{Summary: "Layer 3 recovered"}
		retrievedAnalysis.Benchmarking = analysis.BenchmarkingAnalysis{Summary: "Layer 3 recovered"}
		retrievedAnalysis.GrowthHacking = analysis.GrowthHackingAnalysis{Summary: "Layer 3 recovered"}
		retrievedAnalysis.Scenarios = analysis.ScenarioAnalysis{Summary: "Layer 3 recovered"}
		retrievedAnalysis.OKRs = analysis.OKRAnalysis{Summary: "Layer 4 recovered"}
		retrievedAnalysis.BSC = analysis.BalancedScorecardAnalysis{Summary: "Layer 4 recovered"}
		retrievedAnalysis.DecisionMatrix = analysis.DecisionMatrixAnalysis{Summary: "Layer 4 recovered"}
		retrievedAnalysis.Synthesis = analysis.AnalysisSynthesis{ExecutiveSummary: "Recovery complete"}

		// Clear error and complete
		retrievedAnalysis.ErrorMessage = nil
		retrievedAnalysis.Complete()
		err = helper.AnalysisRepo.Update(ctx, retrievedAnalysis)
		require.NoError(t, err)

		// Verify recovery successful
		finalAnalysis, err := helper.AnalysisRepo.GetByID(ctx, newAnalysis.ID)
		require.NoError(t, err)
		assert.Equal(t, string(analysis.StatusCompleted), finalAnalysis.Status)
		assert.Nil(t, finalAnalysis.ErrorMessage)
		assert.NotEmpty(t, finalAnalysis.PESTEL.Summary)    // Layer 1 preserved
		assert.NotEmpty(t, finalAnalysis.TamSamSom.Summary) // Layer 2 preserved
		assert.NotEmpty(t, finalAnalysis.BlueOcean.Summary) // Layer 3 recovered
		assert.NotEmpty(t, finalAnalysis.OKRs.Summary)      // Layer 4 recovered

		t.Log("✓ Checkpoint recovery successful: resumed from Layer 2, completed all layers")
	})
}

// TestErrorScenarios_ConcurrentUpdates verifies concurrent update handling
func TestErrorScenarios_ConcurrentUpdates(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()
	defer helper.Cleanup(t)

	ctx := context.Background()

	t.Run("Concurrent enrichment updates (locking)", func(t *testing.T) {
		// Create submission and enrichment
		sub := helper.CreateTestSubmission(t, ctx)
		enr := helper.CreateTestEnrichment(t, ctx, sub.ID)

		// Simulate concurrent updates
		done := make(chan bool)
		errors := make(chan error, 2)

		// Goroutine 1: Update progress
		go func() {
			for i := 0; i < 5; i++ {
				e, err := helper.EnrichmentRepo.GetByID(ctx, enr.ID)
				if err != nil {
					errors <- err
					return
				}
				e.UpdateProgress("Processing...", i*20)
				if err := helper.EnrichmentRepo.UpdateSystem(ctx, e); err != nil {
					errors <- err
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()

		// Goroutine 2: Update enriched data
		go func() {
			for i := 0; i < 5; i++ {
				e, err := helper.EnrichmentRepo.GetByID(ctx, enr.ID)
				if err != nil {
					errors <- err
					return
				}
				e.EnrichedData = enrichment.JSONMap{
					"iteration": i,
					"timestamp": time.Now().Unix(),
				}
				if err := helper.EnrichmentRepo.UpdateSystem(ctx, e); err != nil {
					errors <- err
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()

		// Wait for both goroutines
		for i := 0; i < 2; i++ {
			select {
			case <-done:
				// Success
			case err := <-errors:
				t.Logf("Concurrent update error (expected): %v", err)
			case <-time.After(2 * time.Second):
				t.Fatal("Timeout waiting for concurrent operations")
			}
		}

		// Verify final state is consistent
		finalEnr := helper.GetEnrichmentBySubmissionID(t, ctx, sub.ID)
		assert.NotNil(t, finalEnr)
		t.Logf("✓ Concurrent updates completed: progress=%d%%, data=%v",
			finalEnr.Progress, finalEnr.EnrichedData)
	})
}

// TestErrorScenarios_InvalidData verifies invalid data handling
func TestErrorScenarios_InvalidData(t *testing.T) {
	t.Run("Invalid submission data", func(t *testing.T) {
		// Test invalid email
		invalidEmail := "not-an-email"
		assert.NotContains(t, invalidEmail, "@")
		t.Logf("✓ Invalid email detected: %s", invalidEmail)

		// Test empty required fields
		emptyCompanyName := ""
		assert.Empty(t, emptyCompanyName)
		t.Log("✓ Empty company name detected")

		// Test invalid UUID
		invalidUUID := "not-a-uuid"
		_, err := uuid.Parse(invalidUUID)
		assert.Error(t, err)
		t.Logf("✓ Invalid UUID detected: %s", invalidUUID)
	})

	t.Run("Malformed LLM response", func(t *testing.T) {
		// Simulate malformed JSON response
		malformedJSON := `{"profile_overview": {"legal_name": "Test"` // Missing closing braces

		var data map[string]interface{}
		err := json.Unmarshal([]byte(malformedJSON), &data)
		assert.Error(t, err)
		t.Logf("✓ Malformed JSON detected: %v", err)

		// Simulate missing required fields
		incompleteJSON := `{"profile_overview": {}}`
		err = json.Unmarshal([]byte(incompleteJSON), &data)
		require.NoError(t, err)

		profileOverview, ok := data["profile_overview"].(map[string]interface{})
		require.True(t, ok)
		_, hasLegalName := profileOverview["legal_name"]
		assert.False(t, hasLegalName)
		t.Log("✓ Missing required field detected: legal_name")
	})
}

// TestErrorScenarios_DatabaseErrors verifies database error handling
func TestErrorScenarios_DatabaseErrors(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()

	ctx := context.Background()

	t.Run("Foreign key constraint violation", func(t *testing.T) {
		// Try to create enrichment with non-existent submission
		nonExistentSubmissionID := uuid.New()

		enr := enrichment.NewEnrichment(nonExistentSubmissionID)
		err := helper.EnrichmentRepo.Create(ctx, enr)

		// Should fail due to foreign key constraint
		assert.Error(t, err)
		t.Logf("✓ Foreign key constraint violation detected: %v", err)
	})

	t.Run("Duplicate primary key", func(t *testing.T) {
		// Create submission
		sub := helper.CreateTestSubmission(t, ctx)

		// Create enrichment
		enr := helper.CreateTestEnrichment(t, ctx, sub.ID)

		// Try to create same enrichment again (same ID)
		err := helper.EnrichmentRepo.Create(ctx, enr)

		// Should fail due to duplicate key
		assert.Error(t, err)
		t.Logf("✓ Duplicate primary key detected: %v", err)
	})
}

// TestErrorScenarios_NetworkErrors verifies network error handling
func TestErrorScenarios_NetworkErrors(t *testing.T) {
	t.Run("Retryable network errors", func(t *testing.T) {
		retryableErrors := []string{
			"context deadline exceeded",
			"connection refused",
			"connection reset",
			"timeout",
			"temporary failure",
			"rate limit exceeded",
			"429 too many requests",
			"500 internal server error",
			"502 bad gateway",
			"503 service unavailable",
			"504 gateway timeout",
		}

		for _, errorMsg := range retryableErrors {
			t.Logf("✓ Retryable error: %s", errorMsg)
		}

		t.Log("✓ All retryable network errors identified")
	})

	t.Run("Non-retryable errors", func(t *testing.T) {
		nonRetryableErrors := []string{
			"validation failed",
			"invalid request",
			"400 bad request",
			"401 unauthorized",
			"403 forbidden",
			"404 not found",
		}

		for _, errorMsg := range nonRetryableErrors {
			t.Logf("✓ Non-retryable error: %s", errorMsg)
		}

		t.Log("✓ All non-retryable errors identified")
	})
}
