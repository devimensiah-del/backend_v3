-- Migration 029: Simplify enrichment and analysis statuses
-- Purpose: Remove approval workflow - verified fields protect data instead

-- ============================================
-- ENRICHMENT STATUS SIMPLIFICATION
-- ============================================
-- Old: pending, completed, approved
-- New: pending, completed, failed

-- Convert any 'approved' enrichments to 'completed'
UPDATE enrichments SET status = 'completed' WHERE status = 'approved';

-- Drop old constraint and add new one
ALTER TABLE enrichments DROP CONSTRAINT IF EXISTS enrichments_status_check;
ALTER TABLE enrichments ADD CONSTRAINT enrichments_status_check
    CHECK (status IN ('pending', 'completed', 'failed'));

-- ============================================
-- ANALYSIS STATUS SIMPLIFICATION
-- ============================================
-- Old: pending, processing, completed, approved, sent
-- New: pending, processing, completed, failed

-- Convert 'approved' and 'sent' to 'completed' (is_visible_to_user handles visibility)
UPDATE analyses SET status = 'completed' WHERE status IN ('approved', 'sent');

-- Drop old constraint and add new one
ALTER TABLE analyses DROP CONSTRAINT IF EXISTS analyses_status_check;
ALTER TABLE analyses ADD CONSTRAINT analyses_status_check
    CHECK (status IN ('pending', 'processing', 'completed', 'failed'));

COMMENT ON COLUMN enrichments.status IS 'Workflow status: pending (waiting), completed (done), failed (error)';
COMMENT ON COLUMN analyses.status IS 'Workflow status: pending (queued), processing (running), completed (done), failed (error)';
