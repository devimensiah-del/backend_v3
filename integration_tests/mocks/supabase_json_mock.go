package mocks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SupabaseJSONMock loads mock data from JSON file
type SupabaseJSONMock struct {
	// In-memory storage
	files map[string]*StoredFile
	mu    sync.RWMutex

	// Configuration loaded from JSON
	ProjectURL         string
	DefaultBucket      string
	MaxFileSize        int
	AllowedContentTypes []string

	// Test scenarios from JSON
	TestScenarios map[string]TestScenario
}

// StoredFile represents a file in mock storage
type StoredFile struct {
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
	PublicURL   string    `json:"public_url"`
	Data        []byte    `json:"-"` // Actual file content (not in JSON)
}

// TestScenario represents a test case from JSON
type TestScenario struct {
	Description    string                 `json:"description"`
	Input          map[string]interface{} `json:"input"`
	ExpectedOutput map[string]interface{} `json:"expected_output"`
}

// JSONMockData represents the structure of supabase_mock_data.json
type JSONMockData struct {
	Storage struct {
		Buckets map[string]struct {
			Public bool                      `json:"public"`
			Files  map[string]*StoredFile    `json:"files"`
		} `json:"buckets"`
	} `json:"storage"`
	Auth struct {
		Users    map[string]interface{} `json:"users"`
		Sessions map[string]interface{} `json:"sessions"`
	} `json:"auth"`
	TestScenarios map[string]TestScenario `json:"test_scenarios"`
	Config        struct {
		ProjectURL          string   `json:"project_url"`
		DefaultBucket       string   `json:"default_bucket"`
		MaxFileSize         int      `json:"max_file_size"`
		AllowedContentTypes []string `json:"allowed_content_types"`
	} `json:"config"`
}

// NewSupabaseJSONMock creates a mock from JSON file
func NewSupabaseJSONMock(jsonPath string) (*SupabaseJSONMock, error) {
	// If no path provided, use default
	if jsonPath == "" {
		jsonPath = filepath.Join("mocks", "supabase_mock_data.json")
	}

	// Read JSON file
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read mock data file: %w", err)
	}

	// Parse JSON
	var mockData JSONMockData
	if err := json.Unmarshal(data, &mockData); err != nil {
		return nil, fmt.Errorf("failed to parse mock data JSON: %w", err)
	}

	// Create mock instance
	mock := &SupabaseJSONMock{
		files:               make(map[string]*StoredFile),
		ProjectURL:          mockData.Config.ProjectURL,
		DefaultBucket:       mockData.Config.DefaultBucket,
		MaxFileSize:         mockData.Config.MaxFileSize,
		AllowedContentTypes: mockData.Config.AllowedContentTypes,
		TestScenarios:       mockData.TestScenarios,
	}

	// Load pre-existing files from JSON
	for bucketName, bucket := range mockData.Storage.Buckets {
		for filePath, fileInfo := range bucket.Files {
			key := fmt.Sprintf("%s/%s", bucketName, filePath)
			mock.files[key] = fileInfo
		}
	}

	return mock, nil
}

// NewSupabaseJSONMockDefault creates a mock with embedded default data
func NewSupabaseJSONMockDefault() *SupabaseJSONMock {
	// Try to load from file, fallback to in-memory defaults
	mock, err := NewSupabaseJSONMock("mocks/supabase_mock_data.json")
	if err != nil {
		// Fallback to basic in-memory mock
		return &SupabaseJSONMock{
			files:               make(map[string]*StoredFile),
			ProjectURL:          "https://mock-project.supabase.co",
			DefaultBucket:       "reports",
			MaxFileSize:         52428800, // 50MB
			AllowedContentTypes: []string{"application/pdf", "application/json"},
			TestScenarios:       make(map[string]TestScenario),
		}
	}
	return mock
}

// Upload stores a file and returns a public URL
func (m *SupabaseJSONMock) Upload(ctx context.Context, path string, data []byte, contentType string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate content type
	if !m.isContentTypeAllowed(contentType) {
		return "", fmt.Errorf("content type '%s' not allowed", contentType)
	}

	// Validate file size
	if len(data) > m.MaxFileSize {
		return "", fmt.Errorf("file size %d exceeds maximum %d", len(data), m.MaxFileSize)
	}

	// Generate key
	key := fmt.Sprintf("%s/%s", m.DefaultBucket, path)

	// Store file
	m.files[key] = &StoredFile{
		Size:        int64(len(data)),
		ContentType: contentType,
		CreatedAt:   time.Now(),
		PublicURL:   fmt.Sprintf("%s/storage/v1/object/public/%s/%s", m.ProjectURL, m.DefaultBucket, path),
		Data:        data,
	}

	return m.files[key].PublicURL, nil
}

// GetFile retrieves a file from storage
func (m *SupabaseJSONMock) GetFile(path string) (*StoredFile, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", m.DefaultBucket, path)
	file, exists := m.files[key]
	return file, exists
}

// GetFileData retrieves file content
func (m *SupabaseJSONMock) GetFileData(path string) ([]byte, bool) {
	file, exists := m.GetFile(path)
	if !exists {
		return nil, false
	}
	return file.Data, true
}

// ListFiles returns all stored file paths
func (m *SupabaseJSONMock) ListFiles() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	paths := make([]string, 0, len(m.files))
	for path := range m.files {
		paths = append(paths, path)
	}
	return paths
}

// Clear removes all files from storage
func (m *SupabaseJSONMock) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.files = make(map[string]*StoredFile)
}

// FileCount returns the number of stored files
func (m *SupabaseJSONMock) FileCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.files)
}

// GetTestScenario retrieves a test scenario by name
func (m *SupabaseJSONMock) GetTestScenario(name string) (TestScenario, bool) {
	scenario, exists := m.TestScenarios[name]
	return scenario, exists
}

// isContentTypeAllowed checks if content type is in allowed list
func (m *SupabaseJSONMock) isContentTypeAllowed(contentType string) bool {
	for _, allowed := range m.AllowedContentTypes {
		if contentType == allowed {
			return true
		}
	}
	return false
}

// GetFileMetadata returns file metadata without content
func (m *SupabaseJSONMock) GetFileMetadata(path string) (map[string]interface{}, bool) {
	file, exists := m.GetFile(path)
	if !exists {
		return nil, false
	}

	metadata := map[string]interface{}{
		"size":         file.Size,
		"content_type": file.ContentType,
		"created_at":   file.CreatedAt,
		"public_url":   file.PublicURL,
	}

	return metadata, true
}

// DeleteFile removes a file from storage
func (m *SupabaseJSONMock) DeleteFile(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s/%s", m.DefaultBucket, path)
	if _, exists := m.files[key]; !exists {
		return fmt.Errorf("file not found: %s", path)
	}

	delete(m.files, key)
	return nil
}

// ExportToJSON exports current state to JSON (for debugging/testing)
func (m *SupabaseJSONMock) ExportToJSON() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	exportData := map[string]interface{}{
		"project_url": m.ProjectURL,
		"bucket":      m.DefaultBucket,
		"files":       m.files,
		"file_count":  len(m.files),
	}

	return json.MarshalIndent(exportData, "", "  ")
}
