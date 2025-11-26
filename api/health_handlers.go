package api

import (
	"fmt"
	"net/http"
	"time"

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

// HealthMetrics represents comprehensive system metrics
type HealthMetrics struct {
	SubmissionsLast24h     int      `json:"submissions_last_24h"`
	EnrichmentSuccessRate  string   `json:"enrichment_success_rate"`
	AnalysisSuccessRate    string   `json:"analysis_success_rate"`
	AvgAnalysisTimeSeconds float64  `json:"avg_analysis_time_seconds"`
	TotalCostLast24hUSD    float64  `json:"total_cost_last_24h_usd"`
	TotalTokensLast24h     int64    `json:"total_tokens_last_24h"`
	LLMRequestsLast24h     int64    `json:"llm_requests_last_24h"`
	ErrorsLast24h          []string `json:"errors_last_24h"`
	LastUpdated            string   `json:"last_updated"`
}

// GetMetrics handles GET /api/v1/admin/metrics with system-wide statistics
func (h *Handler) GetMetrics(c *gin.Context) {
	ctx := c.Request.Context()
	since := time.Now().Add(-24 * time.Hour)

	metrics := HealthMetrics{
		ErrorsLast24h: []string{},
		LastUpdated:   time.Now().Format(time.RFC3339),
	}

	// Query submissions count
	var submissionCount int
	err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM submissions WHERE created_at > $1
	`, since).Scan(&submissionCount)
	if err == nil {
		metrics.SubmissionsLast24h = submissionCount
	}

	// Query enrichment success rate
	var enrichTotal, enrichSuccess int
	err = h.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status IN ('completed', 'approved'))
		FROM enrichments WHERE created_at > $1
	`, since).Scan(&enrichTotal, &enrichSuccess)
	if err == nil {
		metrics.EnrichmentSuccessRate = fmt.Sprintf("%.0f%%", safePercent(enrichSuccess, enrichTotal))
	}

	// Query analysis success rate
	var analysisTotal, analysisSuccess int
	err = h.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status IN ('completed', 'approved', 'sent'))
		FROM analyses WHERE created_at > $1
	`, since).Scan(&analysisTotal, &analysisSuccess)
	if err == nil {
		metrics.AnalysisSuccessRate = fmt.Sprintf("%.0f%%", safePercent(analysisSuccess, analysisTotal))
	}

	// Query avg analysis time
	var avgTime float64
	err = h.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(processing_time_ms) / 1000.0, 0)
		FROM analyses
		WHERE status IN ('completed', 'approved', 'sent') AND created_at > $1
	`, since).Scan(&avgTime)
	if err == nil {
		metrics.AvgAnalysisTimeSeconds = avgTime
	}

	// Query LLM usage (if table exists)
	var totalCost float64
	var totalTokens, llmRequests int64
	err = h.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(cost_usd), 0),
			COALESCE(SUM(total_tokens), 0),
			COUNT(*)
		FROM llm_usage_logs WHERE created_at > $1
	`, since).Scan(&totalCost, &totalTokens, &llmRequests)
	if err == nil {
		metrics.TotalCostLast24hUSD = totalCost
		metrics.TotalTokensLast24h = totalTokens
		metrics.LLMRequestsLast24h = llmRequests
	}

	// Query recent errors
	rows, err := h.db.QueryContext(ctx, `
		SELECT DISTINCT error_message FROM (
			SELECT error_message FROM enrichments
			WHERE status = 'failed' AND created_at > $1 AND error_message IS NOT NULL
			UNION ALL
			SELECT error_message FROM analyses
			WHERE status = 'failed' AND created_at > $1 AND error_message IS NOT NULL
		) e LIMIT 10
	`, since)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var msg string
			if rows.Scan(&msg) == nil && msg != "" {
				metrics.ErrorsLast24h = append(metrics.ErrorsLast24h, msg)
			}
		}
	}

	h.logger.Info().
		Int("submissions", metrics.SubmissionsLast24h).
		Float64("cost_usd", metrics.TotalCostLast24hUSD).
		Int64("llm_requests", metrics.LLMRequestsLast24h).
		Msg("Metrics retrieved")

	c.JSON(http.StatusOK, metrics)
}

// safePercent calculates percentage, returning 100 if denominator is 0
func safePercent(num, denom int) float64 {
	if denom == 0 {
		return 100.0 // No attempts = 100% success (vacuous truth)
	}
	return float64(num) / float64(denom) * 100
}
