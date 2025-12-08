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

// NOTE: Auth-specific rate limiting tests were removed when AuthRateLimiter was replaced
// with standard golang.org/x/time/rate implementation. Global rate limiting now applies
// to all endpoints including auth routes.

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
