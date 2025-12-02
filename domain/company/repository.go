package company

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines the interface for company data access
type Repository interface {
	// Core CRUD
	Create(ctx context.Context, company *Company) error
	GetByID(ctx context.Context, id uuid.UUID) (*Company, error)
	GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*Company, error)
	Update(ctx context.Context, company *Company) error

	// User companies
	GetUserCompanies(ctx context.Context, userID uuid.UUID) ([]*Company, error)
	ListAll(ctx context.Context, limit, offset int) ([]*Company, int, error)

	// CNPJ verification (company-level)
	GetVerifiedByCNPJ(ctx context.Context, cnpj string) (*Company, error)
	IsVerifiedCNPJExists(ctx context.Context, cnpj string) (bool, error)

	// Field-level verification
	GetFieldVerifications(ctx context.Context, companyID uuid.UUID) ([]*FieldVerification, error)
	VerifyField(ctx context.Context, companyID uuid.UUID, fieldName string, verifiedBy *uuid.UUID) error
	UnverifyField(ctx context.Context, companyID uuid.UUID, fieldName string) error
	VerifyAllFields(ctx context.Context, companyID uuid.UUID, verifiedBy *uuid.UUID) error
	UnverifyAllFields(ctx context.Context, companyID uuid.UUID) error
	GetVerifiedFieldNames(ctx context.Context, companyID uuid.UUID) ([]string, error)
	ExpireFieldVerifications(ctx context.Context, companyID *uuid.UUID) (int, error)

	// History tracking
	RecordHistory(ctx context.Context, entry *DataHistoryEntry) error
	GetHistory(ctx context.Context, companyID uuid.UUID) ([]*DataHistoryEntry, error)
	GetFieldHistory(ctx context.Context, companyID uuid.UUID, fieldName string) ([]*DataHistoryEntry, error)

	// Submission links
	LinkSubmission(ctx context.Context, link *CompanySubmission) error
	GetSubmissions(ctx context.Context, companyID uuid.UUID) ([]*CompanySubmission, error)

	// Re-enrich/Re-analyze support
	GetLastSubmissionForCompany(ctx context.Context, companyID uuid.UUID) (*LastSubmissionInfo, error)
	GetLastCompletedEnrichmentForCompany(ctx context.Context, companyID uuid.UUID) (*LastEnrichmentInfo, error)
	GetLatestEnrichmentStatus(ctx context.Context, companyID uuid.UUID) (*EnrichmentStatus, error)
	GetEnrichmentsHistory(ctx context.Context, companyID uuid.UUID) ([]*EnrichmentStatus, error)

	// Analysis history (for admin dashboard)
	GetAnalysesHistory(ctx context.Context, companyID uuid.UUID) ([]*AnalysisHistoryItem, error)

	// Transaction support
	WithTx(ctx context.Context, fn func(Repository) error) error
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
	tx *sqlx.Tx // Optional transaction
}

// NewRepository creates a new PostgreSQL-backed repository
func NewRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// querier returns either the transaction or the database connection
func (r *PostgresRepository) querier() sqlx.ExtContext {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// Create inserts a new company record
func (r *PostgresRepository) Create(ctx context.Context, company *Company) error {
	query := `
		INSERT INTO companies (
			id, name, cnpj, website,
			industry, company_size, location, target_market, funding_stage,
			annual_revenue_min, annual_revenue_max,
			foundation_year, legal_name, headquarters, sector, target_audience, value_proposition,
			employees_range, revenue_estimate, business_model, competitors, market_share_status,
			digital_maturity, strengths, weaknesses,
			linkedin_url, twitter_handle,
			is_verified, allowed_users, owner_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9,
			$10, $11,
			$12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22,
			$23, $24, $25,
			$26, $27,
			$28, $29, $30,
			$31, $32
		)
	`

	_, err := r.querier().ExecContext(ctx, query,
		company.ID, company.Name, company.CNPJ, company.Website,
		company.Industry, company.CompanySize, company.Location, company.TargetMarket, company.FundingStage,
		company.AnnualRevenueMin, company.AnnualRevenueMax,
		company.FoundationYear, company.LegalName, company.Headquarters, company.Sector, company.TargetAudience, company.ValueProposition,
		company.EmployeesRange, company.RevenueEstimate, company.BusinessModel, company.Competitors, company.MarketShareStatus,
		company.DigitalMaturity, company.Strengths, company.Weaknesses,
		company.LinkedInURL, company.TwitterHandle,
		company.IsVerified, company.AllowedUsers, company.OwnerID,
		company.CreatedAt, company.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create company: %w", err)
	}
	return nil
}

// GetByID retrieves a company by its ID
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Company, error) {
	query := `
		SELECT
			id, name, cnpj, website,
			industry, company_size, location, target_market, funding_stage,
			annual_revenue_min, annual_revenue_max,
			foundation_year, legal_name, headquarters, sector, target_audience, value_proposition,
			employees_range, revenue_estimate, business_model, competitors, market_share_status,
			digital_maturity, strengths, weaknesses,
			linkedin_url, twitter_handle,
			is_verified, allowed_users, owner_id,
			created_at, updated_at
		FROM companies
		WHERE id = $1
	`

	var company Company
	err := sqlx.GetContext(ctx, r.querier(), &company, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get company by ID: %w", err)
	}
	return &company, nil
}

// GetBySubmissionID retrieves the company linked to a submission
func (r *PostgresRepository) GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*Company, error) {
	query := `
		SELECT
			c.id, c.name, c.cnpj, c.website,
			c.industry, c.company_size, c.location, c.target_market, c.funding_stage,
			c.annual_revenue_min, c.annual_revenue_max,
			c.foundation_year, c.legal_name, c.headquarters, c.sector, c.target_audience, c.value_proposition,
			c.employees_range, c.revenue_estimate, c.business_model, c.competitors, c.market_share_status,
			c.digital_maturity, c.strengths, c.weaknesses,
			c.linkedin_url, c.twitter_handle,
			c.is_verified, c.allowed_users, c.owner_id,
			c.created_at, c.updated_at
		FROM companies c
		JOIN company_submissions cs ON cs.company_id = c.id
		WHERE cs.submission_id = $1
		  AND cs.is_primary = true
	`

	var company Company
	err := sqlx.GetContext(ctx, r.querier(), &company, query, submissionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get company by submission ID: %w", err)
	}
	return &company, nil
}

// Update updates an existing company record
func (r *PostgresRepository) Update(ctx context.Context, company *Company) error {
	company.UpdatedAt = time.Now()

	query := `
		UPDATE companies SET
			name = $2, cnpj = $3, website = $4,
			industry = $5, company_size = $6, location = $7, target_market = $8, funding_stage = $9,
			annual_revenue_min = $10, annual_revenue_max = $11,
			foundation_year = $12, legal_name = $13, headquarters = $14, sector = $15, target_audience = $16, value_proposition = $17,
			employees_range = $18, revenue_estimate = $19, business_model = $20, competitors = $21, market_share_status = $22,
			digital_maturity = $23, strengths = $24, weaknesses = $25,
			linkedin_url = $26, twitter_handle = $27,
			is_verified = $28, allowed_users = $29, owner_id = $30,
			updated_at = $31
		WHERE id = $1
	`

	_, err := r.querier().ExecContext(ctx, query,
		company.ID, company.Name, company.CNPJ, company.Website,
		company.Industry, company.CompanySize, company.Location, company.TargetMarket, company.FundingStage,
		company.AnnualRevenueMin, company.AnnualRevenueMax,
		company.FoundationYear, company.LegalName, company.Headquarters, company.Sector, company.TargetAudience, company.ValueProposition,
		company.EmployeesRange, company.RevenueEstimate, company.BusinessModel, company.Competitors, company.MarketShareStatus,
		company.DigitalMaturity, company.Strengths, company.Weaknesses,
		company.LinkedInURL, company.TwitterHandle,
		company.IsVerified, company.AllowedUsers, company.OwnerID,
		company.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update company: %w", err)
	}
	return nil
}

// RecordHistory inserts a data history entry
func (r *PostgresRepository) RecordHistory(ctx context.Context, entry *DataHistoryEntry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.ChangedAt.IsZero() {
		entry.ChangedAt = time.Now()
	}

	query := `
		INSERT INTO company_data_history (
			id, company_id, field_name, old_value, new_value,
			source, source_id, changed_by, changed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.querier().ExecContext(ctx, query,
		entry.ID, entry.CompanyID, entry.FieldName, entry.OldValue, entry.NewValue,
		entry.Source, entry.SourceID, entry.ChangedBy, entry.ChangedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to record history: %w", err)
	}
	return nil
}

// GetHistory retrieves all history entries for a company
func (r *PostgresRepository) GetHistory(ctx context.Context, companyID uuid.UUID) ([]*DataHistoryEntry, error) {
	query := `
		SELECT
			id, company_id, field_name, old_value, new_value,
			source, source_id, changed_by, changed_at
		FROM company_data_history
		WHERE company_id = $1
		ORDER BY changed_at DESC
	`

	var entries []*DataHistoryEntry
	err := sqlx.SelectContext(ctx, r.querier(), &entries, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get company history: %w", err)
	}
	return entries, nil
}

// GetFieldHistory retrieves history for a specific field
func (r *PostgresRepository) GetFieldHistory(ctx context.Context, companyID uuid.UUID, fieldName string) ([]*DataHistoryEntry, error) {
	query := `
		SELECT
			id, company_id, field_name, old_value, new_value,
			source, source_id, changed_by, changed_at
		FROM company_data_history
		WHERE company_id = $1 AND field_name = $2
		ORDER BY changed_at DESC
	`

	var entries []*DataHistoryEntry
	err := sqlx.SelectContext(ctx, r.querier(), &entries, query, companyID, fieldName)
	if err != nil {
		return nil, fmt.Errorf("failed to get field history: %w", err)
	}
	return entries, nil
}

// LinkSubmission creates a link between a company and a submission
func (r *PostgresRepository) LinkSubmission(ctx context.Context, link *CompanySubmission) error {
	if link.LinkedAt.IsZero() {
		link.LinkedAt = time.Now()
	}

	query := `
		INSERT INTO company_submissions (company_id, submission_id, is_primary, linked_at, linked_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (company_id, submission_id) DO UPDATE SET
			is_primary = EXCLUDED.is_primary,
			linked_at = EXCLUDED.linked_at,
			linked_by = EXCLUDED.linked_by
	`

	_, err := r.querier().ExecContext(ctx, query,
		link.CompanyID, link.SubmissionID, link.IsPrimary, link.LinkedAt, link.LinkedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to link submission: %w", err)
	}
	return nil
}

// GetSubmissions retrieves all submission links for a company
func (r *PostgresRepository) GetSubmissions(ctx context.Context, companyID uuid.UUID) ([]*CompanySubmission, error) {
	query := `
		SELECT company_id, submission_id, is_primary, linked_at, linked_by
		FROM company_submissions
		WHERE company_id = $1
		ORDER BY linked_at DESC
	`

	var links []*CompanySubmission
	err := sqlx.SelectContext(ctx, r.querier(), &links, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get company submissions: %w", err)
	}
	return links, nil
}

// GetVerifiedByCNPJ retrieves a verified company by its CNPJ
func (r *PostgresRepository) GetVerifiedByCNPJ(ctx context.Context, cnpj string) (*Company, error) {
	query := `
		SELECT
			id, name, cnpj, website,
			industry, company_size, location, target_market, funding_stage,
			annual_revenue_min, annual_revenue_max,
			foundation_year, legal_name, headquarters, sector, target_audience, value_proposition,
			employees_range, revenue_estimate, business_model, competitors, market_share_status,
			digital_maturity, strengths, weaknesses,
			linkedin_url, twitter_handle,
			is_verified, allowed_users, owner_id,
			created_at, updated_at
		FROM companies
		WHERE cnpj = $1 AND is_verified = true
	`

	var company Company
	err := sqlx.GetContext(ctx, r.querier(), &company, query, cnpj)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get verified company by CNPJ: %w", err)
	}
	return &company, nil
}

// IsVerifiedCNPJExists checks if a verified company with the given CNPJ exists
func (r *PostgresRepository) IsVerifiedCNPJExists(ctx context.Context, cnpj string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM companies WHERE cnpj = $1 AND is_verified = true)`

	var exists bool
	err := sqlx.GetContext(ctx, r.querier(), &exists, query, cnpj)
	if err != nil {
		return false, fmt.Errorf("failed to check verified CNPJ existence: %w", err)
	}
	return exists, nil
}

// WithTx executes the given function within a transaction
func (r *PostgresRepository) WithTx(ctx context.Context, fn func(Repository) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	txRepo := &PostgresRepository{db: r.db, tx: tx}

	if err := fn(txRepo); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	return nil
}

// GetLastSubmissionForCompany retrieves the most recent submission linked to this company
func (r *PostgresRepository) GetLastSubmissionForCompany(ctx context.Context, companyID uuid.UUID) (*LastSubmissionInfo, error) {
	query := `
		SELECT
			cs.submission_id,
			s.business_challenge,
			cs.linked_at
		FROM company_submissions cs
		JOIN submissions s ON s.id = cs.submission_id
		WHERE cs.company_id = $1
		ORDER BY cs.linked_at DESC
		LIMIT 1
	`

	var info LastSubmissionInfo
	err := sqlx.GetContext(ctx, r.querier(), &info, query, companyID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get last submission for company: %w", err)
	}
	return &info, nil
}

// GetLastCompletedEnrichmentForCompany retrieves the most recent completed enrichment for this company
// With new status model: pending → completed → failed (no more 'approved')
func (r *PostgresRepository) GetLastCompletedEnrichmentForCompany(ctx context.Context, companyID uuid.UUID) (*LastEnrichmentInfo, error) {
	// First try direct company_id link (new schema)
	query := `
		SELECT
			e.id as enrichment_id,
			e.submission_id,
			COALESCE(e.completed_at, e.updated_at) as completed_at
		FROM enrichments e
		WHERE e.company_id = $1
		  AND e.status = 'completed'
		ORDER BY COALESCE(e.completed_at, e.updated_at) DESC
		LIMIT 1
	`

	var info LastEnrichmentInfo
	err := sqlx.GetContext(ctx, r.querier(), &info, query, companyID)
	if err == nil {
		return &info, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get last completed enrichment for company: %w", err)
	}

	// Fallback: try via company_submissions link (old data before migration 027)
	fallbackQuery := `
		SELECT
			e.id as enrichment_id,
			e.submission_id,
			COALESCE(e.completed_at, e.updated_at) as completed_at
		FROM enrichments e
		JOIN company_submissions cs ON cs.submission_id = e.submission_id
		WHERE cs.company_id = $1
		  AND e.status = 'completed'
		ORDER BY COALESCE(e.completed_at, e.updated_at) DESC
		LIMIT 1
	`

	err = sqlx.GetContext(ctx, r.querier(), &info, fallbackQuery, companyID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get last completed enrichment for company: %w", err)
	}
	return &info, nil
}

// GetLatestEnrichmentStatus retrieves the most recent enrichment status for this company
// Shows latest enrichment regardless of status (pending, completed, failed)
// Uses a single query that checks both direct company_id link and company_submissions join
func (r *PostgresRepository) GetLatestEnrichmentStatus(ctx context.Context, companyID uuid.UUID) (*EnrichmentStatus, error) {
	// Combined query: first try direct link, then via company_submissions
	// This handles both new enrichments (with company_id) and old ones (via join table)
	query := `
		SELECT
			e.id as enrichment_id,
			e.submission_id,
			e.status,
			e.progress,
			COALESCE(e.current_step, '') as current_step,
			e.started_at,
			e.completed_at,
			COALESCE(e.error_message, '') as error_message,
			e.created_at,
			e.updated_at
		FROM enrichments e
		LEFT JOIN company_submissions cs ON cs.submission_id = e.submission_id
		WHERE e.company_id = $1 OR cs.company_id = $1
		ORDER BY e.created_at DESC
		LIMIT 1
	`

	var status EnrichmentStatus
	err := sqlx.GetContext(ctx, r.querier(), &status, query, companyID)
	if err == nil {
		return &status, nil
	}
	if err == sql.ErrNoRows {
		return nil, nil // No enrichment found
	}
	return nil, fmt.Errorf("failed to get latest enrichment status: %w", err)
}

// GetEnrichmentsHistory retrieves all enrichments for a company (for admin dashboard history)
// Returns enrichments ordered by created_at DESC (newest first)
func (r *PostgresRepository) GetEnrichmentsHistory(ctx context.Context, companyID uuid.UUID) ([]*EnrichmentStatus, error) {
	query := `
		SELECT
			e.id as enrichment_id,
			e.submission_id,
			e.status,
			e.progress,
			COALESCE(e.current_step, '') as current_step,
			e.started_at,
			e.completed_at,
			COALESCE(e.error_message, '') as error_message,
			e.created_at,
			e.updated_at
		FROM enrichments e
		LEFT JOIN company_submissions cs ON cs.submission_id = e.submission_id
		WHERE e.company_id = $1 OR cs.company_id = $1
		ORDER BY e.created_at DESC
	`

	var enrichments []*EnrichmentStatus
	err := sqlx.SelectContext(ctx, r.querier(), &enrichments, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get enrichments history: %w", err)
	}
	return enrichments, nil
}

// GetUserCompanies retrieves all companies where user is owner OR in allowed_users
func (r *PostgresRepository) GetUserCompanies(ctx context.Context, userID uuid.UUID) ([]*Company, error) {
	query := `
		SELECT
			id, name, cnpj, website,
			industry, company_size, location, target_market, funding_stage,
			annual_revenue_min, annual_revenue_max,
			foundation_year, legal_name, headquarters, sector, target_audience, value_proposition,
			employees_range, revenue_estimate, business_model, competitors, market_share_status,
			digital_maturity, strengths, weaknesses,
			linkedin_url, twitter_handle,
			is_verified, allowed_users, owner_id,
			created_at, updated_at
		FROM companies
		WHERE owner_id = $1 OR $1 = ANY(allowed_users)
		ORDER BY updated_at DESC
	`

	var companies []*Company
	err := sqlx.SelectContext(ctx, r.querier(), &companies, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user companies: %w", err)
	}
	return companies, nil
}

// ListAll retrieves all companies with pagination (for admin)
func (r *PostgresRepository) ListAll(ctx context.Context, limit, offset int) ([]*Company, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM companies`
	var total int
	err := sqlx.GetContext(ctx, r.querier(), &total, countQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count companies: %w", err)
	}

	// Get paginated results
	query := `
		SELECT
			id, name, cnpj, website,
			industry, company_size, location, target_market, funding_stage,
			annual_revenue_min, annual_revenue_max,
			foundation_year, legal_name, headquarters, sector, target_audience, value_proposition,
			employees_range, revenue_estimate, business_model, competitors, market_share_status,
			digital_maturity, strengths, weaknesses,
			linkedin_url, twitter_handle,
			is_verified, allowed_users, owner_id,
			created_at, updated_at
		FROM companies
		ORDER BY updated_at DESC
		LIMIT $1 OFFSET $2
	`

	var companies []*Company
	err = sqlx.SelectContext(ctx, r.querier(), &companies, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list companies: %w", err)
	}
	return companies, total, nil
}

// GetAnalysesHistory retrieves all analyses for a company with their associated challenges
// Returns analyses ordered by creation date (newest first)
func (r *PostgresRepository) GetAnalysesHistory(ctx context.Context, companyID uuid.UUID) ([]*AnalysisHistoryItem, error) {
	query := `
		SELECT
			a.id as analysis_id,
			a.submission_id,
			a.status,
			s.business_challenge,
			a.is_blurred,
			a.is_visible_to_user,
			a.is_public,
			a.access_code,
			a.pdf_url,
			a.completed_at,
			a.created_at,
			a.updated_at
		FROM analyses a
		JOIN submissions s ON s.id = a.submission_id
		LEFT JOIN company_submissions cs ON cs.submission_id = a.submission_id
		WHERE a.company_id = $1 OR cs.company_id = $1
		ORDER BY a.created_at DESC
	`

	var history []*AnalysisHistoryItem
	err := sqlx.SelectContext(ctx, r.querier(), &history, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get analyses history: %w", err)
	}
	return history, nil
}

// ============================================
// Field Verification Methods
// ============================================

// GetFieldVerifications retrieves all field verifications for a company
func (r *PostgresRepository) GetFieldVerifications(ctx context.Context, companyID uuid.UUID) ([]*FieldVerification, error) {
	query := `
		SELECT id, company_id, field_name, verified_at, verified_by
		FROM company_field_verifications
		WHERE company_id = $1
		ORDER BY field_name
	`

	var verifications []*FieldVerification
	err := sqlx.SelectContext(ctx, r.querier(), &verifications, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get field verifications: %w", err)
	}
	return verifications, nil
}

// VerifyField marks a specific field as verified (protected from re-enrichment)
func (r *PostgresRepository) VerifyField(ctx context.Context, companyID uuid.UUID, fieldName string, verifiedBy *uuid.UUID) error {
	query := `
		INSERT INTO company_field_verifications (company_id, field_name, verified_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (company_id, field_name) DO UPDATE SET
			verified_at = NOW(),
			verified_by = EXCLUDED.verified_by
	`

	_, err := r.querier().ExecContext(ctx, query, companyID, fieldName, verifiedBy)
	if err != nil {
		return fmt.Errorf("failed to verify field %s: %w", fieldName, err)
	}
	return nil
}

// UnverifyField removes verification from a specific field (allows re-enrichment)
func (r *PostgresRepository) UnverifyField(ctx context.Context, companyID uuid.UUID, fieldName string) error {
	query := `DELETE FROM company_field_verifications WHERE company_id = $1 AND field_name = $2`

	_, err := r.querier().ExecContext(ctx, query, companyID, fieldName)
	if err != nil {
		return fmt.Errorf("failed to unverify field %s: %w", fieldName, err)
	}
	return nil
}

// VerifyAllFields marks all fields as verified for a company
func (r *PostgresRepository) VerifyAllFields(ctx context.Context, companyID uuid.UUID, verifiedBy *uuid.UUID) error {
	// Insert each verifiable field individually using upsert
	query := `
		INSERT INTO company_field_verifications (company_id, field_name, verified_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (company_id, field_name) DO UPDATE SET
			verified_at = NOW(),
			verified_by = EXCLUDED.verified_by
	`

	for _, fieldName := range VerifiableFields {
		_, err := r.querier().ExecContext(ctx, query, companyID, fieldName, verifiedBy)
		if err != nil {
			return fmt.Errorf("failed to verify field %s: %w", fieldName, err)
		}
	}
	return nil
}

// UnverifyAllFields removes all field verifications for a company
func (r *PostgresRepository) UnverifyAllFields(ctx context.Context, companyID uuid.UUID) error {
	query := `DELETE FROM company_field_verifications WHERE company_id = $1`

	_, err := r.querier().ExecContext(ctx, query, companyID)
	if err != nil {
		return fmt.Errorf("failed to unverify all fields: %w", err)
	}
	return nil
}

// GetVerifiedFieldNames retrieves just the field names that are verified (for quick lookup)
func (r *PostgresRepository) GetVerifiedFieldNames(ctx context.Context, companyID uuid.UUID) ([]string, error) {
	query := `SELECT field_name FROM company_field_verifications WHERE company_id = $1`

	var fields []string
	err := sqlx.SelectContext(ctx, r.querier(), &fields, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get verified field names: %w", err)
	}
	return fields, nil
}

// ExpireFieldVerifications removes expired field verifications based on deprecation_months config
// If companyID is nil, checks all companies (for cron job usage)
func (r *PostgresRepository) ExpireFieldVerifications(ctx context.Context, companyID *uuid.UUID) (int, error) {
	// Use the database function if it exists, otherwise use direct query
	query := `
		WITH expired_fields AS (
			SELECT cfv.id
			FROM company_field_verifications cfv
			JOIN field_deprecation_config fdc ON cfv.field_name = fdc.field_name
			WHERE fdc.deprecation_months > 0
			AND cfv.verified_at < NOW() - (fdc.deprecation_months || ' months')::INTERVAL
			AND ($1::uuid IS NULL OR cfv.company_id = $1)
		)
		DELETE FROM company_field_verifications
		WHERE id IN (SELECT id FROM expired_fields)
	`

	result, err := r.querier().ExecContext(ctx, query, companyID)
	if err != nil {
		return 0, fmt.Errorf("failed to expire field verifications: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}
