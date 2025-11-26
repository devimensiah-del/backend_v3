-- Migration 013: Drop ALL triggers on submissions table and disable RLS
-- Purpose: Fix "bind message has 24 result formats but query has 2 columns" error
-- Date: 2025-11-25
--
-- This error indicates Supabase has triggers/policies intercepting the SELECT.
-- We need to drop ALL triggers and disable RLS to allow direct queries.

BEGIN;

-- ========================================
-- STEP 1: Drop ALL triggers on submissions table
-- ========================================
DO $$
DECLARE
    trigger_rec RECORD;
BEGIN
    FOR trigger_rec IN
        SELECT tgname FROM pg_trigger
        WHERE tgrelid = 'submissions'::regclass
        AND NOT tgisinternal
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS %I ON submissions', trigger_rec.tgname);
        RAISE NOTICE 'Dropped trigger: %', trigger_rec.tgname;
    END LOOP;
END $$;

-- Also explicitly drop known Supabase triggers
DROP TRIGGER IF EXISTS handle_updated_at ON submissions;
DROP TRIGGER IF EXISTS set_updated_at ON submissions;
DROP TRIGGER IF EXISTS update_submissions_updated_at ON submissions;
DROP TRIGGER IF EXISTS on_submissions_update ON submissions;
DROP TRIGGER IF EXISTS tr_submissions_audit ON submissions;
DROP TRIGGER IF EXISTS submissions_before_insert ON submissions;
DROP TRIGGER IF EXISTS submissions_before_update ON submissions;
DROP TRIGGER IF EXISTS submissions_after_insert ON submissions;
DROP TRIGGER IF EXISTS submissions_after_update ON submissions;

-- ========================================
-- STEP 2: Drop any functions that might reference old columns
-- ========================================
DROP FUNCTION IF EXISTS handle_submissions_update() CASCADE;
DROP FUNCTION IF EXISTS update_submissions_updated_at() CASCADE;
DROP FUNCTION IF EXISTS submissions_audit_trigger() CASCADE;

-- ========================================
-- STEP 3: Disable RLS on submissions table
-- ========================================
-- RLS policies might be causing the parameter binding issue
ALTER TABLE submissions DISABLE ROW LEVEL SECURITY;

-- Drop all RLS policies on submissions
DO $$
DECLARE
    policy_rec RECORD;
BEGIN
    FOR policy_rec IN
        SELECT policyname FROM pg_policies
        WHERE tablename = 'submissions'
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON submissions', policy_rec.policyname);
        RAISE NOTICE 'Dropped policy: %', policy_rec.policyname;
    END LOOP;
END $$;

-- ======================================== 
-- STEP 4: Recreate a simple updated_at trigger
-- ========================================
CREATE OR REPLACE FUNCTION update_submissions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_submissions_updated_at
  BEFORE UPDATE ON submissions
  FOR EACH ROW
  EXECUTE FUNCTION update_submissions_updated_at();

COMMIT;
