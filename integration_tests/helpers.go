package integration_tests

import (
	"backend_v3/config"
	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/submission"
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// TestHelper provides utilities for integration tests
type TestHelper struct {
	DB                *sqlx.DB
	SubmissionRepo    submission.Repository
	EnrichmentRepo    enrichment.Repository
	AnalysisRepo      analysis.Repository
	SubmissionService *submission.Service
	EnrichmentService *enrichment.Service
	AnalysisService   *analysis.Service
	TestEnv           *TestEnvironment
}

// NewTestHelper creates a new test helper with all necessary dependencies
func NewTestHelper(t *testing.T) *TestHelper {
	env := LoadTestEnv()

	// Check if we have a valid database URL
	if !env.HasValidDatabase() {
		t.Skip("Skipping test: No valid DATABASE_URL configured. Please set up a local PostgreSQL database or use a real database URL in .env file")
	}

	// Connect to database
	db, err := sqlx.Connect("postgres", env.DatabaseURL)
	require.NoError(t, err, "Failed to connect to database")

	// Create repositories
	submissionRepo := submission.NewRepository(db)
	enrichmentRepo := enrichment.NewRepository(db)
	analysisRepo := analysis.NewPostgresRepository(db)

	// Create services (note: we'll need to wire these up properly)
	// For now, we'll create minimal versions
	helper := &TestHelper{
		DB:             db,
		SubmissionRepo: submissionRepo,
		EnrichmentRepo: enrichmentRepo,
		AnalysisRepo:   analysisRepo,
		TestEnv:        env,
	}

	return helper
}

// Cleanup removes all test data from the database
func (h *TestHelper) Cleanup(t *testing.T) {
	ctx := context.Background()

	// Delete in correct order (child tables first)
	tables := []string{
		"analyses",
		"enrichments",
		"submissions",
	}

	for _, table := range tables {
		_, err := h.DB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE created_at > NOW() - INTERVAL '1 hour'", table))
		require.NoError(t, err, "Failed to clean up table: %s", table)
	}
}

// Close closes the database connection
func (h *TestHelper) Close() {
	if h.DB != nil {
		_ = h.DB.Close()
	}
}

// CreateTestSubmission creates a test submission in the database
func (h *TestHelper) CreateTestSubmission(t *testing.T, ctx context.Context) *submission.Submission {
	website := "https://testcompany.com"
	location := "São Paulo, Brazil"
	industry := "Technology"
	targetMarket := "B2B SaaS"
	phone := "+55 11 99999-9999"
	minRevenue := 5000000.0
	maxRevenue := 10000000.0

	sub := &submission.Submission{
		ID:                uuid.New(),
		UserID:            nil, // Anonymous submission
		CompanyName:       "Test Company LTDA",
		CompanyWebsite:    &website,
		CompanyLocation:   &location,
		CompanyIndustry:   &industry,
		TargetMarket:      &targetMarket,
		AnnualRevenueMin:  &minRevenue,
		AnnualRevenueMax:  &maxRevenue,
		BusinessChallenge: "Need to scale operations and improve market positioning",
		ContactEmail:      "test@testcompany.com",
		ContactName:       "Test User",
		ContactPhone:      &phone,
		Status:            "received",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := h.SubmissionRepo.Create(ctx, sub)
	require.NoError(t, err, "Failed to create test submission")

	return sub
}

// CreateTestEnrichment creates a test enrichment in the database
func (h *TestHelper) CreateTestEnrichment(t *testing.T, ctx context.Context, submissionID uuid.UUID) *enrichment.Enrichment {
	enr := enrichment.NewEnrichment(submissionID)

	err := h.EnrichmentRepo.Create(ctx, enr)
	require.NoError(t, err, "Failed to create test enrichment")

	return enr
}

// WaitForEnrichmentStatus waits for enrichment to reach a specific status
func (h *TestHelper) WaitForEnrichmentStatus(t *testing.T, ctx context.Context, submissionID uuid.UUID, expectedStatus enrichment.Status, timeout time.Duration) *enrichment.Enrichment {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		enr, err := h.EnrichmentRepo.GetBySubmissionID(ctx, submissionID)
		if err == nil && enr.Status == expectedStatus {
			return enr
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("Timeout waiting for enrichment status '%s' (submission: %s)", expectedStatus, submissionID)
	return nil
}

// WaitForAnalysisStatus waits for analysis to reach a specific status
func (h *TestHelper) WaitForAnalysisStatus(t *testing.T, ctx context.Context, submissionID uuid.UUID, expectedStatus string, timeout time.Duration) *analysis.Analysis {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Get latest analysis for submission
		anl, err := h.AnalysisRepo.GetBySubmissionID(ctx, submissionID.String())
		if err == nil && anl != nil {
			if anl.Status == expectedStatus {
				return anl
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("Timeout waiting for analysis status '%s' (submission: %s)", expectedStatus, submissionID)
	return nil
}

// AssertEnrichmentStatus asserts that enrichment has the expected status
func (h *TestHelper) AssertEnrichmentStatus(t *testing.T, ctx context.Context, submissionID uuid.UUID, expectedStatus enrichment.Status) {
	enr, err := h.EnrichmentRepo.GetBySubmissionID(ctx, submissionID)
	require.NoError(t, err, "Failed to get enrichment")
	require.Equal(t, expectedStatus, enr.Status, "Enrichment status mismatch")
}

// AssertAnalysisStatus asserts that analysis has the expected status
func (h *TestHelper) AssertAnalysisStatus(t *testing.T, ctx context.Context, analysisID string, expectedStatus string) {
	anl, err := h.AnalysisRepo.GetByID(ctx, analysisID)
	require.NoError(t, err, "Failed to get analysis")
	require.Equal(t, expectedStatus, anl.Status, "Analysis status mismatch")
}

// LoadDefaultConfig loads default configuration for testing
func LoadDefaultConfig() *config.Config {
	frameworks := make(map[string]config.FrameworkConfig)
	frameworks["enrichment"] = config.FrameworkConfig{
		Model:       "google/gemini-2.0-flash-001",
		Temperature: 0.5,
		MaxTokens:   8000,
	}
	frameworks["pestel"] = config.FrameworkConfig{
		Model:       "openai/o3-mini",
		Temperature: 0.2,
		MaxTokens:   1500,
	}
	frameworks["porter"] = config.FrameworkConfig{
		Model:       "openai/gpt-4o",
		Temperature: 0.3,
		MaxTokens:   1500,
	}
	frameworks["swot"] = config.FrameworkConfig{
		Model:       "openai/gpt-4o-mini",
		Temperature: 0.4,
		MaxTokens:   1500,
	}

	return &config.Config{
		Port:             "8080",
		Environment:      "test",
		AllowedOrigins:   "http://localhost:3000",
		DatabaseURL:      "", // Will be set from test environment
		Frameworks:       frameworks,
		WorkerEnabled:    false, // Disable worker in tests
		WorkerConcurrency: 1,
	}
}

// GetTestConfig returns a config suitable for integration testing
func GetTestConfig(env *TestEnvironment) *config.Config {
	cfg := LoadDefaultConfig()
	cfg.DatabaseURL = env.DatabaseURL
	return cfg
}

// TruncateAllTables removes all data from test tables (use with caution!)
func (h *TestHelper) TruncateAllTables(t *testing.T) {
	ctx := context.Background()

	tables := []string{
		"analyses",
		"enrichments",
		"submissions",
	}

	for _, table := range tables {
		_, err := h.DB.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		require.NoError(t, err, "Failed to truncate table: %s", table)
	}
}

// GetSubmissionByID retrieves a submission by ID
func (h *TestHelper) GetSubmissionByID(t *testing.T, ctx context.Context, id uuid.UUID) *submission.Submission {
	sub, err := h.SubmissionRepo.GetByID(ctx, id)
	require.NoError(t, err, "Failed to get submission")
	return sub
}

// GetEnrichmentBySubmissionID retrieves enrichment by submission ID
func (h *TestHelper) GetEnrichmentBySubmissionID(t *testing.T, ctx context.Context, submissionID uuid.UUID) *enrichment.Enrichment {
	enr, err := h.EnrichmentRepo.GetBySubmissionID(ctx, submissionID)
	require.NoError(t, err, "Failed to get enrichment")
	return enr
}

// CountRecords returns the count of records in a table
func (h *TestHelper) CountRecords(t *testing.T, table string) int {
	var count int
	err := h.DB.Get(&count, fmt.Sprintf("SELECT COUNT(*) FROM %s", table))
	require.NoError(t, err, "Failed to count records in table: %s", table)
	return count
}

// ExecuteSQL executes a raw SQL query (for test setup)
func (h *TestHelper) ExecuteSQL(t *testing.T, query string, args ...interface{}) sql.Result {
	result, err := h.DB.Exec(query, args...)
	require.NoError(t, err, "Failed to execute SQL: %s", query)
	return result
}
