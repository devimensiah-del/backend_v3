package mocks

import (
	"backend_v3/adapter/scraper"
	"context"
	"fmt"
	"strings"
)

// MockScraperClient is a mock implementation of scraper.Client
type MockScraperClient struct {
	// Predefined responses for known domains
	responses map[string]scraper.MetaData
	// Default response for unknown domains
	defaultResponse scraper.MetaData
	// Track scrape calls
	CallCount int
}

// NewMockScraperClient creates a new mock scraper client with default responses
func NewMockScraperClient() *scraper.Client {
	// Note: We can't directly mock the scraper.Client since it's a concrete type,
	// but we can create a wrapper or return a client that won't make real HTTP calls.
	// For now, we'll create a mock client structure.

	// Since scraper.Client doesn't have an interface, we'll need to modify our approach.
	// The best approach is to return a scraper.Client but recognize that it will fail
	// gracefully in tests when no network is available.
	return scraper.NewClient()
}

// CreateMockScraperWithResponses creates a mock with predefined responses
func CreateMockScraperWithResponses() *MockScraperClient {
	mock := &MockScraperClient{
		responses: make(map[string]scraper.MetaData),
		defaultResponse: scraper.MetaData{
			Title:       "Mock Company",
			Description: "A mock company for testing purposes",
			Keywords:    []string{"mock", "testing", "company"},
			Language:    "pt-BR",
			OGImage:     "https://mock-company.com/image.png",
		},
		CallCount: 0,
	}

	// Add some predefined responses for common test domains
	mock.responses["testcompany.com"] = scraper.MetaData{
		Title:       "Test Company - Cloud Solutions",
		Description: "Leading provider of cloud-based enterprise solutions",
		Keywords:    []string{"cloud", "enterprise", "saas", "technology"},
		Language:    "pt-BR",
		OGImage:     "https://testcompany.com/og-image.png",
	}

	mock.responses["example.com"] = scraper.MetaData{
		Title:       "Example Corporation",
		Description: "Example company for testing",
		Keywords:    []string{"example", "test"},
		Language:    "en",
		OGImage:     "https://example.com/logo.png",
	}

	return mock
}

// Scrape returns mock metadata for the given URL
func (m *MockScraperClient) Scrape(ctx context.Context, url string) (scraper.MetaData, error) {
	m.CallCount++

	// Extract domain from URL
	domain := extractDomain(url)

	// Check if we have a predefined response for this domain
	if data, exists := m.responses[domain]; exists {
		return data, nil
	}

	// Return default response
	return m.defaultResponse, nil
}

// AddResponse adds a custom response for a specific domain
func (m *MockScraperClient) AddResponse(domain string, data scraper.MetaData) {
	m.responses[domain] = data
}

// SetDefaultResponse sets the default response for unknown domains
func (m *MockScraperClient) SetDefaultResponse(data scraper.MetaData) {
	m.defaultResponse = data
}

// extractDomain extracts the domain from a URL
func extractDomain(url string) string {
	// Remove protocol
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")

	// Split by / and take first part
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[0]
	}

	return url
}

// GetCallCount returns the number of times Scrape was called
func (m *MockScraperClient) GetCallCount() int {
	return m.CallCount
}

// Reset resets the call count and clears custom responses
func (m *MockScraperClient) Reset() {
	m.CallCount = 0
	m.responses = make(map[string]scraper.MetaData)
}

// MockScraperError simulates a scraper error
type MockScraperError struct {
	Message string
}

func (e *MockScraperError) Error() string {
	return fmt.Sprintf("mock scraper error: %s", e.Message)
}
