-- =============================================================================
-- ARCHIVE: Analysis Table Evolution (Migrations 006, 008, 011, 014, 015, 026, 028)
-- Original files: 006_simplify_analysis_schema.sql, 008_schema_alignment.sql,
--                 011_drop_analysis_triggers.sql, 014_analysis_access_code.sql,
--                 015_add_is_blurred.sql, 026_analyses_company_id.sql,
--                 028_analyses_pdf_columns.sql
-- Status: APPLIED TO PRODUCTION
-- =============================================================================
-- This file is for historical reference only. DO NOT RUN.
-- =============================================================================

-- Migration 006: Simplified analysis schema
-- - Consolidated framework columns into individual JSONB fields
-- - Added processing_time_ms for performance tracking

-- Migration 008: Schema alignment with frontend expectations
-- - Ensured consistent field names across API responses

-- Migration 011: Dropped automatic triggers on analysis status changes
-- - Moved trigger logic to application layer

-- Migration 014: Added access_code for public report sharing
-- - access_code VARCHAR for unique shareable links
-- - access_code_created_at for expiration tracking

-- Migration 015: Added is_blurred for paywall system
-- - is_blurred BOOLEAN DEFAULT true
-- - Controls whether analysis appears blurred to non-paying users

-- Migration 026: Added company_id foreign key
-- - Links analyses to verified companies table

-- Migration 028: Added PDF columns to analyses
-- - pdf_url TEXT for Supabase Storage URL
-- - pdf_generated_at TIMESTAMPTZ
