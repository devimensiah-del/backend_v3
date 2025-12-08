-- =============================================================================
-- BASELINE SCHEMA - Production State as of 2024-12-04
-- =============================================================================
-- This represents the production database schema after migrations 001-031.
-- DO NOT RUN THIS ON PRODUCTION - it's for reference and fresh setups only.
-- For new environments, run this first, then apply v2_* migrations.
-- =============================================================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- CORE TABLES
-- =============================================================================

-- User Profiles (linked to Supabase auth.users)
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
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_profiles_id_fkey FOREIGN KEY (id) REFERENCES auth.users(id)
);

-- Submissions (entry point for analysis requests)
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
    business_challenge TEXT NOT NULL,
    additional_notes TEXT,
    linkedin_url VARCHAR,
    twitter_handle VARCHAR,
    status VARCHAR NOT NULL DEFAULT 'received' CHECK (status = 'received'),
    user_id UUID,
    cnpj VARCHAR,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_submissions_user FOREIGN KEY (user_id) REFERENCES user_profiles(id)
);

-- Companies (verified company records)
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
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Company-Submission linking table
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

-- Company data change history
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

-- Company field verifications
CREATE TABLE IF NOT EXISTS company_field_verifications (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    company_id UUID NOT NULL,
    field_name TEXT NOT NULL,
    verified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    verified_by UUID,
    deprecation_category TEXT NOT NULL DEFAULT 'static',
    CONSTRAINT company_field_verifications_company_id_fkey FOREIGN KEY (company_id) REFERENCES companies(id)
);

-- Field deprecation configuration
CREATE TABLE IF NOT EXISTS field_deprecation_config (
    field_name TEXT NOT NULL PRIMARY KEY,
    category TEXT NOT NULL,
    deprecation_months INTEGER NOT NULL,
    description TEXT
);

-- =============================================================================
-- WORKFLOW TABLES
-- =============================================================================

-- Enrichments (AI data gathering layer)
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
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT enrichments_submission_id_fkey FOREIGN KEY (submission_id) REFERENCES submissions(id),
    CONSTRAINT fk_enrichments_company FOREIGN KEY (company_id) REFERENCES companies(id)
);

-- Analyses (11 strategic frameworks)
CREATE TABLE IF NOT EXISTS analyses (
    id UUID NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
    submission_id UUID NOT NULL,
    enrichment_id UUID NOT NULL,
    company_id UUID,
    -- Framework outputs (JSONB columns)
    swot JSONB,
    pestel JSONB,
    porter JSONB,
    okrs JSONB,
    synthesis JSONB,
    tam_sam_som JSONB,
    benchmarking JSONB,
    blue_ocean JSONB,
    growth_hacking JSONB,
    scenarios JSONB,
    bsc JSONB,
    decision_matrix JSONB,
    -- Status and workflow
    status VARCHAR NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'approved', 'sent', 'failed')),
    error_message TEXT,
    processing_time_ms BIGINT,
    -- Visibility and access
    is_visible_to_user BOOLEAN NOT NULL DEFAULT false,
    is_blurred BOOLEAN NOT NULL DEFAULT true,
    is_public BOOLEAN NOT NULL DEFAULT false,
    access_code VARCHAR,
    access_code_created_at TIMESTAMPTZ,
    -- PDF generation
    pdf_url TEXT,
    pdf_generated_at TIMESTAMPTZ,
    -- Approval workflow
    approved_at TIMESTAMPTZ,
    approved_by UUID,
    sent_at TIMESTAMPTZ,
    sent_to VARCHAR,
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT analyses_submission_id_fkey FOREIGN KEY (submission_id) REFERENCES submissions(id),
    CONSTRAINT analyses_enrichment_id_fkey FOREIGN KEY (enrichment_id) REFERENCES enrichments(id),
    CONSTRAINT fk_analyses_company FOREIGN KEY (company_id) REFERENCES companies(id)
);

-- Reports (PDF generation tracking)
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

-- =============================================================================
-- MACROECONOMICS TABLES
-- =============================================================================

-- Macro data sources (BCB, IBGE, etc.)
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

-- Macro indicator types (SELIC, IPCA, USD/BRL)
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

-- Macro indicator sources (linking types to sources)
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

-- Macro indicator values (actual data points)
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

-- Macro fetch logs (API call tracking)
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

-- =============================================================================
-- LOGGING TABLES
-- =============================================================================

-- LLM usage logs (cost tracking)
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

-- =============================================================================
-- INDEXES
-- =============================================================================

-- Submissions
CREATE INDEX IF NOT EXISTS idx_submissions_user_id ON submissions(user_id);
CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions(status);
CREATE INDEX IF NOT EXISTS idx_submissions_created_at ON submissions(created_at);

-- Enrichments
CREATE INDEX IF NOT EXISTS idx_enrichments_submission_id ON enrichments(submission_id);
CREATE INDEX IF NOT EXISTS idx_enrichments_status ON enrichments(status);
CREATE INDEX IF NOT EXISTS idx_enrichments_company_id ON enrichments(company_id);

-- Analyses
CREATE INDEX IF NOT EXISTS idx_analyses_submission_id ON analyses(submission_id);
CREATE INDEX IF NOT EXISTS idx_analyses_enrichment_id ON analyses(enrichment_id);
CREATE INDEX IF NOT EXISTS idx_analyses_status ON analyses(status);
CREATE INDEX IF NOT EXISTS idx_analyses_company_id ON analyses(company_id);
CREATE INDEX IF NOT EXISTS idx_analyses_access_code ON analyses(access_code);

-- Reports
CREATE INDEX IF NOT EXISTS idx_reports_submission_id ON reports(submission_id);
CREATE INDEX IF NOT EXISTS idx_reports_analysis_id ON reports(analysis_id);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status);

-- Companies
CREATE INDEX IF NOT EXISTS idx_companies_cnpj ON companies(cnpj);
CREATE INDEX IF NOT EXISTS idx_companies_owner_id ON companies(owner_id);

-- Macro indicators
CREATE INDEX IF NOT EXISTS idx_macro_indicator_values_type ON macro_indicator_values(indicator_type_id);
CREATE INDEX IF NOT EXISTS idx_macro_indicator_values_date ON macro_indicator_values(effective_date);

-- LLM usage
CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_submission ON llm_usage_logs(submission_id);
CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_created ON llm_usage_logs(created_at);
