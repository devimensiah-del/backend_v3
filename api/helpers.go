package api

import (
	"fmt"
	"strings"

	apperrors "backend_v3/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// parseUUID safely parses a UUID string and returns an error if invalid
func parseUUID(id string) (uuid.UUID, error) {
	u, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID format: %w", err)
	}
	return u, nil
}

// stringToPtr converts a string to *string, returning nil for empty strings
func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ptrToString safely converts *string to string, returning empty string for nil
func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// buildBusinessChallengeString concatenates strategic fields with " | " separator
// Skips empty fields to avoid creating empty sections
func buildBusinessChallengeString(strategicGoal, currentChallenges, competitivePosition string) string {
	var parts []string

	if strategicGoal != "" {
		parts = append(parts, strategicGoal)
	}
	if currentChallenges != "" {
		parts = append(parts, currentChallenges)
	}
	if competitivePosition != "" {
		parts = append(parts, competitivePosition)
	}

	return strings.Join(parts, " | ")
}

// respondError centralizes JSON error responses with logging.
// userMsg should be localized for the frontend (Portuguese).
func respondError(c *gin.Context, logger zerolog.Logger, status int, err error, userMsg string) {
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", c.FullPath()).
			Str("method", c.Request.Method).
			Msg("API error")
	}
	c.JSON(status, ErrorResponse{
		Error:   userMsg,
		Message: userMsg,
	})
}

// RespondAppError handles AppError responses with i18n support.
// Reads Accept-Language header and returns localized error message.
func RespondAppError(c *gin.Context, logger zerolog.Logger, err error) {
	// Get language preference from Accept-Language header
	lang := c.GetHeader("Accept-Language")
	if lang == "" {
		lang = "pt-BR" // Default to Portuguese for Brazilian market
	}

	// Log the error with internal details
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", c.FullPath()).
			Str("method", c.Request.Method).
			Msg("API error")
	}

	// Check if it's an AppError
	if appErr, ok := apperrors.AsAppError(err); ok {
		response := gin.H{
			"error": gin.H{
				"code":    appErr.Code,
				"message": appErr.LocalizedMessage(lang),
			},
		}

		// Include details if present
		if len(appErr.Details) > 0 {
			response["error"].(gin.H)["details"] = appErr.Details
		}

		c.JSON(appErr.HTTPStatus, response)
		return
	}

	// Fallback for non-AppError
	c.JSON(500, gin.H{
		"error": gin.H{
			"code":    "ERR_INTERNAL",
			"message": apperrors.ErrInternal.LocalizedMessage(lang),
		},
	})
}

// RespondValidationError returns a validation error with field details
func RespondValidationError(c *gin.Context, logger zerolog.Logger, field, message string) {
	lang := c.GetHeader("Accept-Language")
	if lang == "" {
		lang = "pt-BR"
	}

	logger.Warn().
		Str("field", field).
		Str("message", message).
		Str("path", c.FullPath()).
		Msg("Validation error")

	c.JSON(400, gin.H{
		"error": gin.H{
			"code":    "ERR_VALIDATION",
			"message": apperrors.ErrValidation.LocalizedMessage(lang),
			"details": gin.H{
				"field":   field,
				"message": message,
			},
		},
	})
}
