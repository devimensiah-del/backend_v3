package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// =============================================================================
// WIZARD ENDPOINTS INTEGRATION TESTS
// =============================================================================
// Endpoints tested:
//   - POST /api/v1/wizard/start (auth required) - takes company_id + challenge_id
//   - GET /api/v1/analyses/:id/wizard (auth required)
//   - POST /api/v1/analyses/:id/wizard/generate (auth required)
//   - POST /api/v1/analyses/:id/wizard/approve (auth required)
//   - POST /api/v1/analyses/:id/wizard/refine (auth required)
//   - GET /api/v1/analyses/:id/wizard/summary (auth required)
// =============================================================================

func TestWizardStart(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Start wizard without auth
	t.Run("POST /api/v1/wizard/start - Without authentication", func(t *testing.T) {
		start := time.Now()
		resp, err := client.POST("/api/v1/wizard/start", map[string]string{
			"company_id":   "00000000-0000-0000-0000-000000000000",
			"challenge_id": "00000000-0000-0000-0000-000000000000",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/wizard/start",
			Method:      "POST",
			Description: "Start wizard without authentication",
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
			t.Logf("Correctly rejected unauthenticated request")
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 401, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 2: Start wizard with missing fields
	t.Run("POST /api/v1/wizard/start - Missing required fields", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.POST("/api/v1/wizard/start", map[string]string{})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/wizard/start",
			Method:      "POST",
			Description: "Start wizard with missing required fields",
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
			t.Logf("Correctly rejected missing fields: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 400, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 3: Start wizard with invalid UUIDs
	t.Run("POST /api/v1/wizard/start - Invalid UUIDs", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.POST("/api/v1/wizard/start", map[string]string{
			"company_id":   "invalid-uuid",
			"challenge_id": "invalid-uuid",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/wizard/start",
			Method:      "POST",
			Description: "Start wizard with invalid UUIDs",
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

		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusInternalServerError {
			result.Passed = true
			t.Logf("Correctly rejected invalid UUIDs: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 400/500, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestWizardGetState(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Get wizard state without auth
	t.Run("GET /api/v1/analyses/:id/wizard - Without authentication", func(t *testing.T) {
		analysisID := GetConfig().TestAnalysisID
		if analysisID == "" {
			analysisID = "00000000-0000-0000-0000-000000000000"
		}

		start := time.Now()
		resp, err := client.GET("/api/v1/analyses/" + analysisID + "/wizard")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/analyses/:id/wizard",
			Method:      "GET",
			Description: "Get wizard state without authentication",
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
			t.Logf("Correctly rejected unauthenticated request")
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 401, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 2: Get wizard state for non-existent analysis
	t.Run("GET /api/v1/analyses/:id/wizard - Non-existent analysis", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/analyses/00000000-0000-0000-0000-000000000000/wizard")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/analyses/.../wizard",
			Method:      "GET",
			Description: "Get wizard state for non-existent analysis",
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

		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
			result.Passed = true
			t.Logf("Correctly returned error: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 404/400, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestWizardGenerate(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Generate step without auth
	t.Run("POST /api/v1/analyses/:id/wizard/generate - Without authentication", func(t *testing.T) {
		analysisID := GetConfig().TestAnalysisID
		if analysisID == "" {
			analysisID = "00000000-0000-0000-0000-000000000000"
		}

		start := time.Now()
		resp, err := client.POST("/api/v1/analyses/"+analysisID+"/wizard/generate", map[string]interface{}{
			"humanContext": "Test context",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/analyses/:id/wizard/generate",
			Method:      "POST",
			Description: "Generate wizard step without authentication",
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
			t.Logf("Correctly rejected unauthenticated request")
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 401, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 2: Generate step for non-existent analysis
	t.Run("POST /api/v1/analyses/:id/wizard/generate - Non-existent analysis", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.POST("/api/v1/analyses/00000000-0000-0000-0000-000000000000/wizard/generate", map[string]interface{}{
			"humanContext": "Test context",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/analyses/.../wizard/generate",
			Method:      "POST",
			Description: "Generate step for non-existent analysis",
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

		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusInternalServerError {
			result.Passed = true
			t.Logf("Correctly returned error: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 404/400/500, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestWizardApprove(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Approve step without auth
	t.Run("POST /api/v1/analyses/:id/wizard/approve - Without authentication", func(t *testing.T) {
		analysisID := GetConfig().TestAnalysisID
		if analysisID == "" {
			analysisID = "00000000-0000-0000-0000-000000000000"
		}

		start := time.Now()
		resp, err := client.POST("/api/v1/analyses/"+analysisID+"/wizard/approve", nil)
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/analyses/:id/wizard/approve",
			Method:      "POST",
			Description: "Approve wizard step without authentication",
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
			t.Logf("Correctly rejected unauthenticated request")
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 401, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestWizardRefine(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Refine step without auth
	t.Run("POST /api/v1/analyses/:id/wizard/refine - Without authentication", func(t *testing.T) {
		analysisID := GetConfig().TestAnalysisID
		if analysisID == "" {
			analysisID = "00000000-0000-0000-0000-000000000000"
		}

		start := time.Now()
		resp, err := client.POST("/api/v1/analyses/"+analysisID+"/wizard/refine", map[string]interface{}{
			"humanContext": "Please focus on...",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/analyses/:id/wizard/refine",
			Method:      "POST",
			Description: "Refine wizard step without authentication",
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
			t.Logf("Correctly rejected unauthenticated request")
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 401, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestWizardSummary(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Get summary without auth
	t.Run("GET /api/v1/analyses/:id/wizard/summary - Without authentication", func(t *testing.T) {
		analysisID := GetConfig().TestAnalysisID
		if analysisID == "" {
			analysisID = "00000000-0000-0000-0000-000000000000"
		}

		start := time.Now()
		resp, err := client.GET("/api/v1/analyses/" + analysisID + "/wizard/summary")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/analyses/:id/wizard/summary",
			Method:      "GET",
			Description: "Get wizard summary without authentication",
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
			t.Logf("Correctly rejected unauthenticated request")
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 401, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 2: Get summary for non-existent analysis
	t.Run("GET /api/v1/analyses/:id/wizard/summary - Non-existent analysis", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/analyses/00000000-0000-0000-0000-000000000000/wizard/summary")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/analyses/:id/wizard/summary",
			Method:      "GET",
			Description: "Get wizard summary for non-existent analysis",
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
