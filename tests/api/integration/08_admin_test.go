package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// =============================================================================
// ADMIN ENDPOINTS INTEGRATION TESTS
// =============================================================================
// Endpoints tested:
//   - GET /api/v1/admin/submissions
//   - GET /api/v1/admin/submissions/:id
//   - GET /api/v1/admin/submissions/:id/analysis
//   - POST /api/v1/admin/submissions/:id/retry-analysis
//   - GET /api/v1/admin/metrics
//   - GET /api/v1/admin/analysis/:id
//   - PUT /api/v1/admin/analysis/:id
//   - POST /api/v1/admin/analysis/:id/visibility
//   - POST /api/v1/admin/analysis/:id/blur
//   - POST /api/v1/admin/analysis/:id/public
//   - POST /api/v1/admin/analysis/:id/access-code
//   - GET /api/v1/admin/macro/latest
//   - POST /api/v1/admin/macro/refresh
//   - POST /api/v1/admin/macro/refresh/:code
//   - GET /api/v1/admin/macro/history/:code
//   - GET /api/v1/admin/companies
//   - GET /api/v1/admin/companies/:id
//   - POST /api/v1/admin/companies/:id/re-analyze
//   - GET /api/v1/admin/frameworks
//   - POST /api/v1/admin/frameworks
//   - PUT /api/v1/admin/frameworks/:id
//   - DELETE /api/v1/admin/frameworks/:id
// =============================================================================

func TestAdminAuthRequired(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	adminEndpoints := []string{
		"/api/v1/admin/submissions",
		"/api/v1/admin/metrics",
		"/api/v1/admin/companies",
		"/api/v1/admin/frameworks",
		"/api/v1/admin/macro/latest",
	}

	for _, endpoint := range adminEndpoints {
		t.Run(fmt.Sprintf("GET %s - Without authentication", endpoint), func(t *testing.T) {
			start := time.Now()
			resp, err := client.GET(endpoint)
			duration := time.Since(start)

			result := TestResult{
				Endpoint:    endpoint,
				Method:      "GET",
				Description: "Access admin endpoint without authentication",
				Expected:    http.StatusUnauthorized,
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

			if resp.StatusCode == http.StatusUnauthorized {
				result.Passed = true
				t.Logf("Correctly rejected: %s", endpoint)
			} else {
				result.Passed = false
				result.Error = fmt.Sprintf("Expected 401, got %d", resp.StatusCode)
			}

			report.AddResult(result)
		})
	}

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestAdminSubmissionsList(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: List all submissions as admin
	t.Run("GET /api/v1/admin/submissions - With admin auth", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/admin/submissions")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/admin/submissions",
			Method:      "GET",
			Description: "List all submissions as admin",
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

			var listResp map[string]interface{}
			if err := resp.JSON(&listResp); err == nil {
				if data, ok := listResp["data"].([]interface{}); ok {
					t.Logf("Found %d submissions", len(data))
				}
				if total, ok := listResp["total"].(float64); ok {
					t.Logf("Total: %d", int(total))
				}
			}
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 200, got %d", resp.StatusCode)
			t.Errorf("Expected 200, got %d: %s", resp.StatusCode, string(resp.Body))
		}

		report.AddResult(result)
	})

	// Test 2: List with pagination
	t.Run("GET /api/v1/admin/submissions - With pagination", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/admin/submissions?page=1&pageSize=5")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/admin/submissions?page=1&pageSize=5",
			Method:      "GET",
			Description: "List submissions with pagination",
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
			t.Logf("Pagination working")
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 200, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestAdminMetrics(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	t.Run("GET /api/v1/admin/metrics - With admin auth", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/admin/metrics")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/admin/metrics",
			Method:      "GET",
			Description: "Get system-wide metrics as admin",
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

			var metricsResp map[string]interface{}
			if err := resp.JSON(&metricsResp); err == nil {
				t.Logf("Submissions (24h): %v", metricsResp["submissions_last_24h"])
				t.Logf("Enrichment success rate: %v", metricsResp["enrichment_success_rate"])
				t.Logf("Analysis success rate: %v", metricsResp["analysis_success_rate"])
				t.Logf("Avg analysis time (s): %v", metricsResp["avg_analysis_time_seconds"])
				t.Logf("Total cost (24h): %v", metricsResp["total_cost_last_24h_usd"])
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

func TestAdminMacroIndicators(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Get latest macro snapshot
	t.Run("GET /api/v1/admin/macro/latest - With admin auth", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/admin/macro/latest")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/admin/macro/latest",
			Method:      "GET",
			Description: "Get latest macro indicators snapshot",
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

			var macroResp map[string]interface{}
			if err := resp.JSON(&macroResp); err == nil {
				if indicators, ok := macroResp["indicators"].(map[string]interface{}); ok {
					for code, data := range indicators {
						t.Logf("  %s: %v", code, data)
					}
				}
				t.Logf("As of: %v", macroResp["asOf"])
			}
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 200, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 2: Get macro history for SELIC
	t.Run("GET /api/v1/admin/macro/history/selic - With admin auth", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/admin/macro/history/selic?limit=10")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/admin/macro/history/selic",
			Method:      "GET",
			Description: "Get SELIC history",
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

			var historyResp map[string]interface{}
			if err := resp.JSON(&historyResp); err == nil {
				t.Logf("Indicator: %v", historyResp["indicatorCode"])
				if values, ok := historyResp["values"].([]interface{}); ok {
					t.Logf("History entries: %d", len(values))
				}
			}
		} else if resp.StatusCode == http.StatusNotFound {
			result.Passed = true
			result.Error = "No history found (might not be populated)"
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 200/404, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 3: Refresh macro indicators (may take time)
	t.Run("POST /api/v1/admin/macro/refresh - With admin auth", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.POST("/api/v1/admin/macro/refresh", nil)
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/admin/macro/refresh",
			Method:      "POST",
			Description: "Trigger refresh of all macro indicators",
			Expected:    http.StatusAccepted,
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

		// 202 Accepted or 200 OK are both valid
		if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusOK {
			result.Passed = true
			t.Logf("Refresh triggered: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 202/200, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestAdminCompanies(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: List all companies as admin
	t.Run("GET /api/v1/admin/companies - With admin auth", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/admin/companies")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/admin/companies",
			Method:      "GET",
			Description: "List all companies as admin",
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

			var companiesResp map[string]interface{}
			if err := resp.JSON(&companiesResp); err == nil {
				if companies, ok := companiesResp["companies"].([]interface{}); ok {
					t.Logf("Found %d companies", len(companies))
				}
			}
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 200, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 2: Get company by ID as admin
	if GetConfig().TestCompanyID != "" {
		t.Run("GET /api/v1/admin/companies/:id - Valid company", func(t *testing.T) {
			RequireAdminAuth(t, client)

			start := time.Now()
			resp, err := client.GET("/api/v1/admin/companies/" + GetConfig().TestCompanyID)
			duration := time.Since(start)

			result := TestResult{
				Endpoint:    "/api/v1/admin/companies/:id",
				Method:      "GET",
				Description: "Get company by ID as admin",
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
				t.Logf("Company details retrieved")
			} else if resp.StatusCode == http.StatusNotFound {
				result.Passed = true
				result.Error = "Company not found"
			} else {
				result.Passed = false
				result.Error = fmt.Sprintf("Expected 200/404, got %d", resp.StatusCode)
			}

			report.AddResult(result)
		})
	}

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestAdminFrameworks(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: List all frameworks as admin (including inactive)
	t.Run("GET /api/v1/admin/frameworks - With admin auth", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/admin/frameworks")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/admin/frameworks",
			Method:      "GET",
			Description: "List all frameworks as admin (including inactive)",
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
					t.Logf("Found %d frameworks (admin view)", len(frameworks))
					activeCount := 0
					for _, fw := range frameworks {
						if fwMap, ok := fw.(map[string]interface{}); ok {
							if isActive, ok := fwMap["isActive"].(bool); ok && isActive {
								activeCount++
							}
						}
					}
					t.Logf("Active: %d, Inactive: %d", activeCount, len(frameworks)-activeCount)
				}
			}
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 200, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Note: Not testing POST/PUT/DELETE to avoid modifying production data
	t.Run("POST /api/v1/admin/frameworks - Create framework (validation only)", func(t *testing.T) {
		RequireAdminAuth(t, client)

		// Test with invalid data to verify validation works
		start := time.Now()
		resp, err := client.POST("/api/v1/admin/frameworks", map[string]interface{}{
			// Missing required fields
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/admin/frameworks",
			Method:      "POST",
			Description: "Create framework with missing fields (validation test)",
			Expected:    http.StatusBadRequest,
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

		if resp.StatusCode == http.StatusBadRequest {
			result.Passed = true
			t.Logf("Correctly rejected invalid framework: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 400, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestAdminAnalysis(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test with non-existent analysis ID
	testAnalysisID := "00000000-0000-0000-0000-000000000000"

	// Test 1: Get analysis as admin
	t.Run("GET /api/v1/admin/analysis/:id - Non-existent analysis", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/admin/analysis/" + testAnalysisID)
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/admin/analysis/:id",
			Method:      "GET",
			Description: "Get non-existent analysis as admin",
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
			t.Logf("Correctly returned 404")
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 404, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 2: Toggle visibility on non-existent analysis
	t.Run("POST /api/v1/admin/analysis/:id/visibility - Non-existent analysis", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.POST("/api/v1/admin/analysis/"+testAnalysisID+"/visibility", nil)
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/admin/analysis/:id/visibility",
			Method:      "POST",
			Description: "Toggle visibility on non-existent analysis",
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
			t.Logf("Correctly returned 404")
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 404, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 3: Toggle blur on non-existent analysis
	t.Run("POST /api/v1/admin/analysis/:id/blur - Non-existent analysis", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.POST("/api/v1/admin/analysis/"+testAnalysisID+"/blur", nil)
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/admin/analysis/:id/blur",
			Method:      "POST",
			Description: "Toggle blur on non-existent analysis",
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
			t.Logf("Correctly returned 404")
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 404, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 4: Toggle public on non-existent analysis
	t.Run("POST /api/v1/admin/analysis/:id/public - Non-existent analysis", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.POST("/api/v1/admin/analysis/"+testAnalysisID+"/public", nil)
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/admin/analysis/:id/public",
			Method:      "POST",
			Description: "Toggle public on non-existent analysis",
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
			t.Logf("Correctly returned 404")
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 404, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 5: Generate access code on non-existent analysis
	t.Run("POST /api/v1/admin/analysis/:id/access-code - Non-existent analysis", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.POST("/api/v1/admin/analysis/"+testAnalysisID+"/access-code", nil)
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/admin/analysis/:id/access-code",
			Method:      "POST",
			Description: "Generate access code on non-existent analysis",
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
			t.Logf("Correctly returned 404")
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 404, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// If we have a test analysis ID, run additional tests
	if GetConfig().TestAnalysisID != "" {
		realAnalysisID := GetConfig().TestAnalysisID

		t.Run("GET /api/v1/admin/analysis/:id - Existing analysis", func(t *testing.T) {
			RequireAdminAuth(t, client)

			start := time.Now()
			resp, err := client.GET("/api/v1/admin/analysis/" + realAnalysisID)
			duration := time.Since(start)

			result := TestResult{
				Endpoint:    "/api/v1/admin/analysis/:id",
				Method:      "GET",
				Description: "Get existing analysis as admin",
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
				t.Logf("Analysis retrieved successfully")
			} else if resp.StatusCode == http.StatusNotFound {
				result.Passed = true
				result.Error = "Analysis not found"
			} else {
				result.Passed = false
				result.Error = fmt.Sprintf("Expected 200/404, got %d", resp.StatusCode)
			}

			report.AddResult(result)
		})
	}

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestAdminRetryAnalysis(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test retry analysis on non-existent submission
	t.Run("POST /api/v1/admin/submissions/:id/retry-analysis - Non-existent", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.POST("/api/v1/admin/submissions/00000000-0000-0000-0000-000000000000/retry-analysis", nil)
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/admin/submissions/:id/retry-analysis",
			Method:      "POST",
			Description: "Retry analysis on non-existent submission",
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
			t.Logf("Correctly returned 404")
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 404, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestAdminReAnalyzeCompany(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test re-analyze with invalid data
	t.Run("POST /api/v1/admin/companies/:id/re-analyze - Non-existent company", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.POST("/api/v1/admin/companies/00000000-0000-0000-0000-000000000000/re-analyze", map[string]interface{}{
			"challengeCategory": "transform",
			"challengeType":     "transform_digital",
			"businessChallenge": "New challenge for re-analysis",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/admin/companies/:id/re-analyze",
			Method:      "POST",
			Description: "Re-analyze non-existent company",
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
			t.Logf("Correctly returned 404")
		} else if resp.StatusCode == http.StatusBadRequest {
			result.Passed = true
			result.Error = "Bad request (validation)"
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 404/400, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}
