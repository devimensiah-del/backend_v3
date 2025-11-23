package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"backend_v3/config"

	"github.com/sony/gobreaker"
)

// GenerationOptions holds model-specific parameters for LLM generation
// Supports heterogeneous model routing with framework-specific configurations
type GenerationOptions struct {
	Model       string
	Temperature float64
	MaxTokens   int
}

// Client wraps OpenRouter API with retry logic and circuit breaker
type Client struct {
	apiKey         string
	baseURL        string
	httpClient     *http.Client
	circuitBreaker *gobreaker.CircuitBreaker
}

// NewClient creates a new LLM client
func NewClient(apiKey string) *Client {
	return NewClientWithBaseURL(apiKey, "https://openrouter.ai/api/v1")
}

// NewClientWithBaseURL creates a new LLM client with a custom base URL (useful for testing)
func NewClientWithBaseURL(apiKey, baseURL string) *Client {
	cbSettings := gobreaker.Settings{
		Name:        "llm-client",
		MaxRequests: 2,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 3
		},
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		circuitBreaker: gobreaker.NewCircuitBreaker(cbSettings),
	}
}

// GenerateStructuredWithOptions is the new "Magic Method" for framework-specific model routing.
// Supports heterogeneous model configurations with different temperatures and max_tokens per framework.
func (c *Client) GenerateStructuredWithOptions(ctx context.Context, opts GenerationOptions, promptTemplate string, data interface{}, targetSchema interface{}) error {

	// 1. Prepare the Context
	finalPrompt := promptTemplate

	dataMap, isMap := data.(map[string]interface{})
	if isMap {
		for key, val := range dataMap {
			valBytes, _ := json.MarshalIndent(val, "", "  ")
			placeholder := fmt.Sprintf("{{%s}}", strings.ToUpper(key))

			if strings.Contains(finalPrompt, placeholder) {
				finalPrompt = strings.ReplaceAll(finalPrompt, placeholder, string(valBytes))
			} else {
				finalPrompt += fmt.Sprintf("\n\n%s:\n%s", key, string(valBytes))
			}
		}
	} else {
		valBytes, _ := json.MarshalIndent(data, "", "  ")
		finalPrompt += fmt.Sprintf("\n\nContext Data:\n%s", string(valBytes))
	}

	// 2. Create Request using GenerationOptions (framework-specific config)
	req := &Request{
		Model:        opts.Model,
		SystemPrompt: "You are a JSON-only API. Return strictly valid JSON matching the requested schema.",
		Messages: []Message{
			{Role: "user", Content: finalPrompt},
		},
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
	}

	// 3. Call API
	resp, err := c.Call(ctx, req)
	if err != nil {
		return err
	}

	// 4. Clean Markdown
	cleanJson := strings.TrimSpace(resp.Content)
	cleanJson = strings.TrimPrefix(cleanJson, "```json")
	cleanJson = strings.TrimPrefix(cleanJson, "```")
	cleanJson = strings.TrimSuffix(cleanJson, "```")

	// 5. Unmarshal into Target
	if err := json.Unmarshal([]byte(cleanJson), targetSchema); err != nil {
		return fmt.Errorf("failed to parse LLM JSON response: %w. Content: %s", err, cleanJson)
	}

	return nil
}

// GenerateStructured (DEPRECATED) - kept for backward compatibility.
// Use GenerateStructuredWithOptions for framework-specific routing.
func (c *Client) GenerateStructured(ctx context.Context, model string, promptTemplate string, data interface{}, targetSchema interface{}) error {
	opts := GenerationOptions{
		Model:       model,
		Temperature: 0.5, // Default temperature
		MaxTokens:   4000,
	}
	return c.GenerateStructuredWithOptions(ctx, opts, promptTemplate, data, targetSchema)
}

// Helper to convert FrameworkConfig to GenerationOptions
func NewGenerationOptions(cfg config.FrameworkConfig) GenerationOptions {
	return GenerationOptions{
		Model:       cfg.Model,
		Temperature: cfg.Temperature,
		MaxTokens:   cfg.MaxTokens,
	}
}

// Call makes an LLM API request (Low-level)
func (c *Client) Call(ctx context.Context, req *Request) (*Response, error) {
	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.callWithRetry(ctx, req, 3)
	})

	if err != nil {
		if err == gobreaker.ErrOpenState {
			return nil, fmt.Errorf("LLM service unavailable (Circuit Open): %w", err)
		}
		return nil, err
	}

	return result.(*Response), nil
}

func (c *Client) callWithRetry(ctx context.Context, req *Request, maxRetries int) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err := c.makeRequest(ctx, req)
		if err != nil {
			lastErr = err
			if isRetryable(err) {
				continue
			}
			return nil, err
		}
		return resp, nil
	}
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (c *Client) makeRequest(ctx context.Context, req *Request) (*Response, error) {
	body := c.buildRequestBody(req)
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("HTTP-Referer", "[https://imensiah.com](https://imensiah.com)")
	httpReq.Header.Set("X-Title", "Imensiah Business Intelligence")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return c.parseResponse(result), nil
}

func (c *Client) buildRequestBody(req *Request) map[string]interface{} {
	messages := []map[string]string{
		{"role": "system", "content": req.SystemPrompt},
	}
	for _, msg := range req.Messages {
		messages = append(messages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	body := map[string]interface{}{
		"model":       req.Model,
		"messages":    messages,
		"temperature": 0.7,
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}

	// Tools logic for Gemini
	if strings.Contains(req.Model, "gemini") && len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, 0)
		for _, tool := range req.Tools {
			if tool == "search" {
				tools = append(tools, map[string]interface{}{
					"google_search": map[string]interface{}{},
				})
			}
		}
		// Note: OpenRouter specific tool format might vary slightly by provider,
		// but this is the standard Gemini structure.
		if len(tools) > 0 {
			body["tools"] = tools
		}
	}
	return body
}

func (c *Client) parseResponse(result map[string]interface{}) *Response {
	resp := &Response{Sources: []Source{}}

	if usage, ok := result["usage"].(map[string]interface{}); ok {
		if total, ok := usage["total_tokens"].(float64); ok {
			resp.Tokens = int(total)
		}
	}

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					resp.Content = content
				}
			}
		}
	}
	return resp
}

func isRetryable(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "429") || strings.Contains(msg, "500") || strings.Contains(msg, "503")
}

// Structs
type Request struct {
	Model        string
	SystemPrompt string
	Messages     []Message
	Tools        []string
	MaxURLs      int
	Temperature  float64
	MaxTokens    int
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Response struct {
	Content string
	Sources []Source
	Tokens  int
	Model   string
}

type Source struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}
