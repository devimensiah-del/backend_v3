# Mock Clients for Integration Testing

## Overview

This directory contains reusable mock implementations for external services used in integration tests.

## Available Mocks

### 1. LLM Mock (`llm_mock.go`)

Mocks OpenRouter/LLM API calls with predefined responses.

**Features:**
- Predefined enrichment profile responses
- Predefined analysis framework responses
- Configurable success/failure modes
- Call count tracking

**Usage:**
```go
llmClient := mocks.NewMockLLMClient()
resp, err := llmClient.Call(ctx, &llm.Request{...})
```

### 2. Supabase Storage Mock - In-Memory (`supabase_mock.go`)

Simple in-memory mock for Supabase storage operations.

**Features:**
- In-memory file storage (map-based)
- Mock public URL generation
- Thread-safe operations
- File listing and retrieval

**Usage:**
```go
storage := mocks.NewMockSupabaseStorage()
url, err := storage.Upload(ctx, "path/to/file.pdf", data, "application/pdf")

// Retrieve file
data, exists := storage.GetFile("path/to/file.pdf")
```

### 3. Supabase Storage Mock - JSON-Based ⭐ NEW (`supabase_json_mock.go`)

**Advanced mock that loads test data from JSON files.**

**Why JSON-based mocking is smart:**

✅ **Separation of concerns** - Test data separate from test code
✅ **Data-driven testing** - Multiple scenarios from single JSON file
✅ **Version controlled** - Test data versioned alongside code
✅ **Easy updates** - Change test scenarios without recompiling
✅ **Reusability** - Share mock data across test suites
✅ **Documentation** - JSON serves as test scenario documentation

**Features:**
- Loads predefined files from `supabase_mock_data.json`
- Configurable from JSON (max file size, allowed content types, etc.)
- Test scenario definitions in JSON
- File metadata tracking
- All features of in-memory mock PLUS persistent test scenarios

**JSON Structure (`supabase_mock_data.json`):**

```json
{
  "storage": {
    "buckets": {
      "reports": {
        "public": true,
        "files": {
          "test-submission-123/report-v1.pdf": {
            "size": 1024567,
            "content_type": "application/pdf",
            "created_at": "2025-01-15T10:30:00Z",
            "public_url": "https://..."
          }
        }
      }
    }
  },
  "test_scenarios": {
    "successful_upload": {
      "description": "Successful PDF upload",
      "input": {...},
      "expected_output": {...}
    }
  },
  "config": {
    "project_url": "https://mock-project.supabase.co",
    "max_file_size": 52428800,
    "allowed_content_types": ["application/pdf"]
  }
}
```

**Usage:**

```go
// Load from JSON file
mock, err := mocks.NewSupabaseJSONMock("mocks/supabase_mock_data.json")
if err != nil {
    t.Fatal(err)
}

// Or use default (tries to load, fallbacks to in-memory)
mock := mocks.NewSupabaseJSONMockDefault()

// Upload file
url, err := mock.Upload(ctx, "submission-123/report.pdf", data, "application/pdf")

// Retrieve file with metadata
file, exists := mock.GetFile("submission-123/report.pdf")
if exists {
    fmt.Println("Size:", file.Size)
    fmt.Println("Created:", file.CreatedAt)
    fmt.Println("Public URL:", file.PublicURL)
}

// Get only file data
data, exists := mock.GetFileData("submission-123/report.pdf")

// Get only metadata
metadata, exists := mock.GetFileMetadata("submission-123/report.pdf")

// Use test scenarios
scenario, exists := mock.GetTestScenario("successful_upload")
if exists {
    fmt.Println("Testing:", scenario.Description)
    // Use scenario.Input and scenario.ExpectedOutput for test
}

// Delete file
err := mock.DeleteFile("submission-123/report.pdf")

// Export current state to JSON (for debugging)
jsonData, _ := mock.ExportToJSON()
fmt.Println(string(jsonData))
```

**Test Scenarios:**

The JSON file defines reusable test scenarios:

```go
func TestUploadScenarios(t *testing.T) {
    mock := mocks.NewSupabaseJSONMockDefault()

    // Test successful upload scenario
    scenario, _ := mock.GetTestScenario("successful_upload")
    input := scenario.Input
    expected := scenario.ExpectedOutput

    url, err := mock.Upload(
        ctx,
        input["path"].(string),
        []byte("test data"),
        input["content_type"].(string),
    )

    assert.NoError(t, err)
    assert.Equal(t, expected["status_code"], 201)
}
```

### 4. Scraper Mock (`scraper_mock.go`)

Mocks web scraping operations with predefined metadata.

**Features:**
- Predefined responses for known domains
- Default fallback response
- Custom response registration
- Call count tracking

**Usage:**
```go
scraper := mocks.CreateMockScraperWithResponses()
meta, err := scraper.Scrape(ctx, "https://testcompany.com")

// Add custom response
scraper.AddResponse("example.com", scraper.MetaData{
    Title: "Example",
    Description: "Test company",
})
```

## Comparison: In-Memory vs JSON-Based Mocking

| Feature | In-Memory Mock | JSON-Based Mock |
|---------|---------------|-----------------|
| Setup complexity | Simple | Moderate |
| Test data location | In code | JSON file |
| Easy to update | No (requires recompile) | Yes (edit JSON) |
| Version controlled | Code only | JSON + Code |
| Reusability | High | Very High |
| Test scenarios | Code-defined | JSON-defined |
| Best for | Quick tests | Complex scenarios |

## When to Use Each Mock

### Use In-Memory Mock (`supabase_mock.go`) when:
- ✅ Quick, simple tests
- ✅ No need for predefined files
- ✅ Test data can be created inline
- ✅ No test scenario reuse needed

### Use JSON-Based Mock (`supabase_json_mock.go`) when:
- ✅ Testing multiple scenarios
- ✅ Need predefined file states
- ✅ Sharing test data across tests
- ✅ Versioning test scenarios
- ✅ Complex file metadata testing
- ✅ Documenting expected behavior

## Creating Custom Test Scenarios

Add new scenarios to `supabase_mock_data.json`:

```json
{
  "test_scenarios": {
    "your_new_scenario": {
      "description": "Describe what this scenario tests",
      "input": {
        "bucket": "reports",
        "path": "test-file.pdf",
        "content_type": "application/pdf"
      },
      "expected_output": {
        "public_url": "https://...",
        "status_code": 201
      }
    }
  }
}
```

Then use in tests:

```go
scenario, _ := mock.GetTestScenario("your_new_scenario")
// Use scenario.Input and scenario.ExpectedOutput
```

## Running Mock Tests

Test all mocks:

```bash
cd backend_v3/integration_tests/mocks
go test -v
```

Test specific mock:

```bash
go test -v -run TestSupabaseJSONMock
```

Run benchmarks:

```bash
go test -bench=. -benchmem
```

## Extending Mocks

### Adding New Mock Response

**For LLM Mock:**

Edit `llm_mock.go` and update `getDefaultEnrichmentResponse()` or `getDefaultAnalysisResponse()`.

**For Supabase Mock (JSON-based):**

Edit `supabase_mock_data.json` and add files to the `storage.buckets.reports.files` section.

**For Scraper Mock:**

```go
scraper := mocks.CreateMockScraperWithResponses()
scraper.AddResponse("newdomain.com", scraper.MetaData{
    Title: "New Domain",
    Description: "Custom response",
})
```

### Adding New Test Scenario

Edit `supabase_mock_data.json` and add to `test_scenarios`:

```json
{
  "test_scenarios": {
    "new_scenario_name": {
      "description": "What this tests",
      "input": {...},
      "expected_output": {...}
    }
  }
}
```

## Best Practices

1. **Keep JSON file in version control** - Test scenarios are part of the codebase
2. **Document scenarios** - Use descriptive names and descriptions
3. **Update both mocks when schema changes** - Keep JSON and code in sync
4. **Use realistic data** - Mimic production data structures
5. **Test both success and failure scenarios** - Add error cases to JSON
6. **Clean up between tests** - Use `mock.Clear()` in test cleanup

## Future Enhancements

Potential improvements:

- [ ] Add authentication mock scenarios
- [ ] Add database mock with JSON fixtures
- [ ] Add API response mocking
- [ ] Generate mock data from OpenAPI specs
- [ ] Validate JSON schema on load
- [ ] Support multiple JSON files per service
- [ ] Add randomized data generation

## Summary

The mock clients provide:

✅ **Realistic behavior** without external dependencies
✅ **Fast execution** (no network calls)
✅ **Deterministic tests** (same input = same output)
✅ **Flexible testing** (easy to modify scenarios)
✅ **Version control friendly** (JSON + code)
✅ **CI/CD ready** (no credentials needed)

Choose JSON-based mocking for complex, reusable test scenarios. Choose in-memory mocking for quick, simple tests.
