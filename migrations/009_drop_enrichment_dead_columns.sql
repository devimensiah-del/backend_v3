-- Migration 009: Drop unused columns from enrichments table
-- Purpose: Remove dead code - these columns were never written to
-- Date: 2025-11-24
--
-- These fields existed in the model but were never populated:
-- - enrichment_score: Never calculated or saved
-- - processing_time_ms: Never tracked (unlike analysis domain which uses it)
-- - approved_at: Never set on approval
-- - approved_by: Never set on approval
--
-- Note: The analysis domain DOES use these fields, so we only drop from enrichments.

BEGIN;

ALTER TABLE enrichments
    DROP COLUMN IF EXISTS enrichment_score,
    DROP COLUMN IF EXISTS processing_time_ms,
    DROP COLUMN IF EXISTS approved_at,
    DROP COLUMN IF EXISTS approved_by;

COMMIT;
