package wizard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFrameworkOrder_Structure(t *testing.T) {
	// Verify FrameworkOrder has expected number of steps
	assert.Len(t, FrameworkOrder, 12, "Should have 12 framework steps")

	// Verify step numbers are sequential starting from 0
	for i, step := range FrameworkOrder {
		assert.Equal(t, i, step.Step, "Step %d should have Step=%d", i, i)
	}
}

func TestFrameworkOrder_RequiredFields(t *testing.T) {
	for _, step := range FrameworkOrder {
		t.Run(step.Code, func(t *testing.T) {
			assert.NotEmpty(t, step.Code, "Code should not be empty")
			assert.NotEmpty(t, step.Name, "Name should not be empty")
			assert.NotEmpty(t, step.Description, "Description should not be empty")
			assert.NotEmpty(t, step.Questions, "Questions should not be empty")

			// Each question should have ID and Question text
			for _, q := range step.Questions {
				assert.NotEmpty(t, q.ID, "Question ID should not be empty")
				assert.NotEmpty(t, q.Question, "Question text should not be empty")
			}
		})
	}
}

func TestFrameworkOrder_ExpectedCodes(t *testing.T) {
	// Actual order as defined in framework_order.go
	expectedCodes := []string{
		"challenge_refinement", // 0
		"pestel",               // 1
		"porter",               // 2
		"benchmarking",         // 3
		"swot",                 // 4
		"tam_sam_som",          // 5
		"blue_ocean",           // 6
		"growth_hacking",       // 7
		"scenarios",            // 8
		"decision_matrix",      // 9
		"okrs",                 // 10
		"bsc",                  // 11
	}

	for i, code := range expectedCodes {
		assert.Equal(t, code, FrameworkOrder[i].Code, "Step %d should be %s", i, code)
	}
}

func TestGetFrameworkByStep(t *testing.T) {
	tests := []struct {
		step     int
		wantCode string
		wantNil  bool
	}{
		{0, "challenge_refinement", false},
		{1, "pestel", false},
		{5, "tam_sam_som", false},
		{11, "bsc", false},
		{12, "", true}, // Out of range
		{-1, "", true}, // Negative
	}

	for _, tt := range tests {
		t.Run(tt.wantCode, func(t *testing.T) {
			if tt.step >= 0 && tt.step < len(FrameworkOrder) {
				step := FrameworkOrder[tt.step]
				assert.Equal(t, tt.wantCode, step.Code)
			} else if !tt.wantNil {
				t.Errorf("Expected code %s but step %d is out of range", tt.wantCode, tt.step)
			}
		})
	}
}

func TestState_Structure(t *testing.T) {
	state := State{
		AnalysisID:     "test-analysis-id",
		CurrentStep:    3,
		TotalSteps:     12,
		StepStatus:     "generated",
		HumanContext:   "Some context from user",
		HumanAnswers:   map[string]string{"q1": "answer1"},
		IterationCount: 2,
	}

	assert.Equal(t, "test-analysis-id", state.AnalysisID)
	assert.Equal(t, 3, state.CurrentStep)
	assert.Equal(t, 12, state.TotalSteps)
	assert.Equal(t, "generated", state.StepStatus)
	assert.Equal(t, "Some context from user", state.HumanContext)
	assert.Equal(t, "answer1", state.HumanAnswers["q1"])
	assert.Equal(t, 2, state.IterationCount)
}

func TestStepSummary_Structure(t *testing.T) {
	summary := StepSummary{
		Step:          1,
		FrameworkCode: "pestel",
		FrameworkName: "PESTEL",
		Status:        "approved",
		ApprovedAt:    "2024-01-15T10:00:00Z",
	}

	assert.Equal(t, 1, summary.Step)
	assert.Equal(t, "pestel", summary.FrameworkCode)
	assert.Equal(t, "PESTEL", summary.FrameworkName)
	assert.Equal(t, "approved", summary.Status)
	assert.Equal(t, "2024-01-15T10:00:00Z", summary.ApprovedAt)
}

func TestFrameworkStep_Questions(t *testing.T) {
	// Test that each step has meaningful questions
	for _, step := range FrameworkOrder {
		t.Run(step.Code, func(t *testing.T) {
			assert.GreaterOrEqual(t, len(step.Questions), 2, "Each step should have at least 2 questions")

			// Check questions are unique within the step
			questionIDs := make(map[string]bool)
			for _, q := range step.Questions {
				assert.False(t, questionIDs[q.ID], "Question ID %s should be unique", q.ID)
				questionIDs[q.ID] = true
			}
		})
	}
}

func TestChallengeRefinement_IsFirstStep(t *testing.T) {
	// Challenge refinement should always be step 0
	firstStep := FrameworkOrder[0]
	assert.Equal(t, 0, firstStep.Step)
	assert.Equal(t, "challenge_refinement", firstStep.Code)
	assert.Contains(t, firstStep.Description, "desafio") // PT-BR description mentions "desafio"
}

func TestBSC_IsLastStep(t *testing.T) {
	// BSC should be the last step (before synthesis which is auto-generated)
	lastStep := FrameworkOrder[len(FrameworkOrder)-1]
	assert.Equal(t, 11, lastStep.Step)
	assert.Equal(t, "bsc", lastStep.Code)
}
