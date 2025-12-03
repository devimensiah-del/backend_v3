# Plan T-02: Macroeconomics Domain Tests

**Track:** T-testing (parallel)
**Plan:** 02 of 04
**Status:** Ready (no dependencies)

---

## Objective

Add comprehensive tests for `domain/macroeconomics/` - currently has 0 tests.

---

## Context

@file:domain/macroeconomics/model.go - Indicator entities
@file:domain/macroeconomics/repository.go - Database operations
@file:domain/macroeconomics/service.go - Business logic, scheduling

**Critical functionality to test:**
- Indicator CRUD
- Snapshot creation/retrieval
- Scheduled updates
- Cache behavior
- External API integration

---

## Tasks

### Task 1: Create repository tests
**Type:** create
**Files:** `domain/macroeconomics/repository_test.go`
**Action:**
Test cases:
- SaveIndicator creates new indicator
- SaveIndicator updates existing indicator
- GetLatestByCode returns most recent
- GetLatestSnapshot returns all latest indicators
- ListIndicators with pagination
- GetHistorical returns time series

Use sqlmock for database mocking.

**Verify:** `go test ./domain/macroeconomics/... -run Repository -v`

---

### Task 2: Create service tests
**Type:** create
**Files:** `domain/macroeconomics/service_test.go`
**Action:**
Test cases:
- GetLatestSnapshot returns cached data
- GetLatestSnapshot fetches if cache expired
- RefreshIndicators calls all providers
- RefreshIndicators handles provider errors gracefully
- GetIndicatorHistory returns formatted data
- ScheduleUpdates registers timers correctly

**Verify:** `go test ./domain/macroeconomics/... -run Service -v`

---

### Task 3: Create provider mock
**Type:** create
**Files:** `domain/macroeconomics/provider_mock_test.go`
**Action:**
Create mock provider for testing:
```go
type MockProvider struct {
    mock.Mock
}

func (m *MockProvider) FetchSELIC(ctx context.Context) (*Indicator, error) {
    args := m.Called(ctx)
    return args.Get(0).(*Indicator), args.Error(1)
}
// ... other methods
```

**Verify:** `go build ./domain/macroeconomics/...`

---

### Task 4: Create integration test
**Type:** create
**Files:** `domain/macroeconomics/integration_test.go`
**Action:**
Test full flow with real database (if available):
- Save indicator → retrieve → verify
- Create snapshot → get latest → verify

Use build tag `// +build integration`

**Verify:** `go test ./domain/macroeconomics/... -tags=integration -v`

---

## Verification

```bash
# All macroeconomics tests pass
go test ./domain/macroeconomics/... -v

# Coverage report
go test ./domain/macroeconomics/... -cover

# Target: 75%+ coverage
```

---

## Success Criteria

- [ ] repository_test.go exists with 6+ test cases
- [ ] service_test.go exists with 6+ test cases
- [ ] Provider mock created
- [ ] Integration test exists (optional)
- [ ] All tests pass
- [ ] 75%+ code coverage

---

## Output

Create `T-02-SUMMARY.md` documenting:
- Test files created
- Test count
- Coverage percentage
- Commit hash
