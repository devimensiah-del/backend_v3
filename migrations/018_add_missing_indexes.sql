-- Migration: Add missing indexes for frequently queried columns
-- These indexes improve performance for dashboard listings and user-specific queries

-- Index for user-specific submission lookups
-- Used in: submissions.List() with user_id filter
CREATE INDEX IF NOT EXISTS idx_submissions_user_id
ON submissions(user_id)
WHERE deleted_at IS NULL;

-- Index for email-based lookups
-- Used in: GetByEmail() for user-specific queries
CREATE INDEX IF NOT EXISTS idx_submissions_contact_email
ON submissions(contact_email)
WHERE deleted_at IS NULL;

-- Composite index for common listing queries
-- Used in: List() with ORDER BY created_at DESC
CREATE INDEX IF NOT EXISTS idx_submissions_listing
ON submissions(created_at DESC)
WHERE deleted_at IS NULL;

-- Index for analyses listing by creation date
-- Used in: analyses.List() ORDER BY created_at DESC
CREATE INDEX IF NOT EXISTS idx_analyses_created_at
ON analyses(created_at DESC)
WHERE deleted_at IS NULL;

-- Index for enrichments by creation date
-- Used in: GetBySubmissionID() ORDER BY created_at DESC
CREATE INDEX IF NOT EXISTS idx_enrichments_created_at
ON enrichments(created_at DESC);

-- Add comments for documentation
COMMENT ON INDEX idx_submissions_user_id IS 'Speeds up user-specific submission queries';
COMMENT ON INDEX idx_submissions_contact_email IS 'Speeds up email-based lookups';
COMMENT ON INDEX idx_submissions_listing IS 'Optimizes dashboard listing queries';
COMMENT ON INDEX idx_analyses_created_at IS 'Optimizes analysis listing queries';
COMMENT ON INDEX idx_enrichments_created_at IS 'Optimizes enrichment lookups by submission';
