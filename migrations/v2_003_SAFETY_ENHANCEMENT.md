# v2_003 Migration Safety Enhancement

**Date:** 2025-12-06
**Based on:** Phase 5 Audit Recommendation (019-migrations-issues.md, Issue #5)

## Issue

Migration `v2_003_drop_legacy_columns.sql` drops framework columns from the `analyses` table without verifying that `v2_002_framework_results.sql` completed successfully. This could cause permanent data loss if v2_002 failed.

## Recommended Enhancement

Add the following safety check **BEFORE** the DROP COLUMN statements in `v2_003_drop_legacy_columns.sql`:

```sql
-- =============================================================================
-- SAFETY CHECK: Verify v2_002 ran successfully before dropping columns
-- Added: 2025-12-06 (Phase 5 audit recommendation)
-- =============================================================================
-- This prevents data loss if v2_002 migration didn't complete

DO $$
DECLARE
    unmigrated_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO unmigrated_count
    FROM analyses
    WHERE (framework_results IS NULL OR framework_results = '{}')
      AND (pestel IS NOT NULL OR porter IS NOT NULL OR swot IS NOT NULL
           OR tam_sam_som IS NOT NULL OR benchmarking IS NOT NULL
           OR blue_ocean IS NOT NULL OR growth_hacking IS NOT NULL
           OR scenarios IS NOT NULL OR okrs IS NOT NULL
           OR bsc IS NOT NULL OR decision_matrix IS NOT NULL
           OR synthesis IS NOT NULL);

    IF unmigrated_count > 0 THEN
        RAISE EXCEPTION 'v2_003 ABORT: % analyses have framework data in old columns but not in framework_results. Run v2_002 first!', unmigrated_count;
    END IF;

    -- Log success
    RAISE NOTICE 'Safety check passed: All framework data migrated to framework_results';
END $$;
```

## Location

Insert this block immediately after the header comments and before line 10:
```sql
ALTER TABLE analyses DROP COLUMN IF EXISTS pestel;
```

## Impact

- **Risk Mitigation:** Prevents accidental data loss if v2_002 didn't complete
- **Zero Downtime:** Check runs in milliseconds
- **Clear Errors:** Exception message shows exact count of unmigrated rows

## Status

**RECOMMENDED BUT NOT APPLIED**: Due to file locking during cleanup phase, this enhancement was not applied to the migration file. It should be added manually before running migrations in any new environment.

For production databases that have already run v2_003, this enhancement is not needed (columns are already dropped).

## Verification

To verify v2_002 completed successfully (run before applying v2_003):

```sql
SELECT COUNT(*) FROM analyses WHERE framework_results IS NULL OR framework_results = '{}';
-- Expected: 0 (or only rows where all old framework columns are also NULL)

SELECT COUNT(*) FROM analyses WHERE pestel IS NOT NULL;
-- If this returns > 0 after v2_002, v2_002 needs investigation
```

## Related Files

- Migration: `migrations/v2_003_drop_legacy_columns.sql`
- Data Migration: `migrations/v2_002_framework_results.sql`
- Audit Report: `docs/audit/019-migrations-issues.md` (Issue #5)
- Phase Summary: `docs/audit/PHASE5-SUMMARY.md`
