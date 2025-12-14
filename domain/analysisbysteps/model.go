package analysisbysteps

import "time"

// StepStatus represents the status of an analysis step
type StepStatus string

const (
	StatusPending    StepStatus = "pending"
	StatusGenerating StepStatus = "generating"
	StatusGenerated  StepStatus = "generated"
	StatusApproved   StepStatus = "approved"
	StatusFailed     StepStatus = "failed"
)

// AnalysisStep represents a single step in the human-in-the-loop analysis flow
type AnalysisStep struct {
	ID            string     `db:"id" json:"id"`
	AnalysisID    string     `db:"analysis_id" json:"analysis_id"`
	FrameworkCode string     `db:"framework_code" json:"framework_code"`
	StepNumber    int        `db:"step_number" json:"step_number"`
	AIOutput      *string    `db:"ai_output" json:"ai_output"`
	HumanEdited   *string    `db:"human_edited" json:"human_edited"`
	Visible       bool       `db:"visible" json:"visible"`
	Status        StepStatus `db:"status" json:"status"`
	GeneratedAt   *time.Time `db:"generated_at" json:"generated_at"`
	ApprovedAt    *time.Time `db:"approved_at" json:"approved_at"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
}

// GetEffectiveOutput returns the human-edited content if available, otherwise AI output
// Mirrors SQL: COALESCE(human_edited, ai_output)
func (s *AnalysisStep) GetEffectiveOutput() *string {
	if s.HumanEdited != nil {
		return s.HumanEdited
	}
	return s.AIOutput
}

// IsEdited returns true if this step has been edited by a human
func (s *AnalysisStep) IsEdited() bool {
	return s.HumanEdited != nil
}

// IsApproved returns true if this step has been approved
func (s *AnalysisStep) IsApproved() bool {
	return s.Status == StatusApproved
}
