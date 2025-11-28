package macrodata

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MacroContext represents all real-time Brazilian macro-economic data
// This is the unified structure that will be injected into enrichment
type MacroContext struct {
	EconomicIndicators EconomicIndicators `json:"economic_indicators"`
	ExchangeRates      ExchangeRates      `json:"exchange_rates"`
	DataSources        []string           `json:"data_sources"`
	LastUpdated        time.Time          `json:"last_updated"`
	FetchErrors        []string           `json:"fetch_errors,omitempty"` // Log any API failures
}

// EconomicIndicators holds all Brazilian economic data
type EconomicIndicators struct {
	SELIC *SELICData `json:"selic"`
	IPCA  *IPCAData  `json:"ipca"`
}

// ExchangeRates holds currency pair data
type ExchangeRates struct {
	USDtoBRL *ExchangeRateData `json:"usd_to_brl"`
}

// MacroDataProvider aggregates all macro data sources
type MacroDataProvider struct {
	bcbClient          *BCBClient
	ibgeClient         *IBGEClient
	exchangeRateClient *ExchangeRateClient
	bcbExchangeClient  *BCBExchangeRateClient
	awesomeAPIClient   *AwesomeAPIClient // Primary source for USD/BRL
	cache              MacroContextCache
	mu                 sync.RWMutex
}

// MacroContextCache holds cached macro data with TTL
type MacroContextCache struct {
	data      *MacroContext
	expiresAt time.Time
	mu        sync.RWMutex
	ttl       time.Duration
}

// NewMacroDataProvider creates a new macro data provider with all clients
func NewMacroDataProvider() *MacroDataProvider {
	return &MacroDataProvider{
		bcbClient:          NewBCBClient(),
		ibgeClient:         NewIBGEClient(),
		exchangeRateClient: NewExchangeRateClient(),
		bcbExchangeClient:  NewBCBExchangeRateClient(),
		awesomeAPIClient:   NewAwesomeAPIClient(), // Primary source for USD/BRL
		cache: MacroContextCache{
			ttl: 6 * time.Hour, // Cache macro data for 6 hours
		},
	}
}

// FetchLatestMacroData retrieves all latest macro-economic data
// Uses parallel fetching to minimize latency
// Falls back gracefully if any single API fails
func (p *MacroDataProvider) FetchLatestMacroData(ctx context.Context) (*MacroContext, error) {
	// Check cache first
	if cached := p.getFromCache(); cached != nil {
		return cached, nil
	}

	result := &MacroContext{
		DataSources: []string{},
		FetchErrors: []string{},
		LastUpdated: time.Now(),
	}

	// Parallel fetching with timeout
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 1. Fetch SELIC (BCB)
	wg.Add(1)
	go func() {
		defer wg.Done()
		selic, err := p.bcbClient.FetchLatestSELIC(ctx)
		mu.Lock()
		defer mu.Unlock()

		if err != nil {
			result.FetchErrors = append(result.FetchErrors, fmt.Sprintf("SELIC fetch failed: %v", err))
		} else {
			result.EconomicIndicators.SELIC = selic
			result.DataSources = append(result.DataSources, selic.Source)
		}
	}()

	// 2. Fetch IPCA (IBGE)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ipca, err := p.ibgeClient.FetchLatestIPCA(ctx)
		mu.Lock()
		defer mu.Unlock()

		if err != nil {
			result.FetchErrors = append(result.FetchErrors, fmt.Sprintf("IPCA fetch failed: %v", err))
		} else {
			result.EconomicIndicators.IPCA = ipca
			result.DataSources = append(result.DataSources, ipca.Source)
		}
	}()

	// 3. Fetch USD/BRL Exchange Rate (try AwesomeAPI first, then BCB, then fallback)
	wg.Add(1)
	go func() {
		defer wg.Done()

		var usdBrl *ExchangeRateData
		var err error

		// Try AwesomeAPI first (fast, reliable Brazilian API)
		if p.awesomeAPIClient != nil {
			usdBrl, err = p.awesomeAPIClient.FetchUSDToBRL(ctx)
		}

		// Fallback to BCB (authoritative)
		if err != nil || usdBrl == nil {
			usdBrl, err = p.bcbExchangeClient.FetchUSDoBRL(ctx)
		}

		// Final fallback to exchangerate-api.com
		if err != nil || usdBrl == nil {
			usdBrl, err = p.exchangeRateClient.FetchUSDToBRL(ctx)
			if err != nil {
				mu.Lock()
				result.FetchErrors = append(result.FetchErrors, fmt.Sprintf("USD/BRL fetch failed (all sources): %v", err))
				mu.Unlock()
				return
			}
		}

		mu.Lock()
		defer mu.Unlock()
		result.ExchangeRates.USDtoBRL = usdBrl
		result.DataSources = append(result.DataSources, usdBrl.Source)
	}()

	// Wait for all goroutines
	wg.Wait()

	// Cache the result if we got at least some data
	if len(result.DataSources) > 0 {
		p.setInCache(result)
	}

	return result, nil
}

// getFromCache retrieves cached data if still valid
func (p *MacroDataProvider) getFromCache() *MacroContext {
	p.cache.mu.RLock()
	defer p.cache.mu.RUnlock()

	if p.cache.data != nil && time.Now().Before(p.cache.expiresAt) {
		return p.cache.data
	}

	return nil
}

// setInCache stores data with expiration
func (p *MacroDataProvider) setInCache(data *MacroContext) {
	p.cache.mu.Lock()
	defer p.cache.mu.Unlock()

	p.cache.data = data
	p.cache.expiresAt = time.Now().Add(p.cache.ttl)
}

// InvalidateCache forces a fresh fetch on next call
func (p *MacroDataProvider) InvalidateCache() {
	p.cache.mu.Lock()
	defer p.cache.mu.Unlock()

	p.cache.data = nil
	p.cache.expiresAt = time.Time{}
}

// SetCacheTTL allows customization of cache duration
func (p *MacroDataProvider) SetCacheTTL(ttl time.Duration) {
	p.cache.mu.Lock()
	defer p.cache.mu.Unlock()

	p.cache.ttl = ttl
}

// FetchSpecificIndicator fetches a single economic indicator
// Useful for targeted refreshes
func (p *MacroDataProvider) FetchSpecificIndicator(ctx context.Context, indicator string) (interface{}, error) {
	switch indicator {
	case "selic":
		return p.bcbClient.FetchLatestSELIC(ctx)
	case "ipca":
		return p.ibgeClient.FetchLatestIPCA(ctx)
	case "usd_brl":
		// Try AwesomeAPI first (fast, reliable)
		if p.awesomeAPIClient != nil {
			if rate, err := p.awesomeAPIClient.FetchUSDToBRL(ctx); err == nil {
				return rate, nil
			}
		}
		// Fallback to BCB
		return p.bcbExchangeClient.FetchUSDoBRL(ctx)
	default:
		return nil, fmt.Errorf("unknown indicator: %s", indicator)
	}
}

// HealthCheck verifies all API endpoints are accessible
func (p *MacroDataProvider) HealthCheck(ctx context.Context) map[string]bool {
	health := make(map[string]bool)
	var wg sync.WaitGroup
	var mu sync.Mutex

	apis := []struct {
		name string
		fn   func(context.Context) error
	}{
		{
			name: "bcb_selic",
			fn: func(ctx context.Context) error {
				_, err := p.bcbClient.FetchLatestSELIC(ctx)
				return err
			},
		},
		{
			name: "ibge_ipca",
			fn: func(ctx context.Context) error {
				_, err := p.ibgeClient.FetchLatestIPCA(ctx)
				return err
			},
		},
		{
			name: "awesomeapi_exchange",
			fn: func(ctx context.Context) error {
				if p.awesomeAPIClient == nil {
					return fmt.Errorf("awesomeapi client not initialized")
				}
				_, err := p.awesomeAPIClient.FetchUSDToBRL(ctx)
				return err
			},
		},
		{
			name: "bcb_exchange",
			fn: func(ctx context.Context) error {
				_, err := p.bcbExchangeClient.FetchUSDoBRL(ctx)
				return err
			},
		},
		{
			name: "fallback_exchange",
			fn: func(ctx context.Context) error {
				_, err := p.exchangeRateClient.FetchUSDToBRL(ctx)
				return err
			},
		},
	}

	for _, api := range apis {
		wg.Add(1)
		go func(apiName string, fn func(context.Context) error) {
			defer wg.Done()
			err := fn(ctx)

			mu.Lock()
			health[apiName] = err == nil
			mu.Unlock()
		}(api.name, api.fn)
	}

	wg.Wait()
	return health
}

// GetSummary returns a human-readable summary of current macro indicators
func (c *MacroContext) GetSummary() string {
	summary := "Brazilian Macro-Economic Context:\n"

	if c.EconomicIndicators.SELIC != nil {
		summary += fmt.Sprintf("- SELIC Rate: %.2f%% (as of %s)\n",
			c.EconomicIndicators.SELIC.Rate,
			c.EconomicIndicators.SELIC.Date.Format("2006-01-02"))
	}

	if c.EconomicIndicators.IPCA != nil {
		summary += fmt.Sprintf("- IPCA Inflation: %.2f%% (%s)\n",
			c.EconomicIndicators.IPCA.Rate,
			c.EconomicIndicators.IPCA.MonthYear)
	}

	if c.ExchangeRates.USDtoBRL != nil {
		summary += fmt.Sprintf("- USD/BRL: %.2f (as of %s)\n",
			c.ExchangeRates.USDtoBRL.Rate,
			c.ExchangeRates.USDtoBRL.Date.Format("2006-01-02 15:04"))
	}

	summary += fmt.Sprintf("- Last Updated: %s\n", c.LastUpdated.Format("2006-01-02 15:04:05"))
	summary += fmt.Sprintf("- Data Sources: %d\n", len(c.DataSources))

	if len(c.FetchErrors) > 0 {
		summary += fmt.Sprintf("- Warnings: %d API failures (graceful degradation active)\n", len(c.FetchErrors))
	}

	return summary
}
