package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"backend_v3/config"
	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/submission"
	jobtypes "backend_v3/jobs/types"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Job Payload Types
type AnalysisJobPayload struct {
	SubmissionID string `json:"submission_id"`
	CompanyID    string `json:"company_id"`
	ChallengeID  string `json:"challenge_id"` // REQUIRED: Links analysis to specific challenge
}

type Worker struct {
	server            *asynq.Server
	mux               *asynq.ServeMux
	submissionService *submission.Service
	enrichmentService *enrichment.Service // For reading enrichment data in analysis jobs
	analysisService   *analysis.Service
	logger            zerolog.Logger
	redisOpt          asynq.RedisClientOpt
	asynqClient       *asynq.Client // Reused client for job enqueueing
	redisClient       *redis.Client // For DLQ storage
	cfg               *config.Config
}

func NewWorker(
	cfg *config.Config,
	submissionSvc *submission.Service,
	enrichmentSvc *enrichment.Service, // DEPRECATED: Will be removed after cleanup
	analysisSvc *analysis.Service,
	logger zerolog.Logger,
) *Worker {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.RedisURL,
		Password: cfg.RedisPassword,
	}

	// Redis client for DLQ storage
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURL,
		Password: cfg.RedisPassword,
	})

	w := &Worker{
		submissionService: submissionSvc,
		enrichmentService: enrichmentSvc,
		analysisService:   analysisSvc,
		logger:            logger.With().Str("component", "worker").Logger(),
		redisOpt:          redisOpt,
		asynqClient:       asynq.NewClient(redisOpt),
		redisClient:       redisClient,
		cfg:               cfg,
	}

	// Configure Asynq server with retry logic
	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: cfg.WorkerConcurrency,
			Queues:      map[string]int{"critical": 6, "default": 3, "low": 1},

			// Exponential backoff retry function
			RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
				// Calculate exponential backoff: initial_delay * 2^n
				delay := time.Duration(cfg.JobRetryInitialDelay) * time.Second * (1 << uint(n))
				maxDelay := time.Duration(cfg.JobRetryMaxDelay) * time.Second

				// Cap at max delay
				if delay > maxDelay {
					delay = maxDelay
				}

				w.logger.Info().
					Str("task_type", t.Type()).
					Int("retry_count", n).
					Dur("delay", delay).
					Msg("Scheduling job retry with exponential backoff")

				return delay
			},

			// Error handler - only updates DB on FINAL failure
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				w.handleJobError(ctx, task, err)
			}),
		},
	)

	w.server = srv
	w.mux = asynq.NewServeMux()

	// Register handlers with retry configuration
	w.logger.Info().
		Str("task_type", jobtypes.TypeAnalysis).
		Msg("📝 Registering handler for task type")
	w.mux.HandleFunc(jobtypes.TypeAnalysis, w.HandleAnalysisJob)

	return w
}

func (w *Worker) Start() error {
	w.logger.Info().
		Str("redis_addr", w.redisOpt.Addr).
		Bool("has_password", w.redisOpt.Password != "").
		Msg("🔌 Worker connecting to Redis for job processing")

	err := w.server.Start(w.mux)
	if err != nil {
		w.logger.Error().Err(err).Msg("❌ Asynq server failed to start")
	}
	return err
}

// Stop gracefully shuts down the worker and closes connections
func (w *Worker) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	w.logger.Info().Msg("Initiating graceful worker shutdown...")

	// Shutdown Asynq server (waits for in-progress jobs)
	w.server.Shutdown()

	// Close clients
	w.asynqClient.Close()
	w.redisClient.Close()

	w.logger.Info().Msg("Worker shutdown complete")
	<-ctx.Done()
}

// Health checks worker health
func (w *Worker) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Check Redis connection
	if err := w.redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis unhealthy: %w", err)
	}

	return nil
}

// --- Job Handlers ---

func (w *Worker) HandleAnalysisJob(ctx context.Context, task *asynq.Task) error {
	// FIRST LINE - absolute minimum logging to confirm handler is called
	w.logger.Info().Msg("🔥🔥🔥 HANDLER ENTRY - analysis_job received")

	startTime := time.Now()

	// Get task metadata safely
	var taskID string
	if rw := task.ResultWriter(); rw != nil {
		taskID = rw.TaskID()
	} else {
		taskID = "unknown"
	}
	retryCount, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)

	// Debug log with payload
	w.logger.Info().
		Str("task_id", taskID).
		Str("raw_payload", string(task.Payload())).
		Msg("🔥 ANALYSIS JOB HANDLER INVOKED - Worker is processing")

	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, time.Duration(w.cfg.AnalysisTimeout)*time.Second)
	defer cancel()

	// Structured logger with correlation ID
	jobLogger := w.logger.With().
		Str("job_type", jobtypes.TypeAnalysis).
		Str("task_id", taskID).
		Str("correlation_id", taskID).
		Int("retry_count", retryCount).
		Int("max_retries", maxRetry).
		Logger()

	// 1. Parse payload
	// SAFETY: Wrap unmarshal errors with SkipRetry - malformed payloads can NEVER succeed
	var payload AnalysisJobPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		jobLogger.Error().Err(err).Msg("Failed to unmarshal analysis job payload - skipping retry (poison pill)")
		return fmt.Errorf("%w: invalid payload format: %v", asynq.SkipRetry, err)
	}

	// 2. Validate UUIDs
	_, err := uuid.Parse(payload.SubmissionID)
	if err != nil {
		jobLogger.Error().
			Err(err).
			Str("submission_id", payload.SubmissionID).
			Msg("Invalid submission UUID format")
		return asynq.SkipRetry
	}

	_, err = uuid.Parse(payload.CompanyID)
	if err != nil {
		jobLogger.Error().
			Err(err).
			Str("company_id", payload.CompanyID).
			Msg("Invalid company UUID format")
		return asynq.SkipRetry
	}

	// Validate challenge_id (REQUIRED for data integrity)
	challengeUUID, err := uuid.Parse(payload.ChallengeID)
	if err != nil {
		jobLogger.Error().
			Err(err).
			Str("challenge_id", payload.ChallengeID).
			Msg("Invalid challenge UUID format - challenge_id is REQUIRED")
		return asynq.SkipRetry
	}

	jobLogger.Info().
		Str("sub_id", payload.SubmissionID).
		Str("company_id", payload.CompanyID).
		Str("challenge_id", payload.ChallengeID).
		Msg("Analysis job started")

	// 3. Run strategic cascade analysis
	// Note: Company data and submission data are fetched internally by analysisService
	_, err = w.analysisService.RunAnalysis(ctx, payload.SubmissionID, payload.CompanyID, challengeUUID)
	if err != nil {
		jobLogger.Error().
			Err(err).
			Str("sub_id", payload.SubmissionID).
			Dur("duration", time.Since(startTime)).
			Msg("Analysis job failed")

		// Check if error is retryable
		if isRetryableError(err) {
			return fmt.Errorf("retryable error: %w", err)
		}

		// Permanent failure - skip retry
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	// 5. Success
	jobLogger.Info().
		Str("sub_id", payload.SubmissionID).
		Dur("duration", time.Since(startTime)).
		Int64("duration_ms", time.Since(startTime).Milliseconds()).
		Msg("Analysis job completed successfully")

	return nil
}

// --- Error Handling ---

// handleJobError is called by Asynq ErrorHandler
// Only updates database to "failed" when max retries exhausted
func (w *Worker) handleJobError(ctx context.Context, task *asynq.Task, err error) {
	retried, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)

	w.logger.Error().
		Err(err).
		Str("task_type", task.Type()).
		Str("task_payload", string(task.Payload())).
		Int("retry_count", retried).
		Int("max_retry", maxRetry).
		Bool("is_final_failure", retried >= maxRetry).
		Msg("Job execution failed")

	// Only on FINAL failure - update database and archive to DLQ
	if retried >= maxRetry {
		w.markJobAsFailed(ctx, task, err)
		w.moveToDLQ(ctx, task, err)
	}
}

// markJobAsFailed updates database status to "failed"
func (w *Worker) markJobAsFailed(ctx context.Context, task *asynq.Task, err error) {
	errorMsg := err.Error()

	switch task.Type() {
	case jobtypes.TypeAnalysis:
		var payload AnalysisJobPayload
		if unmarshalErr := json.Unmarshal(task.Payload(), &payload); unmarshalErr != nil {
			w.logger.Error().Err(unmarshalErr).Msg("Failed to unmarshal payload for marking failed")
			return
		}

		if err := w.analysisService.MarkAsFailed(ctx, payload.SubmissionID, errorMsg); err != nil {
			w.logger.Error().
				Err(err).
				Str("sub_id", payload.SubmissionID).
				Msg("Failed to mark analysis as failed in database")
		} else {
			w.logger.Info().
				Str("sub_id", payload.SubmissionID).
				Msg("Analysis marked as failed in database")
		}
	}
}

// moveToDLQ archives permanently failed jobs to Dead Letter Queue
func (w *Worker) moveToDLQ(ctx context.Context, task *asynq.Task, err error) {
	retried, _ := asynq.GetRetryCount(ctx)

	dlqEntry := map[string]interface{}{
		"task_type":   task.Type(),
		"payload":     string(task.Payload()),
		"error":       err.Error(),
		"failed_at":   time.Now().Format(time.RFC3339),
		"retry_count": retried,
		"task_id":     task.ResultWriter().TaskID(),
	}

	data, marshalErr := json.Marshal(dlqEntry)
	if marshalErr != nil {
		w.logger.Error().Err(marshalErr).Msg("Failed to marshal DLQ entry")
		return
	}

	key := fmt.Sprintf("dlq:%s:%d", task.Type(), time.Now().Unix())
	ttl := time.Duration(w.cfg.DeadLetterQueueTTL) * time.Hour

	if err := w.redisClient.Set(ctx, key, data, ttl).Err(); err != nil {
		w.logger.Error().
			Err(err).
			Str("dlq_key", key).
			Msg("Failed to store job in Dead Letter Queue")
	} else {
		w.logger.Warn().
			Str("dlq_key", key).
			Str("task_type", task.Type()).
			Dur("ttl", ttl).
			Msg("Job moved to Dead Letter Queue")
	}
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Network/timeout errors - retryable
	if strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "temporary failure") {
		return true
	}

	// Rate limit errors - retryable
	if strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "too many requests") {
		return true
	}

	// Server errors - retryable
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") ||
		strings.Contains(errStr, "internal server error") {
		return true
	}

	// Database connection errors - retryable
	if strings.Contains(errStr, "database") && strings.Contains(errStr, "connection") {
		return true
	}

	// Validation errors - NOT retryable
	if strings.Contains(errStr, "validation failed") ||
		strings.Contains(errStr, "invalid") ||
		strings.Contains(errStr, "400") {
		return false
	}

	// Default: not retryable (be conservative)
	return false
}

// --- Job Enqueueing ---

func (w *Worker) enqueueJob(typeName string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Generate task ID for deduplication
	var taskID string
	switch p := payload.(type) {
	case AnalysisJobPayload:
		taskID = fmt.Sprintf("analysis:%s", p.SubmissionID)
	default:
		taskID = fmt.Sprintf("%s:%d", typeName, time.Now().UnixNano())
	}

	// Configure task options based on type
	var opts []asynq.Option
	opts = append(opts, asynq.TaskID(taskID))
	opts = append(opts, asynq.Retention(24*time.Hour))

	if typeName == jobtypes.TypeAnalysis {
		opts = append(opts, asynq.MaxRetry(w.cfg.AnalysisMaxRetries))
		opts = append(opts, asynq.Timeout(time.Duration(w.cfg.AnalysisTimeout)*time.Second))
	}

	// Enqueue task
	_, err = w.asynqClient.Enqueue(asynq.NewTask(typeName, data), opts...)
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	w.logger.Info().
		Str("task_type", typeName).
		Str("task_id", taskID).
		Msg("Job enqueued successfully")

	return nil
}

// Task Creation Helpers

// NewAnalysisTask creates a new analysis task
func NewAnalysisTask(payload AnalysisJobPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(jobtypes.TypeAnalysis, data), nil
}
