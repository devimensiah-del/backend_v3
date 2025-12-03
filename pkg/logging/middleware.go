package logging

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Middleware creates a Gin middleware that attaches a logger to each request
// It generates a request_id (UUID), extracts user_id from context (if present),
// attaches the logger to gin context, sets X-Request-ID header, and logs request completion
func Middleware(baseLogger *Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Generate unique request ID
		requestID := uuid.New().String()

		// Start with base logger + request_id
		logger := baseLogger.WithRequestID(requestID)

		// Extract user_id from JWT auth middleware if present
		if userID, exists := c.Get("userID"); exists {
			if userIDStr, ok := userID.(string); ok {
				logger = logger.WithUserID(userIDStr)
			}
		}

		// Attach logger to gin context
		c.Set("logger", logger)

		// Set response header for request tracing
		c.Writer.Header().Set("X-Request-ID", requestID)

		// Process request
		c.Next()

		// Log request completion
		duration := time.Since(start)
		statusCode := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()

		logEvent := logger.Info()
		if statusCode >= 500 {
			logEvent = logger.Error()
		} else if statusCode >= 400 {
			logEvent = logger.Warn()
		}

		logEvent.
			Str("method", method).
			Str("path", path).
			Int("status", statusCode).
			Dur("latency", duration).
			Int64("duration_ms", duration.Milliseconds()).
			Str("ip", clientIP).
			Int("response_size", c.Writer.Size()).
			Msg("HTTP request completed")
	}
}
