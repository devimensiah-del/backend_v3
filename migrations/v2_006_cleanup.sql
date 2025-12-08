-- =============================================================================
-- Migration: v2_007_cleanup
-- Purpose: Remove deprecated tables and simplify schema
-- Date: 2024-12-02
-- Replaces: 043_remove_field_verification.sql
-- =============================================================================

-- Drop the field_verifications table (no longer used)
DROP TABLE IF EXISTS company_field_verifications CASCADE;
DROP INDEX IF EXISTS idx_field_verifications_company;
DROP INDEX IF EXISTS idx_field_verifications_field;

-- Mark is_verified as deprecated but keep for potential future use
COMMENT ON COLUMN companies.is_verified IS 'Simple reviewed flag. Not actively used in business logic.';
