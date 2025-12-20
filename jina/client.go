package jina

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// CNPJData contains structured data extracted from CNPJ registry (casadosdados.com.br)
type CNPJData struct {
	CNPJ           string   // Formatted CNPJ (e.g., "52.530.745/0001-66")
	LegalName      string   // Razão social
	TradeName      string   // Nome fantasia
	FoundationDate string   // Data de abertura (e.g., "16/10/2023")
	FoundationYear string   // Extracted year (e.g., "2023")
	Address        string   // Full address
	City           string   // City
	State          string   // State
	Phone          string   // Phone number
	Email          string   // Email
	CNAEPrimary    string   // Primary CNAE description
	CNAECodes      []string // All CNAE codes
	CapitalSocial  string   // Capital social
	CompanyType    string   // Micro Empresa, etc.
	Partners       []string // Partners with roles
	Status         string   // Ativa, Inativa, etc.
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

// FetchCNPJData fetches company data from casadosdados.com.br using the CNPJ number
// Returns nil if the CNPJ is invalid or data cannot be fetched
func (c *Client) FetchCNPJData(ctx context.Context, cnpj string) (*CNPJData, error) {
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

	// Parse the markdown content
	data := parseCNPJContent(content, cnpjDigits)

	log.Info().
		Str("cnpj", cnpj).
		Str("legal_name", data.LegalName).
		Int("partners_count", len(data.Partners)).
		Msg("CNPJ data fetched successfully")

	return data, nil
}

// FetchCNPJDataSafe is like FetchCNPJData but never returns an error
// Returns nil on any failure - use this when you want non-blocking behavior
func (c *Client) FetchCNPJDataSafe(ctx context.Context, cnpj string) *CNPJData {
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

// parseCNPJContent extracts structured data from casadosdados markdown content
func parseCNPJContent(content, cnpjDigits string) *CNPJData {
	data := &CNPJData{}

	// Format CNPJ
	if len(cnpjDigits) == 14 {
		data.CNPJ = fmt.Sprintf("%s.%s.%s/%s-%s",
			cnpjDigits[0:2], cnpjDigits[2:5], cnpjDigits[5:8],
			cnpjDigits[8:12], cnpjDigits[12:14])
	}

	lines := strings.Split(content, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)
		lineLower := strings.ToLower(line)

		// Legal Name - usually in a heading or after "Razão Social:"
		if strings.Contains(lineLower, "razão social") || strings.Contains(lineLower, "legal name") {
			if colonIdx := strings.Index(line, ":"); colonIdx != -1 {
				data.LegalName = strings.TrimSpace(line[colonIdx+1:])
			} else if i+1 < len(lines) {
				data.LegalName = strings.TrimSpace(lines[i+1])
			}
		}

		// Trade Name
		if strings.Contains(lineLower, "nome fantasia") || strings.Contains(lineLower, "trade name") {
			if colonIdx := strings.Index(line, ":"); colonIdx != -1 {
				data.TradeName = strings.TrimSpace(line[colonIdx+1:])
			}
		}

		// Foundation date
		if strings.Contains(lineLower, "abertura") || strings.Contains(lineLower, "established") || strings.Contains(lineLower, "founded") {
			// Look for date pattern DD/MM/YYYY
			dateRegex := regexp.MustCompile(`(\d{2}/\d{2}/\d{4})`)
			if matches := dateRegex.FindStringSubmatch(line); len(matches) > 0 {
				data.FoundationDate = matches[1]
				// Extract year
				parts := strings.Split(matches[1], "/")
				if len(parts) == 3 {
					data.FoundationYear = parts[2]
				}
			}
		}

		// Address
		if strings.Contains(lineLower, "address") || strings.Contains(lineLower, "endereço") || strings.Contains(lineLower, "location") {
			if colonIdx := strings.Index(line, ":"); colonIdx != -1 {
				data.Address = strings.TrimSpace(line[colonIdx+1:])
			}
		}

		// City/State from location patterns
		if strings.Contains(lineLower, "são paulo") || strings.Contains(lineLower, "rio de janeiro") {
			// Try to extract city, state pattern
			cityStateRegex := regexp.MustCompile(`([A-Za-záéíóúãõâêîôûÁÉÍÓÚÃÕÂÊÎÔÛ\s]+),?\s*([A-Z]{2})`)
			if matches := cityStateRegex.FindStringSubmatch(line); len(matches) > 2 {
				if data.City == "" {
					data.City = strings.TrimSpace(matches[1])
					data.State = matches[2]
				}
			}
		}

		// Phone
		if strings.Contains(lineLower, "phone") || strings.Contains(lineLower, "telefone") || strings.Contains(lineLower, "tel:") {
			phoneRegex := regexp.MustCompile(`\+?[\d\s\(\)\-]{10,}`)
			if matches := phoneRegex.FindString(line); matches != "" {
				data.Phone = strings.TrimSpace(matches)
			}
		}

		// Email
		if strings.Contains(lineLower, "email") || strings.Contains(lineLower, "e-mail") {
			emailRegex := regexp.MustCompile(`[\w\.\-]+@[\w\.\-]+\.\w+`)
			if matches := emailRegex.FindString(line); matches != "" {
				data.Email = matches
			}
		}

		// Capital Social
		if strings.Contains(lineLower, "capital") {
			capitalRegex := regexp.MustCompile(`R\$\s*[\d\.,]+`)
			if matches := capitalRegex.FindString(line); matches != "" {
				data.CapitalSocial = matches
			}
		}

		// CNAE Primary
		if strings.Contains(lineLower, "primary") && strings.Contains(lineLower, "activity") ||
			strings.Contains(lineLower, "atividade principal") ||
			strings.Contains(lineLower, "primary business") {
			// Next line or after colon often has the description
			if colonIdx := strings.Index(line, ":"); colonIdx != -1 {
				data.CNAEPrimary = strings.TrimSpace(line[colonIdx+1:])
			}
		}

		// CNAE codes - look for patterns like "XX.XX-X-XX"
		cnaeRegex := regexp.MustCompile(`\d{2}\.\d{2}-\d-\d{2}`)
		if matches := cnaeRegex.FindAllString(line, -1); len(matches) > 0 {
			for _, code := range matches {
				if !contains(data.CNAECodes, code) {
					data.CNAECodes = append(data.CNAECodes, code)
				}
			}
		}

		// Company Type
		if strings.Contains(lineLower, "micro empresa") || strings.Contains(lineLower, "microempresa") {
			data.CompanyType = "Micro Empresa"
		} else if strings.Contains(lineLower, "pequeno porte") || strings.Contains(lineLower, "epp") {
			data.CompanyType = "Empresa de Pequeno Porte"
		} else if strings.Contains(lineLower, "médio porte") {
			data.CompanyType = "Empresa de Médio Porte"
		}

		// Partners - look for patterns with "Sócio" or "Partner"
		if strings.Contains(lineLower, "sócio") || strings.Contains(lineLower, "partner") ||
			strings.Contains(lineLower, "administrador") {
			// Extract name and role
			// Common patterns: "Name (Role)" or "Name - Role"
			partnerLine := line
			// Remove bullet points
			partnerLine = strings.TrimLeft(partnerLine, "•-*123456789. ")

			if partnerLine != "" && !strings.Contains(lineLower, "quadro de") &&
				!strings.Contains(lineLower, "partners:") && len(partnerLine) < 100 {
				// Clean up the partner entry
				if !contains(data.Partners, partnerLine) {
					data.Partners = append(data.Partners, partnerLine)
				}
			}
		}

		// Status
		if strings.Contains(lineLower, "status:") || strings.Contains(lineLower, "situação:") {
			if strings.Contains(lineLower, "ativa") || strings.Contains(lineLower, "active") {
				data.Status = "Ativa"
			} else if strings.Contains(lineLower, "inativa") || strings.Contains(lineLower, "inactive") {
				data.Status = "Inativa"
			}
		}
	}

	// Try to extract legal name from first heading if not found
	if data.LegalName == "" {
		for _, line := range lines {
			if strings.HasPrefix(line, "#") {
				// Remove # and trim
				name := strings.TrimSpace(strings.TrimLeft(line, "#"))
				if len(name) > 3 && !strings.Contains(strings.ToLower(name), "cnpj") {
					data.LegalName = name
					break
				}
			}
		}
	}

	return data
}

// contains checks if a slice contains a string
func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
