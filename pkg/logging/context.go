package logging

import (
	"context"

	"github.com/gin-gonic/gin"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const loggerKey contextKey = "logger"

// FromContext extracts the logger from a context.Context
// Returns a default logger if not found
func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(loggerKey).(*Logger); ok {
		return logger
	}
	// Return default logger if not found in context
	return NewLogger("production")
}

// ToContext stores the logger in a context.Context
func ToContext(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromGin extracts the logger from a gin.Context
// Returns a default logger if not found
func FromGin(c *gin.Context) *Logger {
	if logger, exists := c.Get("logger"); exists {
		if typedLogger, ok := logger.(*Logger); ok {
			return typedLogger
		}
	}
	// Return default logger if not found in gin context
	return NewLogger("production")
}
