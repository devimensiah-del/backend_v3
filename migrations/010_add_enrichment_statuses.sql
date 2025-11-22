-- ==================== MIGRATION 010: ADD ENRICHMENT STATUSES ====================
-- Purpose: Add 'finished' and 'approved' statuses to enrichments table
-- Date: 2025-11-22
-- Dependencies: 008_consolidate_enrichment_schema.sql
--
-- CONTEXT:
-- - Current enrichment statuses: 'pending', 'processing', 'completed', 'failed'
-- - Adding: 'finished' (enrichment complete, awaiting approval)
-- - Adding: 'approved' (admin has approved enrichment for analysis)
--
-- WORKFLOW:
-- pending → processing → finished → approved → (analysis starts)
--                     ↓
--                   failed
-- ==================================================================================

BEGIN;

-- ==================== 1. DROP EXISTING CONSTRAINT ====================
-- Remove the old status constraint so we can add new statuses
ALTER TABLE enrichments
DROP CONSTRAINT IF EXISTS valid_enrichment_status;

-- ==================== 2. ADD NEW STATUS CONSTRAINT ====================
-- Add new constraint with 'finished' and 'approved' statuses
--
-- Status Definitions:
-- - pending: Enrichment queued but not started
-- - processing: Enrichment in progress (API calls being made)
-- - finished: Enrichment completed successfully, awaiting admin review
-- - approved: Admin has approved enrichment, ready for analysis
-- - failed: Enrichment failed due to errors
ALTER TABLE enrichments
ADD CONSTRAINT valid_enrichment_status CHECK (status IN (
    'pending',
    'processing',
    'finished',    -- NEW: Enrichment complete, awaiting approval
    'approved',    -- NEW: Admin approved, ready for analysis
    'failed'
));

-- ==================== 3. UPDATE EXISTING DATA ====================
-- Migrate existing 'completed' status to 'finished' (if any exists)
-- This maintains the workflow - 'completed' records become 'finished' awaiting approval
UPDATE enrichments
SET status = 'finished'
WHERE status = 'completed';

-- ==================== 4. ADD APPROVAL TRACKING ====================
-- Add column to track when enrichment was approved
ALTER TABLE enrichments
ADD COLUMN IF NOT EXISTS approved_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS approved_by UUID; -- References user who approved (nullable)

-- Add index for approved enrichments
CREATE INDEX IF NOT EXISTS idx_enrichments_approved_at
ON enrichments(approved_at)
WHERE approved_at IS NOT NULL;

-- ==================== 5. ADD LOCKING MECHANISM ====================
-- Add is_locked column to prevent concurrent modifications
-- This is critical for workflow integrity
ALTER TABLE enrichments
ADD COLUMN IF NOT EXISTS is_locked BOOLEAN NOT NULL DEFAULT FALSE;

-- Add index for locked enrichments (for monitoring)
CREATE INDEX IF NOT EXISTS idx_enrichments_locked
ON enrichments(is_locked)
WHERE is_locked = TRUE;

-- ==================== 6. UPDATE CONSTRAINTS FOR DATA INTEGRITY ====================
-- Ensure approved_at is set when status is 'approved'
-- Note: This is implemented as a trigger instead of a CHECK constraint
-- because CHECK constraints can't reference other columns in time-based logic

CREATE OR REPLACE FUNCTION enforce_enrichment_approval_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    -- When status changes to 'approved', set approved_at if not already set
    IF NEW.status = 'approved' AND NEW.approved_at IS NULL THEN
        NEW.approved_at = NOW();
    END IF;

    -- When status changes away from 'approved', clear approved_at
    IF NEW.status != 'approved' AND OLD.status = 'approved' THEN
        NEW.approved_at = NULL;
        NEW.approved_by = NULL;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger
DROP TRIGGER IF EXISTS trigger_enrichment_approval_timestamp ON enrichments;
CREATE TRIGGER trigger_enrichment_approval_timestamp
    BEFORE UPDATE ON enrichments
    FOR EACH ROW
    EXECUTE FUNCTION enforce_enrichment_approval_timestamp();

-- ==================== 7. ADD DOCUMENTATION ====================
COMMENT ON CONSTRAINT valid_enrichment_status ON enrichments IS
'Valid enrichment statuses: pending → processing → finished → approved (or failed at any point)';

COMMENT ON COLUMN enrichments.approved_at IS
'Timestamp when enrichment was approved by admin for analysis processing';

COMMENT ON COLUMN enrichments.approved_by IS
'UUID of user who approved the enrichment (nullable, references users table)';

COMMENT ON COLUMN enrichments.is_locked IS
'Prevents concurrent modifications during workflow processing';

-- ==================== 8. VERIFY DATA INTEGRITY ====================
-- Check for any orphaned or inconsistent records
DO $$
DECLARE
    orphaned_count INTEGER;
    inconsistent_count INTEGER;
BEGIN
    -- Check for enrichments with 'approved' status but no approved_at
    SELECT COUNT(*) INTO inconsistent_count
    FROM enrichments
    WHERE status = 'approved' AND approved_at IS NULL;

    IF inconsistent_count > 0 THEN
        RAISE WARNING 'Found % enrichments with approved status but no approved_at timestamp', inconsistent_count;
        -- Fix them
        UPDATE enrichments
        SET approved_at = updated_at
        WHERE status = 'approved' AND approved_at IS NULL;
        RAISE NOTICE 'Fixed % inconsistent enrichment records', inconsistent_count;
    END IF;

    -- Check for enrichments without corresponding submissions
    SELECT COUNT(*) INTO orphaned_count
    FROM enrichments e
    LEFT JOIN submissions s ON e.submission_id = s.id
    WHERE s.id IS NULL;

    IF orphaned_count > 0 THEN
        RAISE WARNING 'Found % orphaned enrichments without corresponding submissions', orphaned_count;
    END IF;
END $$;

-- ==================== 9. PERFORMANCE OPTIMIZATION ====================
-- Update existing indices for better query performance
DROP INDEX IF EXISTS idx_enrichments_status;
CREATE INDEX idx_enrichments_status
ON enrichments(status, created_at DESC)
WHERE status IN ('pending', 'processing', 'finished');

-- Index for admin review queue (finished but not approved)
CREATE INDEX idx_enrichments_review_queue
ON enrichments(created_at DESC)
WHERE status = 'finished' AND approved_at IS NULL;

COMMIT;

-- ==================== ROLLBACK INSTRUCTIONS ====================
-- To rollback this migration:
-- BEGIN;
-- ALTER TABLE enrichments DROP CONSTRAINT IF EXISTS valid_enrichment_status;
-- ALTER TABLE enrichments ADD CONSTRAINT valid_enrichment_status
--   CHECK (status IN ('pending', 'processing', 'completed', 'failed'));
-- UPDATE enrichments SET status = 'completed' WHERE status IN ('finished', 'approved');
-- ALTER TABLE enrichments DROP COLUMN IF EXISTS approved_at;
-- ALTER TABLE enrichments DROP COLUMN IF EXISTS approved_by;
-- ALTER TABLE enrichments DROP COLUMN IF EXISTS is_locked;
-- DROP TRIGGER IF EXISTS trigger_enrichment_approval_timestamp ON enrichments;
-- DROP FUNCTION IF EXISTS enforce_enrichment_approval_timestamp();
-- COMMIT;
