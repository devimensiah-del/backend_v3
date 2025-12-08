package logging

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog.Logger with contextual field methods
type Logger struct {
	zerolog.Logger
}

// NewLogger creates a new Logger instance with environment-specific configuration
// env should be "development" or "production"
func NewLogger(env string) *Logger {
	// Configure zerolog output
	zerolog.TimeFieldFormat = time.RFC3339

	var zlog zerolog.Logger

	if env == "development" {
		// Development: human-readable console output with colors
		zlog = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
			Level(zerolog.DebugLevel).
			With().
			Timestamp().
			Logger()
	} else {
		// Production: JSON output for structured logging
		zlog = zerolog.New(os.Stderr).
			Level(zerolog.InfoLevel).
			With().
			Timestamp().
			Logger()
	}

	return &Logger{Logger: zlog}
}

// WithRequestID returns a new logger with request_id field
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{Logger: l.Logger.With().Str("request_id", requestID).Logger()}
}

// WithSubmissionID returns a new logger with submission_id field
func (l *Logger) WithSubmissionID(submissionID string) *Logger {
	return &Logger{Logger: l.Logger.With().Str("submission_id", submissionID).Logger()}
}

// WithUserID returns a new logger with user_id field
func (l *Logger) WithUserID(userID string) *Logger {
	return &Logger{Logger: l.Logger.With().Str("user_id", userID).Logger()}
}

// WithComponent returns a new logger with component field
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{Logger: l.Logger.With().Str("component", component).Logger()}
}

// WithCompanyID returns a new logger with company_id field
func (l *Logger) WithCompanyID(companyID string) *Logger {
	return &Logger{Logger: l.Logger.With().Str("company_id", companyID).Logger()}
}

// WithAnalysisID returns a new logger with analysis_id field
func (l *Logger) WithAnalysisID(analysisID string) *Logger {
	return &Logger{Logger: l.Logger.With().Str("analysis_id", analysisID).Logger()}
}

// WithJobID returns a new logger with job_id field
func (l *Logger) WithJobID(jobID string) *Logger {
	return &Logger{Logger: l.Logger.With().Str("job_id", jobID).Logger()}
}

// WithDuration returns a new logger with duration_ms field
func (l *Logger) WithDuration(durationMs int64) *Logger {
	return &Logger{Logger: l.Logger.With().Int64("duration_ms", durationMs).Logger()}
}

// WithError returns a new logger with error field
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return &Logger{Logger: l.Logger.With().Err(err).Logger()}
}
