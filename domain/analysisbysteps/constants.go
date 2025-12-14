package analysisbysteps

// FrameworkMeta contains metadata for each analysis framework step
type FrameworkMeta struct {
	Code         string
	Name         string
	GuidanceText string // Human checkpoint reflection text
}

// FrameworkOrder defines the 14-step analysis sequence (0-13)
var FrameworkOrder = []FrameworkMeta{
	{Code: "challenge_refinement", Name: "Refinamento do Desafio", GuidanceText: "Revise se o desafio está claro e específico. Este é realmente o problema ou apenas um sintoma?"},
	{Code: "pestel", Name: "Análise PESTEL", GuidanceText: "Considere quais fatores externos realmente impactam este negócio. Algum fator foi ignorado ou superestimado?"},
	{Code: "porter", Name: "5 Forças de Porter", GuidanceText: "Avalie a intensidade competitiva. Os concorrentes listados estão corretos? Algum foi esquecido?"},
	{Code: "benchmarking", Name: "Benchmarking", GuidanceText: "Os players comparados são realmente relevantes? As métricas fazem sentido para este contexto?"},
	{Code: "swot", Name: "Análise SWOT", GuidanceText: "As forças listadas realmente geram valor? As fraquezas são críticas para o desafio?"},
	{Code: "swotcross", Name: "SWOT Cruzado", GuidanceText: "As estratégias cruzadas são viáveis? Priorize as que atacam o desafio diretamente."},
	{Code: "tam_sam_som", Name: "TAM-SAM-SOM", GuidanceText: "O dimensionamento de mercado está realista? O SOM é operacionalmente alcançável?"},
	{Code: "blue_ocean", Name: "Blue Ocean", GuidanceText: "A curva de valor proposta realmente diferencia? O cliente pagaria por isso?"},
	{Code: "growth_hacking", Name: "Growth Hacking", GuidanceText: "As táticas de crescimento são aplicáveis ao estágio atual da empresa?"},
	{Code: "scenarios", Name: "Cenários", GuidanceText: "Os cenários cobrem riscos relevantes? Há plano de contingência para o pior caso?"},
	{Code: "decision_matrix", Name: "Matriz de Decisão", GuidanceText: "Os critérios de decisão refletem as prioridades reais? Os pesos estão calibrados?"},
	{Code: "okrs", Name: "OKRs", GuidanceText: "Os objetivos são ambiciosos mas alcançáveis? Os key results são mensuráveis?"},
	{Code: "bsc", Name: "Balanced Scorecard", GuidanceText: "As perspectivas estão balanceadas? Os indicadores são acionáveis?"},
	{Code: "synthesis", Name: "Síntese Executiva", GuidanceText: "A síntese captura as principais conclusões? As recomendações são claras e priorizadas?"},
}

// GetStepNumber returns the step number (0-13) for a given framework code, or -1 if not found
func GetStepNumber(frameworkCode string) int {
	for i, meta := range FrameworkOrder {
		if meta.Code == frameworkCode {
			return i
		}
	}
	return -1
}

// GetFrameworkMeta returns the metadata for a given framework code, or nil if not found
func GetFrameworkMeta(frameworkCode string) *FrameworkMeta {
	for i := range FrameworkOrder {
		if FrameworkOrder[i].Code == frameworkCode {
			return &FrameworkOrder[i]
		}
	}
	return nil
}

// TotalSteps returns the total number of framework steps
func TotalSteps() int {
	return len(FrameworkOrder)
}
