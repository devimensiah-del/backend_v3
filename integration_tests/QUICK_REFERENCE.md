# Integration Tests - Quick Reference Card

## 🚀 Running Tests

```bash
cd backend_v3/integration_tests

# Quick test run
./run_tests.sh

# Verbose output
./run_tests.sh -v

# Benchmarks
./run_tests.sh -bench

# Specific test
go test -v -run TestEndToEndWorkflow
```

## 🎯 Writing a New Test

```go
func TestYourFeature(t *testing.T) {
    helper := NewTestHelper(t)
    defer helper.Close()
    defer helper.Cleanup(t)

    ctx := context.Background()

    // Create test data
    sub := helper.CreateTestSubmission(t, ctx)
    enr := helper.CreateTestEnrichment(t, ctx, sub.ID)

    // Test your feature...

    // Assert results
    helper.AssertEnrichmentStatus(t, ctx, sub.ID, enrichment.StatusFinished)
}
```

## 🔧 Using Mocks

### LLM Mock
```go
llmClient := helper.TestEnv.GetLLMClient()
// Automatically uses mock if no API key
```

### Supabase Mock (In-Memory)
```go
storage := mocks.NewMockSupabaseStorage()
url, _ := storage.Upload(ctx, "test.pdf", data, "application/pdf")
```

### Supabase Mock (JSON-Based) ⭐
```go
mock := mocks.NewSupabaseJSONMockDefault()

// Use predefined scenario
scenario, _ := mock.GetTestScenario("successful_upload")
url, _ := mock.Upload(ctx, scenario.Input["path"], data, "application/pdf")
```

## 📝 Adding Test Scenarios

Edit `mocks/supabase_mock_data.json`:

```json
{
  "test_scenarios": {
    "my_new_scenario": {
      "description": "What this tests",
      "input": {
        "bucket": "reports",
        "path": "test.pdf",
        "content_type": "application/pdf"
      },
      "expected_output": {
        "status_code": 201
      }
    }
  }
}
```

## 🛠️ Helper Methods

| Method | Purpose |
|--------|---------|
| `CreateTestSubmission(t, ctx)` | Create test submission |
| `CreateTestEnrichment(t, ctx, subID)` | Create test enrichment |
| `WaitForEnrichmentStatus(t, ctx, subID, status, timeout)` | Wait for status |
| `AssertEnrichmentStatus(t, ctx, subID, status)` | Assert status |
| `Cleanup(t)` | Remove test data |
| `CountRecords(t, table)` | Count table records |

## 🌍 Environment

### With Real Services (.env file)
```bash
DATABASE_URL=postgres://...
OPENROUTER_API_KEY=sk-or-xxx
SUPABASE_URL=https://xxx.supabase.co
```

### With Mocks (No .env needed)
Tests automatically use mocks!

## 📊 Test Examples

### Basic Workflow
```go
sub := helper.CreateTestSubmission(t, ctx)
enr := helper.CreateTestEnrichment(t, ctx, sub.ID)
enr.Finish()
helper.EnrichmentRepo.UpdateSystem(ctx, enr)
helper.AssertEnrichmentStatus(t, ctx, sub.ID, enrichment.StatusFinished)
```

### JSON Test Scenario
```go
mock := mocks.NewSupabaseJSONMockDefault()
scenario, _ := mock.GetTestScenario("successful_upload")
url, err := mock.Upload(ctx, scenario.Input["path"], data, "application/pdf")
assert.NoError(t, err)
```

### Error Testing
```go
mock := mocks.NewSupabaseJSONMockDefault()
_, err := mock.Upload(ctx, "test.exe", data, "application/x-msdownload")
assert.Error(t, err)
assert.Contains(t, err.Error(), "not allowed")
```

## 🔍 Debugging

### View Test Output
```bash
go test -v -run TestYourTest
```

### Check Database
```go
count := helper.CountRecords(t, "submissions")
fmt.Printf("Submissions: %d\n", count)
```

### Export Mock State
```go
mock := mocks.NewSupabaseJSONMockDefault()
jsonData, _ := mock.ExportToJSON()
fmt.Println(string(jsonData))
```

## ⚙️ Configuration

### Test Database
Set in `.env`:
```bash
DATABASE_URL=postgres://user:pass@localhost:5432/test_db
```

### Mock Behavior
Edit `mocks/supabase_mock_data.json`:
```json
{
  "config": {
    "max_file_size": 52428800,
    "allowed_content_types": ["application/pdf"]
  }
}
```

## 🎯 Common Patterns

### Test Full Workflow
```go
// 1. Create submission
sub := helper.CreateTestSubmission(t, ctx)

// 2. Create enrichment
enr := helper.CreateTestEnrichment(t, ctx, sub.ID)

// 3. Process enrichment
enr.UpdateProgress("Processing...", 50)
helper.EnrichmentRepo.UpdateSystem(ctx, enr)

// 4. Complete enrichment
enr.Finish()
helper.EnrichmentRepo.UpdateSystem(ctx, enr)

// 5. Approve
enr.Status = enrichment.StatusApproved
helper.EnrichmentRepo.UpdateSystem(ctx, enr)

// 6. Assert
helper.AssertEnrichmentStatus(t, ctx, sub.ID, enrichment.StatusApproved)
```

### Test Error Scenario
```go
// Create invalid data
invalidData := make([]byte, mock.MaxFileSize+1)

// Attempt operation
_, err := mock.Upload(ctx, "too-large.pdf", invalidData, "application/pdf")

// Assert error
assert.Error(t, err)
assert.Contains(t, err.Error(), "exceeds maximum")
```

## 📖 Documentation

- **README.md** - Comprehensive guide
- **mocks/README.md** - Mock documentation
- **INTEGRATION_TESTING_GUIDE.md** - Architecture guide
- **SUMMARY.md** - What was created
- **QUICK_REFERENCE.md** - This file

## 🆘 Troubleshooting

### Tests fail: "database not found"
→ Set `DATABASE_URL` in `.env`

### Tests use real API when mocks expected
→ Remove/rename `.env` file

### Mock responses don't match schema
→ Edit `mocks/llm_mock.go` → `getDefaultEnrichmentResponse()`

### JSON mock file not found
→ Check path: `mocks/supabase_mock_data.json`

## 📦 Files Structure

```
integration_tests/
├── test_env.go              # Auto-detect real/mock
├── helpers.go               # Test utilities
├── workflow_test.go         # Tests
├── run_tests.sh             # Runner
└── mocks/
    ├── llm_mock.go          # LLM mock
    ├── supabase_mock.go     # In-memory storage
    ├── supabase_json_mock.go ⭐ JSON storage
    ├── supabase_mock_data.json ⭐ Test data
    └── scraper_mock.go      # Scraper mock
```

## ✅ Checklist for New Tests

- [ ] Create test function: `TestYourFeature(t *testing.T)`
- [ ] Initialize helper: `helper := NewTestHelper(t)`
- [ ] Add cleanup: `defer helper.Cleanup(t)`
- [ ] Create test data: `CreateTestSubmission()`, etc.
- [ ] Perform operations
- [ ] Assert results: `AssertEnrichmentStatus()`, etc.
- [ ] Run test: `go test -v -run TestYourFeature`

## 🎓 Best Practices

✅ **Always cleanup** - Use `defer helper.Cleanup(t)`
✅ **Use assertions** - `require` for critical, `assert` for optional
✅ **Use JSON mocks** - For complex scenarios
✅ **Document scenarios** - Add descriptions to JSON
✅ **Test errors** - Don't just test happy paths
✅ **Parallel safe** - Tests can run concurrently

---

**Need more help?** Check `README.md` or `SUMMARY.md`
