# Plan T-01: Company Domain Tests

**Track:** T-testing (parallel)
**Plan:** 01 of 04
**Status:** Ready (no dependencies)

---

## Objective

Add comprehensive tests for `domain/company/` - currently has 0 tests.

---

## Context

@file:domain/company/model.go - Company entity
@file:domain/company/repository.go - Database operations
@file:domain/company/service.go - Business logic

**Critical business logic to test:**
- Company CRUD operations
- CNPJ validation
- Owner/allowed users permissions
- Field verification logic

---

## Tasks

### Task 1: Create repository tests with sqlmock
**Type:** create
**Files:** `domain/company/repository_test.go`
**Action:**
Test cases:
- Create company successfully
- Create fails on duplicate CNPJ
- GetByID returns company
- GetByID returns nil for not found
- GetByCNPJ returns company
- ListByOwner returns all owned companies
- Update modifies fields
- Delete soft-deletes (sets deleted_at)
- AddAllowedUser works correctly
- RemoveAllowedUser works correctly

Use `github.com/DATA-DOG/go-sqlmock` for mocking.

**Verify:** `go test ./domain/company/... -run Repository -v`

---

### Task 2: Create service tests with mocked repository
**Type:** create
**Files:** `domain/company/service_test.go`
**Action:**
Test cases:
- Create validates CNPJ format
- Create prevents duplicate CNPJ
- GetByID returns error for not found
- GetByID returns error for not owner
- Update only allowed by owner
- Delete only allowed by owner
- VerifyField updates verification status
- IsUserAllowed checks owner and allowed_users

Use `testify/mock` for repository mock.

**Verify:** `go test ./domain/company/... -run Service -v`

---

### Task 3: Create validation tests
**Type:** create
**Files:** `domain/company/validation_test.go`
**Action:**
Test CNPJ validation specifically:
- Valid CNPJ formats
- Invalid CNPJ (wrong checksum)
- Invalid CNPJ (wrong length)
- Empty CNPJ
- CNPJ with special characters

**Verify:** `go test ./domain/company/... -run Validation -v`

---

## Verification

```bash
# All company tests pass
go test ./domain/company/... -v

# Coverage report
go test ./domain/company/... -cover

# Target: 80%+ coverage
```

---

## Success Criteria

- [ ] repository_test.go exists with 10+ test cases
- [ ] service_test.go exists with 8+ test cases
- [ ] validation_test.go exists
- [ ] All tests pass
- [ ] 80%+ code coverage

---

## Output

Create `T-01-SUMMARY.md` documenting:
- Test files created
- Test count
- Coverage percentage
- Commit hash
