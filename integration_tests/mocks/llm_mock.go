package mocks

import (
	"backend_v3/llm"
	"context"
	"encoding/json"

	"github.com/sony/gobreaker"
)

// MockLLMClient is a mock implementation of llm.Client for testing
type MockLLMClient struct {
	llm.Client
	// Response templates for different scenarios
	EnrichmentResponse string
	AnalysisResponse   string
	ShouldFail         bool
	CallCount          int
}

// NewMockLLMClient creates a new mock LLM client with predefined responses
func NewMockLLMClient() *llm.Client {
	// Return a real client structure but with mock base URL
	// In real tests, we'd intercept HTTP calls or use a test server
	// For now, this returns a client that won't make actual API calls
	// if the mock server URL is unreachable
	return llm.NewClientWithBaseURL("mock-api-key", "http://localhost:0")
}

// NewMockLLMClientWithResponses creates a mock with custom responses
func NewMockLLMClientWithResponses(enrichmentJSON, analysisJSON string) *MockLLMClient {
	return &MockLLMClient{
		EnrichmentResponse: enrichmentJSON,
		AnalysisResponse:   analysisJSON,
		ShouldFail:         false,
		CallCount:          0,
	}
}

// getDefaultEnrichmentResponse returns a valid enrichment profile JSON
func getDefaultEnrichmentResponse() string {
	profile := map[string]interface{}{
		"profile_overview": map[string]string{
			"legal_name":      "Test Company LTDA",
			"website":         "https://testcompany.com",
			"foundation_year": "2020",
			"headquarters":    "São Paulo, Brazil",
		},
		"market_position": map[string]string{
			"sector":            "Technology",
			"target_audience":   "B2B SaaS",
			"value_proposition": "Cloud-based solutions for enterprise",
		},
		"financials": map[string]string{
			"employees_range":  "50-200",
			"revenue_estimate": "R$ 5-10M annually",
			"business_model":   "Subscription-based SaaS",
		},
		"competitive_landscape": map[string]interface{}{
			"competitors": []string{
				"Competitor A",
				"Competitor B",
				"Competitor C",
			},
			"market_share_status": "Emerging player with 5% market share",
		},
		"strategic_assessment": map[string]interface{}{
			"digital_maturity": 7,
			"strengths": []string{
				"Strong technical team",
				"Innovative product",
				"Good customer retention",
			},
			"weaknesses": []string{
				"Limited marketing budget",
				"Small sales team",
				"Dependency on single market",
			},
		},
		"macro_context": map[string]interface{}{
			"economic_indicators": map[string]interface{}{
				"country":              "Brazil",
				"gdp_growth":           "+2.5% (2025 forecast)",
				"inflation_rate":       "IPCA: 4.8% a.a.",
				"interest_rate":        "SELIC: 11.75%",
				"exchange_rate":        "USD/BRL: 5.20",
				"unemployment_rate":    "7.2%",
				"political_stability":  "Moderate risk",
				"economic_outlook":     "Moderate growth expected",
				"recent_policy_changes": []string{"Tax reform 2025"},
			},
			"industry_trends": map[string]interface{}{
				"industry_sector":       "Technology",
				"growth_rate":           "+15% CAGR",
				"key_trends":            []string{"AI adoption", "Cloud migration", "Digital transformation"},
				"technology_adoption":   "High",
				"market_concentration":  "Fragmented",
				"barriers_to_entry":     "Medium",
				"mergers_acquisitions":  []string{},
			},
			"regulatory_landscape": map[string]interface{}{
				"recent_regulations":      []string{"LGPD enforcement"},
				"upcoming_changes":        []string{},
				"compliance_requirements": "LGPD, ISO 27001",
				"industry_standards":      []string{"ISO 27001"},
			},
			"market_signals": map[string]interface{}{
				"commodity_prices":    []string{},
				"supply_chain_status": "Normal",
				"consumer_sentiment":  "Cautiously optimistic",
				"competitor_activity": []string{},
				"emerging_threats":    []string{},
			},
			"data_sources":  []string{"web_search", "industry_reports"},
			"last_updated":  "2025-01-15",
		},
	}

	bytes, _ := json.Marshal(profile)
	return string(bytes)
}

// getDefaultAnalysisResponse returns a valid analysis JSON (simplified)
func getDefaultAnalysisResponse() string {
	analysis := map[string]interface{}{
		"pestel": map[string]interface{}{
			"political": []map[string]string{
				{"factor": "Regulatory stability", "impact": "Medium", "description": "Moderate political risk"},
			},
			"economic": []map[string]string{
				{"factor": "Economic growth", "impact": "High", "description": "Positive growth outlook"},
			},
			"social": []map[string]string{
				{"factor": "Digital adoption", "impact": "High", "description": "High technology adoption"},
			},
			"technological": []map[string]string{
				{"factor": "Cloud infrastructure", "impact": "High", "description": "Strong cloud ecosystem"},
			},
			"environmental": []map[string]string{
				{"factor": "Sustainability", "impact": "Low", "description": "Growing ESG awareness"},
			},
			"legal": []map[string]string{
				{"factor": "LGPD compliance", "impact": "High", "description": "Strict data protection laws"},
			},
		},
		"porter": map[string]interface{}{
			"competitive_rivalry": map[string]string{
				"level":       "High",
				"description": "Multiple established competitors",
			},
			"supplier_power": map[string]string{
				"level":       "Medium",
				"description": "Cloud providers have moderate power",
			},
			"buyer_power": map[string]string{
				"level":       "Medium",
				"description": "Enterprise customers have negotiation power",
			},
			"threat_of_substitutes": map[string]string{
				"level":       "Medium",
				"description": "Alternative solutions exist",
			},
			"threat_of_new_entrants": map[string]string{
				"level":       "Medium",
				"description": "Moderate barriers to entry",
			},
		},
		"swot": map[string]interface{}{
			"strengths": []string{
				"Strong technical team",
				"Innovative product",
			},
			"weaknesses": []string{
				"Limited marketing budget",
			},
			"opportunities": []string{
				"Growing market",
				"Digital transformation trend",
			},
			"threats": []string{
				"Intense competition",
			},
		},
	}

	bytes, _ := json.Marshal(analysis)
	return string(bytes)
}

// Call intercepts LLM calls and returns mock data
func (m *MockLLMClient) Call(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	m.CallCount++

	if m.ShouldFail {
		return nil, gobreaker.ErrOpenState
	}

	// Determine which response to return based on request content
	var content string
	if containsEnrichmentKeywords(req) {
		content = m.EnrichmentResponse
	} else {
		content = m.AnalysisResponse
	}

	return &llm.Response{
		Content: content,
		Sources: []llm.Source{
			{URL: "https://mock-source.com", Title: "Mock Source", Snippet: "Mock data"},
		},
		Tokens: 1000,
		Model:  req.Model,
	}, nil
}

// containsEnrichmentKeywords checks if the request is for enrichment
func containsEnrichmentKeywords(req *llm.Request) bool {
	for _, msg := range req.Messages {
		if len(msg.Content) > 0 && (
			len(req.Tools) > 0 || // Enrichment uses tools
			req.SystemPrompt == "You are a JSON-only Corporate Intelligence Agent.") {
			return true
		}
	}
	return false
}
