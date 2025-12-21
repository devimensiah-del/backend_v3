-- Migration: v2_028_merge_cnpj_duplicates
-- Description: One-time script to merge companies with duplicate CNPJs
-- Strategy: Keep the MOST RECENT company, move submissions/challenges from older ones
-- Date: 2024-12-21

-- This migration should be run ONCE to clean up existing duplicates
-- After this, the application allows different users to have same CNPJ (no unique constraint)

BEGIN;

-- Step 1: Create a temp table to identify duplicates and which company to keep
-- For each CNPJ, we keep the most recently created company
-- NOTE: Use company_core (the actual table), not companies (the view)
CREATE TEMP TABLE cnpj_merge_plan AS
WITH ranked_companies AS (
    SELECT
        id,
        name,
        cnpj_normalized,
        created_at,
        ROW_NUMBER() OVER (
            PARTITION BY cnpj_normalized
            ORDER BY created_at DESC  -- Most recent first
        ) as rn,
        COUNT(*) OVER (PARTITION BY cnpj_normalized) as duplicate_count
    FROM company_core
    WHERE cnpj_normalized IS NOT NULL
      AND cnpj_normalized != ''
      AND deleted_at IS NULL
)
SELECT
    id as company_id,
    name as company_name,
    cnpj_normalized,
    created_at,
    rn,
    duplicate_count,
    CASE WHEN rn = 1 THEN true ELSE false END as keep_this,
    FIRST_VALUE(id) OVER (
        PARTITION BY cnpj_normalized
        ORDER BY created_at DESC
    ) as target_company_id
FROM ranked_companies
WHERE duplicate_count > 1;

-- Log what we're about to do
DO $$
DECLARE
    dup_count INTEGER;
    companies_to_merge INTEGER;
BEGIN
    SELECT COUNT(DISTINCT cnpj_normalized) INTO dup_count FROM cnpj_merge_plan;
    SELECT COUNT(*) INTO companies_to_merge FROM cnpj_merge_plan WHERE keep_this = false;

    RAISE NOTICE 'Found % CNPJs with duplicates', dup_count;
    RAISE NOTICE 'Will merge % older companies into their newer counterparts', companies_to_merge;
END $$;

-- Step 2: Handle company_submissions
-- Delete ALL submission links from old companies being merged
-- The target company (newest) already has its own submission links
-- Trying to move them causes conflicts when multiple old companies share emails
DELETE FROM company_submissions cs
USING cnpj_merge_plan mp
WHERE cs.company_id = mp.company_id
  AND mp.keep_this = false;

-- Step 3: Move challenges from old companies to new companies
UPDATE challenges ch
SET company_id = mp.target_company_id
FROM cnpj_merge_plan mp
WHERE ch.company_id = mp.company_id
  AND mp.keep_this = false;

-- Step 4: Move analyses from old companies to new companies
UPDATE analyses a
SET company_id = mp.target_company_id
FROM cnpj_merge_plan mp
WHERE a.company_id = mp.company_id
  AND mp.keep_this = false;

-- Step 5: Soft-delete the old companies (update company_core, not the view)
UPDATE company_core c
SET deleted_at = NOW()
FROM cnpj_merge_plan mp
WHERE c.id = mp.company_id
  AND mp.keep_this = false;

-- Step 6: Log the results
DO $$
DECLARE
    merged_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO merged_count
    FROM cnpj_merge_plan
    WHERE keep_this = false;

    RAISE NOTICE 'Successfully merged % duplicate companies', merged_count;
END $$;

-- Clean up
DROP TABLE cnpj_merge_plan;

COMMIT;

-- Verification query (run after migration to confirm)
-- SELECT cnpj_normalized, COUNT(*) as count
-- FROM companies
-- WHERE cnpj_normalized IS NOT NULL
--   AND cnpj_normalized != ''
--   AND deleted_at IS NULL
-- GROUP BY cnpj_normalized
-- HAVING COUNT(*) > 1;
