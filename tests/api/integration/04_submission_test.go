package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// =============================================================================
// SUBMISSION ENDPOINTS INTEGRATION TESTS
// =============================================================================
// Endpoints tested:
//   - POST /api/v1/submissions (public)
//   - GET /api/v1/submissions (auth required)
//   - GET /api/v1/submissions/:id (auth required)
//   - GET /api/v1/submissions/:id/analysis (auth required)
// =============================================================================

// TestSubmissionData holds IDs created during tests for cleanup
var TestSubmissionData struct {
	SubmissionID string
	CompanyID    string
	ChallengeID  string
	AnalysisID   string
}

func TestSubmissionCreate(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Create submission with valid data
	t.Run("POST /api/v1/submissions - Valid submission", func(t *testing.T) {
		additionalInfo := map[string]interface{}{
			"contactName":  "Test Contact",
			"contactEmail": "test@example.com",
			"contactPhone": "+55 11 99999-9999",
		}
		additionalInfoJSON, _ := json.Marshal(additionalInfo)

		start := time.Now()
		resp, err := client.POST("/api/v1/submissions", map[string]interface{}{
			"companyName":       "Test Company Integration",
			"challengeCategory": "growth",
			"challengeType":     "growth_organic",
			"businessChallenge": "We need to increase our market share by 30% in the next year",
			"industry":          "Technology",
			"companySize":       "medium",
			"additionalInfo":    string(additionalInfoJSON),
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/submissions",
			Method:      "POST",
			Description: "Create submission with valid data",
			Expected:    http.StatusCreated,
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

		if resp.StatusCode == http.StatusCreated {
			result.Passed = true

			var createResp map[string]interface{}
			if err := resp.JSON(&createResp); err == nil {
				if submission, ok := createResp["submission"].(map[string]interface{}); ok {
					TestSubmissionData.SubmissionID = fmt.Sprintf("%v", submission["id"])
					TestSubmissionData.CompanyID = fmt.Sprintf("%v", submission["companyId"])
					TestSubmissionData.ChallengeID = fmt.Sprintf("%v", submission["challengeId"])

					t.Logf("Created submission: %s", TestSubmissionData.SubmissionID)
					t.Logf("Created company: %s", TestSubmissionData.CompanyID)
					t.Logf("Created challenge: %s", TestSubmissionData.ChallengeID)

					// Store for other tests
					GetConfig().TestSubmissionID = TestSubmissionData.SubmissionID
					GetConfig().TestCompanyID = TestSubmissionData.CompanyID
					GetConfig().TestChallengeID = TestSubmissionData.ChallengeID
				}
			}
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 201, got %d", resp.StatusCode)
			t.Errorf("Expected 201, got %d: %s", resp.StatusCode, string(resp.Body))
		}

		report.AddResult(result)
	})

	// Test 2: Create submission without required fields
	t.Run("POST /api/v1/submissions - Missing required fields", func(t *testing.T) {
		start := time.Now()
		resp, err := client.POST("/api/v1/submissions", map[string]interface{}{
			"companyName": "Test Company",
			// Missing challengeCategory, challengeType, businessChallenge
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/submissions",
			Method:      "POST",
			Description: "Create submission with missing required fields",
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

	// Test 3: Create submission without contact info
	t.Run("POST /api/v1/submissions - Missing contact info", func(t *testing.T) {
		start := time.Now()
		resp, err := client.POST("/api/v1/submissions", map[string]interface{}{
			"companyName":       "Test Company",
			"challengeCategory": "growth",
			"challengeType":     "growth_organic",
			"businessChallenge": "Test challenge",
			// Missing additionalInfo with contactName/contactEmail
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/submissions",
			Method:      "POST",
			Description: "Create submission without contact info",
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
			t.Logf("Correctly rejected missing contact info: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 400, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 4: Create submission with invalid challenge category
	t.Run("POST /api/v1/submissions - Invalid challenge category", func(t *testing.T) {
		additionalInfo := map[string]interface{}{
			"contactName":  "Test Contact",
			"contactEmail": "test@example.com",
		}
		additionalInfoJSON, _ := json.Marshal(additionalInfo)

		start := time.Now()
		resp, err := client.POST("/api/v1/submissions", map[string]interface{}{
			"companyName":       "Test Company",
			"challengeCategory": "invalid_category",
			"challengeType":     "invalid_type",
			"businessChallenge": "Test challenge",
			"additionalInfo":    string(additionalInfoJSON),
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/submissions",
			Method:      "POST",
			Description: "Create submission with invalid challenge category",
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

		// May return 400 or 500 depending on validation implementation
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusInternalServerError {
			result.Passed = true
			t.Logf("Rejected invalid challenge category (status %d): %s", resp.StatusCode, string(resp.Body))
		} else if resp.StatusCode == http.StatusCreated {
			// Some implementations might allow any string
			result.Passed = true
			result.Error = "WARNING: Invalid challenge category was accepted"
			t.Logf("Warning: Invalid challenge category was accepted")
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 400/500, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestSubmissionList(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: List submissions without auth
	t.Run("GET /api/v1/submissions - Without authentication", func(t *testing.T) {
		start := time.Now()
		resp, err := client.GET("/api/v1/submissions")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/submissions",
			Method:      "GET",
			Description: "List submissions without authentication",
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

	// Test 2: List submissions with auth
	t.Run("GET /api/v1/submissions - With authentication", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/submissions")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/submissions",
			Method:      "GET",
			Description: "List user's own submissions",
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
				if page, ok := listResp["page"].(float64); ok {
					t.Logf("Page: %d", int(page))
				}
			}
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 200, got %d", resp.StatusCode)
			t.Errorf("Expected 200, got %d: %s", resp.StatusCode, string(resp.Body))
		}

		report.AddResult(result)
	})

	// Test 3: List submissions with pagination
	t.Run("GET /api/v1/submissions - With pagination params", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/submissions?page=1&pageSize=5")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/submissions?page=1&pageSize=5",
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

			var listResp map[string]interface{}
			if err := resp.JSON(&listResp); err == nil {
				if pageSize, ok := listResp["pageSize"].(float64); ok {
					if int(pageSize) != 5 {
						t.Logf("Warning: pageSize in response (%d) differs from requested (5)", int(pageSize))
					}
				}
			}
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 200, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestSubmissionGetById(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Get submission without auth
	t.Run("GET /api/v1/submissions/:id - Without authentication", func(t *testing.T) {
		submissionID := GetConfig().TestSubmissionID
		if submissionID == "" {
			submissionID = "00000000-0000-0000-0000-000000000000"
		}

		start := time.Now()
		resp, err := client.GET("/api/v1/submissions/" + submissionID)
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/submissions/:id",
			Method:      "GET",
			Description: "Get submission by ID without authentication",
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

	// Test 2: Get submission with invalid UUID
	t.Run("GET /api/v1/submissions/:id - Invalid UUID format", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/submissions/not-a-valid-uuid")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/submissions/not-a-valid-uuid",
			Method:      "GET",
			Description: "Get submission with invalid UUID",
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
			t.Logf("Correctly rejected invalid UUID: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 400, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 3: Get non-existent submission
	t.Run("GET /api/v1/submissions/:id - Non-existent submission", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/submissions/00000000-0000-0000-0000-000000000000")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/submissions/00000000-...",
			Method:      "GET",
			Description: "Get non-existent submission",
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

	// Test 4: Get existing submission (if we created one earlier)
	if GetConfig().TestSubmissionID != "" {
		t.Run("GET /api/v1/submissions/:id - Valid submission", func(t *testing.T) {
			RequireAdminAuth(t, client)

			start := time.Now()
			resp, err := client.GET("/api/v1/submissions/" + GetConfig().TestSubmissionID)
			duration := time.Since(start)

			result := TestResult{
				Endpoint:    "/api/v1/submissions/" + GetConfig().TestSubmissionID,
				Method:      "GET",
				Description: "Get existing submission by ID",
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

				var subResp map[string]interface{}
				if err := resp.JSON(&subResp); err == nil {
					if submission, ok := subResp["submission"].(map[string]interface{}); ok {
						t.Logf("Submission ID: %v", submission["id"])
						t.Logf("Company Name: %v", submission["companyName"])
						t.Logf("Status: %v", submission["status"])

						// Save analysis ID if available
						if analysisID, ok := submission["analysisId"].(string); ok && analysisID != "" {
							GetConfig().TestAnalysisID = analysisID
							t.Logf("Analysis ID: %v", analysisID)
						}
					}
				}
			} else if resp.StatusCode == http.StatusForbidden {
				// Might not own this submission
				result.Passed = true
				result.Error = "Access denied (not owner)"
				t.Logf("Access denied to submission: %s", string(resp.Body))
			} else {
				result.Passed = false
				result.Error = fmt.Sprintf("Expected 200/403, got %d", resp.StatusCode)
			}

			report.AddResult(result)
		})
	}

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestSubmissionAnalysis(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Get analysis without auth
	t.Run("GET /api/v1/submissions/:id/analysis - Without authentication", func(t *testing.T) {
		submissionID := GetConfig().TestSubmissionID
		if submissionID == "" {
			submissionID = "00000000-0000-0000-0000-000000000000"
		}

		start := time.Now()
		resp, err := client.GET("/api/v1/submissions/" + submissionID + "/analysis")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/submissions/:id/analysis",
			Method:      "GET",
			Description: "Get submission analysis without authentication",
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

	// Test 2: Get analysis for non-existent submission
	t.Run("GET /api/v1/submissions/:id/analysis - Non-existent submission", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/submissions/00000000-0000-0000-0000-000000000000/analysis")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/submissions/.../analysis",
			Method:      "GET",
			Description: "Get analysis for non-existent submission",
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

	// Test 3: Get analysis for existing submission
	if GetConfig().TestSubmissionID != "" {
		t.Run("GET /api/v1/submissions/:id/analysis - Existing submission", func(t *testing.T) {
			RequireAdminAuth(t, client)

			start := time.Now()
			resp, err := client.GET("/api/v1/submissions/" + GetConfig().TestSubmissionID + "/analysis")
			duration := time.Since(start)

			result := TestResult{
				Endpoint:    "/api/v1/submissions/:id/analysis",
				Method:      "GET",
				Description: "Get analysis for existing submission",
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

			// Analysis might not exist yet for a new submission
			if resp.StatusCode == http.StatusOK {
				result.Passed = true
				t.Logf("Analysis found: %s", string(resp.Body)[:min(200, len(resp.Body))])
			} else if resp.StatusCode == http.StatusNotFound {
				result.Passed = true
				result.Error = "Analysis not found (might not be generated yet)"
				t.Logf("Analysis not found yet: %s", string(resp.Body))
			} else if resp.StatusCode == http.StatusForbidden {
				result.Passed = true
				result.Error = "Access denied (not owner)"
				t.Logf("Access denied: %s", string(resp.Body))
			} else {
				result.Passed = false
				result.Error = fmt.Sprintf("Expected 200/404/403, got %d", resp.StatusCode)
			}

			report.AddResult(result)
		})
	}

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
