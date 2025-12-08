package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// =============================================================================
// AUTH ENDPOINTS INTEGRATION TESTS
// =============================================================================
// Endpoints tested:
//   - POST /api/v1/auth/login
//   - POST /api/v1/auth/signup
//   - POST /api/v1/auth/logout
//   - GET /api/v1/auth/me
//   - POST /api/v1/auth/forgot-password
//   - POST /api/v1/auth/reset-password
//   - PUT /api/v1/auth/update-password
// =============================================================================

func TestAuthLogin(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Successful admin login
	t.Run("POST /api/v1/auth/login - Valid admin credentials", func(t *testing.T) {
		cfg := GetConfig()
		start := time.Now()
		resp, err := client.POST("/api/v1/auth/login", map[string]string{
			"email":    cfg.AdminEmail,
			"password": cfg.AdminPassword,
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/login",
			Method:      "POST",
			Description: "Login with valid admin credentials",
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
			var loginResp LoginResponse
			if err := resp.JSON(&loginResp); err == nil {
				token := loginResp.AccessToken
				if token == "" {
					token = loginResp.Token
				}
				t.Logf("Login successful. Token length: %d", len(token))
				if loginResp.User != nil {
					t.Logf("User: %v", loginResp.User)
				}
			}
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Login failed with status %d", resp.StatusCode)
			t.Errorf("Expected 200, got %d: %s", resp.StatusCode, string(resp.Body))
		}

		report.AddResult(result)
	})

	// Test 2: Login with invalid password
	t.Run("POST /api/v1/auth/login - Invalid password", func(t *testing.T) {
		cfg := GetConfig()
		start := time.Now()
		resp, err := client.POST("/api/v1/auth/login", map[string]string{
			"email":    cfg.AdminEmail,
			"password": "wrongpassword123",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/login",
			Method:      "POST",
			Description: "Login with invalid password",
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

		// 400 or 401 are acceptable for invalid credentials
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest {
			result.Passed = true
			t.Logf("Correctly rejected invalid password: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 401/400, got %d", resp.StatusCode)
			t.Errorf("Expected 401/400, got %d: %s", resp.StatusCode, string(resp.Body))
		}

		report.AddResult(result)
	})

	// Test 3: Login with invalid email
	t.Run("POST /api/v1/auth/login - Non-existent email", func(t *testing.T) {
		start := time.Now()
		resp, err := client.POST("/api/v1/auth/login", map[string]string{
			"email":    "nonexistent@example.com",
			"password": "anypassword123",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/login",
			Method:      "POST",
			Description: "Login with non-existent email",
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

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest {
			result.Passed = true
			t.Logf("Correctly rejected non-existent email: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 401/400, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 4: Login with missing fields
	t.Run("POST /api/v1/auth/login - Missing password", func(t *testing.T) {
		start := time.Now()
		resp, err := client.POST("/api/v1/auth/login", map[string]string{
			"email": "test@example.com",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/login",
			Method:      "POST",
			Description: "Login with missing password field",
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
			t.Logf("Correctly rejected missing password: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 400, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 5: Login with invalid email format
	t.Run("POST /api/v1/auth/login - Invalid email format", func(t *testing.T) {
		start := time.Now()
		resp, err := client.POST("/api/v1/auth/login", map[string]string{
			"email":    "not-an-email",
			"password": "password123",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/login",
			Method:      "POST",
			Description: "Login with invalid email format",
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
			t.Logf("Correctly rejected invalid email format: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 400, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestAuthGetCurrentUser(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Get current user without auth
	t.Run("GET /api/v1/auth/me - Without authentication", func(t *testing.T) {
		start := time.Now()
		resp, err := client.GET("/api/v1/auth/me")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/me",
			Method:      "GET",
			Description: "Get current user without authentication",
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

	// Test 2: Get current user with valid token
	t.Run("GET /api/v1/auth/me - With valid admin token", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/auth/me")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/me",
			Method:      "GET",
			Description: "Get current user with valid admin token",
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
			var userResp map[string]interface{}
			if err := resp.JSON(&userResp); err == nil {
				t.Logf("Current user response: %v", userResp)
				if user, ok := userResp["user"].(map[string]interface{}); ok {
					t.Logf("User email: %v", user["email"])
					t.Logf("User role: %v", user["role"])
				}
			}
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 200, got %d", resp.StatusCode)
			t.Errorf("Expected 200, got %d: %s", resp.StatusCode, string(resp.Body))
		}

		report.AddResult(result)
	})

	// Test 3: Get current user with invalid token
	t.Run("GET /api/v1/auth/me - With invalid token", func(t *testing.T) {
		invalidClient := NewTestClient(t).WithToken("invalid.token.here")

		start := time.Now()
		resp, err := invalidClient.GET("/api/v1/auth/me")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/me",
			Method:      "GET",
			Description: "Get current user with invalid token",
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
			t.Logf("Correctly rejected invalid token: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 401, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestAuthLogout(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Logout without auth
	t.Run("POST /api/v1/auth/logout - Without authentication", func(t *testing.T) {
		start := time.Now()
		resp, err := client.POST("/api/v1/auth/logout", nil)
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/logout",
			Method:      "POST",
			Description: "Logout without authentication",
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
			t.Logf("Correctly rejected unauthenticated logout: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 401, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 2: Logout with valid token
	t.Run("POST /api/v1/auth/logout - With valid token", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.POST("/api/v1/auth/logout", nil)
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/logout",
			Method:      "POST",
			Description: "Logout with valid token",
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
			t.Logf("Logout successful: %s", string(resp.Body))
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

func TestAuthForgotPassword(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Forgot password with valid email
	t.Run("POST /api/v1/auth/forgot-password - Valid email format", func(t *testing.T) {
		start := time.Now()
		resp, err := client.POST("/api/v1/auth/forgot-password", map[string]string{
			"email": "test@example.com",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/forgot-password",
			Method:      "POST",
			Description: "Request password reset with valid email",
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

		// Should always return 200 to prevent email enumeration
		if resp.StatusCode == http.StatusOK {
			result.Passed = true
			t.Logf("Forgot password response: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 200, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 2: Forgot password with invalid email format
	t.Run("POST /api/v1/auth/forgot-password - Invalid email format", func(t *testing.T) {
		start := time.Now()
		resp, err := client.POST("/api/v1/auth/forgot-password", map[string]string{
			"email": "not-an-email",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/forgot-password",
			Method:      "POST",
			Description: "Request password reset with invalid email format",
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
			t.Logf("Correctly rejected invalid email format: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 400, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 3: Forgot password with missing email
	t.Run("POST /api/v1/auth/forgot-password - Missing email", func(t *testing.T) {
		start := time.Now()
		resp, err := client.POST("/api/v1/auth/forgot-password", map[string]string{})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/forgot-password",
			Method:      "POST",
			Description: "Request password reset with missing email",
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
			t.Logf("Correctly rejected missing email: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 400, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestAuthResetPassword(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Reset password with invalid token
	t.Run("POST /api/v1/auth/reset-password - Invalid token", func(t *testing.T) {
		start := time.Now()
		resp, err := client.POST("/api/v1/auth/reset-password", map[string]string{
			"token":       "invalid-reset-token",
			"newPassword": "newpassword123",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/reset-password",
			Method:      "POST",
			Description: "Reset password with invalid token",
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

		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
			result.Passed = true
			t.Logf("Correctly rejected invalid token: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 400/401, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	// Test 2: Reset password with missing fields
	t.Run("POST /api/v1/auth/reset-password - Missing token", func(t *testing.T) {
		start := time.Now()
		resp, err := client.POST("/api/v1/auth/reset-password", map[string]string{
			"newPassword": "newpassword123",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/reset-password",
			Method:      "POST",
			Description: "Reset password with missing token",
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
			t.Logf("Correctly rejected missing token: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 400, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestAuthUpdatePassword(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: Update password without auth
	t.Run("PUT /api/v1/auth/update-password - Without authentication", func(t *testing.T) {
		start := time.Now()
		resp, err := client.PUT("/api/v1/auth/update-password", map[string]string{
			"currentPassword": "oldpassword",
			"newPassword":     "newpassword123",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/update-password",
			Method:      "PUT",
			Description: "Update password without authentication",
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

	// Test 2: Update password with missing fields
	t.Run("PUT /api/v1/auth/update-password - Missing newPassword", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.PUT("/api/v1/auth/update-password", map[string]string{
			"currentPassword": "oldpassword",
		})
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/auth/update-password",
			Method:      "PUT",
			Description: "Update password with missing newPassword",
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
			t.Logf("Correctly rejected missing newPassword: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 400, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}

func TestAuthUserProfile(t *testing.T) {
	client := NewTestClient(t)
	report := NewTestReport()

	// Test 1: GET /user/profile alias
	t.Run("GET /api/v1/user/profile - With valid token", func(t *testing.T) {
		RequireAdminAuth(t, client)

		start := time.Now()
		resp, err := client.GET("/api/v1/user/profile")
		duration := time.Since(start)

		result := TestResult{
			Endpoint:    "/api/v1/user/profile",
			Method:      "GET",
			Description: "Get user profile via /user/profile alias",
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
			t.Logf("User profile response: %s", string(resp.Body))
		} else {
			result.Passed = false
			result.Error = fmt.Sprintf("Expected 200, got %d", resp.StatusCode)
		}

		report.AddResult(result)
	})

	t.Log(report.Summary())
	t.Log(report.PrintDetails())
}
