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

// BCBSELICResponse represents the response from BCB SELIC API
// API: https://api.bcb.gov.br/dados/serie/bcdata.sgs.4189/dados/ultimos/1?formato=json
// Series 4189 = SELIC daily rate (taxa diária)
type BCBSELICResponse []struct {
	Data  string `json:"data"`
	Valor string `json:"valor"`
}

// SELICData holds the latest SELIC rate information
type SELICData struct {
	Rate     float64   `json:"rate"`
	Date     time.Time `json:"date"`
	Source   string    `json:"source"`
	Accuracy string    `json:"accuracy"` // "Authoritative" - official government API
	RawDate  string    `json:"raw_date"` // For debugging
}

// BCBClient handles Banco Central do Brasil API calls
type BCBClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewBCBClient creates a new BCB client
func NewBCBClient() *BCBClient {
	return &BCBClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.bcb.gov.br",
	}
}

// FetchLatestSELIC retrieves the latest SELIC rate from Banco Central
// SELIC (Sistema Especial de Liquidação e de Custódia) is Brazil's primary interest rate
// Series 4189 = taxa diária (daily rate), updated daily
func (b *BCBClient) FetchLatestSELIC(ctx context.Context) (*SELICData, error) {
	// Use series 4189 (SELIC daily rate) with /ultimos/1 for latest value
	url := fmt.Sprintf("%s/dados/serie/bcdata.sgs.4189/dados/ultimos/1?formato=json", b.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "ImensiahBot/1.0 (+https://imensiah.com)")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("BCB API request failed: %w", err)
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

	var bcbResp BCBSELICResponse
	if err := json.Unmarshal(body, &bcbResp); err != nil {
		return nil, fmt.Errorf("failed to parse BCB response: %w", err)
	}

	if len(bcbResp) == 0 {
		return nil, fmt.Errorf("no SELIC data returned from BCB API")
	}

	// Get the latest data point (first/only item when using /ultimos/1)
	latest := bcbResp[0]

	// Parse rate (valor is in percentage, e.g., "14.90")
	rate, err := strconv.ParseFloat(strings.TrimSpace(latest.Valor), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SELIC rate: %w", err)
	}

	// Parse date (format: "dd/mm/yyyy")
	parsedDate, err := time.Parse("02/01/2006", strings.TrimSpace(latest.Data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse SELIC date: %w", err)
	}

	return &SELICData{
		Rate:     rate,
		Date:     parsedDate,
		RawDate:  latest.Data,
		Source:   "Banco Central do Brasil (BCB) - Série 4189",
		Accuracy: "Authoritative",
	}, nil
}

// FetchSELICHistory retrieves historical SELIC rates (optional, for analysis)
func (b *BCBClient) FetchSELICHistory(ctx context.Context, days int) ([]SELICData, error) {
	url := fmt.Sprintf("%s/dados/serie/4390/dados?formato=json", b.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "ImensiahBot/1.0 (+https://imensiah.com)")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("BCB API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BCB API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var bcbResp BCBSELICResponse
	if err := json.Unmarshal(body, &bcbResp); err != nil {
		return nil, fmt.Errorf("failed to parse BCB response: %w", err)
	}

	var result []SELICData
	cutoff := time.Now().AddDate(0, 0, -days)

	for _, item := range bcbResp {
		parsedDate, err := time.Parse("02/01/2006", strings.TrimSpace(item.Data))
		if err != nil {
			continue
		}

		if parsedDate.Before(cutoff) {
			continue
		}

		rate, err := strconv.ParseFloat(strings.TrimSpace(item.Valor), 64)
		if err != nil {
			continue
		}

		result = append(result, SELICData{
			Rate:     rate,
			Date:     parsedDate,
			RawDate:  item.Data,
			Source:   "https://api.bcb.gov.br/dados/serie/4390/dados?formato=json",
			Accuracy: "Authoritative",
		})
	}

	return result, nil
}
