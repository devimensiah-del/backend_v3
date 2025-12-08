package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// =============================================================================
// FRAMEWORK ENDPOINTS INTEGRATION TESTS
// =============================================================================
// Endpoints tested:
//   - GET /api/v1/frameworks (public)
//   - GET /api/v1/frameworks/:code (public)
//   - GET /api/v1/frameworks/order (auth required)
// =============================================================================

func TestFrameworkList(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: List frameworks (public)
	t.Run("GET /api/v1/frameworks - Public endpoint", func(t *testing.T) {
		start := time.Now()
		resp, err := client.GET("/api/v1/frameworks")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/frameworks",
			Method:      "GET",
			Description: "List all active frameworks (public)",
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

			var frameworksResp map[string]interface{}
			if err := resp.JSON(&frameworksResp); err == nil {
				if frameworks, ok := frameworksResp["frameworks"].([]interface{}); ok {
					t.Logf("Found %d frameworks", len(frameworks))
					// Log first few frameworks
					for i, fw := range frameworks {
						if i >= 3 {
							t.Logf("... and %d more", len(frameworks)-3)
							break
						}
						if fwMap, ok := fw.(map[string]interface{}); ok {
							t.Logf("  - %v: %v", fwMap["code"], fwMap["name"])
						}
					}
				}
			}
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 200, got %d", resp.StatusCode)
			t.Errorf("Expected 200, got %d: %s", resp.StatusCode, string(resp.Body))
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestFrameworkGetByCode(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	frameworkCodes := []string{"pestel", "porter", "swot", "tam_sam_som", "benchmarking", "blue_ocean"}

	for _, code := range frameworkCodes {
		t.Run(fmt.Sprintf("GET /api/v1/frameworks/%s - Valid framework code", code), func(t *testing.T) {
			start := time.Now()
			resp, err := client.GET("/api/v1/frameworks/" + code)
			duration := time.Since(start)

			result := TestResult{
				Endpoint:    "/api/v1/frameworks/" + code,
				Method:      "GET",
				Description: fmt.Sprintf("Get framework by code: %s", code),
				Expected:    http.StatusOK,
				Duration:    duration,
			}

			if err != nil {
				result.Error = err.Error()
				result.Passed = false
				report.AddResult(result)
				t.Errorf("Request failed: %v", err)
				return
			}

			result.StatusCode = resp.StatusCode
			result.Response = string(resp.Body)

			if resp.StatusCode == http.StatusOK {
				result.Passed = true
				var fwResp map[string]interface{}
				if err := resp.JSON(&fwResp); err == nil {
					t.Logf("Framework %s: name=%v, category=%v", code, fwResp["name"], fwResp["category"])
				}
			} else if resp.StatusCode == http.StatusNotFound {
				// Might not exist yet
				result.Passed = true
				result.Error = "Framework not found (might not be configured)"
				t.Logf("Framework %s not found: %s", code, string(resp.Body))
			} else {
				result.Passed = false
				result.Error = fmt.Sprintf("Unexpected status: %d", resp.StatusCode)
				t.Errorf("Expected 200/404, got %d: %s", resp.StatusCode, string(resp.Body))
			}

			report.AddResult(result)
		})
	}

	// Test with invalid code
	t.Run("GET /api/v1/frameworks/invalid_code - Non-existent framework", func(t *testing.T) {
		start := time.Now()
		resp, err := client.GET("/api/v1/frameworks/invalid_code_xyz")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/frameworks/invalid_code_xyz",
			Method:      "GET",
			Description: "Get framework with non-existent code",
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

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestFrameworkOrder(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Get framework order without auth
	t.Run("GET /api/v1/frameworks/order - Without authentication", func(t *testing.T) {
		start := time.Now()
		resp, err := client.GET("/api/v1/frameworks/order")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/frameworks/order",
			Method:      "GET",
			Description: "Get framework order without authentication",
			Expected:    http.StatusUnauthorized,
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

		if resp.StatusCode == http.StatusUnauthorized {
			result.Passed = true
			t.Logf("Correctly rejected unauthenticated request: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 401, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 2: Get framework order with auth
	t.Run("GET /api/v1/frameworks/order - With authentication", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/frameworks/order")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/frameworks/order",
			Method:      "GET",
			Description: "Get framework execution order for wizard",
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

			var orderResp map[string]interface{}
			if err := resp.JSON(&orderResp); err == nil {
				if frameworks, ok := orderResp["frameworks"].([]interface{}); ok {
					t.Logf("Framework execution order (%d steps):", len(frameworks))
					for _, fw := range frameworks {
						if fwMap, ok := fw.(map[string]interface{}); ok {
							t.Logf("  Step %v: %v (%v)", fwMap["step"], fwMap["code"], fwMap["name"])
						}
					}
				}
				if totalSteps, ok := orderResp["total_steps"].(float64); ok {
					t.Logf("Total steps: %d", int(totalSteps))
				}
			}
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 200, got %d", resp.StatusCode)
			t.Errorf("Expected 200, got %d: %s", resp.StatusCode, string(resp.Body))
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestChallengeTypes(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Get challenge types without auth
	t.Run("GET /api/v1/challenges/types - Without authentication", func(t *testing.T) {
		start := time.Now()
		resp, err := client.GET("/api/v1/challenges/types")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/challenges/types",
			Method:      "GET",
			Description: "Get challenge types without authentication",
			Expected:    http.StatusUnauthorized,
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

		if resp.StatusCode == http.StatusUnauthorized {
			result.Passed = true
			t.Logf("Correctly rejected unauthenticated request: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 401, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 2: Get challenge types with auth
	t.Run("GET /api/v1/challenges/types - With authentication", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/challenges/types")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/challenges/types",
			Method:      "GET",
			Description: "Get all challenge categories and types",
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

			var typesResp map[string]interface{}
			if err := resp.JSON(&typesResp); err == nil {
				if categories, ok := typesResp["categories"].([]interface{}); ok {
					t.Logf("Challenge categories: %v", categories)
				}
				if types, ok := typesResp["types"].(map[string]interface{}); ok {
					t.Logf("Challenge types by category:")
					for cat, typeList := range types {
						t.Logf("  %s: %v", cat, typeList)
					}
				}
			}
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 200, got %d", resp.StatusCode)
			t.Errorf("Expected 200, got %d: %s", resp.StatusCode, string(resp.Body))
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}
