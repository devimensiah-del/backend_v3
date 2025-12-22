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

	// Prompt configuration
	PromptSystem       *string `json:"prompt_system,omitempty" db:"prompt_system"`
	PromptUser         string  `json:"prompt_user" db:"prompt_user"`
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
