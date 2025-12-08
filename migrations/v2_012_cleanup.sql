-- =============================================================================
-- Migration: v2_012_cleanup
-- Purpose: Remove unused columns and add missing indexes based on Phase 5 audit
-- Date: 2025-12-06
-- Replaces: N/A (cleanup migration)
-- Dependencies: All v2_001 through v2_011 migrations must be applied
-- =============================================================================

BEGIN;

-- =============================================================================
-- PART 1: Drop unused enrichment columns from companies table (v2_005)
-- =============================================================================
-- These columns were added in v2_005 but never used in Go code
-- Verified by domain/company/model.go and repository.go analysis

ALTER TABLE companies DROP COLUMN IF EXISTS sector_description;
ALTER TABLE companies DROP COLUMN IF EXISTS sector_growth_rate;
ALTER TABLE companies DROP COLUMN IF EXISTS sector_trends;
ALTER TABLE companies DROP COLUMN IF EXISTS key_industry_trends;
ALTER TABLE companies DROP COLUMN IF EXISTS macro_context;
ALTER TABLE companies DROP COLUMN IF EXISTS recent_news;
ALTER TABLE companies DROP COLUMN IF EXISTS competitive_landscape;
ALTER TABLE companies DROP COLUMN IF EXISTS data_confidence;
ALTER TABLE companies DROP COLUMN IF EXISTS data_gaps;
ALTER TABLE companies DROP COLUMN IF EXISTS enriched_at;
ALTER TABLE companies DROP COLUMN IF EXISTS enrichment_source;

COMMENT ON TABLE companies IS 'Enrichment data stored in foundation_year, legal_name, business_summary, etc. (direct fields)';

-- =============================================================================
-- PART 2: Drop orphaned field_deprecation_config table
-- =============================================================================
-- No Go model or repository queries use this table (verified via grep)

DROP TABLE IF EXISTS field_deprecation_config CASCADE;

-- =============================================================================
-- PART 3: Add missing index on analyses.challenge_id
-- =============================================================================
-- GetByChallengeID() uses this column but no index exists
-- Without index, this is a full table scan

CREATE INDEX IF NOT EXISTS idx_analyses_challenge_id ON analyses(challenge_id);

-- =============================================================================
-- PART 4: Make analyses.company_id NOT NULL (after populating any nulls)
-- =============================================================================
-- Every analysis must have a company (via challenge)
-- First populate any NULL values (shouldn't exist but be safe)

UPDATE analyses
SET company_id = (
    SELECT c.id
    FROM challenges ch
    JOIN companies c ON ch.company_id = c.id
    WHERE ch.id = analyses.challenge_id
)
WHERE company_id IS NULL;

-- Verify no nulls remain
DO $$
DECLARE
    null_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO null_count
    FROM analyses
    WHERE company_id IS NULL;

    IF null_count > 0 THEN
        RAISE EXCEPTION 'v2_012 ABORT: % analyses still have NULL company_id after population attempt', null_count;
    END IF;
END $$;

-- Now enforce NOT NULL constraint
ALTER TABLE analyses ALTER COLUMN company_id SET NOT NULL;

-- =============================================================================
-- PART 5: Documentation updates
-- =============================================================================

COMMENT ON COLUMN analyses.submission_id IS
  'Historical reference to originating submission. Nullable. May reference soft-deleted submissions (deleted_at IS NOT NULL).';

COMMENT ON COLUMN analyses.company_id IS
  'Company this analysis belongs to. Derived from challenge.company_id. NOT NULL enforced.';

COMMENT ON COLUMN analyses.challenge_id IS
  'Strategic challenge this analysis addresses. NOT NULL. Links to challenges(id).';

COMMIT;

-- =============================================================================
-- VERIFICATION QUERIES (run manually after migration)
-- =============================================================================

-- Verify cleanup
-- SELECT COUNT(*) FROM companies WHERE sector_description IS NOT NULL;  -- Should fail (column dropped)
-- SELECT COUNT(*) FROM field_deprecation_config;  -- Should fail (table dropped)
-- SELECT COUNT(*) FROM analyses WHERE company_id IS NULL;  -- Should be 0
-- \d analyses  -- Verify idx_analyses_challenge_id exists

-- =============================================================================
-- ROLLBACK (if needed)
-- =============================================================================

-- NOTE: Rolling back will NOT restore the dropped columns with their data
-- The dropped columns were never populated, so this is safe

-- BEGIN;
--
-- -- Recreate field_deprecation_config (empty)
-- CREATE TABLE field_deprecation_config (
--     field_name TEXT NOT NULL PRIMARY KEY,
--     category TEXT NOT NULL,
--     deprecation_months INTEGER NOT NULL,
--     description TEXT
-- );
--
-- -- Drop index
-- DROP INDEX IF EXISTS idx_analyses_challenge_id;
--
-- -- Remove NOT NULL constraint
-- ALTER TABLE analyses ALTER COLUMN company_id DROP NOT NULL;
--
-- -- Recreate unused columns (they will be empty)
-- ALTER TABLE companies ADD COLUMN sector_description TEXT;
-- ALTER TABLE companies ADD COLUMN sector_growth_rate VARCHAR(50);
-- ALTER TABLE companies ADD COLUMN sector_trends JSONB;
-- ALTER TABLE companies ADD COLUMN key_industry_trends TEXT[];
-- ALTER TABLE companies ADD COLUMN macro_context JSONB;
-- ALTER TABLE companies ADD COLUMN recent_news JSONB;
-- ALTER TABLE companies ADD COLUMN competitive_landscape JSONB;
-- ALTER TABLE companies ADD COLUMN data_confidence VARCHAR(20);
-- ALTER TABLE companies ADD COLUMN data_gaps TEXT[];
-- ALTER TABLE companies ADD COLUMN enriched_at TIMESTAMPTZ;
-- ALTER TABLE companies ADD COLUMN enrichment_source VARCHAR(50);
--
-- COMMIT;
