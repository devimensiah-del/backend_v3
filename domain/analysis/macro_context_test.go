package analysis

import (
	"testing"

	"github.com/rs/zerolog"
)

// TestExtractMacroContext_WithValidData tests macro-context extraction with full data
func TestExtractMacroContext_WithValidData(t *testing.T) {
	service := &Service{
		logger: zerolog.Nop(),
	}

	enrichmentData := map[string]interface{}{
		"profile_overview": map[string]interface{}{
			"legal_name": "Test Company",
		},
		"macro_context": map[string]interface{}{
			"economic_indicators": map[string]interface{}{
				"country":       "Brazil",
				"gdp_growth":    "+2.5%",
				"interest_rate": "SELIC: 11.75%",
			},
			"industry_trends": map[string]interface{}{
				"growth_rate": "+12% CAGR",
				"key_trends":  []string{"IoT", "Sustainability"},
			},
		},
	}

	result := service.extractMacroContext(enrichmentData)

	// Verify macro_context was extracted
	if result == nil {
		t.Fatal("Expected macro_context to be extracted, got nil")
	}

	// Verify nested data is accessible
	if econ, ok := result["economic_indicators"].(map[string]interface{}); ok {
		if country, ok := econ["country"].(string); !ok || country != "Brazil" {
			t.Errorf("Expected country to be 'Brazil', got '%v'", country)
		}
	} else {
		t.Error("Expected economic_indicators to be a map")
	}
}

// TestExtractMacroContext_BackwardCompatibility tests old enrichments without macro_context
func TestExtractMacroContext_BackwardCompatibility(t *testing.T) {
	service := &Service{
		logger: zerolog.Nop(),
	}

	// Old enrichment data without macro_context
	enrichmentData := map[string]interface{}{
		"profile_overview": map[string]interface{}{
			"legal_name": "Test Company",
		},
		"financials": map[string]interface{}{
			"revenue_estimate": "R$ 5M",
		},
	}

	result := service.extractMacroContext(enrichmentData)

	// Should return empty map, not nil or error
	if result == nil {
		t.Fatal("Expected empty map for backward compatibility, got nil")
	}

	if len(result) != 0 {
		t.Errorf("Expected empty map, got %d keys", len(result))
	}
}

// TestExtractMacroContext_NilEnrichmentData tests nil enrichment data
func TestExtractMacroContext_NilEnrichmentData(t *testing.T) {
	service := &Service{
		logger: zerolog.Nop(),
	}

	result := service.extractMacroContext(nil)

	// Should return empty map, not panic
	if result == nil {
		t.Fatal("Expected empty map for nil input, got nil")
	}

	if len(result) != 0 {
		t.Errorf("Expected empty map, got %d keys", len(result))
	}
}

// TestExtractMacroContext_InvalidMacroContextType tests invalid macro_context type
func TestExtractMacroContext_InvalidMacroContextType(t *testing.T) {
	service := &Service{
		logger: zerolog.Nop(),
	}

	// macro_context is not a map (invalid)
	enrichmentData := map[string]interface{}{
		"profile_overview": map[string]interface{}{
			"legal_name": "Test Company",
		},
		"macro_context": "invalid_string_instead_of_map",
	}

	result := service.extractMacroContext(enrichmentData)

	// Should return empty map for invalid type
	if result == nil {
		t.Fatal("Expected empty map for invalid type, got nil")
	}

	if len(result) != 0 {
		t.Errorf("Expected empty map for invalid type, got %d keys", len(result))
	}
}

// TestMacroContext_FullJSONStructure tests the complete macro_context structure
func TestMacroContext_FullJSONStructure(t *testing.T) {
	service := &Service{
		logger: zerolog.Nop(),
	}

	// Full macro_context structure as would be returned by enrichment
	enrichmentData := map[string]interface{}{
		"macro_context": map[string]interface{}{
			"economic_indicators": map[string]interface{}{
				"country":               "Brazil",
				"gdp_growth":            "+2.5% (2025 forecast)",
				"inflation_rate":        "IPCA: 4.8% a.a.",
				"interest_rate":         "SELIC: 11.75%",
				"exchange_rate":         "USD/BRL: 5.20",
				"unemployment_rate":     "7.2%",
				"political_stability":   "Moderate risk",
				"economic_outlook":      "Moderate growth",
				"recent_policy_changes": []interface{}{"Tax reform 2025", "Carbon law"},
			},
			"industry_trends": map[string]interface{}{
				"industry_sector":      "Agribusiness Tech",
				"growth_rate":          "+12% CAGR",
				"key_trends":           []interface{}{"IoT", "Sustainability", "Digital transformation"},
				"technology_adoption":  "High cloud/AI adoption",
				"market_concentration": "Fragmented",
				"barriers_to_entry":    "Medium",
				"mergers_acquisitions": []interface{}{"Company X acquired Y"},
			},
			"regulatory_landscape": map[string]interface{}{
				"recent_regulations":      []interface{}{"Lei do Agro 2025"},
				"upcoming_changes":        []interface{}{"Carbon tax proposal"},
				"compliance_requirements": "ISO 9001",
				"industry_standards":      []interface{}{"ISO 9001", "ISO 14001"},
			},
			"market_signals": map[string]interface{}{
				"commodity_prices":    []interface{}{"Steel +12% YoY"},
				"supply_chain_status": "Moderate delays",
				"consumer_sentiment":  "Cautious",
				"competitor_activity": []interface{}{"Competitor X new product"},
				"emerging_threats":    []interface{}{"Low-cost Chinese entrant"},
			},
			"data_sources": []interface{}{"url1", "url2"},
			"last_updated": "2025-11-22",
		},
	}

	result := service.extractMacroContext(enrichmentData)

	// Verify all sections are present
	sections := []string{"economic_indicators", "industry_trends", "regulatory_landscape", "market_signals"}
	for _, section := range sections {
		if _, ok := result[section]; !ok {
			t.Errorf("Expected section '%s' to be present in macro_context", section)
		}
	}

	// Verify data_sources and last_updated
	if _, ok := result["data_sources"]; !ok {
		t.Error("Expected 'data_sources' to be present")
	}
	if _, ok := result["last_updated"]; !ok {
		t.Error("Expected 'last_updated' to be present")
	}
}
