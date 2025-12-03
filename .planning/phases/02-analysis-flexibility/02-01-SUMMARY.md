# Summary: 02-01 Add framework_results JSONB Column

**Status:** Complete
**Date:** 2025-12-02

## Files Created/Modified

| File | Action | Description |
|------|--------|-------------|
| migrations/034_add_framework_results.sql | created | Add JSONB column with GIN index for dynamic framework storage |
| domain/analysis/model.go | modified | Add FrameworkResults field and syncFrameworkResults() helper |
| domain/analysis/repository.go | modified | Implement dual-write in Create(), Update(), UpdateWithTx() |
| domain/analysis/repository_test.go | modified | Update test mocks to include framework_results and is_public |

## Dual-Write Implementation

- **syncFrameworkResults() helper**: Syncs 11 individual framework fields + synthesis → framework_results map
- **Called before Create() and Update() operations**: Ensures both storage formats stay synchronized
- **Enables rollback if issues discovered**: Old columns remain functional during transition period

### Key Implementation Details

1. **Migration 034**: Creates `framework_results JSONB DEFAULT '{}'` with GIN index for efficient JSONB queries
2. **Model field**: `FrameworkResults map[string]json.RawMessage` stores all frameworks in one column
3. **Sync logic**: Only stores non-empty frameworks (skips `{}` and `null` values)
4. **Repository changes**: All INSERT/UPDATE queries include framework_results parameter

## Verification Results

- **Build**: ✅ PASS
- **Vet**: ✅ PASS
- **Tests**: ✅ PASS (39 tests)

### Test Coverage

All existing tests passing:
- Repository CRUD operations (Create, Update, UpdateWithTx, GetByID, List, Delete)
- JSONB serialization for all 11 frameworks
- Transaction handling
- Service layer operations

## Migration SQL

```sql
ALTER TABLE analyses
ADD COLUMN IF NOT EXISTS framework_results JSONB DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_analyses_framework_results
ON analyses USING gin(framework_results);
```

## Deviations

None - Implementation follows plan exactly.

## Next Steps

1. Run migration 034 on database: `psql $DATABASE_URL -f migrations/034_add_framework_results.sql`
2. Verify dual-write in production environment
3. Monitor for any sync issues between old and new columns
4. Future phase: Migrate API consumers to use framework_results, then deprecate individual columns

## Technical Notes

- **Storage format**: `{"pestel": {...}, "swot": {...}, "porter": {...}, ...}`
- **Backward compatibility**: All 11 individual JSONB columns remain functional
- **Performance**: GIN index enables efficient querying of specific frameworks
- **Error handling**: syncFrameworkResults() returns errors if JSON marshaling fails

## Commit

**Hash**: bf99565
**Message**: feat(02-01): Add framework_results JSONB column
