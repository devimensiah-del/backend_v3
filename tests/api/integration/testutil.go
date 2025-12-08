// Package integration provides API integration test utilities
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

// TestConfig holds configuration for integration tests
type TestConfig struct {
	BaseURL       string
	AdminEmail    string
	AdminPassword string
	AdminToken    string

	// Cached test data
	TestSubmissionID string
	TestCompanyID    string
	TestAnalysisID   string
	TestChallengeID  string
}

var (
	config     *TestConfig
	configOnce sync.Once
)

// GetConfig returns the singleton test configuration
func GetConfig() *TestConfig {
	configOnce.Do(func() {
		config = &TestConfig{
			BaseURL:       getEnv("TEST_API_URL", "http://localhost:8080"),
			AdminEmail:    getEnv("TEST_ADMIN_EMAIL", "renatodaprado@gmail.com"),
			AdminPassword: getEnv("TEST_ADMIN_PASSWORD", "B3nt02o12!?!?!?"),
		}
	})
	return config
}

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// TestClient wraps http.Client for API testing
type TestClient struct {
	*http.Client
	BaseURL string
	Token   string
	T       *testing.T
}

// NewTestClient creates a new test client
func NewTestClient(t *testing.T) *TestClient {
	return &TestClient{
		Client:  &http.Client{Timeout: 30 * time.Second},
		BaseURL: GetConfig().BaseURL,
		T:       t,
	}
}

// WithToken sets the auth token for requests
func (c *TestClient) WithToken(token string) *TestClient {
	c.Token = token
	return c
}

// APIResponse represents a generic API response
type APIResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// JSON parses the response body as JSON into the given interface
func (r *APIResponse) JSON(v interface{}) error {
	return json.Unmarshal(r.Body, v)
}

// Request makes an HTTP request to the API
func (c *TestClient) Request(method, path string, body interface{}) (*APIResponse, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	url := c.BaseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:3000") // Required for CORS
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return &APIResponse{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Headers:    resp.Header,
	}, nil
}

// GET makes a GET request
func (c *TestClient) GET(path string) (*APIResponse, error) {
	return c.Request(http.MethodGet, path, nil)
}

// POST makes a POST request
func (c *TestClient) POST(path string, body interface{}) (*APIResponse, error) {
	return c.Request(http.MethodPost, path, body)
}

// PUT makes a PUT request
func (c *TestClient) PUT(path string, body interface{}) (*APIResponse, error) {
	return c.Request(http.MethodPut, path, body)
}

// DELETE makes a DELETE request
func (c *TestClient) DELETE(path string) (*APIResponse, error) {
	return c.Request(http.MethodDelete, path, nil)
}

// TestResult documents a test result
type TestResult struct {
	Endpoint    string
	Method      string
	Description string
	StatusCode  int
	Expected    int
	Passed      bool
	Error       string
	Response    string
	Duration    time.Duration
}

// TestReport collects all test results
type TestReport struct {
	Results   []TestResult
	StartTime time.Time
	EndTime   time.Time
	mu        sync.Mutex
}

// NewTestReport creates a new test report
func NewTestReport() *TestReport {
	return &TestReport{
		Results:   make([]TestResult, 0),
		StartTime: time.Now(),
	}
}

// AddResult adds a test result to the report
func (r *TestReport) AddResult(result TestResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Results = append(r.Results, result)
}

// Summary returns a summary of test results
func (r *TestReport) Summary() string {
	r.EndTime = time.Now()
	passed := 0
	failed := 0
	for _, res := range r.Results {
		if res.Passed {
			passed++
		} else {
			failed++
		}
	}

	return fmt.Sprintf(`
========================================
API INTEGRATION TEST REPORT
========================================
Total Tests: %d
Passed: %d
Failed: %d
Duration: %v
========================================
`, len(r.Results), passed, failed, r.EndTime.Sub(r.StartTime))
}

// PrintDetails prints detailed results for failed tests
func (r *TestReport) PrintDetails() string {
	var output bytes.Buffer
	output.WriteString("\n=== DETAILED RESULTS ===\n\n")

	for _, res := range r.Results {
		status := "PASS"
		if !res.Passed {
			status = "FAIL"
		}
		output.WriteString(fmt.Sprintf("[%s] %s %s\n", status, res.Method, res.Endpoint))
		output.WriteString(fmt.Sprintf("    Description: %s\n", res.Description))
		output.WriteString(fmt.Sprintf("    Expected: %d, Got: %d\n", res.Expected, res.StatusCode))
		output.WriteString(fmt.Sprintf("    Duration: %v\n", res.Duration))
		if !res.Passed {
			output.WriteString(fmt.Sprintf("    Error: %s\n", res.Error))
			if res.Response != "" {
				respPreview := res.Response
				if len(respPreview) > 500 {
					respPreview = respPreview[:500] + "..."
				}
				output.WriteString(fmt.Sprintf("    Response: %s\n", respPreview))
			}
		}
		output.WriteString("\n")
	}

	return output.String()
}

// LoginResponse represents the auth login response
type LoginResponse struct {
	User        map[string]interface{} `json:"user"`
	AccessToken string                 `json:"access_token"`
	Token       string                 `json:"token"`
	TokenType   string                 `json:"token_type"`
	ExpiresIn   int                    `json:"expires_in"`
}

// GetAdminToken logs in as admin and returns the token
func GetAdminToken(client *TestClient) (string, error) {
	cfg := GetConfig()

	// Check if we already have a token
	if cfg.AdminToken != "" {
		return cfg.AdminToken, nil
	}

	resp, err := client.POST("/api/v1/auth/login", map[string]string{
		"email":    cfg.AdminEmail,
		"password": cfg.AdminPassword,
	})
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(resp.Body))
	}

	var loginResp LoginResponse
	if err := resp.JSON(&loginResp); err != nil {
		return "", fmt.Errorf("parse login response: %w", err)
	}

	token := loginResp.AccessToken
	if token == "" {
		token = loginResp.Token
	}

	if token == "" {
		return "", fmt.Errorf("no token in response: %s", string(resp.Body))
	}

	cfg.AdminToken = token
	return token, nil
}

// RequireAdminAuth sets up the client with admin authentication
func RequireAdminAuth(t *testing.T, client *TestClient) {
	token, err := GetAdminToken(client)
	if err != nil {
		t.Fatalf("Failed to get admin token: %v", err)
	}
	client.WithToken(token)
}

// AssertStatus checks if the response status matches expected
func AssertStatus(t *testing.T, resp *APIResponse, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Errorf("Expected status %d, got %d. Body: %s", expected, resp.StatusCode, string(resp.Body))
	}
}

// AssertJSON checks if the response contains expected JSON fields
func AssertJSON(t *testing.T, resp *APIResponse, key string) {
	t.Helper()
	var data map[string]interface{}
	if err := resp.JSON(&data); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
		return
	}
	if _, ok := data[key]; !ok {
		t.Errorf("Expected JSON key '%s' not found in response: %s", key, string(resp.Body))
	}
}
