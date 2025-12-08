# Archived Migrations (001-031)

These migrations have already been applied to production and are kept here for historical reference only.

**DO NOT RUN THESE** - Use `000_baseline.sql` for fresh setups instead.

## Consolidated Files

| File | Original Migrations | Purpose |
|------|---------------------|---------|
| `01_initial_schema.sql` | 001 | Core tables creation |
| `02_constraints_triggers.sql` | 002-004, 007 | Status constraints, triggers, indexes |
| `03_enrichment_evolution.sql` | 005, 009, 010, 021, 023 | Enrichment table changes |
| `04_analysis_evolution.sql` | 006, 008, 011, 014, 015, 026, 028 | Analysis table changes |
| `05_submission_changes.sql` | 012, 013 | Submission CNPJ, trigger cleanup |
| `06_companies_macroeconomics.sql` | 017-020, 022, 024, 025, 027 | Companies table, macro indicators |
| `07_visibility_deprecation.sql` | 029-031 | Status simplification, visibility |

## Migration History

- **Production baseline**: Migrations 001-031 applied
- **Archived**: December 2024
- **New migrations**: Use `v2_*` prefix
