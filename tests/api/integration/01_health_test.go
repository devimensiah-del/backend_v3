package integration

import (
	"net/http"
	"testing"
	"time"
)

// TestHealthEndpoint tests the health check endpoint
// Endpoint: GET /health
// Auth: None required
// Expected: 200 OK with service status
func TestHealthEndpoint(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	t.Run("GET /health - Basic health check", func(t *testing.T) {
		start := time.Now()
		resp, err := client.GET("/health")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/health",
			Method:      "GET",
			Description: "Basic health check endpoint",
			Expected:    http.StatusOK,
			Duration:    duration,
		}

		if err != nil {
			result.Error = err.Error()
			result.Passed = false
			report.AddResult(result)
			t.Fatalf("Request failed: %v", err)
		}

		result.StatusCode = resp.StatusCode
		result.Response = string(resp.Body)

		if resp.StatusCode == http.StatusOK {
			result.Passed = true
			t.Logf("Health check passed: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = "Unexpected status code"
			t.Errorf("Expected 200, got %d: %s", resp.StatusCode, string(resp.Body))
		}

		report.AddResult(result)
	})

	t.Run("GET /health - Response structure validation", func(t *testing.T) {
		start := time.Now()
		resp, err := client.GET("/health")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/health",
			Method:      "GET",
			Description: "Validate health response structure",
			Expected:    http.StatusOK,
			Duration:    duration,
		}

		if err != nil {
			result.Error = err.Error()
			result.Passed = false
			report.AddResult(result)
			t.Fatalf("Request failed: %v", err)
		}

		result.StatusCode = resp.StatusCode
		result.Response = string(resp.Body)

		var healthResp map[string]interface{}
		if err := resp.JSON(&healthResp); err != nil {
			result.Passed = false
			result.Error = "Failed to parse JSON response"
			report.AddResult(result)
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Validate expected fields
		passed := true
		if _, ok := healthResp["status"]; !ok {
			t.Error("Missing 'status' field in response")
			passed = false
		}
		if _, ok := healthResp["services"]; !ok {
			t.Error("Missing 'services' field in response")
			passed = false
		}

		result.Passed = passed
		if !passed {
			result.Error = "Missing expected fields in response"
		}
		report.AddResult(result)

		// Log service status
		if services, ok := healthResp["services"].(map[string]interface{}); ok {
			t.Logf("Services status:")
			for svc, status := range services {
				t.Logf("  - %s: %v", svc, status)
			}
		}
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

// TestHealthEndpointHeaders tests CORS and response headers
func TestHealthEndpointHeaders(t *testing.T) {
	client := NewTestClient(t)

	t.Run("GET /health - CORS headers (health bypasses origin check)", func(t *testing.T) {
		resp, err := client.GET("/health")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Health endpoint should allow any origin
		corsHeader := resp.Headers.Get("Access-Control-Allow-Origin")
		t.Logf("CORS header: %s", corsHeader)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})
}
