-- Migration: v2_026_add_submitter_email
-- Description: Add submitter_email to company_submissions for duplicate prevention
-- Date: 2024-12-21

BEGIN;

-- Step 1: Add submitter_email column to company_submissions
ALTER TABLE company_submissions
ADD COLUMN IF NOT EXISTS submitter_email VARCHAR;

-- Step 2: Backfill existing records from submissions table
-- Normalize email to lowercase and trim whitespace
UPDATE company_submissions cs
SET submitter_email = LOWER(TRIM(s.contact_email))
FROM submissions s
WHERE s.id = cs.submission_id
  AND cs.submitter_email IS NULL;

-- Step 3: For any orphaned records, set a placeholder
UPDATE company_submissions
SET submitter_email = 'unknown@legacy.migration'
WHERE submitter_email IS NULL;

-- Step 4: Remove duplicate (company_id, submitter_email) pairs
-- Keep only the most recent submission link for each (company, email) pair
DELETE FROM company_submissions cs1
USING company_submissions cs2
WHERE cs1.company_id = cs2.company_id
  AND cs1.submitter_email = cs2.submitter_email
  AND cs1.linked_at < cs2.linked_at;

-- Step 5: Handle any remaining duplicates with same linked_at (keep arbitrary one)
DELETE FROM company_submissions cs1
USING company_submissions cs2
WHERE cs1.company_id = cs2.company_id
  AND cs1.submitter_email = cs2.submitter_email
  AND cs1.submission_id < cs2.submission_id
  AND cs1.linked_at = cs2.linked_at;

-- Step 6: Make submitter_email NOT NULL after backfill
ALTER TABLE company_submissions
ALTER COLUMN submitter_email SET NOT NULL;

-- Step 7: Create unique index on (company_id, submitter_email)
-- This prevents the same user from submitting for the same company twice
CREATE UNIQUE INDEX IF NOT EXISTS idx_company_submissions_company_email_unique
ON company_submissions (company_id, submitter_email);

COMMIT;
