-- MVP Simplification Migration
-- Drop unused columns from analyses table
-- Drop unused tables (frameworks, llm_usage_logs, macro_*)
--
-- This migration simplifies the schema for MVP launch by:
-- 1. Removing deprecated/unused columns from analyses
-- 2. Dropping the dynamic frameworks table (hardcoding in code)
-- 3. Dropping LLM usage logging (never used)
-- 4. Dropping macroeconomics tables (hardcoding indicator values)

-- ============================================
-- PART 1: Remove unused columns from analyses
-- ============================================

-- is_blurred: Deprecated in v2_010 - wizard flow replaces paywall blur
ALTER TABLE analyses DROP COLUMN IF EXISTS is_blurred;

-- approved_at/approved_by: Duplicates of analysis_steps.approved_at/by
ALTER TABLE analyses DROP COLUMN IF EXISTS approved_at;
ALTER TABLE analyses DROP COLUMN IF EXISTS approved_by;

-- sent_at/sent_to: Never set or read anywhere in codebase
ALTER TABLE analyses DROP COLUMN IF EXISTS sent_at;
ALTER TABLE analyses DROP COLUMN IF EXISTS sent_to;

-- ============================================
-- PART 2: Drop unused tables
-- ============================================

-- LLM usage logging table - never populated, dead code
DROP TABLE IF EXISTS llm_usage_logs;

-- Dynamic frameworks table - hardcoding in code instead
DROP TABLE IF EXISTS frameworks;

-- ============================================
-- PART 3: Drop macroeconomics tables
-- ============================================
-- Hardcoding indicator values in analysis workflow
-- Drop in correct order due to foreign key constraints

DROP TABLE IF EXISTS macro_fetch_logs;
DROP TABLE IF EXISTS macro_indicator_values;
DROP TABLE IF EXISTS macro_indicator_sources;
DROP TABLE IF EXISTS macro_indicator_types;
DROP TABLE IF EXISTS macro_data_sources;

-- ============================================
-- Summary of changes:
-- - analyses: 5 columns dropped
-- - Tables dropped: 8 (frameworks, llm_usage_logs, 5x macro_*)
-- - Schema now has 8 tables (was 16)
-- ============================================
