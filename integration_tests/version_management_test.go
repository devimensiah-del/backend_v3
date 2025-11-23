package integration_tests

import (
	"backend_v3/domain/analysis"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersionManagement_CreateVersion verifies version creation
func TestVersionManagement_CreateVersion(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()
	defer helper.Cleanup(t)

	ctx := context.Background()

	t.Run("CreateVersion() creates new analysis with incremented version", func(t *testing.T) {
		// Create submission and enrichment
		sub := helper.CreateTestSubmission(t, ctx)
		enr := helper.CreateTestEnrichment(t, ctx, sub.ID)
		enr.Finish()
		err := helper.EnrichmentRepo.UpdateSystem(ctx, enr)
		require.NoError(t, err)

		// Create version 1 analysis
		v1Analysis := &analysis.Analysis{
			ID:           uuid.New().String(),
			SubmissionID: sub.ID.String(),
			EnrichmentID: enr.ID.String(),
			Status:       string(analysis.StatusCompleted),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Version:      1,
			IsLatest:     true,
		}

		v1Analysis.PESTEL = analysis.PESTELAnalysis{Summary: "Version 1 PESTEL"}
		v1Analysis.Porter = analysis.PorterAnalysis{Summary: "Version 1 Porter"}

		err = helper.AnalysisRepo.Create(ctx, v1Analysis)
		require.NoError(t, err)
		t.Logf("✓ Version 1 created: %s (version: %d, is_latest: %t)", v1Analysis.ID, v1Analysis.Version, v1Analysis.IsLatest)

		// Create version 2 using CreateNewVersion()
		v2Analysis := v1Analysis.CreateNewVersion()
		v2Analysis.ID = uuid.New().String()
		require.Equal(t, 2, v2Analysis.Version)
		require.NotNil(t, v2Analysis.ParentAnalysisID)
		require.Equal(t, v1Analysis.ID, *v2Analysis.ParentAnalysisID)

		// Verify version 2 copied all framework data
		assert.Equal(t, v1Analysis.PESTEL.Summary, v2Analysis.PESTEL.Summary)
		assert.Equal(t, v1Analysis.Porter.Summary, v2Analysis.Porter.Summary)

		// Update version 1 to not be latest
		v1Analysis.IsLatest = false
		err = helper.AnalysisRepo.Update(ctx, v1Analysis)
		require.NoError(t, err)

		// Save version 2
		err = helper.AnalysisRepo.Create(ctx, v2Analysis)
		require.NoError(t, err)
		t.Logf("✓ Version 2 created: %s (version: %d, parent: %s, is_latest: %t)",
			v2Analysis.ID, v2Analysis.Version, *v2Analysis.ParentAnalysisID, v2Analysis.IsLatest)

		// Verify both versions exist
		retrievedV1, err := helper.AnalysisRepo.GetByID(ctx, v1Analysis.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, retrievedV1.Version)
		assert.False(t, retrievedV1.IsLatest)

		retrievedV2, err := helper.AnalysisRepo.GetByID(ctx, v2Analysis.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, retrievedV2.Version)
		assert.True(t, retrievedV2.IsLatest)

		t.Log("✓ Version creation verified: v1 (not latest), v2 (latest)")
	})
}

// TestVersionManagement_ParentLink verifies parent_analysis_id links
func TestVersionManagement_ParentLink(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()
	defer helper.Cleanup(t)

	ctx := context.Background()

	t.Run("parent_analysis_id links to previous version", func(t *testing.T) {
		// Create submission and enrichment
		sub := helper.CreateTestSubmission(t, ctx)
		enr := helper.CreateTestEnrichment(t, ctx, sub.ID)
		enr.Finish()
		err := helper.EnrichmentRepo.UpdateSystem(ctx, enr)
		require.NoError(t, err)

		// Create version 1 (no parent)
		v1Analysis := &analysis.Analysis{
			ID:               uuid.New().String(),
			SubmissionID:     sub.ID.String(),
			EnrichmentID:     enr.ID.String(),
			Status:           string(analysis.StatusCompleted),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			Version:          1,
			IsLatest:         false, // Will be updated to false when v2 is created
			ParentAnalysisID: nil,   // Version 1 has no parent
		}

		err = helper.AnalysisRepo.Create(ctx, v1Analysis)
		require.NoError(t, err)
		require.Nil(t, v1Analysis.ParentAnalysisID)
		t.Logf("✓ Version 1: parent_analysis_id = nil")

		// Create version 2 (parent = v1)
		v2Analysis := v1Analysis.CreateNewVersion()
		v2Analysis.ID = uuid.New().String()
		v2Analysis.ParentAnalysisID = &v1Analysis.ID

		err = helper.AnalysisRepo.Create(ctx, v2Analysis)
		require.NoError(t, err)
		require.NotNil(t, v2Analysis.ParentAnalysisID)
		require.Equal(t, v1Analysis.ID, *v2Analysis.ParentAnalysisID)
		t.Logf("✓ Version 2: parent_analysis_id = %s (v1)", *v2Analysis.ParentAnalysisID)

		// Create version 3 (parent = v2)
		v3Analysis := v2Analysis.CreateNewVersion()
		v3Analysis.ID = uuid.New().String()
		v3Analysis.ParentAnalysisID = &v2Analysis.ID

		// Update v2 to not be latest
		v2Analysis.IsLatest = false
		err = helper.AnalysisRepo.Update(ctx, v2Analysis)
		require.NoError(t, err)

		err = helper.AnalysisRepo.Create(ctx, v3Analysis)
		require.NoError(t, err)
		require.NotNil(t, v3Analysis.ParentAnalysisID)
		require.Equal(t, v2Analysis.ID, *v3Analysis.ParentAnalysisID)
		t.Logf("✓ Version 3: parent_analysis_id = %s (v2)", *v3Analysis.ParentAnalysisID)

		// Verify version chain: v1 → v2 → v3
		assert.Nil(t, v1Analysis.ParentAnalysisID)
		assert.Equal(t, v1Analysis.ID, *v2Analysis.ParentAnalysisID)
		assert.Equal(t, v2Analysis.ID, *v3Analysis.ParentAnalysisID)

		t.Log("✓ Parent chain verified: v1 (root) → v2 → v3 (latest)")
	})
}

// TestVersionManagement_IsLatestFlag verifies is_latest flag updates
func TestVersionManagement_IsLatestFlag(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()
	defer helper.Cleanup(t)

	ctx := context.Background()

	t.Run("is_latest flag updated correctly", func(t *testing.T) {
		// Create submission and enrichment
		sub := helper.CreateTestSubmission(t, ctx)
		enr := helper.CreateTestEnrichment(t, ctx, sub.ID)
		enr.Finish()
		err := helper.EnrichmentRepo.UpdateSystem(ctx, enr)
		require.NoError(t, err)

		// Create version 1 (is_latest = true initially)
		v1Analysis := &analysis.Analysis{
			ID:           uuid.New().String(),
			SubmissionID: sub.ID.String(),
			EnrichmentID: enr.ID.String(),
			Status:       string(analysis.StatusCompleted),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Version:      1,
			IsLatest:     true,
		}

		err = helper.AnalysisRepo.Create(ctx, v1Analysis)
		require.NoError(t, err)
		require.True(t, v1Analysis.IsLatest)
		t.Logf("✓ Version 1: is_latest = true (initially)")

		// Create version 2
		v2Analysis := v1Analysis.CreateNewVersion()
		v2Analysis.ID = uuid.New().String()

		// Update v1: is_latest = false
		v1Analysis.IsLatest = false
		err = helper.AnalysisRepo.Update(ctx, v1Analysis)
		require.NoError(t, err)

		// Create v2: is_latest = true
		err = helper.AnalysisRepo.Create(ctx, v2Analysis)
		require.NoError(t, err)

		// Verify flags
		updatedV1, err := helper.AnalysisRepo.GetByID(ctx, v1Analysis.ID)
		require.NoError(t, err)
		assert.False(t, updatedV1.IsLatest)
		t.Logf("✓ Version 1: is_latest = false (after v2 created)")

		updatedV2, err := helper.AnalysisRepo.GetByID(ctx, v2Analysis.ID)
		require.NoError(t, err)
		assert.True(t, updatedV2.IsLatest)
		t.Logf("✓ Version 2: is_latest = true (current)")

		// Create version 3
		v3Analysis := v2Analysis.CreateNewVersion()
		v3Analysis.ID = uuid.New().String()

		// Update v2: is_latest = false
		v2Analysis.IsLatest = false
		err = helper.AnalysisRepo.Update(ctx, v2Analysis)
		require.NoError(t, err)

		// Create v3: is_latest = true
		err = helper.AnalysisRepo.Create(ctx, v3Analysis)
		require.NoError(t, err)

		// Verify all flags
		finalV1, err := helper.AnalysisRepo.GetByID(ctx, v1Analysis.ID)
		require.NoError(t, err)
		assert.False(t, finalV1.IsLatest)

		finalV2, err := helper.AnalysisRepo.GetByID(ctx, v2Analysis.ID)
		require.NoError(t, err)
		assert.False(t, finalV2.IsLatest)

		finalV3, err := helper.AnalysisRepo.GetByID(ctx, v3Analysis.ID)
		require.NoError(t, err)
		assert.True(t, finalV3.IsLatest)

		t.Log("✓ is_latest flags verified: v1 (false), v2 (false), v3 (true)")
	})
}

// TestVersionManagement_GetLatestVersion verifies GetLatestVersion()
func TestVersionManagement_GetLatestVersion(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()
	defer helper.Cleanup(t)

	ctx := context.Background()

	t.Run("GetLatestVersion() returns correct version", func(t *testing.T) {
		// Create submission and enrichment
		sub := helper.CreateTestSubmission(t, ctx)
		enr := helper.CreateTestEnrichment(t, ctx, sub.ID)
		enr.Finish()
		err := helper.EnrichmentRepo.UpdateSystem(ctx, enr)
		require.NoError(t, err)

		// Create version 1
		v1Analysis := &analysis.Analysis{
			ID:           uuid.New().String(),
			SubmissionID: sub.ID.String(),
			EnrichmentID: enr.ID.String(),
			Status:       string(analysis.StatusCompleted),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Version:      1,
			IsLatest:     false, // Will be overridden
		}
		err = helper.AnalysisRepo.Create(ctx, v1Analysis)
		require.NoError(t, err)

		// Get latest (should be v1)
		latest, err := helper.AnalysisRepo.GetLatestVersionBySubmissionID(ctx, sub.ID.String())
		require.NoError(t, err)
		require.NotNil(t, latest)
		assert.Equal(t, v1Analysis.ID, latest.ID)
		assert.Equal(t, 1, latest.Version)
		t.Logf("✓ GetLatestVersion() returned v1: %s", latest.ID)

		// Create version 2
		v2Analysis := v1Analysis.CreateNewVersion()
		v2Analysis.ID = uuid.New().String()

		v1Analysis.IsLatest = false
		err = helper.AnalysisRepo.Update(ctx, v1Analysis)
		require.NoError(t, err)

		err = helper.AnalysisRepo.Create(ctx, v2Analysis)
		require.NoError(t, err)

		// Get latest (should be v2)
		latest, err = helper.AnalysisRepo.GetLatestVersionBySubmissionID(ctx, sub.ID.String())
		require.NoError(t, err)
		require.NotNil(t, latest)
		assert.Equal(t, v2Analysis.ID, latest.ID)
		assert.Equal(t, 2, latest.Version)
		t.Logf("✓ GetLatestVersion() returned v2: %s", latest.ID)

		// Create version 3
		v3Analysis := v2Analysis.CreateNewVersion()
		v3Analysis.ID = uuid.New().String()

		v2Analysis.IsLatest = false
		err = helper.AnalysisRepo.Update(ctx, v2Analysis)
		require.NoError(t, err)

		err = helper.AnalysisRepo.Create(ctx, v3Analysis)
		require.NoError(t, err)

		// Get latest (should be v3)
		latest, err = helper.AnalysisRepo.GetLatestVersionBySubmissionID(ctx, sub.ID.String())
		require.NoError(t, err)
		require.NotNil(t, latest)
		assert.Equal(t, v3Analysis.ID, latest.ID)
		assert.Equal(t, 3, latest.Version)
		t.Logf("✓ GetLatestVersion() returned v3: %s", latest.ID)

		t.Log("✓ GetLatestVersion() always returns current version")
	})
}

// TestVersionManagement_GetAllVersions verifies GetAllVersions()
func TestVersionManagement_GetAllVersions(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()
	defer helper.Cleanup(t)

	ctx := context.Background()

	t.Run("GetAllVersions() returns all versions in order", func(t *testing.T) {
		// Create submission and enrichment
		sub := helper.CreateTestSubmission(t, ctx)
		enr := helper.CreateTestEnrichment(t, ctx, sub.ID)
		enr.Finish()
		err := helper.EnrichmentRepo.UpdateSystem(ctx, enr)
		require.NoError(t, err)

		// Create version 1
		v1Analysis := &analysis.Analysis{
			ID:           uuid.New().String(),
			SubmissionID: sub.ID.String(),
			EnrichmentID: enr.ID.String(),
			Status:       string(analysis.StatusCompleted),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Version:      1,
			IsLatest:     false,
		}
		err = helper.AnalysisRepo.Create(ctx, v1Analysis)
		require.NoError(t, err)

		// Create version 2
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		v2Analysis := v1Analysis.CreateNewVersion()
		v2Analysis.ID = uuid.New().String()
		v2Analysis.CreatedAt = time.Now()
		err = helper.AnalysisRepo.Create(ctx, v2Analysis)
		require.NoError(t, err)

		// Create version 3
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		v3Analysis := v2Analysis.CreateNewVersion()
		v3Analysis.ID = uuid.New().String()
		v3Analysis.CreatedAt = time.Now()

		v2Analysis.IsLatest = false
		err = helper.AnalysisRepo.Update(ctx, v2Analysis)
		require.NoError(t, err)

		err = helper.AnalysisRepo.Create(ctx, v3Analysis)
		require.NoError(t, err)

		// Get all versions
		allVersions, err := helper.AnalysisRepo.GetAllVersionsBySubmissionID(ctx, sub.ID.String())
		require.NoError(t, err)
		require.Len(t, allVersions, 3)

		// Verify order (should be descending by version: v3, v2, v1)
		assert.Equal(t, 3, allVersions[0].Version)
		assert.Equal(t, v3Analysis.ID, allVersions[0].ID)

		assert.Equal(t, 2, allVersions[1].Version)
		assert.Equal(t, v2Analysis.ID, allVersions[1].ID)

		assert.Equal(t, 1, allVersions[2].Version)
		assert.Equal(t, v1Analysis.ID, allVersions[2].ID)

		t.Logf("✓ GetAllVersions() returned 3 versions in order: v3, v2, v1")
		for i, v := range allVersions {
			t.Logf("  %d. Version %d: %s (is_latest: %t)", i+1, v.Version, v.ID, v.IsLatest)
		}
	})
}

// TestVersionManagement_DataCopying verifies framework data copying
func TestVersionManagement_DataCopying(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Close()
	defer helper.Cleanup(t)

	ctx := context.Background()

	t.Run("CreateNewVersion() copies all framework data", func(t *testing.T) {
		// Create submission and enrichment
		sub := helper.CreateTestSubmission(t, ctx)
		enr := helper.CreateTestEnrichment(t, ctx, sub.ID)
		enr.Finish()
		err := helper.EnrichmentRepo.UpdateSystem(ctx, enr)
		require.NoError(t, err)

		// Create version 1 with complete data
		v1Analysis := &analysis.Analysis{
			ID:           uuid.New().String(),
			SubmissionID: sub.ID.String(),
			EnrichmentID: enr.ID.String(),
			Status:       string(analysis.StatusCompleted),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Version:      1,
			IsLatest:     true,
		}

		// Populate all frameworks
		v1Analysis.PESTEL = analysis.PESTELAnalysis{Summary: "V1 PESTEL"}
		v1Analysis.Porter = analysis.PorterAnalysis{Summary: "V1 Porter"}
		v1Analysis.TamSamSom = analysis.TamSamSomAnalysis{Summary: "V1 TAM/SAM/SOM"}
		v1Analysis.SWOT = analysis.SWOTAnalysis{Summary: "V1 SWOT"}
		v1Analysis.BlueOcean = analysis.BlueOceanAnalysis{Summary: "V1 Blue Ocean"}
		v1Analysis.Benchmarking = analysis.BenchmarkingAnalysis{Summary: "V1 Benchmarking"}
		v1Analysis.GrowthHacking = analysis.GrowthHackingAnalysis{Summary: "V1 Growth Hacking"}
		v1Analysis.Scenarios = analysis.ScenarioAnalysis{Summary: "V1 Scenarios"}
		v1Analysis.OKRs = analysis.OKRAnalysis{Summary: "V1 OKRs"}
		v1Analysis.BSC = analysis.BalancedScorecardAnalysis{Summary: "V1 BSC"}
		v1Analysis.DecisionMatrix = analysis.DecisionMatrixAnalysis{Summary: "V1 Decision Matrix"}
		v1Analysis.Synthesis = analysis.AnalysisSynthesis{ExecutiveSummary: "V1 Synthesis"}

		err = helper.AnalysisRepo.Create(ctx, v1Analysis)
		require.NoError(t, err)

		// Create version 2
		v2Analysis := v1Analysis.CreateNewVersion()
		v2Analysis.ID = uuid.New().String()

		// Verify all framework data was copied
		assert.Equal(t, "V1 PESTEL", v2Analysis.PESTEL.Summary)
		assert.Equal(t, "V1 Porter", v2Analysis.Porter.Summary)
		assert.Equal(t, "V1 TAM/SAM/SOM", v2Analysis.TamSamSom.Summary)
		assert.Equal(t, "V1 SWOT", v2Analysis.SWOT.Summary)
		assert.Equal(t, "V1 Blue Ocean", v2Analysis.BlueOcean.Summary)
		assert.Equal(t, "V1 Benchmarking", v2Analysis.Benchmarking.Summary)
		assert.Equal(t, "V1 Growth Hacking", v2Analysis.GrowthHacking.Summary)
		assert.Equal(t, "V1 Scenarios", v2Analysis.Scenarios.Summary)
		assert.Equal(t, "V1 OKRs", v2Analysis.OKRs.Summary)
		assert.Equal(t, "V1 BSC", v2Analysis.BSC.Summary)
		assert.Equal(t, "V1 Decision Matrix", v2Analysis.DecisionMatrix.Summary)
		assert.Equal(t, "V1 Synthesis", v2Analysis.Synthesis.ExecutiveSummary)

		t.Log("✓ All 11 frameworks + synthesis copied from v1 to v2")

		// Modify v2 data
		v2Analysis.PESTEL.Summary = "V2 PESTEL (updated)"
		v2Analysis.Synthesis.ExecutiveSummary = "V2 Synthesis (updated)"

		v1Analysis.IsLatest = false
		err = helper.AnalysisRepo.Update(ctx, v1Analysis)
		require.NoError(t, err)

		err = helper.AnalysisRepo.Create(ctx, v2Analysis)
		require.NoError(t, err)

		// Verify v1 data unchanged
		retrievedV1, err := helper.AnalysisRepo.GetByID(ctx, v1Analysis.ID)
		require.NoError(t, err)
		assert.Equal(t, "V1 PESTEL", retrievedV1.PESTEL.Summary)
		assert.Equal(t, "V1 Synthesis", retrievedV1.Synthesis.ExecutiveSummary)

		// Verify v2 has updated data
		retrievedV2, err := helper.AnalysisRepo.GetByID(ctx, v2Analysis.ID)
		require.NoError(t, err)
		assert.Equal(t, "V2 PESTEL (updated)", retrievedV2.PESTEL.Summary)
		assert.Equal(t, "V2 Synthesis (updated)", retrievedV2.Synthesis.ExecutiveSummary)

		t.Log("✓ Version 2 modifications independent from version 1")
	})
}
