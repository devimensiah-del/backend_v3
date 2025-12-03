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
	// Validate PESTEL framework
	var pestel PESTELAnalysis
	if err := analysis.GetFramework("pestel", &pestel); err == nil {
		v.trimStringArray(&pestel.Political, 10, "PESTEL.Political")
		v.trimStringArray(&pestel.Economic, 10, "PESTEL.Economic")
		v.trimStringArray(&pestel.Social, 10, "PESTEL.Social")
		v.trimStringArray(&pestel.Technological, 10, "PESTEL.Technological")
		v.trimStringArray(&pestel.Environmental, 10, "PESTEL.Environmental")
		v.trimStringArray(&pestel.Legal, 10, "PESTEL.Legal")
		analysis.SetFramework("pestel", pestel)
	}

	// Validate SWOT framework
	var swot SWOTAnalysis
	if err := analysis.GetFramework("swot", &swot); err == nil {
		v.trimSWOTItemArray(&swot.Strengths, 10, "SWOT.Strengths")
		v.trimSWOTItemArray(&swot.Weaknesses, 10, "SWOT.Weaknesses")
		v.trimSWOTItemArray(&swot.Opportunities, 10, "SWOT.Opportunities")
		v.trimSWOTItemArray(&swot.Threats, 10, "SWOT.Threats")
		analysis.SetFramework("swot", swot)
	}

	// Validate Blue Ocean framework
	var blueOcean BlueOceanAnalysis
	if err := analysis.GetFramework("blue_ocean", &blueOcean); err == nil {
		v.trimStringArray(&blueOcean.Eliminate, 10, "BlueOcean.Eliminate")
		v.trimStringArray(&blueOcean.Reduce, 10, "BlueOcean.Reduce")
		v.trimStringArray(&blueOcean.Raise, 10, "BlueOcean.Raise")
		v.trimStringArray(&blueOcean.Create, 10, "BlueOcean.Create")
		analysis.SetFramework("blue_ocean", blueOcean)
	}

	// Validate BSC framework
	var bsc BalancedScorecardAnalysis
	if err := analysis.GetFramework("bsc", &bsc); err == nil {
		v.trimStringArray(&bsc.Financial, 10, "BSC.Financial")
		v.trimStringArray(&bsc.Customer, 10, "BSC.Customer")
		v.trimStringArray(&bsc.Internal, 10, "BSC.Internal")
		v.trimStringArray(&bsc.LearningGrowth, 10, "BSC.LearningGrowth")
		analysis.SetFramework("bsc", bsc)
	}

	// Validate OKRs framework
	var okrs OKRAnalysis
	if err := analysis.GetFramework("okrs", &okrs); err == nil {
		// V2: Plan90Days should be exactly 3 months
		if len(okrs.Plan90Days) > 3 {
			v.logger.Warn().Msg("Plan90Days exceeded 3 months, trimming")
			okrs.Plan90Days = okrs.Plan90Days[:3]
		}
		// V1 Legacy: Max 4 quarters (relaxed from 3)
		if len(okrs.Quarters) > 4 {
			v.logger.Warn().Msg("OKRs exceeded 4 quarters, trimming")
			okrs.Quarters = okrs.Quarters[:4]
		}
		analysis.SetFramework("okrs", okrs)
	}

	// Validate Benchmarking framework
	var benchmarking BenchmarkingAnalysis
	if err := analysis.GetFramework("benchmarking", &benchmarking); err == nil {
		v.trimStringArray(&benchmarking.PerformanceGaps, 10, "Benchmarking.Gaps")
		v.trimStringArray(&benchmarking.BestPractices, 10, "Benchmarking.Practices")
		analysis.SetFramework("benchmarking", benchmarking)
	}

	// Validate Growth Hacking framework
	var growthHacking GrowthHackingAnalysis
	if err := analysis.GetFramework("growth_hacking", &growthHacking); err == nil {
		if len(growthHacking.LeapLoop.Steps) > 8 {
			v.logger.Warn().Msg("LEAP Loop exceeded 8 steps, trimming")
			growthHacking.LeapLoop.Steps = growthHacking.LeapLoop.Steps[:8]
		}
		if len(growthHacking.ScaleLoop.Steps) > 8 {
			v.logger.Warn().Msg("SCALE Loop exceeded 8 steps, trimming")
			growthHacking.ScaleLoop.Steps = growthHacking.ScaleLoop.Steps[:8]
		}
		analysis.SetFramework("growth_hacking", growthHacking)
	}

	// Validate Scenarios framework
	var scenarios ScenarioAnalysis
	if err := analysis.GetFramework("scenarios", &scenarios); err == nil {
		v.trimStringArray(&scenarios.EarlyWarningSignals, 10, "Scenarios.EarlyWarningSignals")
		analysis.SetFramework("scenarios", scenarios)
	}
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
