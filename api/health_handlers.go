package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// --- HEALTH ENDPOINT ---

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	services := gin.H{
		"database": "healthy",
		"redis":    "healthy",
	}

	health := gin.H{
		"status":   "healthy",
		"services": services,
	}

	// Check database
	if err := h.db.Ping(); err != nil {
		h.logger.Error().
			Err(err).
			Str("endpoint", "/health").
			Msg("Health check FAILED: Database ping failed")
		health["status"] = "unhealthy"
		services["database"] = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, health)
		return
	}

	// Check Redis if available
	if h.redisClient != nil {
		if err := h.redisClient.Ping(c.Request.Context()).Err(); err != nil {
			h.logger.Error().
				Err(err).
				Str("endpoint", "/health").
				Msg("Health check FAILED: Redis ping failed")
			health["status"] = "unhealthy"
			services["redis"] = "unhealthy"
			c.JSON(http.StatusServiceUnavailable, health)
			return
		}
	}

	// Log successful health checks at debug level to avoid log spam
	h.logger.Debug().
		Str("endpoint", "/health").
		Msg("Health check passed")

	c.JSON(http.StatusOK, health)
}
