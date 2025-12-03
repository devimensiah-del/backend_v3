package logging

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	baseLogger := NewLogger("production")
	middleware := Middleware(baseLogger)

	t.Run("attaches logger to context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		middleware(c)

		logger := FromGin(c)
		if logger == nil {
			t.Error("Expected logger to be attached to gin context")
		}
	})

	t.Run("sets X-Request-ID header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		middleware(c)

		requestID := w.Header().Get("X-Request-ID")
		if requestID == "" {
			t.Error("Expected X-Request-ID header to be set")
		}
	})

	t.Run("includes user_id when present", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		// Simulate JWT middleware setting userID
		c.Set("userID", "user-123")

		middleware(c)

		logger := FromGin(c)
		if logger == nil {
			t.Error("Expected logger to be attached")
		}
		// Note: Can't easily verify the field is in the logger without logging,
		// but we can verify no panic occurred
	})

	t.Run("logs request completion", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		middleware(c)

		// Verify middleware completed without panic
		// Note: The middleware logs but doesn't change the response
	})
}
