# IMENSIAH Backend v3 - Comprehensive Refactor

**Created:** 2025-12-02
**Status:** Active
**Version Target:** v1.0

---

## Vision

Transform the IMENSIAH backend from a rigid, 11-framework monolith into a flexible, maintainable platform that:

1. **Supports dynamic frameworks** - Add/remove analysis frameworks without code changes
2. **Eliminates technical debt** - Remove vestigial report domain (now handled by analysis)
3. **Enables i18n** - Portuguese-first error messages for Brazilian market
4. **Improves observability** - Centralized, structured logging with request correlation
5. **Increases reliability** - Fill critical test gaps (company, macroeconomics, llm)

---

## Current State

### Architecture
- Go REST API with Gin framework
- Domain-Driven Design: `submission → enrichment → analysis → report`
- Background jobs via Asynq/Redis
- AI via OpenRouter (4-model approach: PreSearch, Enrichment, Primary, Synthesis)
- PostgreSQL via Supabase, PDF via Gotenberg

### Critical Issues (from audit)

| Issue | Impact | Complexity |
|-------|--------|------------|
| Report domain vestigial | Dead code, confusion | High (31 files) |
| 11 hardcoded frameworks | Product inflexibility | Medium |
| Scattered logging | Debugging difficulty | Low |
| English-only errors | Brazilian UX | Low |
| Missing tests (company, macro, llm) | Reliability risk | Medium |

### Key Finding
Migration 028 already moved `pdf_url`, `pdf_generated_at` to analyses table. Report domain is now vestigial - analysis IS the report.

---

## Success Criteria

### v1.0 Complete When:
- [ ] Frameworks table exists and analysis uses it dynamically
- [ ] Report domain completely removed (0 references)
- [ ] All errors have Portuguese translations
- [ ] Centralized logger with request_id, submission_id correlation
- [ ] Test coverage for company, macroeconomics, llm domains
- [ ] Build passes with no dead code
- [ ] All migrations applied successfully

### Measurable Targets:
- Report references: 31 files → 0 files
- Framework flexibility: 0 (hardcoded) → Full (database-driven)
- Error i18n: 0% → 100% user-facing errors
- Test coverage gaps: 3 domains → 0 domains

---

## Constraints

1. **Frontend dependency**: Frontend currently calls `/api/v1/submissions/:id/report` - coordinate removal
2. **Database migrations**: Must be reversible, tested in staging first
3. **No breaking API changes** for non-report endpoints
4. **Maintain existing tests** - all must pass throughout refactor

---

## Out of Scope (v1.0)

- API versioning (v2)
- New framework implementations
- Performance optimization beyond removing dead code
- Frontend changes (separate repo)

---

## References

- `analysis/CODEBASE_AUDIT.md` - Comprehensive 10-area audit document
- `CLAUDE.md` - Project context and patterns
- `migrations/028_analyses_pdf_columns.sql` - PDF migration proof
