//go:build integration
// +build integration

package framework

import (
	"context"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestDB(t *testing.T) *sqlx.DB {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres.mtrgloqfsijrykjrckrb:XSxdSfEiF0EDDHUI@aws-1-us-east-1.pooler.supabase.com:5432/postgres"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	require.NoError(t, err, "Failed to connect to database")
	return db
}

func TestIntegration_GetActiveFrameworks(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()

	// Get all active frameworks
	frameworks, err := svc.GetActive(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(frameworks), 12, "Should have at least 12 frameworks seeded")

	// Verify expected frameworks exist
	codes := make(map[string]bool)
	for _, fw := range frameworks {
		codes[fw.Code] = true
	}

	expectedCodes := []string{"pestel", "porter", "swot", "tam_sam_som", "benchmarking", "blue_ocean", "scenarios", "growth_hacking", "decision_matrix", "okrs", "bsc", "synthesis"}
	for _, code := range expectedCodes {
		assert.True(t, codes[code], "Framework %s should exist", code)
	}
}

func TestIntegration_GetFrameworkByCode(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()

	// Get PESTEL framework
	fw, err := svc.GetByCode(ctx, "pestel")
	require.NoError(t, err)
	require.NotNil(t, fw)

	assert.Equal(t, "pestel", fw.Code)
	assert.Equal(t, "PESTEL Analysis", fw.Name)
	assert.Equal(t, 1, fw.Layer)
	assert.True(t, fw.IsBase)
	assert.True(t, fw.IsActive)
	assert.NotEmpty(t, fw.PromptUser)
}

func TestIntegration_BuildExecutionPlan(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()

	// Build execution plan
	plan, err := svc.BuildExecutionPlan(ctx)
	require.NoError(t, err)
	require.NotNil(t, plan)

	// Should have multiple layers
	assert.GreaterOrEqual(t, len(plan.Layers), 4, "Should have at least 4 layers")

	// Layer 1 should have PESTEL, Porter, TAM-SAM-SOM
	layer1 := plan.Layers[0]
	assert.Equal(t, 1, layer1.LayerNumber)
	assert.GreaterOrEqual(t, len(layer1.Frameworks), 3, "Layer 1 should have at least 3 frameworks")
}

func TestIntegration_ResolveDependencies(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()

	// Get SWOT framework (has dependencies on PESTEL and Porter)
	swot, err := svc.GetByCode(ctx, "swot")
	require.NoError(t, err)
	require.NotNil(t, swot)

	// Resolve dependencies
	deps, err := svc.ResolveDependencies(ctx, swot.ID)
	require.NoError(t, err)

	// SWOT should depend on PESTEL and Porter
	assert.GreaterOrEqual(t, len(deps), 2, "SWOT should have at least 2 dependencies")

	depCodes := make(map[string]bool)
	for _, dep := range deps {
		depCodes[dep.Code] = true
	}

	assert.True(t, depCodes["pestel"], "SWOT should depend on PESTEL")
	assert.True(t, depCodes["porter"], "SWOT should depend on Porter")
}

func TestIntegration_FrameworkModelConfig(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	logger := zerolog.Nop()
	svc := NewService(repo, logger)
	ctx := context.Background()

	// Get PESTEL and check model config
	fw, err := svc.GetByCode(ctx, "pestel")
	require.NoError(t, err)
	require.NotNil(t, fw)

	// Should have valid model config
	assert.NotEmpty(t, fw.ModelConfig.Model, "Model should be configured")
	assert.Greater(t, fw.ModelConfig.MaxTokens, 0, "MaxTokens should be > 0")
	assert.Greater(t, fw.ModelConfig.Temperature, 0.0, "Temperature should be > 0")
}
