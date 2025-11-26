package llm

// ModelPricing defines the cost per 1 million tokens for each model
// Prices are from OpenRouter (as of January 2025)
type ModelPricing struct {
	InputPerMillion  float64 // Cost per 1M input tokens (USD)
	OutputPerMillion float64 // Cost per 1M output tokens (USD)
}

// PricingTable contains pricing for all supported models
// Source: https://openrouter.ai/docs#models
var PricingTable = map[string]ModelPricing{
	// Google Gemini models
	"google/gemini-2.5-flash":       {InputPerMillion: 0.075, OutputPerMillion: 0.30},
	"google/gemini-2.5-pro":         {InputPerMillion: 1.25, OutputPerMillion: 5.00},
	"google/gemini-2.5-pro-preview": {InputPerMillion: 1.25, OutputPerMillion: 5.00},
	"google/gemini-2.0-flash":       {InputPerMillion: 0.10, OutputPerMillion: 0.40},
	"google/gemini-flash-1.5":       {InputPerMillion: 0.075, OutputPerMillion: 0.30},

	// Anthropic Claude models
	"anthropic/claude-sonnet-4":       {InputPerMillion: 3.00, OutputPerMillion: 15.00},
	"anthropic/claude-3.5-sonnet":     {InputPerMillion: 3.00, OutputPerMillion: 15.00},
	"anthropic/claude-3-opus":         {InputPerMillion: 15.00, OutputPerMillion: 75.00},
	"anthropic/claude-opus-4":         {InputPerMillion: 15.00, OutputPerMillion: 75.00},
	"anthropic/claude-3-haiku":        {InputPerMillion: 0.25, OutputPerMillion: 1.25},

	// OpenAI models
	"openai/gpt-4o":          {InputPerMillion: 2.50, OutputPerMillion: 10.00},
	"openai/gpt-4o-mini":     {InputPerMillion: 0.15, OutputPerMillion: 0.60},
	"openai/gpt-4-turbo":     {InputPerMillion: 10.00, OutputPerMillion: 30.00},
	"openai/gpt-4.1":         {InputPerMillion: 2.00, OutputPerMillion: 8.00},
	"openai/gpt-4.1-mini":    {InputPerMillion: 0.40, OutputPerMillion: 1.60},
	"openai/o1":              {InputPerMillion: 15.00, OutputPerMillion: 60.00},
	"openai/o1-mini":         {InputPerMillion: 3.00, OutputPerMillion: 12.00},

	// Perplexity models (for pre-search and enrichment)
	"perplexity/sonar-pro":          {InputPerMillion: 3.00, OutputPerMillion: 15.00},
	"perplexity/sonar-pro-search":   {InputPerMillion: 3.00, OutputPerMillion: 15.00}, // With search capability
	"perplexity/sonar":              {InputPerMillion: 1.00, OutputPerMillion: 1.00},
	"perplexity/sonar-reasoning":    {InputPerMillion: 1.00, OutputPerMillion: 5.00},

	// Meta Llama models (cost-effective fallbacks)
	"meta-llama/llama-3.3-70b-instruct": {InputPerMillion: 0.35, OutputPerMillion: 0.40},
	"meta-llama/llama-3.1-8b-instruct":  {InputPerMillion: 0.055, OutputPerMillion: 0.055},
}

// DefaultPricing is used for unknown models (conservative estimate)
var DefaultPricing = ModelPricing{
	InputPerMillion:  1.00,
	OutputPerMillion: 3.00,
}

// GetPricing returns the pricing for a model, or default if not found
func GetPricing(model string) ModelPricing {
	if pricing, ok := PricingTable[model]; ok {
		return pricing
	}
	return DefaultPricing
}

// CalculateCost computes the cost in USD for a given model and token counts
func CalculateCost(model string, inputTokens, outputTokens int) float64 {
	pricing := GetPricing(model)
	inputCost := float64(inputTokens) / 1_000_000 * pricing.InputPerMillion
	outputCost := float64(outputTokens) / 1_000_000 * pricing.OutputPerMillion
	return inputCost + outputCost
}

// UsageLog represents a logged LLM API call
type UsageLog struct {
	ID            string  `json:"id" db:"id"`
	SubmissionID  *string `json:"submission_id,omitempty" db:"submission_id"`
	FrameworkName string  `json:"framework_name" db:"framework_name"`
	ModelUsed     string  `json:"model_used" db:"model_used"`
	InputTokens   int     `json:"input_tokens" db:"input_tokens"`
	OutputTokens  int     `json:"output_tokens" db:"output_tokens"`
	TotalTokens   int     `json:"total_tokens" db:"total_tokens"`
	CostUSD       float64 `json:"cost_usd" db:"cost_usd"`
	LatencyMS     *int    `json:"latency_ms,omitempty" db:"latency_ms"`
	IsFallback    bool    `json:"is_fallback" db:"is_fallback"`
	ErrorMessage  *string `json:"error_message,omitempty" db:"error_message"`
	CreatedAt     string  `json:"created_at" db:"created_at"`
}
