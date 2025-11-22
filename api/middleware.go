package api

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
)

// CORSMiddleware handles Cross-Origin Resource Sharing
// Supports multiple origins separated by commas
func CORSMiddleware(allowedOrigins string, logger zerolog.Logger) gin.HandlerFunc {
	origins := strings.Split(allowedOrigins, ",")
	return func(c *gin.Context) {
		requestOrigin := c.Request.Header.Get("Origin")
		path := c.Request.URL.Path

		// CRITICAL FIX: Allow health checks and monitoring endpoints to bypass CORS
		// Railway, Docker, Kubernetes health checkers don't send Origin headers
		if path == "/health" || path == "/metrics" || path == "/ready" || path == "/live" {
			logger.Debug().
				Str("path", path).
				Str("method", c.Request.Method).
				Str("user_agent", c.Request.UserAgent()).
				Msg("Health/monitoring endpoint - skipping CORS")
			c.Next()
			return
		}

		// For browser requests without Origin header (like direct URL access), allow through
		if requestOrigin == "" {
			logger.Debug().
				Str("path", path).
				Str("method", c.Request.Method).
				Msg("No Origin header - allowing request")
			c.Next()
			return
		}

		// Check if the request origin is in the allowed list
		originAllowed := false
		for _, allowedOrigin := range origins {
			trimmedOrigin := strings.TrimSpace(allowedOrigin)
			if trimmedOrigin == requestOrigin || trimmedOrigin == "*" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", requestOrigin)
				originAllowed = true
				break
			}
		}

		// SECURITY: Reject unauthorized origins explicitly
		if !originAllowed {
			logger.Warn().
				Str("origin", requestOrigin).
				Str("path", path).
				Str("allowed_origins", allowedOrigins).
				Msg("CORS: Origin not allowed")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"message": "Origin not allowed by CORS policy",
			})
			return
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			logger.Debug().Str("origin", requestOrigin).Msg("CORS preflight request")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// AuthMiddleware validates JWT tokens from Supabase using the HMAC secret
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "Missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and Validate Token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "Invalid or expired token"})
			c.Abort()
			return
		}

		// Extract Claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// Supabase stores the user UUID in "sub"
			if sub, ok := claims["sub"].(string); ok {
				c.Set("userID", sub)
			}

			// Check for role (usually in 'role' or 'app_metadata')
			role := "user"
			if r, ok := claims["role"].(string); ok {
				role = r
			}
			// Supabase specific: check app_metadata for admin flag
			if appMeta, ok := claims["app_metadata"].(map[string]interface{}); ok {
				if appRole, ok := appMeta["role"].(string); ok {
					role = appRole
				}
			}
			c.Set("userRole", role)
		}

		c.Next()
	}
}

// AdminAuthMiddleware ensures user has admin privileges
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("userRole")
		if !exists {
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Access denied"})
			c.Abort()
			return
		}

		// Check if role is admin, super_admin, or service_role (Supabase)
		userRole := fmt.Sprintf("%v", role)
		if userRole != "admin" && userRole != "super_admin" && userRole != "service_role" {
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// LoggingMiddleware logs all HTTP requests with detailed context
func LoggingMiddleware(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method
		userAgent := c.Request.UserAgent()
		origin := c.Request.Header.Get("Origin")

		// Log request START (especially for health checks)
		logger.Info().
			Str("method", method).
			Str("path", path).
			Str("query", query).
			Str("ip", c.ClientIP()).
			Str("user_agent", userAgent).
			Str("origin", origin).
			Msg("→ HTTP request started")

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// Log request completion with status
		logEvent := logger.Info()
		if statusCode >= 500 {
			logEvent = logger.Error()
		} else if statusCode >= 400 {
			logEvent = logger.Warn()
		}

		logEvent.
			Str("method", method).
			Str("path", path).
			Str("query", query).
			Int("status", statusCode).
			Dur("latency", latency).
			Str("ip", c.ClientIP()).
			Str("user_agent", userAgent).
			Int("response_size", c.Writer.Size()).
			Msg("← HTTP request completed")
	}
}

// RecoveryMiddleware recovers from panics with detailed logging
func RecoveryMiddleware(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error().
					Interface("panic", err).
					Str("path", c.Request.URL.Path).
					Str("method", c.Request.Method).
					Str("ip", c.ClientIP()).
					Msg("PANIC recovered in HTTP handler")
				c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error", Message: "Unexpected error"})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// RateLimitMiddleware limits requests per IP
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	// Simple in-memory implementation for MVP with thread-safety
	type rateLimiter struct {
		count     int
		resetTime time.Time
		mu        sync.Mutex
	}

	limiters := make(map[string]*rateLimiter)
	var mapMu sync.RWMutex

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		// Get or create limiter for this IP
		mapMu.RLock()
		limiter, exists := limiters[ip]
		mapMu.RUnlock()

		if !exists {
			mapMu.Lock()
			// Double-check after acquiring write lock
			if limiter, exists = limiters[ip]; !exists {
				limiter = &rateLimiter{count: 0, resetTime: now.Add(1 * time.Minute)}
				limiters[ip] = limiter
			}
			mapMu.Unlock()
		}

		// Lock the specific IP's limiter
		limiter.mu.Lock()
		defer limiter.mu.Unlock()

		// Reset if time window expired
		if now.After(limiter.resetTime) {
			limiter.count = 0
			limiter.resetTime = now.Add(1 * time.Minute)
		}

		// Check rate limit
		if limiter.count >= requestsPerMinute {
			c.JSON(http.StatusTooManyRequests, ErrorResponse{Error: "Rate limit exceeded", Message: "Too many requests"})
			c.Abort()
			return
		}

		limiter.count++
		c.Next()
	}
}

// RequestIDMiddleware adds X-Request-ID header
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := fmt.Sprintf("%d", time.Now().UnixNano())
		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}
