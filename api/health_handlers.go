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
		health["status"] = "unhealthy"
		services["database"] = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, health)
		return
	}

	// Check Redis if available
	if h.redisClient != nil {
		if err := h.redisClient.Ping(c.Request.Context()).Err(); err != nil {
			health["status"] = "unhealthy"
			services["redis"] = "unhealthy"
			c.JSON(http.StatusServiceUnavailable, health)
			return
		}
	}

	c.JSON(http.StatusOK, health)
}
