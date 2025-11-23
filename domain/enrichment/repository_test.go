package enrichment_test

import (
	"backend_v3/domain/enrichment"
	"context"
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// REPOSITORY TESTS WITH SQLMOCK
// ============================================================================

// TestCreate_Success tests creating an enrichment record
func TestCreate_Success(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := enrichment.NewRepository(sqlxDB)

	enrichID := uuid.New()
	submissionID := uuid.New()
	now := time.Now()

	e := &enrichment.Enrichment{
		ID:            enrichID,
		SubmissionID:  submissionID,
		Status:        enrichment.StatusPending,
		Progress:      0,
		CurrentStep:   "Queued for enrichment",
		IsLocked:      false,
		SourcesStatus: enrichment.JSONMap{},
		EnrichedData:  enrichment.JSONMap{},
		RetryCount:    0,
		MaxRetries:    3,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Expect INSERT query
	mock.ExpectExec(`INSERT INTO enrichments`).
		WithArgs(
			enrichID,
			submissionID,
			string(enrichment.StatusPending),
			0,
			"Queued for enrichment",
			false,
			sqlmock.AnyArg(), // sources_status JSON
			sqlmock.AnyArg(), // enriched_data JSON
			nil,              // started_at
			nil,              // completed_at
			"",               // error_message
			0,                // retry_count
			3,                // max_retries
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Execute
	err = repo.Create(context.Background(), e)

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetByID_Success tests retrieving enrichment by ID
func TestGetByID_Success(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := enrichment.NewRepository(sqlxDB)

	enrichID := uuid.New()
	submissionID := uuid.New()
	now := time.Now()

	sourcesJSON := []byte(`{"dns":"success","scraper":"success"}`)
	enrichedJSON := []byte(`{"profile_overview":{"legal_name":"Test Corp"}}`)

	// Expect SELECT query
	rows := sqlmock.NewRows([]string{
		"id", "submission_id", "status", "progress", "current_step", "is_locked",
		"sources_status", "enriched_data", "started_at", "completed_at",
		"error_message", "retry_count", "max_retries", "created_at", "updated_at",
	}).AddRow(
		enrichID, submissionID, string(enrichment.StatusFinished), 100,
		"Complete", false, sourcesJSON, enrichedJSON,
		now, now, "", 0, 3, now, now,
	)

	mock.ExpectQuery(`SELECT \* FROM enrichments WHERE id = \$1`).
		WithArgs(enrichID).
		WillReturnRows(rows)

	// Execute
	result, err := repo.GetByID(context.Background(), enrichID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, enrichID, result.ID)
	assert.Equal(t, submissionID, result.SubmissionID)
	assert.Equal(t, enrichment.StatusFinished, result.Status)
	assert.Equal(t, 100, result.Progress)

	// Verify JSONB data was scanned correctly
	assert.NotNil(t, result.SourcesStatus)
	assert.NotNil(t, result.EnrichedData)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetByID_NotFound tests handling of not found
func TestGetByID_NotFound(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := enrichment.NewRepository(sqlxDB)

	enrichID := uuid.New()

	// Expect SELECT query returning no rows
	mock.ExpectQuery(`SELECT \* FROM enrichments WHERE id = \$1`).
		WithArgs(enrichID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// Execute
	result, err := repo.GetByID(context.Background(), enrichID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "enrichment not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetBySubmissionID_ReturnsLatest tests ORDER BY created_at DESC LIMIT 1
func TestGetBySubmissionID_ReturnsLatest(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := enrichment.NewRepository(sqlxDB)

	submissionID := uuid.New()
	latestEnrichID := uuid.New()
	now := time.Now()

	sourcesJSON := []byte(`{}`)
	enrichedJSON := []byte(`{}`)

	// Expect SELECT query with ORDER BY and LIMIT
	rows := sqlmock.NewRows([]string{
		"id", "submission_id", "status", "progress", "current_step", "is_locked",
		"sources_status", "enriched_data", "started_at", "completed_at",
		"error_message", "retry_count", "max_retries", "created_at", "updated_at",
	}).AddRow(
		latestEnrichID, submissionID, string(enrichment.StatusFinished), 100,
		"Complete", false, sourcesJSON, enrichedJSON,
		now, now, "", 0, 3, now, now,
	)

	mock.ExpectQuery(`SELECT \* FROM enrichments WHERE submission_id = \$1 ORDER BY created_at DESC LIMIT 1`).
		WithArgs(submissionID).
		WillReturnRows(rows)

	// Execute
	result, err := repo.GetBySubmissionID(context.Background(), submissionID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, latestEnrichID, result.ID)
	assert.Equal(t, submissionID, result.SubmissionID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateSystem_OnlyWhenNotLocked tests that UpdateSystem includes is_locked = FALSE
func TestUpdateSystem_OnlyWhenNotLocked(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := enrichment.NewRepository(sqlxDB)

	enrichID := uuid.New()
	now := time.Now()

	e := &enrichment.Enrichment{
		ID:            enrichID,
		SubmissionID:  uuid.New(),
		Status:        enrichment.StatusFinished,
		Progress:      100,
		CurrentStep:   "Complete",
		IsLocked:      false,
		SourcesStatus: enrichment.JSONMap{"dns": "success"},
		EnrichedData:  enrichment.JSONMap{"test": "data"},
		StartedAt:     &now,
		CompletedAt:   &now,
		ErrorMessage:  "",
		RetryCount:    0,
		UpdatedAt:     now,
	}

	// Expect UPDATE with WHERE id = X AND is_locked = FALSE
	mock.ExpectExec(`UPDATE enrichments SET .+ WHERE id = .+ AND is_locked = FALSE`).
		WithArgs(
			string(enrichment.StatusFinished),
			100,
			"Complete",
			sqlmock.AnyArg(), // sources_status JSON
			sqlmock.AnyArg(), // enriched_data JSON
			sqlmock.AnyArg(), // started_at
			sqlmock.AnyArg(), // completed_at
			"",
			0,
			sqlmock.AnyArg(), // updated_at
			enrichID,
		).
		WillReturnResult(sqlmock.NewResult(0, 1)) // 1 row affected

	// Execute
	err = repo.UpdateSystem(context.Background(), e)

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateSystem_SkipsWhenLocked tests that locked records are not updated
func TestUpdateSystem_SkipsWhenLocked(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := enrichment.NewRepository(sqlxDB)

	enrichID := uuid.New()
	now := time.Now()

	e := &enrichment.Enrichment{
		ID:            enrichID,
		SubmissionID:  uuid.New(),
		Status:        enrichment.StatusFinished,
		Progress:      100,
		CurrentStep:   "Complete",
		IsLocked:      true, // Locked by user
		SourcesStatus: enrichment.JSONMap{},
		EnrichedData:  enrichment.JSONMap{},
		UpdatedAt:     now,
	}

	// Expect UPDATE but 0 rows affected (because is_locked = TRUE)
	mock.ExpectExec(`UPDATE enrichments SET .+ WHERE id = .+ AND is_locked = FALSE`).
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), enrichID,
		).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

	// Execute
	err = repo.UpdateSystem(context.Background(), e)

	// Assert
	// Should NOT return error even when 0 rows affected (by design)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateUser_LocksRecord tests that UpdateUser sets is_locked = TRUE
func TestUpdateUser_LocksRecord(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := enrichment.NewRepository(sqlxDB)

	enrichID := uuid.New()

	e := &enrichment.Enrichment{
		ID:            enrichID,
		SubmissionID:  uuid.New(),
		Status:        enrichment.StatusFinished,
		EnrichedData:  enrichment.JSONMap{"updated": "data"},
		IsLocked:      false, // Will be set to true
		UpdatedAt:     time.Now(),
	}

	// Expect UPDATE with is_locked = TRUE
	mock.ExpectExec(`UPDATE enrichments SET enriched_data = .+, is_locked = TRUE, updated_at = .+ WHERE id = .+`).
		WithArgs(
			sqlmock.AnyArg(), // enriched_data JSON
			sqlmock.AnyArg(), // updated_at
			enrichID,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Execute
	err = repo.UpdateUser(context.Background(), e)

	// Assert
	assert.NoError(t, err)
	assert.True(t, e.IsLocked) // Should be set to true by UpdateUser
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateUser_ReturnsErrorWhenNotFound tests error handling
func TestUpdateUser_ReturnsErrorWhenNotFound(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := enrichment.NewRepository(sqlxDB)

	enrichID := uuid.New()

	e := &enrichment.Enrichment{
		ID:           enrichID,
		SubmissionID: uuid.New(),
		EnrichedData: enrichment.JSONMap{},
		UpdatedAt:    time.Now(),
	}

	// Expect UPDATE but 0 rows affected
	mock.ExpectExec(`UPDATE enrichments SET`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), enrichID).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

	// Execute
	err = repo.UpdateUser(context.Background(), e)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "enrichment not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ============================================================================
// JSONB COLUMN TESTS
// ============================================================================

// TestJSONB_UpdateMergesData tests JSONB column updates
func TestJSONB_SourcesStatusUpdate(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := enrichment.NewRepository(sqlxDB)

	enrichID := uuid.New()

	sourcesStatus := enrichment.JSONMap{
		"dns":     "success",
		"scraper": "success",
		"ai":      "partial",
	}

	e := &enrichment.Enrichment{
		ID:            enrichID,
		SubmissionID:  uuid.New(),
		Status:        enrichment.StatusFinished,
		SourcesStatus: sourcesStatus,
		EnrichedData:  enrichment.JSONMap{},
		UpdatedAt:     time.Now(),
	}

	// Expect UPDATE with JSONB
	mock.ExpectExec(`UPDATE enrichments SET .+ WHERE id = .+ AND is_locked = FALSE`).
		WithArgs(
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			AnyJSONContaining(map[string]interface{}{"dns": "success"}), // sources_status
			sqlmock.AnyArg(), // enriched_data
			sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			enrichID,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Execute
	err = repo.UpdateSystem(context.Background(), e)

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestJSONB_EnrichedDataUpdate tests enriched_data JSONB updates
func TestJSONB_EnrichedDataUpdate(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := enrichment.NewRepository(sqlxDB)

	enrichID := uuid.New()

	enrichedData := enrichment.JSONMap{
		"profile_overview": map[string]interface{}{
			"legal_name": "Test Corporation",
			"website":    "https://test.com",
		},
		"market_position": map[string]interface{}{
			"sector": "Technology",
		},
	}

	e := &enrichment.Enrichment{
		ID:            enrichID,
		SubmissionID:  uuid.New(),
		Status:        enrichment.StatusFinished,
		SourcesStatus: enrichment.JSONMap{},
		EnrichedData:  enrichedData,
		UpdatedAt:     time.Now(),
	}

	// Expect UPDATE with nested JSONB
	mock.ExpectExec(`UPDATE enrichments SET .+ WHERE id = .+ AND is_locked = FALSE`).
		WithArgs(
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(), // sources_status
			AnyJSONContaining(map[string]interface{}{"profile_overview": map[string]interface{}{"legal_name": "Test Corporation"}}),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			enrichID,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Execute
	err = repo.UpdateSystem(context.Background(), e)

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ============================================================================
// CUSTOM SQLMOCK MATCHERS
// ============================================================================

// AnyJSONContaining is a custom matcher for JSONB fields
type anyJSONContaining struct {
	expected map[string]interface{}
}

func AnyJSONContaining(expected map[string]interface{}) sqlmock.Argument {
	return &anyJSONContaining{expected: expected}
}

func (a *anyJSONContaining) Match(v driver.Value) bool {
	bytes, ok := v.([]byte)
	if !ok {
		return false
	}

	var actual map[string]interface{}
	if err := json.Unmarshal(bytes, &actual); err != nil {
		return false
	}

	// Check if expected keys exist in actual
	for key, expectedValue := range a.expected {
		actualValue, exists := actual[key]
		if !exists {
			return false
		}

		// Deep comparison for nested maps
		if expectedMap, ok := expectedValue.(map[string]interface{}); ok {
			actualMap, ok := actualValue.(map[string]interface{})
			if !ok {
				return false
			}
			for k, v := range expectedMap {
				if actualMap[k] != v {
					return false
				}
			}
		} else if actualValue != expectedValue {
			return false
		}
	}

	return true
}
