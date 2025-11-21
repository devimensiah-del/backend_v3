package enrichment

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// JSONMap handles the JSONB interaction with Postgres
type JSONMap map[string]interface{}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = make(JSONMap)
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, m)
}

// =================================================================
// NEW: Agentic Data Structures
// These are not DB tables, but structs to enforce the AI's JSON output
// =================================================================

type StrategicProfile struct {
	Overview CompanyOverview    `json:"overview"`
	Strategy StrategicInference `json:"strategicInference"`
	Market   MarketPosition     `json:"marketPosition"`
	Digital  DigitalPresence    `json:"digitalPresence"`
}

type CompanyOverview struct {
	Description   string   `json:"description"`
	Founded       string   `json:"founded,omitempty"`
	Headquarters  string   `json:"headquarters,omitempty"`
	EmployeeCount string   `json:"employeeCount,omitempty"`
	RevenueEst    string   `json:"revenueEstimation,omitempty"`
	Sources       []string `json:"sources"`
}

type StrategicInference struct {
	ValueArchetype  string   `json:"valuePropositionArchetype"` // e.g. "High Touch/Premium"
	TargetSegment   string   `json:"targetCustomerSegment"`     // e.g. "B2B Enterprise"
	BrandTone       string   `json:"brandTone"`                 // e.g. "Formal"
	DigitalMaturity int      `json:"digitalMaturityScore"`      // 1-10
	Weaknesses      []string `json:"observedWeaknesses"`        // Inferred operational gaps
}

type MarketPosition struct {
	Industry       string   `json:"industry"`
	Competitors    []string `json:"competitors"`
	Differentiator string   `json:"keyDifferentiator"`
}

type DigitalPresence struct {
	WebsiteURL  string   `json:"websiteUrl"`
	LinkedInURL string   `json:"linkedinUrl,omitempty"`
	RecentNews  []string `json:"recentNews"`
}

// =================================================================
// Database Entity (Unchanged from Schema 004)
// =================================================================

type Enrichment struct {
	ID           uuid.UUID `json:"id" db:"id"`
	SubmissionID uuid.UUID `json:"submissionId" db:"submission_id"`

	Status      Status `json:"status" db:"status"`
	Progress    int    `json:"progress" db:"progress"`
	CurrentStep string `json:"currentStep" db:"current_step"`

	IsLocked bool `json:"isLocked" db:"is_locked"`

	// We map the StrategicProfile into this JSONMap for storage
	SourcesStatus JSONMap `json:"sourcesStatus,omitempty" db:"sources_status"`
	EnrichedData  JSONMap `json:"enrichedData,omitempty" db:"enriched_data"`

	StartedAt    *time.Time `json:"startedAt,omitempty" db:"started_at"`
	CompletedAt  *time.Time `json:"completedAt,omitempty" db:"completed_at"`
	ErrorMessage string     `json:"errorMessage,omitempty" db:"error_message"`
	RetryCount   int        `json:"retryCount" db:"retry_count"`
	MaxRetries   int        `json:"maxRetries" db:"max_retries"`

	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

// Constructors and Methods (Start, UpdateProgress, etc.) remain identical...
// (Omitting strictly for brevity, paste previous standard methods here)
func NewEnrichment(submissionID uuid.UUID) *Enrichment {
	now := time.Now()
	return &Enrichment{
		ID:            uuid.New(),
		SubmissionID:  submissionID,
		Status:        StatusPending,
		Progress:      0,
		CurrentStep:   "Queued for enrichment",
		IsLocked:      false,
		SourcesStatus: make(JSONMap),
		EnrichedData:  make(JSONMap),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// ... Add Start(), UpdateProgress(), Complete(), Fail() from previous chat ...
func (e *Enrichment) Start() {
	e.Status = StatusProcessing
	n := time.Now()
	e.StartedAt = &n
	e.UpdatedAt = n
}
func (e *Enrichment) Complete() {
	e.Status = StatusCompleted
	n := time.Now()
	e.CompletedAt = &n
	e.Progress = 100
}
func (e *Enrichment) Fail(err error)                      { e.Status = StatusFailed; e.ErrorMessage = err.Error() }
func (e *Enrichment) UpdateProgress(step string, pct int) { e.CurrentStep = step; e.Progress = pct }
