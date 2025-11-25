-- Migration 011: Drop triggers on analyses table that reference dropped columns
-- Purpose: Fix "record 'new' has no field 'parent_analysis_id'" error
-- Date: 2025-11-24
--
-- The analyses table had columns (parent_analysis_id, version, is_latest) dropped in migration 006,
-- but Supabase may have auto-created triggers that still reference these columns.
-- This migration drops those problematic triggers.

BEGIN;

-- Drop any triggers on the analyses table that might reference dropped columns
-- Common Supabase auto-generated trigger names:
DROP TRIGGER IF EXISTS handle_updated_at ON analyses;
DROP TRIGGER IF EXISTS set_updated_at ON analyses;
DROP TRIGGER IF EXISTS update_analyses_updated_at ON analyses;
DROP TRIGGER IF EXISTS on_analyses_update ON analyses;
DROP TRIGGER IF EXISTS tr_analyses_audit ON analyses;
DROP TRIGGER IF EXISTS analyses_before_insert ON analyses;
DROP TRIGGER IF EXISTS analyses_before_update ON analyses;
DROP TRIGGER IF EXISTS analyses_after_insert ON analyses;
DROP TRIGGER IF EXISTS analyses_after_update ON analyses;

-- Drop any function that might reference the dropped columns
DROP FUNCTION IF EXISTS handle_analyses_update() CASCADE;
DROP FUNCTION IF EXISTS update_analyses_updated_at() CASCADE;
DROP FUNCTION IF EXISTS analyses_audit_trigger() CASCADE;

-- Recreate a simple updated_at trigger that doesn't reference dropped columns
CREATE OR REPLACE FUNCTION update_analyses_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_analyses_updated_at
  BEFORE UPDATE ON analyses
  FOR EACH ROW
  EXECUTE FUNCTION update_analyses_updated_at();

COMMIT;
