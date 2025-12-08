-- =============================================================================
-- V2 ROLLBACK TO BASELINE
-- =============================================================================
-- Purpose: Rollback ALL v2_* migrations back to 000_baseline.sql state
-- Date: 2024-12-07
--
-- WARNING: This is a DESTRUCTIVE migration that will:
-- - Drop v2 tables (challenges, analysis_steps, analysis_versions)
-- - Restore dropped tables (enrichments, reports, frameworks, llm_usage_logs,
--   macro_*, field_deprecation_config, company_data_history, company_field_verifications)
-- - Restore dropped columns on analyses (framework columns, enrichment_id, etc.)
-- - Restore dropped columns on submissions (status, business_challenge)
-- - Remove columns added by v2 (framework_results, wizard columns, etc.)
--
-- DATA LOSS:
-- - All data in v2-specific tables (challenges, analysis_steps, analysis_versions) will be LOST
-- - framework_results JSONB data will be LOST (baseline uses individual columns)
-- - wizard progress will be LOST
--
-- BEFORE RUNNING:
-- 1. BACKUP YOUR DATABASE
-- 2. Test on a staging environment first
-- 3. Have your application ready to run baseline-compatible code
-- =============================================================================

BEGIN;

-- =============================================================================
-- PART 1: Drop v2-added indexes
-- =============================================================================

DROP INDEX IF EXISTS idx_companies_enrichment_status;
DROP INDEX IF EXISTS idx_analyses_challenge_id;
DROP INDEX IF EXISTS idx_analysis_steps_analysis;
DROP INDEX IF EXISTS idx_analysis_steps_status;
DROP INDEX IF EXISTS idx_analysis_steps_analysis_step;
DROP INDEX IF EXISTS idx_analysis_versions_analysis;
DROP INDEX IF EXISTS idx_analysis_versions_version;
DROP INDEX IF EXISTS idx_challenges_company_id;
DROP INDEX IF EXISTS idx_challenges_category;
DROP INDEX IF EXISTS idx_challenges_type;
DROP INDEX IF EXISTS idx_challenges_deleted_at;
DROP INDEX IF EXISTS idx_analyses_framework_results;
DROP INDEX IF EXISTS idx_frameworks_code;
DROP INDEX IF EXISTS idx_frameworks_category;
DROP INDEX IF EXISTS idx_frameworks_active;
DROP INDEX IF EXISTS idx_submissions_email_lower;
DROP INDEX IF EXISTS idx_company_submissions_submission_id;

-- =============================================================================
-- PART 2: Drop v2-added tables
-- =============================================================================

DROP TABLE IF EXISTS analysis_versions CASCADE;
DROP TABLE IF EXISTS analysis_steps CASCADE;
DROP TABLE IF EXISTS challenges CASCADE;

-- =============================================================================
-- PART 3: Drop v2-added constraints on analyses
-- =============================================================================

ALTER TABLE analyses DROP CONSTRAINT IF EXISTS analyses_challenge_id_fkey;

-- =============================================================================
-- PART 4: Restore dropped columns on analyses table
-- =============================================================================

-- Restore legacy framework columns (from v2_003)
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS pestel JSONB;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS porter JSONB;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS swot JSONB;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS tam_sam_som JSONB;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS benchmarking JSONB;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS blue_ocean JSONB;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS growth_hacking JSONB;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS scenarios JSONB;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS okrs JSONB;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS bsc JSONB;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS decision_matrix JSONB;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS synthesis JSONB;

-- Restore enrichment_id column (from v2_007)
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS enrichment_id UUID;

-- Restore is_blurred (from v2_015)
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS is_blurred BOOLEAN NOT NULL DEFAULT true;

-- Restore approved_at/approved_by/sent_at/sent_to (from v2_015)
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS approved_at TIMESTAMPTZ;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS approved_by UUID;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS sent_at TIMESTAMPTZ;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS sent_to VARCHAR;

-- =============================================================================
-- PART 5: Remove v2-added columns from analyses
-- =============================================================================

ALTER TABLE analyses DROP COLUMN IF EXISTS framework_results;
ALTER TABLE analyses DROP COLUMN IF EXISTS current_step;
ALTER TABLE analyses DROP COLUMN IF EXISTS wizard_mode;
ALTER TABLE analyses DROP COLUMN IF EXISTS steps_completed;
ALTER TABLE analyses DROP COLUMN IF EXISTS challenge_id;
ALTER TABLE analyses DROP COLUMN IF EXISTS version;
ALTER TABLE analyses DROP COLUMN IF EXISTS version_notes;

-- Make submission_id NOT NULL again (was relaxed in v2_008)
-- Note: This might fail if there are NULL values - handle data first
UPDATE analyses SET submission_id = NULL WHERE submission_id IS NULL;
-- Cannot set NOT NULL if there are NULL values - commented out for safety
-- ALTER TABLE analyses ALTER COLUMN submission_id SET NOT NULL;

-- Make company_id nullable again (v2_012 made it NOT NULL)
ALTER TABLE analyses ALTER COLUMN company_id DROP NOT NULL;

-- =============================================================================
-- PART 6: Restore dropped columns on companies table
-- =============================================================================

-- Restore enrichment columns added in v2_005 then dropped in v2_012
ALTER TABLE companies ADD COLUMN IF NOT EXISTS sector_description TEXT;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS sector_growth_rate VARCHAR(50);
ALTER TABLE companies ADD COLUMN IF NOT EXISTS sector_trends JSONB;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS key_industry_trends TEXT[];
ALTER TABLE companies ADD COLUMN IF NOT EXISTS macro_context JSONB;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS recent_news JSONB;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS competitive_landscape JSONB;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS data_confidence VARCHAR(20);
ALTER TABLE companies ADD COLUMN IF NOT EXISTS data_gaps TEXT[];
ALTER TABLE companies ADD COLUMN IF NOT EXISTS enriched_at TIMESTAMPTZ;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS enrichment_source VARCHAR(50);

-- =============================================================================
-- PART 7: Remove v2-added columns from companies
-- =============================================================================

ALTER TABLE companies DROP COLUMN IF EXISTS enrichment_status;
ALTER TABLE companies DROP COLUMN IF EXISTS enrichment_completed_at;
ALTER TABLE companies DROP COLUMN IF EXISTS enrichment_error;

-- =============================================================================
-- PART 8: Restore submissions table columns
-- =============================================================================

-- Restore status column (dropped in v2_008)
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS status VARCHAR NOT NULL DEFAULT 'received'
    CHECK (status = 'received');

-- Restore business_challenge (moved to challenges in v2_008)
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS business_challenge TEXT;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions(status);

-- =============================================================================
-- PART 9: Recreate enrichments table (dropped in v2_007)
-- =============================================================================

CREATE TABLE IF NOT EXISTS enrichments (
    id UUID NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
    submission_id UUID NOT NULL UNIQUE,
    status VARCHAR NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'approved', 'failed')),
    progress INTEGER NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
    current_step TEXT,
    sources_status JSONB,
    sources_used TEXT[],
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    is_locked BOOLEAN DEFAULT false,
    data JSONB DEFAULT '{}',
    auto_trigger_analysis BOOLEAN NOT NULL DEFAULT false,
    approved_at TIMESTAMPTZ,
    company_id UUID,
    raw_ai_response JSONB,
    processing_time_ms INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT enrichments_submission_id_fkey FOREIGN KEY (submission_id) REFERENCES submissions(id),
    CONSTRAINT fk_enrichments_company FOREIGN KEY (company_id) REFERENCES companies(id)
);

CREATE INDEX IF NOT EXISTS idx_enrichments_submission_id ON enrichments(submission_id);
CREATE INDEX IF NOT EXISTS idx_enrichments_status ON enrichments(status);
CREATE INDEX IF NOT EXISTS idx_enrichments_company_id ON enrichments(company_id);

-- Add foreign key from analyses to enrichments
ALTER TABLE analyses ADD CONSTRAINT analyses_enrichment_id_fkey
    FOREIGN KEY (enrichment_id) REFERENCES enrichments(id);

CREATE INDEX IF NOT EXISTS idx_analyses_enrichment_id ON analyses(enrichment_id);

-- =============================================================================
-- PART 10: Recreate reports table (dropped in v2_003)
-- =============================================================================

CREATE TABLE IF NOT EXISTS reports (
    id UUID NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
    submission_id UUID NOT NULL UNIQUE,
    analysis_id UUID NOT NULL,
    -- Page content
    cover_page TEXT,
    executive_summary TEXT,
    table_of_contents TEXT,
    swot_page TEXT,
    pestel_page TEXT,
    porter_page TEXT,
    okr_page TEXT,
    strategic_priorities_page TEXT,
    risks_mitigation_page TEXT,
    appendix_page TEXT,
    tam_sam_som_page TEXT,
    benchmarking_page TEXT,
    blue_ocean_page TEXT,
    growth_hacking_page TEXT,
    scenarios_page TEXT,
    bsc_page TEXT,
    decision_matrix_page TEXT,
    divider_part1_page TEXT,
    pestel_pes_page TEXT,
    pestel_tel_page TEXT,
    divider_part2_page TEXT,
    divider_part3_page TEXT,
    growth_loops_page TEXT,
    divider_part4_page TEXT,
    recommendations_page TEXT,
    roadmap_page TEXT,
    financial_projections_page TEXT,
    risk_assessment_page TEXT,
    -- PDF generation
    pdf_url VARCHAR,
    pdf_generated_at TIMESTAMPTZ,
    pdf_generation_status VARCHAR NOT NULL DEFAULT 'pending' CHECK (pdf_generation_status IN ('pending', 'processing', 'completed', 'failed')),
    -- Status
    status VARCHAR NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    error_message TEXT,
    generation_time_ms BIGINT,
    total_pages INTEGER NOT NULL DEFAULT 24,
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT reports_submission_id_fkey FOREIGN KEY (submission_id) REFERENCES submissions(id),
    CONSTRAINT reports_analysis_id_fkey FOREIGN KEY (analysis_id) REFERENCES analyses(id)
);

CREATE INDEX IF NOT EXISTS idx_reports_submission_id ON reports(submission_id);
CREATE INDEX IF NOT EXISTS idx_reports_analysis_id ON reports(analysis_id);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status);

-- =============================================================================
-- PART 11: Recreate frameworks table (dropped in v2_015)
-- =============================================================================

CREATE TABLE IF NOT EXISTS frameworks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    name_pt VARCHAR(100) NOT NULL,
    description TEXT,
    description_pt TEXT,
    category VARCHAR(50) NOT NULL,
    layer_order INTEGER NOT NULL DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    requires_enrichment BOOLEAN DEFAULT true,
    timeout_seconds INTEGER DEFAULT 60,
    prompt_template TEXT NOT NULL,
    output_schema JSONB NOT NULL,
    preferred_model VARCHAR(50),
    temperature DECIMAL(3,2) DEFAULT 0.7,
    depends_on TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_frameworks_code ON frameworks(code);
CREATE INDEX IF NOT EXISTS idx_frameworks_category ON frameworks(category);
CREATE INDEX IF NOT EXISTS idx_frameworks_active ON frameworks(is_active) WHERE is_active = true;

-- =============================================================================
-- PART 12: Recreate llm_usage_logs table (dropped in v2_015)
-- =============================================================================

CREATE TABLE IF NOT EXISTS llm_usage_logs (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    submission_id UUID,
    framework_name TEXT NOT NULL,
    model_used TEXT NOT NULL,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    cost_usd NUMERIC NOT NULL DEFAULT 0,
    latency_ms INTEGER,
    is_fallback BOOLEAN NOT NULL DEFAULT false,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT llm_usage_logs_submission_id_fkey FOREIGN KEY (submission_id) REFERENCES submissions(id)
);

CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_submission ON llm_usage_logs(submission_id);
CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_created ON llm_usage_logs(created_at);

-- =============================================================================
-- PART 13: Recreate macroeconomics tables (dropped in v2_015)
-- =============================================================================

CREATE TABLE IF NOT EXISTS macro_data_sources (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    code VARCHAR NOT NULL UNIQUE,
    name VARCHAR NOT NULL,
    base_url TEXT,
    priority SMALLINT DEFAULT 1,
    is_authoritative BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS macro_indicator_types (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    code VARCHAR NOT NULL UNIQUE,
    name VARCHAR NOT NULL,
    category VARCHAR NOT NULL,
    unit VARCHAR,
    frequency VARCHAR NOT NULL,
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS macro_indicator_sources (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    indicator_type_id UUID NOT NULL,
    source_id UUID NOT NULL,
    priority SMALLINT DEFAULT 1,
    endpoint_config JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    last_success_at TIMESTAMPTZ,
    consecutive_failures SMALLINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT macro_indicator_sources_indicator_type_id_fkey FOREIGN KEY (indicator_type_id) REFERENCES macro_indicator_types(id),
    CONSTRAINT macro_indicator_sources_source_id_fkey FOREIGN KEY (source_id) REFERENCES macro_data_sources(id)
);

CREATE TABLE IF NOT EXISTS macro_indicator_values (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    indicator_type_id UUID NOT NULL,
    source_id UUID NOT NULL,
    value NUMERIC NOT NULL,
    effective_date DATE NOT NULL,
    reference_period VARCHAR,
    raw_response JSONB,
    fetched_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT macro_indicator_values_indicator_type_id_fkey FOREIGN KEY (indicator_type_id) REFERENCES macro_indicator_types(id),
    CONSTRAINT macro_indicator_values_source_id_fkey FOREIGN KEY (source_id) REFERENCES macro_data_sources(id)
);

CREATE TABLE IF NOT EXISTS macro_fetch_logs (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    indicator_source_id UUID,
    indicator_code VARCHAR,
    source_code VARCHAR,
    status VARCHAR NOT NULL,
    records_inserted INTEGER DEFAULT 0,
    records_updated INTEGER DEFAULT 0,
    error_message TEXT,
    response_time_ms INTEGER,
    triggered_by VARCHAR DEFAULT 'scheduler',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT macro_fetch_logs_indicator_source_id_fkey FOREIGN KEY (indicator_source_id) REFERENCES macro_indicator_sources(id)
);

CREATE INDEX IF NOT EXISTS idx_macro_indicator_values_type ON macro_indicator_values(indicator_type_id);
CREATE INDEX IF NOT EXISTS idx_macro_indicator_values_date ON macro_indicator_values(effective_date);

-- =============================================================================
-- PART 14: Recreate field_deprecation_config (dropped in v2_012)
-- =============================================================================

CREATE TABLE IF NOT EXISTS field_deprecation_config (
    field_name TEXT NOT NULL PRIMARY KEY,
    category TEXT NOT NULL,
    deprecation_months INTEGER NOT NULL,
    description TEXT
);

-- =============================================================================
-- PART 15: Recreate company_data_history (dropped in v2_009)
-- =============================================================================

-- Need data_source enum if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'data_source') THEN
        CREATE TYPE data_source AS ENUM ('submission', 'enrichment', 'admin', 'api');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS company_data_history (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    company_id UUID NOT NULL,
    field_name TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT,
    source data_source NOT NULL,
    source_id UUID,
    changed_by UUID,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT company_data_history_company_id_fkey FOREIGN KEY (company_id) REFERENCES companies(id)
);

-- =============================================================================
-- PART 16: Recreate company_field_verifications (dropped in v2_006)
-- =============================================================================

CREATE TABLE IF NOT EXISTS company_field_verifications (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    company_id UUID NOT NULL,
    field_name TEXT NOT NULL,
    verified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    verified_by UUID,
    deprecation_category TEXT NOT NULL DEFAULT 'static',
    CONSTRAINT company_field_verifications_company_id_fkey FOREIGN KEY (company_id) REFERENCES companies(id)
);

COMMIT;

-- =============================================================================
-- VERIFICATION QUERIES (run manually)
-- =============================================================================
-- \dt                                    -- Verify tables exist
-- \d analyses                            -- Verify columns restored
-- \d enrichments                         -- Verify enrichments table
-- \d reports                             -- Verify reports table
-- SELECT COUNT(*) FROM enrichments;      -- Will be 0 (data not restored)
-- SELECT COUNT(*) FROM reports;          -- Will be 0 (data not restored)
