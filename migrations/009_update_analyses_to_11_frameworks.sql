-- Migration: Update analyses table to support 11 strategic frameworks
-- Migrates from old 7-framework schema to new 11-framework schema

BEGIN;

-- Add new framework columns
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS tam_sam_som JSONB DEFAULT '{}'::jsonb;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS benchmarking JSONB DEFAULT '{}'::jsonb;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS blue_ocean JSONB DEFAULT '{}'::jsonb;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS growth_hacking JSONB DEFAULT '{}'::jsonb;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS scenarios JSONB DEFAULT '{}'::jsonb;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS bsc JSONB DEFAULT '{}'::jsonb;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS decision_matrix JSONB DEFAULT '{}'::jsonb;

-- Keep old columns for backward compatibility (don't drop them)
-- This allows gradual migration if needed
-- If you want to drop them eventually, uncomment:
-- ALTER TABLE analyses DROP COLUMN IF EXISTS bcg_matrix;
-- ALTER TABLE analyses DROP COLUMN IF EXISTS value_proposition;
-- ALTER TABLE analyses DROP COLUMN IF EXISTS business_model;

COMMIT;
