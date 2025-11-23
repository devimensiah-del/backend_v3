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

// ============================================================================
// TEST HELPERS
// ============================================================================

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	return sqlxDB, mock
}

func createTestSubmission() *submission.Submission {
	website := "https://test.com"
	userID := uuid.New()
	return &submission.Submission{
		ID:                uuid.New(),
		CompanyName:       "Test Corp",
		CompanyWebsite:    &website,
		ContactName:       "John Doe",
		ContactEmail:      "john@test.com",
		BusinessChallenge: "Need to scale",
		Status:            submission.StatusReceived,
		UserID:            &userID,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// ============================================================================
// TEST CREATE
// ============================================================================

func TestRepository_Create(t *testing.T) {
	tests := []struct {
		name      string
		sub       *submission.Submission
		mockSetup func(sqlmock.Sqlmock, *submission.Submission)
		wantErr   bool
	}{
		{
			name: "success - creates submission",
			sub:  createTestSubmission(),
			mockSetup: func(mock sqlmock.Sqlmock, sub *submission.Submission) {
				mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO submissions`)).
					WithArgs(
						sub.ID, sub.CompanyName, sub.CompanyWebsite, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
						sub.ContactName, sub.ContactEmail, sqlmock.AnyArg(), sqlmock.AnyArg(),
						sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
						sub.BusinessChallenge, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
						sub.Status, sub.UserID, sub.CreatedAt, sub.UpdatedAt,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "success - creates submission with nil UserID (public submission)",
			sub: func() *submission.Submission {
				sub := createTestSubmission()
				sub.UserID = nil // Public submission
				return sub
			}(),
			mockSetup: func(mock sqlmock.Sqlmock, sub *submission.Submission) {
				mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO submissions`)).
					WithArgs(
						sub.ID, sub.CompanyName, sub.CompanyWebsite, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
						sub.ContactName, sub.ContactEmail, sqlmock.AnyArg(), sqlmock.AnyArg(),
						sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
						sub.BusinessChallenge, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
						sub.Status, nil, sub.CreatedAt, sub.UpdatedAt,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "failure - database error",
			sub:  createTestSubmission(),
			mockSetup: func(mock sqlmock.Sqlmock, sub *submission.Submission) {
				mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO submissions`)).
					WillReturnError(errors.New("database connection failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db, mock := setupMockDB(t)
			defer db.Close()

			tt.mockSetup(mock, tt.sub)

			repo := submission.NewRepository(db)

			// Execute
			err := repo.Create(context.Background(), tt.sub)

			// Verify
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ============================================================================
// TEST GET BY ID
// ============================================================================

func TestRepository_GetByID(t *testing.T) {
	testID := uuid.New()

	tests := []struct {
		name      string
		id        uuid.UUID
		mockSetup func(sqlmock.Sqlmock)
		wantErr   bool
		wantNil   bool
	}{
		{
			name: "success - found submission",
			id:   testID,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "company_name", "company_website", "contact_name", "contact_email",
					"business_challenge", "status", "user_id", "created_at", "updated_at", "deleted_at",
				}).AddRow(
					testID, "Test Corp", "https://test.com", "John Doe", "john@test.com",
					"Need to scale", submission.StatusReceived, uuid.New(), time.Now(), time.Now(), nil,
				)

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM submissions WHERE id = $1 AND deleted_at IS NULL`)).
					WithArgs(testID).
					WillReturnRows(rows)
			},
			wantErr: false,
			wantNil: false,
		},
		{
			name: "failure - not found",
			id:   testID,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM submissions WHERE id = $1 AND deleted_at IS NULL`)).
					WithArgs(testID).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			wantNil: true,
		},
		{
			name: "success - soft deleted submission not returned (deleted_at IS NULL clause)",
			id:   testID,
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Simulates that deleted submissions are filtered by WHERE clause
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM submissions WHERE id = $1 AND deleted_at IS NULL`)).
					WithArgs(testID).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			wantNil: true,
		},
		{
			name: "failure - database error",
			id:   testID,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM submissions WHERE id = $1 AND deleted_at IS NULL`)).
					WithArgs(testID).
					WillReturnError(errors.New("database connection failed"))
			},
			wantErr: true,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db, mock := setupMockDB(t)
			defer db.Close()

			tt.mockSetup(mock)

			repo := submission.NewRepository(db)

			// Execute
			result, err := repo.GetByID(context.Background(), tt.id)

			// Verify
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.id, result.ID)
				assert.Equal(t, submission.StatusReceived, result.Status)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ============================================================================
// TEST GET BY EMAIL
// ============================================================================

func TestRepository_GetByEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		mockSetup func(sqlmock.Sqlmock)
		wantCount int
		wantErr   bool
	}{
		{
			name:  "success - multiple submissions found",
			email: "john@test.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "company_name", "contact_name", "contact_email", "business_challenge",
					"status", "created_at", "updated_at", "deleted_at",
				}).
					AddRow(uuid.New(), "Corp A", "John", "john@test.com", "Challenge A", submission.StatusReceived, time.Now(), time.Now(), nil).
					AddRow(uuid.New(), "Corp B", "John", "john@test.com", "Challenge B", submission.StatusReceived, time.Now(), time.Now(), nil).
					AddRow(uuid.New(), "Corp C", "John", "john@test.com", "Challenge C", submission.StatusReceived, time.Now(), time.Now(), nil)

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM submissions WHERE contact_email = $1 AND deleted_at IS NULL ORDER BY created_at DESC`)).
					WithArgs("john@test.com").
					WillReturnRows(rows)
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:  "success - none found",
			email: "nobody@test.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "company_name", "contact_name", "contact_email", "business_challenge",
					"status", "created_at", "updated_at", "deleted_at",
				})

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM submissions WHERE contact_email = $1 AND deleted_at IS NULL ORDER BY created_at DESC`)).
					WithArgs("nobody@test.com").
					WillReturnRows(rows)
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:  "success - soft deleted submissions excluded (deleted_at IS NULL)",
			email: "deleted@test.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				// No rows returned because deleted_at IS NULL filters them out
				rows := sqlmock.NewRows([]string{
					"id", "company_name", "contact_name", "contact_email", "business_challenge",
					"status", "created_at", "updated_at", "deleted_at",
				})

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM submissions WHERE contact_email = $1 AND deleted_at IS NULL ORDER BY created_at DESC`)).
					WithArgs("deleted@test.com").
					WillReturnRows(rows)
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:  "failure - database error",
			email: "error@test.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM submissions WHERE contact_email = $1 AND deleted_at IS NULL ORDER BY created_at DESC`)).
					WithArgs("error@test.com").
					WillReturnError(errors.New("database connection failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db, mock := setupMockDB(t)
			defer db.Close()

			tt.mockSetup(mock)

			repo := submission.NewRepository(db)

			// Execute
			result, err := repo.GetByEmail(context.Background(), tt.email)

			// Verify
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				// Result can be empty slice, not nil
				if tt.wantCount == 0 {
					assert.Empty(t, result)
				} else {
					assert.NotNil(t, result)
					assert.Len(t, result, tt.wantCount)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ============================================================================
// TEST LIST (with filters, sorting, pagination)
// ============================================================================

func TestRepository_List(t *testing.T) {
	receivedStatus := submission.StatusReceived
	email := "john@test.com"
	userID := uuid.New()

	tests := []struct {
		name      string
		opts      *submission.ListOptions
		mockSetup func(sqlmock.Sqlmock)
		wantCount int
		wantTotal int
		wantErr   bool
	}{
		{
			name: "success - default options (no filters)",
			opts: nil, // Will use defaults: Limit 50, Offset 0, OrderBy created_at DESC
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Count query
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(100)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM submissions WHERE deleted_at IS NULL`)).
					WillReturnRows(countRows)

				// List query
				rows := sqlmock.NewRows([]string{
					"id", "company_name", "contact_name", "contact_email", "business_challenge",
					"status", "created_at", "updated_at", "deleted_at",
				})
				for i := 0; i < 50; i++ {
					rows.AddRow(uuid.New(), "Corp", "Name", "email@test.com", "Challenge", submission.StatusReceived, time.Now(), time.Now(), nil)
				}

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM submissions WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`)).
					WithArgs(50, 0).
					WillReturnRows(rows)
			},
			wantCount: 50,
			wantTotal: 100,
			wantErr:   false,
		},
		{
			name: "success - with status filter",
			opts: &submission.ListOptions{
				Limit:   10,
				Offset:  0,
				Status:  &receivedStatus,
				OrderBy: "created_at",
				Order:   "DESC",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Count query with status filter
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(50)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM submissions WHERE deleted_at IS NULL AND status = $1`)).
					WithArgs(receivedStatus).
					WillReturnRows(countRows)

				// List query with status filter
				rows := sqlmock.NewRows([]string{
					"id", "company_name", "contact_name", "contact_email", "business_challenge",
					"status", "created_at", "updated_at", "deleted_at",
				})
				for i := 0; i < 10; i++ {
					rows.AddRow(uuid.New(), "Corp", "Name", "email@test.com", "Challenge", submission.StatusReceived, time.Now(), time.Now(), nil)
				}

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM submissions WHERE deleted_at IS NULL AND status = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`)).
					WithArgs(receivedStatus, 10, 0).
					WillReturnRows(rows)
			},
			wantCount: 10,
			wantTotal: 50,
			wantErr:   false,
		},
		{
			name: "success - with email filter",
			opts: &submission.ListOptions{
				Limit:   20,
				Offset:  0,
				Email:   &email,
				OrderBy: "company_name",
				Order:   "ASC",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Count query with email filter
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(5)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM submissions WHERE deleted_at IS NULL AND contact_email = $1`)).
					WithArgs(email).
					WillReturnRows(countRows)

				// List query with email filter and company_name ASC sort
				rows := sqlmock.NewRows([]string{
					"id", "company_name", "contact_name", "contact_email", "business_challenge",
					"status", "created_at", "updated_at", "deleted_at",
				})
				for i := 0; i < 5; i++ {
					rows.AddRow(uuid.New(), "Corp", "Name", email, "Challenge", submission.StatusReceived, time.Now(), time.Now(), nil)
				}

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM submissions WHERE deleted_at IS NULL AND contact_email = $1 ORDER BY company_name ASC LIMIT $2 OFFSET $3`)).
					WithArgs(email, 20, 0).
					WillReturnRows(rows)
			},
			wantCount: 5,
			wantTotal: 5,
			wantErr:   false,
		},
		{
			name: "success - with user_id filter",
			opts: &submission.ListOptions{
				Limit:   30,
				Offset:  10,
				UserID:  &userID,
				OrderBy: "updated_at",
				Order:   "ASC",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Count query with user_id filter
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(25)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM submissions WHERE deleted_at IS NULL AND user_id = $1`)).
					WithArgs(userID).
					WillReturnRows(countRows)

				// List query with user_id filter and pagination
				rows := sqlmock.NewRows([]string{
					"id", "company_name", "contact_name", "contact_email", "business_challenge",
					"status", "user_id", "created_at", "updated_at", "deleted_at",
				})
				for i := 0; i < 15; i++ {
					rows.AddRow(uuid.New(), "Corp", "Name", "email@test.com", "Challenge", submission.StatusReceived, userID, time.Now(), time.Now(), nil)
				}

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM submissions WHERE deleted_at IS NULL AND user_id = $1 ORDER BY updated_at ASC LIMIT $2 OFFSET $3`)).
					WithArgs(userID, 30, 10).
					WillReturnRows(rows)
			},
			wantCount: 15,
			wantTotal: 25,
			wantErr:   false,
		},
		{
			name: "success - multiple filters combined",
			opts: &submission.ListOptions{
				Limit:   10,
				Offset:  0,
				Status:  &receivedStatus,
				Email:   &email,
				UserID:  &userID,
				OrderBy: "status",
				Order:   "DESC",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Count query with all filters
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(3)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM submissions WHERE deleted_at IS NULL AND status = $1 AND contact_email = $2 AND user_id = $3`)).
					WithArgs(receivedStatus, email, userID).
					WillReturnRows(countRows)

				// List query with all filters
				rows := sqlmock.NewRows([]string{
					"id", "company_name", "contact_name", "contact_email", "business_challenge",
					"status", "user_id", "created_at", "updated_at", "deleted_at",
				})
				for i := 0; i < 3; i++ {
					rows.AddRow(uuid.New(), "Corp", "Name", email, "Challenge", submission.StatusReceived, userID, time.Now(), time.Now(), nil)
				}

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM submissions WHERE deleted_at IS NULL AND status = $1 AND contact_email = $2 AND user_id = $3 ORDER BY status DESC LIMIT $4 OFFSET $5`)).
					WithArgs(receivedStatus, email, userID, 10, 0).
					WillReturnRows(rows)
			},
			wantCount: 3,
			wantTotal: 3,
			wantErr:   false,
		},
		{
			name: "success - SQL injection prevention (invalid OrderBy ignored)",
			opts: &submission.ListOptions{
				Limit:   10,
				Offset:  0,
				OrderBy: "id; DROP TABLE submissions; --", // Malicious input
				Order:   "DESC",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Should fall back to safe default: created_at DESC
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(10)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM submissions WHERE deleted_at IS NULL`)).
					WillReturnRows(countRows)

				rows := sqlmock.NewRows([]string{
					"id", "company_name", "contact_name", "contact_email", "business_challenge",
					"status", "created_at", "updated_at", "deleted_at",
				})
				for i := 0; i < 10; i++ {
					rows.AddRow(uuid.New(), "Corp", "Name", "email@test.com", "Challenge", submission.StatusReceived, time.Now(), time.Now(), nil)
				}

				// Should use safe default created_at (not the malicious input)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM submissions WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`)).
					WithArgs(10, 0).
					WillReturnRows(rows)
			},
			wantCount: 10,
			wantTotal: 10,
			wantErr:   false,
		},
		{
			name: "failure - count query error",
			opts: &submission.ListOptions{Limit: 10, Offset: 0},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM submissions WHERE deleted_at IS NULL`)).
					WillReturnError(errors.New("database connection failed"))
			},
			wantErr: true,
		},
		{
			name: "failure - list query error",
			opts: &submission.ListOptions{Limit: 10, Offset: 0},
			mockSetup: func(mock sqlmock.Sqlmock) {
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(100)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM submissions WHERE deleted_at IS NULL`)).
					WillReturnRows(countRows)

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM submissions WHERE deleted_at IS NULL`)).
					WillReturnError(errors.New("database connection failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db, mock := setupMockDB(t)
			defer db.Close()

			tt.mockSetup(mock)

			repo := submission.NewRepository(db)

			// Execute
			result, total, err := repo.List(context.Background(), tt.opts)

			// Verify
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.wantCount)
				assert.Equal(t, tt.wantTotal, total)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ============================================================================
// TEST DELETE (Soft Delete)
// ============================================================================

func TestRepository_Delete(t *testing.T) {
	testID := uuid.New()

	tests := []struct {
		name      string
		id        uuid.UUID
		mockSetup func(sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name: "success - soft delete sets deleted_at timestamp",
			id:   testID,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE submissions SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`)).
					WithArgs(sqlmock.AnyArg(), testID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "failure - submission not found",
			id:   testID,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE submissions SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`)).
					WithArgs(sqlmock.AnyArg(), testID).
					WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected
			},
			wantErr: true,
		},
		{
			name: "failure - already deleted (deleted_at IS NULL prevents double delete)",
			id:   testID,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE submissions SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`)).
					WithArgs(sqlmock.AnyArg(), testID).
					WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected
			},
			wantErr: true,
		},
		{
			name: "failure - database error",
			id:   testID,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE submissions SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`)).
					WithArgs(sqlmock.AnyArg(), testID).
					WillReturnError(errors.New("database connection failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db, mock := setupMockDB(t)
			defer db.Close()

			tt.mockSetup(mock)

			repo := submission.NewRepository(db)

			// Execute
			err := repo.Delete(context.Background(), tt.id)

			// Verify
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ============================================================================
// TEST ANALYTICS METHODS
// ============================================================================

func TestRepository_GetTotalCount(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(sqlmock.Sqlmock)
		wantCount int
		wantErr   bool
	}{
		{
			name: "success - returns total count",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(150)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM submissions WHERE deleted_at IS NULL`)).
					WillReturnRows(rows)
			},
			wantCount: 150,
			wantErr:   false,
		},
		{
			name: "success - zero submissions",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM submissions WHERE deleted_at IS NULL`)).
					WillReturnRows(rows)
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "failure - database error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM submissions WHERE deleted_at IS NULL`)).
					WillReturnError(errors.New("database connection failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupMockDB(t)
			defer db.Close()

			tt.mockSetup(mock)

			repo := submission.NewRepository(db)

			result, err := repo.GetTotalCount(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetActiveCount(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(sqlmock.Sqlmock)
		wantCount int
		wantErr   bool
	}{
		{
			name: "success - returns active count",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(80)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(DISTINCT s.id) FROM submissions s LEFT JOIN analyses a`)).
					WillReturnRows(rows)
			},
			wantCount: 80,
			wantErr:   false,
		},
		{
			name: "success - zero active submissions",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(DISTINCT s.id) FROM submissions s LEFT JOIN analyses a`)).
					WillReturnRows(rows)
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "failure - database error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(DISTINCT s.id) FROM submissions s LEFT JOIN analyses a`)).
					WillReturnError(errors.New("database connection failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupMockDB(t)
			defer db.Close()

			tt.mockSetup(mock)

			repo := submission.NewRepository(db)

			result, err := repo.GetActiveCount(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetCompletedCount(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(sqlmock.Sqlmock)
		wantCount int
		wantErr   bool
	}{
		{
			name: "success - returns completed count",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(70)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(DISTINCT s.id) FROM submissions s INNER JOIN analyses a`)).
					WillReturnRows(rows)
			},
			wantCount: 70,
			wantErr:   false,
		},
		{
			name: "success - zero completed submissions",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(DISTINCT s.id) FROM submissions s INNER JOIN analyses a`)).
					WillReturnRows(rows)
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "failure - database error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(DISTINCT s.id) FROM submissions s INNER JOIN analyses a`)).
					WillReturnError(errors.New("database connection failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupMockDB(t)
			defer db.Close()

			tt.mockSetup(mock)

			repo := submission.NewRepository(db)

			result, err := repo.GetCompletedCount(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
