package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
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

// NOTE: GetMetrics moved to admin_handlers.go (admin-only endpoint)

// QueueDiagnostic handles GET /admin/queue-diagnostic
// Returns queue status and can clear stuck tasks
func (h *Handler) QueueDiagnostic(c *gin.Context) {
	// Create inspector using same Redis config
	redisAddr := h.redisClient.Options().Addr
	redisPassword := h.redisClient.Options().Password

	inspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
	})
	defer inspector.Close()

	// Get queue info for all queues
	queues := []string{"critical", "default", "low"}
	queueInfo := make(map[string]interface{})

	for _, q := range queues {
		info, err := inspector.GetQueueInfo(q)
		if err != nil {
			queueInfo[q] = gin.H{"error": err.Error()}
			continue
		}
		queueInfo[q] = gin.H{
			"pending":   info.Pending,
			"active":    info.Active,
			"scheduled": info.Scheduled,
			"retry":     info.Retry,
			"archived":  info.Archived,
			"completed": info.Completed,
			"processed": info.Processed,
			"failed":    info.Failed,
		}

		h.logger.Info().
			Str("queue", q).
			Int("pending", info.Pending).
			Int("active", info.Active).
			Int("scheduled", info.Scheduled).
			Int("retry", info.Retry).
			Int("archived", info.Archived).
			Msg("📊 Queue diagnostic")
	}

	// Show archived tasks details (last 5)
	archivedDetails := make([]map[string]interface{}, 0)
	archivedTasks, err := inspector.ListArchivedTasks("default", asynq.PageSize(5))
	if err == nil {
		for _, t := range archivedTasks {
			archivedDetails = append(archivedDetails, map[string]interface{}{
				"id":           t.ID,
				"type":         t.Type,
				"payload":      string(t.Payload),
				"last_error":   t.LastErr,
				"last_failed":  t.LastFailedAt,
				"max_retry":    t.MaxRetry,
				"retried":      t.Retried,
			})
		}
	}
	queueInfo["archived_details"] = archivedDetails

	// Check if clear=true param is set
	if c.Query("clear") == "true" {
		h.logger.Warn().Msg("🧹 Clearing all queues...")
		for _, q := range queues {
			// Delete all tasks in different states
			deleted, err := inspector.DeleteAllPendingTasks(q)
			if err != nil {
				h.logger.Error().Err(err).Str("queue", q).Msg("Failed to clear pending tasks")
			} else {
				h.logger.Info().Int("deleted", deleted).Str("queue", q).Msg("Cleared pending tasks")
			}

			deleted, err = inspector.DeleteAllScheduledTasks(q)
			if err != nil {
				h.logger.Error().Err(err).Str("queue", q).Msg("Failed to clear scheduled tasks")
			} else {
				h.logger.Info().Int("deleted", deleted).Str("queue", q).Msg("Cleared scheduled tasks")
			}

			deleted, err = inspector.DeleteAllRetryTasks(q)
			if err != nil {
				h.logger.Error().Err(err).Str("queue", q).Msg("Failed to clear retry tasks")
			} else {
				h.logger.Info().Int("deleted", deleted).Str("queue", q).Msg("Cleared retry tasks")
			}

			deleted, err = inspector.DeleteAllArchivedTasks(q)
			if err != nil {
				h.logger.Error().Err(err).Str("queue", q).Msg("Failed to clear archived tasks")
			} else {
				h.logger.Info().Int("deleted", deleted).Str("queue", q).Msg("Cleared archived tasks")
			}
		}
		queueInfo["cleared"] = true
	}

	c.JSON(http.StatusOK, gin.H{
		"redis_addr": redisAddr,
		"queues":     queueInfo,
	})
}
