package enrichment

import (
	"encoding/json"
	"fmt"
	"strings"
)

// =============================================================================
// STEP-SPECIFIC PARSING
// =============================================================================

// ParseStep1Response parses the LLM response into Step1BasicInfo
func ParseStep1Response(content string) (*Step1BasicInfo, error) {
	cleanJSON := extractJSON(content)

	var result Step1BasicInfo
	if err := json.Unmarshal([]byte(cleanJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to parse step1 JSON: %w (content: %.500s)", err, cleanJSON)
	}

	// Validate confidence score
	if result.ConfidenceScore < 0 || result.ConfidenceScore > 100 {
		result.ConfidenceScore = 50
	}

	return &result, nil
}

// ParseStep2Response parses the LLM response into Step2BusinessModel
func ParseStep2Response(content string) (*Step2BusinessModel, error) {
	cleanJSON := extractJSON(content)

	var result Step2BusinessModel
	if err := json.Unmarshal([]byte(cleanJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to parse step2 JSON: %w (content: %.500s)", err, cleanJSON)
	}

	// Validate confidence score
	if result.ConfidenceScore < 0 || result.ConfidenceScore > 100 {
		result.ConfidenceScore = 50
	}

	return &result, nil
}

// ParseStep3Response parses the LLM response into Step3CompetitiveIntel
func ParseStep3Response(content string) (*Step3CompetitiveIntel, error) {
	cleanJSON := extractJSON(content)

	var result Step3CompetitiveIntel
	if err := json.Unmarshal([]byte(cleanJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to parse step3 JSON: %w (content: %.500s)", err, cleanJSON)
	}

	// Validate confidence score
	if result.ConfidenceScore < 0 || result.ConfidenceScore > 100 {
		result.ConfidenceScore = 50
	}

	return &result, nil
}

// =============================================================================
// LEGACY JSON PARSING
// =============================================================================

// ParseEnrichmentResponse parses the LLM response into EnrichedCompanyData
// DEPRECATED: Use ParseStep1Response, ParseStep2Response, ParseStep3Response
// Handles flexible JSON structures where models may return objects instead of strings
func ParseEnrichmentResponse(content string) (*EnrichedCompanyData, error) {
	// Clean and extract JSON
	cleanJSON := extractJSON(content)

	// First try strict parsing
	var result EnrichedCompanyData
	if err := json.Unmarshal([]byte(cleanJSON), &result); err != nil {
		// Fallback: try flexible parsing with intermediate struct
		result, flexErr := parseFlexibleJSON(cleanJSON)
		if flexErr != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w (content: %.500s)", err, cleanJSON)
		}
		return result, nil
	}

	// Validate confidence score
	if result.ConfidenceScore < 0 || result.ConfidenceScore > 100 {
		result.ConfidenceScore = 50 // Default to medium confidence
	}

	return &result, nil
}

// extractJSON cleans the response and extracts JSON content
func extractJSON(content string) string {
	cleanJSON := strings.TrimSpace(content)

	// Remove markdown code blocks if present
	if after, ok := strings.CutPrefix(cleanJSON, "```json"); ok {
		cleanJSON = after
	}
	if after, ok := strings.CutPrefix(cleanJSON, "```"); ok {
		cleanJSON = after
	}
	if strings.HasSuffix(cleanJSON, "```") {
		cleanJSON = strings.TrimSuffix(cleanJSON, "```")
	}
	cleanJSON = strings.TrimSpace(cleanJSON)

	// Find JSON boundaries (handles conversational text before/after JSON)
	startObj := strings.Index(cleanJSON, "{")
	startArr := strings.Index(cleanJSON, "[")
	endObj := strings.LastIndex(cleanJSON, "}")
	endArr := strings.LastIndex(cleanJSON, "]")

	// Determine which structure we have (object vs array)
	start := -1
	end := -1
	if startObj != -1 && endObj != -1 {
		if startArr == -1 || startObj < startArr {
			start, end = startObj, endObj
		}
	}
	if startArr != -1 && endArr != -1 {
		if start == -1 || startArr < start {
			start, end = startArr, endArr
		}
	}

	if start != -1 && end != -1 && end > start {
		cleanJSON = cleanJSON[start : end+1]
	}

	return cleanJSON
}

// =============================================================================
// FLEXIBLE JSON PARSING
// =============================================================================

// parseFlexibleJSON handles JSON where arrays may contain objects instead of strings
// Also handles nested structures where data might be under "empresa", "produtos", etc.
func parseFlexibleJSON(jsonStr string) (*EnrichedCompanyData, error) {
	// Parse into generic map first
	var rawMap map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &rawMap); err != nil {
		return nil, err
	}

	// Flatten nested structures (e.g., "empresa": {...} -> top level)
	flatMap := flattenNestedStructure(rawMap)

	// Convert back to JSON with normalized arrays
	normalized := normalizeArrayFields(flatMap)

	// Marshal and unmarshal into final struct
	normalizedJSON, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}

	var result EnrichedCompanyData
	if err := json.Unmarshal(normalizedJSON, &result); err != nil {
		return nil, err
	}

	// Validate confidence score
	if result.ConfidenceScore < 0 || result.ConfidenceScore > 100 {
		result.ConfidenceScore = 50
	}

	return &result, nil
}

// =============================================================================
// FIELD MAPPINGS
// =============================================================================

// fieldMappings maps Portuguese nested keys to our flat structure
var fieldMappings = map[string]string{
	"nome":                "legal_name",
	"razao_social":        "legal_name",
	"ano_fundacao":        "foundation_year",
	"fundacao":            "foundation_year",
	"numero_funcionarios": "employees_range",
	"funcionarios":        "employees_range",
	"cidade":              "location",
	"sede":                "headquarters",
	"setor":               "industry",
	"descricao":           "business_model",
	"modelo_negocio":      "business_model",
	"produtos":            "main_products",
	"produtos_servicos":   "main_products",
	"noticias":            "recent_news",
	"noticias_recentes":   "recent_news",
	"noticias_2024":       "recent_news",
	"executivos":          "key_executives",
	"executivos_chave":    "key_executives",
	"concorrentes":        "competitors",
	"forcas":              "strengths",
	"fraquezas":           "weaknesses",
	"oportunidades":       "opportunities",
	"ameacas":             "threats",
	"fontes":              "sources",
}

// arrayFields lists all fields that should be string arrays
var arrayFields = []string{
	"main_products", "service_areas", "tech_stack", "certifications",
	"key_partnerships", "recent_news", "key_executives", "customer_segments",
	"unique_selling_points", "competitors", "competitor_details",
	"strengths", "weaknesses", "opportunities", "threats",
	"strategic_challenges", "industry_trends", "sources",
}

// =============================================================================
// NORMALIZATION HELPERS
// =============================================================================

// flattenNestedStructure extracts data from nested objects like "empresa", "sede"
func flattenNestedStructure(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Process each key
	for key, value := range m {
		// Check if it's a nested object we should flatten
		if nested, ok := value.(map[string]interface{}); ok {
			// Special handling for known nested containers
			if key == "empresa" || key == "company" || key == "dados" || key == "data" {
				// Merge nested fields into result
				for nestedKey, nestedValue := range nested {
					targetKey := nestedKey
					if mapped, ok := fieldMappings[nestedKey]; ok {
						targetKey = mapped
					}
					// Handle nested sede object
					if nestedKey == "sede" {
						if sedeObj, ok := nestedValue.(map[string]interface{}); ok {
							if cidade, ok := sedeObj["cidade"].(string); ok {
								if estado, ok := sedeObj["estado"].(string); ok {
									result["headquarters"] = fmt.Sprintf("%s, %s", cidade, estado)
								} else {
									result["headquarters"] = cidade
								}
							}
							continue
						}
					}
					result[targetKey] = nestedValue
				}
				continue
			}
		}

		// Map Portuguese field names to English
		targetKey := key
		if mapped, ok := fieldMappings[key]; ok {
			targetKey = mapped
		}
		result[targetKey] = value
	}

	return result
}

// normalizeArrayFields converts object arrays to string arrays
func normalizeArrayFields(m map[string]interface{}) map[string]interface{} {
	for _, field := range arrayFields {
		if arr, ok := m[field].([]interface{}); ok {
			stringArr := make([]string, 0, len(arr))
			for _, item := range arr {
				stringArr = append(stringArr, objectToString(item))
			}
			m[field] = stringArr
		}
	}

	return m
}

// objectToString converts any value to a string representation
func objectToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case map[string]interface{}:
		// Try common field patterns for news/product objects
		if title, ok := val["titulo"].(string); ok {
			if date, ok := val["data"].(string); ok {
				return fmt.Sprintf("%s: %s", date, title)
			}
			return title
		}
		if name, ok := val["nome"].(string); ok {
			if desc, ok := val["descricao"].(string); ok {
				return fmt.Sprintf("%s - %s", name, desc)
			}
			return name
		}
		if name, ok := val["name"].(string); ok {
			if desc, ok := val["description"].(string); ok {
				return fmt.Sprintf("%s - %s", name, desc)
			}
			return name
		}
		// Generic: join all string values
		var parts []string
		for _, v := range val {
			if s, ok := v.(string); ok {
				parts = append(parts, s)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, " - ")
		}
		// Last resort: marshal to JSON
		b, _ := json.Marshal(val)
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}
