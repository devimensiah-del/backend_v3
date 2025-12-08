-- =============================================================================
-- Migration: v2_002_framework_results
-- Purpose: Add generic framework_results column and migrate existing data
-- Date: 2024-12-02
-- Replaces: 034_add_framework_results.sql, 035_migrate_framework_results.sql
-- =============================================================================

-- Add generic framework results column
ALTER TABLE analyses
ADD COLUMN IF NOT EXISTS framework_results JSONB DEFAULT '{}';

-- Index for querying specific frameworks (GIN index for JSONB)
CREATE INDEX IF NOT EXISTS idx_analyses_framework_results
ON analyses USING gin(framework_results);

COMMENT ON COLUMN analyses.framework_results IS
'Generic storage for all framework results. Structure: {"pestel": {...}, "swot": {...}, ...}';

-- =============================================================================
-- DATA MIGRATION: Move existing framework data to framework_results
-- Only update rows that don't already have framework_results populated
-- This is idempotent - safe to run multiple times
-- =============================================================================

UPDATE analyses
SET framework_results = jsonb_strip_nulls(jsonb_build_object(
    'pestel', pestel,
    'porter', porter,
    'swot', swot,
    'tam_sam_som', tam_sam_som,
    'benchmarking', benchmarking,
    'blue_ocean', blue_ocean,
    'growth_hacking', growth_hacking,
    'scenarios', scenarios,
    'okrs', okrs,
    'bsc', bsc,
    'decision_matrix', decision_matrix,
    'synthesis', synthesis
))
WHERE framework_results IS NULL
   OR framework_results = '{}'::jsonb
   OR jsonb_typeof(framework_results) = 'null';

-- Verification query (run manually):
-- SELECT COUNT(*) FROM analyses WHERE framework_results IS NULL OR framework_results = '{}';
