-- Migration: v2_032_company_framework_results.sql
-- Description: Create company_framework_results table for per-company, per-framework result storage
-- This enables result caching, reuse across analyses, and granular retry

CREATE TABLE IF NOT EXISTS company_framework_results (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,

    -- Relationships
    company_id UUID NOT NULL,                 -- The company this result belongs to
    framework_id UUID NOT NULL,               -- The framework that produced this result
    challenge_id UUID,                        -- Optional: specific challenge context (NULL = company-wide base frameworks)

    -- Result data
    result JSONB,                             -- The framework output (NULL while pending/processing)

    -- Status tracking (same pattern as enrichment steps)
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending, processing, completed, failed
    error_message TEXT,                       -- Error details if failed

    -- Versioning (for regeneration tracking)
    version INTEGER NOT NULL DEFAULT 1,
    is_current BOOLEAN NOT NULL DEFAULT true, -- Only one version is current per company+framework+challenge

    -- Context used to generate this result (for cache invalidation)
    context_hash VARCHAR(64),                 -- SHA256 of input context (company data + dependencies)

    -- Metadata
    generated_at TIMESTAMPTZ,                 -- When the LLM generated this result
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT fk_company_framework_results_company
        FOREIGN KEY (company_id) REFERENCES company_core(id) ON DELETE CASCADE,
    CONSTRAINT fk_company_framework_results_framework
        FOREIGN KEY (framework_id) REFERENCES frameworks(id) ON DELETE RESTRICT,
    CONSTRAINT fk_company_framework_results_challenge
        FOREIGN KEY (challenge_id) REFERENCES challenges(id) ON DELETE CASCADE,
    CONSTRAINT chk_status_valid
        CHECK (status IN ('pending', 'processing', 'completed', 'failed'))
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_cfr_company_id ON company_framework_results(company_id);
CREATE INDEX IF NOT EXISTS idx_cfr_framework_id ON company_framework_results(framework_id);
CREATE INDEX IF NOT EXISTS idx_cfr_challenge_id ON company_framework_results(challenge_id) WHERE challenge_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cfr_status ON company_framework_results(status) WHERE status IN ('pending', 'processing');
CREATE INDEX IF NOT EXISTS idx_cfr_current ON company_framework_results(company_id, framework_id, is_current) WHERE is_current = true;

-- Unique constraint: only one current version per company+framework+challenge combo
-- Uses COALESCE to handle NULL challenge_id (base frameworks)
CREATE UNIQUE INDEX IF NOT EXISTS uq_cfr_current
ON company_framework_results(company_id, framework_id, COALESCE(challenge_id, '00000000-0000-0000-0000-000000000000'))
WHERE is_current = true;

COMMENT ON TABLE company_framework_results IS 'Individual framework results per company. Enables result caching, reuse across analyses, and granular retry.';
COMMENT ON COLUMN company_framework_results.status IS 'pending = waiting to run, processing = LLM call in progress, completed = result available, failed = error occurred';
COMMENT ON COLUMN company_framework_results.context_hash IS 'SHA256 hash of input context for cache invalidation. Regenerate if company data or dependencies changed.';
COMMENT ON COLUMN company_framework_results.challenge_id IS 'NULL for base frameworks (PESTEL, Porter, SWOT). Set for challenge-specific frameworks.';
