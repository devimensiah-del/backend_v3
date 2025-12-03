-- Migration 035: Migrate existing framework data to framework_results
-- Purpose: Consolidate 11 JSONB columns into single framework_results map
-- Note: Run this BEFORE dropping columns (migration 036)

-- Only update rows that don't already have framework_results populated
-- This is idempotent - safe to run multiple times

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

-- Verification queries (run manually to check results):
-- SELECT COUNT(*) FROM analyses WHERE framework_results IS NULL OR framework_results = '{}';
-- SELECT id, jsonb_object_keys(framework_results) FROM analyses LIMIT 5;
