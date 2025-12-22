package framework

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	return sqlxDB, mock
}

func createTestFramework() *Framework {
	return &Framework{
		ID:          uuid.New(),
		Code:        "pestel",
		Name:        "PESTEL Analysis",
		Description: stringPtr("Macro-environmental analysis"),
		Layer:       1,
		IsBase:      true,
		IsActive:    true,
		PromptUser:  "Analyze PESTEL factors...",
		ModelConfig: ModelConfig{
			Model:         "google/gemini-2.5-flash",
			Temperature:   0.5,
			MaxTokens:     8000,
			FallbackModel: "openai/gpt-4.1-mini",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func createTestResult() *CompanyFrameworkResult {
	now := time.Now()
	return &CompanyFrameworkResult{
		ID:          uuid.New(),
		CompanyID:   uuid.New(),
		FrameworkID: uuid.New(),
		Status:      StatusPending,
		Version:     1,
		IsCurrent:   true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func stringPtr(s string) *string {
	return &s
}

// =============================================================================
// Framework CRUD Tests
// =============================================================================

func TestRepositoryCreate(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	framework := createTestFramework()

	mock.ExpectExec("INSERT INTO frameworks").
		WithArgs(
			sqlmock.AnyArg(), // id
			framework.Code,
			framework.Name,
			framework.Description,
			framework.Layer,
			framework.IsBase,
			framework.IsActive,
			framework.PromptSystem,
			framework.PromptUser,
			framework.PromptJSONTemplate,
			sqlmock.AnyArg(), // model_config
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, framework)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryGetByCode(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	framework := createTestFramework()
	modelConfigJSON, _ := json.Marshal(framework.ModelConfig)

	rows := sqlmock.NewRows([]string{
		"id", "code", "name", "description", "layer", "is_base", "is_active",
		"prompt_system", "prompt_user", "prompt_json_template",
		"model_config", "created_at", "updated_at",
	}).AddRow(
		framework.ID, framework.Code, framework.Name, framework.Description,
		framework.Layer, framework.IsBase, framework.IsActive,
		framework.PromptSystem, framework.PromptUser, framework.PromptJSONTemplate,
		modelConfigJSON, framework.CreatedAt, framework.UpdatedAt,
	)

	mock.ExpectQuery("SELECT \\* FROM frameworks WHERE code = \\$1").
		WithArgs("pestel").
		WillReturnRows(rows)

	result, err := repo.GetByCode(ctx, "pestel")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "pestel", result.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryGetByCodeNotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	mock.ExpectQuery("SELECT \\* FROM frameworks WHERE code = \\$1").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByCode(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryGetActive(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	framework := createTestFramework()
	modelConfigJSON, _ := json.Marshal(framework.ModelConfig)

	rows := sqlmock.NewRows([]string{
		"id", "code", "name", "description", "layer", "is_base", "is_active",
		"prompt_system", "prompt_user", "prompt_json_template",
		"model_config", "created_at", "updated_at",
	}).AddRow(
		framework.ID, framework.Code, framework.Name, framework.Description,
		framework.Layer, framework.IsBase, framework.IsActive,
		framework.PromptSystem, framework.PromptUser, framework.PromptJSONTemplate,
		modelConfigJSON, framework.CreatedAt, framework.UpdatedAt,
	)

	mock.ExpectQuery("SELECT \\* FROM frameworks WHERE is_active = true ORDER BY layer, code").
		WillReturnRows(rows)

	results, err := repo.GetActive(ctx)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "pestel", results[0].Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryGetBaseFrameworks(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	framework := createTestFramework()
	modelConfigJSON, _ := json.Marshal(framework.ModelConfig)

	rows := sqlmock.NewRows([]string{
		"id", "code", "name", "description", "layer", "is_base", "is_active",
		"prompt_system", "prompt_user", "prompt_json_template",
		"model_config", "created_at", "updated_at",
	}).AddRow(
		framework.ID, framework.Code, framework.Name, framework.Description,
		framework.Layer, framework.IsBase, framework.IsActive,
		framework.PromptSystem, framework.PromptUser, framework.PromptJSONTemplate,
		modelConfigJSON, framework.CreatedAt, framework.UpdatedAt,
	)

	mock.ExpectQuery("SELECT \\* FROM frameworks WHERE is_base = true AND is_active = true ORDER BY layer, code").
		WillReturnRows(rows)

	results, err := repo.GetBaseFrameworks(ctx)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.True(t, results[0].IsBase)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// =============================================================================
// Dependencies Tests
// =============================================================================

func TestRepositoryDependencies(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	frameworkID := uuid.New()
	dependsOnID := uuid.New()

	dep := &FrameworkDependency{
		FrameworkID:    frameworkID,
		DependsOnID:    dependsOnID,
		DependencyType: DependencyRequired,
	}

	// Test AddDependency
	mock.ExpectExec("INSERT INTO framework_dependencies").
		WithArgs(
			sqlmock.AnyArg(), // id
			dep.FrameworkID,
			dep.DependsOnID,
			dep.DependencyType,
			sqlmock.AnyArg(), // created_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.AddDependency(ctx, dep)
	assert.NoError(t, err)

	// Test GetDependencies
	rows := sqlmock.NewRows([]string{
		"id", "framework_id", "depends_on_id", "dependency_type", "created_at",
	}).AddRow(
		uuid.New(), frameworkID, dependsOnID, DependencyRequired, time.Now(),
	)

	mock.ExpectQuery("SELECT \\* FROM framework_dependencies WHERE framework_id = \\$1").
		WithArgs(frameworkID).
		WillReturnRows(rows)

	deps, err := repo.GetDependencies(ctx, frameworkID)
	assert.NoError(t, err)
	assert.Len(t, deps, 1)
	assert.Equal(t, dependsOnID, deps[0].DependsOnID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// =============================================================================
// Company Framework Results Tests
// =============================================================================

func TestRepositorySaveCompanyResult(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	result := createTestResult()

	mock.ExpectExec("INSERT INTO company_framework_results").
		WithArgs(
			sqlmock.AnyArg(), // id
			result.CompanyID,
			result.FrameworkID,
			result.ChallengeID,
			result.Result,
			result.Status,
			result.ErrorMessage,
			result.Version,
			result.IsCurrent,
			result.ContextHash,
			result.GeneratedAt,
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateResult(ctx, result)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryGetCompanyResults(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	companyID := uuid.New()
	frameworkID := uuid.New()
	resultID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "company_id", "framework_id", "challenge_id",
		"result", "status", "error_message",
		"version", "is_current", "context_hash",
		"generated_at", "created_at", "updated_at",
	}).AddRow(
		resultID, companyID, frameworkID, nil,
		json.RawMessage(`{"summary": "test"}`), StatusCompleted, nil,
		1, true, nil,
		now, now, now,
	)

	mock.ExpectQuery("SELECT \\* FROM company_framework_results").
		WithArgs(companyID).
		WillReturnRows(rows)

	results, err := repo.GetCompanyResults(ctx, companyID, nil)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, StatusCompleted, results[0].Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryUpdateResultStatus(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	resultID := uuid.New()
	errorMsg := "LLM timeout"

	mock.ExpectExec("UPDATE company_framework_results").
		WithArgs(resultID, StatusFailed, &errorMsg, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateResultStatus(ctx, resultID, StatusFailed, &errorMsg)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryUpdateResultCompleted(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	resultID := uuid.New()
	resultData := []byte(`{"political": ["Factor 1"]}`)
	generatedAt := time.Now()

	mock.ExpectExec("UPDATE company_framework_results").
		WithArgs(resultID, StatusCompleted, resultData, generatedAt, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateResultCompleted(ctx, resultID, resultData, generatedAt)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryInvalidateResults(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	companyID := uuid.New()
	frameworkID := uuid.New()

	mock.ExpectExec("UPDATE company_framework_results").
		WithArgs(companyID, frameworkID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))

	err := repo.InvalidateResults(ctx, companyID, frameworkID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
