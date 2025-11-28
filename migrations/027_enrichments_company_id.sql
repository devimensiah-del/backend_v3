-- Migration 027: Link enrichments directly to companies
-- Purpose: Enable company-based re-enrichment without creating new submissions

-- Step 1: Add company_id column (nullable initially for backfill)
ALTER TABLE enrichments ADD COLUMN IF NOT EXISTS company_id UUID;

-- Step 2: Backfill company_id from existing submission -> company_submissions linkage
UPDATE enrichments e
SET company_id = cs.company_id
FROM company_submissions cs
WHERE cs.submission_id = e.submission_id
  AND cs.is_primary = true
  AND e.company_id IS NULL;

-- Step 3: Add foreign key constraint
ALTER TABLE enrichments
    ADD CONSTRAINT fk_enrichments_company
    FOREIGN KEY (company_id) REFERENCES companies(id);

-- Step 4: Create indexes for company-based queries
CREATE INDEX IF NOT EXISTS idx_enrichments_company ON enrichments(company_id);
CREATE INDEX IF NOT EXISTS idx_enrichments_company_created ON enrichments(company_id, created_at DESC);

COMMENT ON COLUMN enrichments.company_id IS 'Direct link to company for re-enrichment support';
