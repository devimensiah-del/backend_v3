package submission_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"backend_v3/domain/submission"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HELPERS
// =============================================================================

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	return sqlx.NewDb(mockDB, "postgres"), mock
}

func allColumns() []string {
	return []string{
		"id", "company_name", "cnpj", "company_website", "company_industry", "company_size", "company_location",
		"contact_name", "contact_email", "contact_phone", "contact_position",
		"target_market", "annual_revenue_min", "annual_revenue_max", "funding_stage",
		"additional_notes", "linkedin_url", "twitter_handle",
		"user_id", "created_at", "updated_at", "deleted_at",
	}
}

func createTestSubmission() *submission.Submission {
	return &submission.Submission{
		ID:           uuid.New(),
		CompanyName:  "Test Corp",
		ContactName:  "John Doe",
		ContactEmail: "john@test.com",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// =============================================================================
// TEST: Create
// =============================================================================

func TestRepository_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		sub := createTestSubmission()
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO submissions`)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		repo := submission.NewRepository(db)
		err := repo.Create(context.Background(), sub)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO submissions`)).
			WillReturnError(errors.New("connection failed"))

		repo := submission.NewRepository(db)
		err := repo.Create(context.Background(), createTestSubmission())

		assert.Error(t, err)
	})
}

// =============================================================================
// TEST: GetByID
// =============================================================================

func TestRepository_GetByID(t *testing.T) {
	testID := uuid.New()

	t.Run("found", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		rows := sqlmock.NewRows(allColumns()).AddRow(
			testID, "Test Corp", nil, nil, nil, nil, nil,
			"John Doe", "john@test.com", nil, nil,
			nil, nil, nil, nil, nil, nil, nil,
			nil, time.Now(), time.Now(), nil,
		)
		mock.ExpectQuery(`SELECT .+ FROM submissions WHERE id = \$1`).
			WithArgs(testID).
			WillReturnRows(rows)

		repo := submission.NewRepository(db)
		result, err := repo.GetByID(context.Background(), testID)

		assert.NoError(t, err)
		assert.Equal(t, testID, result.ID)
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		mock.ExpectQuery(`SELECT .+ FROM submissions WHERE id = \$1`).
			WithArgs(testID).
			WillReturnError(sql.ErrNoRows)

		repo := submission.NewRepository(db)
		_, err := repo.GetByID(context.Background(), testID)

		assert.Error(t, err)
		assert.True(t, submission.IsNotFound(err))
	})
}

// =============================================================================
// TEST: List
// =============================================================================

func TestRepository_List(t *testing.T) {
	t.Run("with defaults", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		// Count query
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM submissions WHERE deleted_at IS NULL`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

		// List query
		rows := sqlmock.NewRows(allColumns())
		for i := 0; i < 10; i++ {
			rows.AddRow(uuid.New(), "Corp", nil, nil, nil, nil, nil,
				"Name", "email@test.com", nil, nil, nil, nil, nil, nil, nil, nil, nil,
				nil, time.Now(), time.Now(), nil)
		}
		mock.ExpectQuery(`SELECT .+ FROM submissions WHERE deleted_at IS NULL ORDER BY created_at DESC`).
			WillReturnRows(rows)

		repo := submission.NewRepository(db)
		result, total, err := repo.List(context.Background(), nil)

		assert.NoError(t, err)
		assert.Len(t, result, 10)
		assert.Equal(t, 10, total)
	})

	t.Run("rejects invalid orderBy (SQL injection)", func(t *testing.T) {
		db, _ := setupMockDB(t)
		defer db.Close()

		repo := submission.NewRepository(db)
		_, _, err := repo.List(context.Background(), &submission.ListOptions{
			OrderBy: "id; DROP TABLE submissions; --",
		})

		assert.Error(t, err)
		assert.True(t, submission.IsValidation(err))
	})
}

// =============================================================================
// TEST: Delete (Soft Delete)
// =============================================================================

func TestRepository_Delete(t *testing.T) {
	testID := uuid.New()

	t.Run("success", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE submissions SET deleted_at`)).
			WithArgs(sqlmock.AnyArg(), testID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		repo := submission.NewRepository(db)
		err := repo.Delete(context.Background(), testID)

		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE submissions SET deleted_at`)).
			WithArgs(sqlmock.AnyArg(), testID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		repo := submission.NewRepository(db)
		err := repo.Delete(context.Background(), testID)

		assert.True(t, submission.IsNotFound(err))
	})
}

// =============================================================================
// TEST: Transaction Support
// =============================================================================

func TestRepository_Transaction(t *testing.T) {
	t.Run("begin and commit", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO submissions`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		repo := submission.NewRepository(db)
		txRepo, err := repo.BeginTx(context.Background())
		require.NoError(t, err)

		err = txRepo.Create(context.Background(), createTestSubmission())
		assert.NoError(t, err)

		err = txRepo.Commit()
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("begin and rollback", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectRollback()

		repo := submission.NewRepository(db)
		txRepo, err := repo.BeginTx(context.Background())
		require.NoError(t, err)

		err = txRepo.Rollback()
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
