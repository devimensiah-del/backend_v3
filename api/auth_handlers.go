package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	apperrors "backend_v3/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// SubmissionLinkingService defines the contract for linking anonymous submissions
// Using an interface avoids import cycles with the submission package
type SubmissionLinkingService interface {
	LinkAnonymousToUser(ctx context.Context, userID uuid.UUID, email string) error
}

// AuthHandlers holds all service dependencies and configuration for auth handlers
type AuthHandlers struct {
	DB                *sqlx.DB
	Logger            zerolog.Logger
	SupabaseURL       string
	SupabaseAnonKey   string
	JWTSecret         string
	MockMode          bool
	SubmissionService SubmissionLinkingService // Optional - for linking anonymous submissions on signup
}

// NewAuthHandlers creates a new auth handler set with all dependencies
func NewAuthHandlers(
	db *sqlx.DB,
	logger zerolog.Logger,
	supabaseURL string,
	supabaseAnonKey string,
	jwtSecret string,
	submissionSvc SubmissionLinkingService, // Optional - can be nil for graceful degradation
) *AuthHandlers {
	mockMode := strings.EqualFold(supabaseURL, "mock") || strings.EqualFold(os.Getenv("MOCK_AUTH"), "true")
	return &AuthHandlers{
		DB:                db,
		Logger:            logger,
		SupabaseURL:       supabaseURL,
		SupabaseAnonKey:   supabaseAnonKey,
		JWTSecret:         jwtSecret,
		MockMode:          mockMode,
		SubmissionService: submissionSvc,
	}
}

// GetCurrentUser returns the authenticated user's profile
func (h *AuthHandlers) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		RespondAppError(c, h.Logger, apperrors.ErrUnauthorized)
		return
	}

	var profile UserProfile
	query := `
		SELECT id, email, full_name, role, is_active, password_set, created_at, updated_at
		FROM user_profiles
		WHERE id = $1
	`

	err := h.DB.Get(&profile, query, userID)
	if err != nil {
		RespondAppError(c, h.Logger, apperrors.ErrNotFound.WithInternal(err))
		return
	}

	c.JSON(http.StatusOK, UserProfileResponse{User: profile})
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandlers) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Requisição inválida",
			Message: "Por favor, informe e-mail e senha válidos",
		})
		return
	}

	if h.MockMode {
		userID, role, err := h.ensureUserExists(req.Email)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Mock login failed: ensure user")
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Falha no login", Message: "Erro de autenticação"})
			return
		}
		token, err := h.issueMockToken(userID, role, req.Email)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Mock login failed: token")
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Falha no login", Message: "Erro ao gerar token"})
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

		// Check if user exists but needs to set password
		var profile UserProfile
		profileErr := h.DB.Get(&profile, `
			SELECT id, email, full_name, role, is_active, password_set, created_at, updated_at
			FROM user_profiles WHERE email = $1
		`, req.Email)
		if profileErr == nil && !profile.PasswordSet {
			// User exists but was auto-created without password
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":   "PASSWORD_NOT_SET",
				"code":    "PASSWORD_NOT_SET",
				"message": "Você precisa definir uma senha para continuar",
				"user": gin.H{
					"id":    profile.ID,
					"email": profile.Email,
				},
			})
			return
		}

		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Falha no login", Message: err.Error()})
		return
	}

	if accessToken, ok := resp["access_token"].(string); ok {
		resp["token"] = accessToken
	}

	// Fetch user profile from database to get role
	var profile UserProfile
	query := `
		SELECT id, email, full_name, role, is_active, created_at, updated_at
		FROM user_profiles
		WHERE email = $1
	`
	if err := h.DB.Get(&profile, query, req.Email); err == nil {
		resp["user"] = gin.H{
			"id":         profile.ID,
			"email":      profile.Email,
			"full_name":  profile.FullName,
			"role":       profile.Role,
			"is_active":  profile.IsActive,
			"created_at": profile.CreatedAt,
			"updated_at": profile.UpdatedAt,
		}
	} else {
		// Profile doesn't exist - create it from Supabase user data
		// This handles users who signed up before profile creation was implemented
		if userID := h.extractUserIDFromResponse(resp); userID != "" {
			if createErr := h.createUserProfile(userID, req.Email, ""); createErr != nil {
				h.Logger.Warn().Err(createErr).Str("email", req.Email).Msg("Failed to create missing profile on login")
			} else {
				resp["user"] = gin.H{
					"id":        userID,
					"email":     req.Email,
					"full_name": strings.Split(req.Email, "@")[0],
					"role":      "user",
				}
			}
		}
	}

	c.JSON(http.StatusOK, resp)
}

// Signup handles POST /api/v1/auth/signup
func (h *AuthHandlers) Signup(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Requisição inválida",
			Message: "Por favor, preencha todos os campos obrigatórios",
		})
		return
	}

	if h.MockMode {
		userID, role, err := h.ensureUserExists(req.Email)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Mock signup failed: ensure user")
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Falha no cadastro", Message: "Erro de autenticação"})
			return
		}
		token, err := h.issueMockToken(userID, role, req.Email)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Mock signup failed: token")
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Falha no cadastro", Message: "Erro ao gerar token"})
			return
		}

		// Link anonymous submissions to the new user (non-blocking)
		h.linkAnonymousSubmissions(userID, req.Email)

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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Falha no cadastro", Message: err.Error()})
		return
	}

	if accessToken, ok := resp["access_token"].(string); ok {
		resp["token"] = accessToken
	}

	// Extract userID from Supabase response
	userID := h.extractUserIDFromResponse(resp)
	if userID != "" {
		// Create user_profiles record (Supabase only creates auth.users)
		if err := h.createUserProfile(userID, req.Email, req.FullName); err != nil {
			h.Logger.Error().Err(err).Str("user_id", userID).Msg("Failed to create user profile")
			// Continue anyway - user can still use the app, profile can be created on first login
		}

		// Link anonymous submissions to the new user (non-blocking)
		h.linkAnonymousSubmissions(userID, req.Email)

		// Add user info to response
		resp["user"] = gin.H{
			"id":        userID,
			"email":     req.Email,
			"full_name": req.FullName,
			"role":      "user",
		}
	}

	c.JSON(http.StatusCreated, resp)
}

// Logout handles POST /api/v1/auth/logout
func (h *AuthHandlers) Logout(c *gin.Context) {
	if h.MockMode {
		c.JSON(http.StatusOK, MessageResponse{Message: "Logout realizado com sucesso"})
		return
	}

	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Não autorizado", Message: "Token não fornecido"})
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
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Falha no logout", Message: "Erro ao encerrar sessão"})
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusNoContent && httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		h.Logger.Error().Int("status", httpResp.StatusCode).Str("body", string(body)).Msg("Logout failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Falha no logout",
			Message: "Não foi possível encerrar a sessão",
		})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Logout realizado com sucesso"})
}

// ForgotPassword handles POST /api/v1/auth/forgot-password
func (h *AuthHandlers) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Requisição inválida",
			Message: "Por favor, informe um e-mail válido",
		})
		return
	}

	if h.MockMode {
		c.JSON(http.StatusOK, PasswordResetResponse{
			Message: "E-mail de recuperação enviado",
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
		Message: "Se existe uma conta com este e-mail, você receberá instruções para redefinir sua senha.",
	})
}

// ResetPassword handles POST /api/v1/auth/reset-password
func (h *AuthHandlers) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Requisição inválida",
			Message: "Token e nova senha são obrigatórios",
		})
		return
	}

	if h.MockMode {
		c.JSON(http.StatusOK, MessageResponse{Message: "Senha redefinida com sucesso"})
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Falha ao redefinir senha", Message: "Token inválido ou expirado"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UpdatePassword handles PUT /api/v1/auth/update-password
func (h *AuthHandlers) UpdatePassword(c *gin.Context) {
	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Requisição inválida",
			Message: "Senha atual e nova senha são obrigatórias",
		})
		return
	}

	if h.MockMode {
		c.JSON(http.StatusOK, MessageResponse{Message: "Senha atualizada com sucesso"})
		return
	}

	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Não autorizado", Message: "Token não fornecido"})
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
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Falha na atualização", Message: "Erro ao atualizar senha"})
		return
	}
	defer httpResp.Body.Close()

	body, _ := io.ReadAll(httpResp.Body)

	if httpResp.StatusCode != http.StatusOK {
		h.Logger.Error().Int("status", httpResp.StatusCode).Str("body", string(body)).Msg("Update password failed")
		c.JSON(httpResp.StatusCode, ErrorResponse{Error: "Falha na atualização", Message: "Não foi possível atualizar a senha"})
		return
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	c.JSON(http.StatusOK, result)
}

// CreateUserWithoutPassword creates a Supabase user without password via Admin API
// Used for auto-creating users during landing page submission
// Returns the user ID or error
func (h *AuthHandlers) CreateUserWithoutPassword(email, fullName string) (string, error) {
	if h.MockMode {
		// In mock mode, just create the user profile directly
		userID, _, err := h.ensureUserExists(email)
		if err != nil {
			return "", err
		}
		// Mark as password not set
		h.DB.Exec("UPDATE user_profiles SET password_set = FALSE WHERE id = $1", userID)
		return userID, nil
	}

	// Use Supabase Admin API to create user without password
	adminURL := fmt.Sprintf("%s/auth/v1/admin/users", h.SupabaseURL)

	payload := map[string]interface{}{
		"email":         email,
		"email_confirm": true, // Already confirmed, no verification needed
		"user_metadata": map[string]string{
			"full_name": fullName,
		},
	}

	payloadBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", adminURL, bytes.NewBuffer(payloadBytes))
	req.Header.Set("apikey", h.SupabaseAnonKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("SUPABASE_SERVICE_ROLE_KEY")))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to create user via Supabase Admin API")
		return "", fmt.Errorf("failed to create user: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		h.Logger.Error().
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Msg("Supabase Admin API error")

		// Check if user already exists
		if strings.Contains(string(body), "already been registered") ||
			strings.Contains(string(body), "user_already_exists") {
			return "", fmt.Errorf("USER_EXISTS")
		}
		return "", fmt.Errorf("failed to create user: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	userID, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("no user ID in response")
	}

	// Create user profile with password_set = false
	if err := h.createUserProfileWithoutPassword(userID, email, fullName); err != nil {
		h.Logger.Error().Err(err).Str("user_id", userID).Msg("Failed to create user profile")
		// Don't fail - user was created in Supabase, profile can be created on first login
	}

	h.Logger.Info().
		Str("user_id", userID).
		Str("email", email).
		Msg("User created without password via Admin API")

	return userID, nil
}

// IssueTemporaryToken generates a JWT token with 1-hour expiry for auto-created users
func (h *AuthHandlers) IssueTemporaryToken(userID, role, email string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  role,
		"exp":   time.Now().Add(1 * time.Hour).Unix(), // 1 hour expiry
		"iat":   time.Now().Unix(),
		"temp":  true, // Mark as temporary session
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.JWTSecret))
}

// SetPassword handles POST /api/v1/auth/set-password
// For users who were auto-created without a password
func (h *AuthHandlers) SetPassword(c *gin.Context) {
	var req SetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Requisição inválida",
			Message: "Senha deve ter pelo menos 6 caracteres",
		})
		return
	}

	// Get user ID from token
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Não autorizado",
			Message: "Token inválido ou expirado",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Erro interno",
			Message: "ID de usuário inválido",
		})
		return
	}

	// Check if user actually needs to set password
	var profile UserProfile
	err := h.DB.Get(&profile, `
		SELECT id, email, full_name, role, is_active, password_set, created_at, updated_at
		FROM user_profiles WHERE id = $1
	`, userIDStr)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Usuário não encontrado",
			Message: "Perfil de usuário não existe",
		})
		return
	}

	if profile.PasswordSet {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Senha já definida",
			Message: "Use a opção 'alterar senha' para modificar sua senha existente",
		})
		return
	}

	if h.MockMode {
		// In mock mode, just update the flag
		_, err = h.DB.Exec("UPDATE user_profiles SET password_set = TRUE, updated_at = NOW() WHERE id = $1", userIDStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Erro ao definir senha",
				Message: "Falha ao atualizar perfil",
			})
			return
		}

		// Issue new 24-hour token
		token, _ := h.issueMockToken(userIDStr, profile.Role, profile.Email)
		c.JSON(http.StatusOK, gin.H{
			"access_token": token,
			"token":        token,
			"message":      "Senha definida com sucesso",
			"user": gin.H{
				"id":          profile.ID,
				"email":       profile.Email,
				"role":        profile.Role,
				"passwordSet": true,
			},
		})
		return
	}

	// Update password in Supabase via Admin API
	adminURL := fmt.Sprintf("%s/auth/v1/admin/users/%s", h.SupabaseURL, userIDStr)

	payload := map[string]interface{}{
		"password": req.Password,
	}

	payloadBytes, _ := json.Marshal(payload)

	httpReq, _ := http.NewRequest("PUT", adminURL, bytes.NewBuffer(payloadBytes))
	httpReq.Header.Set("apikey", h.SupabaseAnonKey)
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("SUPABASE_SERVICE_ROLE_KEY")))
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to set password via Supabase Admin API")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Erro ao definir senha",
			Message: "Falha na comunicação com serviço de autenticação",
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		h.Logger.Error().
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Msg("Supabase Admin API error setting password")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Erro ao definir senha",
			Message: "Falha ao atualizar senha no serviço de autenticação",
		})
		return
	}

	// Update password_set flag in our database
	_, err = h.DB.Exec("UPDATE user_profiles SET password_set = TRUE, updated_at = NOW() WHERE id = $1", userIDStr)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to update password_set flag")
		// Don't fail - password was set in Supabase
	}

	// Login the user to get a fresh token
	loginURL := fmt.Sprintf("%s/auth/v1/token?grant_type=password", h.SupabaseURL)
	loginPayload := map[string]string{
		"email":    profile.Email,
		"password": req.Password,
	}

	loginResp, err := h.callSupabaseAuth("POST", loginURL, loginPayload)
	if err != nil {
		// Password was set but login failed - user can login manually
		c.JSON(http.StatusOK, gin.H{
			"message": "Senha definida com sucesso. Faça login para continuar.",
		})
		return
	}

	// Add user info to response
	loginResp["user"] = gin.H{
		"id":          profile.ID,
		"email":       profile.Email,
		"full_name":   profile.FullName,
		"role":        profile.Role,
		"passwordSet": true,
	}
	loginResp["message"] = "Senha definida com sucesso"

	if accessToken, ok := loginResp["access_token"].(string); ok {
		loginResp["token"] = accessToken
	}

	h.Logger.Info().
		Str("user_id", userIDStr).
		Str("email", profile.Email).
		Msg("Password set successfully for auto-created user")

	c.JSON(http.StatusOK, loginResp)
}

// GetUserIDByEmail looks up a user's ID by their email address
func (h *AuthHandlers) GetUserIDByEmail(email string) (string, error) {
	var userID string
	err := h.DB.Get(&userID, "SELECT id FROM user_profiles WHERE email = $1", email)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}
	return userID, nil
}

// createUserProfileWithoutPassword creates a user_profiles record with password_set = false
func (h *AuthHandlers) createUserProfileWithoutPassword(userID, email, fullName string) error {
	if fullName == "" {
		fullName = strings.Split(email, "@")[0]
	}

	_, err := h.DB.Exec(
		`INSERT INTO user_profiles (id, email, full_name, role, is_active, password_set, created_at, updated_at)
		 VALUES ($1, $2, $3, 'user', TRUE, FALSE, NOW(), NOW())
		 ON CONFLICT (id) DO UPDATE SET password_set = FALSE, updated_at = NOW()`,
		userID, email, fullName,
	)
	return err
}

// createUserProfile creates a user_profiles record for a newly signed up Supabase user
func (h *AuthHandlers) createUserProfile(userID, email, fullName string) error {
	// Check if profile already exists (idempotent)
	var exists bool
	err := h.DB.Get(&exists, "SELECT EXISTS(SELECT 1 FROM user_profiles WHERE id = $1)", userID)
	if err == nil && exists {
		return nil // Already exists, nothing to do
	}

	// Default full_name to email prefix if not provided
	if fullName == "" {
		fullName = strings.Split(email, "@")[0]
	}

	_, err = h.DB.Exec(
		`INSERT INTO user_profiles (id, email, full_name, role, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, 'user', TRUE, NOW(), NOW())
		 ON CONFLICT (id) DO NOTHING`,
		userID, email, fullName,
	)
	return err
}

// Helpers for mock auth flow
func (h *AuthHandlers) ensureUserExists(email string) (string, string, error) {
	// Check if user already exists - fetch both id and role from database
	var userData struct {
		ID   string `db:"id"`
		Role string `db:"role"`
	}
	err := h.DB.Get(&userData, "SELECT id, role FROM user_profiles WHERE email = $1", email)
	if err == nil && userData.ID != "" {
		return userData.ID, userData.Role, nil
	}

	// User doesn't exist - create with default role based on email
	role := "user"
	if strings.Contains(strings.ToLower(email), "admin") {
		role = "admin"
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

// supabaseErrorTranslations maps Supabase error codes to Portuguese messages
var supabaseErrorTranslations = map[string]string{
	"user_already_exists":        "Este e-mail já está cadastrado",
	"invalid_credentials":        "E-mail ou senha inválidos",
	"email_not_confirmed":        "E-mail ainda não foi verificado",
	"user_not_found":             "Usuário não encontrado",
	"invalid_grant":              "Credenciais inválidas",
	"email_address_invalid":      "Endereço de e-mail inválido",
	"weak_password":              "Senha muito fraca. Use pelo menos 6 caracteres",
	"over_request_rate_limit":    "Muitas tentativas. Aguarde alguns minutos",
	"over_email_send_rate_limit": "Muitos e-mails enviados. Aguarde alguns minutos",
	"signup_disabled":            "Cadastro temporariamente desabilitado",
	"user_banned":                "Conta suspensa",
	"session_not_found":          "Sessão não encontrada",
	"flow_state_not_found":       "Sessão expirada. Tente novamente",
	"flow_state_expired":         "Sessão expirada. Tente novamente",
	"same_password":              "A nova senha deve ser diferente da atual",
	"validation_failed":          "Dados inválidos. Verifique os campos",
}

// translateSupabaseError extracts the error code from Supabase response and returns a Portuguese message
func translateSupabaseError(body []byte) string {
	var supaErr struct {
		ErrorCode string `json:"error_code"`
		Msg       string `json:"msg"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal(body, &supaErr); err != nil {
		return "Erro de autenticação. Tente novamente"
	}

	// Check error_code first (newer Supabase format)
	if supaErr.ErrorCode != "" {
		if translated, ok := supabaseErrorTranslations[supaErr.ErrorCode]; ok {
			return translated
		}
	}

	// Fallback to error field (older format)
	if supaErr.Error != "" {
		if translated, ok := supabaseErrorTranslations[supaErr.Error]; ok {
			return translated
		}
	}

	// Default message
	return "Erro de autenticação. Tente novamente"
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
		translatedMsg := translateSupabaseError(body)
		return nil, fmt.Errorf("%s", translatedMsg)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// extractUserIDFromResponse safely extracts the user ID from a Supabase auth response
// Returns empty string if extraction fails (graceful degradation)
func (h *AuthHandlers) extractUserIDFromResponse(resp map[string]interface{}) string {
	user, ok := resp["user"].(map[string]interface{})
	if !ok {
		h.Logger.Debug().Msg("No 'user' object in auth response")
		return ""
	}
	id, ok := user["id"].(string)
	if !ok {
		h.Logger.Debug().Msg("No 'id' string in user object")
		return ""
	}
	return id
}

// linkAnonymousSubmissions links anonymous submissions to a newly signed-up user
// Runs in a goroutine to avoid blocking the signup response
func (h *AuthHandlers) linkAnonymousSubmissions(userIDStr, email string) {
	if h.SubmissionService == nil {
		return // Graceful degradation if service not configured
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.Logger.Warn().Str("user_id", userIDStr).Err(err).Msg("Invalid user ID format - cannot link submissions")
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := h.SubmissionService.LinkAnonymousToUser(ctx, userID, email); err != nil {
			h.Logger.Error().
				Err(err).
				Str("email", email).
				Str("user_id", userIDStr).
				Msg("Failed to link anonymous submissions")
		} else {
			h.Logger.Info().
				Str("email", email).
				Str("user_id", userIDStr).
				Msg("Anonymous submissions linked to new user")
		}
	}()
}
