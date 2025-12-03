# Plan T-03: LLM Client Tests

**Track:** T-testing (parallel)
**Plan:** 03 of 04
**Status:** Ready (no dependencies)

---

## Objective

Add comprehensive tests for `llm/` package - currently only has sanitizer_test.go.

---

## Context

@file:llm/client.go - OpenRouter API client
@file:llm/prompts.go - Prompt templates
@file:llm/sanitizer.go - Output sanitizer (already tested)

**Critical functionality to test:**
- API request construction
- Response parsing
- Retry logic with fallback models
- Circuit breaker behavior
- Rate limit handling
- Structured output parsing

---

## Tasks

### Task 1: Create HTTP mock helper
**Type:** create
**Files:** `llm/mock_test.go`
**Action:**
Create mock HTTP server for testing:
```go
func newMockOpenRouter(t *testing.T, handler http.HandlerFunc) *httptest.Server {
    return httptest.NewServer(handler)
}

func mockSuccessResponse(content string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]interface{}{
            "choices": []map[string]interface{}{
                {"message": map[string]string{"content": content}},
            },
        })
    }
}

func mockErrorResponse(status int, code string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(status)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": map[string]string{"code": code},
        })
    }
}
```

**Verify:** `go build ./llm/...`

---

### Task 2: Create client tests
**Type:** create
**Files:** `llm/client_test.go`
**Action:**
Test cases:
- Generate returns parsed response
- Generate retries on 429 rate limit
- Generate uses fallback on primary failure
- Generate returns error after max retries
- GenerateStructured parses JSON correctly
- GenerateStructured handles malformed JSON
- Circuit breaker opens after failures
- Circuit breaker closes after success

**Verify:** `go test ./llm/... -run Client -v`

---

### Task 3: Create prompt tests
**Type:** create
**Files:** `llm/prompts_test.go`
**Action:**
Test cases:
- BuildPrompt substitutes variables correctly
- BuildPrompt handles missing variables
- Each framework prompt generates valid output
- Prompt templates don't exceed token limits (approximate)

**Verify:** `go test ./llm/... -run Prompt -v`

---

### Task 4: Create retry tests
**Type:** create
**Files:** `llm/retry_test.go`
**Action:**
Test cases:
- Retry with exponential backoff
- Retry stops at max attempts
- Retry uses fallback model on persistent failure
- Non-retryable errors fail immediately (400, 401)
- Retryable errors trigger retry (429, 500, 503)

**Verify:** `go test ./llm/... -run Retry -v`

---

## Verification

```bash
# All LLM tests pass
go test ./llm/... -v

# Coverage report
go test ./llm/... -cover

# Target: 70%+ coverage (API tests are harder)
```

---

## Success Criteria

- [ ] Mock HTTP server helper exists
- [ ] client_test.go exists with 8+ test cases
- [ ] prompts_test.go exists
- [ ] retry_test.go exists
- [ ] All tests pass
- [ ] 70%+ code coverage

---

## Output

Create `T-03-SUMMARY.md` documenting:
- Test files created
- Test count
- Coverage percentage
- Commit hash
