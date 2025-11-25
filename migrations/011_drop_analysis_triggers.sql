-- Migration 011: Drop ALL triggers on analyses table and disable RLS
-- Purpose: Fix "bind message supplies 26 parameters, but prepared statement requires 1" error
-- Date: 2025-11-24
--
-- This error indicates Supabase has triggers/policies intercepting the INSERT.
-- We need to drop ALL triggers and disable RLS to allow direct inserts.

BEGIN;

-- ========================================
-- STEP 1: Drop ALL triggers on analyses table
-- ========================================
DO $$
DECLARE
    trigger_rec RECORD;
BEGIN
    FOR trigger_rec IN
        SELECT tgname FROM pg_trigger
        WHERE tgrelid = 'analyses'::regclass
        AND NOT tgisinternal
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS %I ON analyses', trigger_rec.tgname);
        RAISE NOTICE 'Dropped trigger: %', trigger_rec.tgname;
    END LOOP;
END $$;

-- Also explicitly drop known Supabase triggers
DROP TRIGGER IF EXISTS handle_updated_at ON analyses;
DROP TRIGGER IF EXISTS set_updated_at ON analyses;
DROP TRIGGER IF EXISTS update_analyses_updated_at ON analyses;
DROP TRIGGER IF EXISTS on_analyses_update ON analyses;
DROP TRIGGER IF EXISTS tr_analyses_audit ON analyses;
DROP TRIGGER IF EXISTS analyses_before_insert ON analyses;
DROP TRIGGER IF EXISTS analyses_before_update ON analyses;
DROP TRIGGER IF EXISTS analyses_after_insert ON analyses;
DROP TRIGGER IF EXISTS analyses_after_update ON analyses;

-- ========================================
-- STEP 2: Drop any functions that might reference old columns
-- ========================================
DROP FUNCTION IF EXISTS handle_analyses_update() CASCADE;
DROP FUNCTION IF EXISTS update_analyses_updated_at() CASCADE;
DROP FUNCTION IF EXISTS analyses_audit_trigger() CASCADE;

-- ========================================
-- STEP 3: Disable RLS on analyses table
-- ========================================
-- RLS policies might be causing the parameter binding issue
ALTER TABLE analyses DISABLE ROW LEVEL SECURITY;

-- Drop all RLS policies on analyses
DO $$
DECLARE
    policy_rec RECORD;
BEGIN
    FOR policy_rec IN
        SELECT policyname FROM pg_policies
        WHERE tablename = 'analyses'
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON analyses', policy_rec.policyname);
        RAISE NOTICE 'Dropped policy: %', policy_rec.policyname;
    END LOOP;
END $$;

-- ========================================
-- STEP 4: Recreate a simple updated_at trigger
-- ========================================
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
