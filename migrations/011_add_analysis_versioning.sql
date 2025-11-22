-- ==================== MIGRATION 011: ADD ANALYSIS VERSIONING ====================
-- Purpose: Add versioning and additional statuses to analyses table
-- Date: 2025-11-22
-- Dependencies: 009_update_analyses_to_11_frameworks.sql
--
-- CONTEXT:
-- - Current analysis statuses: 'pending', 'processing', 'completed', 'failed'
-- - Adding: 'approved' (admin approved analysis)
-- - Adding: 'sent' (report/analysis sent to client)
-- - Adding versioning: support for multiple analysis versions per submission
--
-- USE CASES FOR VERSIONING:
-- 1. Admin requests changes/improvements to analysis
-- 2. Company updates enrichment data and re-runs analysis
-- 3. Historical tracking of analysis iterations
-- 4. A/B testing different analysis approaches
--
-- WORKFLOW:
-- pending → processing → completed → approved → sent
--                     ↓
--                   failed
-- ==================================================================================

BEGIN;

-- ==================== 1. ADD VERSION COLUMNS ====================
-- Add versioning support to track multiple analysis iterations
ALTER TABLE analyses
ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1,
ADD COLUMN IF NOT EXISTS parent_analysis_id UUID REFERENCES analyses(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS is_latest BOOLEAN NOT NULL DEFAULT TRUE;

-- ==================== 2. ADD STATUS COLUMNS ====================
-- Add approval and sent tracking
ALTER TABLE analyses
ADD COLUMN IF NOT EXISTS approved_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS approved_by UUID, -- References user who approved
ADD COLUMN IF NOT EXISTS sent_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS sent_to VARCHAR(255); -- Email address report was sent to

-- ==================== 3. DROP EXISTING STATUS CONSTRAINT ====================
ALTER TABLE analyses
DROP CONSTRAINT IF EXISTS valid_analysis_status;

-- ==================== 4. ADD NEW STATUS CONSTRAINT ====================
-- Status Definitions:
-- - pending: Analysis queued but not started
-- - processing: Analysis frameworks being computed
-- - completed: All 11 frameworks completed successfully
-- - approved: Admin has approved the analysis for sending
-- - sent: Report/analysis has been sent to client
-- - failed: Analysis failed due to errors
ALTER TABLE analyses
ADD CONSTRAINT valid_analysis_status CHECK (status IN (
    'pending',
    'processing',
    'completed',
    'approved',    -- NEW: Admin approved analysis
    'sent',        -- NEW: Report sent to client
    'failed'
));

-- ==================== 5. ADD SOFT DELETE COLUMN ====================
-- Add soft delete support BEFORE creating indexes that use it
ALTER TABLE analyses
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

-- ==================== 6. UPDATE EXISTING DATA FOR VERSIONING ====================
-- Mark all existing analyses as version 1 and latest
UPDATE analyses
SET version = 1, is_latest = TRUE
WHERE version IS NULL OR version = 0;

-- ==================== 7. CREATE VERSIONING CONSTRAINTS ====================
-- Ensure only one analysis per submission can be marked as latest
CREATE UNIQUE INDEX IF NOT EXISTS idx_analyses_latest_per_submission
ON analyses(submission_id)
WHERE is_latest = TRUE AND deleted_at IS NULL;

-- Prevent circular references in parent_analysis_id
CREATE OR REPLACE FUNCTION prevent_analysis_circular_reference()
RETURNS TRIGGER AS $$
BEGIN
    -- Prevent self-reference
    IF NEW.parent_analysis_id = NEW.id THEN
        RAISE EXCEPTION 'Analysis cannot be its own parent';
    END IF;

    -- Prevent circular references (parent cannot be a child)
    IF NEW.parent_analysis_id IS NOT NULL THEN
        IF EXISTS (
            SELECT 1 FROM analyses
            WHERE id = NEW.parent_analysis_id
            AND parent_analysis_id = NEW.id
        ) THEN
            RAISE EXCEPTION 'Circular reference detected in analysis versioning';
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_prevent_analysis_circular_reference ON analyses;
CREATE TRIGGER trigger_prevent_analysis_circular_reference
    BEFORE INSERT OR UPDATE ON analyses
    FOR EACH ROW
    EXECUTE FUNCTION prevent_analysis_circular_reference();

-- ==================== 7. CREATE VERSION INCREMENT TRIGGER ====================
-- Automatically increment version when creating a new analysis from parent
CREATE OR REPLACE FUNCTION auto_increment_analysis_version()
RETURNS TRIGGER AS $$
DECLARE
    parent_version INTEGER;
BEGIN
    -- If parent_analysis_id is set, inherit and increment version
    IF NEW.parent_analysis_id IS NOT NULL THEN
        SELECT version INTO parent_version
        FROM analyses
        WHERE id = NEW.parent_analysis_id;

        IF parent_version IS NOT NULL THEN
            NEW.version = parent_version + 1;
        END IF;

        -- Mark parent as no longer latest
        UPDATE analyses
        SET is_latest = FALSE
        WHERE id = NEW.parent_analysis_id;
    END IF;

    -- Mark any other analyses for this submission as not latest
    UPDATE analyses
    SET is_latest = FALSE
    WHERE submission_id = NEW.submission_id
      AND id != NEW.id
      AND is_latest = TRUE;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_auto_increment_analysis_version ON analyses;
CREATE TRIGGER trigger_auto_increment_analysis_version
    BEFORE INSERT ON analyses
    FOR EACH ROW
    EXECUTE FUNCTION auto_increment_analysis_version();

-- ==================== 8. CREATE STATUS TIMESTAMP TRIGGERS ====================
-- Automatically set timestamps when status changes
CREATE OR REPLACE FUNCTION update_analysis_status_timestamps()
RETURNS TRIGGER AS $$
BEGIN
    -- Set approved_at when status changes to 'approved'
    IF NEW.status = 'approved' AND OLD.status != 'approved' AND NEW.approved_at IS NULL THEN
        NEW.approved_at = NOW();
    END IF;

    -- Set sent_at when status changes to 'sent'
    IF NEW.status = 'sent' AND OLD.status != 'sent' AND NEW.sent_at IS NULL THEN
        NEW.sent_at = NOW();
    END IF;

    -- Clear timestamps if status changes away from approved/sent
    IF NEW.status != 'approved' AND OLD.status = 'approved' THEN
        NEW.approved_at = NULL;
        NEW.approved_by = NULL;
    END IF;

    IF NEW.status != 'sent' AND OLD.status = 'sent' THEN
        NEW.sent_at = NULL;
        NEW.sent_to = NULL;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_analysis_status_timestamps ON analyses;
CREATE TRIGGER trigger_update_analysis_status_timestamps
    BEFORE UPDATE ON analyses
    FOR EACH ROW
    EXECUTE FUNCTION update_analysis_status_timestamps();

-- ==================== 9. CREATE PERFORMANCE INDICES ====================
-- Index for non-deleted analyses
CREATE INDEX IF NOT EXISTS idx_analyses_not_deleted
ON analyses(submission_id, created_at DESC)
WHERE deleted_at IS NULL;

-- ==================== 10. CREATE ADDITIONAL PERFORMANCE INDICES ====================
-- Index for version queries
CREATE INDEX IF NOT EXISTS idx_analyses_version
ON analyses(submission_id, version DESC, created_at DESC);

-- Index for parent-child relationships
CREATE INDEX IF NOT EXISTS idx_analyses_parent
ON analyses(parent_analysis_id)
WHERE parent_analysis_id IS NOT NULL;

-- Index for status and timestamps
CREATE INDEX IF NOT EXISTS idx_analyses_status_updated
ON analyses(status, updated_at DESC)
WHERE deleted_at IS NULL;

-- Index for approval queue (completed but not approved)
CREATE INDEX IF NOT EXISTS idx_analyses_approval_queue
ON analyses(completed_at DESC)
WHERE status = 'completed' AND approved_at IS NULL AND deleted_at IS NULL;

-- Index for sent analyses tracking
CREATE INDEX IF NOT EXISTS idx_analyses_sent
ON analyses(sent_at DESC)
WHERE status = 'sent' AND deleted_at IS NULL;

-- ==================== 11. ADD HELPER FUNCTION FOR VERSION HISTORY ====================
-- Function to get complete version history for a submission
CREATE OR REPLACE FUNCTION get_analysis_version_history(p_submission_id UUID)
RETURNS TABLE (
    id UUID,
    version INTEGER,
    status VARCHAR(50),
    parent_analysis_id UUID,
    is_latest BOOLEAN,
    created_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    approved_at TIMESTAMP WITH TIME ZONE,
    sent_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        a.id,
        a.version,
        a.status,
        a.parent_analysis_id,
        a.is_latest,
        a.created_at,
        a.completed_at,
        a.approved_at,
        a.sent_at
    FROM analyses a
    WHERE a.submission_id = p_submission_id
      AND a.deleted_at IS NULL
    ORDER BY a.version DESC, a.created_at DESC;
END;
$$ LANGUAGE plpgsql;

-- ==================== 12. ADD DOCUMENTATION ====================
COMMENT ON CONSTRAINT valid_analysis_status ON analyses IS
'Valid analysis statuses: pending → processing → completed → approved → sent (or failed at any point)';

COMMENT ON COLUMN analyses.version IS
'Version number for this analysis (auto-incremented from parent_analysis_id)';

COMMENT ON COLUMN analyses.parent_analysis_id IS
'Reference to previous version of analysis (null for first version)';

COMMENT ON COLUMN analyses.is_latest IS
'Indicates if this is the latest/current version for the submission';

COMMENT ON COLUMN analyses.approved_at IS
'Timestamp when analysis was approved by admin';

COMMENT ON COLUMN analyses.approved_by IS
'UUID of user who approved the analysis';

COMMENT ON COLUMN analyses.sent_at IS
'Timestamp when report/analysis was sent to client';

COMMENT ON COLUMN analyses.sent_to IS
'Email address the report was sent to';

COMMENT ON COLUMN analyses.deleted_at IS
'Soft delete timestamp (null for active records)';

-- ==================== 13. VERIFY DATA INTEGRITY ====================
DO $$
DECLARE
    orphaned_count INTEGER;
    inconsistent_count INTEGER;
    duplicate_latest INTEGER;
BEGIN
    -- Check for analyses without corresponding submissions
    SELECT COUNT(*) INTO orphaned_count
    FROM analyses a
    LEFT JOIN submissions s ON a.submission_id = s.id
    WHERE s.id IS NULL AND a.deleted_at IS NULL;

    IF orphaned_count > 0 THEN
        RAISE WARNING 'Found % orphaned analyses without corresponding submissions', orphaned_count;
    END IF;

    -- Check for analyses without corresponding enrichments
    SELECT COUNT(*) INTO orphaned_count
    FROM analyses a
    LEFT JOIN enrichments e ON a.enrichment_id = e.id
    WHERE e.id IS NULL AND a.deleted_at IS NULL;

    IF orphaned_count > 0 THEN
        RAISE WARNING 'Found % analyses without corresponding enrichments', orphaned_count;
    END IF;

    -- Check for approved analyses without approved_at
    SELECT COUNT(*) INTO inconsistent_count
    FROM analyses
    WHERE status = 'approved' AND approved_at IS NULL;

    IF inconsistent_count > 0 THEN
        RAISE WARNING 'Found % analyses with approved status but no timestamp', inconsistent_count;
        UPDATE analyses SET approved_at = updated_at WHERE status = 'approved' AND approved_at IS NULL;
        RAISE NOTICE 'Fixed % inconsistent approval timestamps', inconsistent_count;
    END IF;

    -- Check for sent analyses without sent_at
    SELECT COUNT(*) INTO inconsistent_count
    FROM analyses
    WHERE status = 'sent' AND sent_at IS NULL;

    IF inconsistent_count > 0 THEN
        RAISE WARNING 'Found % analyses with sent status but no timestamp', inconsistent_count;
        UPDATE analyses SET sent_at = updated_at WHERE status = 'sent' AND sent_at IS NULL;
        RAISE NOTICE 'Fixed % inconsistent sent timestamps', inconsistent_count;
    END IF;

    -- Check for multiple latest analyses per submission
    SELECT COUNT(*) INTO duplicate_latest
    FROM (
        SELECT submission_id, COUNT(*) as cnt
        FROM analyses
        WHERE is_latest = TRUE AND deleted_at IS NULL
        GROUP BY submission_id
        HAVING COUNT(*) > 1
    ) subq;

    IF duplicate_latest > 0 THEN
        RAISE WARNING 'Found % submissions with multiple latest analyses', duplicate_latest;
    END IF;
END $$;

COMMIT;

-- ==================== ROLLBACK INSTRUCTIONS ====================
-- To rollback this migration:
-- BEGIN;
-- DROP FUNCTION IF EXISTS get_analysis_version_history(UUID);
-- DROP TRIGGER IF EXISTS trigger_update_analysis_status_timestamps ON analyses;
-- DROP FUNCTION IF EXISTS update_analysis_status_timestamps();
-- DROP TRIGGER IF EXISTS trigger_auto_increment_analysis_version ON analyses;
-- DROP FUNCTION IF EXISTS auto_increment_analysis_version();
-- DROP TRIGGER IF EXISTS trigger_prevent_analysis_circular_reference ON analyses;
-- DROP FUNCTION IF EXISTS prevent_analysis_circular_reference();
-- ALTER TABLE analyses DROP CONSTRAINT IF EXISTS valid_analysis_status;
-- ALTER TABLE analyses ADD CONSTRAINT valid_analysis_status
--   CHECK (status IN ('pending', 'processing', 'completed', 'failed'));
-- ALTER TABLE analyses DROP COLUMN IF EXISTS version;
-- ALTER TABLE analyses DROP COLUMN IF EXISTS parent_analysis_id;
-- ALTER TABLE analyses DROP COLUMN IF EXISTS is_latest;
-- ALTER TABLE analyses DROP COLUMN IF EXISTS approved_at;
-- ALTER TABLE analyses DROP COLUMN IF EXISTS approved_by;
-- ALTER TABLE analyses DROP COLUMN IF EXISTS sent_at;
-- ALTER TABLE analyses DROP COLUMN IF EXISTS sent_to;
-- ALTER TABLE analyses DROP COLUMN IF EXISTS deleted_at;
-- COMMIT;
