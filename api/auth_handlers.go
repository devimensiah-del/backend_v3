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
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Não autorizado", Message: "Usuário não autenticado"})
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
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Não encontrado", Message: "Perfil do usuário não encontrado"})
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
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Falha no login", Message: err.Error()})
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

	// Extract userID from Supabase response and link anonymous submissions
	if userID := h.extractUserIDFromResponse(resp); userID != "" {
		h.linkAnonymousSubmissions(userID, req.Email)
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
