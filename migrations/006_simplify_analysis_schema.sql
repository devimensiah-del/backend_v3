-- Simplify analyses schema to enforce one-to-one relationship with submissions
-- Drops legacy versioning columns and adds a unique constraint on submission_id

-- Remove self-referencing FK before dropping the column
ALTER TABLE analyses DROP CONSTRAINT IF EXISTS fk_parent_analysis;

-- Drop versioning columns
ALTER TABLE analyses
  DROP COLUMN IF EXISTS version,
  DROP COLUMN IF EXISTS parent_analysis_id,
  DROP COLUMN IF EXISTS is_latest;

-- Enforce one-to-one relationship between submission and analysis
ALTER TABLE analyses
  ADD CONSTRAINT analyses_submission_id_unique UNIQUE (submission_id);
