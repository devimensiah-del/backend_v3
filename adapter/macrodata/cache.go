package macrodata

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache provides persistent caching for macro data across service restarts
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

// CacheKeyPrefix is the prefix for all macro data cache keys
const CacheKeyPrefix = "macrodata:"

// CacheKeys for different data types
const (
	KeySELIC         = CacheKeyPrefix + "selic"
	KeyIPCA          = CacheKeyPrefix + "ipca"
	KeyUSDtoBRL      = CacheKeyPrefix + "usd_brl"
	KeyMacroContext  = CacheKeyPrefix + "macro_context"
	KeyLastFetchTime = CacheKeyPrefix + "last_fetch"
)

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(redisClient *redis.Client, ttl time.Duration) *RedisCache {
	return &RedisCache{
		client: redisClient,
		ttl:    ttl,
	}
}

// CacheSELIC stores SELIC data in Redis
func (rc *RedisCache) CacheSELIC(ctx context.Context, selic *SELICData) error {
	data, err := json.Marshal(selic)
	if err != nil {
		return fmt.Errorf("failed to marshal SELIC data: %w", err)
	}

	return rc.client.Set(ctx, KeySELIC, data, rc.ttl).Err()
}

// GetCachedSELIC retrieves SELIC data from Redis
func (rc *RedisCache) GetCachedSELIC(ctx context.Context) (*SELICData, error) {
	val, err := rc.client.Get(ctx, KeySELIC).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss, not an error
	}
	if err != nil {
		return nil, fmt.Errorf("Redis get failed: %w", err)
	}

	var selic SELICData
	if err := json.Unmarshal([]byte(val), &selic); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SELIC data: %w", err)
	}

	return &selic, nil
}

// CacheIPCA stores IPCA data in Redis
func (rc *RedisCache) CacheIPCA(ctx context.Context, ipca *IPCAData) error {
	data, err := json.Marshal(ipca)
	if err != nil {
		return fmt.Errorf("failed to marshal IPCA data: %w", err)
	}

	return rc.client.Set(ctx, KeyIPCA, data, rc.ttl).Err()
}

// GetCachedIPCA retrieves IPCA data from Redis
func (rc *RedisCache) GetCachedIPCA(ctx context.Context) (*IPCAData, error) {
	val, err := rc.client.Get(ctx, KeyIPCA).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Redis get failed: %w", err)
	}

	var ipca IPCAData
	if err := json.Unmarshal([]byte(val), &ipca); err != nil {
		return nil, fmt.Errorf("failed to unmarshal IPCA data: %w", err)
	}

	return &ipca, nil
}

// CacheExchangeRate stores exchange rate data in Redis
func (rc *RedisCache) CacheExchangeRate(ctx context.Context, rate *ExchangeRateData) error {
	data, err := json.Marshal(rate)
	if err != nil {
		return fmt.Errorf("failed to marshal exchange rate data: %w", err)
	}

	return rc.client.Set(ctx, KeyUSDtoBRL, data, rc.ttl).Err()
}

// GetCachedExchangeRate retrieves exchange rate data from Redis
func (rc *RedisCache) GetCachedExchangeRate(ctx context.Context) (*ExchangeRateData, error) {
	val, err := rc.client.Get(ctx, KeyUSDtoBRL).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Redis get failed: %w", err)
	}

	var rate ExchangeRateData
	if err := json.Unmarshal([]byte(val), &rate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal exchange rate data: %w", err)
	}

	return &rate, nil
}

// CacheMacroContext stores the entire macro context in Redis
func (rc *RedisCache) CacheMacroContext(ctx context.Context, macro *MacroContext) error {
	data, err := json.Marshal(macro)
	if err != nil {
		return fmt.Errorf("failed to marshal macro context: %w", err)
	}

	return rc.client.Set(ctx, KeyMacroContext, data, rc.ttl).Err()
}

// GetCachedMacroContext retrieves the entire macro context from Redis
func (rc *RedisCache) GetCachedMacroContext(ctx context.Context) (*MacroContext, error) {
	val, err := rc.client.Get(ctx, KeyMacroContext).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Redis get failed: %w", err)
	}

	var macro MacroContext
	if err := json.Unmarshal([]byte(val), &macro); err != nil {
		return nil, fmt.Errorf("failed to unmarshal macro context: %w", err)
	}

	return &macro, nil
}

// InvalidateAllCache clears all macro data from cache
func (rc *RedisCache) InvalidateAllCache(ctx context.Context) error {
	keys := []string{KeySELIC, KeyIPCA, KeyUSDtoBRL, KeyMacroContext, KeyLastFetchTime}

	for _, key := range keys {
		if err := rc.client.Del(ctx, key).Err(); err != nil {
			return fmt.Errorf("failed to delete cache key %s: %w", key, err)
		}
	}

	return nil
}

// SetLastFetchTime records when macro data was last successfully fetched
func (rc *RedisCache) SetLastFetchTime(ctx context.Context) error {
	return rc.client.Set(ctx, KeyLastFetchTime, time.Now().Unix(), rc.ttl).Err()
}

// GetLastFetchTime retrieves the timestamp of last successful fetch
func (rc *RedisCache) GetLastFetchTime(ctx context.Context) (time.Time, error) {
	val, err := rc.client.Get(ctx, KeyLastFetchTime).Result()
	if err == redis.Nil {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("Redis get failed: %w", err)
	}

	var timestamp int64
	if err := json.Unmarshal([]byte(val), &timestamp); err != nil {
		return time.Time{}, fmt.Errorf("failed to unmarshal timestamp: %w", err)
	}

	return time.Unix(timestamp, 0), nil
}

// CacheStats provides statistics about cache status
type CacheStats struct {
	TotalKeys     int       `json:"total_keys"`
	LastFetchTime time.Time `json:"last_fetch_time"`
	CachedItems   map[string]bool `json:"cached_items"`
}

// GetCacheStats returns statistics about what's currently cached
func (rc *RedisCache) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	stats := &CacheStats{
		CachedItems: make(map[string]bool),
	}

	// Check each key
	keys := map[string]string{
		"selic":          KeySELIC,
		"ipca":           KeyIPCA,
		"usd_brl":        KeyUSDtoBRL,
		"macro_context":  KeyMacroContext,
	}

	for name, key := range keys {
		exists := rc.client.Exists(ctx, key).Val()
		stats.CachedItems[name] = exists > 0
		if exists > 0 {
			stats.TotalKeys++
		}
	}

	// Get last fetch time
	lastFetch, _ := rc.GetLastFetchTime(ctx)
	stats.LastFetchTime = lastFetch

	return stats, nil
}

// MacroDataProviderWithRedis enhances MacroDataProvider with Redis caching
type MacroDataProviderWithRedis struct {
	provider  *MacroDataProvider
	cache    *RedisCache
}

// NewMacroDataProviderWithRedis creates a provider with Redis caching
func NewMacroDataProviderWithRedis(redisClient *redis.Client) *MacroDataProviderWithRedis {
	return &MacroDataProviderWithRedis{
		provider: NewMacroDataProvider(),
		cache:    NewRedisCache(redisClient, 6*time.Hour),
	}
}

// FetchLatestMacroDataWithCache fetches data with Redis caching
func (mpr *MacroDataProviderWithRedis) FetchLatestMacroDataWithCache(ctx context.Context) (*MacroContext, error) {
	// Try Redis cache first
	cached, err := mpr.cache.GetCachedMacroContext(ctx)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Fetch fresh data
	data, err := mpr.provider.FetchLatestMacroData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch macro data: %w", err)
	}

	// Store in Redis
	if err := mpr.cache.CacheMacroContext(ctx, data); err != nil {
		// Log but don't fail - caching is a nice-to-have
		fmt.Printf("Warning: failed to cache macro data: %v\n", err)
	}

	// Record fetch time
	_ = mpr.cache.SetLastFetchTime(ctx)

	return data, nil
}

// InvalidateAndRefresh clears cache and fetches fresh data
func (mpr *MacroDataProviderWithRedis) InvalidateAndRefresh(ctx context.Context) (*MacroContext, error) {
	if err := mpr.cache.InvalidateAllCache(ctx); err != nil {
		// Log but continue
		fmt.Printf("Warning: failed to invalidate cache: %v\n", err)
	}

	return mpr.FetchLatestMacroDataWithCache(ctx)
}
