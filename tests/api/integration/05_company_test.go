package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// =============================================================================
// COMPANY ENDPOINTS INTEGRATION TESTS
// =============================================================================
// Endpoints tested:
//   - POST /api/v1/companies (auth required)
//   - GET /api/v1/companies (auth required)
//   - GET /api/v1/companies/:id (auth required)
// =============================================================================

func TestCompanyCreate(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Create company without auth
	t.Run("POST /api/v1/companies - Without authentication", func(t *testing.T) {
		start := time.Now()
		resp, err := client.POST("/api/v1/companies", map[string]interface{}{
			"name": "Test Company Direct",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/companies",
			Method:      "POST",
			Description: "Create company without authentication",
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

	// Test 2: Create company with valid data
	t.Run("POST /api/v1/companies - Valid company data", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.POST("/api/v1/companies", map[string]interface{}{
			"name":          "Test Company Direct Create",
			"website":       "https://testcompany.com",
			"industry":      "Technology",
			"company_size":  "medium",
			"location":      "São Paulo, SP",
			"target_market": "B2B",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/companies",
			Method:      "POST",
			Description: "Create company with valid data",
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
				if id, ok := createResp["id"].(string); ok {
					t.Logf("Created company ID: %s", id)
					// Don't overwrite TestCompanyID if already set from submission
					if GetConfig().TestCompanyID == "" {
						GetConfig().TestCompanyID = id
					}
				}
				t.Logf("Enrichment status: %v", createResp["enrichmentStatus"])
			}
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 201, got %d", resp.StatusCode)
			t.Errorf("Expected 201, got %d: %s", resp.StatusCode, string(resp.Body))
		}

		report.AddResult(result)
	})

	// Test 3: Create company without required name
	t.Run("POST /api/v1/companies - Missing required name", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.POST("/api/v1/companies", map[string]interface{}{
			"website": "https://testcompany.com",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/companies",
			Method:      "POST",
			Description: "Create company without required name",
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
			t.Logf("Correctly rejected missing name: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 400, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestCompanyList(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: List companies without auth
	t.Run("GET /api/v1/companies - Without authentication", func(t *testing.T) {
		start := time.Now()
		resp, err := client.GET("/api/v1/companies")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/companies",
			Method:      "GET",
			Description: "List companies without authentication",
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

	// Test 2: List user's companies with auth
	t.Run("GET /api/v1/companies - With authentication", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/companies")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/companies",
			Method:      "GET",
			Description: "List user's own companies",
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
				if companies, ok := listResp["companies"].([]interface{}); ok {
					t.Logf("Found %d companies", len(companies))
					for i, co := range companies {
						if i >= 3 {
							t.Logf("... and %d more", len(companies)-3)
							break
						}
						if coMap, ok := co.(map[string]interface{}); ok {
							t.Logf("  - %v (enrichment: %v)", coMap["name"], coMap["enrichmentStatus"])
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

func TestCompanyGetById(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Get company without auth
	t.Run("GET /api/v1/companies/:id - Without authentication", func(t *testing.T) {
		companyID := GetConfig().TestCompanyID
		if companyID == "" {
			companyID = "00000000-0000-0000-0000-000000000000"
		}

		start := time.Now()
		resp, err := client.GET("/api/v1/companies/" + companyID)
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/companies/:id",
			Method:      "GET",
			Description: "Get company by ID without authentication",
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

	// Test 2: Get company with invalid UUID
	t.Run("GET /api/v1/companies/:id - Invalid UUID format", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/companies/not-a-valid-uuid")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/companies/not-a-valid-uuid",
			Method:      "GET",
			Description: "Get company with invalid UUID",
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

	// Test 3: Get non-existent company
	t.Run("GET /api/v1/companies/:id - Non-existent company", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/companies/00000000-0000-0000-0000-000000000000")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/companies/00000000-...",
			Method:      "GET",
			Description: "Get non-existent company",
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

	// Test 4: Get existing company
	if GetConfig().TestCompanyID != "" {
		t.Run("GET /api/v1/companies/:id - Valid company", func(t *testing.T) {
			RequireAdminAuth(t, client)

			start := time.Now()
			resp, err := client.GET("/api/v1/companies/" + GetConfig().TestCompanyID)
			duration := time.Since(start)

			result := TestResult{
				Endpoint:    "/api/v1/companies/" + GetConfig().TestCompanyID,
				Method:      "GET",
				Description: "Get existing company by ID",
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

				var coResp map[string]interface{}
				if err := resp.JSON(&coResp); err == nil {
					t.Logf("Company: %v", coResp["name"])
					t.Logf("Industry: %v", coResp["industry"])
					t.Logf("Enrichment Status: %v", coResp["enrichmentStatus"])
					if enrichedAt, ok := coResp["enrichmentCompletedAt"]; ok {
						t.Logf("Enriched At: %v", enrichedAt)
					}
				}
			} else if resp.StatusCode == http.StatusForbidden {
				result.Passed = true
				result.Error = "Access denied (not owner)"
				t.Logf("Access denied: %s", string(resp.Body))
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
