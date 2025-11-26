# Brazilian Macro-Economic Data Adapter

Real-time integration with official Brazilian economic data sources. Provides accurate, authoritative macro-economic context for strategic analysis.

## Data Sources

| Source | Data | API | Freshness | Accuracy |
|--------|------|-----|-----------|----------|
| **Banco Central do Brasil (BCB)** | SELIC Rate | `https://api.bcb.gov.br/dados/serie/4390/dados` | Daily (6 PM BR time) | Authoritative |
| **IBGE** | IPCA Inflation | `https://api.ibge.gov.br/v1/agregados/1737/periodos/-1/variaveis/2266` | Monthly | Authoritative |
| **BCB** | USD/BRL Exchange | `https://api.bcb.gov.br/v1/moedas/220/dados` | Daily | Authoritative |
| **ExchangeRate-API** | USD/BRL (Fallback) | `https://api.exchangerate-api.com/v4/latest/USD` | Real-time | High |

## Usage

### Basic Usage - Fetch All Macro Data

```go
package main

import (
	"context"
	"log"
	"time"

	"imensiah/backend_v3/adapter/macrodata"
)

func main() {
	// Create provider
	provider := macrodata.NewMacroDataProvider()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Fetch all latest macro data
	macro, err := provider.FetchLatestMacroData(ctx)
	if err != nil {
		log.Fatalf("failed to fetch macro data: %v", err)
	}

	// Use the data
	log.Printf("SELIC: %.2f%% (as of %s)", macro.EconomicIndicators.SELIC.Rate, macro.EconomicIndicators.SELIC.Date)
	log.Printf("IPCA: %.2f%% (%s)", macro.EconomicIndicators.IPCA.Rate, macro.EconomicIndicators.IPCA.MonthYear)
	log.Printf("USD/BRL: %.2f", macro.ExchangeRates.USDtoBRL.Rate)

	// Print summary
	log.Println(macro.GetSummary())
}
```

### Fetch Individual Indicators

```go
provider := macrodata.NewMacroDataProvider()
ctx := context.Background()

// Fetch just SELIC
selic, err := provider.bcbClient.FetchLatestSELIC(ctx)
log.Printf("SELIC Rate: %.2f%%", selic.Rate)

// Fetch just IPCA
ipca, err := provider.ibgeClient.FetchLatestIPCA(ctx)
log.Printf("IPCA: %.2f%% (%s)", ipca.Rate, ipca.MonthYear)

// Fetch exchange rate
usdBrl, err := provider.bcbExchangeClient.FetchUSDoBRL(ctx)
log.Printf("USD/BRL: %.2f", usdBrl.Rate)
```

### With Redis Caching

```go
import "github.com/redis/go-redis/v9"

// Create Redis client
redisClient := redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

// Create provider with Redis caching
providerWithCache := macrodata.NewMacroDataProviderWithRedis(redisClient)
ctx := context.Background()

// Fetches from Redis if cached, otherwise fetches fresh and caches
macro, err := providerWithCache.FetchLatestMacroDataWithCache(ctx)
if err != nil {
	log.Fatalf("failed to fetch macro data: %v", err)
}

// Force refresh (invalidate cache and fetch fresh)
macro, err = providerWithCache.InvalidateAndRefresh(ctx)

// Check cache stats
stats, _ := providerWithCache.cache.GetCacheStats(ctx)
log.Printf("Cached items: %+v", stats.CachedItems)
```

### Health Check

```go
provider := macrodata.NewMacroDataProvider()
ctx := context.Background()

// Check all API endpoints
health := provider.HealthCheck(ctx)
log.Printf("API Health: %+v", health)

// Output:
// API Health: map[bcb_selic:true ibge_ipca:true bcb_exchange:true fallback_exchange:true]
```

## Integration with Enrichment

### Step 1: Initialize in your enrichment service

```go
import "imensiah/backend_v3/adapter/macrodata"

type EnrichmentService struct {
	// ... existing fields
	macroProvider *macrodata.MacroDataProvider
	redisClient   *redis.Client
}

func NewEnrichmentService(db *sqlx.DB, redisClient *redis.Client) *EnrichmentService {
	return &EnrichmentService{
		// ... existing initialization
		macroProvider:  macrodata.NewMacroDataProvider(),
		redisClient:    redisClient,
	}
}
```

### Step 2: Fetch macro data before LLM call

```go
// In your enrichment workflow, before calling the LLM:

macro, err := s.macroProvider.FetchLatestMacroData(ctx)
if err != nil {
	log.Warnf("failed to fetch macro data: %v", err)
	// Continue without macro data (graceful degradation)
} else {
	// Inject into LLM prompt
	macroJSON, _ := json.Marshal(macro)

	prompt := strings.ReplaceAll(
		prompt,
		"{{REAL_TIME_MACRO_DATA}}",
		string(macroJSON),
	)
}
```

### Step 3: Use in prompts

```
You have access to real-time Brazilian macro-economic data:

{{REAL_TIME_MACRO_DATA}}

Use this official data (sources: BCB, IBGE) as context for your analysis.
If any of this data is outdated (>90 days), flag it in your response.
```

## Data Structures

### MacroContext (Top-level)

```go
type MacroContext struct {
	EconomicIndicators EconomicIndicators  // SELIC, IPCA, etc.
	ExchangeRates      ExchangeRates       // Currency pairs
	DataSources        []string            // URLs of data sources
	LastUpdated        time.Time           // When this was fetched
	FetchErrors        []string            // Any API failures (graceful degradation)
}
```

### SELICData

```go
type SELICData struct {
	Rate      float64   // e.g., 10.75 (percentage)
	Date      time.Time // When this rate became effective
	Source    string    // "https://api.bcb.gov.br/dados/serie/4390/dados"
	Accuracy  string    // "Authoritative"
	RawDate   string    // "15/11/2025"
}
```

### IPCAData

```go
type IPCAData struct {
	Rate        float64   // e.g., 4.50 (percentage)
	Period      string    // "202411" (November 2024)
	Date        time.Time
	MonthYear   string    // "November 2024" (human-readable)
	Source      string
	Accuracy    string    // "Authoritative"
}
```

### ExchangeRateData

```go
type ExchangeRateData struct {
	FromCurrency string    // "USD"
	ToCurrency   string    // "BRL"
	Rate         float64   // e.g., 5.42
	Date         time.Time
	Source       string
	Accuracy     string    // "Authoritative" or "High"
}
```

## Error Handling

The adapter uses **graceful degradation**:

- If BCB SELIC API fails → `FetchErrors` includes error, continues
- If IBGE IPCA API fails → continues, IPCA might be nil
- If BCB exchange rate fails → tries fallback (exchangerate-api.com)
- If all exchange rate APIs fail → continues without USD/BRL data

**Example:**

```go
macro, err := provider.FetchLatestMacroData(ctx)
if err != nil {
	log.Warnf("fetch failed: %v", err)
}

// Check what succeeded
if macro.EconomicIndicators.SELIC != nil {
	log.Println("SELIC data available")
}

if macro.EconomicIndicators.IPCA != nil {
	log.Println("IPCA data available")
}

// Check what failed
for _, errMsg := range macro.FetchErrors {
	log.Warnf("API error: %s", errMsg)
}
```

## Caching Strategy

### In-Memory Cache (Default)

- TTL: 6 hours
- Auto-expires after 6 hours
- Invalidatable: `provider.InvalidateCache()`
- Survives only for current process lifecycle

### Redis Cache (Optional)

- TTL: 6 hours (configurable)
- Survives process restarts
- Cache keys: `macrodata:selic`, `macrodata:ipca`, `macrodata:usd_brl`, `macrodata:macro_context`
- Useful for multi-instance deployments

**When to use which:**

- **Single instance**: In-memory cache is fine
- **Multiple instances**: Use Redis cache for consistency
- **High-traffic**: Use Redis cache to reduce API calls

## Testing

Run the test suite:

```bash
go test -v ./adapter/macrodata/
```

Tests include:
- Individual API client tests (BCB, IBGE, ExchangeRate)
- Aggregated provider tests
- Caching logic tests
- Data parsing tests
- Error handling tests

## API Rate Limits

| API | Rate Limit | Notes |
|-----|-----------|-------|
| BCB SELIC | Unlimited | Official government API |
| IBGE | Unlimited | Official government API |
| BCB Exchange | Unlimited | Official government API |
| ExchangeRate-API | 1,500/month (free tier) | Fallback only, rarely needed |

**Cost: $0** (all official APIs are free)

## Performance

Typical fetch times (with caching):

- **First fetch**: 5-10 seconds (parallel calls)
- **Cached fetch**: 1-5ms
- **Cache hit rate**: ~95% (6-hour TTL)

## Future Enhancements

- Add more economic indicators (employment, GDP forecast, etc.)
- Add sector-specific data (agribusiness, technology, etc.)
- Add market sentiment indicators
- Add international indicators (US Fed rate, global inflation, etc.)
- Webhook support for real-time updates
- Historical data aggregation

## Troubleshooting

### "API request timeout"
- Increase context timeout to 15-30 seconds
- Check network connectivity
- Check if BCB/IBGE APIs are up

### "No data returned from API"
- API might be down or changed
- Check the URLs in source code
- Run health check: `provider.HealthCheck(ctx)`

### "Cache not updating"
- Call `provider.InvalidateCache()` to force refresh
- Check Redis connectivity if using Redis cache

### "Data is stale"
- SELIC updates daily at 6 PM Brazil time
- IPCA updates monthly (around 10th of next month)
- Exchange rates update daily during market hours
- Check `LastUpdated` timestamp in `MacroContext`

## References

- [Banco Central do Brasil API](https://www.bcb.gov.br/en/open-data-portal)
- [IBGE SIDRA API](https://sidra.ibge.gov.br/api/v1)
- [ExchangeRate-API](https://www.exchangerate-api.com/)
