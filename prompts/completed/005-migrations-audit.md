<objective>
Audit ALL database migrations for correctness, alignment with business logic, and consistency with repository files.

This is Phase 5 of the comprehensive codebase audit. Every migration must be reviewed to ensure the database schema supports the business logic documented in domain audits.
</objective>

<context>
Read @CLAUDE.md for migration structure overview.
Read domain audit files in @./docs/audit/ to understand required data structures.

Migration structure:
```
migrations/
├── 000_baseline.sql       # Production schema snapshot (001-031 consolidated)
├── archive/               # Historical reference only (DO NOT RUN)
├── v2_001_frameworks_table.sql    # Dynamic frameworks
├── v2_002_framework_results.sql   # Consolidate JSONB
├── v2_003_drop_legacy_columns.sql # Remove old columns
├── v2_004_wizard_system.sql       # Human-in-the-loop
├── v2_005_company_enrichment.sql  # Enriched data → companies
├── v2_006_submission_challenges.sql
└── v2_007_cleanup.sql
```

Key tables:
- `submissions` - Entry data, linked to optional `user_id`
- `companies` - Verified company records with enriched data
- `analyses` - Framework outputs in `framework_results` JSONB
- `frameworks` - Dynamic framework configuration
- `analysis_steps` - Wizard step tracking
- `macro_indicators` - Economic indicators
</context>

<audit_checklist>

**For Each Migration:**

1. **Correctness**
   - Is the SQL syntactically correct?
   - Are constraints properly defined?
   - Are indexes appropriate?
   - Are foreign keys correct?

2. **Business Logic Alignment**
   - Does the schema support the domain models?
   - Are required fields present?
   - Are data types appropriate?
   - Do constraints match business rules?

3. **Repository Compatibility**
   - Does the schema match repository queries?
   - Are column names consistent with model fields?
   - Are any repository queries using deprecated columns?

4. **Migration Safety**
   - Is the migration reversible?
   - Are there potential data loss scenarios?
   - Is the migration idempotent where possible?
   - Are there appropriate DEFAULT values?

5. **Naming Conventions**
   - Are table/column names consistent?
   - Do names match domain terminology?
   - Are there any legacy naming inconsistencies?

**Baseline Review (000_baseline.sql):**
- Is it a complete representation of production schema?
- Are all tables, indexes, and constraints included?
- Is it consistent with the v2_* migrations?

**Cross-Reference Validation:**
- Compare each domain's model.go with table schema
- Compare each domain's repository.go with schema columns
- Identify any mismatches or deprecated usage
</audit_checklist>

<output>
Create audit files:
- `./docs/audit/017-migrations-inventory.md` - Complete list of all migrations with status
- `./docs/audit/018-schema-domain-mapping.md` - Maps each table to domain package
- `./docs/audit/019-migrations-issues.md` - Issues found in migrations

Create `./docs/audit/PHASE5-SUMMARY.md` with:
- Migration review summary
- Schema-repository mismatches
- Recommended schema fixes
- Data integrity concerns
</output>

<constraints>
- Do NOT modify migrations - audit only
- Note any pending migrations that haven't been applied
- Flag any migrations that might cause data loss
- Identify deprecated columns still in use
</constraints>

<verification>
Before completing:
- Every migration file reviewed (baseline + all v2_*)
- Schema mapped to domain models
- Repository queries validated against schema
- All issues documented with severity
</verification>
