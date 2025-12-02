-- Migration: 031_analysis_is_public.sql
-- Description: Add is_public column to analyses table for 4-state visibility matrix
-- Date: 2025-01-28

-- Add is_public column to analyses table
-- When true: report is accessible via access code without login
-- When false: report requires authentication even with access code
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT false;

-- Comment for clarity
COMMENT ON COLUMN analyses.is_public IS 'When true, report is accessible via access code without login. When false, requires authentication.';

-- Index for queries filtering by public reports
CREATE INDEX IF NOT EXISTS idx_analyses_is_public ON analyses(is_public) WHERE is_public = true;
