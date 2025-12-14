package analysisbysteps

// StartResponse contains the result of starting a new step-by-step analysis
type StartResponse struct {
	AnalysisID  string         `json:"analysis_id"`
	ChallengeID string         `json:"challenge_id"`
	TotalSteps  int            `json:"total_steps"`
	CurrentStep int            `json:"current_step"`
	Steps       []AnalysisStep `json:"steps"`
}

// ApproveResponse contains the result of approving a step
type ApproveResponse struct {
	ApprovedStep *AnalysisStep `json:"approved_step"`
	NextStep     *AnalysisStep `json:"next_step,omitempty"`
	IsComplete   bool          `json:"is_complete"`
	CurrentStep  int           `json:"current_step"`
}

// StepStateResponse provides the current step state with all previous steps as context
type StepStateResponse struct {
	AnalysisID      string         `json:"analysis_id"`
	CurrentStep     int            `json:"current_step"`
	TotalSteps      int            `json:"total_steps"`
	CurrentStepData *AnalysisStep  `json:"current_step_data"`
	PreviousSteps   []AnalysisStep `json:"previous_steps"` // Read-only context
	FrameworkMeta   *FrameworkMeta `json:"framework_meta"`
}
