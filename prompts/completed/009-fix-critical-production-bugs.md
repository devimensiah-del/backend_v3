# Critical Production Bug Fixes

## Context

This prompt addresses 3 critical bugs identified during a comprehensive code review that could cause data loss, job failures, and production incidents in the IMENSIAH backend.

## Bug #1: Worker Startup Race - Permanent Job Discard (CRITICAL)

### Location
`jobs/worker.go:469-472` and `jobs/worker.go:532-535`

### Problem
The macro job handlers return `asynq.SkipRetry` when `macroService == nil`:

```go
if w.macroService == nil {
    jobLogger.Error().Msg("Macro service not configured - skipping job")
    return fmt.Errorf("%w: macro service not configured", asynq.SkipRetry)
}
```

**Why this is dangerous**: `asynq.SkipRetry` permanently discards the job with NO retry. If a macro job arrives during the brief window between worker startup and `SetMacroService()` injection, it's lost forever.

### Fix Required
Replace `asynq.SkipRetry` with a standard retryable error. The job will then retry after a delay, by which time the service should be injected:

```go
if w.macroService == nil {
    jobLogger.Error().Msg("Macro service not yet configured - will retry")
    return fmt.Errorf("macro service not yet configured")
}
```

### Files to Modify
- `jobs/worker.go` - Lines ~469-472 (HandleMacroFetchJob)
- `jobs/worker.go` - Lines ~532-535 (HandleMacroRefreshAllJob)

---

## Bug #2: JSON Parsing Fragility in LLM Responses (HIGH PRIORITY)

### Location
- `llm/client.go:112-115`
- `domain/enrichment/service.go:188-193`

### Problem
Both files use `strings.TrimPrefix` to clean markdown code blocks from LLM responses:

```go
cleanJson := strings.TrimSpace(resp.Content)
cleanJson = strings.TrimPrefix(cleanJson, "```json")
cleanJson = strings.TrimPrefix(cleanJson, "```")
cleanJson = strings.TrimSuffix(cleanJson, "```")
```

**Why this is dangerous**: LLMs sometimes add conversational text before the JSON:

```
Here's the analysis you requested:

```json
{"key": "value"}
```
```

The current approach fails to extract the JSON from this output because the conversational text isn't removed.

### Fix Required
Use index-based extraction to find the actual JSON object/array boundaries:

```go
// Find JSON boundaries (handles conversational text before/after JSON)
cleanJSON := strings.TrimSpace(content)

// Try to find JSON object or array
startObj := strings.Index(cleanJSON, "{")
startArr := strings.Index(cleanJSON, "[")
endObj := strings.LastIndex(cleanJSON, "}")
endArr := strings.LastIndex(cleanJSON, "]")

// Determine which structure we have (object vs array)
start := -1
end := -1
if startObj != -1 && endObj != -1 {
    if startArr == -1 || startObj < startArr {
        start, end = startObj, endObj
    }
}
if startArr != -1 && endArr != -1 {
    if start == -1 || startArr < start {
        start, end = startArr, endArr
    }
}

if start != -1 && end != -1 && end > start {
    cleanJSON = cleanJSON[start : end+1]
}
```

### Files to Modify
- `llm/client.go` - Replace lines 112-115 in `GenerateStructuredWithOptions`
- `domain/enrichment/service.go` - Replace lines 188-193 in `parseEnrichmentResponse`

---

## Bug #3: Missing Database Performance Indexes (MEDIUM)

### Location
`migrations/` - New migration file needed

### Problem
Several query patterns lack appropriate indexes:
1. `submissions` - No index on `lower(contact_email)` for case-insensitive lookups
2. `challenges` - No index on `company_id` for company→challenges queries
3. `company_submissions` - No index on `submission_id` for join lookups

### Fix Required
Create a new migration file `v2_013_add_performance_indexes.sql`:

```sql
-- Performance indexes for common query patterns
-- Safe to run in production (CREATE INDEX CONCURRENTLY)

-- Case-insensitive email lookup on submissions
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_submissions_email_lower
ON submissions (lower(contact_email));

-- Challenge lookups by company
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_challenges_company_id
ON challenges (company_id) WHERE deleted_at IS NULL;

-- Company-submission join optimization
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_company_submissions_submission_id
ON company_submissions (submission_id);
```

### Files to Create
- `migrations/v2_013_add_performance_indexes.sql`

---

## Verification Steps

After applying fixes:

1. **Bug #1 Verification**:
   ```bash
   # Start worker without injecting macroService, enqueue a macro job
   # Observe job retries instead of being discarded
   ```

2. **Bug #2 Verification**:
   ```bash
   go test -v ./llm/... -run TestJSONParsing
   go test -v ./domain/enrichment/... -run TestParseEnrichmentResponse
   ```
   Add test cases with conversational LLM responses.

3. **Bug #3 Verification**:
   ```bash
   # In psql:
   EXPLAIN ANALYZE SELECT * FROM submissions WHERE lower(contact_email) = 'test@example.com';
   # Should show "Index Scan" instead of "Seq Scan"
   ```

---

## Implementation Order

1. **Fix Bug #1 first** - Prevents data loss in production
2. **Fix Bug #2 second** - Improves LLM response handling robustness
3. **Fix Bug #3 last** - Performance improvement, not urgent

---

## Testing Requirements

- [ ] Unit test for JSON extraction with conversational prefix
- [ ] Unit test for JSON extraction with markdown code blocks
- [ ] Unit test for JSON extraction with array responses
- [ ] Integration test verifying macro job retries on nil service
- [ ] Query plan verification for new indexes
