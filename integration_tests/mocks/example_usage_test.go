package mocks

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Example 1: Basic Upload and Retrieval Test
func TestExample_BasicUploadAndRetrieval(t *testing.T) {
	// Initialize mock with default configuration
	mock := NewSupabaseJSONMockDefault()
	defer mock.Clear() // Clean up after test

	ctx := context.Background()

	// Prepare test data
	testContent := []byte("This is a test PDF report content")
	filePath := "company-123/analysis-report-v1.pdf"

	// Upload file
	publicURL, err := mock.Upload(ctx, filePath, testContent, "application/pdf")
	require.NoError(t, err, "Upload should succeed")
	assert.Contains(t, publicURL, "mock-project.supabase.co")
	assert.Contains(t, publicURL, filePath)

	// Retrieve and verify file
	retrievedData, exists := mock.GetFileData(filePath)
	require.True(t, exists, "File should exist")
	assert.Equal(t, testContent, retrievedData, "Content should match")

	// Verify metadata
	metadata, exists := mock.GetFileMetadata(filePath)
	require.True(t, exists, "Metadata should exist")
	assert.Equal(t, int64(len(testContent)), metadata["size"])
	assert.Equal(t, "application/pdf", metadata["content_type"])
}

// Example 2: Using JSON Test Scenarios
func TestExample_UsingJSONScenarios(t *testing.T) {
	// Load mock from JSON file (includes pre-defined scenarios)
	mock, err := NewSupabaseJSONMock("supabase_mock_data.json")
	require.NoError(t, err, "Should load JSON mock successfully")

	ctx := context.Background()

	// Get test scenario from JSON
	scenario, exists := mock.GetTestScenario("successful_upload")
	require.True(t, exists, "Test scenario should exist")

	t.Logf("Testing scenario: %s", scenario.Description)

	// Extract scenario inputs
	path := scenario.Input["path"].(string)
	contentType := scenario.Input["content_type"].(string)

	// Execute test based on scenario
	testData := []byte("Mock PDF content for scenario test")
	publicURL, err := mock.Upload(ctx, path, testData, contentType)

	// Verify expected outputs
	require.NoError(t, err)
	expectedURL := scenario.ExpectedOutput["public_url"].(string)
	assert.Equal(t, expectedURL, publicURL)

	t.Logf("Scenario test passed: %s", scenario.Description)
}

// Example 3: Testing File Validation Rules
func TestExample_FileValidation(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()

	ctx := context.Background()

	tests := []struct {
		name        string
		contentType string
		dataSize    int
		shouldFail  bool
		errorText   string
	}{
		{
			name:        "Valid PDF upload",
			contentType: "application/pdf",
			dataSize:    1024,
			shouldFail:  false,
		},
		{
			name:        "Invalid content type",
			contentType: "application/x-msdownload",
			dataSize:    1024,
			shouldFail:  true,
			errorText:   "not allowed",
		},
		{
			name:        "File too large",
			contentType: "application/pdf",
			dataSize:    mock.MaxFileSize + 1,
			shouldFail:  true,
			errorText:   "exceeds maximum",
		},
		{
			name:        "Valid JSON upload",
			contentType: "application/json",
			dataSize:    512,
			shouldFail:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testData := make([]byte, tt.dataSize)
			path := "test-validation/" + tt.name + ".file"

			_, err := mock.Upload(ctx, path, testData, tt.contentType)

			if tt.shouldFail {
				assert.Error(t, err, "Should fail validation")
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText)
				}
			} else {
				assert.NoError(t, err, "Should pass validation")
			}
		})
	}
}

// Example 4: Testing Multiple File Operations
func TestExample_MultipleFileOperations(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()
	defer mock.Clear()

	ctx := context.Background()

	// Upload multiple files
	files := map[string][]byte{
		"submission-001/report-v1.pdf": []byte("Report version 1"),
		"submission-001/report-v2.pdf": []byte("Report version 2"),
		"submission-002/report-v1.pdf": []byte("Another company report"),
	}

	for path, content := range files {
		_, err := mock.Upload(ctx, path, content, "application/pdf")
		require.NoError(t, err, "Upload should succeed for %s", path)
	}

	// Verify file count
	assert.Equal(t, len(files), mock.FileCount())

	// List all files
	storedFiles := mock.ListFiles()
	assert.Equal(t, len(files), len(storedFiles))

	// Delete one file
	err := mock.DeleteFile("submission-001/report-v1.pdf")
	require.NoError(t, err)
	assert.Equal(t, len(files)-1, mock.FileCount())

	// Verify deletion
	_, exists := mock.GetFile("submission-001/report-v1.pdf")
	assert.False(t, exists, "Deleted file should not exist")

	// Other files should still exist
	_, exists = mock.GetFile("submission-001/report-v2.pdf")
	assert.True(t, exists, "Other files should still exist")
}

// Example 5: Testing Pre-loaded Data from JSON
func TestExample_PreloadedJSONData(t *testing.T) {
	// Load mock with pre-existing files from JSON
	mock, err := NewSupabaseJSONMock("supabase_mock_data.json")
	require.NoError(t, err)

	// Check that pre-loaded files exist
	preloadedFiles := []string{
		"test-submission-123/report-v1.pdf",
		"test-submission-456/report-v1.pdf",
		"test-submission-456/report-v2.pdf",
	}

	for _, path := range preloadedFiles {
		file, exists := mock.GetFile(path)
		require.True(t, exists, "Pre-loaded file should exist: %s", path)

		// Verify metadata from JSON
		assert.Greater(t, file.Size, int64(0))
		assert.Equal(t, "application/pdf", file.ContentType)
		assert.Contains(t, file.PublicURL, path)

		t.Logf("Found pre-loaded file: %s (size: %d bytes)", path, file.Size)
	}
}

// Example 6: Concurrent Access Testing
func TestExample_ConcurrentUploads(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()
	defer mock.Clear()

	ctx := context.Background()
	numGoroutines := 20

	// Channel to collect results
	results := make(chan error, numGoroutines)

	// Concurrent uploads
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			path := "concurrent-test/file-" + string(rune(index)) + ".pdf"
			testData := []byte("Concurrent upload content")

			_, err := mock.Upload(ctx, path, testData, "application/pdf")
			results <- err
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err, "Concurrent upload should succeed")
	}

	// Verify all files were uploaded
	assert.Equal(t, numGoroutines, mock.FileCount())
}

// Example 7: Export Current State for Debugging
func TestExample_ExportState(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()

	ctx := context.Background()

	// Upload some test data (only using allowed content types)
	testFiles := []struct {
		path        string
		content     string
		contentType string
	}{
		{"debug/file1.pdf", "Content 1", "application/pdf"},
		{"debug/file2.json", `{"key": "value"}`, "application/json"},
		{"debug/file3.pdf", "Another PDF", "application/pdf"},
	}

	for _, tf := range testFiles {
		_, err := mock.Upload(ctx, tf.path, []byte(tf.content), tf.contentType)
		require.NoError(t, err)
	}

	// Export current state to JSON
	exportedJSON, err := mock.ExportToJSON()
	require.NoError(t, err)

	// Verify export contains our data
	exportStr := string(exportedJSON)
	assert.Contains(t, exportStr, "debug/file1.pdf")
	assert.Contains(t, exportStr, "debug/file2.json")
	assert.Contains(t, exportStr, "mock-project.supabase.co")

	t.Logf("Exported state:\n%s", exportStr)
}

// Example 8: Integration with Service Layer
type MockReportService struct {
	storage *SupabaseJSONMock
}

func NewMockReportService() *MockReportService {
	return &MockReportService{
		storage: NewSupabaseJSONMockDefault(),
	}
}

func (s *MockReportService) SaveReport(ctx context.Context, submissionID string, version int, content []byte) (string, error) {
	path := fmt.Sprintf("%s/report-v%d.pdf", submissionID, version)
	return s.storage.Upload(ctx, path, content, "application/pdf")
}

func (s *MockReportService) GetReport(submissionID string, version int) ([]byte, error) {
	path := fmt.Sprintf("%s/report-v%d.pdf", submissionID, version)
	data, exists := s.storage.GetFileData(path)
	if !exists {
		return nil, fmt.Errorf("report not found: %s", path)
	}
	return data, nil
}

func TestExample_ServiceLayerIntegration(t *testing.T) {
	service := NewMockReportService()
	ctx := context.Background()

	submissionID := "test-company-789"
	reportContent := []byte("Full business analysis report content")

	// Save report
	publicURL, err := service.SaveReport(ctx, submissionID, 1, reportContent)
	require.NoError(t, err)
	assert.Contains(t, publicURL, submissionID)

	// Retrieve report
	retrievedContent, err := service.GetReport(submissionID, 1)
	require.NoError(t, err)
	assert.Equal(t, reportContent, retrievedContent)

	// Try to get non-existent version
	_, err = service.GetReport(submissionID, 99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
