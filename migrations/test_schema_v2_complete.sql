-- =============================================================================
-- TEST DATABASE SCHEMA - COMPLETE V2 (CURRENT PRODUCTION)
-- =============================================================================
-- Purpose: Create a fresh test database with the CURRENT production schema
-- Date: 2024-12-07
--
-- This schema represents the state AFTER all v2_* migrations have been applied.
-- Use this to create a new test database that matches production.
--
-- TO USE:
-- 1. Create a new PostgreSQL database
-- 2. Connect to it and run this entire file
-- 3. Your test database will match production schema
--
-- TABLES INCLUDED (8 total):
-- - user_profiles (auth)
-- - submissions (entry form)
-- - companies (company data + enrichment)
-- - company_submissions (linking)
-- - challenges (strategic challenges)
-- - analyses (framework results)
-- - analysis_steps (wizard steps)
-- - analysis_versions (version history)
--
-- TABLES NOT INCLUDED (dropped in v2):
-- - enrichments (merged into companies)
-- - reports (merged into analyses)
-- - frameworks (hardcoded in code)
-- - llm_usage_logs (never used)
-- - macro_* tables (hardcoded values)
-- - field_deprecation_config (unused)
-- - company_data_history (unused)
-- - company_field_verifications (unused)
-- =============================================================================

BEGIN;

-- =============================================================================
-- EXTENSIONS
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- ENUMS
-- =============================================================================

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
        CREATE TYPE user_role AS ENUM ('user', 'admin');
    END IF;
END$$;

-- =============================================================================
-- TABLE: user_profiles
-- =============================================================================

CREATE TABLE IF NOT EXISTS user_profiles (
    id UUID NOT NULL PRIMARY KEY,
    email VARCHAR NOT NULL,
    full_name VARCHAR,
    avatar_url TEXT,
    role user_role NOT NULL DEFAULT 'user',
    is_active BOOLEAN DEFAULT true,
    organization_name VARCHAR,
    job_title VARCHAR,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    -- Note: In Supabase, add: CONSTRAINT user_profiles_id_fkey FOREIGN KEY (id) REFERENCES auth.users(id)
);

-- =============================================================================
-- TABLE: submissions
-- =============================================================================

CREATE TABLE IF NOT EXISTS submissions (
    id UUID NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
    company_name VARCHAR NOT NULL,
    company_website VARCHAR,
    company_industry VARCHAR,
    company_size VARCHAR,
    company_location VARCHAR,
    contact_name VARCHAR NOT NULL,
    contact_email VARCHAR NOT NULL,
    contact_phone VARCHAR,
    contact_position VARCHAR,
    target_market TEXT,
    annual_revenue_min NUMERIC,
    annual_revenue_max NUMERIC,
    funding_stage VARCHAR,
    additional_notes TEXT,
    linkedin_url VARCHAR,
    twitter_handle VARCHAR,
    cnpj VARCHAR,
    user_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_submissions_user FOREIGN KEY (user_id) REFERENCES user_profiles(id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_submissions_user_id ON submissions(user_id);
CREATE INDEX IF NOT EXISTS idx_submissions_created_at ON submissions(created_at);
CREATE INDEX IF NOT EXISTS idx_submissions_email_lower ON submissions (lower(contact_email));

-- =============================================================================
-- TABLE: companies
-- =============================================================================

CREATE TABLE IF NOT EXISTS companies (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
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
    digital_maturity INTEGER CHECK (digital_maturity IS NULL OR digital_maturity BETWEEN 1 AND 10),
    strengths JSONB DEFAULT '[]',
    weaknesses JSONB DEFAULT '[]',
    linkedin_url TEXT,
    twitter_handle TEXT,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    allowed_users UUID[] NOT NULL DEFAULT '{}',
    owner_id UUID,
    -- Enrichment tracking (added in v2_007)
    enrichment_status VARCHAR(20) DEFAULT 'pending'
        CHECK (enrichment_status IN ('pending', 'processing', 'completed', 'failed')),
    enrichment_completed_at TIMESTAMPTZ,
    enrichment_error TEXT,
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_companies_cnpj ON companies(cnpj);
CREATE INDEX IF NOT EXISTS idx_companies_owner_id ON companies(owner_id);
CREATE INDEX IF NOT EXISTS idx_companies_enrichment_status ON companies(enrichment_status);

-- =============================================================================
-- TABLE: company_submissions (linking table)
-- =============================================================================

CREATE TABLE IF NOT EXISTS company_submissions (
    company_id UUID NOT NULL,
    submission_id UUID NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT true,
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    linked_by UUID,
    PRIMARY KEY (company_id, submission_id),
    CONSTRAINT company_submissions_company_id_fkey FOREIGN KEY (company_id) REFERENCES companies(id),
    CONSTRAINT company_submissions_submission_id_fkey FOREIGN KEY (submission_id) REFERENCES submissions(id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_company_submissions_submission_id ON company_submissions(submission_id);

-- =============================================================================
-- TABLE: challenges (added in v2_008)
-- =============================================================================

CREATE TABLE IF NOT EXISTS challenges (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    company_id UUID NOT NULL,
    challenge_category VARCHAR(50) NOT NULL CHECK (challenge_category IN (
        'growth', 'transform', 'transition', 'compete', 'funding'
    )),
    challenge_type VARCHAR(50) NOT NULL CHECK (challenge_type IN (
        -- Growth types
        'growth_organic', 'growth_geographic', 'growth_segment', 'growth_product', 'growth_channel',
        -- Transform types
        'transform_digital', 'transform_model', 'transform_culture', 'transform_operational',
        -- Transition types
        'transition_succession', 'transition_exit', 'transition_merger', 'transition_turnaround',
        -- Compete types
        'compete_differentiate', 'compete_defend', 'compete_reposition',
        -- Funding types
        'funding_raise', 'funding_debt', 'funding_ipo'
    )),
    business_challenge TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT challenges_company_id_fkey FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_challenges_company_id ON challenges(company_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_challenges_category ON challenges(challenge_category);
CREATE INDEX IF NOT EXISTS idx_challenges_type ON challenges(challenge_type);
CREATE INDEX IF NOT EXISTS idx_challenges_deleted_at ON challenges(deleted_at);

-- =============================================================================
-- TABLE: analyses
-- =============================================================================

CREATE TABLE IF NOT EXISTS analyses (
    id UUID NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
    submission_id UUID,  -- Nullable: historical reference only
    company_id UUID NOT NULL,
    challenge_id UUID NOT NULL,  -- Required: links to strategic challenge
    -- Framework results (consolidated JSONB)
    framework_results JSONB DEFAULT '{}',
    -- Wizard state (added in v2_004)
    current_step INTEGER DEFAULT 0,
    wizard_mode BOOLEAN DEFAULT true,
    steps_completed TEXT[] DEFAULT '{}',
    -- Version tracking (added in v2_011)
    version INTEGER DEFAULT 1,
    version_notes TEXT,
    -- Status and workflow
    status VARCHAR NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'approved', 'sent', 'failed')),
    error_message TEXT,
    processing_time_ms BIGINT,
    -- Visibility and access
    is_visible_to_user BOOLEAN NOT NULL DEFAULT false,
    is_public BOOLEAN NOT NULL DEFAULT false,
    access_code VARCHAR,
    access_code_created_at TIMESTAMPTZ,
    -- PDF generation
    pdf_url TEXT,
    pdf_generated_at TIMESTAMPTZ,
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    -- Foreign keys
    CONSTRAINT analyses_submission_id_fkey FOREIGN KEY (submission_id) REFERENCES submissions(id),
    CONSTRAINT fk_analyses_company FOREIGN KEY (company_id) REFERENCES companies(id),
    CONSTRAINT analyses_challenge_id_fkey FOREIGN KEY (challenge_id) REFERENCES challenges(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_analyses_submission_id ON analyses(submission_id);
CREATE INDEX IF NOT EXISTS idx_analyses_status ON analyses(status);
CREATE INDEX IF NOT EXISTS idx_analyses_company_id ON analyses(company_id);
CREATE INDEX IF NOT EXISTS idx_analyses_access_code ON analyses(access_code);
CREATE INDEX IF NOT EXISTS idx_analyses_challenge_id ON analyses(challenge_id);
CREATE INDEX IF NOT EXISTS idx_analyses_framework_results ON analyses USING gin(framework_results);

-- =============================================================================
-- TABLE: analysis_steps (wizard - added in v2_004)
-- =============================================================================

CREATE TABLE IF NOT EXISTS analysis_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analysis_id UUID NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,
    step_number INTEGER NOT NULL,
    framework_code VARCHAR(50) NOT NULL,
    -- Status: pending | generating | generated | approved | failed
    status VARCHAR(20) DEFAULT 'pending',
    -- AI-generated output
    output JSONB,
    -- Human refinement
    human_context TEXT,
    refinement_notes TEXT,
    -- Clarifying questions
    clarifying_questions JSONB,
    human_answers JSONB,
    -- Iteration tracking
    iteration_count INTEGER DEFAULT 0,
    -- Error tracking
    error_message TEXT,
    processing_time_ms INTEGER,
    -- Version history (added in v2_011)
    previous_outputs JSONB DEFAULT '[]',
    -- Timestamps
    generated_at TIMESTAMPTZ,
    approved_at TIMESTAMPTZ,
    approved_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    -- Unique constraint
    UNIQUE(analysis_id, framework_code)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_analysis_steps_analysis ON analysis_steps(analysis_id);
CREATE INDEX IF NOT EXISTS idx_analysis_steps_status ON analysis_steps(status);
CREATE INDEX IF NOT EXISTS idx_analysis_steps_analysis_step ON analysis_steps(analysis_id, step_number);

-- =============================================================================
-- TABLE: analysis_versions (added in v2_011)
-- =============================================================================

CREATE TABLE IF NOT EXISTS analysis_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analysis_id UUID NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    framework_results JSONB NOT NULL,
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    change_summary TEXT,
    UNIQUE(analysis_id, version)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_analysis_versions_analysis ON analysis_versions(analysis_id);
CREATE INDEX IF NOT EXISTS idx_analysis_versions_version ON analysis_versions(analysis_id, version DESC);

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE submissions IS 'Public submission form - entry point that creates Company + first Challenge';
COMMENT ON TABLE companies IS 'Company records with inline enrichment. Enrichment data stored directly.';
COMMENT ON TABLE challenges IS 'Strategic business challenges for companies - defines the problem to solve';
COMMENT ON TABLE analyses IS 'Analysis results - framework data stored in framework_results JSONB column';
COMMENT ON TABLE analysis_steps IS 'Tracks individual framework steps in the wizard. Each step requires human approval.';
COMMENT ON TABLE analysis_versions IS 'Immutable history of all analysis versions for audit trail and rollback';

COMMENT ON COLUMN companies.enrichment_status IS 'One-time enrichment status: pending, processing, completed, or failed';
COMMENT ON COLUMN challenges.challenge_category IS 'Top-level: growth, transform, transition, compete, funding';
COMMENT ON COLUMN challenges.challenge_type IS 'Specific type: growth_organic, transition_exit, etc.';
COMMENT ON COLUMN analyses.submission_id IS 'Historical reference to originating submission. Nullable.';
COMMENT ON COLUMN analyses.challenge_id IS 'Required: The challenge this analysis addresses';
COMMENT ON COLUMN analyses.framework_results IS 'Generic storage for all framework results: {"pestel": {...}, "swot": {...}, ...}';
COMMENT ON COLUMN analysis_steps.human_context IS 'Additional context from human to inject into AI regeneration.';
COMMENT ON COLUMN analysis_steps.previous_outputs IS 'Array of previous outputs from refinements.';

COMMIT;

-- =============================================================================
-- VERIFICATION QUERIES (run after applying)
-- =============================================================================
-- \dt                          -- List all tables (should be 8)
-- \d analyses                  -- Verify analyses structure
-- \d challenges                -- Verify challenges structure
-- \d analysis_steps            -- Verify wizard structure
-- SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'analyses';  -- Check indexes
