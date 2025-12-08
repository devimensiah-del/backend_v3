-- Migration: v2_013_drop_legacy_analysis_columns.sql
-- Description: Clean up legacy columns from analyses table and drop analysis_versions table
--
-- This migration removes fields that are no longer needed in the v2 architecture:
-- - submission_id: Analyses now link via company_id + challenge_id (not submission)
-- - version_notes: Not used in current workflow
-- - processing_time_ms: Tracked at step level in analysis_steps instead
-- - access_code_created_at: Not tracked (only access_code needed)
-- - pdf_url: PDF generation removed from v2
-- - pdf_generated_at: PDF generation removed from v2
--
-- Also drops:
-- - analysis_versions table: Version audit trail not implemented in v2

BEGIN;

-- =============================================================================
-- PART 1: Drop columns from analyses table
-- =============================================================================

-- Drop submission_id (analyses now link via company_id + challenge_id)
ALTER TABLE analyses DROP COLUMN IF EXISTS submission_id;

-- Drop version tracking notes (not used)
ALTER TABLE analyses DROP COLUMN IF EXISTS version_notes;

-- Drop processing time (tracked per-step in analysis_steps instead)
ALTER TABLE analyses DROP COLUMN IF EXISTS processing_time_ms;

-- Drop access code timestamp (not needed, only access_code matters)
ALTER TABLE analyses DROP COLUMN IF EXISTS access_code_created_at;

-- Drop PDF-related columns (PDF generation removed from v2)
ALTER TABLE analyses DROP COLUMN IF EXISTS pdf_url;
ALTER TABLE analyses DROP COLUMN IF EXISTS pdf_generated_at;

-- =============================================================================
-- PART 2: Drop analysis_versions table
-- =============================================================================

-- Drop indexes first
DROP INDEX IF EXISTS idx_analysis_versions_analysis;
DROP INDEX IF EXISTS idx_analysis_versions_version;

-- Drop the table
DROP TABLE IF EXISTS analysis_versions CASCADE;

-- =============================================================================
-- PART 3: Add comment documenting the cleanup
-- =============================================================================

COMMENT ON TABLE analyses IS 'Strategic analysis results linking company + challenge to framework outputs. V2: No PDF generation, no version history, timing tracked in analysis_steps.';

COMMIT;

-- =============================================================================
-- ROLLBACK INSTRUCTIONS (manual if needed)
-- =============================================================================
--
-- To restore these columns:
--
-- ALTER TABLE analyses ADD COLUMN submission_id UUID REFERENCES submissions(id);
-- ALTER TABLE analyses ADD COLUMN version_notes TEXT;
-- ALTER TABLE analyses ADD COLUMN processing_time_ms BIGINT;
-- ALTER TABLE analyses ADD COLUMN access_code_created_at TIMESTAMPTZ;
-- ALTER TABLE analyses ADD COLUMN pdf_url TEXT;
-- ALTER TABLE analyses ADD COLUMN pdf_generated_at TIMESTAMPTZ;
--
-- To restore analysis_versions table, see v2_011_analysis_versioning.sql
