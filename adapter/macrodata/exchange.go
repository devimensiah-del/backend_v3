package macrodata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// ExchangeRateAPIResponse represents the response from exchangerate-api.com
type ExchangeRateAPIResponse struct {
	Result   string             `json:"result"`
	BaseCode string             `json:"base_code"`
	Rates    map[string]float64 `json:"rates"` // Note: v4 uses "rates", not "conversion_rates"
}

// AwesomeAPIResponse represents the response from economia.awesomeapi.com.br
type AwesomeAPIResponse map[string]struct {
	Code       string `json:"code"`
	Codein     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
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

// AwesomeAPIClient handles exchange rates from economia.awesomeapi.com.br
// This is a reliable Brazilian API for real-time exchange rates
type AwesomeAPIClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewAwesomeAPIClient creates a new AwesomeAPI client
func NewAwesomeAPIClient() *AwesomeAPIClient {
	return &AwesomeAPIClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://economia.awesomeapi.com.br",
	}
}

// FetchUSDToBRL retrieves USD/BRL rate from AwesomeAPI (reliable Brazilian API)
func (a *AwesomeAPIClient) FetchUSDToBRL(ctx context.Context) (*ExchangeRateData, error) {
	endpoint := fmt.Sprintf("%s/json/last/USD-BRL", a.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "ImensiahBot/1.0 (+https://imensiah.com)")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("AwesomeAPI request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AwesomeAPI returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp AwesomeAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse AwesomeAPI response: %w", err)
	}

	usdBrl, ok := apiResp["USDBRL"]
	if !ok {
		return nil, fmt.Errorf("USDBRL not found in AwesomeAPI response")
	}

	// Parse bid rate (compra)
	rate, err := strconv.ParseFloat(usdBrl.Bid, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse exchange rate: %w", err)
	}

	// Parse timestamp
	timestamp, _ := strconv.ParseInt(usdBrl.Timestamp, 10, 64)
	rateTime := time.Unix(timestamp, 0)

	return &ExchangeRateData{
		FromCurrency: "USD",
		ToCurrency:   "BRL",
		Rate:         rate,
		Date:         rateTime,
		Source:       "AwesomeAPI (economia.awesomeapi.com.br)",
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

// FetchUSDoBRL retrieves USD/BRL rate from Banco Central (more authoritative)
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
		Data       string  `json:"data"`
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
		Source:       "Banco Central do Brasil (BCB)",
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
