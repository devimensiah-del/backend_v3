package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.Nop()

	tests := []struct {
		name           string
		allowedOrigins string
		requestOrigin  string
		requestMethod  string
		requestPath    string
		expectedStatus int
		expectedHeader string
	}{
		{
			name:           "Valid origin allowed",
			allowedOrigins: "http://localhost:3000",
			requestOrigin:  "http://localhost:3000",
			requestMethod:  "GET",
			requestPath:    "/api/v1/submissions",
			expectedStatus: http.StatusOK,
			expectedHeader: "http://localhost:3000",
		},
		{
			name:           "Health check without origin allowed",
			allowedOrigins: "http://localhost:3000",
			requestOrigin:  "",
			requestMethod:  "GET",
			requestPath:    "/health",
			expectedStatus: http.StatusOK,
			expectedHeader: "*",
		},
		{
			name:           "POST without origin rejected",
			allowedOrigins: "http://localhost:3000",
			requestOrigin:  "",
			requestMethod:  "POST",
			requestPath:    "/api/v1/submissions",
			expectedStatus: http.StatusForbidden,
			expectedHeader: "",
		},
		{
			name:           "GET without origin allowed",
			allowedOrigins: "http://localhost:3000",
			requestOrigin:  "",
			requestMethod:  "GET",
			requestPath:    "/api/v1/submissions",
			expectedStatus: http.StatusOK,
			expectedHeader: "",
		},
		{
			name:           "Unauthorized origin rejected",
			allowedOrigins: "http://localhost:3000",
			requestOrigin:  "http://evil.com",
			requestMethod:  "GET",
			requestPath:    "/api/v1/submissions",
			expectedStatus: http.StatusForbidden,
			expectedHeader: "",
		},
		{
			name:           "Multiple origins - first match",
			allowedOrigins: "http://localhost:3000,https://app.example.com",
			requestOrigin:  "http://localhost:3000",
			requestMethod:  "GET",
			requestPath:    "/api/v1/submissions",
			expectedStatus: http.StatusOK,
			expectedHeader: "http://localhost:3000",
		},
		{
			name:           "Multiple origins - second match",
			allowedOrigins: "http://localhost:3000,https://app.example.com",
			requestOrigin:  "https://app.example.com",
			requestMethod:  "GET",
			requestPath:    "/api/v1/submissions",
			expectedStatus: http.StatusOK,
			expectedHeader: "https://app.example.com",
		},
		{
			name:           "OPTIONS preflight request",
			allowedOrigins: "http://localhost:3000",
			requestOrigin:  "http://localhost:3000",
			requestMethod:  "OPTIONS",
			requestPath:    "/api/v1/submissions",
			expectedStatus: http.StatusNoContent,
			expectedHeader: "http://localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			router := gin.New()
			router.Use(CORSMiddleware(tt.allowedOrigins, logger))
			router.Any(tt.requestPath, func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			// Create request
			req, _ := http.NewRequest(tt.requestMethod, tt.requestPath, nil)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}

			// Execute
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedHeader != "" {
				assert.Equal(t, tt.expectedHeader, w.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestAuthRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Allow requests under limit", func(t *testing.T) {
		limiter := NewAuthRateLimiter()
		router := gin.New()
		router.Use(AuthRateLimitMiddleware(limiter))
		router.POST("/auth/login", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		// Make 5 requests (should all succeed)
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest("POST", "/auth/login", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}
	})

	t.Run("Block requests over limit", func(t *testing.T) {
		limiter := NewAuthRateLimiter()
		router := gin.New()
		router.Use(AuthRateLimitMiddleware(limiter))
		router.POST("/auth/login", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		// Make 6 requests (6th should be blocked)
		for i := 0; i < 6; i++ {
			req, _ := http.NewRequest("POST", "/auth/login", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if i < 5 {
				assert.Equal(t, http.StatusOK, w.Code)
			} else {
				assert.Equal(t, http.StatusTooManyRequests, w.Code)
				assert.Contains(t, w.Header().Get("Retry-After"), "")
			}
		}
	})

	t.Run("Different IPs have separate limits", func(t *testing.T) {
		limiter := NewAuthRateLimiter()
		router := gin.New()
		router.Use(AuthRateLimitMiddleware(limiter))
		router.POST("/auth/login", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		// IP 1: Make 5 requests
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest("POST", "/auth/login", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// IP 2: Should still be allowed (different counter)
		req, _ := http.NewRequest("POST", "/auth/login", nil)
		req.RemoteAddr = "192.168.1.101:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		requestID, exists := c.Get("request_id")
		assert.True(t, exists)
		assert.NotEmpty(t, requestID)
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestRecoveryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.Nop()

	router := gin.New()
	router.Use(RecoveryMiddleware(logger))
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req, _ := http.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should recover and return 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
