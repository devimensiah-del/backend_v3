package logging

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestFromContext(t *testing.T) {
	// Test with logger in context
	t.Run("logger in context", func(t *testing.T) {
		logger := NewLogger("production")
		ctx := ToContext(context.Background(), logger)

		retrieved := FromContext(ctx)
		if retrieved == nil {
			t.Error("Expected logger, got nil")
		}
	})

	// Test without logger in context (should return default)
	t.Run("no logger in context", func(t *testing.T) {
		ctx := context.Background()
		retrieved := FromContext(ctx)
		if retrieved == nil {
			t.Error("Expected default logger, got nil")
		}
	})
}

func TestToContext(t *testing.T) {
	logger := NewLogger("production")
	ctx := context.Background()

	// Store logger in context
	newCtx := ToContext(ctx, logger)

	// Verify it can be retrieved
	retrieved := FromContext(newCtx)
	if retrieved == nil {
		t.Error("Expected logger to be stored in context")
	}
}

func TestFromGin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test with logger in gin context
	t.Run("logger in gin context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		logger := NewLogger("production")
		c.Set("logger", logger)

		retrieved := FromGin(c)
		if retrieved == nil {
			t.Error("Expected logger, got nil")
		}
	})

	// Test without logger in gin context (should return default)
	t.Run("no logger in gin context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		retrieved := FromGin(c)
		if retrieved == nil {
			t.Error("Expected default logger, got nil")
		}
	})

	// Test with wrong type in gin context
	t.Run("wrong type in gin context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Set("logger", "not a logger")

		retrieved := FromGin(c)
		if retrieved == nil {
			t.Error("Expected default logger when type is wrong, got nil")
		}
	})
}
