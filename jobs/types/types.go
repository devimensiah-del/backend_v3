package types

const (
	TypeEnrichment = "enrichment_job"
	TypeAnalysis   = "analysis_job"
	TypeReport     = "report"

	// Macroeconomics domain job types
	TypeMacroFetch      = "macro_fetch"       // Fetch single indicator
	TypeMacroRefreshAll = "macro_refresh_all" // Refresh all indicators
)
