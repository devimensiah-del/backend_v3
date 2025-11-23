# Test Utilities for Backend V3

Comprehensive test utilities for the Backend V3 Go project with mock implementations, fixtures, and helpers.

## 📁 Files Overview

| File | Purpose | Key Features |
|------|---------|--------------|
| `mocks.go` | Mock implementations | MockLLMClient, MockStorageClient, MockPDFGenerator |
| `fixtures.go` | Test data generators | NewTestSubmission(), NewTestEnrichment(), NewTestAnalysis() |
| `fixture_responses.go` | LLM response fixtures | JSON responses for all 11 analysis frameworks |
| `db.go` | In-memory database | SetupTestDB(), LoadSchema(), SQLite with migrations |
| `assertions.go` | Custom assertions | AssertAnalysisComplete(), AssertEnrichmentHasData() |
| `asynq.go` | Asynq testing helpers | MockAsynqClient, EnqueueAndWait(), task verification |

## 🚀 Quick Start

### 1. Basic Test Setup

```go
package mypackage_test

import (
	"testing"
	"backend_v3/tests/testutils"
)

func TestMyFunction(t *testing.T) {
	// Setup in-memory database
	db := testutils.SetupTestDB(t)
	defer testutils.TeardownTestDB(t, db)

	// Create test data
	submission := testutils.NewTestSubmission()

	// Your test logic here
}
```

### 2. Using Mock LLM Client

```go
func TestEnrichmentWithMockLLM(t *testing.T) {
	// Create mock LLM client with default responses
	mockLLM := testutils.NewMockLLMClient()

	// Configure mock to return on method call
	mockLLM.On("GenerateStructuredWithOptions",
		mock.Anything, // ctx
		mock.Anything, // opts
		mock.MatchedBy(func(prompt string) bool {
			return strings.Contains(prompt, "enrichment")
		}),
		mock.Anything, // data
		mock.Anything, // targetSchema
	).Return(nil)

	// Use mock in service
	enrichmentSvc := enrichment.NewService(db, mockLLM, logger)
	result, err := enrichmentSvc.EnrichSubmission(ctx, submissionID)

	require.NoError(t, err)
	testutils.AssertEnrichmentHasData(t, result)
}
```

### 3. Using Fixtures

```go
func TestCompleteWorkflow(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer testutils.TeardownTestDB(t, db)

	// Create test submission
	submission := testutils.NewTestSubmission()
	submissionID := db.InsertTestSubmission(t)

	// Create test enrichment
	enrichment := testutils.NewTestEnrichment(submission.ID)
	enrichmentID := db.InsertTestEnrichment(t, submissionID)

	// Create test analysis
	analysis := testutils.NewTestAnalysis(submissionID, enrichmentID)

	// Verify analysis completeness
	testutils.AssertAnalysisComplete(t, analysis)
}
```

### 4. Testing Asynq Jobs

```go
func TestJobEnqueuing(t *testing.T) {
	// Create mock asynq client
	mockClient := testutils.NewTestAsynqClient()

	// Enqueue enrichment job
	payload := testutils.CreateEnrichmentPayload(submissionID)
	task := asynq.NewTask("enrichment_job", payload)

	_, err := mockClient.Enqueue(task)
	require.NoError(t, err)

	// Verify job was enqueued
	testutils.AssertTaskEnqueued(t, mockClient, "enrichment_job")
}
```

### 5. Complete Integration Test

```go
func TestFullEnrichmentAnalysisFlow(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer testutils.TeardownTestDB(t, db)

	mockLLM := testutils.NewMockLLMClient()
	mockStorage := testutils.NewMockStorageClient()
	mockPDF := testutils.NewMockPDFGenerator()
	mockAsynq := testutils.NewTestAsynqClient()

	// Configure mocks
	mockLLM.On("GenerateStructuredWithOptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockStorage.On("Upload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("https://mock-url.com/report.pdf", nil)
	mockPDF.On("Convert", mock.Anything, mock.Anything).Return([]byte("%PDF-1.4"), nil)

	// Create services
	submissionSvc := submission.NewService(db.DB, mockAsynq, logger)
	enrichmentSvc := enrichment.NewService(db.DB, mockLLM, logger)
	analysisSvc := analysis.NewService(db.DB, mockLLM, logger)

	// Test workflow
	sub := testutils.NewTestSubmission()

	// 1. Create submission
	created, err := submissionSvc.Create(ctx, sub)
	require.NoError(t, err)
	testutils.AssertValidUUID(t, created.ID.String(), "submission_id")

	// 2. Enrich submission
	enrichment, err := enrichmentSvc.EnrichSubmission(ctx, created.ID)
	require.NoError(t, err)
	testutils.AssertEnrichmentHasData(t, enrichment)

	// 3. Run analysis
	analysis, err := analysisSvc.RunAnalysis(ctx, created.ID.String(), enrichment.ID.String(), enrichment.EnrichedData)
	require.NoError(t, err)
	testutils.AssertAnalysisComplete(t, analysis)

	// Verify mocks were called
	mockLLM.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}
```

## 📋 Available Fixtures

### Submission Fixtures
- `NewTestSubmission()` - Creates valid submission with all fields
- Sample: Acme Tech Solutions (Cloud Infrastructure SaaS)
- Includes: CNPJ, website, industry, contact info, business challenge

### Enrichment Fixtures
- `NewTestEnrichment(submissionID)` - Creates enrichment with UnifiedProfile
- Includes: ProfileOverview, MarketPosition, Financials, CompetitiveLandscape, StrategicAssessment, MacroContext

### Analysis Fixtures
- `NewTestAnalysis(submissionID, enrichmentID)` - Creates complete analysis
- All 11 frameworks populated:
  1. PESTEL
  2. Porter's 7 Forces
  3. SWOT (with confidence & source)
  4. TAM/SAM/SOM
  5. Blue Ocean
  6. OKRs (Quarterly)
  7. Balanced Scorecard
  8. Benchmarking
  9. Growth Hacking (LEAP + SCALE Loops)
  10. Scenario Analysis
  11. Decision Matrix
  12. Synthesis

## 🧪 Custom Assertions

### `AssertSubmissionEqual(t, expected, actual)`
Deep comparison of two submissions (all 18 fields)

### `AssertEnrichmentHasData(t, enrichment)`
Validates:
- UnifiedProfile structure
- ProfileOverview (legal name, website, foundation year, headquarters)
- MarketPosition (sector, target audience, value proposition)
- Financials (employees, revenue, business model)
- CompetitiveLandscape (competitors, market share)
- StrategicAssessment (digital maturity, strengths, weaknesses)
- MacroContext (economic indicators, industry trends, regulatory landscape)

### `AssertAnalysisComplete(t, analysis)`
Validates all 11 frameworks are populated with required fields:
- PESTEL: 6 factors + summary
- Porter: 7 forces + intensities + strategic implications
- SWOT: Items with confidence & source
- TAM/SAM/SOM: Market sizing + assumptions
- Blue Ocean: ERRC grid + value curve
- OKRs: 3 quarters with objectives, key results, investment, timeline
- BSC: 4 perspectives
- Benchmarking: Competitors, gaps, best practices
- Growth Hacking: LEAP + SCALE loops with metrics
- Scenarios: Optimistic/Realist/Pessimistic with probabilities summing to 100
- Decision Matrix: Alternatives, criteria, priority recommendations
- Synthesis: Executive summary, central challenge, 4 main findings

### `AssertJobEnqueued(t, inspector, taskType, submissionID)`
Verifies asynq job was queued with correct payload

## 🗄️ In-Memory Database

### Features
- SQLite in-memory (`:memory:`)
- Auto-loads all migrations from `migrations/` directory
- Converts PostgreSQL syntax to SQLite:
  - UUID → TEXT
  - TIMESTAMP WITH TIME ZONE → DATETIME
  - JSONB → JSON
  - TEXT[] → TEXT
  - DECIMAL → REAL
- Supports foreign keys
- Fast teardown (automatic on close)

### Usage
```go
db := testutils.SetupTestDB(t)
defer testutils.TeardownTestDB(t, db)

// Insert test data
submissionID := db.InsertTestSubmission(t)
enrichmentID := db.InsertTestEnrichment(t, submissionID)
```

## 🎭 Mock Implementations

### MockLLMClient
- Implements `llm.Client` interface
- Auto-detects framework from prompt keywords
- Returns predefined JSON responses
- Configurable via `SetResponse(key, jsonResponse)`
- Tracks call count

### MockStorageClient
- Implements `infrastructure.StorageClient`
- Simulates Supabase file uploads
- Returns mock public URLs
- Tracks uploaded files in memory
- `GetUploadedFile(path)` for verification

### MockPDFGenerator
- Implements `infrastructure.PDFGenerator`
- Simulates Gotenberg HTML→PDF conversion
- Returns mock PDF bytes
- Tracks generated PDFs
- `GetGeneratedPDF(key)` for verification

## 📦 Dependencies

```go
import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/mock"
	"github.com/mattn/go-sqlite3"
	"github.com/hibiken/asynq"
	"github.com/google/uuid"
)
```

## 🎯 Testing Best Practices

### 1. Use t.Helper()
All helper functions call `t.Helper()` to report errors at correct line numbers

### 2. Use require vs assert
- `require.*` - Fails immediately (for critical checks)
- `assert.*` - Continues execution (for multiple validations)

### 3. Parallel Tests
```go
func TestParallel(t *testing.T) {
	t.Parallel() // Run in parallel

	db := testutils.SetupTestDB(t)
	defer testutils.TeardownTestDB(t, db)

	// Each parallel test gets its own in-memory DB
}
```

### 4. Subtests
```go
func TestSubmissionValidation(t *testing.T) {
	tests := []struct {
		name    string
		submission *submission.Submission
		wantErr bool
	}{
		{"valid submission", testutils.NewTestSubmission(), false},
		{"missing company name", &submission.Submission{CompanyName: ""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.submission.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
```

### 5. Cleanup
Always defer cleanup:
```go
db := testutils.SetupTestDB(t)
defer testutils.TeardownTestDB(t, db)

mockClient := testutils.NewTestAsynqClient()
defer mockClient.Close()
```

## 🐛 Troubleshooting

### "migrations directory not found"
**Solution**: Test files must be in a location where migrations/ is accessible (2-4 directories up)

### "Failed to parse LLM JSON response"
**Solution**: Check that mock response matches expected schema. Use `fixture_responses.go` examples.

### "SWOT items missing confidence/source"
**Solution**: Ensure SWOT fixtures use `analysis.SWOTItem` struct with all fields

### "Scenario probabilities don't sum to 100"
**Solution**: Check Optimistic (20) + Realist (60) + Pessimistic (20) = 100

## 📝 Example Test File

See `tests/testutils/example_test.go` for complete working examples.

## 🔗 Related Documentation

- [Domain Models](../../domain/)
- [Migration Files](../../migrations/)
- [Integration Tests](../../integration_tests/)
- [API Handlers](../../api/)

## 📞 Support

For questions or issues with test utilities, check:
1. This README
2. Example tests in `example_test.go`
3. Existing integration tests in `integration_tests/`
