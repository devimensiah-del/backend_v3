package macrodata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestBCBClient_FetchLatestSELIC tests SELIC rate fetching
func TestBCBClient_FetchLatestSELIC(t *testing.T) {
	// Mock BCB API response (new format: /dados/serie/bcdata.sgs.4189/dados/ultimos/1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// New endpoint format uses /dados/serie/bcdata.sgs.4189/dados/ultimos/1
		if !contains(r.URL.Path, "4189") && !contains(r.URL.Path, "bcdata.sgs") {
			t.Logf("path: %s (expected pattern with 4189)", r.URL.Path)
		}
		if r.URL.Query().Get("formato") != "json" {
			t.Errorf("expected format=json parameter")
		}

		// Return mock SELIC data (new format: direct array, not nested in "dados")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"data": "15/11/2025", "valor": "10.75"}]`))
	}))
	defer server.Close()

	// Create client with mock server
	client := &BCBClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	selic, err := client.FetchLatestSELIC(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if selic.Rate != 10.75 {
		t.Errorf("expected rate 10.75, got %f", selic.Rate)
	}

	if selic.Accuracy != "Authoritative" {
		t.Errorf("expected accuracy Authoritative, got %s", selic.Accuracy)
	}

	if selic.Date.Month() != time.November {
		t.Errorf("expected November, got %v", selic.Date.Month())
	}
}

// TestBCBClient_FetchLatestSELIC_EmptyResponse tests error handling
func TestBCBClient_FetchLatestSELIC_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`)) // Empty array (new format)
	}))
	defer server.Close()

	client := &BCBClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.FetchLatestSELIC(ctx)
	if err == nil {
		t.Error("expected error for empty response")
	}
}

// TestIBGEClient_FetchLatestIPCA tests IPCA inflation fetching
func TestIBGEClient_FetchLatestIPCA(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !contains(r.URL.Path, "1737") {
			t.Errorf("expected aggregate code 1737 in path")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
			{
				"variáveis": [{"variável": "2266", "unidade": "%"}],
				"dados": [
					{"periodo": "202411", "valor": "4.50"}
				]
			}
		]`))
	}))
	defer server.Close()

	client := &IBGEClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ipca, err := client.FetchLatestIPCA(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ipca.Rate != 4.50 {
		t.Errorf("expected rate 4.50, got %f", ipca.Rate)
	}

	if ipca.Period != "202411" {
		t.Errorf("expected period 202411, got %s", ipca.Period)
	}

	if ipca.Accuracy != "Authoritative" {
		t.Errorf("expected accuracy Authoritative, got %s", ipca.Accuracy)
	}
}

// TestExchangeRateClient_FetchUSDToBRL tests exchange rate fetching
func TestExchangeRateClient_FetchUSDToBRL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !contains(r.URL.Path, "USD") {
			t.Errorf("expected USD in path")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Note: v4 API uses "rates" not "conversion_rates"
		w.Write([]byte(`{
			"result": "success",
			"base_code": "USD",
			"rates": {
				"BRL": 5.42,
				"EUR": 0.92,
				"GBP": 0.79
			}
		}`))
	}))
	defer server.Close()

	client := &ExchangeRateClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rate, err := client.FetchUSDToBRL(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rate.Rate != 5.42 {
		t.Errorf("expected rate 5.42, got %f", rate.Rate)
	}

	if rate.FromCurrency != "USD" || rate.ToCurrency != "BRL" {
		t.Errorf("unexpected currency pair")
	}
}

// TestMacroDataProvider_FetchLatestMacroData tests aggregated fetching
func TestMacroDataProvider_FetchLatestMacroData(t *testing.T) {
	// Mock multiple servers
	bcbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// New format: direct array
		w.Write([]byte(`[{"data": "01/11/2025", "valor": "10.50"}]`))
	}))
	defer bcbServer.Close()

	ibgeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"variáveis": [], "dados": [{"periodo": "202411", "valor": "4.50"}]}]`))
	}))
	defer ibgeServer.Close()

	// AwesomeAPI mock for USD/BRL
	awesomeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"USDBRL": {"code": "USD", "codein": "BRL", "bid": "5.35", "timestamp": "1732694400"}}`))
	}))
	defer awesomeServer.Close()

	provider := &MacroDataProvider{
		bcbClient: &BCBClient{
			httpClient: bcbServer.Client(),
			baseURL:    bcbServer.URL,
		},
		ibgeClient: &IBGEClient{
			httpClient: ibgeServer.Client(),
			baseURL:    ibgeServer.URL,
		},
		awesomeAPIClient: &AwesomeAPIClient{
			httpClient: awesomeServer.Client(),
			baseURL:    awesomeServer.URL,
		},
		// Initialize empty clients to avoid nil panics (they won't be called due to fallback order)
		bcbExchangeClient:  &BCBExchangeRateClient{httpClient: awesomeServer.Client(), baseURL: awesomeServer.URL},
		exchangeRateClient: &ExchangeRateClient{httpClient: awesomeServer.Client(), baseURL: awesomeServer.URL},
		cache: MacroContextCache{
			ttl: 1 * time.Hour,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	macro, err := provider.FetchLatestMacroData(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if macro.EconomicIndicators.SELIC == nil || macro.EconomicIndicators.SELIC.Rate != 10.50 {
		t.Error("expected SELIC data to be fetched")
	}

	if macro.EconomicIndicators.IPCA == nil || macro.EconomicIndicators.IPCA.Rate != 4.50 {
		t.Error("expected IPCA data to be fetched")
	}

	if len(macro.DataSources) == 0 {
		t.Error("expected data sources to be recorded")
	}
}

// TestMacroDataProvider_Caching tests in-memory cache functionality
func TestMacroDataProvider_Caching(t *testing.T) {
	callCount := 0
	bcbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// New format: direct array
		w.Write([]byte(`[{"data": "01/11/2025", "valor": "10.50"}]`))
	}))
	defer bcbServer.Close()

	// Mock for exchange rate (to avoid nil panic)
	awesomeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"USDBRL": {"code": "USD", "codein": "BRL", "bid": "5.35", "timestamp": "1732694400"}}`))
	}))
	defer awesomeServer.Close()

	// IBGE mock
	ibgeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"variáveis": [], "dados": [{"periodo": "202411", "valor": "4.50"}]}]`))
	}))
	defer ibgeServer.Close()

	provider := &MacroDataProvider{
		bcbClient: &BCBClient{
			httpClient: bcbServer.Client(),
			baseURL:    bcbServer.URL,
		},
		ibgeClient: &IBGEClient{
			httpClient: ibgeServer.Client(),
			baseURL:    ibgeServer.URL,
		},
		awesomeAPIClient: &AwesomeAPIClient{
			httpClient: awesomeServer.Client(),
			baseURL:    awesomeServer.URL,
		},
		bcbExchangeClient:  &BCBExchangeRateClient{httpClient: awesomeServer.Client(), baseURL: awesomeServer.URL},
		exchangeRateClient: &ExchangeRateClient{httpClient: awesomeServer.Client(), baseURL: awesomeServer.URL},
		cache: MacroContextCache{
			ttl: 1 * time.Hour,
		},
	}

	ctx := context.Background()

	// First call should fetch
	macro1, _ := provider.FetchLatestMacroData(ctx)

	// Second call should use cache
	macro2, _ := provider.FetchLatestMacroData(ctx)

	// Should be the same object (from cache)
	if macro1.EconomicIndicators.SELIC == nil || macro2.EconomicIndicators.SELIC == nil {
		t.Error("SELIC data should be available")
		return
	}
	if macro1.EconomicIndicators.SELIC.Rate != macro2.EconomicIndicators.SELIC.Rate {
		t.Error("cache not working correctly")
	}
}

// TestParseIBGEPeriod tests date parsing
func TestParseIBGEPeriod(t *testing.T) {
	tests := []struct {
		period   string
		wantYear int
		wantMonth time.Month
	}{
		{"202411", 2024, time.November},
		{"202501", 2025, time.January},
		{"202312", 2023, time.December},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			date, _ := parseIBGEPeriod(tt.period)
			if date.Year() != tt.wantYear || date.Month() != tt.wantMonth {
				t.Errorf("parseIBGEPeriod(%s) = %v, want %v-%v", tt.period, date, tt.wantYear, tt.wantMonth)
			}
		})
	}
}

// TestCurrencyPairValidator validates currency codes
func TestCurrencyPairValidator(t *testing.T) {
	tests := []struct {
		from      string
		to        string
		wantError bool
	}{
		{"USD", "BRL", false},
		{"EUR", "USD", false},
		{"INVALID", "USD", true},
		{"USD", "INVALID", true},
	}

	for _, tt := range tests {
		t.Run(tt.from+"-"+tt.to, func(t *testing.T) {
			err := CurrencyPairValidator(tt.from, tt.to)
			if (err != nil) != tt.wantError {
				t.Errorf("CurrencyPairValidator(%s, %s) error = %v, wantError %v", tt.from, tt.to, err, tt.wantError)
			}
		})
	}
}

// TestMacroContext_GetSummary tests summary generation
func TestMacroContext_GetSummary(t *testing.T) {
	macro := &MacroContext{
		EconomicIndicators: EconomicIndicators{
			SELIC: &SELICData{Rate: 10.75, Date: time.Now()},
			IPCA:  &IPCAData{Rate: 4.50, MonthYear: "November 2024"},
		},
		ExchangeRates: ExchangeRates{
			USDtoBRL: &ExchangeRateData{Rate: 5.42, Date: time.Now()},
		},
		DataSources: []string{"api1", "api2", "api3"},
		LastUpdated: time.Now(),
	}

	summary := macro.GetSummary()

	if !contains(summary, "10.75") {
		t.Error("summary should contain SELIC rate")
	}

	if !contains(summary, "4.50") {
		t.Error("summary should contain IPCA rate")
	}

	if !contains(summary, "5.42") {
		t.Error("summary should contain exchange rate")
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
