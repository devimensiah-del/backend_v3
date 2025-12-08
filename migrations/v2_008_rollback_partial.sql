-- =============================================================================
-- v2_008 PARTIAL ROLLBACK
-- Run this ONLY if v2_008 failed mid-way and left partial state
-- After running this, re-run the corrected v2_008_challenge_entity.sql
-- =============================================================================

-- Drop the challenge_id column from analyses (if it was added)
ALTER TABLE analyses DROP COLUMN IF EXISTS challenge_id;

-- Drop the challenges table (if it was created)
DROP TABLE IF EXISTS challenges CASCADE;

-- Verify clean state
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'analyses' AND column_name = 'challenge_id') THEN
        RAISE EXCEPTION 'challenge_id column still exists on analyses';
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_name = 'challenges') THEN
        RAISE EXCEPTION 'challenges table still exists';
    END IF;

    RAISE NOTICE 'Rollback complete. Ready to re-run v2_008_challenge_entity.sql';
END $$;
