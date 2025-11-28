//go:build integration

package enrichment_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"backend_v3/domain/enrichment"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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
func createTestSubmission(t *testing.T, tx *sqlx.Tx) uuid.UUID {
	t.Helper()
	subID := uuid.New()
	_, err := tx.Exec(`
		INSERT INTO submissions (id, company_name, contact_name, contact_email, business_challenge, status, created_at, updated_at)
		VALUES ($1, 'Test Company', 'Test User', 'test@test.com', 'Test challenge', 'received', NOW(), NOW())
	`, subID)
	require.NoError(t, err)
	return subID
}

// txRepository wraps transaction for Repository interface
type txRepository struct {
	tx *sqlx.Tx
}

func (r *txRepository) Create(ctx context.Context, e *enrichment.Enrichment) error {
	query := `
		INSERT INTO enrichments (
			id, submission_id, company_id, status, progress, current_step, is_locked,
			sources_status, sources_used, data, started_at, completed_at,
			error_message, retry_count, max_retries, auto_trigger_analysis, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)
	`
	sourcesStatus, _ := e.SourcesStatus.Value()
	enrichedData, _ := e.EnrichedData.Value()

	_, err := r.tx.ExecContext(ctx, query,
		e.ID, e.SubmissionID, e.CompanyID, e.Status, e.Progress, e.CurrentStep, e.IsLocked,
		sourcesStatus, e.SourcesUsed, enrichedData, e.StartedAt, e.CompletedAt,
		e.ErrorMessage, e.RetryCount, e.MaxRetries, e.AutoTriggerAnalysis, e.CreatedAt, e.UpdatedAt,
	)
	return err
}

func (r *txRepository) GetByID(ctx context.Context, id uuid.UUID) (*enrichment.Enrichment, error) {
	var e enrichment.Enrichment
	err := r.tx.GetContext(ctx, &e, `SELECT * FROM enrichments WHERE id = $1`, id)
	return &e, err
}

func (r *txRepository) GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*enrichment.Enrichment, error) {
	var e enrichment.Enrichment
	err := r.tx.GetContext(ctx, &e, `
		SELECT * FROM enrichments WHERE submission_id = $1 ORDER BY created_at DESC LIMIT 1
	`, submissionID)
	return &e, err
}

func (r *txRepository) UpdateSystem(ctx context.Context, e *enrichment.Enrichment) error {
	sourcesStatus, _ := e.SourcesStatus.Value()
	enrichedData, _ := e.EnrichedData.Value()

	// CRITICAL: Must match production repository.go - includes AND is_locked = FALSE
	result, err := r.tx.ExecContext(ctx, `
		UPDATE enrichments SET
			status = $1, progress = $2, current_step = $3,
			sources_status = $4, sources_used = $5, data = $6,
			started_at = $7, completed_at = $8, error_message = $9,
			retry_count = $10, updated_at = $11
		WHERE id = $12
		AND is_locked = FALSE
	`,
		e.Status, e.Progress, e.CurrentStep,
		sourcesStatus, e.SourcesUsed, enrichedData,
		e.StartedAt, e.CompletedAt, e.ErrorMessage,
		e.RetryCount, time.Now(), e.ID,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("enrichment is locked by user or not found")
	}
	return nil
}

func (r *txRepository) UpdateUser(ctx context.Context, e *enrichment.Enrichment) error {
	enrichedData, _ := e.EnrichedData.Value()

	_, err := r.tx.ExecContext(ctx, `
		UPDATE enrichments SET
			data = $1, sources_used = $2, is_locked = TRUE, updated_at = $3
		WHERE id = $4 AND status NOT IN ('processing', 'approved')
	`, enrichedData, e.SourcesUsed, time.Now(), e.ID)
	return err
}

func (r *txRepository) ForceUpdateAndUnlock(ctx context.Context, e *enrichment.Enrichment) error {
	sourcesStatus, _ := e.SourcesStatus.Value()
	enrichedData, _ := e.EnrichedData.Value()

	_, err := r.tx.ExecContext(ctx, `
		UPDATE enrichments SET
			status = $1, progress = $2, current_step = $3,
			sources_status = $4, sources_used = $5, data = $6,
			started_at = $7, completed_at = $8, error_message = $9,
			retry_count = $10, is_locked = FALSE, updated_at = $11
		WHERE id = $12
	`,
		e.Status, e.Progress, e.CurrentStep,
		sourcesStatus, e.SourcesUsed, enrichedData,
		e.StartedAt, e.CompletedAt, e.ErrorMessage,
		e.RetryCount, time.Now(), e.ID,
	)
	return err
}

// =============================================================================
// INTEGRATION TESTS
// =============================================================================

func TestIntegration_CreateEnrichment_SavesToDB(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		// Create submission first (FK constraint)
		subID := createTestSubmission(t, tx)

		// Create enrichment
		e := enrichment.NewEnrichment(subID)
		err := repo.Create(ctx, e)
		require.NoError(t, err, "Create should succeed")

		// Verify it's in DB
		saved, err := repo.GetByID(ctx, e.ID)
		require.NoError(t, err)

		assert.Equal(t, e.ID, saved.ID)
		assert.Equal(t, subID, saved.SubmissionID)
		assert.Equal(t, enrichment.StatusPending, saved.Status)
		assert.Equal(t, 0, saved.Progress)
		assert.False(t, saved.IsLocked)
	})
}

func TestIntegration_GetBySubmissionID_ReturnsEnrichment(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		e := enrichment.NewEnrichment(subID)
		require.NoError(t, repo.Create(ctx, e))

		// Query by submission ID
		found, err := repo.GetBySubmissionID(ctx, subID)
		require.NoError(t, err)
		assert.Equal(t, e.ID, found.ID)
	})
}

func TestIntegration_UpdateSystem_UpdatesProgress(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		e := enrichment.NewEnrichment(subID)
		require.NoError(t, repo.Create(ctx, e))

		// Update progress (status stays "pending" while processing)
		e.Progress = 50
		e.CurrentStep = "Processing data..."
		e.Start()

		err := repo.UpdateSystem(ctx, e)
		require.NoError(t, err)

		// Verify update - status remains "pending" during processing
		saved, err := repo.GetByID(ctx, e.ID)
		require.NoError(t, err)
		assert.Equal(t, enrichment.StatusPending, saved.Status)
		assert.Equal(t, 50, saved.Progress)
		assert.Equal(t, "Processing data...", saved.CurrentStep)
		assert.NotNil(t, saved.StartedAt)
	})
}

func TestIntegration_UpdateUser_LocksEnrichment(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		e := enrichment.NewEnrichment(subID)
		require.NoError(t, repo.Create(ctx, e))

		// User updates data (should lock)
		e.EnrichedData = enrichment.JSONMap{"test": "data"}
		err := repo.UpdateUser(ctx, e)
		require.NoError(t, err)

		// Verify locked
		saved, err := repo.GetByID(ctx, e.ID)
		require.NoError(t, err)
		assert.True(t, saved.IsLocked, "Should be locked after user update")
		assert.Equal(t, "data", saved.EnrichedData["test"])
	})
}

func TestIntegration_ForceUpdateAndUnlock_UnlocksEnrichment(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		e := enrichment.NewEnrichment(subID)
		require.NoError(t, repo.Create(ctx, e))

		// Lock it first
		e.IsLocked = true
		_, err := tx.Exec(`UPDATE enrichments SET is_locked = TRUE WHERE id = $1`, e.ID)
		require.NoError(t, err)

		// Force unlock
		e.Status = enrichment.StatusCompleted
		e.Progress = 100
		e.Finish()
		err = repo.ForceUpdateAndUnlock(ctx, e)
		require.NoError(t, err)

		// Verify unlocked and completed
		saved, err := repo.GetByID(ctx, e.ID)
		require.NoError(t, err)
		assert.False(t, saved.IsLocked, "Should be unlocked")
		assert.Equal(t, enrichment.StatusCompleted, saved.Status)
		assert.Equal(t, 100, saved.Progress)
	})
}

func TestIntegration_JSONMap_SavesAndLoads(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		e := enrichment.NewEnrichment(subID)

		// Set complex JSON data
		e.EnrichedData = enrichment.JSONMap{
			"profile_overview": map[string]interface{}{
				"legal_name": "Test Corp",
				"website":    "https://test.com",
			},
			"market_position": map[string]interface{}{
				"sector":     "Technology",
				"competitors": []string{"Comp A", "Comp B"},
			},
		}
		e.SourcesStatus = enrichment.JSONMap{
			"web_scraper":    "success",
			"dns_validation": "success",
		}
		e.SourcesUsed = pq.StringArray{"web_scraper", "dns_validation", "ai_search"}

		require.NoError(t, repo.Create(ctx, e))

		// Retrieve and verify
		saved, err := repo.GetByID(ctx, e.ID)
		require.NoError(t, err)

		// Verify nested JSON
		profile, ok := saved.EnrichedData["profile_overview"].(map[string]interface{})
		require.True(t, ok, "profile_overview should be a map")
		assert.Equal(t, "Test Corp", profile["legal_name"])

		// Verify sources_status
		assert.Equal(t, "success", saved.SourcesStatus["web_scraper"])

		// Verify sources_used array
		assert.Len(t, saved.SourcesUsed, 3)
		assert.Contains(t, []string(saved.SourcesUsed), "ai_search")
	})
}

func TestIntegration_UniqueSubmissionID_Constraint(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)

		// Create first enrichment
		e1 := enrichment.NewEnrichment(subID)
		require.NoError(t, repo.Create(ctx, e1))

		// Try to create second enrichment for same submission
		e2 := enrichment.NewEnrichment(subID)
		err := repo.Create(ctx, e2)

		// Should fail due to UNIQUE constraint
		assert.Error(t, err, "Should fail - UNIQUE constraint on submission_id")
		assert.Contains(t, err.Error(), "duplicate key")
	})
}

func TestIntegration_StatusTransitions(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		e := enrichment.NewEnrichment(subID)
		require.NoError(t, repo.Create(ctx, e))

		// Initial state: pending
		assert.Equal(t, enrichment.StatusPending, e.Status)

		// Start processing (status stays pending)
		e.Start()
		require.NoError(t, repo.UpdateSystem(ctx, e))

		saved, _ := repo.GetByID(ctx, e.ID)
		assert.Equal(t, enrichment.StatusPending, saved.Status)
		assert.NotNil(t, saved.StartedAt)

		// Complete (pending -> completed)
		e.Finish()
		require.NoError(t, repo.ForceUpdateAndUnlock(ctx, e))

		saved, _ = repo.GetByID(ctx, e.ID)
		assert.Equal(t, enrichment.StatusCompleted, saved.Status)
		assert.Equal(t, 100, saved.Progress)
		assert.NotNil(t, saved.CompletedAt)
	})
}

func TestIntegration_ErrorMessage_SavesOnFailure(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		e := enrichment.NewEnrichment(subID)
		require.NoError(t, repo.Create(ctx, e))

		// Simulate failure (status stays "pending", error message is saved)
		e.Fail(assert.AnError)
		require.NoError(t, repo.UpdateSystem(ctx, e))

		saved, _ := repo.GetByID(ctx, e.ID)
		assert.Equal(t, enrichment.StatusPending, saved.Status) // Status stays pending
		assert.NotEmpty(t, saved.ErrorMessage)                  // But error is recorded
	})
}

// TestIntegration_UpdateSystem_RespectsLock verifies that the worker cannot
// overwrite user edits when the enrichment is locked.
// This is a CRITICAL test - if it fails, user data can be lost to race conditions.
func TestIntegration_UpdateSystem_RespectsLock(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		ctx := context.Background()
		repo := &txRepository{tx: tx}

		subID := createTestSubmission(t, tx)
		e := enrichment.NewEnrichment(subID)
		require.NoError(t, repo.Create(ctx, e))

		// User locks the enrichment by making an edit
		e.EnrichedData = enrichment.JSONMap{"user_edit": "important data"}
		require.NoError(t, repo.UpdateUser(ctx, e))

		// Verify it's locked
		saved, err := repo.GetByID(ctx, e.ID)
		require.NoError(t, err)
		assert.True(t, saved.IsLocked, "Enrichment should be locked after user edit")

		// Now simulate worker trying to update (should FAIL due to lock)
		e.Progress = 75
		e.CurrentStep = "Worker trying to overwrite..."
		e.EnrichedData = enrichment.JSONMap{"worker_data": "this should NOT overwrite user data"}

		err = repo.UpdateSystem(ctx, e)
		assert.Error(t, err, "UpdateSystem should fail when enrichment is locked")
		assert.Contains(t, err.Error(), "locked", "Error should mention lock")

		// Verify user's data is preserved
		final, err := repo.GetByID(ctx, e.ID)
		require.NoError(t, err)
		assert.Equal(t, "important data", final.EnrichedData["user_edit"], "User data should be preserved")
		assert.Nil(t, final.EnrichedData["worker_data"], "Worker data should NOT be present")
	})
}
