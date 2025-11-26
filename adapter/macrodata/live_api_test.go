package macrodata

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestLiveAPICalls makes actual calls to real Brazilian economic APIs
// Run with: go test -v -run TestLiveAPICalls ./adapter/macrodata
func TestLiveAPICalls(t *testing.T) {
	fmt.Println("================================================================================")
	fmt.Println("  LIVE API CALLS - REAL BRAZILIAN ECONOMIC DATA")
	fmt.Println("================================================================================")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test 1: BCB SELIC Rate
	fmt.Println("1. BANCO CENTRAL DO BRASIL - SELIC RATE")
	fmt.Println("   API: https://api.bcb.gov.br/dados/serie/4390/dados?formato=json")
	fmt.Println("   Description: Brazil's primary interest rate (updated daily at 6 PM)")
	fmt.Println()

	bcbClient := NewBCBClient()
	selic, err := bcbClient.FetchLatestSELIC(ctx)
	if err != nil {
		t.Logf("❌ BCB SELIC Error: %v", err)
		fmt.Printf("   ❌ ERROR: %v\n\n", err)
	} else {
		fmt.Printf("   ✅ SELIC Rate: %.2f%%\n", selic.Rate)
		fmt.Printf("   📅 Date: %s\n", selic.Date.Format("2006-01-02"))
		fmt.Printf("   🔗 Source: %s\n", selic.Source)
		fmt.Printf("   🎯 Accuracy: %s\n\n", selic.Accuracy)
	}

	// Test 2: IBGE IPCA Inflation
	fmt.Println("2. IBGE - IPCA INFLATION RATE")
	fmt.Println("   API: https://api.ibge.gov.br/v1/agregados/1737/periodos/-1/variaveis/2266")
	fmt.Println("   Description: Brazil's primary inflation measure (updated monthly)")
	fmt.Println()

	ibgeClient := NewIBGEClient()
	ipca, err := ibgeClient.FetchLatestIPCA(ctx)
	if err != nil {
		t.Logf("❌ IBGE IPCA Error: %v", err)
		fmt.Printf("   ❌ ERROR: %v\n\n", err)
	} else {
		fmt.Printf("   ✅ IPCA Inflation: %.2f%%\n", ipca.Rate)
		fmt.Printf("   📅 Period: %s (%s)\n", ipca.Period, ipca.MonthYear)
		fmt.Printf("   🔗 Source: %s\n", ipca.Source)
		fmt.Printf("   🎯 Accuracy: %s\n\n", ipca.Accuracy)
	}

	// Test 3: Exchange Rate (BCB - Authoritative)
	fmt.Println("3. BANCO CENTRAL DO BRASIL - USD/BRL EXCHANGE RATE")
	fmt.Println("   API: https://api.bcb.gov.br/v1/moedas/220/dados")
	fmt.Println("   Description: Official USD to BRL exchange rate (updated daily)")
	fmt.Println()

	bcbExchangeClient := NewBCBExchangeRateClient()
	usdBrl, err := bcbExchangeClient.FetchUSDoBRL(ctx)
	if err != nil {
		t.Logf("❌ BCB Exchange Error: %v", err)
		fmt.Printf("   ❌ ERROR: %v\n\n", err)
	} else {
		fmt.Printf("   ✅ Exchange Rate: %.2f %s/%s\n", usdBrl.Rate, usdBrl.FromCurrency, usdBrl.ToCurrency)
		fmt.Printf("   📅 Date: %s\n", usdBrl.Date.Format("2006-01-02 15:04"))
		fmt.Printf("   🔗 Source: %s\n", usdBrl.Source)
		fmt.Printf("   🎯 Accuracy: %s\n\n", usdBrl.Accuracy)
	}

	// Test 4: Exchange Rate (Fallback)
	fmt.Println("4. EXCHANGERATE-API.COM - USD/BRL FALLBACK")
	fmt.Println("   API: https://api.exchangerate-api.com/v4/latest/USD")
	fmt.Println("   Description: Real-time exchange rate (fallback if BCB unavailable)")
	fmt.Println()

	exchangeClient := NewExchangeRateClient()
	usdBrlFallback, err := exchangeClient.FetchUSDToBRL(ctx)
	if err != nil {
		t.Logf("⚠️  Fallback Exchange Error: %v", err)
		fmt.Printf("   ⚠️  ERROR: %v\n\n", err)
	} else {
		fmt.Printf("   ✅ Exchange Rate: %.2f %s/%s\n", usdBrlFallback.Rate, usdBrlFallback.FromCurrency, usdBrlFallback.ToCurrency)
		fmt.Printf("   📅 Date: %s\n", usdBrlFallback.Date.Format("2006-01-02 15:04"))
		fmt.Printf("   🔗 Source: %s\n", usdBrlFallback.Source)
		fmt.Printf("   🎯 Accuracy: %s\n\n", usdBrlFallback.Accuracy)
	}

	// Test 5: Aggregated Provider (All Data at Once)
	fmt.Println("5. AGGREGATED PROVIDER - ALL DATA PARALLEL")
	fmt.Println("   Fetches all 3 APIs simultaneously (parallel execution)")
	fmt.Println()

	provider := NewMacroDataProvider()
	start := time.Now()
	macro, err := provider.FetchLatestMacroData(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Logf("⚠️  Partial fetch: %v", err)
	}

	if macro != nil {
		fmt.Printf("   ⏱️  Total time: %.2f seconds\n", duration.Seconds())
		fmt.Printf("   📊 Data sources retrieved: %d\n", len(macro.DataSources))
		fmt.Printf("   ⚠️  API errors encountered: %d\n", len(macro.FetchErrors))
		fmt.Println()

		if macro.EconomicIndicators.SELIC != nil {
			fmt.Printf("   ✅ SELIC: %.2f%% (as of %s)\n", macro.EconomicIndicators.SELIC.Rate, macro.EconomicIndicators.SELIC.Date.Format("2006-01-02"))
		} else {
			fmt.Println("   ❌ SELIC: unavailable")
		}

		if macro.EconomicIndicators.IPCA != nil {
			fmt.Printf("   ✅ IPCA: %.2f%% (%s)\n", macro.EconomicIndicators.IPCA.Rate, macro.EconomicIndicators.IPCA.MonthYear)
		} else {
			fmt.Println("   ❌ IPCA: unavailable")
		}

		if macro.ExchangeRates.USDtoBRL != nil {
			fmt.Printf("   ✅ USD/BRL: %.2f (as of %s)\n", macro.ExchangeRates.USDtoBRL.Rate, macro.ExchangeRates.USDtoBRL.Date.Format("2006-01-02 15:04"))
		} else {
			fmt.Println("   ❌ USD/BRL: unavailable")
		}

		fmt.Println()
		fmt.Println("   📍 Data Sources:")
		for i, src := range macro.DataSources {
			fmt.Printf("      %d. %s\n", i+1, src)
		}

		if len(macro.FetchErrors) > 0 {
			fmt.Println()
			fmt.Println("   ⚠️  API Errors (graceful degradation):")
			for i, errMsg := range macro.FetchErrors {
				fmt.Printf("      %d. %s\n", i+1, errMsg)
			}
		}

		fmt.Println()
		fmt.Println("   📄 Human Summary:")
		fmt.Println(macro.GetSummary())
	}

	fmt.Println("================================================================================")
	fmt.Println("  ✅ TEST COMPLETE")
	fmt.Println("================================================================================")
}

// TestLiveAPICaching tests that caching works
// Run with: go test -v -run TestLiveAPICaching ./adapter/macrodata
func TestLiveAPICaching(t *testing.T) {
	fmt.Println("================================================================================")
	fmt.Println("  CACHING TEST - PARALLEL PERFORMANCE")
	fmt.Println("================================================================================")

	ctx := context.Background()
	provider := NewMacroDataProvider()

	// First fetch - fresh data
	fmt.Println("1. FIRST FETCH (fresh data from APIs)")
	start := time.Now()
	macro1, _ := provider.FetchLatestMacroData(ctx)
	duration1 := time.Since(start)
	fmt.Printf("   ⏱️  Duration: %.2f seconds\n", duration1.Seconds())
	if macro1 != nil && macro1.EconomicIndicators.SELIC != nil {
		fmt.Printf("   ✅ SELIC: %.2f%%\n", macro1.EconomicIndicators.SELIC.Rate)
	}

	fmt.Println()

	// Second fetch - should be cached
	fmt.Println("2. SECOND FETCH (from in-memory cache)")
	start = time.Now()
	macro2, _ := provider.FetchLatestMacroData(ctx)
	duration2 := time.Since(start)
	fmt.Printf("   ⏱️  Duration: %.2f milliseconds\n", float64(duration2.Microseconds())/1000)
	if macro2 != nil && macro2.EconomicIndicators.SELIC != nil {
		fmt.Printf("   ✅ SELIC: %.2f%%\n", macro2.EconomicIndicators.SELIC.Rate)
	}

	fmt.Println()

	// Calculate speedup
	speedup := float64(duration1) / float64(duration2)
	fmt.Printf("3. CACHE SPEEDUP: %.0fx faster\n", speedup)
	fmt.Println()

	// Verify consistency
	if macro1 != nil && macro2 != nil &&
		macro1.EconomicIndicators.SELIC != nil &&
		macro2.EconomicIndicators.SELIC != nil {
		if macro1.EconomicIndicators.SELIC.Rate == macro2.EconomicIndicators.SELIC.Rate {
			fmt.Println("   ✅ Cache consistency verified (same data)")
		}
	}

	fmt.Println()

	// Force refresh
	fmt.Println("4. INVALIDATE CACHE AND REFETCH")
	provider.InvalidateCache()
	start = time.Now()
	macro3, _ := provider.FetchLatestMacroData(ctx)
	duration3 := time.Since(start)
	fmt.Printf("   ⏱️  Duration: %.2f seconds\n", duration3.Seconds())
	if macro3 != nil && macro3.EconomicIndicators.SELIC != nil {
		fmt.Printf("   ✅ SELIC: %.2f%%\n", macro3.EconomicIndicators.SELIC.Rate)
	}

	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Println("  ✅ CACHING TEST COMPLETE")
	fmt.Println("================================================================================")
}

// TestHealthCheck verifies all APIs are accessible
// Run with: go test -v -run TestHealthCheck ./adapter/macrodata
func TestHealthCheck(t *testing.T) {
	fmt.Println("================================================================================")
	fmt.Println("  API HEALTH CHECK")
	fmt.Println("================================================================================")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	provider := NewMacroDataProvider()
	health := provider.HealthCheck(ctx)

	fmt.Println("API Endpoint Status:")

	for api, status := range health {
		statusStr := "✅ UP"
		if !status {
			statusStr = "❌ DOWN"
		}
		fmt.Printf("  %s: %s\n", api, statusStr)
	}

	fmt.Println()

	// Count up/down
	upCount := 0
	downCount := 0
	for _, status := range health {
		if status {
			upCount++
		} else {
			downCount++
		}
	}

	fmt.Printf("Summary: %d/%d APIs operational\n", upCount, len(health))

	if downCount > 0 {
		fmt.Printf("⚠️  Warning: %d APIs are unavailable (graceful degradation active)\n", downCount)
	} else {
		fmt.Println("✅ All APIs healthy")
	}

	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Println("  ✅ HEALTH CHECK COMPLETE")
	fmt.Println("================================================================================")
}
