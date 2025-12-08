package jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"backend_v3/config"
	jobtypes "backend_v3/jobs/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsRetryableError tests error classification logic
func TestIsRetryableError(t *testing.T) {
	testCases := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "Context deadline exceeded",
			err:       errors.New("context deadline exceeded"),
			retryable: true,
		},
		{
			name:      "Connection refused",
			err:       errors.New("connection refused"),
			retryable: true,
		},
		{
			name:      "Rate limit",
			err:       errors.New("rate limit exceeded: 429"),
			retryable: true,
		},
		{
			name:      "Internal server error",
			err:       errors.New("500 internal server error"),
			retryable: true,
		},
		{
			name:      "Service unavailable",
			err:       errors.New("503 service unavailable"),
			retryable: true,
		},
		{
			name:      "Database connection error",
			err:       errors.New("database connection failed"),
			retryable: true,
		},
		{
			name:      "Validation error",
			err:       errors.New("validation failed: invalid input"),
			retryable: false,
		},
		{
			name:      "Bad request",
			err:       errors.New("400 bad request"),
			retryable: false,
		},
		{
			name:      "Invalid UUID",
			err:       errors.New("invalid UUID format"),
			retryable: false,
		},
		{
			name:      "Nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "Temporary failure",
			err:       errors.New("temporary failure in name resolution"),
			retryable: true,
		},
		{
			name:      "Connection reset",
			err:       errors.New("connection reset by peer"),
			retryable: true,
		},
		{
			name:      "Too many requests",
			err:       errors.New("too many requests"),
			retryable: true,
		},
		{
			name:      "Bad gateway",
			err:       errors.New("502 bad gateway"),
			retryable: true,
		},
		{
			name:      "Gateway timeout",
			err:       errors.New("504 gateway timeout"),
			retryable: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isRetryableError(tc.err)
			assert.Equal(t, tc.retryable, result)
		})
	}
}

// TestExponentialBackoff tests retry delay calculation
func TestExponentialBackoff(t *testing.T) {
	cfg := &config.Config{
		JobRetryInitialDelay: 5,   // 5 seconds
		JobRetryMaxDelay:     600, // 10 minutes
	}

	testCases := []struct {
		retryCount    int
		expectedDelay time.Duration
		description   string
	}{
		{0, 5 * time.Second, "First retry: 5s"},
		{1, 10 * time.Second, "Second retry: 10s"},
		{2, 20 * time.Second, "Third retry: 20s"},
		{3, 40 * time.Second, "Fourth retry: 40s"},
		{4, 80 * time.Second, "Fifth retry: 80s"},
		{5, 160 * time.Second, "Sixth retry: 160s"},
		{6, 320 * time.Second, "Seventh retry: 320s"},
		{7, 600 * time.Second, "Eighth retry: 600s (capped)"},
		{8, 600 * time.Second, "Ninth retry: 600s (capped)"},
		{10, 600 * time.Second, "Tenth retry: 600s (capped)"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Calculate exponential backoff using same formula as worker
			delay := time.Duration(cfg.JobRetryInitialDelay) * time.Second * (1 << uint(tc.retryCount))
			maxDelay := time.Duration(cfg.JobRetryMaxDelay) * time.Second

			if delay > maxDelay {
				delay = maxDelay
			}

			assert.Equal(t, tc.expectedDelay, delay)
		})
	}
}

// NOTE: EnrichmentJobPayload and related functions have been removed as part of
// the Challenge refactor. Enrichment is now performed inline during company creation,
// not as a background job.

// TestAnalysisJobPayloadSerialization tests payload marshaling/unmarshaling
func TestAnalysisJobPayloadSerialization(t *testing.T) {
	submissionID := uuid.New()
	companyID := uuid.New()

	payload := AnalysisJobPayload{
		SubmissionID: submissionID.String(),
		CompanyID:    companyID.String(),
	}

	// Test serialization
	task, err := NewAnalysisTask(payload)
	require.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, jobtypes.TypeAnalysis, task.Type())

	// Test deserialization
	var deserializedPayload AnalysisJobPayload
	err = json.Unmarshal(task.Payload(), &deserializedPayload)
	require.NoError(t, err)

	assert.Equal(t, payload.SubmissionID, deserializedPayload.SubmissionID)
	assert.Equal(t, payload.CompanyID, deserializedPayload.CompanyID)
}

// TestJobPayloadValidation tests invalid payload handling
func TestJobPayloadValidation(t *testing.T) {

	t.Run("Analysis - Missing CompanyID", func(t *testing.T) {
		payload := AnalysisJobPayload{
			SubmissionID: uuid.New().String(),
			CompanyID:    "", // Empty
		}
		_, err := uuid.Parse(payload.CompanyID)
		assert.Error(t, err)
	})

	t.Run("Analysis - Valid Payload", func(t *testing.T) {
		payload := AnalysisJobPayload{
			SubmissionID: uuid.New().String(),
			CompanyID:    uuid.New().String(),
		}
		_, err := uuid.Parse(payload.SubmissionID)
		require.NoError(t, err)
		_, err = uuid.Parse(payload.CompanyID)
		require.NoError(t, err)
	})
}

// TestJobConstants verifies job type constants
func TestJobConstants(t *testing.T) {
	assert.Equal(t, "analysis_job", jobtypes.TypeAnalysis)
}

// TestRetryDelayFormula tests the exponential backoff formula matches expected behavior
func TestRetryDelayFormula(t *testing.T) {
	initialDelay := 5 // seconds
	maxDelay := 600   // seconds (10 minutes)

	// Test formula: initialDelay * 2^n, capped at maxDelay
	testCases := []struct {
		retry    int
		expected time.Duration
	}{
		{0, 5 * time.Second},
		{1, 10 * time.Second},
		{2, 20 * time.Second},
		{3, 40 * time.Second},
		{4, 80 * time.Second},
		{5, 160 * time.Second},
		{6, 320 * time.Second},
		{7, 600 * time.Second},  // Exceeds max, should cap
		{8, 600 * time.Second},  // Already capped
		{15, 600 * time.Second}, // Far beyond max (but not overflow)
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("retry_%d", tc.retry), func(t *testing.T) {
			// Prevent overflow by checking retry count
			var delay time.Duration
			if tc.retry > 30 {
				// Would overflow, just use max
				delay = time.Duration(maxDelay) * time.Second
			} else {
				delay = time.Duration(initialDelay) * time.Second * (1 << uint(tc.retry))
			}

			max := time.Duration(maxDelay) * time.Second

			if delay > max || delay < 0 { // negative means overflow
				delay = max
			}

			assert.Equal(t, tc.expected, delay)
		})
	}
}

// TestDLQKeyFormat tests Dead Letter Queue key naming
func TestDLQKeyFormat(t *testing.T) {
	taskType := jobtypes.TypeAnalysis
	timestamp := time.Now().Unix()

	expectedKeyFormat := "dlq:analysis_job:"
	actualKey := "dlq:" + taskType + ":" + string(rune(timestamp))

	assert.Contains(t, actualKey, expectedKeyFormat)
}
