# Quick Start Guide - Test Utilities

## 🚀 5-Minute Setup

### 1. Copy-Paste Template

```go
package mypackage_test

import (
    "context"
    "testing"
    "backend_v3/tests/testutils"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMyFeature(t *testing.T) {
    // Setup database
    db := testutils.SetupTestDB(t)
    defer testutils.TeardownTestDB(t, db)

    // Create mocks
    mockLLM := testutils.NewMockLLMClient()
    mockStorage := testutils.NewMockStorageClient()

    // Create test data
    submission := testutils.NewTestSubmission()

    // Your test logic here
    // ...

    // Assertions
    testutils.AssertValidUUID(t, submission.ID.String(), "submission_id")
}
```

### 2. Run Tests

```bash
# Run all testutils tests
go test ./tests/testutils/...

# Run with coverage
go test -cover ./tests/testutils/...

# Run specific test
go test ./tests/testutils/ -run TestSubmissionFixture
```

## 📋 Common Patterns

### Pattern 1: Test Submission Creation

```go
func TestSubmissionValidation(t *testing.T) {
    submission := testutils.NewTestSubmission()

    err := submission.Validate()

    require.NoError(t, err)
    assert.Equal(t, "Acme Tech Solutions", submission.CompanyName)
}
```

### Pattern 2: Test Enrichment with Mock LLM

```go
func TestEnrichmentService(t *testing.T) {
    db := testutils.SetupTestDB(t)
    defer testutils.TeardownTestDB(t, db)

    mockLLM := testutils.NewMockLLMClient()
    mockLLM.On("GenerateStructuredWithOptions",
        mock.Anything, mock.Anything, mock.Anything,
        mock.Anything, mock.Anything,
    ).Return(nil)

    svc := enrichment.NewService(db.DB, mockLLM, logger)

    submissionID := db.InsertTestSubmission(t)
    result, err := svc.EnrichSubmission(context.Background(), submissionID)

    require.NoError(t, err)
    testutils.AssertEnrichmentHasData(t, result)
}
```

### Pattern 3: Test Analysis Completeness

```go
func TestAnalysisGeneration(t *testing.T) {
    submission := testutils.NewTestSubmission()
    enrichment := testutils.NewTestEnrichment(submission.ID)
    analysis := testutils.NewTestAnalysis(
        submission.ID.String(),
        enrichment.ID.String(),
    )

    testutils.AssertAnalysisComplete(t, analysis)
}
```

### Pattern 4: Test Asynq Job Enqueuing

```go
func TestJobEnqueuing(t *testing.T) {
    mockClient := testutils.NewTestAsynqClient()

    payload := testutils.CreateEnrichmentPayload(submissionID)
    task := testutils.NewAsynqTask("enrichment_job", payload)

    _, err := mockClient.Enqueue(task)

    require.NoError(t, err)
    testutils.AssertTaskEnqueued(t, mockClient, "enrichment_job")
}
```

### Pattern 5: Complete Workflow Test

```go
func TestCompleteWorkflow(t *testing.T) {
    // Setup
    db := testutils.SetupTestDB(t)
    defer testutils.TeardownTestDB(t, db)

    mockLLM := testutils.NewMockLLMClient()
    mockStorage := testutils.NewMockStorageClient()
    mockPDF := testutils.NewMockPDFGenerator()

    // Configure all mocks
    mockLLM.On("GenerateStructuredWithOptions",
        mock.Anything, mock.Anything, mock.Anything,
        mock.Anything, mock.Anything).Return(nil)
    mockStorage.On("Upload",
        mock.Anything, mock.Anything, mock.Anything,
        mock.Anything).Return("https://mock.com/report.pdf", nil)
    mockPDF.On("Convert",
        mock.Anything, mock.Anything).Return([]byte("%PDF-1.4"), nil)

    // Test workflow
    submissionID := db.InsertTestSubmission(t)
    enrichmentID := db.InsertTestEnrichment(t, submissionID)
    analysis := testutils.NewTestAnalysis(submissionID, enrichmentID)

    // Verify
    testutils.AssertAnalysisComplete(t, analysis)
    mockLLM.AssertExpectations(t)
    mockStorage.AssertExpectations(t)
    mockPDF.AssertExpectations(t)
}
```

## 🎯 Cheat Sheet

### Available Fixtures

| Function | Returns | Description |
|----------|---------|-------------|
| `NewTestSubmission()` | `*submission.Submission` | Acme Tech (Cloud SaaS) |
| `NewTestEnrichment(subID)` | `*enrichment.Enrichment` | UnifiedProfile + Macro |
| `NewTestAnalysis(subID, enrID)` | `*analysis.Analysis` | All 11 frameworks |

### Available Mocks

| Function | Returns | Interface |
|----------|---------|-----------|
| `NewMockLLMClient()` | `*MockLLMClient` | `llm.Client` |
| `NewMockStorageClient()` | `*MockStorageClient` | `StorageClient` |
| `NewMockPDFGenerator()` | `*MockPDFGenerator` | `PDFGenerator` |
| `NewTestAsynqClient()` | `*MockAsynqClient` | `asynq.Client` |

### Database Helpers

| Function | Description |
|----------|-------------|
| `SetupTestDB(t)` | Create in-memory SQLite with schema |
| `TeardownTestDB(t, db)` | Close database |
| `InsertTestSubmission(t)` | Insert submission, return ID |
| `InsertTestEnrichment(t, subID)` | Insert enrichment, return ID |

### Assertions

| Function | Validates |
|----------|-----------|
| `AssertSubmissionEqual(t, exp, act)` | Deep submission comparison |
| `AssertEnrichmentHasData(t, enr)` | UnifiedProfile structure |
| `AssertAnalysisComplete(t, ana)` | All 11 frameworks populated |
| `AssertJobEnqueued(t, insp, type, id)` | Asynq job queued |
| `AssertValidUUID(t, uuid, name)` | UUID format |
| `AssertTimeNotZero(t, time, name)` | Time value |
| `AssertJSONValid(t, json, name)` | JSON syntax |

### Asynq Helpers

| Function | Description |
|----------|-------------|
| `CreateEnrichmentPayload(id)` | Build enrichment JSON |
| `CreateAnalysisPayload(subID, enrID)` | Build analysis JSON |
| `ParseEnrichmentPayload(t, payload)` | Extract submission ID |
| `ParseAnalysisPayload(t, payload)` | Extract both IDs |
| `NewAsynqTask(type, payload)` | Create asynq task |
| `AssertTaskEnqueued(t, client, type)` | Verify job queued |

## 🔧 Configuration

### Custom LLM Response

```go
mockLLM := testutils.NewMockLLMClient()

// Override default PESTEL response
customPESTEL := `{"political": ["Custom factor"], ...}`
mockLLM.SetResponse("pestel", customPESTEL)
```

### Custom Mock Storage URL

```go
mockStorage := testutils.NewMockStorageClient()
mockStorage.On("Upload", ...).Return("https://custom-url.com/file.pdf", nil)
```

## 📦 File Structure

```
tests/testutils/
├── mocks.go              # Mock implementations (LLM, Storage, PDF)
├── fixtures.go           # Test data generators
├── fixture_responses.go  # LLM JSON responses
├── db.go                 # In-memory database
├── assertions.go         # Custom assertions
├── asynq.go              # Asynq helpers
├── example_test.go       # Working examples
├── README.md             # Full documentation
├── QUICK_START.md        # This file
└── SUMMARY.md            # Feature summary
```

## 🐛 Common Errors

### Error: "migrations directory not found"
**Fix**: Run tests from project root or ensure migrations/ is accessible

### Error: "Failed to parse LLM JSON"
**Fix**: Use fixture responses from `fixture_responses.go`

### Error: "Mock expectations not met"
**Fix**: Call `mockLLM.AssertExpectations(t)` at end of test

### Error: "Database locked"
**Fix**: Ensure you defer `TeardownTestDB(t, db)`

## 📚 Learn More

- **Full Documentation**: See `README.md`
- **Working Examples**: See `example_test.go`
- **Feature Summary**: See `SUMMARY.md`

## ⚡ Quick Commands

```bash
# Run example tests
go test ./tests/testutils/ -v

# Test with coverage
go test -cover ./tests/testutils/

# Test specific pattern
go test ./tests/testutils/ -run TestMock

# Verbose output
go test ./tests/testutils/ -v -run TestComplete
```

---

**Ready to test!** Start with `example_test.go` and modify for your use case.
