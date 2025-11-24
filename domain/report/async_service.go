package report

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

const (
	// Task type for PDF generation
	TaskTypeGeneratePDF = "report:generate_pdf"

	// Queue name for report processing
	QueueReports = "reports"
)

// PDFGenerationPayload contains the data needed to generate a PDF asynchronously
type PDFGenerationPayload struct {
	SubmissionID string `json:"submission_id"`
	AnalysisID   string `json:"analysis_id"`
	RequestedBy  string `json:"requested_by,omitempty"` // User ID who requested the PDF
}

// AsyncService handles asynchronous PDF generation tasks
type AsyncService struct {
	client *asynq.Client
	logger zerolog.Logger
}

// NewAsyncService creates a new async service for background tasks
func NewAsyncService(redisAddr string, logger zerolog.Logger) (*AsyncService, error) {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})

	return &AsyncService{
		client: client,
		logger: logger.With().Str("service", "report_async").Logger(),
	}, nil
}

// EnqueuePDFGeneration queues a PDF generation task
// Returns the task ID for tracking
func (s *AsyncService) EnqueuePDFGeneration(ctx context.Context, submissionID, analysisID string) (string, error) {
	payload := PDFGenerationPayload{
		SubmissionID: submissionID,
		AnalysisID:   analysisID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TaskTypeGeneratePDF, payloadBytes,
		asynq.Queue(QueueReports),
		asynq.MaxRetry(3),
		asynq.Timeout(5*time.Minute), // 5 minutes max for 24 pages
		asynq.Retention(24*time.Hour), // Keep task info for 24 hours
	)

	info, err := s.client.EnqueueContext(ctx, task)
	if err != nil {
		return "", fmt.Errorf("failed to enqueue task: %w", err)
	}

	s.logger.Info().
		Str("task_id", info.ID).
		Str("submission_id", submissionID).
		Str("analysis_id", analysisID).
		Msg("PDF generation task enqueued")

	return info.ID, nil
}

// Close closes the async client connection
func (s *AsyncService) Close() error {
	return s.client.Close()
}

// PDFGenerationWorker processes PDF generation tasks
type PDFGenerationWorker struct {
	reportService *Service
	logger        zerolog.Logger
}

// NewPDFGenerationWorker creates a new worker for processing PDF tasks
func NewPDFGenerationWorker(reportService *Service, logger zerolog.Logger) *PDFGenerationWorker {
	return &PDFGenerationWorker{
		reportService: reportService,
		logger:        logger.With().Str("worker", "pdf_generation").Logger(),
	}
}

// ProcessTask handles the PDF generation task
func (w *PDFGenerationWorker) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload PDFGenerationPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	w.logger.Info().
		Str("submission_id", payload.SubmissionID).
		Str("analysis_id", payload.AnalysisID).
		Msg("Starting PDF generation")

	start := time.Now()

	// Call the actual PDF generation logic
	pdfURL, err := w.reportService.Publish(ctx, payload.SubmissionID, payload.AnalysisID)
	if err != nil {
		w.logger.Error().
			Err(err).
			Str("submission_id", payload.SubmissionID).
			Str("analysis_id", payload.AnalysisID).
			Msg("PDF generation failed")
		return fmt.Errorf("PDF generation failed: %w", err)
	}

	duration := time.Since(start)

	w.logger.Info().
		Str("submission_id", payload.SubmissionID).
		Str("analysis_id", payload.AnalysisID).
		Str("pdf_url", pdfURL).
		Dur("duration", duration).
		Msg("PDF generation completed successfully")

	return nil
}

// StartWorker starts the Asynq worker server
func StartWorker(redisAddr string, reportService *Service, logger zerolog.Logger) (*asynq.Server, error) {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 3, // Process 3 PDFs concurrently
			Queues: map[string]int{
				QueueReports: 10, // High priority for report queue
			},
			Logger: &asynqLogger{logger: logger},
		},
	)

	mux := asynq.NewServeMux()
	worker := NewPDFGenerationWorker(reportService, logger)
	mux.HandleFunc(TaskTypeGeneratePDF, worker.ProcessTask)

	// Start the server in a goroutine
	go func() {
		if err := srv.Run(mux); err != nil {
			logger.Fatal().Err(err).Msg("Asynq worker server failed to start")
		}
	}()

	logger.Info().Msg("Asynq worker server started successfully")

	return srv, nil
}

// asynqLogger adapts zerolog to asynq.Logger interface
type asynqLogger struct {
	logger zerolog.Logger
}

func (l *asynqLogger) Debug(args ...interface{}) {
	l.logger.Debug().Msgf("%v", args)
}

func (l *asynqLogger) Info(args ...interface{}) {
	l.logger.Info().Msgf("%v", args)
}

func (l *asynqLogger) Warn(args ...interface{}) {
	l.logger.Warn().Msgf("%v", args)
}

func (l *asynqLogger) Error(args ...interface{}) {
	l.logger.Error().Msgf("%v", args)
}

func (l *asynqLogger) Fatal(args ...interface{}) {
	l.logger.Fatal().Msgf("%v", args)
}
