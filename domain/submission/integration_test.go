//go:build integration

package submission_test

import (
	"context"
	"os"
	"testing"
	"time"

	"backend_v3/domain/submission"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestDB returns a database connection for integration tests.
// Set DATABASE_URL env var to your test database connection string.
// Example: DATABASE_URL=postgres://user:pass@localhost:5432/testdb?sslmode=disable
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

// withTxRollback runs a test function within a transaction that always rolls back.
// This keeps tests isolated and the database clean.
func withTxRollback(t *testing.T, db *sqlx.DB, fn func(tx *sqlx.Tx)) {
	t.Helper()

	tx, err := db.Beginx()
	require.NoError(t, err)
	defer tx.Rollback() // Always rollback - test isolation

	fn(tx)
}

// txRepository wraps a transaction to implement Repository interface
type txRepository struct {
	tx *sqlx.Tx
}

func (r *txRepository) Create(ctx context.Context, s *submission.Submission) error {
	query := `
		INSERT INTO submissions (
			id, company_name, cnpj, company_website, company_industry, company_size, company_location,
			contact_name, contact_email, contact_phone, contact_position,
			target_market, annual_revenue_min, annual_revenue_max, funding_stage,
			business_challenge, additional_notes, linkedin_url, twitter_handle,
			status, user_id, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		)
	`
	_, err := r.tx.ExecContext(ctx, query,
		s.ID, s.CompanyName, s.CNPJ, s.CompanyWebsite, s.CompanyIndustry, s.CompanySize,
		s.CompanyLocation, s.ContactName, s.ContactEmail, s.ContactPhone, s.ContactPosition,
		s.TargetMarket, s.AnnualRevenueMin, s.AnnualRevenueMax, s.FundingStage,
		s.BusinessChallenge, s.AdditionalNotes, s.LinkedInURL, s.TwitterHandle,
		s.Status, s.UserID, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *txRepository) GetByID(ctx context.Context, id uuid.UUID) (*submission.Submission, error) {
	var s submission.Submission
	err := r.tx.GetContext(ctx, &s, `SELECT * FROM submissions WHERE id = $1 AND deleted_at IS NULL`, id)
	return &s, err
}

func (r *txRepository) GetByEmail(ctx context.Context, email string) ([]*submission.Submission, error) {
	var subs []*submission.Submission
	err := r.tx.SelectContext(ctx, &subs, `SELECT * FROM submissions WHERE contact_email = $1 AND deleted_at IS NULL`, email)
	return subs, err
}

func (r *txRepository) List(ctx context.Context, opts *submission.ListOptions) ([]*submission.Submission, int, error) {
	return nil, 0, nil // Not needed for these tests
}

func (r *txRepository) Update(ctx context.Context, s *submission.Submission) error {
	return nil // Not needed for these tests
}

func (r *txRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.tx.ExecContext(ctx, `UPDATE submissions SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *txRepository) GetTotalCount(ctx context.Context) (int, error)     { return 0, nil }
func (r *txRepository) GetActiveCount(ctx context.Context) (int, error)    { return 0, nil }
func (r *txRepository) GetCompletedCount(ctx context.Context) (int, error) { return 0, nil }

func (r *txRepository) GetEnrichmentStatus(ctx context.Context, submissionID uuid.UUID) (*submission.EnrichmentStatusRow, error) {
	var row submission.EnrichmentStatusRow
	err := r.tx.GetContext(ctx, &row, `
		SELECT status, started_at, completed_at FROM enrichments
		WHERE submission_id = $1 ORDER BY created_at DESC LIMIT 1
	`, submissionID)
	return &row, err
}

func (r *txRepository) ReserveEnrichment(ctx context.Context, submissionID uuid.UUID) (bool, error) {
	result, err := r.tx.ExecContext(ctx, `
		INSERT INTO enrichments (id, submission_id, status, progress, current_step, created_at, updated_at)
		VALUES ($1, $2, 'pending', 0, 'Queued', NOW(), NOW())
		ON CONFLICT (submission_id) DO NOTHING
	`, uuid.New(), submissionID)
	if err != nil {
		return false, err
	}
	rows, _ := result.RowsAffected()
	return rows > 0, nil
}

// =============================================================================
// INTEGRATION TESTS
// =============================================================================

func TestIntegration_CreateSubmission_SavesToDB(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		repo := &txRepository{tx: tx}
		svc := submission.NewService(repo, nil)
		ctx := context.Background()

		// Create submission
		cnpj := "12.345.678/0001-90"
		sub := &submission.Submission{
			CompanyName:       "Integration Test Corp",
			CNPJ:              &cnpj,
			ContactName:       "Test User",
			ContactEmail:      "test@integration.com",
			BusinessChallenge: "Testing database integration",
		}

		created, err := svc.Create(ctx, sub)
		require.NoError(t, err, "Create should succeed")
		require.NotNil(t, created)

		// Verify it's actually in the database
		saved, err := repo.GetByID(ctx, created.ID)
		require.NoError(t, err, "GetByID should find the submission")

		assert.Equal(t, created.ID, saved.ID)
		assert.Equal(t, "Integration Test Corp", saved.CompanyName)
		assert.Equal(t, "12.345.678/0001-90", *saved.CNPJ)
		assert.Equal(t, "test@integration.com", saved.ContactEmail)
		assert.Equal(t, submission.StatusReceived, saved.Status)
	})
}

func TestIntegration_CreateSubmission_WithAllFields(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		repo := &txRepository{tx: tx}
		svc := submission.NewService(repo, nil)
		ctx := context.Background()

		// Create submission with ALL optional fields
		cnpj := "98.765.432/0001-10"
		website := "https://fulltest.com"
		industry := "Technology"
		size := "51-200"
		location := "São Paulo, Brazil"
		phone := "+55 11 99999-9999"
		position := "CTO"
		market := "B2B SaaS"
		revenueMin := 1000000.0
		revenueMax := 5000000.0
		funding := "Series A"
		notes := "Additional test notes"
		linkedin := "https://linkedin.com/company/test"
		twitter := "@testcompany"

		sub := &submission.Submission{
			CompanyName:       "Full Fields Corp",
			CNPJ:              &cnpj,
			CompanyWebsite:    &website,
			CompanyIndustry:   &industry,
			CompanySize:       &size,
			CompanyLocation:   &location,
			ContactName:       "Full Tester",
			ContactEmail:      "full@test.com",
			ContactPhone:      &phone,
			ContactPosition:   &position,
			TargetMarket:      &market,
			AnnualRevenueMin:  &revenueMin,
			AnnualRevenueMax:  &revenueMax,
			FundingStage:      &funding,
			BusinessChallenge: "Complete field testing",
			AdditionalNotes:   &notes,
			LinkedInURL:       &linkedin,
			TwitterHandle:     &twitter,
		}

		created, err := svc.Create(ctx, sub)
		require.NoError(t, err)

		// Verify all fields persisted
		saved, err := repo.GetByID(ctx, created.ID)
		require.NoError(t, err)

		assert.Equal(t, "Full Fields Corp", saved.CompanyName)
		assert.Equal(t, cnpj, *saved.CNPJ)
		assert.Equal(t, website, *saved.CompanyWebsite)
		assert.Equal(t, industry, *saved.CompanyIndustry)
		assert.Equal(t, size, *saved.CompanySize)
		assert.Equal(t, location, *saved.CompanyLocation)
		assert.Equal(t, phone, *saved.ContactPhone)
		assert.Equal(t, position, *saved.ContactPosition)
		assert.Equal(t, market, *saved.TargetMarket)
		assert.Equal(t, revenueMin, *saved.AnnualRevenueMin)
		assert.Equal(t, revenueMax, *saved.AnnualRevenueMax)
		assert.Equal(t, funding, *saved.FundingStage)
		assert.Equal(t, notes, *saved.AdditionalNotes)
		assert.Equal(t, linkedin, *saved.LinkedInURL)
		assert.Equal(t, twitter, *saved.TwitterHandle)
	})
}

func TestIntegration_CreateSubmission_NilOptionalFields(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		repo := &txRepository{tx: tx}
		svc := submission.NewService(repo, nil)
		ctx := context.Background()

		// Minimal submission - only required fields
		sub := &submission.Submission{
			CompanyName:       "Minimal Corp",
			ContactName:       "Min User",
			ContactEmail:      "min@test.com",
			BusinessChallenge: "Minimal test",
			// All optional fields are nil
		}

		created, err := svc.Create(ctx, sub)
		require.NoError(t, err, "Should handle nil optional fields")

		saved, err := repo.GetByID(ctx, created.ID)
		require.NoError(t, err)

		assert.Equal(t, "Minimal Corp", saved.CompanyName)
		assert.Nil(t, saved.CNPJ)
		assert.Nil(t, saved.CompanyWebsite)
		assert.Nil(t, saved.CompanyIndustry)
	})
}

func TestIntegration_ReserveEnrichment_CreatesRecord(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		repo := &txRepository{tx: tx}
		ctx := context.Background()

		// First create a submission
		subID := uuid.New()
		_, err := tx.ExecContext(ctx, `
			INSERT INTO submissions (id, company_name, contact_name, contact_email, business_challenge, status, created_at, updated_at)
			VALUES ($1, 'Test', 'Test', 'test@test.com', 'Test', 'received', NOW(), NOW())
		`, subID)
		require.NoError(t, err)

		// Reserve enrichment
		reserved, err := repo.ReserveEnrichment(ctx, subID)
		require.NoError(t, err)
		assert.True(t, reserved, "First reserve should succeed")

		// Verify enrichment exists
		status, err := repo.GetEnrichmentStatus(ctx, subID)
		require.NoError(t, err)
		assert.Equal(t, "pending", status.Status)

		// Second reserve should be idempotent (no new row)
		reserved2, err := repo.ReserveEnrichment(ctx, subID)
		require.NoError(t, err)
		assert.False(t, reserved2, "Second reserve should return false (already exists)")
	})
}

func TestIntegration_DeleteSubmission_SoftDeletes(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		repo := &txRepository{tx: tx}
		svc := submission.NewService(repo, nil)
		ctx := context.Background()

		// Create submission
		sub := &submission.Submission{
			CompanyName:       "Delete Test Corp",
			ContactName:       "Delete User",
			ContactEmail:      "delete@test.com",
			BusinessChallenge: "Testing soft delete",
		}

		created, err := svc.Create(ctx, sub)
		require.NoError(t, err)

		// Delete it
		err = repo.Delete(ctx, created.ID)
		require.NoError(t, err)

		// Verify it's not returned (soft deleted)
		_, err = repo.GetByID(ctx, created.ID)
		assert.Error(t, err, "Should not find soft-deleted submission")

		// But it still exists in DB with deleted_at set
		var deletedAt *time.Time
		err = tx.GetContext(ctx, &deletedAt, `SELECT deleted_at FROM submissions WHERE id = $1`, created.ID)
		require.NoError(t, err)
		assert.NotNil(t, deletedAt, "deleted_at should be set")
	})
}

func TestIntegration_GetByEmail_ReturnsMultiple(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	withTxRollback(t, db, func(tx *sqlx.Tx) {
		repo := &txRepository{tx: tx}
		svc := submission.NewService(repo, nil)
		ctx := context.Background()

		email := "multi@test.com"

		// Create 3 submissions with same email
		for i := 1; i <= 3; i++ {
			sub := &submission.Submission{
				CompanyName:       "Company " + string(rune('A'+i-1)),
				ContactName:       "User",
				ContactEmail:      email,
				BusinessChallenge: "Challenge",
			}
			_, err := svc.Create(ctx, sub)
			require.NoError(t, err)
		}

		// Query by email
		results, err := repo.GetByEmail(ctx, email)
		require.NoError(t, err)
		assert.Len(t, results, 3, "Should return all 3 submissions")
	})
}
