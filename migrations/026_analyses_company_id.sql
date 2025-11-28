-- Migration 026: Link analyses directly to companies
-- Purpose: Enable multiple analyses per company (history) and decouple from submissions

-- Step 1: Add company_id column (nullable initially for backfill)
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS company_id UUID;

-- Step 2: Backfill company_id from existing submission -> company_submissions linkage
UPDATE analyses a
SET company_id = cs.company_id
FROM company_submissions cs
WHERE cs.submission_id = a.submission_id::uuid
  AND cs.is_primary = true
  AND a.company_id IS NULL;

-- Step 3: Add foreign key constraint
ALTER TABLE analyses
    ADD CONSTRAINT fk_analyses_company
    FOREIGN KEY (company_id) REFERENCES companies(id);

-- Step 4: Create indexes for company-based queries
CREATE INDEX IF NOT EXISTS idx_analyses_company ON analyses(company_id);
CREATE INDEX IF NOT EXISTS idx_analyses_company_created ON analyses(company_id, created_at DESC);

-- Step 5: DROP unique constraint on submission_id to allow multiple analyses per company
-- This enables analysis history (one company can have many analyses over time)
ALTER TABLE analyses DROP CONSTRAINT IF EXISTS analyses_submission_id_unique;
ALTER TABLE analyses DROP CONSTRAINT IF EXISTS analyses_submission_id_key;

COMMENT ON COLUMN analyses.company_id IS 'Direct link to company for analysis history support';
