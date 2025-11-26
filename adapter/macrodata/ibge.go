package macrodata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// IBGEAggregateResponse represents the IBGE SIDRA API response structure
// API: https://api.ibge.gov.br/v3/agregados/1737/periodos/-1/dados?localidade=BR&variavel=2266
type IBGEAggregateResponse struct {
	Results []struct {
		Variables []struct {
			Variable string `json:"variável"`
			Unit     string `json:"unidade"`
		} `json:"variáveis"`
		Data []map[string]interface{} `json:"dados"`
	} `json:"resultados"`
}

// IBGEIPCAResponse represents a more direct IBGE response structure
type IBGEIPCAResponse struct {
	Results []IBGEResult `json:"resultados"`
}

// IBGEResult is a single result from IBGE
type IBGEResult struct {
	Variables []struct {
		Code string `json:"variavel"`
		Name string `json:"nome"`
	} `json:"variáveis"`
	Data []map[string]string `json:"dados"`
}

// IPCAData holds the latest IPCA inflation data
type IPCAData struct {
	Rate        float64   `json:"rate"`
	Period      string    `json:"period"`     // e.g., "202411" (November 2024)
	Date        time.Time `json:"date"`
	Source      string    `json:"source"`
	Accuracy    string    `json:"accuracy"` // "Authoritative"
	MonthYear   string    `json:"month_year"`
}

// IBGEClient handles IBGE API calls
type IBGEClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewIBGEClient creates a new IBGE client
func NewIBGEClient() *IBGEClient {
	return &IBGEClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.ibge.gov.br",
	}
}

// FetchLatestIPCA retrieves the latest IPCA inflation rate from IBGE
// IPCA (Índice Nacional de Preços ao Consumidor Amplo) is Brazil's primary inflation measure
// Updated monthly by IBGE
func (i *IBGEClient) FetchLatestIPCA(ctx context.Context) (*IPCAData, error) {
	// IBGE SIDRA API endpoint for IPCA
	// 1737 = IPCA aggregate code
	// -1 = latest period
	// 2266 = IPCA variation variable
	// BR = Brazil
	url := fmt.Sprintf("%s/v1/agregados/1737/periodos/-1/variaveis/2266?localidade=BR&formato=json", i.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "ImensiahBot/1.0 (+https://imensiah.com)")

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("IBGE API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("IBGE API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("failed to parse IBGE response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no IPCA data returned from IBGE API")
	}

	// Parse the first result (contains metadata and data)
	firstResult := results[0]

	// Extract the data array
	data, ok := firstResult["dados"].([]interface{})
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("invalid IPCA data structure from IBGE")
	}

	// Get latest data point
	latestRaw := data[0].(map[string]interface{})

	// Extract period and value
	period, ok := latestRaw["periodo"].(string)
	if !ok {
		return nil, fmt.Errorf("missing periodo field in IPCA data")
	}

	valueStr, ok := latestRaw["valor"].(string)
	if !ok {
		return nil, fmt.Errorf("missing valor field in IPCA data")
	}

	rate, err := strconv.ParseFloat(strings.TrimSpace(valueStr), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse IPCA rate: %w", err)
	}

	// Parse period (format: "202411" = November 2024)
	parsedDate, monthYear := parseIBGEPeriod(period)

	return &IPCAData{
		Rate:      rate,
		Period:    period,
		Date:      parsedDate,
		MonthYear: monthYear,
		Source:    "https://api.ibge.gov.br/v1/agregados/1737/periodos/-1/variaveis/2266",
		Accuracy:  "Authoritative",
	}, nil
}

// FetchIPCAHistory retrieves historical IPCA rates (optional)
func (i *IBGEClient) FetchIPCAHistory(ctx context.Context, months int) ([]IPCAData, error) {
	url := fmt.Sprintf("%s/v1/agregados/1737/periodos/-120/variaveis/2266?localidade=BR&formato=json", i.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "ImensiahBot/1.0 (+https://imensiah.com)")

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("IBGE API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IBGE API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("failed to parse IBGE response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no IPCA history returned from IBGE API")
	}

	var ipcaList []IPCAData
	firstResult := results[0]

	data, ok := firstResult["dados"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid IPCA data structure from IBGE")
	}

	cutoff := time.Now().AddDate(0, -months, 0)

	for _, d := range data {
		item := d.(map[string]interface{})

		period, ok := item["periodo"].(string)
		if !ok {
			continue
		}

		valueStr, ok := item["valor"].(string)
		if !ok {
			continue
		}

		rate, err := strconv.ParseFloat(strings.TrimSpace(valueStr), 64)
		if err != nil {
			continue
		}

		parsedDate, monthYear := parseIBGEPeriod(period)

		if parsedDate.Before(cutoff) {
			break
		}

		ipcaList = append(ipcaList, IPCAData{
			Rate:      rate,
			Period:    period,
			Date:      parsedDate,
			MonthYear: monthYear,
			Source:    "https://api.ibge.gov.br/v1/agregados/1737/periodos/-1/variaveis/2266",
			Accuracy:  "Authoritative",
		})
	}

	return ipcaList, nil
}

// parseIBGEPeriod converts IBGE period format (e.g., "202411") to time.Time and human-readable string
func parseIBGEPeriod(period string) (time.Time, string) {
	if len(period) != 6 {
		return time.Now(), period
	}

	year := period[:4]
	month := period[4:6]

	// Try to parse
	dateStr := fmt.Sprintf("%s-%s-01", year, month)
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Now(), period
	}

	monthYear := t.Format("January 2006")
	return t, monthYear
}
