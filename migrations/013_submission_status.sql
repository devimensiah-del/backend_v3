-- ============================================================================
-- Migration 013: Simplify submission status to ONLY 'received'
-- ============================================================================
-- All workflow state is tracked in Enrichment and Analysis entities
-- Submission status NEVER changes after creation - always 'received'
-- This aligns with the domain model: backend_v3/domain/submission/model.go:58
-- ============================================================================

-- Step 1: Drop old constraint from migration 007
ALTER TABLE submissions
DROP CONSTRAINT IF EXISTS valid_status;

-- Step 2: Add new constraint: ONLY 'received' allowed
ALTER TABLE submissions
ADD CONSTRAINT valid_status CHECK (status = 'received');

-- Step 3: Update default to 'received'
ALTER TABLE submissions
ALTER COLUMN status SET DEFAULT 'received';

-- Step 4: Migrate existing data (one-time cleanup)
UPDATE submissions
SET status = 'received'
WHERE status != 'received' AND deleted_at IS NULL;

-- Step 5: Documentation
COMMENT ON COLUMN submissions.status IS 'Always "received" - all workflow state tracked in enrichments and analyses tables';
COMMENT ON CONSTRAINT valid_status ON submissions IS 'Submission status is always "received" - no other values allowed';
