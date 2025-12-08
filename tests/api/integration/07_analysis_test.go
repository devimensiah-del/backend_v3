package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// =============================================================================
// ANALYSIS ENDPOINTS INTEGRATION TESTS
// =============================================================================
// Endpoints tested:
//   - GET /api/v1/public/report/:code (optional auth - public report)
// =============================================================================

func TestPublicReport(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Get public report with invalid code
	t.Run("GET /api/v1/public/report/:code - Invalid access code", func(t *testing.T) {
		start := time.Now()
		resp, err := client.GET("/api/v1/public/report/INVALID123")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/public/report/INVALID123",
			Method:      "GET",
			Description: "Get public report with invalid access code",
			Expected:    http.StatusNotFound,
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

		if resp.StatusCode == http.StatusNotFound {
			result.Passed = true
			t.Logf("Correctly returned 404: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 404, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 2: Get public report with empty code
	t.Run("GET /api/v1/public/report/:code - Empty access code", func(t *testing.T) {
		start := time.Now()
		resp, err := client.GET("/api/v1/public/report/")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/public/report/",
			Method:      "GET",
			Description: "Get public report with empty access code",
			Expected:    http.StatusNotFound,
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

		// Empty path should 404 (route not matched)
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMovedPermanently {
			result.Passed = true
			t.Logf("Correctly returned %d", resp.StatusCode)
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 404/301, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 3: Get public report with very long code (boundary test)
	t.Run("GET /api/v1/public/report/:code - Very long access code", func(t *testing.T) {
		// Create a long code with valid ASCII characters (not null bytes)
		longCode := "ABCD"
		for i := 0; i < 100; i++ {
			longCode += "XXXXXXXXXX" // 1000 characters total
		}

		start := time.Now()
		resp, err := client.GET("/api/v1/public/report/" + longCode)
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/public/report/[long_code]",
			Method:      "GET",
			Description: "Get public report with very long access code",
			Expected:    http.StatusNotFound,
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

		// Should return 404 (not found) or 400 (bad request for invalid input)
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
			result.Passed = true
			t.Logf("Correctly handled long code: %d", resp.StatusCode)
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 404/400, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 4: Get public report with SQL injection attempt
	t.Run("GET /api/v1/public/report/:code - SQL injection attempt", func(t *testing.T) {
		start := time.Now()
		resp, err := client.GET("/api/v1/public/report/'; DROP TABLE analyses; --")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/public/report/[sql_injection]",
			Method:      "GET",
			Description: "Get public report with SQL injection attempt",
			Expected:    http.StatusNotFound,
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

		// Should safely return 404, not error
		if resp.StatusCode == http.StatusNotFound {
			result.Passed = true
			t.Logf("Safely handled SQL injection attempt")
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 404, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 5: Admin preview with authentication
	t.Run("GET /api/v1/public/report/:code - Admin preview (with auth)", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/public/report/TESTCODE")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/public/report/TESTCODE",
			Method:      "GET",
			Description: "Get public report with admin auth (preview mode)",
			Expected:    http.StatusNotFound,
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

		// Should still return 404 if code doesn't exist
		if resp.StatusCode == http.StatusNotFound {
			result.Passed = true
			t.Logf("Admin preview correctly returned 404 for non-existent code")
		} else if resp.StatusCode == http.StatusOK {
			result.Passed = true
			t.Logf("Admin preview found analysis")
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 404/200, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}
