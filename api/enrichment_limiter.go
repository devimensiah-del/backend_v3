package api

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// MaxEnrichmentsPerDay is the maximum number of enrichments allowed per company per 24 hours
const MaxEnrichmentsPerDay = 10

// EnrichmentRateLimiter tracks enrichment requests per company to enforce rate limits
// Thread-safe implementation using sync.RWMutex
type EnrichmentRateLimiter struct {
	mu      sync.RWMutex
	records map[uuid.UUID][]time.Time // companyID -> list of enrichment timestamps
}

// NewEnrichmentRateLimiter creates a new rate limiter instance
func NewEnrichmentRateLimiter() *EnrichmentRateLimiter {
	return &EnrichmentRateLimiter{
		records: make(map[uuid.UUID][]time.Time),
	}
}

// CanEnrich checks if the company can be enriched based on rate limits
// Returns true if enrichment is allowed, false if rate limit exceeded
// Rate limit: Max MaxEnrichmentsPerDay enrichments per 24 hours per company
func (l *EnrichmentRateLimiter) CanEnrich(companyID uuid.UUID) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	timestamps, exists := l.records[companyID]
	if !exists {
		return true // No history, allow enrichment
	}

	// Filter to last 24 hours
	now := time.Now()
	cutoff := now.Add(-24 * time.Hour)
	validTimestamps := filterRecentTimestamps(timestamps, cutoff)

	// Check if under limit
	return len(validTimestamps) < MaxEnrichmentsPerDay
}

// RecordEnrichment records a new enrichment for rate limiting
func (l *EnrichmentRateLimiter) RecordEnrichment(companyID uuid.UUID) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// Get existing timestamps
	timestamps, exists := l.records[companyID]
	if !exists {
		timestamps = []time.Time{}
	}

	// Filter to last 24 hours and add new timestamp
	cutoff := now.Add(-24 * time.Hour)
	validTimestamps := filterRecentTimestamps(timestamps, cutoff)
	validTimestamps = append(validTimestamps, now)

	l.records[companyID] = validTimestamps
}

// GetRetryAfter calculates how long until the next enrichment is allowed
// Returns duration until the oldest record expires (allowing a new enrichment)
func (l *EnrichmentRateLimiter) GetRetryAfter(companyID uuid.UUID) time.Duration {
	l.mu.RLock()
	defer l.mu.RUnlock()

	timestamps, exists := l.records[companyID]
	if !exists {
		return 0 // No history, can enrich now
	}

	// Filter to last 24 hours
	now := time.Now()
	cutoff := now.Add(-24 * time.Hour)
	validTimestamps := filterRecentTimestamps(timestamps, cutoff)

	if len(validTimestamps) < MaxEnrichmentsPerDay {
		return 0 // Under limit, can enrich now
	}

	// Find oldest timestamp (first to expire)
	oldest := validTimestamps[0]
	for _, ts := range validTimestamps[1:] {
		if ts.Before(oldest) {
			oldest = ts
		}
	}

	// Calculate time until oldest expires (24h from oldest timestamp)
	expiresAt := oldest.Add(24 * time.Hour)
	retryAfter := expiresAt.Sub(now)

	if retryAfter < 0 {
		return 0
	}
	return retryAfter
}

// CleanupOldRecords removes expired timestamps for all companies
// Should be called periodically to prevent memory buildup
func (l *EnrichmentRateLimiter) CleanupOldRecords() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-24 * time.Hour)

	for companyID, timestamps := range l.records {
		validTimestamps := filterRecentTimestamps(timestamps, cutoff)
		if len(validTimestamps) == 0 {
			// No recent enrichments, remove entry
			delete(l.records, companyID)
		} else {
			l.records[companyID] = validTimestamps
		}
	}
}

// GetEnrichmentCount returns the number of enrichments in the last 24 hours for a company
func (l *EnrichmentRateLimiter) GetEnrichmentCount(companyID uuid.UUID) int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	timestamps, exists := l.records[companyID]
	if !exists {
		return 0
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	validTimestamps := filterRecentTimestamps(timestamps, cutoff)
	return len(validTimestamps)
}

// filterRecentTimestamps filters timestamps to only include those after the cutoff
func filterRecentTimestamps(timestamps []time.Time, cutoff time.Time) []time.Time {
	result := make([]time.Time, 0, len(timestamps))
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			result = append(result, ts)
		}
	}
	return result
}
