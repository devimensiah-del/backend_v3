-- Reset enrichment data for Koro Sports (2e153503-2e34-4785-9fab-54d3f6851f0e)
-- Run this to test the new EnrichmentContext-based prompts

BEGIN;

-- 1. Reset Step 1 enriched fields in companies table
UPDATE companies SET
    -- Step 1 fields
    foundation_year = NULL,
    legal_name = NULL,
    headquarters = NULL,
    employees_range = NULL,
    trade_name = NULL,
    phone = NULL,
    email = NULL,
    cnae_primary = NULL,
    cnae_codes = '[]',
    capital_social = NULL,
    partners = '[]',
    cnpj_verified = false,
    linkedin_url = NULL,
    twitter_handle = NULL,
    instagram_url = NULL,
    facebook_url = NULL,
    key_executives = '[]',
    -- Step 2 fields
    business_model = NULL,
    sector = NULL,
    main_products = '[]',
    pricing_model = NULL,
    target_audience = NULL,
    value_proposition = NULL,
    customer_segments = '[]',
    unique_selling_points = '[]',
    geographic_regions = '[]',
    -- service_areas not yet migrated
    -- Step 3 fields
    competitors = '[]',
    competitor_details = '[]',
    industry_growth_rate = NULL,
    industry_trends = '[]',
    market_concentration = NULL,
    regulatory_context = NULL,
    market_share_status = NULL,
    recent_news = '[]',
    enrichment_sources = '[]',
    -- Reset enrichment status
    enrichment_status = 'pending',
    enrichment_completed_at = NULL,
    enrichment_error = NULL,
    updated_at = NOW()
WHERE id = '2e153503-2e34-4785-9fab-54d3f6851f0e';

-- 2. Reset company_enrichments table (Step 1/2/3 status and data)
UPDATE company_enrichments SET
    step1_status = 'pending',
    step1_data = NULL,
    step1_error = NULL,
    step1_completed_at = NULL,
    step2_status = 'pending',
    step2_data = NULL,
    step2_error = NULL,
    step2_completed_at = NULL,
    step3_status = 'pending',
    step3_data = NULL,
    step3_error = NULL,
    step3_completed_at = NULL,
    updated_at = NOW()
WHERE company_id = '2e153503-2e34-4785-9fab-54d3f6851f0e';

COMMIT;

-- Verify reset
SELECT id, name, enrichment_status, foundation_year, business_model, competitors
FROM companies
WHERE id = '2e153503-2e34-4785-9fab-54d3f6851f0e';

SELECT company_id, step1_status, step2_status, step3_status
FROM company_enrichments
WHERE company_id = '2e153503-2e34-4785-9fab-54d3f6851f0e';
