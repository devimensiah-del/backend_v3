# Summary: 01-02 Framework Database Migration & Seed

**Status:** Complete
**Date:** 2025-12-02

---

## Files Created

| File | Lines | Purpose |
|------|-------|---------|
| `migrations/032_create_frameworks_table.sql` | 36 | Table schema with indexes and comments |
| `migrations/033_seed_frameworks.sql` | 424 | Seed data for all 11 frameworks |

**Total:** 460 lines of SQL

---

## Migration Details

### 032_create_frameworks_table.sql
- Creates `frameworks` table with 17 columns
- 3 indexes: `code` (unique), `category`, `is_active` (partial)
- Column comments for documentation

### 033_seed_frameworks.sql
- Inserts 11 framework records with full metadata

| Layer | Category | Frameworks | Dependencies |
|-------|----------|------------|--------------|
| 1 | environment | pestel, porter, tam_sam_som | None |
| 2 | positioning | swot, benchmarking | swot→[pestel,porter] |
| 3 | strategy | blue_ocean, growth_hacking, scenarios | Various |
| 4 | execution | okrs, bsc, decision_matrix | Various |

---

## Output Schema Validation

All 11 `output_schema` JSONB objects validated against `domain/analysis/model.go`:
- ✅ PESTEL: 6 arrays + summary
- ✅ Porter: 7 forces + intensities + implications
- ✅ TAM-SAM-SOM: Values + scenarios + confidence
- ✅ SWOT: 4 quadrants with confidence/source
- ✅ Benchmarking: Competitors + gaps + practices
- ✅ Blue Ocean: ERRC + value curve
- ✅ Growth Hacking: LEAP + SCALE loops
- ✅ Scenarios: 3 scenarios + tactics + signals
- ✅ OKRs: 90-day plan + phases + capacity
- ✅ BSC: 4 perspectives
- ✅ Decision Matrix: Recommendations + legal

---

## Deviations

### 1. Prompt Template Storage
- **Plan:** Store full prompt templates in database
- **Actual:** Used reference placeholders (e.g., "See llm/prompts.go:FrameworkPESTELPrompt")
- **Reason:** Each prompt is 100-400 lines. Full text would create 10K+ line migration.
- **Impact:** Minor. Prompts loaded from code until dynamic loader implemented.

### 2. Dependency Configuration (Enhancement)
- **Plan:** Not specified in detail
- **Actual:** Added intelligent `depends_on` arrays:
  - swot → [pestel, porter]
  - growth_hacking → [swot, tam_sam_som]
  - blue_ocean → [porter]
  - okrs → [blue_ocean, decision_matrix]
  - bsc → [blue_ocean]
  - decision_matrix → [scenarios]
- **Impact:** Positive. Enables proper topological execution ordering.

---

## Verification

- SQL syntax: Valid (reviewed)
- Files exist: ✓
- Schema matches model.go: ✓
- Database application: Pending (user to run separately)

---

## Next Steps

Apply migrations to development database:
```bash
psql $DATABASE_URL -f migrations/032_create_frameworks_table.sql
psql $DATABASE_URL -f migrations/033_seed_frameworks.sql
```

---

## Commit

See commit hash below after execution.
