package scraper

import (
	"context"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// MetaData holds the extracted HTML tags
type MetaData struct {
	Title       string
	Description string
	Keywords    []string
	Language    string
	OGImage     string // Bonus: Capture their social preview image
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 3 * time.Second, // Strict timeout: Layer 1 must be fast
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return http.ErrUseLastResponse // Limit redirects
				}
				return nil
			},
		},
	}
}

func (c *Client) Scrape(ctx context.Context, url string) (MetaData, error) {
	// Normalize URL
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return MetaData{}, err
	}

	// Fake a browser User-Agent to avoid being blocked by simple firewalls
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ImensiahBot/1.0; +https://imensiah.com)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return MetaData{}, err
	}
	defer resp.Body.Close()

	return parseHTML(resp.Body)
}

func parseHTML(body interface{ Read([]byte) (int, error) }) (MetaData, error) {
	z := html.NewTokenizer(body)
	var data MetaData

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return data, nil // End of document or error, return what we found
		case html.StartTagToken, html.SelfClosingTagToken:
			t := z.Token()

			if t.Data == "title" {
				// Next token should be text
				if z.Next() == html.TextToken {
					data.Title = strings.TrimSpace(z.Token().Data)
				}
			} else if t.Data == "meta" {
				extractMeta(t, &data)
			} else if t.Data == "html" {
				for _, attr := range t.Attr {
					if attr.Key == "lang" {
						data.Language = attr.Val
					}
				}
			}
		}
	}
}

func extractMeta(t html.Token, data *MetaData) {
	var name, content, property string

	for _, attr := range t.Attr {
		if attr.Key == "name" {
			name = attr.Val
		}
		if attr.Key == "content" {
			content = attr.Val
		}
		if attr.Key == "property" {
			property = attr.Val
		}
	}

	if name == "description" || property == "og:description" {
		if data.Description == "" || len(content) > len(data.Description) {
			data.Description = strings.TrimSpace(content)
		}
	}

	if name == "keywords" {
		parts := strings.Split(content, ",")
		for _, p := range parts {
			data.Keywords = append(data.Keywords, strings.TrimSpace(p))
		}
	}

	if property == "og:image" {
		data.OGImage = content
	}
}
