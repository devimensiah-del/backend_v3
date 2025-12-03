# IMENSIAH Backend v3 Refactor - Roadmap

**Version:** v1.0
**Created:** 2025-12-02
**Status:** Planning

---

## Milestone: v1.0 - Flexible, Clean Backend

**Goal:** Dynamic frameworks, no report domain, i18n errors, full test coverage

**Dependency Graph:**
```
Phase 1 (Foundation)
    └── Phase 2 (Analysis Flexibility)
            └── Phase 3 (Report Removal)
                    └── Phase 5 (Cleanup)

Phase 4 (Error Handling) ──── Parallel (independent)
Track T (Testing) ─────────── Parallel (independent)
```

**Critical Path:** 1 → 2 → 3 → 5
**Parallel Tracks:** 4, T (can execute anytime)

---

## Phase 1: Foundation
**Status:** Complete ✅
**Risk:** Low
**Dependency:** None

Create the foundational components needed for analysis flexibility.

| Plan | Scope | Tasks | Status |
|------|-------|-------|--------|
| 01-01 | Framework domain package | model.go, repository interface, service | ✅ |
| 01-02 | Framework database | migration, seed 11 frameworks | ✅ |
| 01-03 | Centralized logging | pkg/logging with contextual fields | ⬜ (skipped - zerolog already used) |
| 01-04 | Wire framework service | main.go integration, API endpoints | ✅ |

**Exit Criteria:**
- ✅ `domain/framework/` package exists with full CRUD
- ✅ `frameworks` table has 11 seeded frameworks
- ⚠️ Logging: Using existing zerolog, centralized pkg not needed
- ✅ `/api/v1/frameworks` endpoint returns framework list

---

## Phase 2: Analysis Flexibility
**Status:** Blocked (needs Phase 1)
**Risk:** Medium
**Dependency:** Phase 1 complete

Transform analysis from 11 hardcoded columns to dynamic framework execution.

| Plan | Scope | Tasks | Status |
|------|-------|-------|--------|
| 02-01 | Schema evolution | Add framework_results JSONB column | ⬜ |
| 02-02 | Repository refactor | Generic storage, dual-write period | ⬜ |
| 02-03 | Workflow refactor | Data-driven execution from DB | ⬜ |
| 02-04 | Migration completion | Migrate data, drop old columns | ⬜ |

**Exit Criteria:**
- Analysis uses `framework_results` JSONB map
- Workflow loads frameworks from database
- Old 11 JSONB columns removed
- All existing analyses migrated

---

## Phase 3: Report Removal
**Status:** Blocked (needs Phase 2)
**Risk:** High (31 files, breaking API)
**Dependency:** Phase 2 complete

Remove vestigial report domain - analysis already has PDF columns.

| Plan | Scope | Tasks | Status |
|------|-------|-------|--------|
| 03-01 | API cleanup | Remove report handlers, routes | ⬜ |
| 03-02 | Jobs cleanup | Remove report job handler | ⬜ |
| 03-03 | Main.go cleanup | Remove report initialization, adapters | ⬜ |
| 03-04 | Domain deletion | Delete domain/report/ directory | ⬜ |
| 03-05 | Database cleanup | Drop reports table, remove Gotenberg | ⬜ |

**Exit Criteria:**
- Zero imports of `backend_v3/domain/report`
- Zero references to TypeReport job
- `infrastructure/gotenberg.go` deleted
- `reports` table dropped (if exists)
- Build passes, all tests pass

---

## Phase 4: Error Handling (Parallel)
**Status:** Not Started
**Risk:** Low
**Dependency:** None (can run parallel to 1-3)

Implement i18n error handling with Portuguese support.

| Plan | Scope | Tasks | Status |
|------|-------|-------|--------|
| 04-01 | Error package | pkg/errors with AppError struct | ⬜ |
| 04-02 | Error catalog | Define all user-facing errors EN/PT | ⬜ |
| 04-03 | API integration | Update handlers, Accept-Language | ⬜ |

**Exit Criteria:**
- All user-facing errors have Portuguese translations
- API respects Accept-Language header
- Error responses include code + localized message

---

## Phase 5: Cleanup
**Status:** Blocked (needs Phase 3)
**Risk:** Low
**Dependency:** Phase 3 complete

Final cleanup of dead code, naming, artifacts.

| Plan | Scope | Tasks | Status |
|------|-------|-------|--------|
| 05-01 | Dead code removal | Unused functions, imports, comments | ⬜ |
| 05-02 | Naming standards | Consistent struct/method naming | ⬜ |
| 05-03 | Artifact cleanup | Delete *.exe, fix typo directories | ⬜ |

**Exit Criteria:**
- No dead code warnings
- Consistent naming per Go conventions
- No build artifacts in repo
- All comments accurate

---

## Track T: Testing (Parallel)
**Status:** Not Started
**Risk:** Medium
**Dependency:** None (can run parallel to all phases)

Fill critical test coverage gaps identified in audit.

| Plan | Scope | Tasks | Status |
|------|-------|-------|--------|
| T-01 | Company tests | service_test.go, repository mocks | ⬜ |
| T-02 | Macroeconomics tests | service_test.go, scheduler tests | ⬜ |
| T-03 | LLM client tests | client_test.go, retry/fallback tests | ⬜ |
| T-04 | Jobs handler tests | worker_test.go with mocked services | ⬜ |

**Exit Criteria:**
- `domain/company/service_test.go` exists with 80%+ coverage
- `domain/macroeconomics/service_test.go` exists
- `llm/client_test.go` exists with mock API tests
- `jobs/worker_test.go` tests all job handlers

---

## Execution Order

**Recommended sequence (with parallelism):**

```
Week 1:
├── [Sequential] 01-01, 01-02, 01-03, 01-04 (Foundation)
└── [Parallel] T-01, T-02 (Company + Macro tests)

Week 2:
├── [Sequential] 02-01, 02-02, 02-03, 02-04 (Analysis)
└── [Parallel] T-03, T-04 (LLM + Jobs tests)

Week 3:
├── [Sequential] 03-01, 03-02, 03-03, 03-04, 03-05 (Report removal)
└── [Parallel] 04-01, 04-02, 04-03 (Error handling)

Week 4:
└── [Sequential] 05-01, 05-02, 05-03 (Cleanup)
```

**Total:** 23 atomic plans
**Critical path:** 16 plans (Phases 1-3-5)
**Parallel:** 7 plans (Phase 4 + Track T)

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Report removal breaks frontend | Coordinate with frontend team before 03-01 |
| Data migration corrupts analyses | Dual-write in 02-02, verify before 02-04 |
| Framework change breaks AI | Keep prompt templates identical initially |
| Test flakiness blocks progress | Isolate flaky tests, fix in T-track |

---

## Files Reference

**Key files to modify (by phase):**
- Phase 1: main.go, new domain/framework/*, new pkg/logging/*
- Phase 2: domain/analysis/*, migrations/032-034
- Phase 3: main.go, api/*, jobs/*, domain/report/* (delete)
- Phase 4: api/*, new pkg/errors/*
- Phase 5: Various cleanup across codebase

**Audit reference:** `analysis/CODEBASE_AUDIT.md`
