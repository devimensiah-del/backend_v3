-- Migration: v2_025_add_cnpj_normalized
-- Description: Add normalized CNPJ column for lookups (NOT unique - different users can have same CNPJ)
-- Date: 2024-12-21

-- NOTE: 'companies' is a VIEW - the actual table is 'company_core'

BEGIN;

-- Step 1: Add deleted_at column to company_core (for soft deletes during merge)
ALTER TABLE company_core
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Step 2: Add generated column for normalized CNPJ (digits only)
-- This strips all non-digit characters from CNPJ for consistent matching
ALTER TABLE company_core
ADD COLUMN IF NOT EXISTS cnpj_normalized VARCHAR(14)
GENERATED ALWAYS AS (REGEXP_REPLACE(cnpj, '[^0-9]', '', 'g')) STORED;

-- Step 3: Add index for fast lookups (NOT UNIQUE - multiple companies can share same CNPJ)
-- Different users are allowed to create companies with the same CNPJ
-- Uniqueness is enforced at (email + cnpj) level, not company level
CREATE INDEX IF NOT EXISTS idx_company_core_cnpj_normalized
ON company_core (cnpj_normalized)
WHERE cnpj_normalized IS NOT NULL AND cnpj_normalized != '';

-- Step 4: Recreate the 'companies' view to include the new columns
DROP VIEW IF EXISTS companies;

CREATE OR REPLACE VIEW companies AS
SELECT
    -- Core fields
    c.id,
    c.name,
    c.cnpj,
    c.cnpj_normalized,
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
    c.deleted_at,

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

COMMIT;
