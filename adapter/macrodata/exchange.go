package macrodata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ExchangeRateAPIResponse represents the response from exchangerate-api.com
type ExchangeRateAPIResponse struct {
	Result   string                  `json:"result"`
	BaseCode string                  `json:"base_code"`
	Rates    map[string]float64      `json:"conversion_rates"`
}

// ExchangeRateData holds exchange rate information
type ExchangeRateData struct {
	FromCurrency string    `json:"from_currency"`
	ToCurrency   string    `json:"to_currency"`
	Rate         float64   `json:"rate"`
	Date         time.Time `json:"date"`
	Source       string    `json:"source"`
	Accuracy     string    `json:"accuracy"` // "High" - real-time market data
}

// ExchangeRateClient handles exchange rate API calls
type ExchangeRateClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string // Optional, uses free tier if not set
}

// NewExchangeRateClient creates a new exchange rate client
// Using exchangerate-api.com free tier: 1,500 requests/month
func NewExchangeRateClient() *ExchangeRateClient {
	return &ExchangeRateClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.exchangerate-api.com/v4",
	}
}

// FetchUSDToBRL retrieves the current USD to BRL exchange rate
func (e *ExchangeRateClient) FetchUSDToBRL(ctx context.Context) (*ExchangeRateData, error) {
	return e.FetchExchangeRate(ctx, "USD", "BRL")
}

// FetchExchangeRate retrieves exchange rate between two currencies
// Free tier limit: 1,500 requests per month
func (e *ExchangeRateClient) FetchExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (*ExchangeRateData, error) {
	// Construct URL
	endpoint := fmt.Sprintf("%s/latest/%s", e.baseURL, fromCurrency)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "ImensiahBot/1.0 (+https://imensiah.com)")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchange rate API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("exchange rate API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp ExchangeRateAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse exchange rate response: %w", err)
	}

	rate, ok := apiResp.Rates[toCurrency]
	if !ok {
		return nil, fmt.Errorf("currency %s not found in response", toCurrency)
	}

	return &ExchangeRateData{
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		Rate:         rate,
		Date:         time.Now(),
		Source:       fmt.Sprintf("https://api.exchangerate-api.com/v4/latest/%s", fromCurrency),
		Accuracy:     "High", // Real-time market data
	}, nil
}

// BCBExchangeRateClient handles BCB official exchange rates (alternative, more accurate for BRL)
type BCBExchangeRateClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewBCBExchangeRateClient creates a new BCB exchange rate client
// This is more authoritative for BRL pairs than third-party APIs
func NewBCBExchangeRateClient() *BCBExchangeRateClient {
	return &BCBExchangeRateClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.bcb.gov.br",
	}
}

// FetchUSDoBRLfromBCB retrieves USD/BRL rate from Banco Central (more authoritative)
// Endpoint: https://api.bcb.gov.br/v1/moedas
func (b *BCBExchangeRateClient) FetchUSDoBRL(ctx context.Context) (*ExchangeRateData, error) {
	// BCB endpoint for USD (code: 220)
	endpoint := "https://api.bcb.gov.br/v1/moedas/220/dados"

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "ImensiahBot/1.0 (+https://imensiah.com)")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("BCB exchange rate API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("BCB API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// BCB returns an array of exchange rate data points
	var data []struct {
		Data       string `json:"data"`
		TaxaCompra float64 `json:"taxaCompra"` // Buy rate
		TaxaVenda  float64 `json:"taxaVenda"`  // Sell rate
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse BCB exchange rate response: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no exchange rate data returned from BCB API")
	}

	// Use the latest data point (average of buy/sell)
	latest := data[len(data)-1]
	avgRate := (latest.TaxaCompra + latest.TaxaVenda) / 2

	return &ExchangeRateData{
		FromCurrency: "USD",
		ToCurrency:   "BRL",
		Rate:         avgRate,
		Date:         time.Now(),
		Source:       "https://api.bcb.gov.br/v1/moedas/220/dados",
		Accuracy:     "Authoritative", // BCB official data
	}, nil
}

// CurrencyPairValidator validates currency pair codes
func CurrencyPairValidator(from, to string) error {
	validCurrencies := map[string]bool{
		"USD": true,
		"BRL": true,
		"EUR": true,
		"GBP": true,
		"JPY": true,
		"AUD": true,
		"CAD": true,
		"CHF": true,
	}

	if !validCurrencies[from] {
		return fmt.Errorf("unsupported currency: %s", from)
	}

	if !validCurrencies[to] {
		return fmt.Errorf("unsupported currency: %s", to)
	}

	return nil
}

