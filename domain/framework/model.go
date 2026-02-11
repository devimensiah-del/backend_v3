package framework

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Framework represents a strategic analysis framework configuration
type Framework struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	Code        string      `json:"code" db:"code"`
	Name        string      `json:"name" db:"name"`
	Description *string     `json:"description,omitempty" db:"description"`
	Layer       int         `json:"layer" db:"layer"`
	IsBase      bool        `json:"is_base" db:"is_base"`
	IsActive    bool        `json:"is_active" db:"is_active"`

	// Step organization (5-step flow)
	StepNumber     *int    `json:"step_number,omitempty" db:"step_number"`
	StepName       *string `json:"step_name,omitempty" db:"step_name"`
	SubtabOf       *string `json:"subtab_of,omitempty" db:"subtab_of"`
	ExecutionOrder int     `json:"execution_order" db:"execution_order"`

	// Prompt configuration
	PromptSystem       *string `json:"prompt_system,omitempty" db:"prompt_system"`
	PromptUser         string  `json:"prompt_user" db:"prompt_user"`
	PromptUpdate       *string `json:"prompt_update,omitempty" db:"prompt_update"`
	PromptJSONTemplate *string `json:"prompt_json_template,omitempty" db:"prompt_json_template"`

	// Model configuration
	ModelConfig ModelConfig `json:"model_config" db:"model_config"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ModelConfig holds LLM configuration for a framework
type ModelConfig struct {
	Model         string  `json:"model"`
	Temperature   float64 `json:"temperature"`
	MaxTokens     int     `json:"max_tokens"`
	FallbackModel string  `json:"fallback_model"`
}

// Scan implements sql.Scanner for JSONB
func (m *ModelConfig) Scan(value interface{}) error {
	if value == nil {
		*m = ModelConfig{}
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("type assertion failed for ModelConfig")
	}
	return json.Unmarshal(b, m)
}

// Value implements driver.Valuer for JSONB
func (m ModelConfig) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// FrameworkDependency represents a dependency between frameworks
type FrameworkDependency struct {
	ID             uuid.UUID `json:"id" db:"id"`
	FrameworkID    uuid.UUID `json:"framework_id" db:"framework_id"`
	DependsOnID    uuid.UUID `json:"depends_on_id" db:"depends_on_id"`
	DependencyType string    `json:"dependency_type" db:"dependency_type"` // required, optional, enhances
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// CompanyFrameworkResult stores a framework result for a specific company
type CompanyFrameworkResult struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	CompanyID   uuid.UUID       `json:"company_id" db:"company_id"`
	FrameworkID uuid.UUID       `json:"framework_id" db:"framework_id"`
	ChallengeID *uuid.UUID      `json:"challenge_id,omitempty" db:"challenge_id"`
	Result      json.RawMessage `json:"result,omitempty" db:"result"`
	Status      string          `json:"status" db:"status"` // pending, processing, completed, failed
	ErrorMessage *string        `json:"error_message,omitempty" db:"error_message"`
	Version     int             `json:"version" db:"version"`
	IsCurrent   bool            `json:"is_current" db:"is_current"`
	ContextHash *string         `json:"context_hash,omitempty" db:"context_hash"`
	GeneratedAt *time.Time      `json:"generated_at,omitempty" db:"generated_at"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`

	// Stale tracking (for dependency cascade)
	IsStale              bool            `json:"is_stale" db:"is_stale"`
	StaleReason          *string         `json:"stale_reason,omitempty" db:"stale_reason"`
	StaleDetectedAt      *time.Time      `json:"stale_detected_at,omitempty" db:"stale_detected_at"`
	StaleAcknowledgedAt  *time.Time      `json:"stale_acknowledged_at,omitempty" db:"stale_acknowledged_at"`
	StaleAcknowledgedBy  *uuid.UUID      `json:"stale_acknowledged_by,omitempty" db:"stale_acknowledged_by"`
	PreviousResult       *json.RawMessage `json:"previous_result,omitempty" db:"previous_result"`
	PreviousVersion      *int            `json:"previous_version,omitempty" db:"previous_version"`
	UpstreamDataHash     *string         `json:"upstream_data_hash,omitempty" db:"upstream_data_hash"`
}

// Status constants for CompanyFrameworkResult
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// Errors
var (
	ErrResultNotFound = errors.New("framework result not found")
)

// DependencyType constants
const (
	DependencyRequired = "required"
	DependencyOptional = "optional"
	DependencyEnhances = "enhances"
)

// ExecutionPlan represents the resolved execution order for frameworks
type ExecutionPlan struct {
	Layers []ExecutionLayer `json:"layers"`
}

// ExecutionLayer contains frameworks that can run in parallel
type ExecutionLayer struct {
	LayerNumber int          `json:"layer_number"`
	Frameworks  []*Framework `json:"frameworks"`
}

// FrameworkWithDependencies includes the framework and its direct dependencies
type FrameworkWithDependencies struct {
	Framework    *Framework   `json:"framework"`
	Dependencies []*Framework `json:"dependencies"`
}

// Validate validates the framework
func (f *Framework) Validate() error {
	if f.Code == "" {
		return errors.New("framework code is required")
	}
	if f.Name == "" {
		return errors.New("framework name is required")
	}
	if f.PromptUser == "" {
		return errors.New("framework prompt_user is required")
	}
	if f.Layer < 1 || f.Layer > 10 {
		return errors.New("framework layer must be between 1 and 10")
	}
	return nil
}

// Validate validates the company framework result
func (r *CompanyFrameworkResult) Validate() error {
	if r.CompanyID == uuid.Nil {
		return errors.New("company_id is required")
	}
	if r.FrameworkID == uuid.Nil {
		return errors.New("framework_id is required")
	}
	if r.Status != StatusPending && r.Status != StatusProcessing && r.Status != StatusCompleted && r.Status != StatusFailed {
		return errors.New("invalid status")
	}
	return nil
}

// IsCompleted returns true if the result has completed successfully
func (r *CompanyFrameworkResult) IsCompleted() bool {
	return r.Status == StatusCompleted
}

// IsFailed returns true if the result has failed
func (r *CompanyFrameworkResult) IsFailed() bool {
	return r.Status == StatusFailed
}

// IsPending returns true if the result is pending
func (r *CompanyFrameworkResult) IsPending() bool {
	return r.Status == StatusPending
}

// IsProcessing returns true if the result is processing
func (r *CompanyFrameworkResult) IsProcessing() bool {
	return r.Status == StatusProcessing
}

// NeedsAttention returns true if the result is stale and not acknowledged
func (r *CompanyFrameworkResult) NeedsAttention() bool {
	return r.IsStale && r.StaleAcknowledgedAt == nil
}

// =============================================================================
// Step Organization Types
// =============================================================================

// StepDefinition represents a main step in the 5-step flow
type StepDefinition struct {
	Number     int                `json:"number"`
	Name       string             `json:"name"`
	Frameworks []*FrameworkStatus `json:"frameworks"`
}

// FrameworkStatus represents the current status of a framework for a company
type FrameworkStatus struct {
	Code                string   `json:"code"`
	Name                string   `json:"name"`
	Status              string   `json:"status"`
	IsStale             bool     `json:"is_stale"`
	StaleReason         *string  `json:"stale_reason,omitempty"`
	StaleAcknowledgedAt *string  `json:"stale_acknowledged_at,omitempty"`
	Version             int      `json:"version"`
	HasResult           bool     `json:"has_result"`
	CanExecute          bool     `json:"can_execute"`
	MissingDependencies []string `json:"missing_dependencies,omitempty"`
	SubtabOf            *string  `json:"subtab_of,omitempty"`
	ExecutionOrder      int      `json:"execution_order"`
}

// AnalysisState represents the complete analysis state for a company
type AnalysisState struct {
	CompanyID       uuid.UUID               `json:"company_id"`
	Steps           []*StepDefinition       `json:"steps"`
	Frameworks      map[string]*FrameworkStatus `json:"frameworks"`
	OverallProgress int                     `json:"overall_progress"`
	StaleCount      int                     `json:"stale_count"`
}

// =============================================================================
// Update Suggestion Types
// =============================================================================

// UpdateSuggestion represents AI-suggested changes after re-running a framework
type UpdateSuggestion struct {
	FrameworkCode   string            `json:"framework_code"`
	CurrentVersion  int               `json:"current_version"`
	SuggestedResult json.RawMessage   `json:"suggested_result"`
	ChangesSummary  string            `json:"changes_summary"`
	FieldChanges    []FieldChange     `json:"field_changes"`
	UpstreamChanges []string          `json:"upstream_changes"`
}

// FieldChange represents a specific field that changed
type FieldChange struct {
	FieldPath    string `json:"field_path"`
	OldValue     string `json:"old_value"`
	NewValue     string `json:"new_value"`
	ChangeReason string `json:"change_reason"`
}

// StaleFramework represents a stale framework for API response
type StaleFramework struct {
	FrameworkCode       string     `json:"framework_code"`
	FrameworkName       string     `json:"framework_name"`
	StaleReason         string     `json:"stale_reason"`
	StaleDetectedAt     time.Time  `json:"stale_detected_at"`
	StaleAcknowledgedAt *time.Time `json:"stale_acknowledged_at,omitempty"`
	ResultID            uuid.UUID  `json:"result_id"`
}

// ReadinessStatus indicates if a framework is ready to execute
type ReadinessStatus struct {
	Ready               bool     `json:"ready"`
	MissingDependencies []string `json:"missing_dependencies,omitempty"`
}
