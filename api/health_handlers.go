package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// --- HEALTH ENDPOINT ---

// HealthCheck handles GET /health with comprehensive logging
func (h *Handler) HealthCheck(c *gin.Context) {
	h.logger.Info().
		Str("endpoint", "/health").
		Str("method", c.Request.Method).
		Str("user_agent", c.Request.UserAgent()).
		Str("ip", c.ClientIP()).
		Msg("Health check requested")

	services := gin.H{
		"database": "healthy",
		"redis":    "healthy",
	}

	health := gin.H{
		"status":   "healthy",
		"services": services,
	}

	// Check database
	h.logger.Info().Msg("Health check: Pinging database...")
	if err := h.db.Ping(); err != nil {
		h.logger.Error().
			Err(err).
			Str("endpoint", "/health").
			Str("error_type", fmt.Sprintf("%T", err)).
			Msg("Health check FAILED: Database ping failed")
		health["status"] = "unhealthy"
		services["database"] = "unhealthy"
		health["error"] = err.Error()
		c.JSON(http.StatusServiceUnavailable, health)
		return
	}
	h.logger.Info().Msg("Health check: Database ping successful")

	// Check Redis if available
	if h.redisClient != nil {
		h.logger.Info().Msg("Health check: Pinging Redis...")
		if err := h.redisClient.Ping(c.Request.Context()).Err(); err != nil {
			h.logger.Error().
				Err(err).
				Str("endpoint", "/health").
				Str("error_type", fmt.Sprintf("%T", err)).
				Msg("Health check FAILED: Redis ping failed")
			health["status"] = "unhealthy"
			services["redis"] = "unhealthy"
			health["error"] = err.Error()
			c.JSON(http.StatusServiceUnavailable, health)
			return
		}
		h.logger.Info().Msg("Health check: Redis ping successful")
	} else {
		h.logger.Debug().Msg("Health check: Redis client is nil, skipping")
	}

	// Log successful health checks at INFO level for visibility
	h.logger.Info().
		Str("endpoint", "/health").
		Msg("✓ Health check PASSED - returning 200 OK")

	c.JSON(http.StatusOK, health)
}
