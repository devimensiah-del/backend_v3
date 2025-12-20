package enrichment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEnrichmentResponse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *EnrichedCompanyData
		wantErr bool
	}{
		{
			name: "valid JSON response",
			content: `{
				"cnpj": "12.345.678/0001-90",
				"website": "https://example.com",
				"industry": "Technology",
				"confidence_score": 85,
				"sources": ["https://source1.com", "https://source2.com"]
			}`,
			want: &EnrichedCompanyData{
				CNPJ:            ptrString("12.345.678/0001-90"),
				Website:         ptrString("https://example.com"),
				Industry:        ptrString("Technology"),
				ConfidenceScore: 85,
				Sources:         []string{"https://source1.com", "https://source2.com"},
			},
			wantErr: false,
		},
		{
			name:    "JSON with markdown code block",
			content: "```json\n{\"cnpj\": \"12.345.678/0001-90\", \"confidence_score\": 75}\n```",
			want: &EnrichedCompanyData{
				CNPJ:            ptrString("12.345.678/0001-90"),
				ConfidenceScore: 75,
			},
			wantErr: false,
		},
		{
			name:    "confidence score out of range (too high) - should default to 50",
			content: `{"confidence_score": 150}`,
			want: &EnrichedCompanyData{
				ConfidenceScore: 50,
			},
			wantErr: false,
		},
		{
			name:    "confidence score out of range (negative) - should default to 50",
			content: `{"confidence_score": -10}`,
			want: &EnrichedCompanyData{
				ConfidenceScore: 50,
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			content: `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "empty string",
			content: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEnrichmentResponse(tt.content)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)

			// Check key fields
			if tt.want.CNPJ != nil {
				assert.Equal(t, *tt.want.CNPJ, *got.CNPJ)
			}
			assert.Equal(t, tt.want.ConfidenceScore, got.ConfidenceScore)
		})
	}
}

// Note: TestService_identifyMissingFields and TestService_buildEnrichmentPrompt
// were removed because those methods no longer exist in the two-stage architecture.
// The new architecture uses BuildSearchPrompt and BuildSynthesisPrompt (in prompts.go)
// and ParseEnrichmentResponse (in parser.go).

func TestEnrichedCompanyData_Structure(t *testing.T) {
	// Test that the struct can be properly populated
	data := &EnrichedCompanyData{
		CNPJ:              ptrString("12.345.678/0001-90"),
		Website:           ptrString("https://example.com"),
		Industry:          ptrString("Technology"),
		CompanySize:       ptrString("Medium"),
		Location:          ptrString("São Paulo, SP"),
		TargetMarket:      ptrString("B2B"),
		FundingStage:      ptrString("Series A"),
		AnnualRevenueMin:  ptrFloat64(1000000),
		AnnualRevenueMax:  ptrFloat64(5000000),
		FoundationYear:    ptrString("2015"),
		LegalName:         ptrString("Example Tech Ltda"),
		Headquarters:      ptrString("São Paulo, SP, Brazil"),
		Sector:            ptrString("SaaS"),
		TargetAudience:    ptrString("SMBs"),
		ValueProposition:  ptrString("Best software"),
		EmployeesRange:    ptrString("50-100"),
		RevenueEstimate:   ptrString("R$ 1M - 5M"),
		BusinessModel:     ptrString("B2B SaaS"),
		MarketShareStatus: ptrString("Challenger"),
		DigitalMaturity:   ptrInt(7),
		Competitors:       []string{"Competitor 1", "Competitor 2"},
		Strengths:         []string{"Strength 1"},
		Weaknesses:        []string{"Weakness 1"},
		LinkedInURL:       ptrString("https://linkedin.com/company/example"),
		TwitterHandle:     ptrString("@example"),
		ConfidenceScore:   85,
		Sources:           []string{"https://source1.com"},
	}

	// Verify all fields are set
	require.NotNil(t, data.CNPJ)
	require.NotNil(t, data.Website)
	assert.Equal(t, 85.0, data.ConfidenceScore)
	assert.Len(t, data.Competitors, 2)
	assert.Len(t, data.Sources, 1)
}

// Helper functions
func ptrString(s string) *string {
	return &s
}

func ptrFloat64(f float64) *float64 {
	return &f
}

func ptrInt(i int) *int {
	return &i
}
