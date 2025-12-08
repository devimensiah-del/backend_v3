-- =============================================================================
-- Migration: v2_005_company_enrichment
-- Purpose: Merge enriched data into companies table, simplify enrichments
-- Date: 2024-12-02
-- Updated: 2024-12-06 (Removed PART 1 - macro_indicators doesn't exist in prod)
-- Replaces: 040_company_enrichment_merge.sql, 041_simplify_enrichments.sql
-- =============================================================================
--
-- NOTE: Part 1 (SELIC manual update) was removed because production uses the
-- multi-table macro data structure (macro_indicator_types, macro_indicator_values)
-- instead of the simple macro_indicators table. SELIC values should be managed
-- through the existing macro_indicator_values table.
--
-- =============================================================================

-- =============================================================================
-- PART 1: Add enriched fields to companies table
-- Company becomes single source of truth for all company data
-- =============================================================================

-- Sector/Industry enriched data
ALTER TABLE companies ADD COLUMN IF NOT EXISTS sector_description TEXT;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS sector_growth_rate VARCHAR(50);
ALTER TABLE companies ADD COLUMN IF NOT EXISTS sector_trends JSONB;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS key_industry_trends TEXT[];

-- Macro context (SELIC, IPCA, USD/BRL snapshot at enrichment time)
ALTER TABLE companies ADD COLUMN IF NOT EXISTS macro_context JSONB;

-- Recent news about the company/sector
ALTER TABLE companies ADD COLUMN IF NOT EXISTS recent_news JSONB;

-- Competitive landscape
ALTER TABLE companies ADD COLUMN IF NOT EXISTS competitive_landscape JSONB;

-- Data quality tracking
ALTER TABLE companies ADD COLUMN IF NOT EXISTS data_confidence VARCHAR(20);
ALTER TABLE companies ADD COLUMN IF NOT EXISTS data_gaps TEXT[];

-- Enrichment tracking
ALTER TABLE companies ADD COLUMN IF NOT EXISTS enriched_at TIMESTAMPTZ;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS enrichment_source VARCHAR(50);

COMMENT ON COLUMN companies.sector_description IS 'AI-enriched sector analysis';
COMMENT ON COLUMN companies.macro_context IS 'Macro indicators snapshot at enrichment time';
COMMENT ON COLUMN companies.enriched_at IS 'When enrichment was run - NULL means not enriched yet';

-- =============================================================================
-- PART 2: Simplify enrichments table
-- Actual enriched data now lives in companies table
-- =============================================================================

ALTER TABLE enrichments ADD COLUMN IF NOT EXISTS raw_ai_response JSONB;
ALTER TABLE enrichments ADD COLUMN IF NOT EXISTS processing_time_ms INTEGER;

COMMENT ON COLUMN enrichments.status IS 'Status: pending, completed, failed. Approved state deprecated.';
COMMENT ON COLUMN enrichments.data IS 'DEPRECATED: Enriched data now stored in companies table.';
