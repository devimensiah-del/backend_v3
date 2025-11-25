-- Migration: Add is_visible_to_user column to analyses table
-- This controls whether the analysis (and PDF) is visible to the end user
-- By default, even after approval, analyses are NOT visible to users
-- Admin must explicitly toggle visibility after review

-- Add the visibility column
ALTER TABLE analyses
ADD COLUMN IF NOT EXISTS is_visible_to_user BOOLEAN NOT NULL DEFAULT FALSE;

-- Add comment for documentation
COMMENT ON COLUMN analyses.is_visible_to_user IS 'Controls user visibility. Even approved analyses with PDFs are hidden until admin toggles this on.';

-- Create index for filtering visible analyses
CREATE INDEX IF NOT EXISTS idx_analyses_visibility ON analyses(is_visible_to_user) WHERE deleted_at IS NULL;
