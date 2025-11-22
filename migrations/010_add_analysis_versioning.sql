-- Migration: Add versioning support to analyses table
-- This enables tracking multiple versions of analyses for a submission

BEGIN;

-- Add version and parent_analysis_id columns
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS parent_analysis_id UUID;

-- Add foreign key constraint for parent_analysis_id
ALTER TABLE analyses
    ADD CONSTRAINT fk_parent_analysis
    FOREIGN KEY (parent_analysis_id)
    REFERENCES analyses(id)
    ON DELETE SET NULL;

-- Update existing status constraint to include new statuses
ALTER TABLE analyses DROP CONSTRAINT IF EXISTS valid_analysis_status;
ALTER TABLE analyses
    ADD CONSTRAINT valid_analysis_status
    CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'approved', 'sent'));

-- Create index for version queries
CREATE INDEX IF NOT EXISTS idx_analyses_version ON analyses(submission_id, version DESC);
CREATE INDEX IF NOT EXISTS idx_analyses_parent_id ON analyses(parent_analysis_id);

-- Add comment explaining versioning
COMMENT ON COLUMN analyses.version IS 'Version number for this analysis, increments for each new version';
COMMENT ON COLUMN analyses.parent_analysis_id IS 'Reference to the parent analysis this version was created from';

COMMIT;
