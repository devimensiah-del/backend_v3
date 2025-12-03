# Plan T-04: Jobs Worker Handler Tests

**Track:** T-testing (parallel)
**Plan:** 04 of 04
**Status:** Ready (no dependencies)

---

## Objective

Add comprehensive tests for `jobs/worker.go` handlers - currently only has utility test.

---

## Context

@file:jobs/worker.go - Job handlers for enrichment, analysis
@file:jobs/types/types.go - Job type constants

**Critical functionality to test:**
- HandleEnrichmentJob executes enrichment workflow
- HandleAnalysisJob executes analysis workflow
- Job error handling and retries
- Payload parsing
- Service integration

---

## Tasks

### Task 1: Create service mocks
**Type:** create
**Files:** `jobs/mocks_test.go`
**Action:**
Create mocks for services:
```go
type MockEnrichmentService struct {
    mock.Mock
}

func (m *MockEnrichmentService) Enrich(ctx context.Context, submissionID string) error {
    args := m.Called(ctx, submissionID)
    return args.Error(0)
}

type MockAnalysisService struct {
    mock.Mock
}

func (m *MockAnalysisService) RunAnalysis(ctx context.Context, submissionID, enrichmentID string) error {
    args := m.Called(ctx, submissionID, enrichmentID)
    return args.Error(0)
}
```

**Verify:** `go build ./jobs/...`

---

### Task 2: Create enrichment job handler tests
**Type:** create
**Files:** `jobs/enrichment_handler_test.go`
**Action:**
Test cases:
- HandleEnrichmentJob calls service with correct ID
- HandleEnrichmentJob returns nil on success
- HandleEnrichmentJob returns error on service failure
- HandleEnrichmentJob handles invalid payload
- HandleEnrichmentJob respects context cancellation

**Verify:** `go test ./jobs/... -run Enrichment -v`

---

### Task 3: Create analysis job handler tests
**Type:** create
**Files:** `jobs/analysis_handler_test.go`
**Action:**
Test cases:
- HandleAnalysisJob calls service with correct IDs
- HandleAnalysisJob returns nil on success
- HandleAnalysisJob returns error on service failure
- HandleAnalysisJob handles invalid payload
- HandleAnalysisJob respects context timeout

**Verify:** `go test ./jobs/... -run Analysis -v`

---

### Task 4: Create payload parsing tests
**Type:** create
**Files:** `jobs/payload_test.go`
**Action:**
Test cases:
- EnrichmentPayload marshals correctly
- EnrichmentPayload unmarshals correctly
- AnalysisPayload marshals correctly
- AnalysisPayload unmarshals correctly
- Invalid JSON returns error

**Verify:** `go test ./jobs/... -run Payload -v`

---

### Task 5: Create worker integration test
**Type:** create
**Files:** `jobs/worker_integration_test.go`
**Action:**
Test full worker with Redis (if available):
- Enqueue job → worker processes → verify completion

Use build tag `// +build integration`

**Verify:** `go test ./jobs/... -tags=integration -v`

---

## Verification

```bash
# All jobs tests pass
go test ./jobs/... -v

# Coverage report
go test ./jobs/... -cover

# Target: 75%+ coverage
```

---

## Success Criteria

- [ ] Service mocks created
- [ ] Enrichment handler tests exist with 5+ cases
- [ ] Analysis handler tests exist with 5+ cases
- [ ] Payload parsing tests exist
- [ ] Integration test exists (optional)
- [ ] All tests pass
- [ ] 75%+ code coverage

**Testing Track Complete:**
After T-04 completes, all critical test gaps are filled.

---

## Output

Create `T-04-SUMMARY.md` documenting:
- Test files created
- Test count
- Coverage percentage
- Commit hash
