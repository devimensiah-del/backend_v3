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

	// SWOT: Max 4 items per quadrant (now using SWOTItem structure)
	v.trimSWOTItemArray(&analysis.SWOT.Strengths, 4, "SWOT.Strengths")
	v.trimSWOTItemArray(&analysis.SWOT.Weaknesses, 4, "SWOT.Weaknesses")
	v.trimSWOTItemArray(&analysis.SWOT.Opportunities, 4, "SWOT.Opportunities")
	v.trimSWOTItemArray(&analysis.SWOT.Threats, 4, "SWOT.Threats")

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

	// OKRs: Exactly 3 quarters (Q1, Q2, Q3)
	if len(analysis.OKRs.Quarters) > 3 {
		v.logger.Warn().Msg("OKRs exceeded 3 quarters, trimming to Q1-Q3")
		analysis.OKRs.Quarters = analysis.OKRs.Quarters[:3]
	}

	// Benchmarking: Max 3 each
	// These fields now exist in types.go
	v.trimStringArray(&analysis.Benchmarking.PerformanceGaps, 3, "Benchmarking.Gaps")
	v.trimStringArray(&analysis.Benchmarking.BestPractices, 3, "Benchmarking.Practices")

	// Growth Hacking: Validate LEAP and SCALE loops (deprecated fields are optional)
	// Validate LEAP Loop (Acquisition) - 4 steps expected
	if len(analysis.GrowthHacking.LeapLoop.Steps) > 4 {
		v.logger.Warn().Msg("LEAP Loop exceeded 4 steps, trimming")
		analysis.GrowthHacking.LeapLoop.Steps = analysis.GrowthHacking.LeapLoop.Steps[:4]
	}
	// Validate SCALE Loop (Monetization) - 4 steps expected
	if len(analysis.GrowthHacking.ScaleLoop.Steps) > 4 {
		v.logger.Warn().Msg("SCALE Loop exceeded 4 steps, trimming")
		analysis.GrowthHacking.ScaleLoop.Steps = analysis.GrowthHacking.ScaleLoop.Steps[:4]
	}

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

// trimSWOTItemArray trims a SWOTItem array to maxItems
func (v *ContentValidator) trimSWOTItemArray(arr *[]SWOTItem, maxItems int, fieldName string) {
	if arr == nil {
		return
	}
	if len(*arr) > maxItems {
		v.logger.Warn().
			Str("field", fieldName).
			Int("original", len(*arr)).
			Int("trimmed_to", maxItems).
			Msg("Content exceeded limits, trimming SWOTItem array")
		*arr = (*arr)[:maxItems]
	}
}
