-- Migration 036: Drop legacy framework columns
-- Purpose: Remove 11+1 individual JSONB columns now that data is in framework_results
-- WARNING: This is destructive - ensure migration 035 has been run and verified first!
-- ROLLBACK: Not possible - data would need to be restored from framework_results

-- Drop legacy framework columns (data now in framework_results)
ALTER TABLE analyses DROP COLUMN IF EXISTS pestel;
ALTER TABLE analyses DROP COLUMN IF EXISTS porter;
ALTER TABLE analyses DROP COLUMN IF EXISTS swot;
ALTER TABLE analyses DROP COLUMN IF EXISTS tam_sam_som;
ALTER TABLE analyses DROP COLUMN IF EXISTS benchmarking;
ALTER TABLE analyses DROP COLUMN IF EXISTS blue_ocean;
ALTER TABLE analyses DROP COLUMN IF EXISTS growth_hacking;
ALTER TABLE analyses DROP COLUMN IF EXISTS scenarios;
ALTER TABLE analyses DROP COLUMN IF EXISTS okrs;
ALTER TABLE analyses DROP COLUMN IF EXISTS bsc;
ALTER TABLE analyses DROP COLUMN IF EXISTS decision_matrix;
ALTER TABLE analyses DROP COLUMN IF EXISTS synthesis;

-- Add comment documenting the change
COMMENT ON TABLE analyses IS 'Analysis results - framework data stored in framework_results JSONB column';
