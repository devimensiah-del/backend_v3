package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// AuthHandlers holds all service dependencies and configuration for auth handlers
type AuthHandlers struct {
	DB              *sqlx.DB
	Logger          zerolog.Logger
	SupabaseURL     string
	SupabaseAnonKey string
	JWTSecret       string
	MockMode        bool
}

// NewAuthHandlers creates a new auth handler set with all dependencies
func NewAuthHandlers(
	db *sqlx.DB,
	logger zerolog.Logger,
	supabaseURL string,
	supabaseAnonKey string,
	jwtSecret string,
) *AuthHandlers {
	mockMode := strings.EqualFold(supabaseURL, "mock") || strings.EqualFold(os.Getenv("MOCK_AUTH"), "true")
	return &AuthHandlers{
		DB:              db,
		Logger:          logger,
		SupabaseURL:     supabaseURL,
		SupabaseAnonKey: supabaseAnonKey,
		JWTSecret:       jwtSecret,
		MockMode:        mockMode,
	}
}

// GetCurrentUser returns the authenticated user's profile
func (h *AuthHandlers) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "User not authenticated"})
		return
	}

	var profile UserProfile
	query := `
		SELECT id, email, full_name, role, is_active, created_at, updated_at
		FROM user_profiles
		WHERE id = $1
	`

	err := h.DB.Get(&profile, query, userID)
	if err != nil {
		h.Logger.Error().Err(err).Str("user_id", fmt.Sprintf("%v", userID)).Msg("Failed to fetch user profile")
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "User profile not found"})
		return
	}

	c.JSON(http.StatusOK, UserProfileResponse{User: profile})
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandlers) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Please provide a valid email and password",
		})
		return
	}

	if h.MockMode {
		userID, role, err := h.ensureUserExists(req.Email)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Mock login failed: ensure user")
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Login failed", Message: "mock auth error"})
			return
		}
		token, err := h.issueMockToken(userID, role, req.Email)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Mock login failed: token")
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Login failed", Message: "mock token error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"access_token": token,
			"token":        token,
			"user": gin.H{
				"id":    userID,
				"email": req.Email,
				"role":  role,
			},
		})
		return
	}

	authURL := fmt.Sprintf("%s/auth/v1/token?grant_type=password", h.SupabaseURL)

	payload := map[string]string{
		"email":    req.Email,
		"password": req.Password,
	}

	resp, err := h.callSupabaseAuth("POST", authURL, payload)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Login failed")
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Login failed", Message: "Invalid credentials"})
		return
	}

	if accessToken, ok := resp["access_token"].(string); ok {
		resp["token"] = accessToken
	}

	c.JSON(http.StatusOK, resp)
}

// Signup handles POST /api/v1/auth/signup
func (h *AuthHandlers) Signup(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Please provide all required fields",
		})
		return
	}

	if h.MockMode {
		userID, role, err := h.ensureUserExists(req.Email)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Mock signup failed: ensure user")
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Signup failed", Message: "mock auth error"})
			return
		}
		token, err := h.issueMockToken(userID, role, req.Email)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Mock signup failed: token")
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Signup failed", Message: "mock token error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"access_token": token,
			"token":        token,
			"user": gin.H{
				"id":    userID,
				"email": req.Email,
				"role":  role,
			},
		})
		return
	}

	authURL := fmt.Sprintf("%s/auth/v1/signup", h.SupabaseURL)

	payload := map[string]interface{}{
		"email":    req.Email,
		"password": req.Password,
		"data": map[string]string{
			"full_name": req.FullName,
		},
	}

	resp, err := h.callSupabaseAuth("POST", authURL, payload)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Signup failed")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Signup failed", Message: err.Error()})
		return
	}

	if accessToken, ok := resp["access_token"].(string); ok {
		resp["token"] = accessToken
	}

	c.JSON(http.StatusCreated, resp)
}

// Logout handles POST /api/v1/auth/logout
func (h *AuthHandlers) Logout(c *gin.Context) {
	if h.MockMode {
		c.JSON(http.StatusOK, MessageResponse{Message: "Logged out successfully (mock)"})
		return
	}

	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "No token provided"})
		return
	}
	if len(token) > 7 && strings.HasPrefix(token, "Bearer ") {
		token = token[7:]
	}

	authURL := fmt.Sprintf("%s/auth/v1/logout", h.SupabaseURL)

	req, _ := http.NewRequest("POST", authURL, nil)
	req.Header.Set("apikey", h.SupabaseAnonKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	httpResp, err := client.Do(req)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Logout request failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Logout failed", Message: err.Error()})
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusNoContent && httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		h.Logger.Error().Int("status", httpResp.StatusCode).Str("body", string(body)).Msg("Logout failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Logout failed",
			Message: "Failed to invalidate session",
		})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Logged out successfully"})
}

// ForgotPassword handles POST /api/v1/auth/forgot-password
func (h *AuthHandlers) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Please provide a valid email",
		})
		return
	}

	if h.MockMode {
		c.JSON(http.StatusOK, PasswordResetResponse{
			Message: "Password reset email sent (mock)",
		})
		return
	}

	authURL := fmt.Sprintf("%s/auth/v1/recover", h.SupabaseURL)

	payload := map[string]string{
		"email": req.Email,
	}

	_, err := h.callSupabaseAuth("POST", authURL, payload)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Password reset request failed")
	}

	c.JSON(http.StatusOK, PasswordResetResponse{
		Message: "If an account exists with this email, you will receive password reset instructions.",
	})
}

// ResetPassword handles POST /api/v1/auth/reset-password
func (h *AuthHandlers) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Token and new password are required",
		})
		return
	}

	if h.MockMode {
		c.JSON(http.StatusOK, MessageResponse{Message: "Password reset successful (mock)"})
		return
	}

	authURL := fmt.Sprintf("%s/auth/v1/token?grant_type=recovery", h.SupabaseURL)

	payload := map[string]string{
		"token":    req.Token,
		"password": req.NewPassword,
	}

	resp, err := h.callSupabaseAuth("POST", authURL, payload)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Password reset failed")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Password reset failed", Message: "Invalid or expired token"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UpdatePassword handles PUT /api/v1/auth/update-password
func (h *AuthHandlers) UpdatePassword(c *gin.Context) {
	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Current password and new password are required",
		})
		return
	}

	if h.MockMode {
		c.JSON(http.StatusOK, MessageResponse{Message: "Password updated successfully (mock)"})
		return
	}

	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "No token provided"})
		return
	}
	if len(token) > 7 && strings.HasPrefix(token, "Bearer ") {
		token = token[7:]
	}

	authURL := fmt.Sprintf("%s/auth/v1/user", h.SupabaseURL)

	payload := map[string]string{
		"password": req.NewPassword,
	}

	payloadBytes, _ := json.Marshal(payload)

	httpReq, _ := http.NewRequest("PUT", authURL, bytes.NewBuffer(payloadBytes))
	httpReq.Header.Set("apikey", h.SupabaseAnonKey)
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Update password request failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Update failed", Message: err.Error()})
		return
	}
	defer httpResp.Body.Close()

	body, _ := io.ReadAll(httpResp.Body)

	if httpResp.StatusCode != http.StatusOK {
		h.Logger.Error().Int("status", httpResp.StatusCode).Str("body", string(body)).Msg("Update password failed")
		c.JSON(httpResp.StatusCode, ErrorResponse{Error: "Update failed", Message: "Failed to update password"})
		return
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	c.JSON(http.StatusOK, result)
}

// Helpers for mock auth flow
func (h *AuthHandlers) ensureUserExists(email string) (string, string, error) {
	role := "user"
	if strings.Contains(strings.ToLower(email), "admin") {
		role = "admin"
	}

	var userID string
	err := h.DB.Get(&userID, "SELECT id FROM user_profiles WHERE email = $1", email)
	if err == nil && userID != "" {
		return userID, role, nil
	}

	newID := uuid.New().String()
	_, execErr := h.DB.Exec(
		"INSERT INTO user_profiles (id, email, full_name, role, is_active, created_at, updated_at) VALUES ($1, $2, $3, $4, TRUE, NOW(), NOW())",
		newID, email, email, role,
	)
	if execErr != nil {
		return "", "", execErr
	}
	return newID, role, nil
}

func (h *AuthHandlers) issueMockToken(userID, role, email string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  role,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.JWTSecret))
}

// callSupabaseAuth is a helper function to make authenticated requests to Supabase Auth API
func (h *AuthHandlers) callSupabaseAuth(method, url string, payload interface{}) (map[string]interface{}, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", h.SupabaseAnonKey)
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
		h.Logger.Error().Int("status", resp.StatusCode).Str("body", string(body)).Msg("Supabase Auth API error")
		return nil, fmt.Errorf("authentication failed: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}
