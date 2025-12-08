-- =============================================================================
-- Migration: v2_012_simplify_visibility
-- Purpose: Simplify visibility fields - deprecate is_blurred for wizard flow
-- Date: 2024-12-05
-- =============================================================================
--
-- CONTEXT:
-- The wizard-based human-in-the-loop flow replaces the paywall/blur concept.
-- Instead of blurring premium frameworks, the wizard guides users through
-- sequential approval of each framework.
--
-- CHANGES:
-- - Set is_blurred = false for all existing analyses
-- - Change default to false (was true)
-- - Add clarifying comments
--
-- REMAINING VISIBILITY FIELDS:
-- - is_visible_to_user: Admin enables for user dashboard visibility
-- - is_public: Accessible via access_code without authentication
-- - access_code: Shareable link code
-- - access_code_created_at: For expiration tracking
--
-- =============================================================================

BEGIN;

-- Update all existing analyses to not be blurred
UPDATE analyses SET is_blurred = false WHERE is_blurred = true;

-- Change default from true to false
ALTER TABLE analyses ALTER COLUMN is_blurred SET DEFAULT false;

-- Add clarifying comments
COMMENT ON COLUMN analyses.is_visible_to_user IS 'Admin enables for user dashboard visibility. Default false until admin approves.';
COMMENT ON COLUMN analyses.is_blurred IS 'DEPRECATED: Wizard flow replaces paywall blur. Should always be false.';
COMMENT ON COLUMN analyses.is_public IS 'Accessible via access_code without authentication for sharing reports';
COMMENT ON COLUMN analyses.access_code IS 'Shareable link code for public access (e.g., /report/:code)';
COMMENT ON COLUMN analyses.access_code_created_at IS 'Timestamp for access code expiration tracking';

COMMIT;

-- =============================================================================
-- DOWN MIGRATION (Rollback)
-- =============================================================================
-- To rollback:
-- BEGIN;
-- ALTER TABLE analyses ALTER COLUMN is_blurred SET DEFAULT true;
-- -- Note: Cannot restore original is_blurred values without backup
-- COMMIT;
