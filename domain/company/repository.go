package company

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

	// Submission links
	LinkSubmission(ctx context.Context, link *CompanySubmission) error
	UnlinkSubmissions(ctx context.Context, companyID uuid.UUID) error // For saga rollback

	// Analysis history (for admin dashboard)
	GetAnalysesHistory(ctx context.Context, companyID uuid.UUID) ([]*AnalysisHistoryItem, error)

	// Enrichment status updates
	SetEnrichmentProcessing(ctx context.Context, id uuid.UUID) error
	SetEnrichmentCompleted(ctx context.Context, id uuid.UUID, data *enrichment.EnrichedCompanyData) error
	SetEnrichmentFailed(ctx context.Context, id uuid.UUID, errMsg string) error

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
			main_products, recent_news, key_executives, opportunities, threats,
			strategic_challenges, competitor_details, competitive_advantage, market_share,
			tam_estimate, sam_estimate, som_estimate, company_history,
			customer_segments, pricing_model, unique_selling_points,
			industry_growth_rate, industry_trends, regulatory_context, market_concentration, enrichment_sources,
			linkedin_url, twitter_handle,
			enrichment_status, enrichment_completed_at, enrichment_error,
			allowed_users, owner_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9,
			$10, $11,
			$12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22,
			$23, $24, $25,
			$26, $27, $28, $29, $30,
			$31, $32, $33, $34,
			$35, $36, $37, $38,
			$39, $40, $41,
			$42, $43, $44, $45, $46,
			$47, $48,
			$49, $50, $51,
			$52, $53,
			$54, $55
		)
	`

	_, err := r.querier().ExecContext(ctx, query,
		company.ID, company.Name, company.CNPJ, company.Website,
		company.Industry, company.CompanySize, company.Location, company.TargetMarket, company.FundingStage,
		company.AnnualRevenueMin, company.AnnualRevenueMax,
		company.FoundationYear, company.LegalName, company.Headquarters, company.Sector, company.TargetAudience, company.ValueProposition,
		company.EmployeesRange, company.RevenueEstimate, company.BusinessModel, company.Competitors, company.MarketShareStatus,
		company.DigitalMaturity, company.Strengths, company.Weaknesses,
		company.MainProducts, company.RecentNews, company.KeyExecutives, company.Opportunities, company.Threats,
		company.StrategicChallenges, company.CompetitorDetails, company.CompetitiveAdvantage, company.MarketShare,
		company.TAMEstimate, company.SAMEstimate, company.SOMEstimate, company.CompanyHistory,
		company.CustomerSegments, company.PricingModel, company.UniqueSellingPoints,
		company.IndustryGrowthRate, company.IndustryTrends, company.RegulatoryContext, company.MarketConcentration, company.EnrichmentSources,
		company.LinkedInURL, company.TwitterHandle,
		company.EnrichmentStatus, company.EnrichmentCompletedAt, company.EnrichmentError,
		company.AllowedUsers, company.OwnerID,
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
			main_products, recent_news, key_executives, opportunities, threats,
			strategic_challenges, competitor_details, competitive_advantage, market_share,
			tam_estimate, sam_estimate, som_estimate, company_history,
			customer_segments, pricing_model, unique_selling_points,
			industry_growth_rate, industry_trends, regulatory_context, market_concentration, enrichment_sources,
			linkedin_url, twitter_handle,
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
			main_products = $26, recent_news = $27, key_executives = $28, opportunities = $29, threats = $30,
			strategic_challenges = $31, competitor_details = $32, competitive_advantage = $33, market_share = $34,
			tam_estimate = $35, sam_estimate = $36, som_estimate = $37, company_history = $38,
			customer_segments = $39, pricing_model = $40, unique_selling_points = $41,
			industry_growth_rate = $42, industry_trends = $43, regulatory_context = $44, market_concentration = $45, enrichment_sources = $46,
			linkedin_url = $47, twitter_handle = $48,
			enrichment_status = $49, enrichment_completed_at = $50, enrichment_error = $51,
			allowed_users = $52, owner_id = $53,
			updated_at = $54
		WHERE id = $1
	`

	_, err := r.querier().ExecContext(ctx, query,
		company.ID, company.Name, company.CNPJ, company.Website,
		company.Industry, company.CompanySize, company.Location, company.TargetMarket, company.FundingStage,
		company.AnnualRevenueMin, company.AnnualRevenueMax,
		company.FoundationYear, company.LegalName, company.Headquarters, company.Sector, company.TargetAudience, company.ValueProposition,
		company.EmployeesRange, company.RevenueEstimate, company.BusinessModel, company.Competitors, company.MarketShareStatus,
		company.DigitalMaturity, company.Strengths, company.Weaknesses,
		company.MainProducts, company.RecentNews, company.KeyExecutives, company.Opportunities, company.Threats,
		company.StrategicChallenges, company.CompetitorDetails, company.CompetitiveAdvantage, company.MarketShare,
		company.TAMEstimate, company.SAMEstimate, company.SOMEstimate, company.CompanyHistory,
		company.CustomerSegments, company.PricingModel, company.UniqueSellingPoints,
		company.IndustryGrowthRate, company.IndustryTrends, company.RegulatoryContext, company.MarketConcentration, company.EnrichmentSources,
		company.LinkedInURL, company.TwitterHandle,
		company.EnrichmentStatus, company.EnrichmentCompletedAt, company.EnrichmentError,
		company.AllowedUsers, company.OwnerID,
		company.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update company: %w", err)
	}
	return nil
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
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// First unlink any submissions to avoid FK constraint
	if err := r.UnlinkSubmissions(ctx, id); err != nil {
		return fmt.Errorf("failed to unlink submissions before delete: %w", err)
	}

	query := `DELETE FROM companies WHERE id = $1`
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
			industry_growth_rate, industry_trends, regulatory_context, market_concentration, enrichment_sources,
			linkedin_url, twitter_handle,
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
			linkedin_url, twitter_handle,
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
func (r *PostgresRepository) SetEnrichmentProcessing(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE companies
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

// SetEnrichmentCompleted updates company with enriched data and marks as completed
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
			competitors = CASE WHEN competitors IS NULL OR jsonb_array_length(competitors) = 0 THEN $24 ELSE competitors END,
			strengths = CASE WHEN strengths IS NULL OR jsonb_array_length(strengths) = 0 THEN $25 ELSE strengths END,
			weaknesses = CASE WHEN weaknesses IS NULL OR jsonb_array_length(weaknesses) = 0 THEN $26 ELSE weaknesses END,
			-- Enrichment v3 fields
			main_products = CASE WHEN main_products IS NULL OR jsonb_array_length(main_products) = 0 THEN $27 ELSE main_products END,
			recent_news = CASE WHEN recent_news IS NULL OR jsonb_array_length(recent_news) = 0 THEN $28 ELSE recent_news END,
			key_executives = CASE WHEN key_executives IS NULL OR jsonb_array_length(key_executives) = 0 THEN $29 ELSE key_executives END,
			opportunities = CASE WHEN opportunities IS NULL OR jsonb_array_length(opportunities) = 0 THEN $30 ELSE opportunities END,
			threats = CASE WHEN threats IS NULL OR jsonb_array_length(threats) = 0 THEN $31 ELSE threats END,
			strategic_challenges = CASE WHEN strategic_challenges IS NULL OR jsonb_array_length(strategic_challenges) = 0 THEN $32 ELSE strategic_challenges END,
			competitor_details = CASE WHEN competitor_details IS NULL OR jsonb_array_length(competitor_details) = 0 THEN $33 ELSE competitor_details END,
			competitive_advantage = COALESCE(competitive_advantage, $34),
			market_share = COALESCE(market_share, $35),
			tam_estimate = COALESCE(tam_estimate, $36),
			sam_estimate = COALESCE(sam_estimate, $37),
			som_estimate = COALESCE(som_estimate, $38),
			company_history = COALESCE(company_history, $39),
			customer_segments = CASE WHEN customer_segments IS NULL OR jsonb_array_length(customer_segments) = 0 THEN $40 ELSE customer_segments END,
			pricing_model = COALESCE(pricing_model, $41),
			unique_selling_points = CASE WHEN unique_selling_points IS NULL OR jsonb_array_length(unique_selling_points) = 0 THEN $42 ELSE unique_selling_points END,
			-- Industry context
			industry_growth_rate = COALESCE(industry_growth_rate, $43),
			industry_trends = CASE WHEN industry_trends IS NULL OR jsonb_array_length(industry_trends) = 0 THEN $44 ELSE industry_trends END,
			regulatory_context = COALESCE(regulatory_context, $45),
			market_concentration = COALESCE(market_concentration, $46),
			enrichment_sources = CASE WHEN enrichment_sources IS NULL OR jsonb_array_length(enrichment_sources) = 0 THEN $47 ELSE enrichment_sources END,
			linkedin_url = COALESCE(linkedin_url, $48),
			twitter_handle = COALESCE(twitter_handle, $49),
			updated_at = $50
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
		data.LinkedInURL, data.TwitterHandle,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to set enrichment completed: %w", err)
	}
	return nil
}

// SetEnrichmentFailed marks company enrichment as failed with error message
func (r *PostgresRepository) SetEnrichmentFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	query := `
		UPDATE companies
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
