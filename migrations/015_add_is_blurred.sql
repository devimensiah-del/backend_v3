-- Migration: 015_add_is_blurred.sql
-- Description: Add blur control for premium frameworks on analysis
-- When TRUE (default), premium frameworks are blurred for users
-- Admin can toggle this to reveal full content

-- Add blur control column
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS is_blurred BOOLEAN NOT NULL DEFAULT TRUE;

-- Documentation
COMMENT ON COLUMN analyses.is_blurred IS
    'Controls premium framework blur overlay for users. TRUE=blurred (default), FALSE=fully visible. Independent of is_visible_to_user which controls access.';

-- Index for filtering
CREATE INDEX IF NOT EXISTS idx_analyses_is_blurred ON analyses(is_blurred) WHERE deleted_at IS NULL;
