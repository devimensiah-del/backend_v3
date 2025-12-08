-- =============================================================================
-- ARCHIVE: Submission Table Changes (Migrations 012, 013)
-- Original files: 012_add_submission_cnpj.sql, 013_drop_submission_triggers.sql
-- Status: APPLIED TO PRODUCTION
-- =============================================================================
-- This file is for historical reference only. DO NOT RUN.
-- =============================================================================

-- Migration 012: Added CNPJ field to submissions
-- - cnpj VARCHAR for Brazilian company tax ID
-- - Optional field for better company identification

-- Migration 013: Dropped automatic triggers on submission creation
-- - Removed auto-create enrichment trigger
-- - Now handled explicitly in application layer
