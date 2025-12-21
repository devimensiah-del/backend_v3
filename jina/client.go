package jina

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// CNPJRegistryContent contains raw content from CNPJ registry (casadosdados.com.br)
// We pass this raw content to the LLM for extraction instead of parsing in Go
type CNPJRegistryContent struct {
	CNPJ    string // Formatted CNPJ (e.g., "52.530.745/0001-66")
	Content string // Raw markdown content from casadosdados
}

// Client is a simple HTTP client for Jina Reader API
// Jina Reader extracts clean markdown content from web pages
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Jina Reader client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://r.jina.ai",
	}
}

// ReadPage fetches a web page and returns its content as markdown
// Returns empty string on failure (non-blocking - caller should continue without content)
func (c *Client) ReadPage(ctx context.Context, url string) (string, error) {
	if url == "" {
		return "", nil
	}

	// Normalize URL
	url = strings.TrimSpace(url)
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	jinaURL := c.baseURL + "/" + url

	log.Debug().
		Str("url", url).
		Str("jina_url", jinaURL).
		Msg("Fetching page content via Jina Reader")

	req, err := http.NewRequestWithContext(ctx, "GET", jinaURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Jina Reader accepts these headers for better extraction
	req.Header.Set("Accept", "text/markdown")
	req.Header.Set("User-Agent", "Imensiah-Enrichment/1.0")

	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	latency := time.Since(startTime)

	if err != nil {
		log.Warn().
			Err(err).
			Str("url", url).
			Dur("latency", latency).
			Msg("Jina Reader request failed")
		return "", fmt.Errorf("jina request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warn().
			Int("status", resp.StatusCode).
			Str("url", url).
			Dur("latency", latency).
			Msg("Jina Reader returned non-200 status")
		return "", fmt.Errorf("jina returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	content := string(body)

	// Truncate if too long (keep it reasonable for LLM context)
	const maxContentLength = 15000
	if len(content) > maxContentLength {
		content = content[:maxContentLength] + "\n\n[... conteúdo truncado ...]"
	}

	log.Info().
		Str("url", url).
		Int("content_length", len(content)).
		Dur("latency", latency).
		Msg("Jina Reader fetched page content")

	return content, nil
}

// ReadPageSafe is like ReadPage but never returns an error
// Returns empty string on any failure - use this when you want non-blocking behavior
func (c *Client) ReadPageSafe(ctx context.Context, url string) string {
	content, err := c.ReadPage(ctx, url)
	if err != nil {
		log.Warn().
			Err(err).
			Str("url", url).
			Msg("Jina Reader failed, continuing without website content")
		return ""
	}
	return content
}

// FetchCNPJData fetches raw CNPJ registry content from casadosdados.com.br
// Returns the raw markdown content to be passed to LLM for extraction
// Returns nil if the CNPJ is invalid or data cannot be fetched
func (c *Client) FetchCNPJData(ctx context.Context, cnpj string) (*CNPJRegistryContent, error) {
	// Normalize CNPJ - extract only digits
	cnpjDigits := extractDigits(cnpj)
	if len(cnpjDigits) != 14 {
		return nil, fmt.Errorf("invalid CNPJ: must have 14 digits, got %d", len(cnpjDigits))
	}

	// Build casadosdados URL
	url := fmt.Sprintf("https://casadosdados.com.br/solucao/cnpj/%s", cnpjDigits)

	log.Debug().
		Str("cnpj", cnpj).
		Str("url", url).
		Msg("Fetching CNPJ data from casadosdados")

	content, err := c.ReadPage(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CNPJ data: %w", err)
	}

	if content == "" {
		return nil, fmt.Errorf("empty response from casadosdados")
	}

	// Format CNPJ for display
	formattedCNPJ := fmt.Sprintf("%s.%s.%s/%s-%s",
		cnpjDigits[0:2], cnpjDigits[2:5], cnpjDigits[5:8],
		cnpjDigits[8:12], cnpjDigits[12:14])

	log.Info().
		Str("cnpj", formattedCNPJ).
		Int("content_length", len(content)).
		Msg("CNPJ registry content fetched successfully")

	return &CNPJRegistryContent{
		CNPJ:    formattedCNPJ,
		Content: content,
	}, nil
}

// FetchCNPJDataSafe is like FetchCNPJData but never returns an error
// Returns nil on any failure - use this when you want non-blocking behavior
func (c *Client) FetchCNPJDataSafe(ctx context.Context, cnpj string) *CNPJRegistryContent {
	data, err := c.FetchCNPJData(ctx, cnpj)
	if err != nil {
		log.Warn().
			Err(err).
			Str("cnpj", cnpj).
			Msg("CNPJ data fetch failed, continuing without registry data")
		return nil
	}
	return data
}

// extractDigits removes all non-digit characters from a string
func extractDigits(s string) string {
	var digits strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	return digits.String()
}
