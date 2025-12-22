-- Migration: v2_024_split_companies_table_TEST_ONLY.sql
-- Description: Split companies table into step-specific tables (FOR TEST DB ONLY)
-- Date: 2024-12-21
--
-- NOTE: This is a modified version for TEST DB which is missing some legacy columns
-- that exist in production (sector_description, sector_growth_rate, etc.)
--
-- DO NOT RUN THIS IN PRODUCTION - use v2_024_split_companies_table.sql instead

BEGIN;

-- =============================================================================
-- STEP 1: CREATE NEW TABLES
-- =============================================================================

-- Core company table: only user-provided and meta fields
CREATE TABLE IF NOT EXISTS company_core (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    cnpj TEXT,
    website TEXT,
    industry TEXT,
    company_size TEXT,
    location TEXT,
    target_market TEXT,
    funding_stage TEXT,
    annual_revenue_min NUMERIC,
    annual_revenue_max NUMERIC,

    -- Ownership and access
    is_verified BOOLEAN NOT NULL DEFAULT false,
    allowed_users UUID[] NOT NULL DEFAULT '{}',
    owner_id UUID,

    -- Enrichment status (meta)
    enrichment_status VARCHAR DEFAULT 'pending' CHECK (enrichment_status IN ('pending', 'processing', 'completed', 'failed')),
    enrichment_completed_at TIMESTAMPTZ,
    enrichment_error TEXT,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Step 1 data: Basic Info (CNPJ registry + basic search)
CREATE TABLE IF NOT EXISTS company_step1_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL UNIQUE REFERENCES company_core(id) ON DELETE CASCADE,

    -- Legal/Registry info
    legal_name TEXT,
    trade_name TEXT,
    foundation_year TEXT,
    headquarters TEXT,
    employees_range TEXT,

    -- Contact
    phone TEXT,
    email TEXT,

    -- CNPJ Registry data
    cnae_primary TEXT,
    cnae_codes JSONB DEFAULT '[]',
    capital_social TEXT,
    partners JSONB DEFAULT '[]',
    cnpj_verified BOOLEAN DEFAULT false,

    -- Social media
    linkedin_url TEXT,
    twitter_handle TEXT,
    instagram_url TEXT,
    facebook_url TEXT,

    -- Executives
    key_executives JSONB DEFAULT '[]',

    -- Meta
    enrichment_sources JSONB DEFAULT '[]',
    confidence_score NUMERIC,
    completed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Step 2 data: Business Model (website crawl + business search)
CREATE TABLE IF NOT EXISTS company_step2_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL UNIQUE REFERENCES company_core(id) ON DELETE CASCADE,

    -- Business model
    business_model TEXT,
    sector TEXT,
    pricing_model TEXT,

    -- Target
    target_audience TEXT,
    value_proposition TEXT,

    -- Products/Services
    main_products JSONB DEFAULT '[]',
    customer_segments JSONB DEFAULT '[]',
    unique_selling_points JSONB DEFAULT '[]',

    -- Geography
    geographic_regions JSONB DEFAULT '[]',
    service_areas JSONB DEFAULT '[]',

    -- Meta
    enrichment_sources JSONB DEFAULT '[]',
    confidence_score NUMERIC,
    completed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Step 3 data: Competitive Intelligence
CREATE TABLE IF NOT EXISTS company_step3_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL UNIQUE REFERENCES company_core(id) ON DELETE CASCADE,

    -- Competitors
    competitors JSONB DEFAULT '[]',
    competitor_details JSONB DEFAULT '[]',
    competitive_advantage TEXT,

    -- Market
    market_share TEXT,
    market_share_status TEXT,
    market_concentration TEXT,

    -- Industry
    industry_growth_rate TEXT,
    industry_trends JSONB DEFAULT '[]',
    regulatory_context TEXT,

    -- SWOT
    strengths JSONB DEFAULT '[]',
    weaknesses JSONB DEFAULT '[]',
    opportunities JSONB DEFAULT '[]',
    threats JSONB DEFAULT '[]',

    -- Strategic
    strategic_challenges JSONB DEFAULT '[]',
    recent_news JSONB DEFAULT '[]',

    -- Market sizing
    tam_estimate TEXT,
    sam_estimate TEXT,
    som_estimate TEXT,

    -- Meta
    enrichment_sources JSONB DEFAULT '[]',
    confidence_score NUMERIC,
    completed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Legacy data: deprecated fields for backwards compatibility
CREATE TABLE IF NOT EXISTS company_legacy_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL UNIQUE REFERENCES company_core(id) ON DELETE CASCADE,

    -- Deprecated fields (not actively filled by enrichment)
    revenue_estimate TEXT,
    digital_maturity INTEGER CHECK (digital_maturity IS NULL OR (digital_maturity >= 1 AND digital_maturity <= 10)),
    company_history TEXT,
    sector_description TEXT,
    sector_growth_rate VARCHAR,
    sector_trends JSONB,
    key_industry_trends TEXT[],
    macro_context JSONB,
    competitive_landscape JSONB,
    data_confidence VARCHAR,
    data_gaps TEXT[],
    enriched_at TIMESTAMPTZ,
    enrichment_source VARCHAR,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- STEP 2: CREATE INDEXES
-- =============================================================================

CREATE INDEX IF NOT EXISTS idx_company_step1_company_id ON company_step1_data(company_id);
CREATE INDEX IF NOT EXISTS idx_company_step2_company_id ON company_step2_data(company_id);
CREATE INDEX IF NOT EXISTS idx_company_step3_company_id ON company_step3_data(company_id);
CREATE INDEX IF NOT EXISTS idx_company_legacy_company_id ON company_legacy_data(company_id);
CREATE INDEX IF NOT EXISTS idx_company_core_owner_id ON company_core(owner_id);
CREATE INDEX IF NOT EXISTS idx_company_core_enrichment_status ON company_core(enrichment_status);

-- =============================================================================
-- STEP 3: MIGRATE DATA FROM EXISTING COMPANIES TABLE
-- (Modified for TEST DB - only uses columns that exist)
-- =============================================================================

-- Insert into company_core
INSERT INTO company_core (
    id, name, cnpj, website, industry, company_size, location,
    target_market, funding_stage, annual_revenue_min, annual_revenue_max,
    is_verified, allowed_users, owner_id,
    enrichment_status, enrichment_completed_at, enrichment_error,
    created_at, updated_at
)
SELECT
    id, name, cnpj, website, industry, company_size, location,
    target_market, funding_stage, annual_revenue_min, annual_revenue_max,
    is_verified, allowed_users, owner_id,
    enrichment_status, enrichment_completed_at, enrichment_error,
    created_at, updated_at
FROM companies
ON CONFLICT (id) DO NOTHING;

-- Insert into company_step1_data
INSERT INTO company_step1_data (
    company_id, legal_name, trade_name, foundation_year, headquarters,
    employees_range, phone, email, cnae_primary, cnae_codes, capital_social,
    partners, cnpj_verified, linkedin_url, twitter_handle, instagram_url,
    facebook_url, key_executives, enrichment_sources
)
SELECT
    id, legal_name, trade_name, foundation_year, headquarters,
    employees_range, phone, email, cnae_primary,
    COALESCE(cnae_codes, '[]'::jsonb),
    capital_social,
    COALESCE(partners, '[]'::jsonb),
    COALESCE(cnpj_verified, false),
    linkedin_url, twitter_handle, instagram_url, facebook_url,
    COALESCE(key_executives, '[]'::jsonb),
    COALESCE(enrichment_sources, '[]'::jsonb)
FROM companies
ON CONFLICT (company_id) DO NOTHING;

-- Insert into company_step2_data
INSERT INTO company_step2_data (
    company_id, business_model, sector, pricing_model, target_audience,
    value_proposition, main_products, customer_segments, unique_selling_points,
    geographic_regions, service_areas, enrichment_sources
)
SELECT
    id, business_model, sector, pricing_model, target_audience,
    value_proposition,
    COALESCE(main_products, '[]'::jsonb),
    COALESCE(customer_segments, '[]'::jsonb),
    COALESCE(unique_selling_points, '[]'::jsonb),
    COALESCE(geographic_regions, '[]'::jsonb),
    COALESCE(service_areas, '[]'::jsonb),
    COALESCE(enrichment_sources, '[]'::jsonb)
FROM companies
ON CONFLICT (company_id) DO NOTHING;

-- Insert into company_step3_data
INSERT INTO company_step3_data (
    company_id, competitors, competitor_details, competitive_advantage,
    market_share, market_share_status, market_concentration,
    industry_growth_rate, industry_trends, regulatory_context,
    strengths, weaknesses, opportunities, threats,
    strategic_challenges, recent_news,
    tam_estimate, sam_estimate, som_estimate, enrichment_sources
)
SELECT
    id,
    COALESCE(competitors, '[]'::jsonb),
    COALESCE(competitor_details, '[]'::jsonb),
    competitive_advantage,
    market_share, market_share_status, market_concentration,
    industry_growth_rate,
    COALESCE(industry_trends, '[]'::jsonb),
    regulatory_context,
    COALESCE(strengths, '[]'::jsonb),
    COALESCE(weaknesses, '[]'::jsonb),
    COALESCE(opportunities, '[]'::jsonb),
    COALESCE(threats, '[]'::jsonb),
    COALESCE(strategic_challenges, '[]'::jsonb),
    COALESCE(recent_news, '[]'::jsonb),
    tam_estimate, sam_estimate, som_estimate,
    COALESCE(enrichment_sources, '[]'::jsonb)
FROM companies
ON CONFLICT (company_id) DO NOTHING;

-- Insert into company_legacy_data (TEST DB doesn't have legacy columns, so insert empty rows)
INSERT INTO company_legacy_data (
    company_id, revenue_estimate, digital_maturity, company_history
)
SELECT
    id, revenue_estimate, digital_maturity, company_history
FROM companies
ON CONFLICT (company_id) DO NOTHING;

-- =============================================================================
-- STEP 4: RENAME ORIGINAL TABLE (BACKUP, NOT DROP)
-- =============================================================================

ALTER TABLE companies RENAME TO companies_backup_v2024;

-- =============================================================================
-- STEP 5: CREATE BACKWARDS-COMPATIBLE VIEW
-- This view looks exactly like the old companies table
-- =============================================================================

CREATE OR REPLACE VIEW companies AS
SELECT
    -- Core fields
    c.id,
    c.name,
    c.cnpj,
    c.website,
    c.industry,
    c.company_size,
    c.location,
    c.target_market,
    c.funding_stage,
    c.annual_revenue_min,
    c.annual_revenue_max,
    c.is_verified,
    c.allowed_users,
    c.owner_id,
    c.enrichment_status,
    c.enrichment_completed_at,
    c.enrichment_error,
    c.created_at,
    c.updated_at,

    -- Step 1 fields
    s1.legal_name,
    s1.trade_name,
    s1.foundation_year,
    s1.headquarters,
    s1.employees_range,
    s1.phone,
    s1.email,
    s1.cnae_primary,
    COALESCE(s1.cnae_codes, '[]'::jsonb) as cnae_codes,
    s1.capital_social,
    COALESCE(s1.partners, '[]'::jsonb) as partners,
    COALESCE(s1.cnpj_verified, false) as cnpj_verified,
    s1.linkedin_url,
    s1.twitter_handle,
    s1.instagram_url,
    s1.facebook_url,
    COALESCE(s1.key_executives, '[]'::jsonb) as key_executives,

    -- Step 2 fields
    s2.business_model,
    s2.sector,
    s2.pricing_model,
    s2.target_audience,
    s2.value_proposition,
    COALESCE(s2.main_products, '[]'::jsonb) as main_products,
    COALESCE(s2.customer_segments, '[]'::jsonb) as customer_segments,
    COALESCE(s2.unique_selling_points, '[]'::jsonb) as unique_selling_points,
    COALESCE(s2.geographic_regions, '[]'::jsonb) as geographic_regions,
    COALESCE(s2.service_areas, '[]'::jsonb) as service_areas,

    -- Step 3 fields
    COALESCE(s3.competitors, '[]'::jsonb) as competitors,
    COALESCE(s3.competitor_details, '[]'::jsonb) as competitor_details,
    s3.competitive_advantage,
    s3.market_share,
    s3.market_share_status,
    s3.market_concentration,
    s3.industry_growth_rate,
    COALESCE(s3.industry_trends, '[]'::jsonb) as industry_trends,
    s3.regulatory_context,
    COALESCE(s3.strengths, '[]'::jsonb) as strengths,
    COALESCE(s3.weaknesses, '[]'::jsonb) as weaknesses,
    COALESCE(s3.opportunities, '[]'::jsonb) as opportunities,
    COALESCE(s3.threats, '[]'::jsonb) as threats,
    COALESCE(s3.strategic_challenges, '[]'::jsonb) as strategic_challenges,
    COALESCE(s3.recent_news, '[]'::jsonb) as recent_news,
    s3.tam_estimate,
    s3.sam_estimate,
    s3.som_estimate,

    -- Combined enrichment sources from all steps
    COALESCE(
        (SELECT jsonb_agg(DISTINCT value) FROM (
            SELECT jsonb_array_elements(COALESCE(s1.enrichment_sources, '[]'::jsonb))
            UNION ALL
            SELECT jsonb_array_elements(COALESCE(s2.enrichment_sources, '[]'::jsonb))
            UNION ALL
            SELECT jsonb_array_elements(COALESCE(s3.enrichment_sources, '[]'::jsonb))
        ) sources(value)),
        '[]'::jsonb
    ) as enrichment_sources,

    -- Legacy fields
    leg.revenue_estimate,
    leg.digital_maturity,
    leg.company_history,
    leg.sector_description,
    leg.sector_growth_rate,
    leg.sector_trends,
    leg.key_industry_trends,
    leg.macro_context,
    leg.competitive_landscape,
    leg.data_confidence,
    leg.data_gaps,
    leg.enriched_at,
    leg.enrichment_source

FROM company_core c
LEFT JOIN company_step1_data s1 ON s1.company_id = c.id
LEFT JOIN company_step2_data s2 ON s2.company_id = c.id
LEFT JOIN company_step3_data s3 ON s3.company_id = c.id
LEFT JOIN company_legacy_data leg ON leg.company_id = c.id;

-- =============================================================================
-- STEP 6: CREATE TRIGGER FUNCTIONS FOR VIEW UPDATES
-- These allow INSERT/UPDATE/DELETE on the view to work
-- =============================================================================

CREATE OR REPLACE FUNCTION companies_insert_trigger()
RETURNS TRIGGER AS $$
DECLARE
    new_id UUID;
BEGIN
    -- Use provided ID or generate new one
    new_id := COALESCE(NEW.id, gen_random_uuid());

    -- Insert into core
    INSERT INTO company_core (
        id, name, cnpj, website, industry, company_size, location,
        target_market, funding_stage, annual_revenue_min, annual_revenue_max,
        is_verified, allowed_users, owner_id,
        enrichment_status, enrichment_completed_at, enrichment_error,
        created_at, updated_at
    ) VALUES (
        new_id, NEW.name, NEW.cnpj, NEW.website, NEW.industry, NEW.company_size, NEW.location,
        NEW.target_market, NEW.funding_stage, NEW.annual_revenue_min, NEW.annual_revenue_max,
        COALESCE(NEW.is_verified, false), COALESCE(NEW.allowed_users, '{}'), NEW.owner_id,
        COALESCE(NEW.enrichment_status, 'pending'), NEW.enrichment_completed_at, NEW.enrichment_error,
        COALESCE(NEW.created_at, NOW()), COALESCE(NEW.updated_at, NOW())
    );

    -- Insert into step1
    INSERT INTO company_step1_data (company_id) VALUES (new_id);

    -- Insert into step2
    INSERT INTO company_step2_data (company_id) VALUES (new_id);

    -- Insert into step3
    INSERT INTO company_step3_data (company_id) VALUES (new_id);

    -- Insert into legacy
    INSERT INTO company_legacy_data (company_id) VALUES (new_id);

    NEW.id := new_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION companies_update_trigger()
RETURNS TRIGGER AS $$
BEGIN
    -- Update core
    UPDATE company_core SET
        name = NEW.name,
        cnpj = NEW.cnpj,
        website = NEW.website,
        industry = NEW.industry,
        company_size = NEW.company_size,
        location = NEW.location,
        target_market = NEW.target_market,
        funding_stage = NEW.funding_stage,
        annual_revenue_min = NEW.annual_revenue_min,
        annual_revenue_max = NEW.annual_revenue_max,
        is_verified = NEW.is_verified,
        allowed_users = NEW.allowed_users,
        owner_id = NEW.owner_id,
        enrichment_status = NEW.enrichment_status,
        enrichment_completed_at = NEW.enrichment_completed_at,
        enrichment_error = NEW.enrichment_error,
        updated_at = NOW()
    WHERE id = OLD.id;

    -- Update step1
    UPDATE company_step1_data SET
        legal_name = NEW.legal_name,
        trade_name = NEW.trade_name,
        foundation_year = NEW.foundation_year,
        headquarters = NEW.headquarters,
        employees_range = NEW.employees_range,
        phone = NEW.phone,
        email = NEW.email,
        cnae_primary = NEW.cnae_primary,
        cnae_codes = NEW.cnae_codes,
        capital_social = NEW.capital_social,
        partners = NEW.partners,
        cnpj_verified = NEW.cnpj_verified,
        linkedin_url = NEW.linkedin_url,
        twitter_handle = NEW.twitter_handle,
        instagram_url = NEW.instagram_url,
        facebook_url = NEW.facebook_url,
        key_executives = NEW.key_executives,
        updated_at = NOW()
    WHERE company_id = OLD.id;

    -- Update step2
    UPDATE company_step2_data SET
        business_model = NEW.business_model,
        sector = NEW.sector,
        pricing_model = NEW.pricing_model,
        target_audience = NEW.target_audience,
        value_proposition = NEW.value_proposition,
        main_products = NEW.main_products,
        customer_segments = NEW.customer_segments,
        unique_selling_points = NEW.unique_selling_points,
        geographic_regions = NEW.geographic_regions,
        service_areas = NEW.service_areas,
        updated_at = NOW()
    WHERE company_id = OLD.id;

    -- Update step3
    UPDATE company_step3_data SET
        competitors = NEW.competitors,
        competitor_details = NEW.competitor_details,
        competitive_advantage = NEW.competitive_advantage,
        market_share = NEW.market_share,
        market_share_status = NEW.market_share_status,
        market_concentration = NEW.market_concentration,
        industry_growth_rate = NEW.industry_growth_rate,
        industry_trends = NEW.industry_trends,
        regulatory_context = NEW.regulatory_context,
        strengths = NEW.strengths,
        weaknesses = NEW.weaknesses,
        opportunities = NEW.opportunities,
        threats = NEW.threats,
        strategic_challenges = NEW.strategic_challenges,
        recent_news = NEW.recent_news,
        tam_estimate = NEW.tam_estimate,
        sam_estimate = NEW.sam_estimate,
        som_estimate = NEW.som_estimate,
        updated_at = NOW()
    WHERE company_id = OLD.id;

    -- Update legacy
    UPDATE company_legacy_data SET
        revenue_estimate = NEW.revenue_estimate,
        digital_maturity = NEW.digital_maturity,
        company_history = NEW.company_history,
        sector_description = NEW.sector_description,
        sector_growth_rate = NEW.sector_growth_rate,
        sector_trends = NEW.sector_trends,
        key_industry_trends = NEW.key_industry_trends,
        macro_context = NEW.macro_context,
        competitive_landscape = NEW.competitive_landscape,
        data_confidence = NEW.data_confidence,
        data_gaps = NEW.data_gaps,
        enriched_at = NEW.enriched_at,
        enrichment_source = NEW.enrichment_source,
        updated_at = NOW()
    WHERE company_id = OLD.id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION companies_delete_trigger()
RETURNS TRIGGER AS $$
BEGIN
    -- Cascade delete will handle step tables due to FK constraints
    DELETE FROM company_core WHERE id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Create triggers on the view
DROP TRIGGER IF EXISTS companies_insert ON companies;
DROP TRIGGER IF EXISTS companies_update ON companies;
DROP TRIGGER IF EXISTS companies_delete ON companies;

CREATE TRIGGER companies_insert
    INSTEAD OF INSERT ON companies
    FOR EACH ROW EXECUTE FUNCTION companies_insert_trigger();

CREATE TRIGGER companies_update
    INSTEAD OF UPDATE ON companies
    FOR EACH ROW EXECUTE FUNCTION companies_update_trigger();

CREATE TRIGGER companies_delete
    INSTEAD OF DELETE ON companies
    FOR EACH ROW EXECUTE FUNCTION companies_delete_trigger();

-- =============================================================================
-- STEP 7: UPDATE FOREIGN KEY REFERENCES
-- Point existing FKs to company_core instead of companies_backup_v2024
-- =============================================================================

-- Challenges
ALTER TABLE challenges DROP CONSTRAINT IF EXISTS challenges_company_id_fkey;
ALTER TABLE challenges ADD CONSTRAINT challenges_company_id_fkey
    FOREIGN KEY (company_id) REFERENCES company_core(id) ON DELETE CASCADE;

-- Analyses
ALTER TABLE analyses DROP CONSTRAINT IF EXISTS fk_analyses_company;
ALTER TABLE analyses ADD CONSTRAINT fk_analyses_company
    FOREIGN KEY (company_id) REFERENCES company_core(id) ON DELETE SET NULL;

-- Company submissions
ALTER TABLE company_submissions DROP CONSTRAINT IF EXISTS company_submissions_company_id_fkey;
ALTER TABLE company_submissions ADD CONSTRAINT company_submissions_company_id_fkey
    FOREIGN KEY (company_id) REFERENCES company_core(id) ON DELETE CASCADE;

-- Company enrichment
ALTER TABLE company_enrichment DROP CONSTRAINT IF EXISTS company_enrichment_company_id_fkey;
ALTER TABLE company_enrichment ADD CONSTRAINT company_enrichment_company_id_fkey
    FOREIGN KEY (company_id) REFERENCES company_core(id) ON DELETE CASCADE;

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE company_core IS 'Core company data: user-provided fields and meta info';
COMMENT ON TABLE company_step1_data IS 'Step 1 enrichment: Basic Info (CNPJ registry + basic search)';
COMMENT ON TABLE company_step2_data IS 'Step 2 enrichment: Business Model (website + business search)';
COMMENT ON TABLE company_step3_data IS 'Step 3 enrichment: Competitive Intelligence';
COMMENT ON TABLE company_legacy_data IS 'Legacy fields: deprecated, kept for backwards compatibility';
COMMENT ON VIEW companies IS 'Backwards-compatible view joining all company tables';

COMMIT;
