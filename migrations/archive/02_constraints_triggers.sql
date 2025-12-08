-- =============================================================================
-- ARCHIVE: Constraints, Triggers, and Indexes (Migrations 002-004, 007)
-- Original files: 002_status_constraints.sql, 003_drop_deprecated_report_columns.sql,
--                 004_repair_constraints.sql, 007_report_indexes_and_constraints.sql
-- Status: APPLIED TO PRODUCTION
-- =============================================================================
-- This file is for historical reference only. DO NOT RUN.
-- =============================================================================

-- Migration 002: Added status CHECK constraints to all workflow tables
-- - Submission status: always 'received' (derived from children)
-- - Enrichment status: pending, processing, completed, approved, failed
-- - Analysis status: pending, processing, completed, approved, sent, failed
-- - Report status: pending, processing, completed, failed

-- Migration 003: Dropped deprecated report columns from early iterations

-- Migration 004: Repaired any broken foreign key constraints

-- Migration 007: Added indexes for report queries
-- - idx_reports_submission_id
-- - idx_reports_analysis_id
-- - idx_reports_status
