-- Migration 034: Add generic framework_results column to analyses
-- Purpose: Enable dynamic framework storage instead of 11 hardcoded columns

-- Add generic framework results column
ALTER TABLE analyses
ADD COLUMN IF NOT EXISTS framework_results JSONB DEFAULT '{}';

-- Index for querying specific frameworks (GIN index for JSONB)
CREATE INDEX IF NOT EXISTS idx_analyses_framework_results
ON analyses USING gin(framework_results);

-- Comment explaining the structure
COMMENT ON COLUMN analyses.framework_results IS
'Generic storage for all framework results. Structure: {"pestel": {...}, "swot": {...}, ...}';
