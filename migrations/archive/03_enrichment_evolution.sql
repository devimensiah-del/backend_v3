-- =============================================================================
-- ARCHIVE: Enrichment Table Evolution (Migrations 005, 009, 010, 021, 023)
-- Original files: 005_enrichment_idempotency.sql, 009_drop_enrichment_dead_columns.sql,
--                 010_drop_enrichment_triggers.sql, 021_enrichment_auto_analyze.sql,
--                 023_enrichment_approved_at.sql
-- Status: APPLIED TO PRODUCTION
-- =============================================================================
-- This file is for historical reference only. DO NOT RUN.
-- =============================================================================

-- Migration 005: Added idempotency checks to prevent duplicate enrichments

-- Migration 009: Dropped unused columns from enrichment table
-- - Removed: enriched_company_overview, enriched_market_analysis, etc.
-- - Consolidated into single 'data' JSONB column

-- Migration 010: Dropped automatic triggers on enrichment status changes
-- - Moved trigger logic to application layer for better control

-- Migration 021: Added auto_trigger_analysis boolean
-- - When true, automatically starts analysis after enrichment completes

-- Migration 023: Added approved_at timestamp
-- - Tracks when enrichment was approved by admin
