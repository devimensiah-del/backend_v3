package llm

import (
	"regexp"
	"strings"
)

// Sanitizer provides input sanitization to prevent prompt injection attacks.
// CRITICAL: All user input MUST be sanitized before being injected into prompts.
type Sanitizer struct {
	// maxFieldLength prevents DoS via extremely long inputs
	maxFieldLength int
}

// NewSanitizer creates a new sanitizer with default settings
func NewSanitizer() *Sanitizer {
	return &Sanitizer{
		maxFieldLength: 10000, // 10KB max per field
	}
}

// SanitizeForPrompt sanitizes user input before injecting into LLM prompts.
// This prevents prompt injection attacks by:
// 1. Escaping control sequences that could alter prompt behavior
// 2. Truncating excessively long inputs
// 3. Removing potentially harmful patterns
func (s *Sanitizer) SanitizeForPrompt(input string) string {
	if input == "" {
		return ""
	}

	// 1. Truncate to max length (prevents DoS)
	if len(input) > s.maxFieldLength {
		input = input[:s.maxFieldLength] + "... [truncated]"
	}

	// 2. Escape patterns that could be interpreted as prompt instructions
	// These patterns could trick the LLM into ignoring previous instructions
	injectionPatterns := []string{
		"IGNORE PREVIOUS INSTRUCTIONS",
		"IGNORE ALL INSTRUCTIONS",
		"DISREGARD PREVIOUS",
		"FORGET EVERYTHING",
		"NEW INSTRUCTIONS:",
		"SYSTEM:",
		"ASSISTANT:",
		"USER:",
		"```",
		"---",
	}

	result := input
	for _, pattern := range injectionPatterns {
		// Case-insensitive replacement with escaped version
		re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(pattern))
		result = re.ReplaceAllString(result, "[FILTERED]")
	}

	// 3. Escape newline-based injection attempts
	// Multiple newlines followed by instruction-like text
	multiNewlinePattern := regexp.MustCompile(`\n{3,}`)
	result = multiNewlinePattern.ReplaceAllString(result, "\n\n")

	// 4. Remove potential JSON breaking characters in specific contexts
	// (only escape if they appear in suspicious patterns)
	jsonBreakPattern := regexp.MustCompile(`\}\s*\{`)
	result = jsonBreakPattern.ReplaceAllString(result, "} {")

	return strings.TrimSpace(result)
}

// SanitizeMap sanitizes all string values in a map recursively
func (s *Sanitizer) SanitizeMap(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	result := make(map[string]interface{})
	for key, value := range data {
		switch v := value.(type) {
		case string:
			result[key] = s.SanitizeForPrompt(v)
		case map[string]interface{}:
			result[key] = s.SanitizeMap(v)
		case []interface{}:
			result[key] = s.sanitizeSlice(v)
		default:
			result[key] = value
		}
	}
	return result
}

// sanitizeSlice sanitizes all string values in a slice
func (s *Sanitizer) sanitizeSlice(data []interface{}) []interface{} {
	result := make([]interface{}, len(data))
	for i, value := range data {
		switch v := value.(type) {
		case string:
			result[i] = s.SanitizeForPrompt(v)
		case map[string]interface{}:
			result[i] = s.SanitizeMap(v)
		case []interface{}:
			result[i] = s.sanitizeSlice(v)
		default:
			result[i] = value
		}
	}
	return result
}

// SanitizeSubmissionData sanitizes the critical user-provided fields in submission data
// These are the fields most likely to be used for prompt injection
func (s *Sanitizer) SanitizeSubmissionData(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	result := make(map[string]interface{})
	for key, value := range data {
		result[key] = value
	}

	// Sanitize known user-input fields
	userInputFields := []string{
		"company_name",
		"business_challenge",
		"company_website",
		"company_industry",
		"company_location",
		"target_market",
		"funding_stage",
		"company_size",
	}

	for _, field := range userInputFields {
		if val, ok := result[field].(string); ok {
			result[field] = s.SanitizeForPrompt(val)
		}
	}

	return result
}

// DefaultSanitizer is the global sanitizer instance
var DefaultSanitizer = NewSanitizer()

// Sanitize is a convenience function using the default sanitizer
func Sanitize(input string) string {
	return DefaultSanitizer.SanitizeForPrompt(input)
}

// SanitizeData is a convenience function for sanitizing submission data
func SanitizeData(data map[string]interface{}) map[string]interface{} {
	return DefaultSanitizer.SanitizeSubmissionData(data)
}
