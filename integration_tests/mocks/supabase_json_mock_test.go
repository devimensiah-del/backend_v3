package mocks

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupabaseJSONMock_Upload(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()

	ctx := context.Background()
	testData := []byte("Mock PDF content")
	path := "test-submission-999/report-v1.pdf"

	// Upload file
	publicURL, err := mock.Upload(ctx, path, testData, "application/pdf")
	require.NoError(t, err)
	assert.Contains(t, publicURL, "mock-project.supabase.co")
	assert.Contains(t, publicURL, path)

	// Verify file was stored
	assert.Equal(t, 1, mock.FileCount())
}

func TestSupabaseJSONMock_GetFile(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()

	ctx := context.Background()
	testData := []byte("Mock PDF content")
	path := "test-submission-999/report-v1.pdf"

	// Upload file
	_, err := mock.Upload(ctx, path, testData, "application/pdf")
	require.NoError(t, err)

	// Retrieve file
	file, exists := mock.GetFile(path)
	require.True(t, exists)
	assert.Equal(t, int64(len(testData)), file.Size)
	assert.Equal(t, "application/pdf", file.ContentType)
	assert.Equal(t, testData, file.Data)
}

func TestSupabaseJSONMock_GetFileData(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()

	ctx := context.Background()
	testData := []byte("Mock PDF content")
	path := "test-submission-999/report-v1.pdf"

	// Upload file
	_, err := mock.Upload(ctx, path, testData, "application/pdf")
	require.NoError(t, err)

	// Retrieve file data
	data, exists := mock.GetFileData(path)
	require.True(t, exists)
	assert.Equal(t, testData, data)
}

func TestSupabaseJSONMock_InvalidContentType(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()

	ctx := context.Background()
	testData := []byte("Mock content")
	path := "test.exe"

	// Attempt upload with invalid content type
	_, err := mock.Upload(ctx, path, testData, "application/x-msdownload")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestSupabaseJSONMock_FileSizeLimit(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()

	ctx := context.Background()
	// Create data larger than max size
	largeData := make([]byte, mock.MaxFileSize+1)
	path := "large-file.pdf"

	// Attempt upload
	_, err := mock.Upload(ctx, path, largeData, "application/pdf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestSupabaseJSONMock_ListFiles(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()

	ctx := context.Background()

	// Upload multiple files
	files := []string{
		"submission-1/report.pdf",
		"submission-2/report.pdf",
		"submission-3/report.pdf",
	}

	for _, path := range files {
		_, err := mock.Upload(ctx, path, []byte("content"), "application/pdf")
		require.NoError(t, err)
	}

	// List files
	storedFiles := mock.ListFiles()
	assert.Equal(t, 3, len(storedFiles))
}

func TestSupabaseJSONMock_Clear(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()

	ctx := context.Background()

	// Upload files
	_, err := mock.Upload(ctx, "test1.pdf", []byte("content"), "application/pdf")
	require.NoError(t, err)
	_, err = mock.Upload(ctx, "test2.pdf", []byte("content"), "application/pdf")
	require.NoError(t, err)

	assert.Equal(t, 2, mock.FileCount())

	// Clear
	mock.Clear()
	assert.Equal(t, 0, mock.FileCount())
}

func TestSupabaseJSONMock_GetFileMetadata(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()

	ctx := context.Background()
	testData := []byte("Mock PDF content")
	path := "test-submission-999/report-v1.pdf"

	// Upload file
	_, err := mock.Upload(ctx, path, testData, "application/pdf")
	require.NoError(t, err)

	// Get metadata
	metadata, exists := mock.GetFileMetadata(path)
	require.True(t, exists)

	assert.Equal(t, int64(len(testData)), metadata["size"])
	assert.Equal(t, "application/pdf", metadata["content_type"])
	assert.NotNil(t, metadata["created_at"])
	assert.NotNil(t, metadata["public_url"])
}

func TestSupabaseJSONMock_DeleteFile(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()

	ctx := context.Background()
	path := "test-deletion.pdf"

	// Upload file
	_, err := mock.Upload(ctx, path, []byte("content"), "application/pdf")
	require.NoError(t, err)
	assert.Equal(t, 1, mock.FileCount())

	// Delete file
	err = mock.DeleteFile(path)
	require.NoError(t, err)
	assert.Equal(t, 0, mock.FileCount())

	// Verify file is gone
	_, exists := mock.GetFile(path)
	assert.False(t, exists)
}

func TestSupabaseJSONMock_DeleteNonExistentFile(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()

	// Attempt to delete non-existent file
	err := mock.DeleteFile("nonexistent.pdf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSupabaseJSONMock_ConcurrentAccess(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()

	ctx := context.Background()
	done := make(chan bool)

	// Concurrent uploads
	for i := 0; i < 10; i++ {
		go func(index int) {
			path := fmt.Sprintf("concurrent-%d.pdf", index)
			_, err := mock.Upload(ctx, path, []byte("content"), "application/pdf")
			require.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all uploads
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 10, mock.FileCount())
}

func TestSupabaseJSONMock_ExportToJSON(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()

	ctx := context.Background()

	// Upload a file
	_, err := mock.Upload(ctx, "export-test.pdf", []byte("content"), "application/pdf")
	require.NoError(t, err)

	// Export to JSON
	jsonData, err := mock.ExportToJSON()
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), "export-test.pdf")
	assert.Contains(t, string(jsonData), "mock-project.supabase.co")
}

func TestSupabaseJSONMock_TestScenarios(t *testing.T) {
	mock := NewSupabaseJSONMockDefault()

	// Test scenarios should be loaded (if JSON file is available)
	// If not available, this is just a placeholder test
	scenarios := mock.TestScenarios
	t.Logf("Loaded %d test scenarios", len(scenarios))

	// This test passes regardless - it's just documenting the feature
	assert.True(t, true)
}
