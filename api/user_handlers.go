package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// UpdateUserProfile handles PUT /api/v1/user/profile
func (h *Handler) UpdateUserProfile(c *gin.Context) {
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
	err := h.db.Get(&profile, query, req.FullName, userIDStr)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userIDStr).Msg("Failed to update user profile")
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
