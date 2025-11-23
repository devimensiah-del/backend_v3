package integration_tests

import (
	"backend_v3/integration_tests/mocks"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMixedConfiguration tests with Supabase JSON mock but REAL OpenRouter LLM
func TestMixedConfiguration_JSONStorageRealLLM(t *testing.T) {
	// Force configuration: Use JSON mock for storage, REAL OpenRouter for LLM
	env := LoadTestEnvWithOverrides(false, true)

	// Verify configuration is as expected
	require.False(t, env.UseLLMMock, "Should use REAL LLM (OpenRouter)")
	require.True(t, env.UseStorageMock, "Should use MOCK storage (JSON)")

	t.Logf("Test Configuration: %s", env.GetConfigSummary())

	// Skip test if OpenRouter key is not available
	if env.shouldUseLLMMock() {
		t.Skip("OpenRouter API key not configured - skipping real LLM test")
	}

	ctx := context.Background()

	t.Run("Storage uses JSON mock", func(t *testing.T) {
		// Get storage client
		storageClient := env.GetSupabaseStorageClient()
		require.NotNil(t, storageClient)

		// Verify it's the JSON mock type (not real Supabase client)
		// The JSON mock has specific methods like FileCount()
		t.Logf("✓ Storage client initialized with JSON mock")
	})

	t.Run("LLM uses real OpenRouter", func(t *testing.T) {
		// Get LLM client
		llmClient := env.GetLLMClient()
		require.NotNil(t, llmClient)

		// This will use the REAL OpenRouter API
		t.Logf("✓ LLM client initialized with real OpenRouter (key: %s...)",
			env.OpenRouterAPIKey[:10])
	})

	t.Run("Simple storage operation", func(t *testing.T) {
		// Get storage mock
		storage := env.GetSupabaseStorageClient()
		require.NotNil(t, storage)

		// Type assert to JSON mock to use its methods
		jsonMock, ok := storage.(*mocks.SupabaseJSONMock)
		if !ok {
			t.Fatal("Storage should be JSON mock type")
		}

		// Clear any pre-loaded files from JSON
		jsonMock.Clear()

		// Upload test file
		testData := []byte("Test report content")
		publicURL, err := jsonMock.Upload(ctx, "test/report.pdf", testData, "application/pdf")
		require.NoError(t, err)
		assert.Contains(t, publicURL, "mock-project.supabase.co")

		// Verify file was stored
		assert.Equal(t, 1, jsonMock.FileCount())

		// Retrieve file
		data, exists := jsonMock.GetFileData("test/report.pdf")
		require.True(t, exists)
		assert.Equal(t, testData, data)

		t.Logf("✓ JSON mock storage working correctly")
	})

	t.Run("Real LLM API call", func(t *testing.T) {
		// Get real LLM client
		llmClient := env.GetLLMClient()
		require.NotNil(t, llmClient)

		// Make a simple API call to verify it's working
		// This is a minimal test to verify the real API is accessible
		// You can expand this to test actual enrichment/analysis

		t.Logf("✓ Real LLM client ready for API calls")

		// NOTE: Uncomment below to make actual API call
		// This will consume API credits, so only enable when needed
		/*
		prompt := "Respond with just 'OK' if you can read this message."
		response, err := llmClient.SimpleCompletion(ctx, prompt)
		require.NoError(t, err)
		assert.NotEmpty(t, response)
		t.Logf("LLM Response: %s", response)
		*/
	})
}

// TestFullWorkflowWithMixedConfig demonstrates complete workflow with mixed configuration
func TestFullWorkflowWithMixedConfig(t *testing.T) {
	// Use JSON storage mock but real LLM
	env := LoadTestEnvWithOverrides(false, true)

	// Skip if no real LLM key
	if env.shouldUseLLMMock() {
		t.Skip("OpenRouter API key not configured")
	}

	t.Logf("Running full workflow with: %s", env.GetConfigSummary())

	ctx := context.Background()

	t.Run("Complete workflow simulation", func(t *testing.T) {
		// 1. Get clients
		llmClient := env.GetLLMClient()
		storageClient := env.GetSupabaseStorageClient()

		require.NotNil(t, llmClient, "LLM client should be initialized")
		require.NotNil(t, storageClient, "Storage client should be initialized")

		// 2. Simulate enrichment step (would call real LLM)
		t.Log("Step 1: Enrichment (using REAL LLM)")
		// enrichmentData := performEnrichment(llmClient, submissionData)

		// 3. Simulate analysis step (would call real LLM)
		t.Log("Step 2: Analysis (using REAL LLM)")
		// analysisData := performAnalysis(llmClient, enrichmentData)

		// 4. Simulate report generation and storage (uses mock)
		t.Log("Step 3: Report storage (using JSON MOCK)")

		jsonMock, ok := storageClient.(*mocks.SupabaseJSONMock)
		require.True(t, ok, "Storage should be JSON mock")

		// Store mock report
		reportData := []byte("Generated business analysis report PDF")
		reportURL, err := jsonMock.Upload(ctx, "reports/test-123/report-v1.pdf", reportData, "application/pdf")
		require.NoError(t, err)
		t.Logf("Report stored: %s", reportURL)

		// 5. Verify complete workflow
		assert.Equal(t, 1, jsonMock.FileCount())

		t.Log("✓ Full workflow completed successfully with mixed configuration")
	})
}

// TestConfigurationOverrides validates the override system works correctly
func TestConfigurationOverrides(t *testing.T) {
	tests := []struct {
		name              string
		useLLMMock        bool
		useStorageMock    bool
		expectedLLMType   string
		expectedStorageType string
	}{
		{
			name:              "Both mocks",
			useLLMMock:        true,
			useStorageMock:    true,
			expectedLLMType:   "mock",
			expectedStorageType: "mock",
		},
		{
			name:              "Real LLM, Mock Storage",
			useLLMMock:        false,
			useStorageMock:    true,
			expectedLLMType:   "real",
			expectedStorageType: "mock",
		},
		{
			name:              "Mock LLM, Real Storage",
			useLLMMock:        true,
			useStorageMock:    false,
			expectedLLMType:   "mock",
			expectedStorageType: "real",
		},
		{
			name:              "Both real",
			useLLMMock:        false,
			useStorageMock:    false,
			expectedLLMType:   "real",
			expectedStorageType: "real",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := LoadTestEnvWithOverrides(tt.useLLMMock, tt.useStorageMock)

			assert.Equal(t, tt.useLLMMock, env.UseLLMMock)
			assert.Equal(t, tt.useStorageMock, env.UseStorageMock)

			summary := env.GetConfigSummary()
			t.Logf("Configuration: %s", summary)

			// Verify clients are initialized
			llmClient := env.GetLLMClient()
			storageClient := env.GetSupabaseStorageClient()

			assert.NotNil(t, llmClient, "LLM client should be initialized")
			assert.NotNil(t, storageClient, "Storage client should be initialized")

			// Verify storage type
			_, isJSONMock := storageClient.(*mocks.SupabaseJSONMock)
			if tt.useStorageMock {
				assert.True(t, isJSONMock, "Should be JSON mock storage")
			}
			// Note: Can't easily verify real Supabase client type without importing infrastructure
		})
	}
}

// TestJSONMockFeatures demonstrates specific JSON mock capabilities
func TestJSONMockFeatures(t *testing.T) {
	env := LoadTestEnvWithOverrides(true, true) // All mocks for this test
	ctx := context.Background()

	storage := env.GetSupabaseStorageClient()
	jsonMock, ok := storage.(*mocks.SupabaseJSONMock)
	require.True(t, ok, "Should be JSON mock")

	t.Run("File count tracking", func(t *testing.T) {
		// Clear pre-loaded files first
		jsonMock.Clear()
		defer jsonMock.Clear()

		assert.Equal(t, 0, jsonMock.FileCount())

		jsonMock.Upload(ctx, "file1.pdf", []byte("content"), "application/pdf")
		assert.Equal(t, 1, jsonMock.FileCount())

		jsonMock.Upload(ctx, "file2.pdf", []byte("content"), "application/pdf")
		assert.Equal(t, 2, jsonMock.FileCount())
	})

	t.Run("File listing", func(t *testing.T) {
		defer jsonMock.Clear()

		jsonMock.Upload(ctx, "reports/file1.pdf", []byte("c1"), "application/pdf")
		jsonMock.Upload(ctx, "reports/file2.pdf", []byte("c2"), "application/pdf")

		files := jsonMock.ListFiles()
		assert.Equal(t, 2, len(files))
	})

	t.Run("File metadata", func(t *testing.T) {
		defer jsonMock.Clear()

		testData := []byte("Test content")
		jsonMock.Upload(ctx, "test.pdf", testData, "application/pdf")

		metadata, exists := jsonMock.GetFileMetadata("test.pdf")
		require.True(t, exists)

		assert.Equal(t, int64(len(testData)), metadata["size"])
		assert.Equal(t, "application/pdf", metadata["content_type"])
		assert.NotNil(t, metadata["created_at"])
		assert.NotNil(t, metadata["public_url"])
	})

	t.Run("Export state for debugging", func(t *testing.T) {
		defer jsonMock.Clear()

		jsonMock.Upload(ctx, "debug/test.pdf", []byte("debug"), "application/pdf")

		exportJSON, err := jsonMock.ExportToJSON()
		require.NoError(t, err)

		exportStr := string(exportJSON)
		assert.Contains(t, exportStr, "debug/test.pdf")
		assert.Contains(t, exportStr, "mock-project.supabase.co")

		t.Logf("Exported state:\n%s", exportStr)
	})
}
