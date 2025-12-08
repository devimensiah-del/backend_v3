-- =============================================================================
-- ARCHIVE: Visibility & Status Changes (Migrations 016, 029-031)
-- Original files: 016_llm_usage_logs.sql, 029_simplify_statuses.sql,
--                 029a_revert_for_production.sql, 030_field_deprecation.sql,
--                 031_analysis_is_public.sql
-- Status: APPLIED TO PRODUCTION
-- =============================================================================
-- This file is for historical reference only. DO NOT RUN.
-- =============================================================================

-- Migration 016: Created LLM usage logging
-- - llm_usage_logs table for cost tracking
-- - Tracks tokens, cost, latency per framework

-- Migration 029: Simplified status values
-- - Standardized status enums across tables
-- - Note: 029a reverted some changes for production compatibility

-- Migration 030: Added field deprecation configuration
-- - field_deprecation_config table
-- - Tracks how long fields stay "fresh"

-- Migration 031: Added is_public to analyses
-- - is_public BOOLEAN DEFAULT false
-- - Allows analyses to be publicly viewable without access code
