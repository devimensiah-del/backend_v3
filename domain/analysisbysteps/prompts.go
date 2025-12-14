package analysisbysteps

import "backend_v3/llm"

// getPromptForFramework returns the appropriate prompt template for a given framework code
func getPromptForFramework(frameworkCode string) string {
	switch frameworkCode {
	case "challenge_refinement":
		return llm.ChallengeRefinementPrompt
	case "pestel":
		return llm.FrameworkPESTELPrompt
	case "porter":
		return llm.FrameworkPorterPrompt
	case "benchmarking":
		return llm.FrameworkBenchmarkingPrompt
	case "swot":
		return llm.FrameworkSWOTPrompt
	case "swotcross":
		// Use SWOT prompt for cross-analysis (no separate prompt exists yet)
		return llm.FrameworkSWOTPrompt
	case "tam_sam_som":
		return llm.FrameworkTamSamSomPrompt
	case "blue_ocean":
		return llm.FrameworkBlueOceanPrompt
	case "growth_hacking":
		return llm.FrameworkGrowthHackingPrompt
	case "scenarios":
		return llm.FrameworkScenariosPrompt
	case "decision_matrix":
		return llm.FrameworkDecisionMatrixPrompt
	case "okrs":
		return llm.FrameworkOKRsPrompt
	case "bsc":
		return llm.FrameworkBSCPrompt
	case "synthesis":
		return llm.SynthesisPrompt
	default:
		// Return a generic prompt if framework code is unknown
		return "Analyze the following context and provide structured insights:\n\n{{COMPANY_DATA}}\n\n{{CHALLENGE_CONTEXT}}"
	}
}
