package macroeconomics

import (
	"fmt"
	"time"

	jobtypes "backend_v3/jobs/types"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

// Scheduler manages periodic macro data fetch jobs using Asynq Scheduler
// Configured with Brazil timezone (America/Sao_Paulo) for proper cron timing
type Scheduler struct {
	scheduler *asynq.Scheduler
	service   *Service
	entryIDs  []string // Track registered task IDs for cleanup
}

// NewScheduler creates a new scheduler with BRT timezone
func NewScheduler(redisOpt asynq.RedisClientOpt, service *Service) (*Scheduler, error) {
	// Load Brazil timezone
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		// Fallback: try with UTC-3 offset if timezone data unavailable
		log.Warn().Err(err).Msg("Failed to load America/Sao_Paulo timezone, using UTC-3 offset")
		loc = time.FixedZone("BRT", -3*60*60)
	}

	scheduler := asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{
		Location: loc,
		Logger:   &asynqLogger{},
	})

	return &Scheduler{
		scheduler: scheduler,
		service:   service,
		entryIDs:  make([]string, 0),
	}, nil
}

// RegisterTasks registers all periodic macro data fetch tasks
func (s *Scheduler) RegisterTasks() error {
	log.Info().Msg("Registering macro data scheduler tasks (BRT timezone)")

	// SELIC: Weekdays at 6pm BRT (after BCB end-of-day update)
	// BCB typically updates SELIC data by 18:00 BRT
	selicID, err := s.scheduler.Register(
		"0 18 * * 1-5",
		asynq.NewTask(jobtypes.TypeMacroFetch, []byte(`{"code":"selic"}`)),
		asynq.Queue("default"),
		asynq.MaxRetry(3),
		asynq.Timeout(2*time.Minute),
	)
	if err != nil {
		return fmt.Errorf("failed to register SELIC task: %w", err)
	}
	s.entryIDs = append(s.entryIDs, selicID)
	log.Info().Str("id", selicID).Str("schedule", "0 18 * * 1-5 (6pm Mon-Fri BRT)").Msg("Registered SELIC fetch task")

	// IPCA: Monthly on 15th at 10am BRT (IBGE publishes ~10th-15th)
	// Using 15th to ensure data is available
	ipcaID, err := s.scheduler.Register(
		"0 10 15 * *",
		asynq.NewTask(jobtypes.TypeMacroFetch, []byte(`{"code":"ipca"}`)),
		asynq.Queue("default"),
		asynq.MaxRetry(3),
		asynq.Timeout(2*time.Minute),
	)
	if err != nil {
		return fmt.Errorf("failed to register IPCA task: %w", err)
	}
	s.entryIDs = append(s.entryIDs, ipcaID)
	log.Info().Str("id", ipcaID).Str("schedule", "0 10 15 * * (10am 15th BRT)").Msg("Registered IPCA fetch task")

	// USD/BRL: Every 6 hours (exchange rate volatility)
	// 00:00, 06:00, 12:00, 18:00 BRT
	usdBrlID, err := s.scheduler.Register(
		"0 */6 * * *",
		asynq.NewTask(jobtypes.TypeMacroFetch, []byte(`{"code":"usd_brl"}`)),
		asynq.Queue("default"),
		asynq.MaxRetry(3),
		asynq.Timeout(2*time.Minute),
	)
	if err != nil {
		return fmt.Errorf("failed to register USD/BRL task: %w", err)
	}
	s.entryIDs = append(s.entryIDs, usdBrlID)
	log.Info().Str("id", usdBrlID).Str("schedule", "0 */6 * * * (every 6h BRT)").Msg("Registered USD/BRL fetch task")

	// Full refresh: Daily at midnight BRT (catch-all backup)
	// Ensures all indicators are up to date even if individual fetches fail
	refreshAllID, err := s.scheduler.Register(
		"0 0 * * *",
		asynq.NewTask(jobtypes.TypeMacroRefreshAll, nil),
		asynq.Queue("low"), // Lower priority queue
		asynq.MaxRetry(3),
		asynq.Timeout(5*time.Minute),
	)
	if err != nil {
		return fmt.Errorf("failed to register refresh-all task: %w", err)
	}
	s.entryIDs = append(s.entryIDs, refreshAllID)
	log.Info().Str("id", refreshAllID).Str("schedule", "0 0 * * * (midnight BRT)").Msg("Registered full refresh task")

	log.Info().Int("total_tasks", len(s.entryIDs)).Msg("All macro scheduler tasks registered")

	return nil
}

// Start begins the scheduler
func (s *Scheduler) Start() error {
	log.Info().Msg("Starting macro data scheduler")
	return s.scheduler.Start()
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() {
	log.Info().Msg("Stopping macro data scheduler")
	s.scheduler.Shutdown()
}

// GetRegisteredTasks returns the list of registered task entry IDs
func (s *Scheduler) GetRegisteredTasks() []string {
	return s.entryIDs
}

// asynqLogger adapts zerolog to asynq's Logger interface
type asynqLogger struct{}

func (l *asynqLogger) Debug(args ...interface{}) {
	log.Debug().Msgf("%v", args)
}

func (l *asynqLogger) Info(args ...interface{}) {
	log.Info().Msgf("%v", args)
}

func (l *asynqLogger) Warn(args ...interface{}) {
	log.Warn().Msgf("%v", args)
}

func (l *asynqLogger) Error(args ...interface{}) {
	log.Error().Msgf("%v", args)
}

func (l *asynqLogger) Fatal(args ...interface{}) {
	log.Fatal().Msgf("%v", args)
}
