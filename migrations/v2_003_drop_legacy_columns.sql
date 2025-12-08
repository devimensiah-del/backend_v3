-- =============================================================================
-- Migration: v2_003_drop_legacy_columns
-- Purpose: Remove legacy framework columns and reports table
-- Date: 2024-12-02
-- Replaces: 036_drop_legacy_framework_columns.sql, 037_drop_reports_table.sql
-- WARNING: This is destructive - ensure v2_002 has been run and verified first!
-- =============================================================================

-- Drop legacy framework columns (data now in framework_results)
ALTER TABLE analyses DROP COLUMN IF EXISTS pestel;
ALTER TABLE analyses DROP COLUMN IF EXISTS porter;
ALTER TABLE analyses DROP COLUMN IF EXISTS swot;
ALTER TABLE analyses DROP COLUMN IF EXISTS tam_sam_som;
ALTER TABLE analyses DROP COLUMN IF EXISTS benchmarking;
ALTER TABLE analyses DROP COLUMN IF EXISTS blue_ocean;
ALTER TABLE analyses DROP COLUMN IF EXISTS growth_hacking;
ALTER TABLE analyses DROP COLUMN IF EXISTS scenarios;
ALTER TABLE analyses DROP COLUMN IF EXISTS okrs;
ALTER TABLE analyses DROP COLUMN IF EXISTS bsc;
ALTER TABLE analyses DROP COLUMN IF EXISTS decision_matrix;
ALTER TABLE analyses DROP COLUMN IF EXISTS synthesis;

COMMENT ON TABLE analyses IS 'Analysis results - framework data stored in framework_results JSONB column';

-- =============================================================================
-- DROP REPORTS TABLE
-- PDF generation tracking moved to analyses table (pdf_url, pdf_generated_at)
-- =============================================================================

DROP TABLE IF EXISTS reports CASCADE;
DROP INDEX IF EXISTS idx_reports_submission_id;
DROP INDEX IF EXISTS idx_reports_analysis_id;
DROP INDEX IF EXISTS idx_reports_status;
