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
	"backend_v3/domain/framework"
	"backend_v3/domain/submission"
	jobtypes "backend_v3/jobs/types"
	"backend_v3/llm"

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

// FrameworkExecutionPayload represents async framework execution (V2)
type FrameworkExecutionPayload struct {
	CompanyID     string  `json:"company_id"`
	FrameworkCode string  `json:"framework_code"`
	ChallengeID   *string `json:"challenge_id,omitempty"`
	ResultID      string  `json:"result_id"` // ID of company_framework_results to update
}

type Worker struct {
	server            *asynq.Server
	mux               *asynq.ServeMux
	submissionService *submission.Service
	enrichmentService *enrichment.Service // For reading enrichment data in analysis jobs
	analysisService   *analysis.Service
	frameworkService  *framework.Service // V2: Database-driven frameworks (optional)
	llmClient         *llm.Client        // V2: For framework execution
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

	// Framework execution handler (V2) - always register, will check service availability at runtime
	w.logger.Info().
		Str("task_type", jobtypes.TypeFrameworkExecution).
		Msg("📝 Registering handler for framework execution")
	w.mux.HandleFunc(jobtypes.TypeFrameworkExecution, w.HandleFrameworkExecutionJob)

	return w
}

// SetFrameworkService sets the framework service for V2 framework execution
// Call this after NewWorker if FRAMEWORKS_V2_ENABLED=true
func (w *Worker) SetFrameworkService(svc *framework.Service) {
	w.frameworkService = svc
	w.logger.Info().Msg("Framework service injected into worker")
}

// SetLLMClient sets the LLM client for V2 framework execution
func (w *Worker) SetLLMClient(client *llm.Client) {
	w.llmClient = client
	w.logger.Info().Msg("LLM client injected into worker")
}

func (w *Worker) Start() error {
	w.logger.Info().
		Str("redis_addr", w.redisOpt.Addr).
		Bool("has_password", w.redisOpt.Password != "").
		Msg("🔌 Worker connecting to Redis for job processing")

	// Test Redis connection before starting worker
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := w.redisClient.Ping(ctx).Err(); err != nil {
		w.logger.Error().Err(err).
			Str("redis_addr", w.redisOpt.Addr).
			Msg("❌ Worker cannot connect to Redis - aborting start")
		return fmt.Errorf("redis connection failed: %w", err)
	}
	w.logger.Info().Msg("✅ Worker Redis connection verified")

	// Log registered handlers
	w.logger.Info().
		Str("handler_1", "analysis_job").
		Str("handler_2", "framework_execution").
		Msg("📋 Registered job handlers")

	err := w.server.Start(w.mux)
	if err != nil {
		w.logger.Error().Err(err).Msg("❌ Asynq server failed to start")
		return err
	}

	w.logger.Info().Msg("✅ Asynq server started - now listening for jobs")
	return nil
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

// HandleFrameworkExecutionJob handles async execution of a single framework (V2)
func (w *Worker) HandleFrameworkExecutionJob(ctx context.Context, task *asynq.Task) error {
	// FIRST LINE - absolute minimum logging to confirm handler is called
	w.logger.Info().Msg("🔥🔥🔥 HANDLER ENTRY - framework_execution job received by worker")

	startTime := time.Now()

	// Check if framework service is available
	if w.frameworkService == nil {
		w.logger.Error().Msg("Framework execution job received but frameworkService is nil - FRAMEWORKS_V2_ENABLED=false?")
		return fmt.Errorf("%w: framework service not available", asynq.SkipRetry)
	}

	// Get task metadata
	var taskID string
	if rw := task.ResultWriter(); rw != nil {
		taskID = rw.TaskID()
	} else {
		taskID = "unknown"
	}
	retryCount, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)

	// Parse payload
	var payload FrameworkExecutionPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		w.logger.Error().Err(err).Msg("Failed to unmarshal framework execution payload")
		return fmt.Errorf("%w: invalid payload format: %v", asynq.SkipRetry, err)
	}

	jobLogger := w.logger.With().
		Str("job_type", jobtypes.TypeFrameworkExecution).
		Str("task_id", taskID).
		Str("company_id", payload.CompanyID).
		Str("framework_code", payload.FrameworkCode).
		Str("result_id", payload.ResultID).
		Int("retry_count", retryCount).
		Int("max_retries", maxRetry).
		Logger()

	jobLogger.Info().Msg("🔧 Framework execution job started")

	// Parse UUIDs
	companyID, err := uuid.Parse(payload.CompanyID)
	if err != nil {
		jobLogger.Error().Err(err).Msg("Invalid company UUID")
		return fmt.Errorf("%w: invalid company_id", asynq.SkipRetry)
	}

	resultID, err := uuid.Parse(payload.ResultID)
	if err != nil {
		jobLogger.Error().Err(err).Msg("Invalid result UUID")
		return fmt.Errorf("%w: invalid result_id", asynq.SkipRetry)
	}

	var challengeID *uuid.UUID
	if payload.ChallengeID != nil {
		if id, err := uuid.Parse(*payload.ChallengeID); err == nil {
			challengeID = &id
		}
	}

	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, time.Duration(w.cfg.AnalysisTimeout)*time.Second)
	defer cancel()

	// Mark as processing
	if err := w.frameworkService.MarkProcessing(ctx, resultID); err != nil {
		jobLogger.Warn().Err(err).Msg("Failed to mark result as processing")
	}

	// Get the framework
	fw, err := w.frameworkService.GetByCode(ctx, payload.FrameworkCode)
	if err != nil || fw == nil {
		jobLogger.Error().Err(err).Msg("Framework not found")
		_ = w.frameworkService.MarkFailed(ctx, resultID, "Framework not found: "+payload.FrameworkCode)
		return fmt.Errorf("%w: framework not found", asynq.SkipRetry)
	}

	// Get dependency context (results from frameworks this one depends on)
	depContext, err := w.frameworkService.GetDependencyContext(ctx, fw.ID, companyID, challengeID)
	if err != nil {
		jobLogger.Warn().Err(err).Msg("Failed to get dependency context, continuing without")
		depContext = make(map[string]json.RawMessage)
	}

	jobLogger.Info().
		Int("dependency_count", len(depContext)).
		Str("model", fw.ModelConfig.Model).
		Msg("Executing framework with LLM")

	// Execute framework with LLM
	result, err := w.executeFrameworkWithLLM(ctx, fw, depContext, jobLogger)
	if err != nil {
		jobLogger.Error().Err(err).Msg("Framework LLM execution failed")
		_ = w.frameworkService.MarkFailed(ctx, resultID, err.Error())

		// Check if retryable
		if isRetryableError(err) {
			return fmt.Errorf("retryable LLM error: %w", err)
		}
		return fmt.Errorf("%w: LLM execution failed: %v", asynq.SkipRetry, err)
	}

	// Save result
	if err := w.frameworkService.MarkCompleted(ctx, resultID, result); err != nil {
		jobLogger.Error().Err(err).Msg("Failed to mark framework result as completed")
		return fmt.Errorf("failed to save result: %w", err)
	}

	jobLogger.Info().
		Dur("duration", time.Since(startTime)).
		Int64("duration_ms", time.Since(startTime).Milliseconds()).
		Msg("✅ Framework execution job completed")

	return nil
}

// executeFrameworkWithLLM calls the LLM with the framework's prompts
func (w *Worker) executeFrameworkWithLLM(ctx context.Context, fw *framework.Framework, depContext map[string]json.RawMessage, logger zerolog.Logger) (json.RawMessage, error) {
	// Check if LLM client is available
	if w.llmClient == nil {
		logger.Warn().Msg("LLM client not available, returning mock result")
		return json.RawMessage(`{"status": "mock", "note": "LLM client not configured"}`), nil
	}

	// Get prompts - use database values with fallback to code prompts if placeholder
	systemPrompt, userPrompt := w.getFrameworkPrompts(fw, logger)

	// Build the prompt with dependency context
	prompt := userPrompt
	if len(depContext) > 0 {
		contextJSON, _ := json.MarshalIndent(depContext, "", "  ")
		prompt = fmt.Sprintf("%s\n\n## Previous Framework Results:\n%s", prompt, string(contextJSON))
	}

	// If we have a system prompt, prepend it with data priority instruction
	if systemPrompt != "" {
		prompt = fmt.Sprintf("=== SYSTEM CONTEXT ===\n%s\n\n%s\n=== END SYSTEM CONTEXT ===\n\n%s",
			systemPrompt, llm.DataPriorityInstruction, prompt)
	}

	// Build generation options from framework config
	opts := llm.GenerationOptions{
		Model:         fw.ModelConfig.Model,
		Temperature:   fw.ModelConfig.Temperature,
		MaxTokens:     fw.ModelConfig.MaxTokens,
		FallbackModel: fw.ModelConfig.FallbackModel,
	}

	// Use default model if not configured
	if opts.Model == "" {
		opts.Model = "google/gemini-2.0-flash-001"
		opts.FallbackModel = "openai/gpt-4o-mini"
	}
	if opts.MaxTokens == 0 {
		opts.MaxTokens = 8000
	}
	if opts.Temperature == 0 {
		opts.Temperature = 0.5
	}

	logger.Info().
		Str("model", opts.Model).
		Str("fallback", opts.FallbackModel).
		Int("max_tokens", opts.MaxTokens).
		Msg("Calling LLM")

	// Call LLM - we expect JSON output
	var result map[string]interface{}
	err := w.llmClient.GenerateStructuredWithOptions(ctx, opts, prompt, nil, &result)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Marshal result back to JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LLM result: %w", err)
	}

	logger.Info().
		Int("result_size", len(resultJSON)).
		Msg("LLM execution successful")

	return resultJSON, nil
}

// getFrameworkPrompts returns the system and user prompts for a framework.
// It uses database values if populated, falling back to code prompts if the DB has placeholder text.
func (w *Worker) getFrameworkPrompts(fw *framework.Framework, logger zerolog.Logger) (systemPrompt, userPrompt string) {
	// Check if database prompt is a placeholder (contains "[Full prompt" marker)
	isPlaceholder := strings.Contains(fw.PromptUser, "[Full prompt in llm/prompts.go")

	if isPlaceholder || fw.PromptUser == "" {
		// Use fallback prompts from code
		userPrompt = w.getCodePromptForFramework(fw.Code)
		logger.Info().
			Str("framework", fw.Code).
			Bool("was_placeholder", isPlaceholder).
			Msg("Using fallback code prompt for framework")
	} else {
		// Use database prompt
		userPrompt = fw.PromptUser
		logger.Debug().
			Str("framework", fw.Code).
			Msg("Using database prompt for framework")
	}

	// System prompt from database (migrations will populate these)
	if fw.PromptSystem != nil {
		systemPrompt = *fw.PromptSystem
	}

	return systemPrompt, userPrompt
}

// getCodePromptForFramework returns the hardcoded prompt for a framework code.
// This is a fallback when the database prompt is empty or has placeholder text.
func (w *Worker) getCodePromptForFramework(code string) string {
	// Map framework codes to their prompts in llm/prompts.go
	switch code {
	case "pestel":
		return llm.FrameworkPESTELPrompt
	case "porter":
		return llm.FrameworkPorterPrompt
	case "tam_sam_som":
		return llm.FrameworkTamSamSomPrompt
	case "swot":
		return llm.FrameworkSWOTPrompt
	case "benchmarking":
		return llm.FrameworkBenchmarkingPrompt
	case "blue_ocean":
		return llm.FrameworkBlueOceanPrompt
	case "growth_hacking":
		return llm.FrameworkGrowthHackingPrompt
	case "scenarios":
		return llm.FrameworkScenariosPrompt
	case "decision_matrix":
		return llm.FrameworkDecisionMatrixPrompt
	case "okrs":
		return llm.FrameworkOKRsPrompt
	case "bsc":
		return llm.FrameworkBSCPrompt
	case "synthesis":
		return llm.SynthesisPrompt
	case "challenge_refinement":
		return llm.ChallengeRefinementPrompt
	default:
		// Return a generic prompt if framework code unknown
		return fmt.Sprintf("Analyze the company data and provide insights for %s framework. Return JSON.", code)
	}
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

	case jobtypes.TypeFrameworkExecution:
		if w.frameworkService == nil {
			w.logger.Warn().Msg("Cannot mark framework as failed - service not available")
			return
		}

		var payload FrameworkExecutionPayload
		if unmarshalErr := json.Unmarshal(task.Payload(), &payload); unmarshalErr != nil {
			w.logger.Error().Err(unmarshalErr).Msg("Failed to unmarshal framework payload for marking failed")
			return
		}

		resultID, parseErr := uuid.Parse(payload.ResultID)
		if parseErr != nil {
			w.logger.Error().Err(parseErr).Msg("Invalid result_id in framework payload")
			return
		}

		if err := w.frameworkService.MarkFailed(ctx, resultID, errorMsg); err != nil {
			w.logger.Error().
				Err(err).
				Str("result_id", payload.ResultID).
				Msg("Failed to mark framework result as failed in database")
		} else {
			w.logger.Info().
				Str("result_id", payload.ResultID).
				Msg("Framework result marked as failed in database")
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
