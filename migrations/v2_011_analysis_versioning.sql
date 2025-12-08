-- =============================================================================
-- Migration: v2_013_analysis_versioning
-- Purpose: Add version tracking to analyses for audit trail and rollback
-- Date: 2024-12-05
-- =============================================================================
--
-- CONTEXT:
-- When frameworks are refined via the wizard, we need to track all versions
-- of the analysis for:
-- - Audit trail (who changed what, when)
-- - Rollback capability (revert to previous version)
-- - Comparison (see changes between versions)
--
-- DESIGN:
-- - analyses.version: Current version number (increments on each edit)
-- - analysis_versions: Immutable history of all previous versions
-- - Each refinement creates a new version before applying changes
--
-- =============================================================================

BEGIN;

-- =============================================================================
-- PART 1: Add version tracking columns to analyses
-- =============================================================================

-- Current version number, starts at 1
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS version INTEGER DEFAULT 1;

-- Notes about what changed in this version
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS version_notes TEXT;

-- Comments for documentation
COMMENT ON COLUMN analyses.version IS 'Current version number, increments on each framework edit/refinement';
COMMENT ON COLUMN analyses.version_notes IS 'Summary of changes in current version';

-- =============================================================================
-- PART 2: Create analysis_versions history table
-- =============================================================================

CREATE TABLE IF NOT EXISTS analysis_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Link to parent analysis
    analysis_id UUID NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,

    -- Version number (matches analyses.version at time of snapshot)
    version INTEGER NOT NULL,

    -- Snapshot of framework_results at this version
    framework_results JSONB NOT NULL,

    -- Who created this version (NULL for system/automated)
    created_by UUID,

    -- When this version was created
    created_at TIMESTAMPTZ DEFAULT NOW(),

    -- Summary of changes that led to this version
    change_summary TEXT,

    -- Ensure no duplicate versions per analysis
    UNIQUE(analysis_id, version)
);

-- Index for fast lookups by analysis
CREATE INDEX IF NOT EXISTS idx_analysis_versions_analysis ON analysis_versions(analysis_id);

-- Index for ordering by version
CREATE INDEX IF NOT EXISTS idx_analysis_versions_version ON analysis_versions(analysis_id, version DESC);

-- Comments for documentation
COMMENT ON TABLE analysis_versions IS 'Immutable history of all analysis versions for audit trail and rollback';
COMMENT ON COLUMN analysis_versions.analysis_id IS 'The analysis this version belongs to';
COMMENT ON COLUMN analysis_versions.version IS 'Version number at time of snapshot';
COMMENT ON COLUMN analysis_versions.framework_results IS 'Full snapshot of framework_results JSONB at this version';
COMMENT ON COLUMN analysis_versions.created_by IS 'User who triggered the version (NULL for automated)';
COMMENT ON COLUMN analysis_versions.change_summary IS 'Human-readable summary of what changed';

-- =============================================================================
-- PART 3: Add step-level history tracking
-- =============================================================================

-- Track previous outputs in analysis_steps for wizard refinement history
ALTER TABLE analysis_steps ADD COLUMN IF NOT EXISTS previous_outputs JSONB DEFAULT '[]';

COMMENT ON COLUMN analysis_steps.previous_outputs IS 'Array of previous outputs from refinements, preserving step-level audit trail';

COMMIT;

-- =============================================================================
-- DOWN MIGRATION (Rollback)
-- =============================================================================
-- To rollback:
-- BEGIN;
-- ALTER TABLE analysis_steps DROP COLUMN IF EXISTS previous_outputs;
-- DROP INDEX IF EXISTS idx_analysis_versions_version;
-- DROP INDEX IF EXISTS idx_analysis_versions_analysis;
-- DROP TABLE IF EXISTS analysis_versions CASCADE;
-- ALTER TABLE analyses DROP COLUMN IF EXISTS version_notes;
-- ALTER TABLE analyses DROP COLUMN IF EXISTS version;
-- COMMIT;
