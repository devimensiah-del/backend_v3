package jobs

import (
	"context"
	"encoding/json"

	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/report"
	"backend_v3/domain/submission"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

const (
	TypeEnrichment = "enrichment_job"
	TypeAnalysis   = "analysis_job"
	// TypeReport is REMOVED
)

// Job Payload Types
type EnrichmentJobPayload struct {
	SubmissionID string `json:"submission_id"`
}

type AnalysisJobPayload struct {
	SubmissionID string `json:"submission_id"`
	EnrichmentID string `json:"enrichment_id"`
}

type Worker struct {
	server            *asynq.Server
	mux               *asynq.ServeMux
	submissionService *submission.Service
	enrichmentService *enrichment.Service
	analysisService   *analysis.Service
	reportService     *report.Service // Kept just in case, but unused in background jobs
	logger            zerolog.Logger
	redisOpt          asynq.RedisClientOpt // Store full Redis options including password
}

func NewWorker(
	redisAddr string,
	redisPassword string,
	submissionSvc *submission.Service,
	enrichmentSvc *enrichment.Service,
	analysisSvc *analysis.Service,
	reportSvc *report.Service,
	logger zerolog.Logger,
) *Worker {
	redisOpt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
	}

	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 10,
			Queues:      map[string]int{"critical": 6, "default": 3, "low": 1},
		},
	)

	mux := asynq.NewServeMux()
	w := &Worker{
		server:            srv,
		mux:               mux,
		submissionService: submissionSvc,
		enrichmentService: enrichmentSvc,
		analysisService:   analysisSvc,
		reportService:     reportSvc,
		logger:            logger.With().Str("component", "worker").Logger(),
		redisOpt:          redisOpt,
	}

	mux.HandleFunc(TypeEnrichment, w.HandleEnrichmentJob)
	mux.HandleFunc(TypeAnalysis, w.HandleAnalysisJob)
	// Removed HandleReportJob

	return w
}

func (w *Worker) Start() error { return w.server.Start(w.mux) }
func (w *Worker) Stop()        { w.server.Shutdown() }

// --- Handlers ---

func (w *Worker) HandleEnrichmentJob(ctx context.Context, task *asynq.Task) error {
	var payload struct {
		SubmissionID string `json:"submission_id"`
	}
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}

	w.logger.Info().Str("sub_id", payload.SubmissionID).Msg("Job Started: Enrichment Agent")

	// 1. Run Agent
	enrichmentData, err := w.enrichmentService.EnrichSubmission(ctx, uuid.MustParse(payload.SubmissionID))
	if err != nil {
		return err
	}

	// 2. Chain Analysis Job
	nextPayload := map[string]string{
		"submission_id": payload.SubmissionID,
		"enrichment_id": enrichmentData.ID.String(),
	}
	return w.enqueueJob(TypeAnalysis, nextPayload)
}

func (w *Worker) HandleAnalysisJob(ctx context.Context, task *asynq.Task) error {
	var payload struct {
		SubmissionID string `json:"submission_id"`
		EnrichmentID string `json:"enrichment_id"`
	}
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}

	w.logger.Info().Str("sub_id", payload.SubmissionID).Msg("Job Started: Strategic Cascade")

	// 1. Get Data
	enrichmentRecord, err := w.enrichmentService.GetByID(ctx, uuid.MustParse(payload.EnrichmentID))
	if err != nil {
		return err
	}

	// 2. Run Cascade
	_, err = w.analysisService.RunAnalysis(ctx, payload.SubmissionID, payload.EnrichmentID, enrichmentRecord.EnrichedData)
	if err != nil {
		w.submissionService.UpdateStatus(ctx, uuid.MustParse(payload.SubmissionID), "analysis_failed")
		return err
	}

	// 3. STOP. Status is now "completed" (or "ready_for_review").
	// We do NOT queue a report job. The admin must trigger that manually.
	w.submissionService.UpdateStatus(ctx, uuid.MustParse(payload.SubmissionID), "ready_for_review")
	w.logger.Info().Msg("Workflow Paused. Waiting for Admin Review.")
	return nil
}

func (w *Worker) enqueueJob(typeName string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Use w.redisOpt with both address and password
	client := asynq.NewClient(w.redisOpt)
	defer client.Close()

	_, err = client.Enqueue(asynq.NewTask(typeName, data))
	return err
}

// Task Creation Helpers

// NewEnrichmentTask creates a new enrichment task
func NewEnrichmentTask(payload EnrichmentJobPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeEnrichment, data), nil
}

// NewAnalysisTask creates a new analysis task
func NewAnalysisTask(payload AnalysisJobPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeAnalysis, data), nil
}
