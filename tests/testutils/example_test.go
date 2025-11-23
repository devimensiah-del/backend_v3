package testutils_test

import (
	"context"
	"testing"

	"backend_v3/tests/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// =================================================================
// Example 1: Basic Fixture Usage
// =================================================================

func TestSubmissionFixture(t *testing.T) {
	// Create test submission with all fields populated
	submission := testutils.NewTestSubmission()

	// Validate required fields
	assert.NotEmpty(t, submission.ID)
	assert.Equal(t, "Acme Tech Solutions", submission.CompanyName)
	assert.Equal(t, "john.silva@acme-tech.com.br", submission.ContactEmail)
	assert.NotNil(t, submission.CNPJ)
	assert.Equal(t, "12.345.678/0001-90", *submission.CNPJ)
}

func TestEnrichmentFixture(t *testing.T) {
	submission := testutils.NewTestSubmission()
	enrichment := testutils.NewTestEnrichment(submission.ID)

	// Validate enrichment structure
	testutils.AssertEnrichmentHasData(t, enrichment)
	assert.Equal(t, 100, enrichment.Progress)
	assert.Equal(t, "finished", string(enrichment.Status))
}

func TestAnalysisFixture(t *testing.T) {
	submission := testutils.NewTestSubmission()
	enrichment := testutils.NewTestEnrichment(submission.ID)
	analysis := testutils.NewTestAnalysis(
		submission.ID.String(),
		enrichment.ID.String(),
	)

	// Validate all 11 frameworks are populated
	testutils.AssertAnalysisComplete(t, analysis)
}

// =================================================================
// Example 2: In-Memory Database Testing
// =================================================================

func TestDatabaseSetup(t *testing.T) {
	// Create in-memory SQLite database with schema
	db := testutils.SetupTestDB(t)
	defer testutils.TeardownTestDB(t, db)

	// Verify database is accessible
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM submissions").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Database should be empty initially")
}

func TestInsertTestData(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer testutils.TeardownTestDB(t, db)

	// Insert test submission
	submissionID := db.InsertTestSubmission(t)
	testutils.AssertValidUUID(t, submissionID, "submission_id")

	// Verify submission was inserted
	var companyName string
	err := db.DB.QueryRow("SELECT company_name FROM submissions WHERE id = ?", submissionID).Scan(&companyName)
	require.NoError(t, err)
	assert.Equal(t, "Acme Tech Solutions", companyName)

	// Insert test enrichment
	enrichmentID := db.InsertTestEnrichment(t, submissionID)
	testutils.AssertValidUUID(t, enrichmentID, "enrichment_id")

	// Verify enrichment was inserted
	var status string
	err = db.DB.QueryRow("SELECT status FROM enrichments WHERE id = ?", enrichmentID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "finished", status)
}

// =================================================================
// Example 3: Mock LLM Client
// =================================================================

func TestMockLLMClient(t *testing.T) {
	mockLLM := testutils.NewMockLLMClient()

	// Configure mock expectation
	mockLLM.On("GenerateStructuredWithOptions",
		mock.Anything, // ctx
		mock.Anything, // opts
		mock.Anything, // prompt
		mock.Anything, // data
		mock.Anything, // targetSchema
	).Return(nil)

	// Verify mock was configured (in real tests, you would call the service)
	mockLLM.AssertExpectations(t)

	// Note: In actual usage, you would pass the mockLLM to your service
	// and test the service methods that use it
}

func TestMockLLMClientWithCustomResponse(t *testing.T) {
	mockLLM := testutils.NewMockLLMClient()

	// Set custom response for PESTEL analysis
	customPESTEL := `{
		"political": ["Custom political factor"],
		"economic": ["Custom economic factor"],
		"social": ["Custom social factor"],
		"technological": ["Custom tech factor"],
		"environmental": ["Custom environmental factor"],
		"legal": ["Custom legal factor"],
		"summary": "Custom PESTEL summary"
	}`
	mockLLM.SetResponse("pestel", customPESTEL)

	// Configure mock
	mockLLM.On("GenerateStructuredWithOptions",
		mock.Anything,
		mock.Anything,
		mock.MatchedBy(func(prompt string) bool {
			return testutils.Contains(prompt, "PESTEL")
		}),
		mock.Anything,
		mock.Anything,
	).Return(nil)

	// The custom response is now configured for PESTEL analysis
	// In a real test, you would call the service that uses the mock
	assert.NotNil(t, mockLLM)
}

// =================================================================
// Example 4: Mock Storage Client
// =================================================================

func TestMockStorageClient(t *testing.T) {
	mockStorage := testutils.NewMockStorageClient()

	// Configure mock expectation
	mockStorage.On("Upload",
		mock.Anything, // ctx
		"reports/test-report.pdf",
		mock.Anything, // data
		"application/pdf",
	).Return("https://mock-project.supabase.co/storage/v1/object/public/reports/test-report.pdf", nil)

	// Use mock
	ctx := context.Background()
	pdfData := []byte("%PDF-1.4\nMock PDF content")
	url, err := mockStorage.Upload(ctx, "reports/test-report.pdf", pdfData, "application/pdf")

	require.NoError(t, err)
	assert.Contains(t, url, "mock-project.supabase.co")
	assert.Contains(t, url, "test-report.pdf")

	// Verify file was stored
	storedData, ok := mockStorage.GetUploadedFile("reports/test-report.pdf")
	assert.True(t, ok, "File should be stored")
	assert.Equal(t, pdfData, storedData)

	// Verify mock was called
	mockStorage.AssertExpectations(t)
}

// =================================================================
// Example 5: Mock PDF Generator
// =================================================================

func TestMockPDFGenerator(t *testing.T) {
	mockPDF := testutils.NewMockPDFGenerator()

	// Configure mock expectation
	htmlPages := []string{
		"<html><body>Page 1</body></html>",
		"<html><body>Page 2</body></html>",
	}

	mockPDF.On("Convert",
		mock.Anything, // ctx
		htmlPages,
	).Return([]byte("%PDF-1.4\nGenerated PDF"), nil)

	// Use mock
	ctx := context.Background()
	pdfData, err := mockPDF.Convert(ctx, htmlPages)

	require.NoError(t, err)
	assert.NotEmpty(t, pdfData)
	assert.Contains(t, string(pdfData), "%PDF-1.4")

	// Verify mock was called
	mockPDF.AssertExpectations(t)
}

// =================================================================
// Example 6: Mock Asynq Client
// =================================================================

func TestMockAsynqClient(t *testing.T) {
	mockClient := testutils.NewTestAsynqClient()

	// Create enrichment job
	payload := testutils.CreateEnrichmentPayload(testutils.TestSubmissionID.String())
	task := testutils.NewAsynqTask("enrichment_job", payload)

	// Enqueue job
	info, err := mockClient.Enqueue(task)
	require.NoError(t, err)
	assert.NotEmpty(t, info.ID)

	// Verify job was enqueued
	testutils.AssertTaskEnqueued(t, mockClient, "enrichment_job")

	// Verify payload
	tasks := mockClient.GetEnqueuedTasks()
	assert.Len(t, tasks, 1)
	assert.Equal(t, "enrichment_job", tasks[0].Type())

	submissionID := testutils.ParseEnrichmentPayload(t, tasks[0].Payload())
	assert.Equal(t, testutils.TestSubmissionID.String(), submissionID)
}

// =================================================================
// Example 7: Custom Assertions
// =================================================================

func TestCustomAssertions(t *testing.T) {
	submission := testutils.NewTestSubmission()
	enrichment := testutils.NewTestEnrichment(submission.ID)
	analysis := testutils.NewTestAnalysis(
		submission.ID.String(),
		enrichment.ID.String(),
	)

	// Test submission assertion
	submission2 := testutils.NewTestSubmission()
	submission2.ID = submission.ID
	testutils.AssertSubmissionEqual(t, submission, submission2)

	// Test enrichment assertion
	testutils.AssertEnrichmentHasData(t, enrichment)

	// Test analysis assertion
	testutils.AssertAnalysisComplete(t, analysis)

	// Test UUID assertion
	testutils.AssertValidUUID(t, submission.ID.String(), "submission_id")

	// Test time assertion
	testutils.AssertTimeNotZero(t, submission.CreatedAt, "created_at")
}

// =================================================================
// Example 8: Complete Integration Test
// =================================================================

func TestCompleteWorkflow(t *testing.T) {
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

	// Step 1: Create submission
	submissionID := db.InsertTestSubmission(t)
	testutils.AssertValidUUID(t, submissionID, "submission_id")

	// Step 2: Enqueue enrichment job
	enrichmentPayload := testutils.CreateEnrichmentPayload(submissionID)
	enrichmentTask := testutils.NewAsynqTask("enrichment_job", enrichmentPayload)
	_, err := mockAsynq.Enqueue(enrichmentTask)
	require.NoError(t, err)

	// Step 3: Process enrichment
	enrichmentID := db.InsertTestEnrichment(t, submissionID)
	testutils.AssertValidUUID(t, enrichmentID, "enrichment_id")

	// Step 4: Enqueue analysis job
	analysisPayload := testutils.CreateAnalysisPayload(submissionID, enrichmentID)
	analysisTask := testutils.NewAsynqTask("analysis_job", analysisPayload)
	_, err = mockAsynq.Enqueue(analysisTask)
	require.NoError(t, err)

	// Step 5: Create analysis
	analysis := testutils.NewTestAnalysis(submissionID, enrichmentID)
	testutils.AssertAnalysisComplete(t, analysis)

	// Verify all jobs were enqueued
	assert.Len(t, mockAsynq.GetEnqueuedTasks(), 2)
	testutils.AssertTaskEnqueued(t, mockAsynq, "enrichment_job")
	testutils.AssertTaskEnqueued(t, mockAsynq, "analysis_job")

	// Verify mocks were called
	mockLLM.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockPDF.AssertExpectations(t)
}

// =================================================================
// Helper Functions (for examples)
// =================================================================

// Note: These examples use testutils.NewAsynqTask for actual asynq.Task creation
