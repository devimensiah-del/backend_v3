package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// --- AUTH ENDPOINTS ---

// GetCurrentUser returns the authenticated user's profile
func (h *Handler) GetCurrentUser(c *gin.Context) {
	// Get user ID from AuthMiddleware
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "User not authenticated"})
		return
	}

	// Query user_profiles table
	var profile UserProfile
	query := `
		SELECT id, email, full_name, role, is_active, created_at, updated_at
		FROM user_profiles
		WHERE id = $1
	`

	err := h.db.Get(&profile, query, userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID.(string)).Msg("Failed to fetch user profile")
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "User profile not found"})
		return
	}

	c.JSON(http.StatusOK, UserProfileResponse{User: profile})
}

// Login handles POST /api/v1/auth/login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	// Supabase Auth API endpoint for login
	authURL := fmt.Sprintf("%s/auth/v1/token?grant_type=password", h.supabaseURL)

	payload := map[string]string{
		"email":    req.Email,
		"password": req.Password,
	}

	resp, err := h.callSupabaseAuth("POST", authURL, payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("Login failed")
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Login failed", Message: "Invalid credentials"})
		return
	}

	// Frontend expects "token" field, Supabase returns "access_token"
	// Add "token" as alias for compatibility
	if accessToken, ok := resp["access_token"].(string); ok {
		resp["token"] = accessToken
	}

	c.JSON(http.StatusOK, resp)
}

// Signup handles POST /api/v1/auth/signup
func (h *Handler) Signup(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	// Supabase Auth API endpoint for signup
	authURL := fmt.Sprintf("%s/auth/v1/signup", h.supabaseURL)

	payload := map[string]interface{}{
		"email":    req.Email,
		"password": req.Password,
		"data": map[string]string{
			"full_name": req.FullName,
		},
	}

	resp, err := h.callSupabaseAuth("POST", authURL, payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("Signup failed")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Signup failed", Message: err.Error()})
		return
	}

	// Frontend expects "token" field, Supabase returns "access_token"
	// Add "token" as alias for compatibility
	if accessToken, ok := resp["access_token"].(string); ok {
		resp["token"] = accessToken
	}

	c.JSON(http.StatusCreated, resp)
}

// Logout handles POST /api/v1/auth/logout
func (h *Handler) Logout(c *gin.Context) {
	// Extract JWT token from Authorization header
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "No token provided"})
		return
	}

	// Remove "Bearer " prefix
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// Supabase Auth API endpoint for logout
	authURL := fmt.Sprintf("%s/auth/v1/logout", h.supabaseURL)

	// Call Supabase with the user's token
	req, _ := http.NewRequest("POST", authURL, nil)
	req.Header.Set("apikey", h.supabaseAnonKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	httpResp, err := client.Do(req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Logout request failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Logout failed", Message: err.Error()})
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusNoContent && httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		h.logger.Error().Int("status", httpResp.StatusCode).Str("body", string(body)).Msg("Logout failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Logout failed",
			Message: "Failed to invalidate session",
		})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Logged out successfully"})
}

// ForgotPassword handles POST /api/v1/auth/forgot-password
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	// Supabase Auth API endpoint for password recovery
	authURL := fmt.Sprintf("%s/auth/v1/recover", h.supabaseURL)

	payload := map[string]string{
		"email": req.Email,
	}

	_, err := h.callSupabaseAuth("POST", authURL, payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("Password reset request failed")
		// Don't reveal if email exists or not for security
	}

	// Always return success to prevent email enumeration
	c.JSON(http.StatusOK, PasswordResetResponse{
		Message: "If an account exists with this email, you will receive password reset instructions.",
	})
}

// ResetPassword handles POST /api/v1/auth/reset-password
func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	// Supabase Auth API - verify token and update password
	authURL := fmt.Sprintf("%s/auth/v1/token?grant_type=recovery", h.supabaseURL)

	payload := map[string]string{
		"token":    req.Token,
		"password": req.NewPassword,
	}

	resp, err := h.callSupabaseAuth("POST", authURL, payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("Password reset failed")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Password reset failed", Message: "Invalid or expired token"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UpdatePassword handles PUT /api/v1/auth/update-password
func (h *Handler) UpdatePassword(c *gin.Context) {
	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	// Extract JWT token from Authorization header
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "No token provided"})
		return
	}

	// Remove "Bearer " prefix
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// Supabase Auth API endpoint for updating user
	authURL := fmt.Sprintf("%s/auth/v1/user", h.supabaseURL)

	payload := map[string]string{
		"password": req.NewPassword,
	}

	payloadBytes, _ := json.Marshal(payload)

	httpReq, _ := http.NewRequest("PUT", authURL, bytes.NewBuffer(payloadBytes))
	httpReq.Header.Set("apikey", h.supabaseAnonKey)
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Update password request failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Update failed", Message: err.Error()})
		return
	}
	defer httpResp.Body.Close()

	body, _ := io.ReadAll(httpResp.Body)

	if httpResp.StatusCode != http.StatusOK {
		h.logger.Error().Int("status", httpResp.StatusCode).Str("body", string(body)).Msg("Update password failed")
		c.JSON(httpResp.StatusCode, ErrorResponse{Error: "Update failed", Message: "Failed to update password"})
		return
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	c.JSON(http.StatusOK, result)
}

// callSupabaseAuth is a helper function to make authenticated requests to Supabase Auth API
// AUTH FIX: Now uses the correct Supabase Anon Key instead of JWT Secret
func (h *Handler) callSupabaseAuth(method, url string, payload interface{}) (map[string]interface{}, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", h.supabaseAnonKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		h.logger.Error().Int("status", resp.StatusCode).Str("body", string(body)).Msg("Supabase Auth API error")
		return nil, fmt.Errorf("authentication failed: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}
