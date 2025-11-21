package analysis

import (
	"github.com/rs/zerolog"
)

type ContentValidator struct {
	logger zerolog.Logger
}

func NewContentValidator(logger zerolog.Logger) *ContentValidator {
	return &ContentValidator{logger: logger}
}

// ValidateAndNormalize enforces content limits to ensure PDF layout safety
func (v *ContentValidator) ValidateAndNormalize(analysis *Analysis) {
	// PESTEL: Max 3 items per category
	v.trimStringArray(&analysis.PESTEL.Political, 3, "PESTEL.Political")
	v.trimStringArray(&analysis.PESTEL.Economic, 3, "PESTEL.Economic")
	v.trimStringArray(&analysis.PESTEL.Social, 3, "PESTEL.Social")
	v.trimStringArray(&analysis.PESTEL.Technological, 3, "PESTEL.Technological")
	v.trimStringArray(&analysis.PESTEL.Environmental, 3, "PESTEL.Environmental")
	v.trimStringArray(&analysis.PESTEL.Legal, 3, "PESTEL.Legal")

	// SWOT: Max 4 items per quadrant
	v.trimStringArray(&analysis.SWOT.Strengths, 4, "SWOT.Strengths")
	v.trimStringArray(&analysis.SWOT.Weaknesses, 4, "SWOT.Weaknesses")
	v.trimStringArray(&analysis.SWOT.Opportunities, 4, "SWOT.Opportunities")
	v.trimStringArray(&analysis.SWOT.Threats, 4, "SWOT.Threats")

	// Blue Ocean: Max 3 per ERRC category
	v.trimStringArray(&analysis.BlueOcean.Eliminate, 3, "BlueOcean.Eliminate")
	v.trimStringArray(&analysis.BlueOcean.Reduce, 3, "BlueOcean.Reduce")
	v.trimStringArray(&analysis.BlueOcean.Raise, 3, "BlueOcean.Raise")
	v.trimStringArray(&analysis.BlueOcean.Create, 3, "BlueOcean.Create")

	// BSC: Max 2 per perspective
	v.trimStringArray(&analysis.BSC.Financial, 2, "BSC.Financial")
	v.trimStringArray(&analysis.BSC.Customer, 2, "BSC.Customer")
	v.trimStringArray(&analysis.BSC.Internal, 2, "BSC.Internal")
	v.trimStringArray(&analysis.BSC.LearningGrowth, 2, "BSC.LearningGrowth")

	// OKRs: Max 2 objectives
	if len(analysis.OKRs.Objectives) > 2 {
		v.logger.Warn().Msg("OKRs exceeded 2 objectives, trimming")
		analysis.OKRs.Objectives = analysis.OKRs.Objectives[:2]
	}

	// Benchmarking: Max 3 each
	// These fields now exist in types.go
	v.trimStringArray(&analysis.Benchmarking.PerformanceGaps, 3, "Benchmarking.Gaps")
	v.trimStringArray(&analysis.Benchmarking.BestPractices, 3, "Benchmarking.Practices")

	// Growth Hacking: Max 3 each
	// These fields now exist in types.go
	v.trimStringArray(&analysis.GrowthHacking.Hypotheses, 3, "GrowthHacking.Hypotheses")
	v.trimStringArray(&analysis.GrowthHacking.Experiments, 3, "GrowthHacking.Experiments")
	v.trimStringArray(&analysis.GrowthHacking.KeyMetrics, 3, "GrowthHacking.KeyMetrics")

	// Scenarios: Early Warnings
	v.trimStringArray(&analysis.Scenarios.EarlyWarningSignals, 3, "Scenarios.EarlyWarningSignals")
}

func (v *ContentValidator) trimStringArray(arr *[]string, maxItems int, fieldName string) {
	if arr == nil {
		return
	}
	if len(*arr) > maxItems {
		v.logger.Warn().
			Str("field", fieldName).
			Int("original", len(*arr)).
			Int("trimmed_to", maxItems).
			Msg("Content exceeded limits, trimming array")
		*arr = (*arr)[:maxItems]
	}
}
