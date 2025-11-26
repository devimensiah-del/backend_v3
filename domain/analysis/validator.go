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

// ValidateAndNormalize enforces content limits
// NOTE: PDF generation is currently disabled, so limits are relaxed.
// These limits are now more generous (10 items) to allow richer content.
func (v *ContentValidator) ValidateAndNormalize(analysis *Analysis) {
	// PESTEL: Max 10 items per category (relaxed from 3)
	v.trimStringArray(&analysis.PESTEL.Political, 10, "PESTEL.Political")
	v.trimStringArray(&analysis.PESTEL.Economic, 10, "PESTEL.Economic")
	v.trimStringArray(&analysis.PESTEL.Social, 10, "PESTEL.Social")
	v.trimStringArray(&analysis.PESTEL.Technological, 10, "PESTEL.Technological")
	v.trimStringArray(&analysis.PESTEL.Environmental, 10, "PESTEL.Environmental")
	v.trimStringArray(&analysis.PESTEL.Legal, 10, "PESTEL.Legal")

	// SWOT: Max 10 items per quadrant (relaxed from 4)
	v.trimSWOTItemArray(&analysis.SWOT.Strengths, 10, "SWOT.Strengths")
	v.trimSWOTItemArray(&analysis.SWOT.Weaknesses, 10, "SWOT.Weaknesses")
	v.trimSWOTItemArray(&analysis.SWOT.Opportunities, 10, "SWOT.Opportunities")
	v.trimSWOTItemArray(&analysis.SWOT.Threats, 10, "SWOT.Threats")

	// Blue Ocean: Max 10 per ERRC category (relaxed from 3)
	v.trimStringArray(&analysis.BlueOcean.Eliminate, 10, "BlueOcean.Eliminate")
	v.trimStringArray(&analysis.BlueOcean.Reduce, 10, "BlueOcean.Reduce")
	v.trimStringArray(&analysis.BlueOcean.Raise, 10, "BlueOcean.Raise")
	v.trimStringArray(&analysis.BlueOcean.Create, 10, "BlueOcean.Create")

	// BSC: Max 10 per perspective (relaxed from 2)
	v.trimStringArray(&analysis.BSC.Financial, 10, "BSC.Financial")
	v.trimStringArray(&analysis.BSC.Customer, 10, "BSC.Customer")
	v.trimStringArray(&analysis.BSC.Internal, 10, "BSC.Internal")
	v.trimStringArray(&analysis.BSC.LearningGrowth, 10, "BSC.LearningGrowth")

	// OKRs: Support both V2 (Plan90Days) and V1 (Quarters) formats
	// V2: Plan90Days should be exactly 3 months
	if len(analysis.OKRs.Plan90Days) > 3 {
		v.logger.Warn().Msg("Plan90Days exceeded 3 months, trimming")
		analysis.OKRs.Plan90Days = analysis.OKRs.Plan90Days[:3]
	}
	// V1 Legacy: Max 4 quarters (relaxed from 3)
	if len(analysis.OKRs.Quarters) > 4 {
		v.logger.Warn().Msg("OKRs exceeded 4 quarters, trimming")
		analysis.OKRs.Quarters = analysis.OKRs.Quarters[:4]
	}

	// Benchmarking: Max 10 each (relaxed from 3)
	v.trimStringArray(&analysis.Benchmarking.PerformanceGaps, 10, "Benchmarking.Gaps")
	v.trimStringArray(&analysis.Benchmarking.BestPractices, 10, "Benchmarking.Practices")

	// Growth Hacking: Validate LEAP and SCALE loops - max 8 steps (relaxed from 4)
	if len(analysis.GrowthHacking.LeapLoop.Steps) > 8 {
		v.logger.Warn().Msg("LEAP Loop exceeded 8 steps, trimming")
		analysis.GrowthHacking.LeapLoop.Steps = analysis.GrowthHacking.LeapLoop.Steps[:8]
	}
	if len(analysis.GrowthHacking.ScaleLoop.Steps) > 8 {
		v.logger.Warn().Msg("SCALE Loop exceeded 8 steps, trimming")
		analysis.GrowthHacking.ScaleLoop.Steps = analysis.GrowthHacking.ScaleLoop.Steps[:8]
	}

	// Scenarios: Early Warnings - max 10 (relaxed from 3)
	v.trimStringArray(&analysis.Scenarios.EarlyWarningSignals, 10, "Scenarios.EarlyWarningSignals")
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
