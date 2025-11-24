package api

import (
	"fmt"
	"strings"

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

// employeeCountToSize converts employee count to a size category string
func employeeCountToSize(count *int) *string {
	if count == nil {
		return nil
	}

	var size string
	switch {
	case *count <= 10:
		size = "1-10"
	case *count <= 50:
		size = "11-50"
	case *count <= 200:
		size = "51-200"
	case *count <= 500:
		size = "201-500"
	case *count <= 1000:
		size = "501-1000"
	case *count <= 5000:
		size = "1001-5000"
	default:
		size = "5000+"
	}

	return &size
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
