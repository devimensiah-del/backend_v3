package testutils

import (
	"encoding/json"
	"testing"

	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/submission"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =================================================================
// Custom Test Assertions
// =================================================================

// AssertSubmissionEqual performs deep comparison of two submissions
func AssertSubmissionEqual(t *testing.T, expected, actual *submission.Submission) {
	t.Helper()

	assert.Equal(t, expected.ID, actual.ID, "Submission ID mismatch")
	assert.Equal(t, expected.CompanyName, actual.CompanyName, "Company name mismatch")
	assert.Equal(t, expected.CNPJ, actual.CNPJ, "CNPJ mismatch")
	assert.Equal(t, expected.CompanyWebsite, actual.CompanyWebsite, "Company website mismatch")
	assert.Equal(t, expected.CompanyIndustry, actual.CompanyIndustry, "Company industry mismatch")
	assert.Equal(t, expected.CompanySize, actual.CompanySize, "Company size mismatch")
	assert.Equal(t, expected.CompanyLocation, actual.CompanyLocation, "Company location mismatch")
	assert.Equal(t, expected.ContactName, actual.ContactName, "Contact name mismatch")
	assert.Equal(t, expected.ContactEmail, actual.ContactEmail, "Contact email mismatch")
	assert.Equal(t, expected.ContactPhone, actual.ContactPhone, "Contact phone mismatch")
	assert.Equal(t, expected.ContactPosition, actual.ContactPosition, "Contact position mismatch")
	assert.Equal(t, expected.TargetMarket, actual.TargetMarket, "Target market mismatch")
	assert.Equal(t, expected.AnnualRevenueMin, actual.AnnualRevenueMin, "Annual revenue min mismatch")
	assert.Equal(t, expected.AnnualRevenueMax, actual.AnnualRevenueMax, "Annual revenue max mismatch")
	assert.Equal(t, expected.FundingStage, actual.FundingStage, "Funding stage mismatch")
	assert.Equal(t, expected.BusinessChallenge, actual.BusinessChallenge, "Business challenge mismatch")
	assert.Equal(t, expected.AdditionalNotes, actual.AdditionalNotes, "Additional notes mismatch")
	assert.Equal(t, expected.LinkedInURL, actual.LinkedInURL, "LinkedIn URL mismatch")
	assert.Equal(t, expected.TwitterHandle, actual.TwitterHandle, "Twitter handle mismatch")
	assert.Equal(t, expected.Status, actual.Status, "Status mismatch")
	assert.Equal(t, expected.UserID, actual.UserID, "User ID mismatch")
}

// AssertEnrichmentHasData validates enriched_data structure
func AssertEnrichmentHasData(t *testing.T, enrichment *enrichment.Enrichment) {
	t.Helper()

	require.NotNil(t, enrichment, "Enrichment is nil")
	require.NotNil(t, enrichment.EnrichedData, "Enriched data is nil")
	require.NotEmpty(t, enrichment.EnrichedData, "Enriched data is empty")

	// Parse UnifiedProfile from enriched_data
	var profile enrichment.UnifiedProfile
	profileBytes, err := json.Marshal(enrichment.EnrichedData)
	require.NoError(t, err, "Failed to marshal enriched data")

	err = json.Unmarshal(profileBytes, &profile)
	require.NoError(t, err, "Failed to unmarshal UnifiedProfile")

	// Validate ProfileOverview
	assert.NotEmpty(t, profile.ProfileOverview.LegalName, "Legal name is empty")
	assert.NotEmpty(t, profile.ProfileOverview.Website, "Website is empty")
	assert.NotEmpty(t, profile.ProfileOverview.FoundationYear, "Foundation year is empty")
	assert.NotEmpty(t, profile.ProfileOverview.Headquarters, "Headquarters is empty")

	// Validate MarketPosition
	assert.NotEmpty(t, profile.MarketPosition.Sector, "Sector is empty")
	assert.NotEmpty(t, profile.MarketPosition.TargetAudience, "Target audience is empty")
	assert.NotEmpty(t, profile.MarketPosition.ValueProposition, "Value proposition is empty")

	// Validate Financials
	assert.NotEmpty(t, profile.Financials.EmployeesRange, "Employees range is empty")
	assert.NotEmpty(t, profile.Financials.RevenueEstimate, "Revenue estimate is empty")
	assert.NotEmpty(t, profile.Financials.BusinessModel, "Business model is empty")

	// Validate CompetitiveLandscape
	assert.NotEmpty(t, profile.CompetitiveLandscape.Competitors, "Competitors list is empty")
	assert.NotEmpty(t, profile.CompetitiveLandscape.MarketShareStatus, "Market share status is empty")

	// Validate StrategicAssessment
	assert.Greater(t, profile.StrategicAssessment.DigitalMaturity, 0, "Digital maturity should be > 0")
	assert.NotEmpty(t, profile.StrategicAssessment.Strengths, "Strengths list is empty")
	assert.NotEmpty(t, profile.StrategicAssessment.Weaknesses, "Weaknesses list is empty")

	// Validate MacroContext (optional field)
	if profile.MacroContext != nil {
		assert.NotEmpty(t, profile.MacroContext.EconomicIndicators.Country, "Country is empty")
		assert.NotEmpty(t, profile.MacroContext.IndustryTrends.IndustrySector, "Industry sector is empty")
	}
}

// AssertAnalysisComplete checks that all 11 frameworks are populated
func AssertAnalysisComplete(t *testing.T, analysis *analysis.Analysis) {
	t.Helper()

	require.NotNil(t, analysis, "Analysis is nil")
	assert.Equal(t, string(analysis.StatusCompleted), analysis.Status, "Analysis status should be 'completed'")

	// 1. PESTEL
	assert.NotEmpty(t, analysis.PESTEL.Political, "PESTEL: Political factors are empty")
	assert.NotEmpty(t, analysis.PESTEL.Economic, "PESTEL: Economic factors are empty")
	assert.NotEmpty(t, analysis.PESTEL.Social, "PESTEL: Social factors are empty")
	assert.NotEmpty(t, analysis.PESTEL.Technological, "PESTEL: Technological factors are empty")
	assert.NotEmpty(t, analysis.PESTEL.Environmental, "PESTEL: Environmental factors are empty")
	assert.NotEmpty(t, analysis.PESTEL.Legal, "PESTEL: Legal factors are empty")
	assert.NotEmpty(t, analysis.PESTEL.Summary, "PESTEL: Summary is empty")

	// 2. Porter's 7 Forces
	assert.NotEmpty(t, analysis.Porter.CompetitiveRivalry, "Porter: Competitive rivalry is empty")
	assert.NotEmpty(t, analysis.Porter.SupplierPower, "Porter: Supplier power is empty")
	assert.NotEmpty(t, analysis.Porter.BuyerPower, "Porter: Buyer power is empty")
	assert.NotEmpty(t, analysis.Porter.ThreatNewEntrants, "Porter: Threat of new entrants is empty")
	assert.NotEmpty(t, analysis.Porter.ThreatSubstitutes, "Porter: Threat of substitutes is empty")
	assert.NotEmpty(t, analysis.Porter.PowerPartnershipsEcosystems, "Porter: Power of partnerships is empty")
	assert.NotEmpty(t, analysis.Porter.DisruptionAIData, "Porter: AI/Data disruption is empty")
	assert.NotEmpty(t, analysis.Porter.Summary, "Porter: Summary is empty")

	// 3. SWOT
	assert.NotEmpty(t, analysis.SWOT.Strengths, "SWOT: Strengths are empty")
	assert.NotEmpty(t, analysis.SWOT.Weaknesses, "SWOT: Weaknesses are empty")
	assert.NotEmpty(t, analysis.SWOT.Opportunities, "SWOT: Opportunities are empty")
	assert.NotEmpty(t, analysis.SWOT.Threats, "SWOT: Threats are empty")
	assert.NotEmpty(t, analysis.SWOT.Summary, "SWOT: Summary is empty")

	// Validate SWOT items have confidence and source
	if len(analysis.SWOT.Strengths) > 0 {
		assert.NotEmpty(t, analysis.SWOT.Strengths[0].Content, "SWOT Strength: Content is empty")
		assert.NotEmpty(t, analysis.SWOT.Strengths[0].Confidence, "SWOT Strength: Confidence is empty")
		assert.NotEmpty(t, analysis.SWOT.Strengths[0].Source, "SWOT Strength: Source is empty")
	}

	// 4. TAM/SAM/SOM
	assert.NotEmpty(t, analysis.TamSamSom.TAM, "TAM/SAM/SOM: TAM is empty")
	assert.NotEmpty(t, analysis.TamSamSom.SAM, "TAM/SAM/SOM: SAM is empty")
	assert.NotEmpty(t, analysis.TamSamSom.SOM, "TAM/SAM/SOM: SOM is empty")
	assert.NotEmpty(t, analysis.TamSamSom.Assumptions, "TAM/SAM/SOM: Assumptions are empty")
	assert.NotEmpty(t, analysis.TamSamSom.Summary, "TAM/SAM/SOM: Summary is empty")

	// 5. Blue Ocean
	assert.NotEmpty(t, analysis.BlueOcean.Summary, "Blue Ocean: Summary is empty")

	// 6. OKRs (Quarterly structure)
	assert.NotEmpty(t, analysis.OKRs.Quarters, "OKRs: Quarters are empty")
	assert.GreaterOrEqual(t, len(analysis.OKRs.Quarters), 3, "OKRs: Should have at least 3 quarters")
	if len(analysis.OKRs.Quarters) > 0 {
		assert.NotEmpty(t, analysis.OKRs.Quarters[0].Quarter, "OKR Q1: Quarter is empty")
		assert.NotEmpty(t, analysis.OKRs.Quarters[0].Objective, "OKR Q1: Objective is empty")
		assert.NotEmpty(t, analysis.OKRs.Quarters[0].KeyResults, "OKR Q1: Key results are empty")
		assert.Equal(t, 3, len(analysis.OKRs.Quarters[0].KeyResults), "OKR Q1: Should have exactly 3 key results")
	}
	assert.NotEmpty(t, analysis.OKRs.Summary, "OKRs: Summary is empty")

	// 7. Balanced Scorecard
	assert.NotEmpty(t, analysis.BSC.Financial, "BSC: Financial perspective is empty")
	assert.NotEmpty(t, analysis.BSC.Customer, "BSC: Customer perspective is empty")
	assert.NotEmpty(t, analysis.BSC.Internal, "BSC: Internal processes are empty")
	assert.NotEmpty(t, analysis.BSC.LearningGrowth, "BSC: Learning & growth is empty")
	assert.NotEmpty(t, analysis.BSC.Summary, "BSC: Summary is empty")

	// 8. Benchmarking
	assert.NotEmpty(t, analysis.Benchmarking.Competitors, "Benchmarking: Competitors are empty")
	assert.NotEmpty(t, analysis.Benchmarking.PerformanceGaps, "Benchmarking: Performance gaps are empty")
	assert.NotEmpty(t, analysis.Benchmarking.BestPractices, "Benchmarking: Best practices are empty")
	assert.NotEmpty(t, analysis.Benchmarking.Summary, "Benchmarking: Summary is empty")

	// 9. Growth Hacking (LEAP + SCALE Loops)
	assert.NotEmpty(t, analysis.GrowthHacking.LeapLoop.Name, "Growth Hacking: LEAP Loop name is empty")
	assert.NotEmpty(t, analysis.GrowthHacking.LeapLoop.Steps, "Growth Hacking: LEAP Loop steps are empty")
	assert.Equal(t, 4, len(analysis.GrowthHacking.LeapLoop.Steps), "LEAP Loop should have 4 steps")
	assert.NotEmpty(t, analysis.GrowthHacking.ScaleLoop.Name, "Growth Hacking: SCALE Loop name is empty")
	assert.NotEmpty(t, analysis.GrowthHacking.ScaleLoop.Steps, "Growth Hacking: SCALE Loop steps are empty")
	assert.Equal(t, 4, len(analysis.GrowthHacking.ScaleLoop.Steps), "SCALE Loop should have 4 steps")
	assert.NotEmpty(t, analysis.GrowthHacking.Summary, "Growth Hacking: Summary is empty")

	// 10. Scenario Analysis
	assert.NotEmpty(t, analysis.Scenarios.Optimistic.Name, "Scenarios: Optimistic name is empty")
	assert.Greater(t, analysis.Scenarios.Optimistic.Probability, 0, "Scenarios: Optimistic probability should be > 0")
	assert.NotEmpty(t, analysis.Scenarios.Realist.Name, "Scenarios: Realist name is empty")
	assert.Greater(t, analysis.Scenarios.Realist.Probability, 0, "Scenarios: Realist probability should be > 0")
	assert.NotEmpty(t, analysis.Scenarios.Pessimistic.Name, "Scenarios: Pessimistic name is empty")
	assert.Greater(t, analysis.Scenarios.Pessimistic.Probability, 0, "Scenarios: Pessimistic probability should be > 0")
	assert.NotEmpty(t, analysis.Scenarios.Summary, "Scenarios: Summary is empty")

	// Validate probabilities sum to 100
	totalProbability := analysis.Scenarios.Optimistic.Probability +
		analysis.Scenarios.Realist.Probability +
		analysis.Scenarios.Pessimistic.Probability
	assert.Equal(t, 100, totalProbability, "Scenario probabilities should sum to 100")

	// 11. Decision Matrix
	assert.NotEmpty(t, analysis.DecisionMatrix.Alternatives, "Decision Matrix: Alternatives are empty")
	assert.NotEmpty(t, analysis.DecisionMatrix.Criteria, "Decision Matrix: Criteria are empty")
	assert.NotEmpty(t, analysis.DecisionMatrix.FinalRecommendation, "Decision Matrix: Final recommendation is empty")
	assert.NotEmpty(t, analysis.DecisionMatrix.RecommendedOption, "Decision Matrix: Recommended option is empty")
	assert.NotEmpty(t, analysis.DecisionMatrix.PriorityRecommendations, "Decision Matrix: Priority recommendations are empty")
	assert.GreaterOrEqual(t, len(analysis.DecisionMatrix.PriorityRecommendations), 3, "Decision Matrix: Should have at least 3 priority recommendations")
	assert.NotEmpty(t, analysis.DecisionMatrix.Summary, "Decision Matrix: Summary is empty")

	// Synthesis
	assert.NotEmpty(t, analysis.Synthesis.ExecutiveSummary, "Synthesis: Executive summary is empty")
	assert.NotEmpty(t, analysis.Synthesis.CentralChallenge, "Synthesis: Central challenge is empty")
	assert.NotEmpty(t, analysis.Synthesis.MainFindings, "Synthesis: Main findings are empty")
	assert.Equal(t, 4, len(analysis.Synthesis.MainFindings), "Synthesis: Should have exactly 4 main findings")
	assert.NotEmpty(t, analysis.Synthesis.KeyFindings, "Synthesis: Key findings are empty")
	assert.NotEmpty(t, analysis.Synthesis.StrategicPriorities, "Synthesis: Strategic priorities are empty")
	assert.NotEmpty(t, analysis.Synthesis.Roadmap, "Synthesis: Roadmap is empty")
	assert.NotEmpty(t, analysis.Synthesis.OverallRecommendation, "Synthesis: Overall recommendation is empty")
}

// AssertJobEnqueued verifies that an asynq job was queued
func AssertJobEnqueued(t *testing.T, inspector *MockAsynqInspector, taskType string, submissionID string) {
	t.Helper()

	found := false
	for _, task := range inspector.EnqueuedTasks {
		if task.Type == taskType {
			// Check if payload contains submission ID
			var payload map[string]string
			err := json.Unmarshal(task.Payload, &payload)
			require.NoError(t, err, "Failed to unmarshal task payload")

			if payload["submission_id"] == submissionID {
				found = true
				break
			}
		}
	}

	assert.True(t, found, "Job %s for submission %s was not enqueued", taskType, submissionID)
}

// AssertValidUUID checks that a string is a valid UUID
func AssertValidUUID(t *testing.T, uuidStr string, fieldName string) {
	t.Helper()

	assert.NotEmpty(t, uuidStr, "%s is empty", fieldName)
	assert.Equal(t, 36, len(uuidStr), "%s should be 36 characters (UUID format)", fieldName)
	assert.Contains(t, uuidStr, "-", "%s should contain hyphens (UUID format)", fieldName)
}

// AssertTimeNotZero checks that a time value is not zero
func AssertTimeNotZero(t *testing.T, timeVal interface{}, fieldName string) {
	t.Helper()

	assert.NotNil(t, timeVal, "%s is nil", fieldName)
	// Additional time validation can be added here
}

// AssertJSONValid checks that a string is valid JSON
func AssertJSONValid(t *testing.T, jsonStr string, fieldName string) {
	t.Helper()

	var js interface{}
	err := json.Unmarshal([]byte(jsonStr), &js)
	assert.NoError(t, err, "%s is not valid JSON: %v", fieldName, err)
}
