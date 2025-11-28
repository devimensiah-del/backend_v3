package macroeconomics

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// IndicatorType represents an economic indicator definition
type IndicatorType struct {
	ID        uuid.UUID       `db:"id" json:"id"`
	Code      string          `db:"code" json:"code"`
	Name      string          `db:"name" json:"name"`
	Category  string          `db:"category" json:"category"`
	Unit      *string         `db:"unit" json:"unit,omitempty"`
	Frequency string          `db:"frequency" json:"frequency"`
	IsActive  bool            `db:"is_active" json:"is_active"`
	Metadata  json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt time.Time       `db:"updated_at" json:"updated_at"`
}

// DataSource represents an API data source
type DataSource struct {
	ID              uuid.UUID `db:"id" json:"id"`
	Code            string    `db:"code" json:"code"`
	Name            string    `db:"name" json:"name"`
	BaseURL         *string   `db:"base_url" json:"base_url,omitempty"`
	Priority        int       `db:"priority" json:"priority"`
	IsAuthoritative bool      `db:"is_authoritative" json:"is_authoritative"`
	IsActive        bool      `db:"is_active" json:"is_active"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

// IndicatorSource represents the mapping between an indicator and a source
type IndicatorSource struct {
	ID                  uuid.UUID       `db:"id" json:"id"`
	IndicatorTypeID     uuid.UUID       `db:"indicator_type_id" json:"indicator_type_id"`
	SourceID            uuid.UUID       `db:"source_id" json:"source_id"`
	Priority            int             `db:"priority" json:"priority"`
	EndpointConfig      json.RawMessage `db:"endpoint_config" json:"endpoint_config"`
	IsActive            bool            `db:"is_active" json:"is_active"`
	LastSuccessAt       *time.Time      `db:"last_success_at" json:"last_success_at,omitempty"`
	ConsecutiveFailures int             `db:"consecutive_failures" json:"consecutive_failures"`
	CreatedAt           time.Time       `db:"created_at" json:"created_at"`
}

// IndicatorValue represents a single time-series data point
type IndicatorValue struct {
	ID              uuid.UUID       `db:"id" json:"id"`
	IndicatorTypeID uuid.UUID       `db:"indicator_type_id" json:"indicator_type_id"`
	SourceID        uuid.UUID       `db:"source_id" json:"source_id"`
	Value           float64         `db:"value" json:"value"`
	EffectiveDate   time.Time       `db:"effective_date" json:"effective_date"`
	ReferencePeriod *string         `db:"reference_period" json:"reference_period,omitempty"`
	RawResponse     json.RawMessage `db:"raw_response" json:"-"`
	FetchedAt       time.Time       `db:"fetched_at" json:"fetched_at"`
	CreatedAt       time.Time       `db:"created_at" json:"created_at"`
}

// FetchLog represents an audit log entry for a fetch operation
type FetchLog struct {
	ID                uuid.UUID  `db:"id" json:"id"`
	IndicatorSourceID *uuid.UUID `db:"indicator_source_id" json:"indicator_source_id,omitempty"`
	IndicatorCode     *string    `db:"indicator_code" json:"indicator_code,omitempty"`
	SourceCode        *string    `db:"source_code" json:"source_code,omitempty"`
	Status            string     `db:"status" json:"status"`
	RecordsInserted   int        `db:"records_inserted" json:"records_inserted"`
	RecordsUpdated    int        `db:"records_updated" json:"records_updated"`
	ErrorMessage      *string    `db:"error_message" json:"error_message,omitempty"`
	ResponseTimeMs    *int       `db:"response_time_ms" json:"response_time_ms,omitempty"`
	TriggeredBy       string     `db:"triggered_by" json:"triggered_by"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
}

// FetchLogStatus constants
const (
	FetchStatusSuccess = "success"
	FetchStatusFailed  = "failed"
	FetchStatusPartial = "partial"
)

// TriggerSource constants
const (
	TriggerScheduler = "scheduler"
	TriggerManual    = "manual"
	TriggerStartup   = "startup"
)

// LatestSnapshot represents the most recent values for ALL indicators (dynamic)
// Used for enrichment consumption - passes ALL indicators to LLM
type LatestSnapshot struct {
	Indicators map[string]*IndicatorValueSummary `json:"indicators"`
	AsOf       time.Time                          `json:"as_of"`
}

// IndicatorValueSummary is a rich version of IndicatorValue for snapshots
// Includes metadata so LLM can understand what each indicator represents
type IndicatorValueSummary struct {
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	Category        string    `json:"category"`
	Value           float64   `json:"value"`
	Unit            string    `json:"unit"`
	EffectiveDate   time.Time `json:"effective_date"`
	ReferencePeriod *string   `json:"reference_period,omitempty"`
	SourceCode      string    `json:"source_code"`
	FetchedAt       time.Time `json:"fetched_at"`
}

// HasData returns true if at least one indicator has data
func (s *LatestSnapshot) HasData() bool {
	return len(s.Indicators) > 0
}

// Get returns a specific indicator by code (for backwards compatibility)
func (s *LatestSnapshot) Get(code string) *IndicatorValueSummary {
	if s.Indicators == nil {
		return nil
	}
	return s.Indicators[code]
}

// IndicatorWithLatest combines indicator type with its latest value
type IndicatorWithLatest struct {
	Indicator   *IndicatorType         `json:"indicator"`
	LatestValue *IndicatorValueSummary `json:"latest_value,omitempty"`
}

// FetchJobPayload is the payload for macro fetch jobs
type FetchJobPayload struct {
	Code string `json:"code"`
}

// IndicatorHistory represents time-series data for trend analysis
type IndicatorHistory struct {
	IndicatorCode string            `json:"indicator_code"`
	IndicatorName string            `json:"indicator_name"`
	Unit          *string           `json:"unit,omitempty"`
	Values        []*IndicatorValue `json:"values"`
	From          time.Time         `json:"from"`
	To            time.Time         `json:"to"`
}
