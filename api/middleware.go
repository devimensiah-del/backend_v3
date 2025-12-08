package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

// CORSMiddleware handles Cross-Origin Resource Sharing using gin-contrib/cors
func CORSMiddleware(allowedOrigins string, logger zerolog.Logger) gin.HandlerFunc {
	origins := strings.Split(allowedOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	config := cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "Accept", "Cache-Control", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// Special handling for health/monitoring endpoints
	corsHandler := cors.New(config)

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		requestOrigin := c.Request.Header.Get("Origin")

		// Allow all origins for health checks and monitoring endpoints
		if path == "/health" || path == "/metrics" || path == "/ready" || path == "/live" {
			logger.Debug().
				Str("path", path).
				Str("method", c.Request.Method).
				Str("user_agent", c.Request.UserAgent()).
				Msg("Health/monitoring endpoint - allowing all origins")

			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
			c.Next()
			return
		}

		// SECURITY: Reject requests without Origin header on protected routes
		// Empty Origin header can be used to bypass CORS restrictions
		if requestOrigin == "" {
			// Allow GET requests without Origin (RSS readers, curl, etc.)
			// But reject POST/PUT/DELETE which modify data
			if c.Request.Method != "GET" && c.Request.Method != "HEAD" && c.Request.Method != "OPTIONS" {
				logger.Warn().
					Str("path", path).
					Str("method", c.Request.Method).
					Str("ip", c.ClientIP()).
					Str("user_agent", c.Request.UserAgent()).
					Msg("CORS: Rejected non-GET request without Origin header")
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":   "Forbidden",
					"message": "Origin header required for this request",
				})
				return
			}
		}

		// Use standard CORS handler for all other routes
		corsHandler(c)
	}
}

// parseToken extracts and validates JWT token from request, returning userID and role
// Returns (userID, role, ok) where ok=true means token was valid
func parseToken(c *gin.Context, jwtSecret string, db *sqlx.DB) (userID, role string, ok bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", "", false
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", "", false
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
		return "", "", false
	}

	// Extract Claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", false
	}

	// Supabase stores the user UUID in "sub"
	if sub, ok := claims["sub"].(string); ok {
		userID = sub
		c.Set("userID", sub)
	}

	// Supabase may also store email in "email" claim
	if email, ok := claims["email"].(string); ok {
		c.Set("userEmail", email)
	}

	// Fetch role and email from database
	role = "user" // Default fallback
	if userID != "" && db != nil {
		var userInfo struct {
			Role  string `db:"role"`
			Email string `db:"email"`
		}
		err := db.Get(&userInfo, "SELECT role, email FROM user_profiles WHERE id = $1", userID)
		if err == nil {
			role = userInfo.Role
			// Set email from database if not already set from JWT
			if _, exists := c.Get("userEmail"); !exists && userInfo.Email != "" {
				c.Set("userEmail", userInfo.Email)
			}
		}
	}
	c.Set("userRole", role)

	return userID, role, true
}

// AuthMiddleware validates JWT tokens from Supabase using the HMAC secret
// and fetches the user's role from the database
func AuthMiddleware(jwtSecret string, db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _, ok := parseToken(c, jwtSecret, db)
		if !ok {
			// SECURITY: Log authentication failure
			secLogger := NewSecurityEventLogger(zerolog.New(os.Stderr))
			secLogger.LogAuthFailure(
				c.ClientIP(),
				c.Request.UserAgent(),
				c.Request.URL.Path,
				"Invalid or expired token",
			)

			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "Invalid or expired token"})
			c.Abort()
			return
		}

		if userID == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "Missing authorization header"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuthMiddleware parses JWT token if present but doesn't require it
// Used for public endpoints that need to behave differently for authenticated users (e.g., admin preview)
func OptionalAuthMiddleware(jwtSecret string, db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to parse token, but don't fail if it's missing or invalid
		parseToken(c, jwtSecret, db)
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
			// SECURITY: Log authorization failure
			secLogger := NewSecurityEventLogger(zerolog.New(os.Stderr))
			userID, _ := c.Get("userID")
			secLogger.LogAuthzFailure(
				fmt.Sprintf("%v", userID),
				userRole,
				c.ClientIP(),
				c.Request.URL.Path,
				c.Request.Method,
				fmt.Sprintf("Insufficient privileges: role=%s, required=admin/super_admin", userRole),
			)

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

// RateLimitMiddleware limits requests per IP using golang.org/x/time/rate
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	var visitors sync.Map

	// Cleanup goroutine to prevent memory leak
	cleanupTicker := time.NewTicker(5 * time.Minute)
	go func() {
		for range cleanupTicker.C {
			// Remove idle limiters (simple implementation - in production use LRU cache)
			visitors.Range(func(key, value interface{}) bool {
				// Keep all limiters for now - they're lightweight
				// In production, implement LRU eviction based on last access time
				return true
			})
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		// Get or create limiter for this IP
		limiterInterface, _ := visitors.LoadOrStore(ip, rate.NewLimiter(rate.Every(time.Minute/time.Duration(requestsPerMinute)), requestsPerMinute))
		limiter := limiterInterface.(*rate.Limiter)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, ErrorResponse{Error: "Rate limit exceeded", Message: "Too many requests"})
			c.Abort()
			return
		}

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
