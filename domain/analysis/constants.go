package analysis

// Framework code constants for consistent referencing across the codebase.
// Use these instead of magic strings to ensure consistency and enable refactoring.
const (
	FrameworkPESTEL         = "pestel"
	FrameworkPorter         = "porter"
	FrameworkSWOT           = "swot"
	FrameworkOKRs           = "okrs"
	FrameworkTAMSAMSOM      = "tam_sam_som"
	FrameworkBenchmarking   = "benchmarking"
	FrameworkBlueOcean      = "blue_ocean"
	FrameworkGrowthHacking  = "growth_hacking"
	FrameworkScenarios      = "scenarios"
	FrameworkBSC            = "bsc"
	FrameworkDecisionMatrix = "decision_matrix"
	FrameworkSynthesis      = "synthesis"
)

// AllFrameworks returns all framework codes in execution order
var AllFrameworks = []string{
	FrameworkPESTEL,
	FrameworkPorter,
	FrameworkTAMSAMSOM,
	FrameworkSWOT,
	FrameworkBenchmarking,
	FrameworkBlueOcean,
	FrameworkGrowthHacking,
	FrameworkScenarios,
	FrameworkDecisionMatrix,
	FrameworkOKRs,
	FrameworkBSC,
	FrameworkSynthesis,
}

// CriticalFrameworks are frameworks that must complete for a valid analysis
var CriticalFrameworks = []string{
	FrameworkPESTEL,
	FrameworkPorter,
	FrameworkSWOT,
	FrameworkTAMSAMSOM,
}
