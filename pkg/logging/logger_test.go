package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/rs/zerolog"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{"development logger", "development"},
		{"production logger", "production"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.env)
			if logger == nil {
				t.Error("NewLogger returned nil")
			}
		})
	}
}

func TestLoggerContextualFields(t *testing.T) {
	// Use a buffer to capture JSON output
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{Logger: zlog}

	// Test WithRequestID
	t.Run("WithRequestID", func(t *testing.T) {
		buf.Reset()
		reqLogger := logger.WithRequestID("req-123")
		reqLogger.Info().Msg("test message")

		var logEntry map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
			t.Fatalf("Failed to parse log output: %v", err)
		}

		if logEntry["request_id"] != "req-123" {
			t.Errorf("Expected request_id=req-123, got %v", logEntry["request_id"])
		}
	})

	// Test WithSubmissionID
	t.Run("WithSubmissionID", func(t *testing.T) {
		buf.Reset()
		subLogger := logger.WithSubmissionID("sub-456")
		subLogger.Info().Msg("test message")

		var logEntry map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
			t.Fatalf("Failed to parse log output: %v", err)
		}

		if logEntry["submission_id"] != "sub-456" {
			t.Errorf("Expected submission_id=sub-456, got %v", logEntry["submission_id"])
		}
	})

	// Test WithUserID
	t.Run("WithUserID", func(t *testing.T) {
		buf.Reset()
		userLogger := logger.WithUserID("user-789")
		userLogger.Info().Msg("test message")

		var logEntry map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
			t.Fatalf("Failed to parse log output: %v", err)
		}

		if logEntry["user_id"] != "user-789" {
			t.Errorf("Expected user_id=user-789, got %v", logEntry["user_id"])
		}
	})

	// Test WithComponent
	t.Run("WithComponent", func(t *testing.T) {
		buf.Reset()
		compLogger := logger.WithComponent("api")
		compLogger.Info().Msg("test message")

		var logEntry map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
			t.Fatalf("Failed to parse log output: %v", err)
		}

		if logEntry["component"] != "api" {
			t.Errorf("Expected component=api, got %v", logEntry["component"])
		}
	})

	// Test WithEnrichmentID
	t.Run("WithEnrichmentID", func(t *testing.T) {
		buf.Reset()
		enrichLogger := logger.WithEnrichmentID("enrich-101")
		enrichLogger.Info().Msg("test message")

		var logEntry map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
			t.Fatalf("Failed to parse log output: %v", err)
		}

		if logEntry["enrichment_id"] != "enrich-101" {
			t.Errorf("Expected enrichment_id=enrich-101, got %v", logEntry["enrichment_id"])
		}
	})

	// Test WithAnalysisID
	t.Run("WithAnalysisID", func(t *testing.T) {
		buf.Reset()
		analysisLogger := logger.WithAnalysisID("analysis-202")
		analysisLogger.Info().Msg("test message")

		var logEntry map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
			t.Fatalf("Failed to parse log output: %v", err)
		}

		if logEntry["analysis_id"] != "analysis-202" {
			t.Errorf("Expected analysis_id=analysis-202, got %v", logEntry["analysis_id"])
		}
	})

	// Test WithJobID
	t.Run("WithJobID", func(t *testing.T) {
		buf.Reset()
		jobLogger := logger.WithJobID("job-303")
		jobLogger.Info().Msg("test message")

		var logEntry map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
			t.Fatalf("Failed to parse log output: %v", err)
		}

		if logEntry["job_id"] != "job-303" {
			t.Errorf("Expected job_id=job-303, got %v", logEntry["job_id"])
		}
	})

	// Test WithDuration
	t.Run("WithDuration", func(t *testing.T) {
		buf.Reset()
		durLogger := logger.WithDuration(1234)
		durLogger.Info().Msg("test message")

		var logEntry map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
			t.Fatalf("Failed to parse log output: %v", err)
		}

		if logEntry["duration_ms"] != float64(1234) {
			t.Errorf("Expected duration_ms=1234, got %v", logEntry["duration_ms"])
		}
	})

	// Test WithError
	t.Run("WithError", func(t *testing.T) {
		buf.Reset()
		testErr := errors.New("test error")
		errLogger := logger.WithError(testErr)
		errLogger.Error().Msg("test message")

		var logEntry map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
			t.Fatalf("Failed to parse log output: %v", err)
		}

		if logEntry["error"] != "test error" {
			t.Errorf("Expected error='test error', got %v", logEntry["error"])
		}
	})

	// Test WithError with nil
	t.Run("WithError nil", func(t *testing.T) {
		buf.Reset()
		errLogger := logger.WithError(nil)
		errLogger.Info().Msg("test message")

		var logEntry map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
			t.Fatalf("Failed to parse log output: %v", err)
		}

		if _, exists := logEntry["error"]; exists {
			t.Error("Expected no error field when error is nil")
		}
	})
}

func TestLoggerChaining(t *testing.T) {
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{Logger: zlog}

	// Test chaining multiple contextual fields
	chainedLogger := logger.
		WithRequestID("req-999").
		WithSubmissionID("sub-888").
		WithUserID("user-777").
		WithComponent("service")

	chainedLogger.Info().Msg("test message")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	// Verify all fields are present
	if logEntry["request_id"] != "req-999" {
		t.Errorf("Expected request_id=req-999, got %v", logEntry["request_id"])
	}
	if logEntry["submission_id"] != "sub-888" {
		t.Errorf("Expected submission_id=sub-888, got %v", logEntry["submission_id"])
	}
	if logEntry["user_id"] != "user-777" {
		t.Errorf("Expected user_id=user-777, got %v", logEntry["user_id"])
	}
	if logEntry["component"] != "service" {
		t.Errorf("Expected component=service, got %v", logEntry["component"])
	}
}
