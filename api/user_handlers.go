package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx" // Added for db
	"github.com/rs/zerolog"   // Added for logger
)

// Handlers holds all service dependencies and configuration for user handlers
type UserHandlers struct {
	DB     *sqlx.DB
	Logger zerolog.Logger
}

// NewUserHandlers creates a new user.Handlers instance with all dependencies
func NewUserHandlers(
	db *sqlx.DB,
	logger zerolog.Logger,
) *UserHandlers {
	return &UserHandlers{
		DB:     db,
		Logger: logger,
	}
}

// UpdateUserProfile handles PUT /api/v1/user/profile
func (h *UserHandlers) UpdateUserProfile(c *gin.Context) {
	// Get user ID from context (set by AuthMiddleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal error",
			Message: "Invalid user ID format",
		})
		return
	}

	// Parse request body
	var req struct {
		FullName *string `json:"fullName"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Invalid JSON payload",
		})
		return
	}

	// Update user profile in database
	query := `UPDATE user_profiles SET full_name = $1, updated_at = NOW() WHERE id = $2 RETURNING id, email, full_name, role, is_active, created_at, updated_at`

	var profile UserProfile
	err := h.DB.Get(&profile, query, req.FullName, userIDStr)
	if err != nil {
		h.Logger.Error().Err(err).Str("user_id", userIDStr).Msg("Failed to update user profile")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update profile",
			Message: "Could not update user profile",
		})
		return
	}

	c.JSON(http.StatusOK, UserProfileResponse{
		User: profile,
	})
}

// DeleteAccount handles DELETE /api/v1/user
// Soft-deactivates the account (is_active = FALSE) to preserve referential integrity.
func (h *UserHandlers) DeleteAccount(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal error",
			Message: "Invalid user ID format",
		})
		return
	}

	result, err := h.DB.Exec(`
		UPDATE user_profiles
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1
	`, userIDStr)
	if err != nil {
		h.Logger.Error().Err(err).Str("user_id", userIDStr).Msg("Failed to deactivate account")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete account",
			Message: "Could not deactivate account",
		})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Message: "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Account deactivated",
	})
}
