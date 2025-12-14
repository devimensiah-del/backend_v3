-- Migration: v2_020_drop_legacy_wizard_tables.sql
-- Description: Drop deprecated wizard tables after verifying no data is needed
-- Jira: IAH-2
--
-- ⚠️ SAFETY WARNING ⚠️
-- DO NOT RUN THIS MIGRATION UNTIL:
-- 1. All wizard-related code is removed from frontend_v2
-- 2. Any existing analysis_steps data has been migrated or confirmed disposable
-- 3. The analysisbysteps package (analysis_steps_v2) is fully functional
--
-- This migration is IRREVERSIBLE. Backup data if needed.

-- Drop the deprecated wizard table
DROP TABLE IF EXISTS analysis_steps CASCADE;

-- Drop the deprecated frameworks table (dynamic framework configuration)
DROP TABLE IF EXISTS frameworks CASCADE;

-- Note: analysis_steps_v2 (created in v2_019) is the replacement
-- and should already be in use by the analysisbysteps package

-- Verification query (run before dropping to check data):
-- SELECT COUNT(*) as step_count FROM analysis_steps;
-- SELECT COUNT(*) as framework_count FROM frameworks;
