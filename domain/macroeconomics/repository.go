package macroeconomics

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Repository defines the interface for macroeconomics data access
type Repository interface {
	// Indicator Types
	GetIndicatorByCode(ctx context.Context, code string) (*IndicatorType, error)
	GetActiveIndicators(ctx context.Context) ([]*IndicatorType, error)

	// Data Sources
	GetSourceByCode(ctx context.Context, code string) (*DataSource, error)
	GetActiveSources(ctx context.Context) ([]*DataSource, error)

	// Indicator-Source Mapping
	GetSourcesForIndicator(ctx context.Context, indicatorCode string) ([]*IndicatorSourceWithDetails, error)
	UpdateSourceStatus(ctx context.Context, sourceID uuid.UUID, success bool) error

	// Indicator Values (Time-Series)
	GetLatestValue(ctx context.Context, indicatorCode string) (*IndicatorValueWithSource, error)
	GetLatestValues(ctx context.Context, codes []string) (map[string]*IndicatorValueWithSource, error)
	GetAllLatestValues(ctx context.Context) ([]*IndicatorValueWithDetails, error)
	InsertValue(ctx context.Context, value *IndicatorValue) error
	GetHistory(ctx context.Context, indicatorCode string, from, to time.Time, limit int) ([]*IndicatorValue, error)

	// Fetch Logs
	LogFetch(ctx context.Context, log *FetchLog) error
	GetRecentLogs(ctx context.Context, indicatorCode string, limit int) ([]*FetchLog, error)
}

// IndicatorSourceWithDetails combines source with indicator details
type IndicatorSourceWithDetails struct {
	IndicatorSource
	IndicatorCode string `db:"indicator_code"`
	IndicatorName string `db:"indicator_name"`
	SourceCode    string `db:"source_code"`
	SourceName    string `db:"source_name"`
	SourceBaseURL string `db:"source_base_url"`
}

// IndicatorValueWithSource combines value with source details
type IndicatorValueWithSource struct {
	IndicatorValue
	IndicatorCode string `db:"indicator_code"`
	IndicatorName string `db:"indicator_name"`
	SourceCode    string `db:"source_code"`
}

// IndicatorValueWithDetails includes full indicator metadata for enrichment
type IndicatorValueWithDetails struct {
	ID              uuid.UUID `db:"id"`
	Code            string    `db:"code"`
	Name            string    `db:"name"`
	Category        string    `db:"category"`
	Unit            string    `db:"unit"`
	Value           float64   `db:"value"`
	EffectiveDate   time.Time `db:"effective_date"`
	ReferencePeriod *string   `db:"reference_period"`
	SourceCode      string    `db:"source_code"`
	FetchedAt       time.Time `db:"fetched_at"`
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new PostgreSQL repository
func NewRepository(db *sqlx.DB) Repository {
	return &PostgresRepository{db: db}
}

// ==================== INDICATOR TYPES ====================

// GetIndicatorByCode retrieves an indicator type by its code
func (r *PostgresRepository) GetIndicatorByCode(ctx context.Context, code string) (*IndicatorType, error) {
	query := `
		SELECT id, code, name, category, unit, frequency, is_active, metadata, created_at, updated_at
		FROM macro_indicator_types
		WHERE code = $1
	`

	var indicator IndicatorType
	if err := r.db.GetContext(ctx, &indicator, query, code); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("indicator not found: %s", code)
		}
		return nil, fmt.Errorf("failed to get indicator: %w", err)
	}

	return &indicator, nil
}

// GetActiveIndicators retrieves all active indicator types
func (r *PostgresRepository) GetActiveIndicators(ctx context.Context) ([]*IndicatorType, error) {
	query := `
		SELECT id, code, name, category, unit, frequency, is_active, metadata, created_at, updated_at
		FROM macro_indicator_types
		WHERE is_active = true
		ORDER BY code
	`

	var indicators []*IndicatorType
	if err := r.db.SelectContext(ctx, &indicators, query); err != nil {
		return nil, fmt.Errorf("failed to get active indicators: %w", err)
	}

	return indicators, nil
}

// ==================== DATA SOURCES ====================

// GetSourceByCode retrieves a data source by its code
func (r *PostgresRepository) GetSourceByCode(ctx context.Context, code string) (*DataSource, error) {
	query := `
		SELECT id, code, name, base_url, priority, is_authoritative, is_active, created_at
		FROM macro_data_sources
		WHERE code = $1
	`

	var source DataSource
	if err := r.db.GetContext(ctx, &source, query, code); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("source not found: %s", code)
		}
		return nil, fmt.Errorf("failed to get source: %w", err)
	}

	return &source, nil
}

// GetActiveSources retrieves all active data sources
func (r *PostgresRepository) GetActiveSources(ctx context.Context) ([]*DataSource, error) {
	query := `
		SELECT id, code, name, base_url, priority, is_authoritative, is_active, created_at
		FROM macro_data_sources
		WHERE is_active = true
		ORDER BY priority, code
	`

	var sources []*DataSource
	if err := r.db.SelectContext(ctx, &sources, query); err != nil {
		return nil, fmt.Errorf("failed to get active sources: %w", err)
	}

	return sources, nil
}

// ==================== INDICATOR-SOURCE MAPPING ====================

// GetSourcesForIndicator retrieves all sources for a given indicator, ordered by priority
func (r *PostgresRepository) GetSourcesForIndicator(ctx context.Context, indicatorCode string) ([]*IndicatorSourceWithDetails, error) {
	query := `
		SELECT
			s.id, s.indicator_type_id, s.source_id, s.priority, s.endpoint_config,
			s.is_active, s.last_success_at, s.consecutive_failures, s.created_at,
			t.code AS indicator_code, t.name AS indicator_name,
			d.code AS source_code, d.name AS source_name, COALESCE(d.base_url, '') AS source_base_url
		FROM macro_indicator_sources s
		JOIN macro_indicator_types t ON s.indicator_type_id = t.id
		JOIN macro_data_sources d ON s.source_id = d.id
		WHERE t.code = $1 AND s.is_active = true AND d.is_active = true
		ORDER BY s.priority, s.consecutive_failures
	`

	var sources []*IndicatorSourceWithDetails
	if err := r.db.SelectContext(ctx, &sources, query, indicatorCode); err != nil {
		return nil, fmt.Errorf("failed to get sources for indicator %s: %w", indicatorCode, err)
	}

	return sources, nil
}

// UpdateSourceStatus updates the last_success_at and consecutive_failures for a source
func (r *PostgresRepository) UpdateSourceStatus(ctx context.Context, sourceID uuid.UUID, success bool) error {
	var query string
	if success {
		query = `
			UPDATE macro_indicator_sources
			SET last_success_at = now(), consecutive_failures = 0
			WHERE id = $1
		`
	} else {
		query = `
			UPDATE macro_indicator_sources
			SET consecutive_failures = consecutive_failures + 1
			WHERE id = $1
		`
	}

	_, err := r.db.ExecContext(ctx, query, sourceID)
	if err != nil {
		return fmt.Errorf("failed to update source status: %w", err)
	}

	return nil
}

// ==================== INDICATOR VALUES ====================

// GetLatestValue retrieves the most recent value for an indicator
func (r *PostgresRepository) GetLatestValue(ctx context.Context, indicatorCode string) (*IndicatorValueWithSource, error) {
	query := `
		SELECT
			v.id, v.indicator_type_id, v.source_id, v.value, v.effective_date,
			v.reference_period, v.raw_response, v.fetched_at, v.created_at,
			t.code AS indicator_code, t.name AS indicator_name, d.code AS source_code
		FROM macro_indicator_values v
		JOIN macro_indicator_types t ON v.indicator_type_id = t.id
		JOIN macro_data_sources d ON v.source_id = d.id
		WHERE t.code = $1
		ORDER BY v.effective_date DESC, v.fetched_at DESC
		LIMIT 1
	`

	var value IndicatorValueWithSource
	if err := r.db.GetContext(ctx, &value, query, indicatorCode); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No data yet, not an error
		}
		return nil, fmt.Errorf("failed to get latest value for %s: %w", indicatorCode, err)
	}

	return &value, nil
}

// GetLatestValues retrieves the most recent values for multiple indicators
func (r *PostgresRepository) GetLatestValues(ctx context.Context, codes []string) (map[string]*IndicatorValueWithSource, error) {
	if len(codes) == 0 {
		return make(map[string]*IndicatorValueWithSource), nil
	}

	// Use DISTINCT ON to get latest value per indicator
	query := `
		SELECT DISTINCT ON (t.code)
			v.id, v.indicator_type_id, v.source_id, v.value, v.effective_date,
			v.reference_period, v.raw_response, v.fetched_at, v.created_at,
			t.code AS indicator_code, t.name AS indicator_name, d.code AS source_code
		FROM macro_indicator_values v
		JOIN macro_indicator_types t ON v.indicator_type_id = t.id
		JOIN macro_data_sources d ON v.source_id = d.id
		WHERE t.code = ANY($1)
		ORDER BY t.code, v.effective_date DESC, v.fetched_at DESC
	`

	var values []*IndicatorValueWithSource
	if err := r.db.SelectContext(ctx, &values, query, pq.Array(codes)); err != nil {
		return nil, fmt.Errorf("failed to get latest values: %w", err)
	}

	result := make(map[string]*IndicatorValueWithSource)
	for _, v := range values {
		result[v.IndicatorCode] = v
	}

	return result, nil
}

// GetAllLatestValues returns the most recent value for EACH active indicator
// Used by enrichment to get all available macro data for LLM consumption
func (r *PostgresRepository) GetAllLatestValues(ctx context.Context) ([]*IndicatorValueWithDetails, error) {
	query := `
		SELECT DISTINCT ON (t.code)
			v.id,
			t.code,
			t.name,
			t.category,
			COALESCE(t.unit, '') AS unit,
			v.value,
			v.effective_date,
			v.reference_period,
			d.code AS source_code,
			v.fetched_at
		FROM macro_indicator_values v
		JOIN macro_indicator_types t ON v.indicator_type_id = t.id
		JOIN macro_data_sources d ON v.source_id = d.id
		WHERE t.is_active = true
		ORDER BY t.code, v.effective_date DESC, v.fetched_at DESC
	`

	var values []*IndicatorValueWithDetails
	if err := r.db.SelectContext(ctx, &values, query); err != nil {
		return nil, fmt.Errorf("failed to get all latest values: %w", err)
	}

	return values, nil
}

// InsertValue inserts a new indicator value (upsert on unique constraint)
func (r *PostgresRepository) InsertValue(ctx context.Context, value *IndicatorValue) error {
	// Serialize raw_response if present
	var rawResponseJSON []byte
	if value.RawResponse != nil {
		rawResponseJSON = value.RawResponse
	}

	query := `
		INSERT INTO macro_indicator_values (
			indicator_type_id, source_id, value, effective_date, reference_period, raw_response, fetched_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
		ON CONFLICT (indicator_type_id, source_id, effective_date, COALESCE(reference_period, ''))
		DO UPDATE SET
			value = EXCLUDED.value,
			raw_response = EXCLUDED.raw_response,
			fetched_at = EXCLUDED.fetched_at
		RETURNING id, created_at
	`

	referencePeriod := ""
	if value.ReferencePeriod != nil {
		referencePeriod = *value.ReferencePeriod
	}

	err := r.db.QueryRowContext(
		ctx, query,
		value.IndicatorTypeID,
		value.SourceID,
		value.Value,
		value.EffectiveDate,
		sql.NullString{String: referencePeriod, Valid: value.ReferencePeriod != nil},
		rawResponseJSON,
		value.FetchedAt,
	).Scan(&value.ID, &value.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert indicator value: %w", err)
	}

	return nil
}

// GetHistory retrieves historical values for an indicator
func (r *PostgresRepository) GetHistory(ctx context.Context, indicatorCode string, from, to time.Time, limit int) ([]*IndicatorValue, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT
			v.id, v.indicator_type_id, v.source_id, v.value, v.effective_date,
			v.reference_period, v.fetched_at, v.created_at
		FROM macro_indicator_values v
		JOIN macro_indicator_types t ON v.indicator_type_id = t.id
		WHERE t.code = $1 AND v.effective_date BETWEEN $2 AND $3
		ORDER BY v.effective_date DESC
		LIMIT $4
	`

	var values []*IndicatorValue
	if err := r.db.SelectContext(ctx, &values, query, indicatorCode, from, to, limit); err != nil {
		return nil, fmt.Errorf("failed to get history for %s: %w", indicatorCode, err)
	}

	return values, nil
}

// ==================== FETCH LOGS ====================

// LogFetch logs a fetch operation
func (r *PostgresRepository) LogFetch(ctx context.Context, log *FetchLog) error {
	query := `
		INSERT INTO macro_fetch_logs (
			indicator_source_id, indicator_code, source_code, status,
			records_inserted, records_updated, error_message, response_time_ms, triggered_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(
		ctx, query,
		log.IndicatorSourceID,
		log.IndicatorCode,
		log.SourceCode,
		log.Status,
		log.RecordsInserted,
		log.RecordsUpdated,
		log.ErrorMessage,
		log.ResponseTimeMs,
		log.TriggeredBy,
	).Scan(&log.ID, &log.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to log fetch: %w", err)
	}

	return nil
}

// GetRecentLogs retrieves recent fetch logs for an indicator
func (r *PostgresRepository) GetRecentLogs(ctx context.Context, indicatorCode string, limit int) ([]*FetchLog, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT
			l.id, l.indicator_source_id, l.indicator_code, l.source_code, l.status,
			l.records_inserted, l.records_updated, l.error_message, l.response_time_ms,
			l.triggered_by, l.created_at
		FROM macro_fetch_logs l
		WHERE l.indicator_code = $1
		ORDER BY l.created_at DESC
		LIMIT $2
	`

	var logs []*FetchLog
	if err := r.db.SelectContext(ctx, &logs, query, indicatorCode, limit); err != nil {
		return nil, fmt.Errorf("failed to get logs for %s: %w", indicatorCode, err)
	}

	return logs, nil
}

// ==================== HELPER METHODS ====================

// GetEndpointConfig parses the endpoint_config JSONB into a map
func (s *IndicatorSourceWithDetails) GetEndpointConfig() (map[string]interface{}, error) {
	var config map[string]interface{}
	if err := json.Unmarshal(s.EndpointConfig, &config); err != nil {
		return nil, fmt.Errorf("failed to parse endpoint config: %w", err)
	}
	return config, nil
}
