-- Migration: v2_033_analyses_framework_result_ids.sql
-- Description: Add framework_result_ids column to analyses table for V2 mode
-- Keeps framework_results JSONB for backward compatibility (V1 mode)

-- Add column to reference company_framework_results (V2 mode)
-- Empty array in V1 mode (uses framework_results JSONB instead)
ALTER TABLE analyses
ADD COLUMN IF NOT EXISTS framework_result_ids UUID[] DEFAULT '{}';

-- Index for lookups (GIN index for array contains queries)
CREATE INDEX IF NOT EXISTS idx_analyses_framework_result_ids ON analyses USING GIN(framework_result_ids);

COMMENT ON COLUMN analyses.framework_result_ids IS 'V2 mode: Array of company_framework_results.id references. Empty in V1 mode (uses framework_results JSONB instead).';
