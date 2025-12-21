-- Fix: Complete v2_026 migration with duplicate cleanup
-- Run this instead of v2_026_add_submitter_email.sql

BEGIN;

-- Step 1: Add submitter_email column (if not exists)
ALTER TABLE company_submissions
ADD COLUMN IF NOT EXISTS submitter_email VARCHAR;

-- Step 2: Backfill from submissions table
UPDATE company_submissions cs
SET submitter_email = LOWER(TRIM(s.contact_email))
FROM submissions s
WHERE s.id = cs.submission_id
  AND cs.submitter_email IS NULL;

-- Step 3: Set placeholder for orphaned records
UPDATE company_submissions
SET submitter_email = 'unknown@legacy.migration'
WHERE submitter_email IS NULL;

-- Step 4: Create temp table with rows to KEEP (one per company+email)
CREATE TEMP TABLE keep_submissions AS
SELECT DISTINCT ON (company_id, submitter_email)
    company_id, submission_id
FROM company_submissions
ORDER BY company_id, submitter_email, linked_at DESC, submission_id DESC;

-- Step 5: Delete everything NOT in the keep list
DELETE FROM company_submissions
WHERE (company_id, submission_id) NOT IN (
    SELECT company_id, submission_id FROM keep_submissions
);

-- Step 6: Drop temp table
DROP TABLE keep_submissions;

-- Step 7: Verify no duplicates remain
DO $$
DECLARE
    dup_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO dup_count
    FROM (
        SELECT company_id, submitter_email
        FROM company_submissions
        GROUP BY company_id, submitter_email
        HAVING COUNT(*) > 1
    ) dups;

    IF dup_count > 0 THEN
        RAISE EXCEPTION 'Still have % duplicate pairs', dup_count;
    END IF;
END $$;

-- Step 8: Make NOT NULL
ALTER TABLE company_submissions
ALTER COLUMN submitter_email SET NOT NULL;

-- Step 9: Create unique index
CREATE UNIQUE INDEX IF NOT EXISTS idx_company_submissions_company_email_unique
ON company_submissions (company_id, submitter_email);

COMMIT;
