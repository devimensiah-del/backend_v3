package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"backend_v3/llm"

	"github.com/stretchr/testify/mock"
)

// =================================================================
// MockLLMClient - Implements llm.Client interface
// =================================================================

// MockLLMClient is a configurable mock for LLM API calls
// Returns predefined JSON responses for enrichment and analysis
type MockLLMClient struct {
	mock.Mock
	Responses map[string]string // Map prompt keywords to JSON responses
}

// NewMockLLMClient creates a new mock LLM client with default responses
func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		Responses: map[string]string{
			"enrichment": getDefaultEnrichmentResponse(),
			"pestel":     getDefaultPESTELResponse(),
			"porter":     getDefaultPorterResponse(),
			"swot":       getDefaultSWOTResponse(),
			"tam_sam_som": getDefaultTamSamSomResponse(),
			"blue_ocean": getDefaultBlueOceanResponse(),
			"okr":        getDefaultOKRResponse(),
			"bsc":        getDefaultBSCResponse(),
			"benchmarking": getDefaultBenchmarkingResponse(),
			"growth_hacking": getDefaultGrowthHackingResponse(),
			"scenario":   getDefaultScenarioResponse(),
			"decision_matrix": getDefaultDecisionMatrixResponse(),
			"synthesis":  getDefaultSynthesisResponse(),
		},
	}
}

// GenerateStructuredWithOptions mocks LLM API calls with framework-specific options
func (m *MockLLMClient) GenerateStructuredWithOptions(
	ctx context.Context,
	opts llm.GenerationOptions,
	promptTemplate string,
	data interface{},
	targetSchema interface{},
) error {
	args := m.Called(ctx, opts, promptTemplate, data, targetSchema)

	if args.Error(0) != nil {
		return args.Error(0)
	}

	// Auto-detect response type based on prompt content
	var responseJSON string
	switch {
	case contains(promptTemplate, "enrichment") || contains(promptTemplate, "UnifiedProfile"):
		responseJSON = m.Responses["enrichment"]
	case contains(promptTemplate, "PESTEL"):
		responseJSON = m.Responses["pestel"]
	case contains(promptTemplate, "Porter"):
		responseJSON = m.Responses["porter"]
	case contains(promptTemplate, "SWOT"):
		responseJSON = m.Responses["swot"]
	case contains(promptTemplate, "TAM") || contains(promptTemplate, "SAM"):
		responseJSON = m.Responses["tam_sam_som"]
	case contains(promptTemplate, "Blue Ocean"):
		responseJSON = m.Responses["blue_ocean"]
	case contains(promptTemplate, "OKR"):
		responseJSON = m.Responses["okr"]
	case contains(promptTemplate, "Balanced Scorecard") || contains(promptTemplate, "BSC"):
		responseJSON = m.Responses["bsc"]
	case contains(promptTemplate, "Benchmarking"):
		responseJSON = m.Responses["benchmarking"]
	case contains(promptTemplate, "Growth") || contains(promptTemplate, "LEAP"):
		responseJSON = m.Responses["growth_hacking"]
	case contains(promptTemplate, "Scenario"):
		responseJSON = m.Responses["scenario"]
	case contains(promptTemplate, "Decision Matrix"):
		responseJSON = m.Responses["decision_matrix"]
	case contains(promptTemplate, "Synthesis"):
		responseJSON = m.Responses["synthesis"]
	default:
		responseJSON = "{}"
	}

	// Unmarshal into target schema
	return json.Unmarshal([]byte(responseJSON), targetSchema)
}

// SetResponse allows tests to configure custom responses
func (m *MockLLMClient) SetResponse(key string, jsonResponse string) {
	m.Responses[key] = jsonResponse
}

// =================================================================
// MockStorageClient - Implements infrastructure.StorageClient
// =================================================================

// MockStorageClient simulates Supabase file uploads
type MockStorageClient struct {
	mock.Mock
	UploadedFiles map[string][]byte // Track uploaded files
}

// NewMockStorageClient creates a new mock storage client
func NewMockStorageClient() *MockStorageClient {
	return &MockStorageClient{
		UploadedFiles: make(map[string][]byte),
	}
}

// Upload simulates file upload to Supabase Storage
func (m *MockStorageClient) Upload(ctx context.Context, path string, data []byte, contentType string) (string, error) {
	args := m.Called(ctx, path, data, contentType)

	if args.Error(0) != nil {
		return "", args.Error(0)
	}

	// Store file in memory
	m.UploadedFiles[path] = data

	// Return mock public URL
	publicURL := fmt.Sprintf("https://mock-project.supabase.co/storage/v1/object/public/reports/%s", path)
	return publicURL, nil
}

// GetUploadedFile retrieves uploaded file data for verification
func (m *MockStorageClient) GetUploadedFile(path string) ([]byte, bool) {
	data, ok := m.UploadedFiles[path]
	return data, ok
}

// =================================================================
// MockPDFGenerator - Implements infrastructure.PDFGenerator
// =================================================================

// MockPDFGenerator simulates Gotenberg PDF generation
type MockPDFGenerator struct {
	mock.Mock
	GeneratedPDFs map[string][]byte // Track generated PDFs
}

// NewMockPDFGenerator creates a new mock PDF generator
func NewMockPDFGenerator() *MockPDFGenerator {
	return &MockPDFGenerator{
		GeneratedPDFs: make(map[string][]byte),
	}
}

// Convert simulates HTML to PDF conversion
func (m *MockPDFGenerator) Convert(ctx context.Context, htmlPages []string) ([]byte, error) {
	args := m.Called(ctx, htmlPages)

	if args.Error(0) != nil {
		return nil, args.Error(0)
	}

	// Generate mock PDF data
	pdfData := []byte(fmt.Sprintf("%%PDF-1.4\n%% Mock PDF with %d pages\n", len(htmlPages)))

	// Store generated PDF
	key := fmt.Sprintf("pdf_%d", time.Now().Unix())
	m.GeneratedPDFs[key] = pdfData

	return pdfData, nil
}

// GetGeneratedPDF retrieves generated PDF for verification
func (m *MockPDFGenerator) GetGeneratedPDF(key string) ([]byte, bool) {
	data, ok := m.GeneratedPDFs[key]
	return data, ok
}

// =================================================================
// Helper Functions
// =================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Contains is exported for use in tests
func Contains(s, substr string) bool {
	return contains(s, substr)
}
