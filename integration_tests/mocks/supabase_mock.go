package mocks

import (
	"context"
	"fmt"
	"sync"
)

// MockSupabaseStorage is an in-memory mock of Supabase Storage
type MockSupabaseStorage struct {
	// In-memory storage
	files map[string][]byte
	mu    sync.RWMutex

	// Configuration
	ProjectURL string
	Bucket     string
	Token      string
}

// NewMockSupabaseStorage creates a new mock Supabase storage client
func NewMockSupabaseStorage() *MockSupabaseStorage {
	return &MockSupabaseStorage{
		files:      make(map[string][]byte),
		ProjectURL: "https://mock-project.supabase.co",
		Bucket:     "reports",
		Token:      "mock-service-role-key",
	}
}

// Upload stores the file in memory and returns a mock public URL
func (m *MockSupabaseStorage) Upload(ctx context.Context, path string, data []byte, contentType string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store in memory
	m.files[path] = data

	// Return mock public URL
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", m.ProjectURL, m.Bucket, path)
	return publicURL, nil
}

// GetFile retrieves a file from in-memory storage (helper for testing)
func (m *MockSupabaseStorage) GetFile(path string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.files[path]
	return data, exists
}

// ListFiles returns all stored file paths (helper for testing)
func (m *MockSupabaseStorage) ListFiles() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	paths := make([]string, 0, len(m.files))
	for path := range m.files {
		paths = append(paths, path)
	}
	return paths
}

// Clear removes all files from storage (helper for test cleanup)
func (m *MockSupabaseStorage) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.files = make(map[string][]byte)
}

// FileCount returns the number of stored files (helper for testing)
func (m *MockSupabaseStorage) FileCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.files)
}
