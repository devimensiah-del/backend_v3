package company

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"backend_v3/domain/enrichment"

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
	Delete(ctx context.Context, id uuid.UUID) error // Soft delete for saga rollback

	// User companies
	GetUserCompanies(ctx context.Context, userID uuid.UUID) ([]*Company, error)
	ListAll(ctx context.Context, limit, offset int) ([]*Company, int, error)

	// CNPJ duplicate prevention (same email cannot submit same CNPJ twice)
	CheckEmailCNPJExists(ctx context.Context, email string, cnpj string) (bool, error)
	FindByCNPJNormalized(ctx context.Context, cnpj string) (*Company, error) // Used by admin merge

	// Submission links
	LinkSubmission(ctx context.Context, link *CompanySubmission) error
	UnlinkSubmissions(ctx context.Context, companyID uuid.UUID) error // For saga rollback

	// CNPJ duplicate management
	GetDuplicatesForReview(ctx context.Context) ([]*CNPJDuplicateReview, error)
	CreateDuplicateReview(ctx context.Context, review *CNPJDuplicateReview) error
	MergeCompanies(ctx context.Context, targetID, sourceID, mergedBy uuid.UUID) error

	// Analysis history (for admin dashboard)
	GetAnalysesHistory(ctx context.Context, companyID uuid.UUID) ([]*AnalysisHistoryItem, error)

	// Enrichment status updates (legacy)
	SetEnrichmentProcessing(ctx context.Context, id uuid.UUID) error
	SetEnrichmentCompleted(ctx context.Context, id uuid.UUID, data *enrichment.EnrichedCompanyData) error
	SetEnrichmentFailed(ctx context.Context, id uuid.UUID, errMsg string) error

	// Enrichment merge functions (COALESCE for scalars, merge+dedupe for arrays)
	MergeStep1Data(ctx context.Context, id uuid.UUID, data *enrichment.Step1BasicInfo) error
	MergeStep2Data(ctx context.Context, id uuid.UUID, data *enrichment.Step2BusinessModel) error
	MergeStep3Data(ctx context.Context, id uuid.UUID, data *enrichment.Step3CompetitiveIntel) error
	MergeCNPJData(ctx context.Context, id uuid.UUID, data *enrichment.CNPJData) error

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

// Create inserts a new company record into company_core and creates empty step data records.
// The step data records are needed so MergeStep*Data functions can UPDATE them during enrichment.
func (r *PostgresRepository) Create(ctx context.Context, company *Company) error {
	// Insert into company_core (the actual table, not the companies view)
	coreQuery := `
		INSERT INTO company_core (
			id, name, cnpj, website,
			industry, company_size, location, target_market, funding_stage,
			annual_revenue_min, annual_revenue_max,
			enrichment_status, enrichment_completed_at, enrichment_error,
			allowed_users, owner_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9,
			$10, $11,
			$12, $13, $14,
			$15, $16,
			$17, $18
		)
	`

	_, err := r.querier().ExecContext(ctx, coreQuery,
		company.ID, company.Name, company.CNPJ, company.Website,
		company.Industry, company.CompanySize, company.Location, company.TargetMarket, company.FundingStage,
		company.AnnualRevenueMin, company.AnnualRevenueMax,
		company.EnrichmentStatus, company.EnrichmentCompletedAt, company.EnrichmentError,
		company.AllowedUsers, company.OwnerID,
		company.CreatedAt, company.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create company_core: %w", err)
	}

	// Create empty step data records so MergeStep*Data can UPDATE them
	step1Query := `INSERT INTO company_step1_data (company_id) VALUES ($1)`
	_, err = r.querier().ExecContext(ctx, step1Query, company.ID)
	if err != nil {
		return fmt.Errorf("failed to create company_step1_data: %w", err)
	}

	step2Query := `INSERT INTO company_step2_data (company_id) VALUES ($1)`
	_, err = r.querier().ExecContext(ctx, step2Query, company.ID)
	if err != nil {
		return fmt.Errorf("failed to create company_step2_data: %w", err)
	}

	step3Query := `INSERT INTO company_step3_data (company_id) VALUES ($1)`
	_, err = r.querier().ExecContext(ctx, step3Query, company.ID)
	if err != nil {
		return fmt.Errorf("failed to create company_step3_data: %w", err)
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
			main_products, recent_news, key_executives, opportunities, threats,
			strategic_challenges, competitor_details, competitive_advantage, market_share,
			tam_estimate, sam_estimate, som_estimate, company_history,
			customer_segments, pricing_model, unique_selling_points,
			geographic_regions, service_areas,
			industry_growth_rate, industry_trends, regulatory_context, market_concentration, enrichment_sources,
			trade_name, phone, email, cnae_primary, cnae_codes, capital_social, partners, cnpj_verified,
			linkedin_url, twitter_handle, instagram_url, facebook_url,
			enrichment_status, enrichment_completed_at, enrichment_error,
			allowed_users, owner_id,
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
			c.main_products, c.recent_news, c.key_executives, c.opportunities, c.threats,
			c.strategic_challenges, c.competitor_details, c.competitive_advantage, c.market_share,
			c.tam_estimate, c.sam_estimate, c.som_estimate, c.company_history,
			c.customer_segments, c.pricing_model, c.unique_selling_points,
			c.industry_growth_rate, c.industry_trends, c.regulatory_context, c.market_concentration, c.enrichment_sources,
			c.linkedin_url, c.twitter_handle,
			c.enrichment_status, c.enrichment_completed_at, c.enrichment_error,
			c.allowed_users, c.owner_id,
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

// Update updates an existing company record across all tables (company_core and step data tables).
// This is a FULL REPLACEMENT - used for admin edits where all fields are intentionally set.
// For enrichment updates, use MergeStep1Data/MergeStep2Data/MergeStep3Data instead.
func (r *PostgresRepository) Update(ctx context.Context, company *Company) error {
	company.UpdatedAt = time.Now()

	// Update company_core (core fields)
	coreQuery := `
		UPDATE company_core SET
			name = $2, cnpj = $3, website = $4,
			industry = $5, company_size = $6, location = $7, target_market = $8, funding_stage = $9,
			annual_revenue_min = $10, annual_revenue_max = $11,
			enrichment_status = $12, enrichment_completed_at = $13, enrichment_error = $14,
			allowed_users = $15, owner_id = $16,
			updated_at = $17
		WHERE id = $1
	`
	_, err := r.querier().ExecContext(ctx, coreQuery,
		company.ID, company.Name, company.CNPJ, company.Website,
		company.Industry, company.CompanySize, company.Location, company.TargetMarket, company.FundingStage,
		company.AnnualRevenueMin, company.AnnualRevenueMax,
		company.EnrichmentStatus, company.EnrichmentCompletedAt, company.EnrichmentError,
		company.AllowedUsers, company.OwnerID,
		company.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update company_core: %w", err)
	}

	// Update company_step1_data (step 1 enrichment fields)
	step1Query := `
		UPDATE company_step1_data SET
			legal_name = $2, trade_name = $3, foundation_year = $4, headquarters = $5,
			employees_range = $6, phone = $7, email = $8, cnae_primary = $9,
			cnae_codes = $10, capital_social = $11, partners = $12, cnpj_verified = $13,
			linkedin_url = $14, twitter_handle = $15, instagram_url = $16, facebook_url = $17,
			key_executives = $18, enrichment_sources = $19,
			updated_at = $20
		WHERE company_id = $1
	`
	_, err = r.querier().ExecContext(ctx, step1Query,
		company.ID,
		company.LegalName, company.TradeName, company.FoundationYear, company.Headquarters,
		company.EmployeesRange, company.Phone, company.Email, company.CNAEPrimary,
		company.CNAECodes, company.CapitalSocial, company.Partners, company.CNPJVerified,
		company.LinkedInURL, company.TwitterHandle, company.InstagramURL, company.FacebookURL,
		company.KeyExecutives, company.EnrichmentSources,
		company.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update company_step1_data: %w", err)
	}

	// Update company_step2_data (step 2 enrichment fields)
	step2Query := `
		UPDATE company_step2_data SET
			business_model = $2, sector = $3, pricing_model = $4,
			target_audience = $5, value_proposition = $6,
			main_products = $7, customer_segments = $8, unique_selling_points = $9,
			geographic_regions = $10, service_areas = $11,
			updated_at = $12
		WHERE company_id = $1
	`
	_, err = r.querier().ExecContext(ctx, step2Query,
		company.ID,
		company.BusinessModel, company.Sector, company.PricingModel,
		company.TargetAudience, company.ValueProposition,
		company.MainProducts, company.CustomerSegments, company.UniqueSellingPoints,
		company.GeographicRegions, company.ServiceAreas,
		company.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update company_step2_data: %w", err)
	}

	// Update company_step3_data (step 3 enrichment fields)
	step3Query := `
		UPDATE company_step3_data SET
			competitors = $2, competitor_details = $3, competitive_advantage = $4,
			market_share = $5, market_share_status = $6, market_concentration = $7,
			industry_growth_rate = $8, industry_trends = $9, regulatory_context = $10,
			strengths = $11, weaknesses = $12, opportunities = $13, threats = $14,
			strategic_challenges = $15, recent_news = $16,
			tam_estimate = $17, sam_estimate = $18, som_estimate = $19,
			updated_at = $20
		WHERE company_id = $1
	`
	_, err = r.querier().ExecContext(ctx, step3Query,
		company.ID,
		company.Competitors, company.CompetitorDetails, company.CompetitiveAdvantage,
		company.MarketShare, company.MarketShareStatus, company.MarketConcentration,
		company.IndustryGrowthRate, company.IndustryTrends, company.RegulatoryContext,
		company.Strengths, company.Weaknesses, company.Opportunities, company.Threats,
		company.StrategicChallenges, company.RecentNews,
		company.TAMEstimate, company.SAMEstimate, company.SOMEstimate,
		company.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update company_step3_data: %w", err)
	}

	return nil
}

// LinkSubmission creates a link between a company and a submission
func (r *PostgresRepository) LinkSubmission(ctx context.Context, link *CompanySubmission) error {
	if link.LinkedAt.IsZero() {
		link.LinkedAt = time.Now()
	}

	// Normalize email for consistent matching
	normalizedEmail := strings.ToLower(strings.TrimSpace(link.SubmitterEmail))

	query := `
		INSERT INTO company_submissions (company_id, submission_id, submitter_email, is_primary, linked_at, linked_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (company_id, submission_id) DO UPDATE SET
			is_primary = EXCLUDED.is_primary,
			linked_at = EXCLUDED.linked_at,
			linked_by = EXCLUDED.linked_by
	`

	_, err := r.querier().ExecContext(ctx, query,
		link.CompanyID, link.SubmissionID, normalizedEmail, link.IsPrimary, link.LinkedAt, link.LinkedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to link submission: %w", err)
	}
	return nil
}

// UnlinkSubmissions removes all submission links for a company (for saga rollback)
func (r *PostgresRepository) UnlinkSubmissions(ctx context.Context, companyID uuid.UUID) error {
	query := `DELETE FROM company_submissions WHERE company_id = $1`
	_, err := r.querier().ExecContext(ctx, query, companyID)
	if err != nil {
		return fmt.Errorf("failed to unlink submissions: %w", err)
	}
	return nil
}

// Delete performs a hard delete of a company (for saga rollback)
// Note: This is a hard delete because saga rollback needs to fully undo the creation.
// For normal operations, companies should not be deleted (edit in Supabase if needed).
// Deletes from company_core and cascade handles step tables.
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// First unlink any submissions to avoid FK constraint
	if err := r.UnlinkSubmissions(ctx, id); err != nil {
		return fmt.Errorf("failed to unlink submissions before delete: %w", err)
	}

	// Delete from company_core - cascade handles step tables
	query := `DELETE FROM company_core WHERE id = $1`
	result, err := r.querier().ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete company: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("company not found: %s", id)
	}

	return nil
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

// GetUserCompanies retrieves companies where user is owner OR in allowed_users
// Limited to MaxUserCompanies (100) to prevent memory issues with power users
func (r *PostgresRepository) GetUserCompanies(ctx context.Context, userID uuid.UUID) ([]*Company, error) {
	query := `
		SELECT
			id, name, cnpj, website,
			industry, company_size, location, target_market, funding_stage,
			annual_revenue_min, annual_revenue_max,
			foundation_year, legal_name, headquarters, sector, target_audience, value_proposition,
			employees_range, revenue_estimate, business_model, competitors, market_share_status,
			digital_maturity, strengths, weaknesses,
			main_products, recent_news, key_executives, opportunities, threats,
			strategic_challenges, competitor_details, competitive_advantage, market_share,
			tam_estimate, sam_estimate, som_estimate, company_history,
			customer_segments, pricing_model, unique_selling_points,
			geographic_regions, service_areas,
			industry_growth_rate, industry_trends, regulatory_context, market_concentration, enrichment_sources,
			trade_name, phone, email, cnae_primary, cnae_codes, capital_social, partners, cnpj_verified,
			linkedin_url, twitter_handle, instagram_url, facebook_url,
			enrichment_status, enrichment_completed_at, enrichment_error,
			allowed_users, owner_id,
			created_at, updated_at
		FROM companies
		WHERE owner_id = $1 OR $1 = ANY(allowed_users)
		ORDER BY updated_at DESC
		LIMIT $2
	`

	var companies []*Company
	err := sqlx.SelectContext(ctx, r.querier(), &companies, query, userID, MaxUserCompanies)
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
			main_products, recent_news, key_executives, opportunities, threats,
			strategic_challenges, competitor_details, competitive_advantage, market_share,
			tam_estimate, sam_estimate, som_estimate, company_history,
			customer_segments, pricing_model, unique_selling_points,
			industry_growth_rate, industry_trends, regulatory_context, market_concentration, enrichment_sources,
			trade_name, phone, email, cnae_primary, cnae_codes, capital_social, partners, cnpj_verified,
			linkedin_url, twitter_handle, instagram_url, facebook_url,
			enrichment_status, enrichment_completed_at, enrichment_error,
			allowed_users, owner_id,
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
			COALESCE(a.submission_id, '00000000-0000-0000-0000-000000000000'::uuid) as submission_id,
			a.status,
			COALESCE(ch.business_challenge, '') as business_challenge,
			a.is_blurred,
			a.is_visible_to_user,
			a.is_public,
			a.access_code,
			a.pdf_url,
			a.completed_at,
			a.created_at,
			a.updated_at
		FROM analyses a
		LEFT JOIN challenges ch ON ch.id = a.challenge_id
		WHERE a.company_id = $1
		ORDER BY a.created_at DESC
	`

	var history []*AnalysisHistoryItem
	err := sqlx.SelectContext(ctx, r.querier(), &history, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get analyses history: %w", err)
	}
	return history, nil
}

// SetEnrichmentProcessing marks company as currently enriching
// Updates company_core table directly
func (r *PostgresRepository) SetEnrichmentProcessing(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE company_core
		SET enrichment_status = $2,
		    enrichment_error = NULL,
		    updated_at = $3
		WHERE id = $1
	`
	_, err := r.querier().ExecContext(ctx, query, id, EnrichmentProcessing, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set enrichment processing: %w", err)
	}
	return nil
}

// SetEnrichmentCompleted updates company with enriched data and marks as completed.
// DEPRECATED: This method is legacy and should not be used. It attempts to update the
// 'companies' view which is now read-only. Use MergeStep1Data, MergeStep2Data, and
// MergeStep3Data instead for enrichment updates to the individual step tables.
func (r *PostgresRepository) SetEnrichmentCompleted(ctx context.Context, id uuid.UUID, data *enrichment.EnrichedCompanyData) error {
	now := time.Now()

	query := `
		UPDATE companies SET
			enrichment_status = $2,
			enrichment_completed_at = $3,
			enrichment_error = NULL,
			cnpj = COALESCE(cnpj, $4),
			website = COALESCE(website, $5),
			industry = COALESCE(industry, $6),
			company_size = COALESCE(company_size, $7),
			location = COALESCE(location, $8),
			target_market = COALESCE(target_market, $9),
			funding_stage = COALESCE(funding_stage, $10),
			annual_revenue_min = COALESCE(annual_revenue_min, $11),
			annual_revenue_max = COALESCE(annual_revenue_max, $12),
			foundation_year = COALESCE(foundation_year, $13),
			legal_name = COALESCE(legal_name, $14),
			headquarters = COALESCE(headquarters, $15),
			sector = COALESCE(sector, $16),
			target_audience = COALESCE(target_audience, $17),
			value_proposition = COALESCE(value_proposition, $18),
			employees_range = COALESCE(employees_range, $19),
			revenue_estimate = COALESCE(revenue_estimate, $20),
			business_model = COALESCE(business_model, $21),
			market_share_status = COALESCE(market_share_status, $22),
			digital_maturity = COALESCE(digital_maturity, $23),
			competitors = CASE WHEN competitors IS NULL OR jsonb_typeof(competitors) != 'array' OR jsonb_array_length(competitors) = 0 THEN $24 ELSE competitors END,
			strengths = CASE WHEN strengths IS NULL OR jsonb_typeof(strengths) != 'array' OR jsonb_array_length(strengths) = 0 THEN $25 ELSE strengths END,
			weaknesses = CASE WHEN weaknesses IS NULL OR jsonb_typeof(weaknesses) != 'array' OR jsonb_array_length(weaknesses) = 0 THEN $26 ELSE weaknesses END,
			-- Enrichment v3 fields
			main_products = CASE WHEN main_products IS NULL OR jsonb_typeof(main_products) != 'array' OR jsonb_array_length(main_products) = 0 THEN $27 ELSE main_products END,
			recent_news = CASE WHEN recent_news IS NULL OR jsonb_typeof(recent_news) != 'array' OR jsonb_array_length(recent_news) = 0 THEN $28 ELSE recent_news END,
			key_executives = CASE WHEN key_executives IS NULL OR jsonb_typeof(key_executives) != 'array' OR jsonb_array_length(key_executives) = 0 THEN $29 ELSE key_executives END,
			opportunities = CASE WHEN opportunities IS NULL OR jsonb_typeof(opportunities) != 'array' OR jsonb_array_length(opportunities) = 0 THEN $30 ELSE opportunities END,
			threats = CASE WHEN threats IS NULL OR jsonb_typeof(threats) != 'array' OR jsonb_array_length(threats) = 0 THEN $31 ELSE threats END,
			strategic_challenges = CASE WHEN strategic_challenges IS NULL OR jsonb_typeof(strategic_challenges) != 'array' OR jsonb_array_length(strategic_challenges) = 0 THEN $32 ELSE strategic_challenges END,
			competitor_details = CASE WHEN competitor_details IS NULL OR jsonb_typeof(competitor_details) != 'array' OR jsonb_array_length(competitor_details) = 0 THEN $33 ELSE competitor_details END,
			competitive_advantage = COALESCE(competitive_advantage, $34),
			market_share = COALESCE(market_share, $35),
			tam_estimate = COALESCE(tam_estimate, $36),
			sam_estimate = COALESCE(sam_estimate, $37),
			som_estimate = COALESCE(som_estimate, $38),
			company_history = COALESCE(company_history, $39),
			customer_segments = CASE WHEN customer_segments IS NULL OR jsonb_typeof(customer_segments) != 'array' OR jsonb_array_length(customer_segments) = 0 THEN $40 ELSE customer_segments END,
			pricing_model = COALESCE(pricing_model, $41),
			unique_selling_points = CASE WHEN unique_selling_points IS NULL OR jsonb_typeof(unique_selling_points) != 'array' OR jsonb_array_length(unique_selling_points) = 0 THEN $42 ELSE unique_selling_points END,
			-- Industry context
			industry_growth_rate = COALESCE(industry_growth_rate, $43),
			industry_trends = CASE WHEN industry_trends IS NULL OR jsonb_typeof(industry_trends) != 'array' OR jsonb_array_length(industry_trends) = 0 THEN $44 ELSE industry_trends END,
			regulatory_context = COALESCE(regulatory_context, $45),
			market_concentration = COALESCE(market_concentration, $46),
			enrichment_sources = CASE WHEN enrichment_sources IS NULL OR jsonb_typeof(enrichment_sources) != 'array' OR jsonb_array_length(enrichment_sources) = 0 THEN $47 ELSE enrichment_sources END,
			-- CNPJ Registry fields
			trade_name = COALESCE(trade_name, $48),
			phone = COALESCE(phone, $49),
			email = COALESCE(email, $50),
			cnae_primary = COALESCE(cnae_primary, $51),
			cnae_codes = CASE WHEN cnae_codes IS NULL OR jsonb_typeof(cnae_codes) != 'array' OR jsonb_array_length(cnae_codes) = 0 THEN $52 ELSE cnae_codes END,
			capital_social = COALESCE(capital_social, $53),
			partners = CASE WHEN partners IS NULL OR jsonb_typeof(partners) != 'array' OR jsonb_array_length(partners) = 0 THEN $54 ELSE partners END,
			cnpj_verified = CASE WHEN cnpj_verified = false THEN $55 ELSE cnpj_verified END,
			-- Social links
			linkedin_url = COALESCE(linkedin_url, $56),
			twitter_handle = COALESCE(twitter_handle, $57),
			instagram_url = COALESCE(instagram_url, $58),
			facebook_url = COALESCE(facebook_url, $59),
			updated_at = $60
		WHERE id = $1
	`

	// Convert string slices to JSONB for storage
	competitorsJSON, _ := json.Marshal(data.Competitors)
	strengthsJSON, _ := json.Marshal(data.Strengths)
	weaknessesJSON, _ := json.Marshal(data.Weaknesses)
	// Enrichment v3 arrays
	mainProductsJSON, _ := json.Marshal(data.MainProducts)
	recentNewsJSON, _ := json.Marshal(data.RecentNews)
	keyExecutivesJSON, _ := json.Marshal(data.KeyExecutives)
	opportunitiesJSON, _ := json.Marshal(data.Opportunities)
	threatsJSON, _ := json.Marshal(data.Threats)
	strategicChallengesJSON, _ := json.Marshal(data.StrategicChallenges)
	competitorDetailsJSON, _ := json.Marshal(data.CompetitorDetails)
	customerSegmentsJSON, _ := json.Marshal(data.CustomerSegments)
	uniqueSellingPointsJSON, _ := json.Marshal(data.UniqueSellingPts)
	// Industry context arrays
	industryTrendsJSON, _ := json.Marshal(data.IndustryTrends)
	sourcesJSON, _ := json.Marshal(data.Sources)
	// CNPJ Registry arrays
	cnaeCodesJSON, _ := json.Marshal(data.CNAECodes)
	partnersJSON, _ := json.Marshal(data.Partners)

	_, err := r.querier().ExecContext(ctx, query,
		id, EnrichmentCompleted, now,
		data.CNPJ, data.Website, data.Industry, data.CompanySize, data.Location,
		data.TargetMarket, data.FundingStage, data.AnnualRevenueMin, data.AnnualRevenueMax,
		data.FoundationYear, data.LegalName, data.Headquarters, data.Sector,
		data.TargetAudience, data.ValueProposition, data.EmployeesRange,
		data.RevenueEstimate, data.BusinessModel, data.MarketShareStatus,
		data.DigitalMaturity,
		competitorsJSON, strengthsJSON, weaknessesJSON,
		// Enrichment v3 fields
		mainProductsJSON, recentNewsJSON, keyExecutivesJSON, opportunitiesJSON, threatsJSON,
		strategicChallengesJSON, competitorDetailsJSON, data.CompetitiveAdvantage, data.MarketShare,
		data.TAMEstimate, data.SAMEstimate, data.SOMEstimate, data.CompanyHistory,
		customerSegmentsJSON, data.PricingModel, uniqueSellingPointsJSON,
		// Industry context
		data.IndustryGrowthRate, industryTrendsJSON, data.RegulatoryContext, data.MarketConcentration, sourcesJSON,
		// CNPJ Registry fields
		data.TradeName, data.Phone, data.Email, data.CNAEPrimary, cnaeCodesJSON, data.CapitalSocial, partnersJSON, data.CNPJVerified,
		// Social links
		data.LinkedInURL, data.TwitterHandle, data.InstagramURL, data.FacebookURL,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to set enrichment completed: %w", err)
	}
	return nil
}

// SetEnrichmentFailed marks company enrichment as failed with error message
// Updates company_core table directly
func (r *PostgresRepository) SetEnrichmentFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	query := `
		UPDATE company_core
		SET enrichment_status = $2,
		    enrichment_error = $3,
		    updated_at = $4
		WHERE id = $1
	`
	_, err := r.querier().ExecContext(ctx, query, id, EnrichmentFailed, errMsg, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set enrichment failed: %w", err)
	}
	return nil
}

// =============================================================================
// ENRICHMENT MERGE FUNCTIONS
// These functions merge enrichment data into company records using:
// - COALESCE for scalar fields (only update if NULL)
// - Array merge + deduplicate for array fields (append new items)
// =============================================================================

// MergeStep1Data merges Step 1 enrichment data into company record
// Uses COALESCE for scalars, array merge for arrays
// Updates company_step1_data table directly and enrichment_status in company_core
func (r *PostgresRepository) MergeStep1Data(ctx context.Context, id uuid.UUID, data *enrichment.Step1BasicInfo) error {
	now := time.Now()

	// Convert arrays to JSON for merge
	cnaeCodesJSON, _ := json.Marshal(data.CNAECodes)
	partnersJSON, _ := json.Marshal(data.Partners)
	keyExecutivesJSON, _ := json.Marshal(data.KeyExecutives)
	sourcesJSON, _ := json.Marshal(data.Sources)

	foundationYear := ""
	if data.FoundationYear != "" {
		foundationYear = data.FoundationYear.String()
	}

	// Update company_core for enrichment status and core fields (cnpj, website)
	coreQuery := `
		UPDATE company_core SET
			enrichment_status = $2,
			enrichment_completed_at = $3,
			enrichment_error = NULL,
			cnpj = COALESCE(cnpj, $4),
			website = COALESCE(website, $5),
			updated_at = $6
		WHERE id = $1
	`
	_, err := r.querier().ExecContext(ctx, coreQuery,
		id,
		EnrichmentCompleted, now,
		data.CNPJ, data.Website,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to update company_core: %w", err)
	}

	// Update company_step1_data with enrichment results
	step1Query := `
		UPDATE company_step1_data SET
			-- Scalar fields: COALESCE (only set if NULL)
			legal_name = COALESCE(legal_name, $2),
			trade_name = COALESCE(trade_name, $3),
			foundation_year = COALESCE(foundation_year, $4),
			headquarters = COALESCE(headquarters, $5),
			employees_range = COALESCE(employees_range, $6),
			phone = COALESCE(phone, $7),
			email = COALESCE(email, $8),
			cnae_primary = COALESCE(cnae_primary, $9),
			capital_social = COALESCE(capital_social, $10),
			linkedin_url = COALESCE(linkedin_url, $11),
			twitter_handle = COALESCE(twitter_handle, $12),
			instagram_url = COALESCE(instagram_url, $13),
			facebook_url = COALESCE(facebook_url, $14),
			-- Boolean: only set to true, never back to false
			cnpj_verified = CASE WHEN $15 = true THEN true ELSE cnpj_verified END,
			-- Array fields: merge + deduplicate (with type safety check)
			cnae_codes = CASE
				WHEN $16::jsonb IS NULL OR jsonb_typeof($16::jsonb) != 'array' OR jsonb_array_length($16::jsonb) = 0 THEN COALESCE(cnae_codes, '[]'::jsonb)
				WHEN cnae_codes IS NULL OR jsonb_typeof(cnae_codes) != 'array' OR jsonb_array_length(cnae_codes) = 0 THEN $16::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(cnae_codes || $16::jsonb))
			END,
			partners = CASE
				WHEN $17::jsonb IS NULL OR jsonb_typeof($17::jsonb) != 'array' OR jsonb_array_length($17::jsonb) = 0 THEN COALESCE(partners, '[]'::jsonb)
				WHEN partners IS NULL OR jsonb_typeof(partners) != 'array' OR jsonb_array_length(partners) = 0 THEN $17::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(partners || $17::jsonb))
			END,
			key_executives = CASE
				WHEN $18::jsonb IS NULL OR jsonb_typeof($18::jsonb) != 'array' OR jsonb_array_length($18::jsonb) = 0 THEN COALESCE(key_executives, '[]'::jsonb)
				WHEN key_executives IS NULL OR jsonb_typeof(key_executives) != 'array' OR jsonb_array_length(key_executives) = 0 THEN $18::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(key_executives || $18::jsonb))
			END,
			enrichment_sources = CASE
				WHEN $19::jsonb IS NULL OR jsonb_typeof($19::jsonb) != 'array' OR jsonb_array_length($19::jsonb) = 0 THEN COALESCE(enrichment_sources, '[]'::jsonb)
				WHEN enrichment_sources IS NULL OR jsonb_typeof(enrichment_sources) != 'array' OR jsonb_array_length(enrichment_sources) = 0 THEN $19::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(enrichment_sources || $19::jsonb))
			END,
			confidence_score = $20,
			completed_at = $21,
			updated_at = $22
		WHERE company_id = $1
	`

	_, err = r.querier().ExecContext(ctx, step1Query,
		id,
		data.LegalName, data.TradeName, nullIfEmpty(foundationYear),
		data.Headquarters, data.EmployeesRange,
		data.Phone, data.Email, data.CNAEPrimary, data.CapitalSocial,
		data.LinkedInURL, data.TwitterHandle, data.InstagramURL, data.FacebookURL,
		data.CNPJVerified,
		string(cnaeCodesJSON), string(partnersJSON), string(keyExecutivesJSON), string(sourcesJSON),
		data.ConfidenceScore, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to merge step1 data: %w", err)
	}
	return nil
}

// MergeStep2Data merges Step 2 enrichment data into company record
// Uses COALESCE for scalars, array merge for arrays
// Updates company_step2_data table directly
func (r *PostgresRepository) MergeStep2Data(ctx context.Context, id uuid.UUID, data *enrichment.Step2BusinessModel) error {
	now := time.Now()

	// Convert arrays to JSON for merge
	mainProductsJSON, _ := json.Marshal(data.MainProducts)
	customerSegmentsJSON, _ := json.Marshal(data.CustomerSegments)
	uniqueSellingPointsJSON, _ := json.Marshal(data.UniqueSellingPoints)
	geographicRegionsJSON, _ := json.Marshal(data.GeographicRegions)
	serviceAreasJSON, _ := json.Marshal(data.ServiceAreas)
	sourcesJSON, _ := json.Marshal(data.Sources)

	// Update company_core for industry and target_market (core fields)
	coreQuery := `
		UPDATE company_core SET
			industry = COALESCE(industry, $2),
			target_market = COALESCE(target_market, $3),
			updated_at = $4
		WHERE id = $1
	`
	_, err := r.querier().ExecContext(ctx, coreQuery,
		id,
		data.Industry, data.TargetMarket,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to update company_core: %w", err)
	}

	// Update company_step2_data with enrichment results
	step2Query := `
		UPDATE company_step2_data SET
			-- Scalar fields: COALESCE (only set if NULL)
			business_model = COALESCE(business_model, $2),
			sector = COALESCE(sector, $3),
			pricing_model = COALESCE(pricing_model, $4),
			target_audience = COALESCE(target_audience, $5),
			value_proposition = COALESCE(value_proposition, $6),
			-- Array fields: merge + deduplicate (with type safety check)
			main_products = CASE
				WHEN $7::jsonb IS NULL OR jsonb_typeof($7::jsonb) != 'array' OR jsonb_array_length($7::jsonb) = 0 THEN COALESCE(main_products, '[]'::jsonb)
				WHEN main_products IS NULL OR jsonb_typeof(main_products) != 'array' OR jsonb_array_length(main_products) = 0 THEN $7::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(main_products || $7::jsonb))
			END,
			customer_segments = CASE
				WHEN $8::jsonb IS NULL OR jsonb_typeof($8::jsonb) != 'array' OR jsonb_array_length($8::jsonb) = 0 THEN COALESCE(customer_segments, '[]'::jsonb)
				WHEN customer_segments IS NULL OR jsonb_typeof(customer_segments) != 'array' OR jsonb_array_length(customer_segments) = 0 THEN $8::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(customer_segments || $8::jsonb))
			END,
			unique_selling_points = CASE
				WHEN $9::jsonb IS NULL OR jsonb_typeof($9::jsonb) != 'array' OR jsonb_array_length($9::jsonb) = 0 THEN COALESCE(unique_selling_points, '[]'::jsonb)
				WHEN unique_selling_points IS NULL OR jsonb_typeof(unique_selling_points) != 'array' OR jsonb_array_length(unique_selling_points) = 0 THEN $9::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(unique_selling_points || $9::jsonb))
			END,
			geographic_regions = CASE
				WHEN $10::jsonb IS NULL OR jsonb_typeof($10::jsonb) != 'array' OR jsonb_array_length($10::jsonb) = 0 THEN COALESCE(geographic_regions, '[]'::jsonb)
				WHEN geographic_regions IS NULL OR jsonb_typeof(geographic_regions) != 'array' OR jsonb_array_length(geographic_regions) = 0 THEN $10::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(geographic_regions || $10::jsonb))
			END,
			service_areas = CASE
				WHEN $11::jsonb IS NULL OR jsonb_typeof($11::jsonb) != 'array' OR jsonb_array_length($11::jsonb) = 0 THEN COALESCE(service_areas, '[]'::jsonb)
				WHEN service_areas IS NULL OR jsonb_typeof(service_areas) != 'array' OR jsonb_array_length(service_areas) = 0 THEN $11::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(service_areas || $11::jsonb))
			END,
			enrichment_sources = CASE
				WHEN $12::jsonb IS NULL OR jsonb_typeof($12::jsonb) != 'array' OR jsonb_array_length($12::jsonb) = 0 THEN COALESCE(enrichment_sources, '[]'::jsonb)
				WHEN enrichment_sources IS NULL OR jsonb_typeof(enrichment_sources) != 'array' OR jsonb_array_length(enrichment_sources) = 0 THEN $12::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(enrichment_sources || $12::jsonb))
			END,
			confidence_score = $13,
			completed_at = $14,
			updated_at = $15
		WHERE company_id = $1
	`

	_, err = r.querier().ExecContext(ctx, step2Query,
		id,
		data.BusinessModel, data.Sector, data.PricingModel,
		data.TargetAudience, data.ValueProposition,
		string(mainProductsJSON), string(customerSegmentsJSON), string(uniqueSellingPointsJSON),
		string(geographicRegionsJSON), string(serviceAreasJSON), string(sourcesJSON),
		data.ConfidenceScore, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to merge step2 data: %w", err)
	}
	return nil
}

// MergeStep3Data merges Step 3 enrichment data into company record
// Uses COALESCE for scalars, array merge for arrays
// Updates company_step3_data table directly
func (r *PostgresRepository) MergeStep3Data(ctx context.Context, id uuid.UUID, data *enrichment.Step3CompetitiveIntel) error {
	now := time.Now()

	// Convert arrays to JSON for merge
	competitorsJSON, _ := json.Marshal(data.Competitors)
	competitorDetailsJSON, _ := json.Marshal(data.CompetitorDetails)
	industryTrendsJSON, _ := json.Marshal(data.IndustryTrends)
	recentNewsJSON, _ := json.Marshal(data.RecentNews)
	sourcesJSON, _ := json.Marshal(data.Sources)

	// Update company_step3_data with enrichment results
	step3Query := `
		UPDATE company_step3_data SET
			-- Scalar fields: COALESCE (only set if NULL)
			industry_growth_rate = COALESCE(industry_growth_rate, $2),
			market_concentration = COALESCE(market_concentration, $3),
			regulatory_context = COALESCE(regulatory_context, $4),
			market_share_status = COALESCE(market_share_status, $5),
			-- Array fields: merge + deduplicate (with type safety check)
			competitors = CASE
				WHEN $6::jsonb IS NULL OR jsonb_typeof($6::jsonb) != 'array' OR jsonb_array_length($6::jsonb) = 0 THEN COALESCE(competitors, '[]'::jsonb)
				WHEN competitors IS NULL OR jsonb_typeof(competitors) != 'array' OR jsonb_array_length(competitors) = 0 THEN $6::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(competitors || $6::jsonb))
			END,
			competitor_details = CASE
				WHEN $7::jsonb IS NULL OR jsonb_typeof($7::jsonb) != 'array' OR jsonb_array_length($7::jsonb) = 0 THEN COALESCE(competitor_details, '[]'::jsonb)
				WHEN competitor_details IS NULL OR jsonb_typeof(competitor_details) != 'array' OR jsonb_array_length(competitor_details) = 0 THEN $7::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(competitor_details || $7::jsonb))
			END,
			industry_trends = CASE
				WHEN $8::jsonb IS NULL OR jsonb_typeof($8::jsonb) != 'array' OR jsonb_array_length($8::jsonb) = 0 THEN COALESCE(industry_trends, '[]'::jsonb)
				WHEN industry_trends IS NULL OR jsonb_typeof(industry_trends) != 'array' OR jsonb_array_length(industry_trends) = 0 THEN $8::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(industry_trends || $8::jsonb))
			END,
			recent_news = CASE
				WHEN $9::jsonb IS NULL OR jsonb_typeof($9::jsonb) != 'array' OR jsonb_array_length($9::jsonb) = 0 THEN COALESCE(recent_news, '[]'::jsonb)
				WHEN recent_news IS NULL OR jsonb_typeof(recent_news) != 'array' OR jsonb_array_length(recent_news) = 0 THEN $9::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(recent_news || $9::jsonb))
			END,
			enrichment_sources = CASE
				WHEN $10::jsonb IS NULL OR jsonb_typeof($10::jsonb) != 'array' OR jsonb_array_length($10::jsonb) = 0 THEN COALESCE(enrichment_sources, '[]'::jsonb)
				WHEN enrichment_sources IS NULL OR jsonb_typeof(enrichment_sources) != 'array' OR jsonb_array_length(enrichment_sources) = 0 THEN $10::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(enrichment_sources || $10::jsonb))
			END,
			confidence_score = $11,
			completed_at = $12,
			updated_at = $13
		WHERE company_id = $1
	`

	_, err := r.querier().ExecContext(ctx, step3Query,
		id,
		data.IndustryGrowthRate, data.MarketConcentration, data.RegulatoryContext, data.MarketPosition,
		string(competitorsJSON), string(competitorDetailsJSON), string(industryTrendsJSON), string(recentNewsJSON), string(sourcesJSON),
		data.ConfidenceScore, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to merge step3 data: %w", err)
	}
	return nil
}

// MergeCNPJData merges CNPJ registry data into company record
// Called from Step 1 or Step 2 when CNPJ is verified via Casa dos Dados
// Uses COALESCE for scalars, array merge for arrays
// Updates company_step1_data table directly (CNPJ data belongs to Step 1)
func (r *PostgresRepository) MergeCNPJData(ctx context.Context, id uuid.UUID, data *enrichment.CNPJData) error {
	if data == nil {
		return nil // Nothing to merge
	}

	now := time.Now()

	// Convert arrays to JSON for merge
	cnaeCodesJSON, _ := json.Marshal(data.CNAECodes)
	partnersJSON, _ := json.Marshal(data.Partners)

	// Update company_step1_data with CNPJ registry data
	step1Query := `
		UPDATE company_step1_data SET
			-- Scalar fields: COALESCE (only set if NULL)
			legal_name = COALESCE(legal_name, $2),
			trade_name = COALESCE(trade_name, $3),
			foundation_year = COALESCE(foundation_year, $4),
			headquarters = COALESCE(headquarters, $5),
			phone = COALESCE(phone, $6),
			email = COALESCE(email, $7),
			cnae_primary = COALESCE(cnae_primary, $8),
			capital_social = COALESCE(capital_social, $9),
			-- Boolean: set to true (CNPJ was verified)
			cnpj_verified = true,
			-- Array fields: merge + deduplicate (with type safety check)
			cnae_codes = CASE
				WHEN $10::jsonb IS NULL OR jsonb_typeof($10::jsonb) != 'array' OR jsonb_array_length($10::jsonb) = 0 THEN COALESCE(cnae_codes, '[]'::jsonb)
				WHEN cnae_codes IS NULL OR jsonb_typeof(cnae_codes) != 'array' OR jsonb_array_length(cnae_codes) = 0 THEN $10::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(cnae_codes || $10::jsonb))
			END,
			partners = CASE
				WHEN $11::jsonb IS NULL OR jsonb_typeof($11::jsonb) != 'array' OR jsonb_array_length($11::jsonb) = 0 THEN COALESCE(partners, '[]'::jsonb)
				WHEN partners IS NULL OR jsonb_typeof(partners) != 'array' OR jsonb_array_length(partners) = 0 THEN $11::jsonb
				ELSE (SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb) FROM jsonb_array_elements(partners || $11::jsonb))
			END,
			updated_at = $12
		WHERE company_id = $1
	`

	_, err := r.querier().ExecContext(ctx, step1Query,
		id,
		data.LegalName, data.TradeName, data.FoundationYear, data.Headquarters,
		data.Phone, data.Email, data.CNAEPrimary, data.CapitalSocial,
		string(cnaeCodesJSON), string(partnersJSON),
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to merge cnpj data: %w", err)
	}
	return nil
}

// nullIfEmpty returns nil if the string is empty, otherwise returns a pointer to the string
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// =============================================================================
// CNPJ DEDUPLICATION METHODS
// =============================================================================

// NormalizeCNPJ removes all non-digit characters from a CNPJ string.
// This ensures consistent matching regardless of formatting (e.g., "12.345.678/0001-90" -> "12345678000190")
func NormalizeCNPJ(cnpj string) string {
	return regexp.MustCompile(`[^0-9]`).ReplaceAllString(cnpj, "")
}

// FindByCNPJNormalized finds a company by its normalized CNPJ.
// Returns nil, nil if no company is found (not an error).
func (r *PostgresRepository) FindByCNPJNormalized(ctx context.Context, cnpj string) (*Company, error) {
	normalized := NormalizeCNPJ(cnpj)
	if normalized == "" {
		return nil, nil
	}

	query := `
		SELECT
			id, name, cnpj, website,
			industry, company_size, location, target_market, funding_stage,
			annual_revenue_min, annual_revenue_max,
			foundation_year, legal_name, headquarters, sector, target_audience, value_proposition,
			employees_range, revenue_estimate, business_model, competitors, market_share_status,
			digital_maturity, strengths, weaknesses,
			main_products, recent_news, key_executives, opportunities, threats,
			strategic_challenges, competitor_details, competitive_advantage, market_share,
			tam_estimate, sam_estimate, som_estimate, company_history,
			customer_segments, pricing_model, unique_selling_points,
			geographic_regions, service_areas,
			industry_growth_rate, industry_trends, regulatory_context, market_concentration, enrichment_sources,
			trade_name, phone, email, cnae_primary, cnae_codes, capital_social, partners, cnpj_verified,
			linkedin_url, twitter_handle, instagram_url, facebook_url,
			enrichment_status, enrichment_completed_at, enrichment_error,
			allowed_users, owner_id,
			created_at, updated_at
		FROM companies
		WHERE cnpj_normalized = $1
		  AND deleted_at IS NULL
		LIMIT 1
	`

	var company Company
	err := sqlx.GetContext(ctx, r.querier(), &company, query, normalized)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find by cnpj normalized: %w", err)
	}
	return &company, nil
}

// CheckEmailCNPJExists checks if an email has already submitted a company with this CNPJ.
// This prevents the same user from creating multiple companies with the same CNPJ.
// Different users CAN create companies with the same CNPJ.
func (r *PostgresRepository) CheckEmailCNPJExists(ctx context.Context, email string, cnpj string) (bool, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	normalizedCNPJ := NormalizeCNPJ(cnpj)

	if normalizedCNPJ == "" {
		return false, nil // No CNPJ to check
	}

	query := `
		SELECT EXISTS(
			SELECT 1 FROM company_submissions cs
			JOIN companies c ON c.id = cs.company_id
			WHERE cs.submitter_email = $1
			  AND c.cnpj_normalized = $2
			  AND c.deleted_at IS NULL
		)
	`

	var exists bool
	err := r.querier().QueryRowxContext(ctx, query, normalizedEmail, normalizedCNPJ).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check email cnpj exists: %w", err)
	}
	return exists, nil
}

// GetDuplicatesForReview returns all CNPJ duplicate pairs that haven't been reviewed yet.
func (r *PostgresRepository) GetDuplicatesForReview(ctx context.Context) ([]*CNPJDuplicateReview, error) {
	query := `
		SELECT
			r.id, r.cnpj_normalized, r.older_company_id, r.newer_company_id,
			r.created_at, r.reviewed_at, r.reviewed_by, r.action_taken, r.notes
		FROM cnpj_duplicates_review r
		WHERE r.reviewed_at IS NULL
		ORDER BY r.created_at DESC
	`

	rows, err := r.querier().QueryxContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get duplicates for review: %w", err)
	}
	defer rows.Close()

	var reviews []*CNPJDuplicateReview
	for rows.Next() {
		var review CNPJDuplicateReview
		if err := rows.StructScan(&review); err != nil {
			return nil, fmt.Errorf("scan duplicate review: %w", err)
		}

		// Load the related companies
		olderCompany, _ := r.GetByID(ctx, review.OlderCompanyID)
		newerCompany, _ := r.GetByID(ctx, review.NewerCompanyID)
		review.OlderCompany = olderCompany
		review.NewerCompany = newerCompany

		reviews = append(reviews, &review)
	}

	return reviews, nil
}

// CreateDuplicateReview creates a new CNPJ duplicate review entry.
func (r *PostgresRepository) CreateDuplicateReview(ctx context.Context, review *CNPJDuplicateReview) error {
	if review.ID == uuid.Nil {
		review.ID = uuid.New()
	}
	if review.CreatedAt.IsZero() {
		review.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO cnpj_duplicates_review (
			id, cnpj_normalized, older_company_id, newer_company_id, created_at
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (older_company_id, newer_company_id) DO NOTHING
	`

	_, err := r.querier().ExecContext(ctx, query,
		review.ID, review.CNPJNormalized, review.OlderCompanyID, review.NewerCompanyID, review.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create duplicate review: %w", err)
	}
	return nil
}

// MergeCompanies merges the source company into the target company.
// This moves all submissions and challenges from source to target, then soft-deletes the source.
func (r *PostgresRepository) MergeCompanies(ctx context.Context, targetID, sourceID, mergedBy uuid.UUID) error {
	// Verify both companies exist
	target, err := r.GetByID(ctx, targetID)
	if err != nil || target == nil {
		return fmt.Errorf("target company not found: %s", targetID)
	}
	source, err := r.GetByID(ctx, sourceID)
	if err != nil || source == nil {
		return fmt.Errorf("source company not found: %s", sourceID)
	}

	now := time.Now()

	// Use a transaction for atomicity
	return r.WithTx(ctx, func(txRepo Repository) error {
		txPgRepo := txRepo.(*PostgresRepository)

		// 1. Move company_submissions from source to target
		// Handle potential conflicts by updating existing links
		moveSubmissionsQuery := `
			UPDATE company_submissions
			SET company_id = $1
			WHERE company_id = $2
			  AND submission_id NOT IN (
				SELECT submission_id FROM company_submissions WHERE company_id = $1
			  )
		`
		_, err := txPgRepo.querier().ExecContext(ctx, moveSubmissionsQuery, targetID, sourceID)
		if err != nil {
			return fmt.Errorf("move submissions: %w", err)
		}

		// Delete any remaining duplicate links
		deleteDuplicateLinksQuery := `
			DELETE FROM company_submissions WHERE company_id = $1
		`
		_, err = txPgRepo.querier().ExecContext(ctx, deleteDuplicateLinksQuery, sourceID)
		if err != nil {
			return fmt.Errorf("delete duplicate links: %w", err)
		}

		// 2. Move challenges from source to target
		moveChallengesQuery := `
			UPDATE challenges SET company_id = $1 WHERE company_id = $2
		`
		_, err = txPgRepo.querier().ExecContext(ctx, moveChallengesQuery, targetID, sourceID)
		if err != nil {
			return fmt.Errorf("move challenges: %w", err)
		}

		// 3. Soft-delete the source company (update company_core, not the view)
		deleteSourceQuery := `
			UPDATE company_core SET deleted_at = $1 WHERE id = $2
		`
		_, err = txPgRepo.querier().ExecContext(ctx, deleteSourceQuery, now, sourceID)
		if err != nil {
			return fmt.Errorf("delete source company: %w", err)
		}

		// 4. Update the review record to mark as merged
		updateReviewQuery := `
			UPDATE cnpj_duplicates_review
			SET reviewed_at = $1, reviewed_by = $2, action_taken = 'merged'
			WHERE (older_company_id = $3 AND newer_company_id = $4)
			   OR (older_company_id = $4 AND newer_company_id = $3)
		`
		_, err = txPgRepo.querier().ExecContext(ctx, updateReviewQuery, now, mergedBy, targetID, sourceID)
		if err != nil {
			return fmt.Errorf("update review record: %w", err)
		}

		return nil
	})
}
