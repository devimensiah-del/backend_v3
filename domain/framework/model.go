package framework

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Category constants for framework organization
const (
	CategoryEnvironment = "environment" // External analysis: PESTEL, Porter, TAM-SAM-SOM
	CategoryPositioning = "positioning" // Internal/competitive: SWOT, Benchmarking
	CategoryStrategy    = "strategy"    // Strategic options: Blue Ocean, Growth Hacking, Scenarios
	CategoryExecution   = "execution"   // Implementation: OKRs, BSC, Decision Matrix
)

// Framework represents a strategic analysis framework definition
// This model enables dynamic framework execution by storing prompts,
// schemas, and metadata in the database instead of hardcoding them
type Framework struct {
	// Identity
	ID   uuid.UUID `db:"id" json:"id"`
	Code string    `db:"code" json:"code"` // Unique key: "pestel", "swot", "porter", etc.

	// Display Names
	Name   string `db:"name" json:"name"`     // English: "PESTEL Analysis"
	NamePT string `db:"name_pt" json:"name_pt"` // Portuguese: "Análise PESTEL"

	// Descriptions (nullable)
	Description   *string `db:"description" json:"description,omitempty"`
	DescriptionPT *string `db:"description_pt" json:"description_pt,omitempty"`

	// Organization
	Category   string `db:"category" json:"category"`       // CategoryEnvironment, etc.
	LayerOrder int    `db:"layer_order" json:"layer_order"` // Execution order within category (1, 2, 3...)

	// Behavior
	IsActive           bool `db:"is_active" json:"is_active"`                       // Whether framework is enabled
	RequiresEnrichment bool `db:"requires_enrichment" json:"requires_enrichment"` // If true, needs enrichment data

	// LLM Configuration
	TimeoutSeconds int     `db:"timeout_seconds" json:"timeout_seconds"` // Max execution time
	PromptTemplate string  `db:"prompt_template" json:"prompt_template"` // LLM prompt with {{variables}}
	OutputSchema   json.RawMessage `db:"output_schema" json:"output_schema"`     // JSON Schema for validation
	PreferredModel *string `db:"preferred_model" json:"preferred_model,omitempty"` // Optional model override
	Temperature    float64 `db:"temperature" json:"temperature"`                   // LLM temperature (0.0-1.0)

	// Dependencies
	DependsOn pq.StringArray `db:"depends_on" json:"depends_on"` // Framework codes that must run first

	// Timestamps
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// NewFramework creates a new framework with sensible defaults
func NewFramework(code, name, namePT, category string) *Framework {
	now := time.Now()
	return &Framework{
		ID:                 uuid.New(),
		Code:               code,
		Name:               name,
		NamePT:             namePT,
		Category:           category,
		LayerOrder:         1,
		IsActive:           true,
		RequiresEnrichment: true,
		TimeoutSeconds:     300, // 5 minutes default
		Temperature:        0.7,
		DependsOn:          pq.StringArray{},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// Validate validates the framework fields
func (f *Framework) Validate() error {
	if f.Code == "" {
		return errors.New("framework code is required")
	}
	if f.Name == "" {
		return errors.New("framework name is required")
	}
	if f.NamePT == "" {
		return errors.New("framework Portuguese name is required")
	}
	if !isValidCategory(f.Category) {
		return errors.New("invalid framework category")
	}
	if f.LayerOrder < 1 {
		return errors.New("layer order must be at least 1")
	}
	if f.TimeoutSeconds <= 0 {
		return errors.New("timeout seconds must be positive")
	}
	if f.Temperature < 0.0 || f.Temperature > 1.0 {
		return errors.New("temperature must be between 0.0 and 1.0")
	}
	if f.PromptTemplate == "" {
		return errors.New("prompt template is required")
	}
	if len(f.OutputSchema) == 0 {
		return errors.New("output schema is required")
	}
	// Validate JSON schema is valid JSON
	var schemaCheck interface{}
	if err := json.Unmarshal(f.OutputSchema, &schemaCheck); err != nil {
		return errors.New("output schema must be valid JSON")
	}
	return nil
}

// isValidCategory checks if a category is one of the allowed values
func isValidCategory(category string) bool {
	switch category {
	case CategoryEnvironment, CategoryPositioning, CategoryStrategy, CategoryExecution:
		return true
	default:
		return false
	}
}

// IsDependent returns true if this framework depends on another framework
func (f *Framework) IsDependent() bool {
	return len(f.DependsOn) > 0
}

// DependsOnCode checks if this framework depends on a specific framework code
func (f *Framework) DependsOnCode(code string) bool {
	for _, dep := range f.DependsOn {
		if dep == code {
			return true
		}
	}
	return false
}

// Activate enables the framework
func (f *Framework) Activate() {
	f.IsActive = true
	f.UpdatedAt = time.Now()
}

// Deactivate disables the framework (soft delete)
func (f *Framework) Deactivate() {
	f.IsActive = false
	f.UpdatedAt = time.Now()
}
