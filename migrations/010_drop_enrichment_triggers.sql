-- Migration 010: Drop triggers on enrichments table that reference dropped columns
-- Purpose: Fix "record 'new' has no field 'approved_at'" error
-- Date: 2025-11-24
--
-- Supabase may have auto-created triggers that reference the dropped columns.
-- We need to drop those triggers.

BEGIN;

-- Drop any triggers on the enrichments table
-- List all triggers first, then drop them
DO $$
DECLARE
    trigger_record RECORD;
BEGIN
    FOR trigger_record IN
        SELECT trigger_name
        FROM information_schema.triggers
        WHERE event_object_table = 'enrichments'
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS %I ON enrichments', trigger_record.trigger_name);
        RAISE NOTICE 'Dropped trigger: %', trigger_record.trigger_name;
    END LOOP;
END $$;

-- Also drop the handle_updated_at trigger if it exists (common Supabase pattern)
DROP TRIGGER IF EXISTS handle_updated_at ON enrichments;
DROP TRIGGER IF EXISTS set_updated_at ON enrichments;
DROP TRIGGER IF EXISTS update_enrichments_updated_at ON enrichments;

-- Recreate a simple updated_at trigger that doesn't reference dropped columns
CREATE OR REPLACE FUNCTION update_enrichments_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_enrichments_updated_at
    BEFORE UPDATE ON enrichments
    FOR EACH ROW
    EXECUTE FUNCTION update_enrichments_updated_at();

COMMIT;
