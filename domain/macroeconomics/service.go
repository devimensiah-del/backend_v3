package macroeconomics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"backend_v3/adapter/macrodata"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Service handles business logic for macroeconomics domain
type Service struct {
	repo          Repository
	macroProvider *macrodata.MacroDataProvider
}

// NewService creates a new macroeconomics service
func NewService(repo Repository, macroProvider *macrodata.MacroDataProvider) *Service {
	return &Service{
		repo:          repo,
		macroProvider: macroProvider,
	}
}

// GetLatestSnapshot retrieves the most recent values for ALL active indicators
// This is the primary method for enrichment consumption (fast DB read)
// Returns ALL indicators so LLM can intelligently select relevant ones
func (s *Service) GetLatestSnapshot(ctx context.Context) (*LatestSnapshot, error) {
	values, err := s.repo.GetAllLatestValues(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all latest values: %w", err)
	}

	snapshot := &LatestSnapshot{
		Indicators: make(map[string]*IndicatorValueSummary),
		AsOf:       time.Now(),
	}

	// Convert all values to summary format with full metadata
	for _, v := range values {
		snapshot.Indicators[v.Code] = detailsToSummary(v)
	}

	return snapshot, nil
}

// detailsToSummary converts IndicatorValueWithDetails to IndicatorValueSummary
func detailsToSummary(v *IndicatorValueWithDetails) *IndicatorValueSummary {
	return &IndicatorValueSummary{
		Code:            v.Code,
		Name:            v.Name,
		Category:        v.Category,
		Value:           v.Value,
		Unit:            v.Unit,
		EffectiveDate:   v.EffectiveDate,
		ReferencePeriod: v.ReferencePeriod,
		SourceCode:      v.SourceCode,
		FetchedAt:       v.FetchedAt,
	}
}

// valueToSummary converts IndicatorValueWithSource to IndicatorValueSummary (legacy)
func valueToSummary(v *IndicatorValueWithSource) *IndicatorValueSummary {
	return &IndicatorValueSummary{
		Code:            v.IndicatorCode,
		Name:            v.IndicatorName,
		Value:           v.Value,
		EffectiveDate:   v.EffectiveDate,
		ReferencePeriod: v.ReferencePeriod,
		SourceCode:      v.SourceCode,
		FetchedAt:       v.FetchedAt,
	}
}

// FetchIndicator fetches a single indicator from APIs and stores in DB
// Uses fallback chain based on priority
func (s *Service) FetchIndicator(ctx context.Context, code string) error {
	startTime := time.Now()

	logger := log.With().
		Str("component", "macroeconomics").
		Str("indicator", code).
		Logger()

	logger.Info().Msg("Fetching indicator")

	// Get indicator type
	indicator, err := s.repo.GetIndicatorByCode(ctx, code)
	if err != nil {
		return fmt.Errorf("indicator not found: %w", err)
	}

	// Get sources for this indicator (ordered by priority)
	sources, err := s.repo.GetSourcesForIndicator(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to get sources: %w", err)
	}

	if len(sources) == 0 {
		return fmt.Errorf("no active sources for indicator: %s", code)
	}

	// Try each source in priority order
	var lastErr error
	for _, source := range sources {
		fetchStartTime := time.Now()

		value, rawResponse, err := s.fetchFromSource(ctx, code, source)
		if err != nil {
			lastErr = err
			logger.Warn().
				Err(err).
				Str("source", source.SourceCode).
				Msg("Source fetch failed, trying next")

			// Update source status (failure)
			_ = s.repo.UpdateSourceStatus(ctx, source.ID, false)

			// Log the failure
			errMsg := err.Error()
			responseTime := int(time.Since(fetchStartTime).Milliseconds())
			_ = s.repo.LogFetch(ctx, &FetchLog{
				IndicatorSourceID: &source.ID,
				IndicatorCode:     &code,
				SourceCode:        &source.SourceCode,
				Status:            FetchStatusFailed,
				ErrorMessage:      &errMsg,
				ResponseTimeMs:    &responseTime,
				TriggeredBy:       TriggerScheduler,
			})

			continue
		}

		// Success! Store the value
		indicatorValue := &IndicatorValue{
			IndicatorTypeID: indicator.ID,
			SourceID:        source.SourceID,
			Value:           value,
			EffectiveDate:   time.Now(), // Will be overwritten for some indicators
			RawResponse:     rawResponse,
			FetchedAt:       time.Now(),
		}

		// Set effective date and reference period based on indicator type
		s.setEffectiveDate(indicatorValue, code, rawResponse)

		if err := s.repo.InsertValue(ctx, indicatorValue); err != nil {
			return fmt.Errorf("failed to store value: %w", err)
		}

		// Update source status (success)
		_ = s.repo.UpdateSourceStatus(ctx, source.ID, true)

		// Log success
		responseTime := int(time.Since(fetchStartTime).Milliseconds())
		_ = s.repo.LogFetch(ctx, &FetchLog{
			IndicatorSourceID: &source.ID,
			IndicatorCode:     &code,
			SourceCode:        &source.SourceCode,
			Status:            FetchStatusSuccess,
			RecordsInserted:   1,
			ResponseTimeMs:    &responseTime,
			TriggeredBy:       TriggerScheduler,
		})

		logger.Info().
			Str("source", source.SourceCode).
			Float64("value", value).
			Dur("duration", time.Since(startTime)).
			Msg("Indicator fetched and stored successfully")

		return nil
	}

	// All sources failed
	return fmt.Errorf("all sources failed for %s: %w", code, lastErr)
}

// fetchFromSource fetches an indicator from a specific source
func (s *Service) fetchFromSource(ctx context.Context, code string, source *IndicatorSourceWithDetails) (float64, json.RawMessage, error) {
	switch code {
	case "selic":
		return s.fetchSELIC(ctx, source)
	case "ipca":
		return s.fetchIPCA(ctx, source)
	case "usd_brl":
		return s.fetchUSDoBRL(ctx, source)
	default:
		return 0, nil, fmt.Errorf("unknown indicator: %s", code)
	}
}

// fetchSELIC fetches SELIC rate
func (s *Service) fetchSELIC(ctx context.Context, source *IndicatorSourceWithDetails) (float64, json.RawMessage, error) {
	switch source.SourceCode {
	case "bcb_api":
		data, err := s.macroProvider.FetchSpecificIndicator(ctx, "selic")
		if err != nil {
			return 0, nil, err
		}
		selic := data.(*macrodata.SELICData)
		rawJSON, _ := json.Marshal(selic)
		return selic.Rate, rawJSON, nil
	default:
		return 0, nil, fmt.Errorf("unsupported source for SELIC: %s", source.SourceCode)
	}
}

// fetchIPCA fetches IPCA inflation rate
func (s *Service) fetchIPCA(ctx context.Context, source *IndicatorSourceWithDetails) (float64, json.RawMessage, error) {
	switch source.SourceCode {
	case "ibge_sidra":
		data, err := s.macroProvider.FetchSpecificIndicator(ctx, "ipca")
		if err != nil {
			return 0, nil, err
		}
		ipca := data.(*macrodata.IPCAData)
		rawJSON, _ := json.Marshal(ipca)
		return ipca.Rate, rawJSON, nil
	default:
		return 0, nil, fmt.Errorf("unsupported source for IPCA: %s", source.SourceCode)
	}
}

// fetchUSDoBRL fetches USD/BRL exchange rate
func (s *Service) fetchUSDoBRL(ctx context.Context, source *IndicatorSourceWithDetails) (float64, json.RawMessage, error) {
	var data interface{}
	var err error

	switch source.SourceCode {
	case "awesomeapi":
		data, err = s.macroProvider.FetchSpecificIndicator(ctx, "usd_brl")
	case "bcb_api":
		data, err = s.macroProvider.FetchSpecificIndicator(ctx, "usd_brl")
	case "exchangerate_api":
		// The provider already handles fallback internally
		data, err = s.macroProvider.FetchSpecificIndicator(ctx, "usd_brl")
	default:
		return 0, nil, fmt.Errorf("unsupported source for USD/BRL: %s", source.SourceCode)
	}

	if err != nil {
		return 0, nil, err
	}

	exchangeRate := data.(*macrodata.ExchangeRateData)
	rawJSON, _ := json.Marshal(exchangeRate)
	return exchangeRate.Rate, rawJSON, nil
}

// setEffectiveDate sets the effective date and reference period based on raw response
func (s *Service) setEffectiveDate(value *IndicatorValue, code string, rawResponse json.RawMessage) {
	if rawResponse == nil {
		return
	}

	switch code {
	case "selic":
		var selic macrodata.SELICData
		if json.Unmarshal(rawResponse, &selic) == nil {
			value.EffectiveDate = selic.Date
		}
	case "ipca":
		var ipca macrodata.IPCAData
		if json.Unmarshal(rawResponse, &ipca) == nil {
			value.ReferencePeriod = &ipca.MonthYear
			// Parse MonthYear (e.g., "2024-11") to get effective date
			if t, err := time.Parse("2006-01", ipca.MonthYear); err == nil {
				value.EffectiveDate = t
			}
		}
	case "usd_brl":
		var exchange macrodata.ExchangeRateData
		if json.Unmarshal(rawResponse, &exchange) == nil {
			value.EffectiveDate = exchange.Date
		}
	}
}

// RefreshAll fetches all active indicators
func (s *Service) RefreshAll(ctx context.Context) error {
	logger := log.With().Str("component", "macroeconomics").Logger()
	logger.Info().Msg("Starting full macro data refresh")

	indicators, err := s.repo.GetActiveIndicators(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active indicators: %w", err)
	}

	var errors []string
	successCount := 0

	for _, indicator := range indicators {
		if err := s.FetchIndicator(ctx, indicator.Code); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", indicator.Code, err))
			logger.Error().
				Err(err).
				Str("indicator", indicator.Code).
				Msg("Failed to refresh indicator")
		} else {
			successCount++
		}
	}

	logger.Info().
		Int("total", len(indicators)).
		Int("success", successCount).
		Int("failed", len(errors)).
		Msg("Full macro data refresh completed")

	if len(errors) > 0 && successCount == 0 {
		return fmt.Errorf("all indicators failed: %v", errors)
	}

	return nil
}

// GetHistory retrieves historical values for trend analysis
func (s *Service) GetHistory(ctx context.Context, code string, from, to time.Time) (*IndicatorHistory, error) {
	indicator, err := s.repo.GetIndicatorByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("indicator not found: %w", err)
	}

	values, err := s.repo.GetHistory(ctx, code, from, to, 365) // Max 1 year of daily data
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	return &IndicatorHistory{
		IndicatorCode: indicator.Code,
		IndicatorName: indicator.Name,
		Unit:          indicator.Unit,
		Values:        values,
		From:          from,
		To:            to,
	}, nil
}

// GetAllIndicatorsWithLatest retrieves all indicators with their latest values
func (s *Service) GetAllIndicatorsWithLatest(ctx context.Context) ([]*IndicatorWithLatest, error) {
	indicators, err := s.repo.GetActiveIndicators(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get indicators: %w", err)
	}

	codes := make([]string, len(indicators))
	for i, ind := range indicators {
		codes[i] = ind.Code
	}

	latestValues, err := s.repo.GetLatestValues(ctx, codes)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest values: %w", err)
	}

	result := make([]*IndicatorWithLatest, len(indicators))
	for i, ind := range indicators {
		item := &IndicatorWithLatest{
			Indicator: ind,
		}
		if v, ok := latestValues[ind.Code]; ok && v != nil {
			item.LatestValue = valueToSummary(v)
		}
		result[i] = item
	}

	return result, nil
}

// ManualFetch triggers a manual fetch for an indicator (for admin use)
func (s *Service) ManualFetch(ctx context.Context, code string) error {
	startTime := time.Now()

	logger := log.With().
		Str("component", "macroeconomics").
		Str("indicator", code).
		Str("trigger", "manual").
		Logger()

	logger.Info().Msg("Manual fetch triggered")

	// Get indicator type
	indicator, err := s.repo.GetIndicatorByCode(ctx, code)
	if err != nil {
		return fmt.Errorf("indicator not found: %w", err)
	}

	// Get sources for this indicator
	sources, err := s.repo.GetSourcesForIndicator(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to get sources: %w", err)
	}

	if len(sources) == 0 {
		return fmt.Errorf("no active sources for indicator: %s", code)
	}

	// Try each source
	var lastErr error
	for _, source := range sources {
		fetchStartTime := time.Now()

		value, rawResponse, err := s.fetchFromSource(ctx, code, source)
		if err != nil {
			lastErr = err
			logger.Warn().Err(err).Str("source", source.SourceCode).Msg("Source failed")

			_ = s.repo.UpdateSourceStatus(ctx, source.ID, false)

			errMsg := err.Error()
			responseTime := int(time.Since(fetchStartTime).Milliseconds())
			_ = s.repo.LogFetch(ctx, &FetchLog{
				IndicatorSourceID: &source.ID,
				IndicatorCode:     &code,
				SourceCode:        &source.SourceCode,
				Status:            FetchStatusFailed,
				ErrorMessage:      &errMsg,
				ResponseTimeMs:    &responseTime,
				TriggeredBy:       TriggerManual,
			})

			continue
		}

		// Store the value
		indicatorValue := &IndicatorValue{
			IndicatorTypeID: indicator.ID,
			SourceID:        source.SourceID,
			Value:           value,
			EffectiveDate:   time.Now(),
			RawResponse:     rawResponse,
			FetchedAt:       time.Now(),
		}

		s.setEffectiveDate(indicatorValue, code, rawResponse)

		if err := s.repo.InsertValue(ctx, indicatorValue); err != nil {
			return fmt.Errorf("failed to store value: %w", err)
		}

		_ = s.repo.UpdateSourceStatus(ctx, source.ID, true)

		responseTime := int(time.Since(fetchStartTime).Milliseconds())
		_ = s.repo.LogFetch(ctx, &FetchLog{
			IndicatorSourceID: &source.ID,
			IndicatorCode:     &code,
			SourceCode:        &source.SourceCode,
			Status:            FetchStatusSuccess,
			RecordsInserted:   1,
			ResponseTimeMs:    &responseTime,
			TriggeredBy:       TriggerManual,
		})

		logger.Info().
			Str("source", source.SourceCode).
			Float64("value", value).
			Dur("duration", time.Since(startTime)).
			Msg("Manual fetch completed successfully")

		return nil
	}

	return fmt.Errorf("all sources failed: %w", lastErr)
}

// GetIndicatorByCode retrieves an indicator type by code
func (s *Service) GetIndicatorByCode(ctx context.Context, code string) (*IndicatorType, error) {
	return s.repo.GetIndicatorByCode(ctx, code)
}

// GetRecentLogs retrieves recent fetch logs for an indicator
func (s *Service) GetRecentLogs(ctx context.Context, code string, limit int) ([]*FetchLog, error) {
	return s.repo.GetRecentLogs(ctx, code, limit)
}

// Helper to convert to uuid pointer
func uuidPtr(id uuid.UUID) *uuid.UUID {
	return &id
}
