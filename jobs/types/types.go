package types

const (
	TypeAnalysis = "analysis_job"

	// Framework execution job types (V2)
	TypeFrameworkExecution = "framework_execution" // Execute single framework for a company

	// Macroeconomics domain job types
	TypeMacroFetch      = "macro_fetch"       // Fetch single indicator
	TypeMacroRefreshAll = "macro_refresh_all" // Refresh all indicators
)
