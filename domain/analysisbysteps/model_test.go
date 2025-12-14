package analysisbysteps_test

import (
	"testing"
	"time"

	"backend_v3/domain/analysisbysteps"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// TEST: AnalysisStep Model Methods
// =============================================================================

func TestAnalysisStep_GetEffectiveOutput(t *testing.T) {
	aiOutput := `{"ai": "output"}`
	humanEdited := `{"human": "edited"}`

	t.Run("returns human_edited when available", func(t *testing.T) {
		step := &analysisbysteps.AnalysisStep{
			AIOutput:    &aiOutput,
			HumanEdited: &humanEdited,
		}

		effective := step.GetEffectiveOutput()
		assert.NotNil(t, effective)
		assert.Equal(t, humanEdited, *effective)
	})

	t.Run("returns ai_output when no human edit", func(t *testing.T) {
		step := &analysisbysteps.AnalysisStep{
			AIOutput:    &aiOutput,
			HumanEdited: nil,
		}

		effective := step.GetEffectiveOutput()
		assert.NotNil(t, effective)
		assert.Equal(t, aiOutput, *effective)
	})

	t.Run("returns nil when no content", func(t *testing.T) {
		step := &analysisbysteps.AnalysisStep{
			AIOutput:    nil,
			HumanEdited: nil,
		}

		effective := step.GetEffectiveOutput()
		assert.Nil(t, effective)
	})
}

func TestAnalysisStep_IsEdited(t *testing.T) {
	humanEdited := `{"test": "data"}`

	t.Run("returns true when human edited exists", func(t *testing.T) {
		step := &analysisbysteps.AnalysisStep{
			HumanEdited: &humanEdited,
		}
		assert.True(t, step.IsEdited())
	})

	t.Run("returns false when no human edit", func(t *testing.T) {
		step := &analysisbysteps.AnalysisStep{
			HumanEdited: nil,
		}
		assert.False(t, step.IsEdited())
	})
}

func TestAnalysisStep_IsApproved(t *testing.T) {
	t.Run("returns true when status is approved", func(t *testing.T) {
		step := &analysisbysteps.AnalysisStep{
			Status: analysisbysteps.StatusApproved,
		}
		assert.True(t, step.IsApproved())
	})

	t.Run("returns false when status is not approved", func(t *testing.T) {
		statuses := []analysisbysteps.StepStatus{
			analysisbysteps.StatusPending,
			analysisbysteps.StatusGenerating,
			analysisbysteps.StatusGenerated,
			analysisbysteps.StatusFailed,
		}

		for _, status := range statuses {
			step := &analysisbysteps.AnalysisStep{
				Status: status,
			}
			assert.False(t, step.IsApproved(), "status %s should not be approved", status)
		}
	})
}

// =============================================================================
// TEST: Constants and Framework Metadata
// =============================================================================

func TestFrameworkOrder(t *testing.T) {
	t.Run("has correct number of frameworks", func(t *testing.T) {
		assert.Equal(t, 14, analysisbysteps.TotalSteps())
		assert.Len(t, analysisbysteps.FrameworkOrder, 14)
	})

	t.Run("starts with challenge_refinement", func(t *testing.T) {
		assert.Equal(t, "challenge_refinement", analysisbysteps.FrameworkOrder[0].Code)
		assert.Equal(t, "Refinamento do Desafio", analysisbysteps.FrameworkOrder[0].Name)
	})

	t.Run("ends with synthesis", func(t *testing.T) {
		lastIndex := analysisbysteps.TotalSteps() - 1
		assert.Equal(t, "synthesis", analysisbysteps.FrameworkOrder[lastIndex].Code)
		assert.Equal(t, "Síntese Executiva", analysisbysteps.FrameworkOrder[lastIndex].Name)
	})

	t.Run("all frameworks have required fields", func(t *testing.T) {
		for i, meta := range analysisbysteps.FrameworkOrder {
			assert.NotEmpty(t, meta.Code, "framework %d missing code", i)
			assert.NotEmpty(t, meta.Name, "framework %d missing name", i)
			assert.NotEmpty(t, meta.GuidanceText, "framework %d missing guidance text", i)
		}
	})
}

func TestGetStepNumber(t *testing.T) {
	t.Run("returns correct step number for valid framework", func(t *testing.T) {
		assert.Equal(t, 0, analysisbysteps.GetStepNumber("challenge_refinement"))
		assert.Equal(t, 1, analysisbysteps.GetStepNumber("pestel"))
		assert.Equal(t, 13, analysisbysteps.GetStepNumber("synthesis"))
	})

	t.Run("returns -1 for invalid framework code", func(t *testing.T) {
		assert.Equal(t, -1, analysisbysteps.GetStepNumber("invalid_code"))
		assert.Equal(t, -1, analysisbysteps.GetStepNumber(""))
	})
}

func TestGetFrameworkMeta(t *testing.T) {
	t.Run("returns metadata for valid framework", func(t *testing.T) {
		meta := analysisbysteps.GetFrameworkMeta("pestel")
		assert.NotNil(t, meta)
		assert.Equal(t, "pestel", meta.Code)
		assert.Equal(t, "Análise PESTEL", meta.Name)
		assert.NotEmpty(t, meta.GuidanceText)
	})

	t.Run("returns nil for invalid framework code", func(t *testing.T) {
		meta := analysisbysteps.GetFrameworkMeta("invalid_code")
		assert.Nil(t, meta)
	})
}

// =============================================================================
// TEST: Step Status Values
// =============================================================================

func TestStepStatus_Values(t *testing.T) {
	statuses := []analysisbysteps.StepStatus{
		analysisbysteps.StatusPending,
		analysisbysteps.StatusGenerating,
		analysisbysteps.StatusGenerated,
		analysisbysteps.StatusApproved,
		analysisbysteps.StatusFailed,
	}

	expectedValues := []string{
		"pending",
		"generating",
		"generated",
		"approved",
		"failed",
	}

	for i, status := range statuses {
		assert.Equal(t, expectedValues[i], string(status))
	}
}

// =============================================================================
// TEST: Response Types
// =============================================================================

func TestStartResponse(t *testing.T) {
	response := &analysisbysteps.StartResponse{
		AnalysisID:  "test-analysis-id",
		ChallengeID: "test-challenge-id",
		TotalSteps:  14,
		CurrentStep: 0,
		Steps:       []analysisbysteps.AnalysisStep{},
	}

	assert.Equal(t, "test-analysis-id", response.AnalysisID)
	assert.Equal(t, "test-challenge-id", response.ChallengeID)
	assert.Equal(t, 14, response.TotalSteps)
	assert.Equal(t, 0, response.CurrentStep)
	assert.NotNil(t, response.Steps)
}

func TestApproveResponse(t *testing.T) {
	approvedStep := &analysisbysteps.AnalysisStep{
		Status:     analysisbysteps.StatusApproved,
		ApprovedAt: ptrTime(time.Now()),
	}

	nextStep := &analysisbysteps.AnalysisStep{
		Status:     analysisbysteps.StatusPending,
		StepNumber: 1,
	}

	response := &analysisbysteps.ApproveResponse{
		ApprovedStep: approvedStep,
		NextStep:     nextStep,
		IsComplete:   false,
		CurrentStep:  1,
	}

	assert.NotNil(t, response.ApprovedStep)
	assert.NotNil(t, response.NextStep)
	assert.False(t, response.IsComplete)
	assert.Equal(t, 1, response.CurrentStep)
}

func TestStepStateResponse(t *testing.T) {
	currentStep := &analysisbysteps.AnalysisStep{
		StepNumber: 5,
		Status:     analysisbysteps.StatusGenerated,
	}

	previousSteps := []analysisbysteps.AnalysisStep{
		{StepNumber: 0, Status: analysisbysteps.StatusApproved},
		{StepNumber: 1, Status: analysisbysteps.StatusApproved},
	}

	meta := analysisbysteps.GetFrameworkMeta("blue_ocean")

	response := &analysisbysteps.StepStateResponse{
		AnalysisID:      "test-id",
		CurrentStep:     5,
		TotalSteps:      14,
		CurrentStepData: currentStep,
		PreviousSteps:   previousSteps,
		FrameworkMeta:   meta,
	}

	assert.Equal(t, "test-id", response.AnalysisID)
	assert.Equal(t, 5, response.CurrentStep)
	assert.Equal(t, 14, response.TotalSteps)
	assert.NotNil(t, response.CurrentStepData)
	assert.Len(t, response.PreviousSteps, 2)
	assert.NotNil(t, response.FrameworkMeta)
	assert.Equal(t, "blue_ocean", response.FrameworkMeta.Code)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func ptrTime(t time.Time) *time.Time {
	return &t
}
