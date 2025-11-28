-- Migration: Companies Table
-- Purpose: Create companies table to persist company data across submissions
-- Each submission creates a new company record with full data provenance tracking

-- ============================================
-- 1. Create ENUM for data source tracking
-- ============================================
CREATE TYPE company_data_source AS ENUM ('submission', 'enrichment', 'manual');

-- ============================================
-- 2. Create companies table (main entity)
-- ============================================
CREATE TABLE companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Core identifiers (from submission)
    name TEXT NOT NULL,
    cnpj TEXT,
    website TEXT,

    -- Business context (from submission)
    industry TEXT,
    company_size TEXT,
    location TEXT,
    target_market TEXT,
    funding_stage TEXT,
    annual_revenue_min NUMERIC,
    annual_revenue_max NUMERIC,

    -- Enriched data (from AI enrichment)
    foundation_year TEXT,
    legal_name TEXT,
    headquarters TEXT,
    sector TEXT,
    target_audience TEXT,
    value_proposition TEXT,
    employees_range TEXT,
    revenue_estimate TEXT,
    business_model TEXT,
    competitors JSONB DEFAULT '[]',
    market_share_status TEXT,
    digital_maturity INTEGER,
    strengths JSONB DEFAULT '[]',
    weaknesses JSONB DEFAULT '[]',

    -- Social links
    linkedin_url TEXT,
    twitter_handle TEXT,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient lookups
CREATE INDEX idx_companies_name ON companies(name);
CREATE INDEX idx_companies_cnpj ON companies(cnpj) WHERE cnpj IS NOT NULL;
CREATE INDEX idx_companies_industry ON companies(industry) WHERE industry IS NOT NULL;
CREATE INDEX idx_companies_created ON companies(created_at DESC);

-- ============================================
-- 3. Create company_data_history table
-- Tracks ALL changes with provenance (source + timestamp)
-- ============================================
CREATE TABLE company_data_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- What changed
    field_name TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT,

    -- Provenance
    source company_data_source NOT NULL,
    source_id UUID,                   -- submission_id or enrichment_id (depending on source)
    changed_by UUID,                  -- user_id if manual edit

    -- Timestamp
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient history lookups
CREATE INDEX idx_company_history_company ON company_data_history(company_id);
CREATE INDEX idx_company_history_field ON company_data_history(company_id, field_name);
CREATE INDEX idx_company_history_source ON company_data_history(source_id) WHERE source_id IS NOT NULL;
CREATE INDEX idx_company_history_time ON company_data_history(company_id, changed_at DESC);

-- ============================================
-- 4. Create company_submissions junction table
-- Links companies to their submissions (many-to-many for future merging)
-- ============================================
CREATE TABLE company_submissions (
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    is_primary BOOLEAN NOT NULL DEFAULT true,
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    linked_by UUID,                   -- user_id if manually linked

    PRIMARY KEY (company_id, submission_id)
);

CREATE INDEX idx_company_submissions_submission ON company_submissions(submission_id);

-- ============================================
-- 5. Migrate existing submissions to companies
-- ============================================

-- Create company records from existing submissions
INSERT INTO companies (
    id,
    name,
    cnpj,
    website,
    industry,
    company_size,
    location,
    target_market,
    funding_stage,
    annual_revenue_min,
    annual_revenue_max,
    linkedin_url,
    twitter_handle,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid() AS id,
    s.company_name,
    s.cnpj,
    s.company_website,
    s.company_industry,
    s.company_size,
    s.company_location,
    s.target_market,
    s.funding_stage,
    s.annual_revenue_min,
    s.annual_revenue_max,
    s.linkedin_url,
    s.twitter_handle,
    s.created_at,
    s.created_at AS updated_at
FROM submissions s
WHERE s.company_name IS NOT NULL
  AND s.deleted_at IS NULL;

-- Link the created companies to their submissions
-- Using a temporary table to handle the mapping
WITH company_mapping AS (
    SELECT
        c.id AS company_id,
        s.id AS submission_id,
        s.created_at
    FROM companies c
    JOIN submissions s ON
        s.company_name = c.name
        AND s.created_at = c.created_at
        AND s.deleted_at IS NULL
)
INSERT INTO company_submissions (company_id, submission_id, is_primary, linked_at)
SELECT
    company_id,
    submission_id,
    true,
    NOW()
FROM company_mapping;

-- ============================================
-- 6. Create initial history entries for migrated companies
-- Records that data came from submission
-- ============================================

-- History for company name (always present)
INSERT INTO company_data_history (company_id, field_name, new_value, source, source_id, changed_at)
SELECT
    c.id,
    'name',
    c.name,
    'submission',
    cs.submission_id,
    c.created_at
FROM companies c
JOIN company_submissions cs ON cs.company_id = c.id AND cs.is_primary = true;

-- History for cnpj (if present)
INSERT INTO company_data_history (company_id, field_name, new_value, source, source_id, changed_at)
SELECT
    c.id,
    'cnpj',
    c.cnpj,
    'submission',
    cs.submission_id,
    c.created_at
FROM companies c
JOIN company_submissions cs ON cs.company_id = c.id AND cs.is_primary = true
WHERE c.cnpj IS NOT NULL;

-- History for website (if present)
INSERT INTO company_data_history (company_id, field_name, new_value, source, source_id, changed_at)
SELECT
    c.id,
    'website',
    c.website,
    'submission',
    cs.submission_id,
    c.created_at
FROM companies c
JOIN company_submissions cs ON cs.company_id = c.id AND cs.is_primary = true
WHERE c.website IS NOT NULL;

-- History for industry (if present)
INSERT INTO company_data_history (company_id, field_name, new_value, source, source_id, changed_at)
SELECT
    c.id,
    'industry',
    c.industry,
    'submission',
    cs.submission_id,
    c.created_at
FROM companies c
JOIN company_submissions cs ON cs.company_id = c.id AND cs.is_primary = true
WHERE c.industry IS NOT NULL;

-- History for company_size (if present)
INSERT INTO company_data_history (company_id, field_name, new_value, source, source_id, changed_at)
SELECT
    c.id,
    'company_size',
    c.company_size,
    'submission',
    cs.submission_id,
    c.created_at
FROM companies c
JOIN company_submissions cs ON cs.company_id = c.id AND cs.is_primary = true
WHERE c.company_size IS NOT NULL;

-- History for location (if present)
INSERT INTO company_data_history (company_id, field_name, new_value, source, source_id, changed_at)
SELECT
    c.id,
    'location',
    c.location,
    'submission',
    cs.submission_id,
    c.created_at
FROM companies c
JOIN company_submissions cs ON cs.company_id = c.id AND cs.is_primary = true
WHERE c.location IS NOT NULL;

-- History for target_market (if present)
INSERT INTO company_data_history (company_id, field_name, new_value, source, source_id, changed_at)
SELECT
    c.id,
    'target_market',
    c.target_market,
    'submission',
    cs.submission_id,
    c.created_at
FROM companies c
JOIN company_submissions cs ON cs.company_id = c.id AND cs.is_primary = true
WHERE c.target_market IS NOT NULL;

-- History for funding_stage (if present)
INSERT INTO company_data_history (company_id, field_name, new_value, source, source_id, changed_at)
SELECT
    c.id,
    'funding_stage',
    c.funding_stage,
    'submission',
    cs.submission_id,
    c.created_at
FROM companies c
JOIN company_submissions cs ON cs.company_id = c.id AND cs.is_primary = true
WHERE c.funding_stage IS NOT NULL;

-- ============================================
-- 7. Now update companies with enrichment data where available
-- ============================================

-- Update companies with enrichment data (discovered_data fields)
UPDATE companies c
SET
    foundation_year = COALESCE(
        (e.data->'discovered_data'->>'foundation_year'),
        (e.data->'profile_overview'->>'foundation_year')
    ),
    legal_name = (e.data->'profile_overview'->>'legal_name'),
    headquarters = (e.data->'profile_overview'->>'headquarters'),
    sector = (e.data->'market_position'->>'sector'),
    target_audience = (e.data->'market_position'->>'target_audience'),
    value_proposition = (e.data->'market_position'->>'value_proposition'),
    employees_range = (e.data->'financials'->>'employees_range'),
    revenue_estimate = (e.data->'financials'->>'revenue_estimate'),
    business_model = (e.data->'financials'->>'business_model'),
    competitors = COALESCE(e.data->'competitive_landscape'->'competitors', '[]'::jsonb),
    market_share_status = (e.data->'competitive_landscape'->>'market_share_status'),
    digital_maturity = (e.data->'strategic_assessment'->>'digital_maturity')::integer,
    strengths = COALESCE(e.data->'strategic_assessment'->'strengths', '[]'::jsonb),
    weaknesses = COALESCE(e.data->'strategic_assessment'->'weaknesses', '[]'::jsonb),
    -- Also update basic fields if discovered
    cnpj = COALESCE(c.cnpj, (e.data->'discovered_data'->>'cnpj')),
    website = COALESCE(c.website, (e.data->'discovered_data'->>'website')),
    linkedin_url = COALESCE(c.linkedin_url, (e.data->'discovered_data'->>'linkedin_url')),
    twitter_handle = COALESCE(c.twitter_handle, (e.data->'discovered_data'->>'twitter_handle')),
    updated_at = NOW()
FROM company_submissions cs
JOIN enrichments e ON e.submission_id = cs.submission_id
WHERE c.id = cs.company_id
  AND cs.is_primary = true
  AND e.status IN ('completed', 'approved')
  AND e.data IS NOT NULL
  AND e.data != '{}'::jsonb;

-- Add history entries for enriched fields
INSERT INTO company_data_history (company_id, field_name, new_value, source, source_id, changed_at)
SELECT
    c.id,
    'legal_name',
    c.legal_name,
    'enrichment',
    e.id,
    e.completed_at
FROM companies c
JOIN company_submissions cs ON cs.company_id = c.id AND cs.is_primary = true
JOIN enrichments e ON e.submission_id = cs.submission_id
WHERE c.legal_name IS NOT NULL
  AND e.status IN ('completed', 'approved')
  AND e.data IS NOT NULL;

INSERT INTO company_data_history (company_id, field_name, new_value, source, source_id, changed_at)
SELECT
    c.id,
    'sector',
    c.sector,
    'enrichment',
    e.id,
    e.completed_at
FROM companies c
JOIN company_submissions cs ON cs.company_id = c.id AND cs.is_primary = true
JOIN enrichments e ON e.submission_id = cs.submission_id
WHERE c.sector IS NOT NULL
  AND e.status IN ('completed', 'approved')
  AND e.data IS NOT NULL;

INSERT INTO company_data_history (company_id, field_name, new_value, source, source_id, changed_at)
SELECT
    c.id,
    'target_audience',
    c.target_audience,
    'enrichment',
    e.id,
    e.completed_at
FROM companies c
JOIN company_submissions cs ON cs.company_id = c.id AND cs.is_primary = true
JOIN enrichments e ON e.submission_id = cs.submission_id
WHERE c.target_audience IS NOT NULL
  AND e.status IN ('completed', 'approved')
  AND e.data IS NOT NULL;

-- Add comments for documentation
COMMENT ON TABLE companies IS 'Stores company data across submissions with history tracking';
COMMENT ON TABLE company_data_history IS 'Full audit trail of all company data changes with provenance';
COMMENT ON TABLE company_submissions IS 'Junction table linking companies to submissions (supports future merging)';
COMMENT ON COLUMN company_data_history.source IS 'Where this data came from: submission, enrichment, or manual edit';
COMMENT ON COLUMN company_data_history.source_id IS 'FK to submissions.id or enrichments.id depending on source';
COMMENT ON COLUMN company_submissions.is_primary IS 'True if this is the primary submission that created the company';
