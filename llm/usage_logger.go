package llm

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// UsageLogger handles persistent logging of LLM API usage to database
type UsageLogger struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewUsageLogger creates a new usage logger
func NewUsageLogger(db *sqlx.DB, logger zerolog.Logger) *UsageLogger {
	return &UsageLogger{
		db:     db,
		logger: logger.With().Str("component", "llm_usage_logger").Logger(),
	}
}

// LogUsage records an LLM API call to the database
func (l *UsageLogger) LogUsage(ctx context.Context, log *UsageLog) error {
	if l.db == nil {
		l.logger.Warn().Msg("UsageLogger: database not configured, skipping persist")
		return nil
	}

	query := `
		INSERT INTO llm_usage_logs (
			submission_id, framework_name, model_used,
			input_tokens, output_tokens, total_tokens,
			cost_usd, latency_ms, is_fallback, error_message
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		) RETURNING id, created_at
	`

	var id, createdAt string
	err := l.db.QueryRowContext(ctx, query,
		log.SubmissionID,
		log.FrameworkName,
		log.ModelUsed,
		log.InputTokens,
		log.OutputTokens,
		log.TotalTokens,
		log.CostUSD,
		log.LatencyMS,
		log.IsFallback,
		log.ErrorMessage,
	).Scan(&id, &createdAt)

	if err != nil {
		l.logger.Error().Err(err).
			Str("framework", log.FrameworkName).
			Str("model", log.ModelUsed).
			Msg("Failed to log LLM usage")
		return fmt.Errorf("failed to log LLM usage: %w", err)
	}

	l.logger.Debug().
		Str("id", id).
		Str("framework", log.FrameworkName).
		Str("model", log.ModelUsed).
		Int("input_tokens", log.InputTokens).
		Int("output_tokens", log.OutputTokens).
		Float64("cost_usd", log.CostUSD).
		Msg("LLM usage logged")

	return nil
}

// LogFromResponse creates a UsageLog from a Response and logs it
func (l *UsageLogger) LogFromResponse(ctx context.Context, resp *Response, submissionID *string, frameworkName string, isFallback bool, errMsg *string) error {
	if resp == nil {
		return nil
	}

	log := &UsageLog{
		SubmissionID:  submissionID,
		FrameworkName: frameworkName,
		ModelUsed:     resp.Model,
		InputTokens:   resp.InputTokens,
		OutputTokens:  resp.OutputTokens,
		TotalTokens:   resp.Tokens,
		CostUSD:       resp.CostUSD,
		LatencyMS:     &resp.LatencyMS,
		IsFallback:    isFallback,
		ErrorMessage:  errMsg,
	}

	return l.LogUsage(ctx, log)
}

// UsageSummary represents aggregated usage statistics
type UsageSummary struct {
	TotalCostUSD      float64 `json:"total_cost_usd" db:"total_cost_usd"`
	TotalInputTokens  int64   `json:"total_input_tokens" db:"total_input_tokens"`
	TotalOutputTokens int64   `json:"total_output_tokens" db:"total_output_tokens"`
	TotalRequests     int64   `json:"total_requests" db:"total_requests"`
	AvgLatencyMS      float64 `json:"avg_latency_ms" db:"avg_latency_ms"`
	FallbackCount     int64   `json:"fallback_count" db:"fallback_count"`
	ErrorCount        int64   `json:"error_count" db:"error_count"`
}

// GetUsageSummary returns aggregated usage statistics for a time period
func (l *UsageLogger) GetUsageSummary(ctx context.Context, since string) (*UsageSummary, error) {
	if l.db == nil {
		return &UsageSummary{}, nil
	}

	query := `
		SELECT
			COALESCE(SUM(cost_usd), 0) as total_cost_usd,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COUNT(*) as total_requests,
			COALESCE(AVG(latency_ms), 0) as avg_latency_ms,
			COUNT(*) FILTER (WHERE is_fallback = true) as fallback_count,
			COUNT(*) FILTER (WHERE error_message IS NOT NULL) as error_count
		FROM llm_usage_logs
		WHERE created_at > $1
	`

	var summary UsageSummary
	err := l.db.GetContext(ctx, &summary, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage summary: %w", err)
	}

	return &summary, nil
}

// ModelUsageSummary represents usage broken down by model
type ModelUsageSummary struct {
	Model        string  `json:"model" db:"model_used"`
	TotalCost    float64 `json:"total_cost" db:"total_cost"`
	RequestCount int64   `json:"request_count" db:"request_count"`
	AvgLatency   float64 `json:"avg_latency" db:"avg_latency"`
}

// GetUsageByModel returns usage statistics grouped by model
func (l *UsageLogger) GetUsageByModel(ctx context.Context, since string) ([]ModelUsageSummary, error) {
	if l.db == nil {
		return []ModelUsageSummary{}, nil
	}

	query := `
		SELECT
			model_used,
			COALESCE(SUM(cost_usd), 0) as total_cost,
			COUNT(*) as request_count,
			COALESCE(AVG(latency_ms), 0) as avg_latency
		FROM llm_usage_logs
		WHERE created_at > $1
		GROUP BY model_used
		ORDER BY total_cost DESC
	`

	var summaries []ModelUsageSummary
	err := l.db.SelectContext(ctx, &summaries, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage by model: %w", err)
	}

	return summaries, nil
}

// SubmissionUsage represents usage for a single submission
type SubmissionUsage struct {
	SubmissionID string  `json:"submission_id" db:"submission_id"`
	TotalCost    float64 `json:"total_cost" db:"total_cost"`
	TotalTokens  int64   `json:"total_tokens" db:"total_tokens"`
	RequestCount int64   `json:"request_count" db:"request_count"`
}

// GetUsageBySubmission returns total cost and tokens for a specific submission
func (l *UsageLogger) GetUsageBySubmission(ctx context.Context, submissionID string) (*SubmissionUsage, error) {
	if l.db == nil {
		return &SubmissionUsage{SubmissionID: submissionID}, nil
	}

	query := `
		SELECT
			submission_id,
			COALESCE(SUM(cost_usd), 0) as total_cost,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COUNT(*) as request_count
		FROM llm_usage_logs
		WHERE submission_id = $1
		GROUP BY submission_id
	`

	var usage SubmissionUsage
	err := l.db.GetContext(ctx, &usage, query, submissionID)
	if err != nil {
		// Return empty if no logs found
		return &SubmissionUsage{SubmissionID: submissionID}, nil
	}

	return &usage, nil
}
