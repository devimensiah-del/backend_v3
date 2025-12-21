-- Migration: v2_023_add_service_areas.sql
-- Description: Add service_areas column to companies table
-- Date: 2024-12-21

-- =============================================================================
-- ALTER TABLE: companies
-- Add service_areas field (output from Step 2 enrichment)
-- =============================================================================

ALTER TABLE companies
  ADD COLUMN IF NOT EXISTS service_areas JSONB DEFAULT '[]';

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON COLUMN companies.service_areas IS 'Service areas where company provides services (from Step 2 enrichment)';
