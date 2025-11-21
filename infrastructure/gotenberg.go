package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type GotenbergClient struct {
	URL        string // e.g., "http://localhost:3000"
	HTTPClient *http.Client
}

func NewGotenbergClient(url string) *GotenbergClient {
	return &GotenbergClient{
		URL: url,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second, // PDF generation can be slow
		},
	}
}

// Convert sends HTML files to Gotenberg's Chromium module
func (g *GotenbergClient) Convert(ctx context.Context, htmlPages []string) ([]byte, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 1. Add the index.html (Concatenate all pages or use the first one)
	// Gotenberg expects an index.html. For multi-page, we usually combine them
	// or send them as separate files if the template handles breaks.
	// Strategy: We combine them into one big HTML file with page-break CSS.

	var fullHTML bytes.Buffer
	fullHTML.WriteString(`<!DOCTYPE html><html><head><style>.page-break { page-break-after: always; }</style></head><body>`)

	for i, page := range htmlPages {
		fullHTML.WriteString(page)
		if i < len(htmlPages)-1 {
			fullHTML.WriteString(`<div class="page-break"></div>`)
		}
	}
	fullHTML.WriteString(`</body></html>`)

	part, err := writer.CreateFormFile("files", "index.html")
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, &fullHTML)
	if err != nil {
		return nil, err
	}

	// 2. Set Options (A4, margins, etc)
	_ = writer.WriteField("paperWidth", "11.7")  // A4 Landscape width (inches)
	_ = writer.WriteField("paperHeight", "8.27") // A4 Landscape height (inches)
	_ = writer.WriteField("marginTop", "0")
	_ = writer.WriteField("marginBottom", "0")
	_ = writer.WriteField("marginLeft", "0")
	_ = writer.WriteField("marginRight", "0")
	_ = writer.WriteField("printBackground", "true")

	if err := writer.Close(); err != nil {
		return nil, err
	}

	// 3. Execute Request
	req, err := http.NewRequestWithContext(ctx, "POST", g.URL+"/forms/chromium/convert/html", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gotenberg api error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	return io.ReadAll(resp.Body)
}
