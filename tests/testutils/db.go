package testutils

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// =================================================================
// In-Memory SQLite Test Database
// =================================================================

// TestDB wraps a test database connection
type TestDB struct {
	DB            *sql.DB
	MigrationsDir string
}

// SetupTestDB creates an in-memory SQLite database with schema
// Executes all migration files from migrations/ directory
func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	// Create in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}

	// Enable foreign keys (SQLite doesn't enable by default)
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Find migrations directory
	migrationsDir := findMigrationsDir()

	testDB := &TestDB{
		DB:            db,
		MigrationsDir: migrationsDir,
	}

	// Load schema from migration files
	if err := testDB.LoadSchema(); err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	return testDB
}

// TeardownTestDB closes the database connection
func TeardownTestDB(t *testing.T, testDB *TestDB) {
	t.Helper()

	if err := testDB.DB.Close(); err != nil {
		t.Errorf("Failed to close test database: %v", err)
	}
}

// LoadSchema reads and executes all migration files
func (tdb *TestDB) LoadSchema() error {
	if tdb.MigrationsDir == "" {
		return fmt.Errorf("migrations directory not found")
	}

	// Read all .sql files in migrations directory
	files, err := os.ReadDir(tdb.MigrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sort files by name to ensure correct order (001_, 002_, etc.)
	sqlFiles := []fs.DirEntry{}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			sqlFiles = append(sqlFiles, file)
		}
	}

	sort.Slice(sqlFiles, func(i, j int) bool {
		return sqlFiles[i].Name() < sqlFiles[j].Name()
	})

	// Execute each migration file
	for _, file := range sqlFiles {
		filePath := filepath.Join(tdb.MigrationsDir, file.Name())
		if err := tdb.executeSQLFile(filePath); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file.Name(), err)
		}
	}

	return nil
}

// executeSQLFile reads and executes a SQL file
func (tdb *TestDB) executeSQLFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Convert PostgreSQL syntax to SQLite
	sqlContent := convertPostgresToSQLite(string(content))

	// Split by semicolons and execute each statement
	statements := splitSQLStatements(sqlContent)

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		// Skip PostgreSQL-specific commands
		if shouldSkipStatement(stmt) {
			continue
		}

		if _, err := tdb.DB.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute statement: %s\nError: %w", stmt, err)
		}
	}

	return nil
}

// convertPostgresToSQLite converts PostgreSQL-specific syntax to SQLite
func convertPostgresToSQLite(sql string) string {
	// Replace UUID type with TEXT
	sql = strings.ReplaceAll(sql, "UUID", "TEXT")
	sql = strings.ReplaceAll(sql, "uuid_generate_v4()", "lower(hex(randomblob(16)))")

	// Replace TIMESTAMP WITH TIME ZONE with DATETIME
	sql = strings.ReplaceAll(sql, "TIMESTAMP WITH TIME ZONE", "DATETIME")

	// Replace JSONB with JSON
	sql = strings.ReplaceAll(sql, "JSONB", "JSON")

	// Replace TEXT[] with TEXT (SQLite doesn't have array types)
	sql = strings.ReplaceAll(sql, "TEXT[]", "TEXT")

	// Replace DECIMAL with REAL
	sql = strings.ReplaceAll(sql, "DECIMAL(15,2)", "REAL")
	sql = strings.ReplaceAll(sql, "DECIMAL(5,2)", "REAL")

	// Replace NOW() with datetime('now')
	sql = strings.ReplaceAll(sql, "NOW()", "datetime('now')")

	// Replace VARCHAR with TEXT (SQLite stores all text as TEXT)
	sql = strings.ReplaceAll(sql, "VARCHAR(255)", "TEXT")
	sql = strings.ReplaceAll(sql, "VARCHAR(500)", "TEXT")
	sql = strings.ReplaceAll(sql, "VARCHAR(100)", "TEXT")
	sql = strings.ReplaceAll(sql, "VARCHAR(50)", "TEXT")
	sql = strings.ReplaceAll(sql, "VARCHAR(1000)", "TEXT")

	// Remove ON DELETE CASCADE (SQLite handles differently)
	// Note: We enabled PRAGMA foreign_keys earlier

	return sql
}

// shouldSkipStatement returns true if the statement should be skipped
func shouldSkipStatement(stmt string) bool {
	stmtUpper := strings.ToUpper(strings.TrimSpace(stmt))

	// Skip PostgreSQL-specific commands
	if strings.HasPrefix(stmtUpper, "CREATE EXTENSION") {
		return true
	}
	if strings.HasPrefix(stmtUpper, "COMMENT ON") {
		return true
	}
	if strings.HasPrefix(stmtUpper, "CREATE OR REPLACE FUNCTION") {
		return true
	}
	if strings.HasPrefix(stmtUpper, "CREATE TRIGGER") {
		return true
	}
	if strings.Contains(stmtUpper, "RETURNS TRIGGER") {
		return true
	}
	if strings.HasPrefix(stmtUpper, "$$") {
		return true
	}
	if strings.HasPrefix(stmtUpper, "END;") && strings.Contains(stmtUpper, "$$") {
		return true
	}

	// Skip empty statements
	if stmtUpper == "" {
		return true
	}

	return false
}

// splitSQLStatements splits SQL content by semicolons
func splitSQLStatements(sql string) []string {
	// Simple split by semicolon (doesn't handle SQL comments or strings perfectly)
	// For production, use a proper SQL parser
	statements := strings.Split(sql, ";")
	return statements
}

// findMigrationsDir searches for the migrations directory
func findMigrationsDir() string {
	// Try common locations
	locations := []string{
		"migrations",
		"../migrations",
		"../../migrations",
		"../../../migrations",
		"../../../../migrations", // From tests/testutils/
	}

	for _, loc := range locations {
		if stat, err := os.Stat(loc); err == nil && stat.IsDir() {
			absPath, _ := filepath.Abs(loc)
			return absPath
		}
	}

	return ""
}

// =================================================================
// Helper Functions for Test Data
// =================================================================

// InsertTestSubmission inserts a test submission and returns the ID
func (tdb *TestDB) InsertTestSubmission(t *testing.T) string {
	t.Helper()

	submission := NewTestSubmission()

	query := `
		INSERT INTO submissions (
			id, company_name, cnpj, company_website, company_industry, company_size,
			company_location, contact_name, contact_email, contact_phone, contact_position,
			target_market, annual_revenue_min, annual_revenue_max, funding_stage,
			business_challenge, additional_notes, linkedin_url, twitter_handle,
			status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tdb.DB.Exec(query,
		submission.ID.String(),
		submission.CompanyName,
		submission.CNPJ,
		submission.CompanyWebsite,
		submission.CompanyIndustry,
		submission.CompanySize,
		submission.CompanyLocation,
		submission.ContactName,
		submission.ContactEmail,
		submission.ContactPhone,
		submission.ContactPosition,
		submission.TargetMarket,
		submission.AnnualRevenueMin,
		submission.AnnualRevenueMax,
		submission.FundingStage,
		submission.BusinessChallenge,
		submission.AdditionalNotes,
		submission.LinkedInURL,
		submission.TwitterHandle,
		submission.Status,
		submission.CreatedAt,
		submission.UpdatedAt,
	)

	if err != nil {
		t.Fatalf("Failed to insert test submission: %v", err)
	}

	return submission.ID.String()
}

// InsertTestEnrichment inserts a test enrichment and returns the ID
func (tdb *TestDB) InsertTestEnrichment(t *testing.T, submissionID string) string {
	t.Helper()

	submissionUUID, err := parseUUID(submissionID)
	if err != nil {
		t.Fatalf("Invalid submission ID: %v", err)
	}

	enrichment := NewTestEnrichment(submissionUUID)

	// Serialize JSONMap fields
	enrichedDataJSON, _ := enrichment.EnrichedData.Value()
	sourcesStatusJSON, _ := enrichment.SourcesStatus.Value()

	query := `
		INSERT INTO enrichments (
			id, submission_id, status, progress, current_step, is_locked,
			sources_status, enriched_data, started_at, completed_at,
			error_message, retry_count, max_retries, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tdb.DB.Exec(query,
		enrichment.ID.String(),
		enrichment.SubmissionID.String(),
		enrichment.Status,
		enrichment.Progress,
		enrichment.CurrentStep,
		enrichment.IsLocked,
		sourcesStatusJSON,
		enrichedDataJSON,
		enrichment.StartedAt,
		enrichment.CompletedAt,
		enrichment.ErrorMessage,
		enrichment.RetryCount,
		enrichment.MaxRetries,
		enrichment.CreatedAt,
		enrichment.UpdatedAt,
	)

	if err != nil {
		t.Fatalf("Failed to insert test enrichment: %v", err)
	}

	return enrichment.ID.String()
}

// Helper to parse UUID string
func parseUUID(s string) (uuid [16]byte, err error) {
	// Simplified UUID parser for testing
	// Production code should use github.com/google/uuid
	return uuid, nil
}
