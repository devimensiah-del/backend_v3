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
type GenerationOptions struct {
	Model         string
	Temperature   float64
	MaxTokens     int
	FallbackModel string // Used if primary model fails (rate limit, unavailable, timeout)
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

// GenerateStructuredWithOptions generates structured output with automatic fallback on failure.
//
// Fallback Strategy:
//  1. Try primary model (opts.Model)
//  2. On retryable error (rate limit, timeout, 5xx), try fallback model (opts.FallbackModel)
//  3. Return error only if both fail
func (c *Client) GenerateStructuredWithOptions(ctx context.Context, opts GenerationOptions, promptTemplate string, data interface{}, targetSchema interface{}) error {

	// 1. Prepare the Context (same for primary and fallback)
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

	// 2. Try Primary Model
	resp, err := c.callWithModel(ctx, opts.Model, opts.Temperature, opts.MaxTokens, finalPrompt)

	// 3. On failure, try Fallback Model (if configured and error is retryable)
	if err != nil && opts.FallbackModel != "" && opts.FallbackModel != opts.Model {
		if isRetryable(err) {
			// Log the fallback attempt
			fmt.Printf("[LLM] Primary model %s failed (%v), trying fallback %s\n", opts.Model, err, opts.FallbackModel)

			resp, err = c.callWithModel(ctx, opts.FallbackModel, opts.Temperature, opts.MaxTokens, finalPrompt)
			if err != nil {
				return fmt.Errorf("both primary (%s) and fallback (%s) models failed: %w", opts.Model, opts.FallbackModel, err)
			}
			fmt.Printf("[LLM] Fallback model %s succeeded\n", opts.FallbackModel)
		} else {
			// Non-retryable error (e.g., invalid JSON response) - don't try fallback
			return err
		}
	} else if err != nil {
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

// callWithModel makes an LLM call with a specific model configuration
func (c *Client) callWithModel(ctx context.Context, model string, temperature float64, maxTokens int, prompt string) (*Response, error) {
	req := &Request{
		Model:        model,
		SystemPrompt: "You are a JSON-only API. Return strictly valid JSON matching the requested schema.",
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	return c.Call(ctx, req)
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

// NewGenerationOptions converts FrameworkConfig to GenerationOptions (includes fallback)
func NewGenerationOptions(cfg config.FrameworkConfig) GenerationOptions {
	return GenerationOptions{
		Model:         cfg.Model,
		Temperature:   cfg.Temperature,
		MaxTokens:     cfg.MaxTokens,
		FallbackModel: cfg.FallbackModel,
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

// CallWithFallback makes an LLM API request with automatic fallback to another model on failure.
// This is the recommended method for production use as it handles rate limits, timeouts, and model unavailability.
func (c *Client) CallWithFallback(ctx context.Context, req *Request, fallbackModel string) (*Response, error) {
	primaryModel := req.Model

	// Try primary model first
	resp, err := c.Call(ctx, req)
	if err == nil {
		return resp, nil
	}

	// Check if we have a fallback and error is retryable
	if fallbackModel == "" || fallbackModel == primaryModel {
		return nil, err
	}

	if !isRetryable(err) {
		// Non-retryable error - don't try fallback
		return nil, err
	}

	// Log and try fallback model
	fmt.Printf("[LLM] Primary model %s failed (%v), trying fallback %s\n", primaryModel, err, fallbackModel)

	// Create new request with fallback model
	fallbackReq := &Request{
		Model:        fallbackModel,
		SystemPrompt: req.SystemPrompt,
		Messages:     req.Messages,
		Tools:        req.Tools,
		MaxURLs:      req.MaxURLs,
		Temperature:  req.Temperature,
		MaxTokens:    req.MaxTokens,
	}

	resp, err = c.Call(ctx, fallbackReq)
	if err != nil {
		return nil, fmt.Errorf("both primary (%s) and fallback (%s) models failed: %w", primaryModel, fallbackModel, err)
	}

	fmt.Printf("[LLM] Fallback model %s succeeded\n", fallbackModel)
	return resp, nil
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

	fmt.Printf("[LLM] Preparing request to %s (model: %s, tokens: %d)\n", c.baseURL, req.Model, req.MaxTokens)
	startTime := time.Now()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("HTTP-Referer", "https://imensiah.com")
	httpReq.Header.Set("X-Title", "Imensiah Business Intelligence")

	fmt.Printf("[LLM] Sending HTTP request...\n")
	httpResp, err := c.httpClient.Do(httpReq)
	latencyMS := int(time.Since(startTime).Milliseconds())
	if err != nil {
		fmt.Printf("[LLM] HTTP request failed after %v: %v\n", time.Since(startTime), err)
		return nil, err
	}
	defer httpResp.Body.Close()
	fmt.Printf("[LLM] Response received in %v (status: %d)\n", time.Since(startTime), httpResp.StatusCode)

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

	resp := c.parseResponse(result, req.Model)
	resp.LatencyMS = latencyMS

	// Log usage summary
	fmt.Printf("[LLM] Usage: model=%s, input=%d, output=%d, total=%d, cost=$%.6f, latency=%dms\n",
		req.Model, resp.InputTokens, resp.OutputTokens, resp.Tokens, resp.CostUSD, resp.LatencyMS)

	return resp, nil
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

func (c *Client) parseResponse(result map[string]interface{}, model string) *Response {
	resp := &Response{Sources: []Source{}, Model: model}

	// Parse usage data (input/output/total tokens)
	if usage, ok := result["usage"].(map[string]interface{}); ok {
		if promptTokens, ok := usage["prompt_tokens"].(float64); ok {
			resp.InputTokens = int(promptTokens)
		}
		if completionTokens, ok := usage["completion_tokens"].(float64); ok {
			resp.OutputTokens = int(completionTokens)
		}
		if total, ok := usage["total_tokens"].(float64); ok {
			resp.Tokens = int(total)
		}
		// If total not provided, calculate it
		if resp.Tokens == 0 {
			resp.Tokens = resp.InputTokens + resp.OutputTokens
		}
	}

	// Calculate cost based on model pricing
	resp.CostUSD = CalculateCost(model, resp.InputTokens, resp.OutputTokens)

	// Parse response content
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
	Content      string
	Sources      []Source
	Tokens       int // Total tokens (kept for backwards compatibility)
	InputTokens  int // Prompt tokens
	OutputTokens int // Completion tokens
	Model        string
	CostUSD      float64 // Calculated cost
	LatencyMS    int     // Response time in milliseconds
}

type Source struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}
