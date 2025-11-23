# Integration Tests

Comprehensive end-to-end integration testing framework for the Imensiah backend.

## Overview

This testing framework supports **flexible mixed configurations**:

- **All Real**: Uses actual OpenRouter API, Supabase storage (requires credentials)
- **All Mock**: Uses in-memory mocks for all external services (no credentials needed)
- **Mixed** ⭐: Combine real and mock services (e.g., real LLM + mock storage)

The framework automatically detects available credentials or allows manual override.

### 🎯 Recommended: Real LLM + JSON Mock Storage

```go
env := LoadTestEnvWithOverrides(false, true)
// LLM: REAL (OpenRouter) | Storage: MOCK (JSON) | Scraper: MOCK
```

**Benefits:**
- ✅ Real LLM responses for integration testing
- ✅ Fast local storage (no Supabase dependency)
- ✅ Quick test execution
- ✅ No storage costs

## Structure

```
integration_tests/
├── test_env.go              # Environment loader with mixed configuration support
├── helpers.go               # Test utilities (setup, cleanup, assertions)
├── workflow_test.go         # End-to-end workflow tests
├── workflow_mixed_test.go   # NEW: Mixed configuration tests ⭐
└── mocks/
    ├── supabase_json_mock.go      # NEW: JSON-based Supabase mock ⭐
    ├── supabase_mock_data.json    # Test data and scenarios
    ├── example_usage_test.go      # Usage examples
    ├── QUICKSTART.md              # 60-second quick start
    ├── llm_mock.go                # Mock OpenRouter/LLM client
    ├── supabase_mock.go           # Legacy in-memory mock
    └── scraper_mock.go            # Mock web scraper
```

## Running Tests

### Prerequisites

1. **Database**: PostgreSQL must be running and accessible
2. **Optional**: `.env` file with OpenRouter key for real LLM testing

### Run All Integration Tests

```bash
cd backend_v3/integration_tests
go test -v
```

### Run Mixed Configuration Tests (Recommended)

```bash
# Real LLM + JSON Mock Storage
go test -v -run TestMixedConfiguration
```

### Run Specific Test Categories

```bash
# Original workflow tests
go test -v -run TestEndToEndWorkflow

# Mixed configuration tests
go test -v -run TestMixed

# Configuration override tests
go test -v -run TestConfigurationOverrides

# JSON mock feature tests
cd mocks && go test -v -run TestExample
```

### Run Benchmarks

```bash
go test -bench=. -benchmem
```

## Configuration

### Environment Variables

The framework reads these variables from `../.env`:

**Required for Real Mode:**
- `DATABASE_URL` - PostgreSQL connection string
- `OPENROUTER_API_KEY` - OpenRouter API key for LLM calls
- `SUPABASE_URL` - Supabase project URL
- `SUPABASE_ANON_KEY` - Supabase anonymous/service role key

**Optional:**
- `REDIS_ADDR` - Redis address (default: localhost:6379)
- `REDIS_PASSWORD` - Redis password
- `GOTENBERG_URL` - Gotenberg service URL

### Flexible Configuration Options

#### 1. Automatic Detection (Default)

```go
env := LoadTestEnv()
// Auto-detects based on .env credentials
```

#### 2. Manual Override (Recommended for Your Use Case)

```go
// Real OpenRouter LLM + JSON Mock Storage
env := LoadTestEnvWithOverrides(
    false, // useLLMMock - use REAL OpenRouter
    true,  // useStorageMock - use JSON mock
)
```

#### 3. Check Current Configuration

```go
env := LoadTestEnv()
t.Logf("Config: %s", env.GetConfigSummary())
// Output: LLM: REAL (OpenRouter) | Storage: MOCK (JSON) | Scraper: MOCK
```

### Configuration Matrix

| Configuration | LLM | Storage | Speed | Cost | Use Case |
|--------------|-----|---------|-------|------|----------|
| All Mocks | Mock | Mock | ⚡⚡⚡ | Free | Fast unit tests |
| **Mixed (Recommended)** | **Real** | **Mock (JSON)** | **⚡⚡** | **API only** | **LLM integration** ⭐ |
| Real Storage | Mock | Real | ⚡⚡ | Free | Storage tests |
| All Real | Real | Real | ⚡ | Both | Full E2E |

## Test Scenarios

### 1. Full Workflow Test

Tests the complete submission → enrichment → approval → analysis flow:

```go
TestEndToEndWorkflow_SubmissionToEnrichment
```

**Covers:**
- Submission creation with status "received"
- Enrichment creation with status "pending"
- Enrichment progress updates
- Enrichment completion (status → "finished")
- Admin approval (status → "approved")

### 2. Enrichment Status Transitions

Tests all valid status transitions:

```go
TestEnrichmentStatusTransitions
```

**Verifies:**
- `pending` → `finished` → `approved`
- Invalid transitions are prevented

### 3. Data Persistence

Tests JSONB data storage and retrieval:

```go
TestEndToEndWorkflow_SubmissionToEnrichment > "Enrichment data persistence"
```

**Validates:**
- Nested JSON objects persist correctly
- Data can be retrieved and parsed

### 4. Concurrent Access

Tests database locking and concurrent updates:

```go
TestDatabaseOperations > "Concurrent access to enrichment"
```

**Ensures:**
- Multiple goroutines can safely read/write
- No race conditions

## 📚 Complete Documentation

- 📖 **[Mixed Testing Configuration Guide](../../docs/MIXED_TESTING_CONFIG.md)** - Complete guide for mixed configs ⭐
- 📖 **[Supabase JSON Mock Usage](../../docs/SUPABASE_MOCK_USAGE.md)** - Full JSON mock documentation
- 📖 **[Quick Start](./mocks/QUICKSTART.md)** - 60-second quick start
- 📖 **[Testing Guide](../../docs/TESTING.md)** - General testing best practices

## Mock Implementations

### Supabase JSON Mock (`mocks/supabase_json_mock.go`) ⭐ NEW

JSON-based file storage mock with thread-safe in-memory storage:

```go
storage := env.GetSupabaseStorageClient()
jsonMock := storage.(*mocks.SupabaseJSONMock)

// Clear pre-loaded files
jsonMock.Clear()

// Upload
url, err := jsonMock.Upload(ctx, "path/file.pdf", data, "application/pdf")

// Retrieve
content, exists := jsonMock.GetFileData("path/file.pdf")

// Get metadata
metadata, _ := jsonMock.GetFileMetadata("path/file.pdf")

// List files
files := jsonMock.ListFiles()

// Export state for debugging
jsonData, _ := jsonMock.ExportToJSON()
```

**Features:**
- ✅ In-memory storage (map-based)
- ✅ Thread-safe operations (sync.RWMutex)
- ✅ JSON configuration file support
- ✅ Pre-loaded test scenarios
- ✅ Content type validation
- ✅ File size validation
- ✅ State export for debugging
- ✅ Concurrent access safe

### LLM Mock (`mocks/llm_mock.go`)

Returns predefined JSON responses for enrichment and analysis:

```go
llmClient := mocks.NewMockLLMClient()
resp, err := llmClient.Call(ctx, &llm.Request{...})
// Returns mock enrichment profile
```

**Features:**
- Predefined enrichment profile with all required fields
- Predefined analysis data (PESTEL, Porter, SWOT)
- Configurable success/failure modes
- Call count tracking

### Supabase Mock (`mocks/supabase_mock.go`)

In-memory file storage with mock public URLs:

```go
storage := mocks.NewMockSupabaseStorage()
url, err := storage.Upload(ctx, "reports/test.pdf", data, "application/pdf")
// Returns: https://mock-project.supabase.co/storage/v1/object/public/reports/test.pdf
```

**Features:**
- In-memory file storage (map)
- Mock public URL generation
- File retrieval helper for assertions
- Thread-safe operations

### Scraper Mock (`mocks/scraper_mock.go`)

Returns predefined metadata for known domains:

```go
scraper := mocks.CreateMockScraperWithResponses()
meta, err := scraper.Scrape(ctx, "https://testcompany.com")
// Returns: MetaData with title, description, keywords, etc.
```

**Features:**
- Predefined responses for common test domains
- Default fallback response
- Custom response registration
- Call count tracking

## Test Helpers

### TestHelper Structure

The `TestHelper` provides utilities for all tests:

```go
helper := NewTestHelper(t)
defer helper.Close()
defer helper.Cleanup(t)
```

**Key Methods:**

| Method | Description |
|--------|-------------|
| `CreateTestSubmission()` | Create a test submission with realistic data |
| `CreateTestEnrichment()` | Create a test enrichment record |
| `WaitForEnrichmentStatus()` | Poll until enrichment reaches expected status |
| `AssertEnrichmentStatus()` | Assert enrichment has expected status |
| `GetEnrichmentBySubmissionID()` | Retrieve enrichment by submission ID |
| `CountRecords()` | Count records in any table |
| `Cleanup()` | Remove test data (created in last hour) |
| `TruncateAllTables()` | Remove ALL data (use with caution!) |

### Creating Test Data

```go
// Create a submission
sub := helper.CreateTestSubmission(t, ctx)

// Create enrichment
enr := helper.CreateTestEnrichment(t, ctx, sub.ID)

// Wait for status
enr = helper.WaitForEnrichmentStatus(t, ctx, sub.ID, enrichment.StatusFinished, 30*time.Second)
```

## Best Practices

### 1. Always Clean Up

```go
func TestSomething(t *testing.T) {
    helper := NewTestHelper(t)
    defer helper.Close()
    defer helper.Cleanup(t)  // Removes test data

    // Your test code...
}
```

### 2. Use Assertions

```go
// Use require for critical checks (stops test on failure)
require.NoError(t, err, "Failed to create submission")
require.NotNil(t, sub)

// Use assert for non-critical checks (continues test)
assert.Equal(t, "received", sub.Status)
```

### 3. Wait for Async Operations

```go
// Don't assume immediate completion
enr := helper.WaitForEnrichmentStatus(t, ctx, sub.ID, enrichment.StatusFinished, 30*time.Second)
```

### 4. Test Both Modes

Write tests that work in both real and mock mode:

```go
func TestSomething(t *testing.T) {
    helper := NewTestHelper(t)

    // This automatically uses real or mock services
    llmClient := helper.TestEnv.GetLLMClient()

    // Test works regardless of mode
}
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run Integration Tests (Mock Mode)
        env:
          DATABASE_URL: postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable
        run: |
          cd backend_v3/integration_tests
          go test -v
```

**Note**: In CI/CD, tests automatically run in **mock mode** since no OpenRouter/Supabase credentials are provided.

## Troubleshooting

### Tests Fail with "database not found"

Ensure `DATABASE_URL` is set in `.env`:

```bash
DATABASE_URL=postgres://user:password@localhost:5432/dbname
```

### Tests Use Real Services When Mocks Expected

Check that credentials are NOT in `.env`. Temporarily rename:

```bash
mv ../.env ../.env.backup
```

### Mock Responses Don't Match Expected Schema

Update mock responses in `mocks/llm_mock.go`:

```go
func getDefaultEnrichmentResponse() string {
    // Modify this JSON to match your schema
}
```

### Concurrent Test Failures

Increase wait timeout in helpers:

```go
enr := helper.WaitForEnrichmentStatus(t, ctx, sub.ID, status, 60*time.Second)
```

## Future Enhancements

- [ ] Add analysis workflow tests (pending → completed → approved → sent)
- [ ] Add report generation tests (PDF upload to Supabase)
- [ ] Add error scenario tests (LLM failures, network timeouts)
- [ ] Add Asynq job queue integration tests
- [ ] Add versioning tests for analysis
- [ ] Add admin edit operation tests
- [ ] Add concurrent enrichment/analysis tests
- [ ] Add performance benchmarks for full workflow

## Contributing

When adding new tests:

1. Use the `TestHelper` for database operations
2. Support both real and mock modes
3. Clean up test data in `defer` statements
4. Add documentation to this README
5. Update mock responses if schema changes

## License

Internal use only - Imensiah Business Intelligence Platform
