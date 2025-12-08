package wizard

// FrameworkStep represents a single step in the analysis wizard
type FrameworkStep struct {
	Step        int                  `json:"step"`
	Code        string               `json:"code"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Questions   []ClarifyingQuestion `json:"questions"`
}

// ClarifyingQuestion represents a question to guide human review
type ClarifyingQuestion struct {
	ID       string `json:"id"`
	Question string `json:"question"`
}

// FrameworkOrder defines the sequential order of frameworks in the wizard
// Total: 12 steps (0-11) + auto-generated synthesis
var FrameworkOrder = []FrameworkStep{
	{
		Step:        0,
		Code:        "challenge_refinement",
		Name:        "Refinamento do Desafio",
		Description: "Validar se o desafio declarado é o problema real ou apenas um sintoma",
		Questions: []ClarifyingQuestion{
			{ID: "real_problem", Question: "Esse é o verdadeiro problema ou um sintoma?"},
			{ID: "proof_priority", Question: "O que provaria que esse problema é prioritário?"},
			{ID: "metric", Question: "Qual número/métrica mede esse desafio?"},
		},
	},
	{
		Step:        1,
		Code:        "pestel",
		Name:        "PESTEL",
		Description: "Análise do contexto externo e macroeconômico",
		Questions: []ClarifyingQuestion{
			{ID: "factors_apply", Question: "Quais fatores realmente se aplicam a esta empresa?"},
			{ID: "factors_ignore", Question: "Quais fatores podemos ignorar porque são ruído?"},
			{ID: "missing_factors", Question: "Há algum fator externo que a IA não detectou?"},
		},
	},
	{
		Step:        2,
		Code:        "porter",
		Name:        "Porter 5 Forças",
		Description: "Análise de intensidade competitiva usando premissas do PESTEL",
		Questions: []ClarifyingQuestion{
			{ID: "competitors_wrong", Question: "Quais concorrentes a IA errou?"},
			{ID: "competitors_missing", Question: "Quem a IA deixou de fora?"},
			{ID: "threat_profitability", Question: "Qual força realmente ameaça a lucratividade?"},
		},
	},
	{
		Step:        3,
		Code:        "benchmarking",
		Name:        "Benchmarking",
		Description: "Comparação direta com players selecionados via Porter",
		Questions: []ClarifyingQuestion{
			{ID: "players_comparable", Question: "Esses players são realmente comparáveis?"},
			{ID: "volume_vs_quality", Question: "A IA está confundindo volume com qualidade?"},
			{ID: "gap_painful", Question: "Qual gap dói mais hoje?"},
		},
	},
	{
		Step:        4,
		Code:        "swot",
		Name:        "SWOT",
		Description: "Matriz de forças e fraquezas alimentada por dados reais",
		Questions: []ClarifyingQuestion{
			{ID: "strengths_cash", Question: "Quais forças realmente geram caixa?"},
			{ID: "weaknesses_unacceptable", Question: "Quais fraquezas são inaceitáveis para o desafio?"},
			{ID: "opportunities_accessible", Question: "Quais oportunidades são acessíveis?"},
			{ID: "threats_inevitable", Question: "Qual ameaça é inevitável?"},
		},
	},
	{
		Step:        5,
		Code:        "tam_sam_som",
		Name:        "TAM-SAM-SOM",
		Description: "Dimensionamento de mercado usando SWOT para definir escopo",
		Questions: []ClarifyingQuestion{
			{ID: "market_definition", Question: "Estamos definindo mercado por setor, problema ou cliente?"},
			{ID: "som_viable", Question: "Esse SOM é operacionalmente viável?"},
			{ID: "micro_som", Question: "Existe micro-SOM mais lucrativo?"},
		},
	},
	{
		Step:        6,
		Code:        "blue_ocean",
		Name:        "Blue Ocean",
		Description: "Definição da nova curva de valor para sair da competição irrelevante",
		Questions: []ClarifyingQuestion{
			{ID: "red_ocean_exit", Question: "Isso realmente nos tira do oceano vermelho?"},
			{ID: "customer_pay", Question: "O cliente pagaria por esse diferencial?"},
			{ID: "courage_eliminate", Question: "Eliminamos algo que ninguém teve coragem de eliminar?"},
		},
	},
	{
		Step:        7,
		Code:        "growth_hacking",
		Name:        "Growth Loops",
		Description: "Mecanismos de crescimento baseados na estratégia do Oceano Azul",
		Questions: []ClarifyingQuestion{
			{ID: "operational_capacity", Question: "Temos capacidade operacional para esse loop?"},
			{ID: "hypotheses_test", Question: "Quais hipóteses exigem teste imediato?"},
			{ID: "profitable_channel", Question: "Qual canal traz cliente mais lucrativo, não só mais barato?"},
		},
	},
	{
		Step:        8,
		Code:        "scenarios",
		Name:        "Cenários",
		Description: "Simulação de futuros possíveis cruzando Growth, PESTEL e Porter",
		Questions: []ClarifyingQuestion{
			{ID: "most_probable", Question: "Qual cenário é mais provável?"},
			{ID: "plan_killer", Question: "O que pode matar o plano?"},
			{ID: "unacceptable_risk", Question: "O que é risco inaceitável?"},
		},
	},
	{
		Step:        9,
		Code:        "decision_matrix",
		Name:        "Matriz de Decisão",
		Description: "Priorização objetiva baseada em impacto, urgência, custo e risco",
		Questions: []ClarifyingQuestion{
			{ID: "solves_main_challenge", Question: "Estamos priorizando o que resolve o desafio principal?"},
			{ID: "dependencies", Question: "Alguma iniciativa depende de outra?"},
			{ID: "parallel_possible", Question: "O que pode ser feito em paralelo?"},
		},
	},
	{
		Step:        10,
		Code:        "okrs",
		Name:        "Plano 90 Dias",
		Description: "Roteiro tático de execução imediata com OKRs",
		Questions: []ClarifyingQuestion{
			{ID: "fits_90_days", Question: "Isso cabe em 90 dias?"},
			{ID: "who_delivers", Question: "Quem entrega?"},
			{ID: "measurable_now", Question: "O que pode ser medido agora?"},
		},
	},
	{
		Step:        11,
		Code:        "bsc",
		Name:        "BSC",
		Description: "Painel de controle final para monitoramento da estratégia",
		Questions: []ClarifyingQuestion{
			{ID: "represents_strategy", Question: "O BSC representa a estratégia real?"},
			{ID: "winning_indicators", Question: "Esses indicadores permitem saber se estamos ganhando?"},
			{ID: "vital_metrics", Question: "Há excesso de métricas? Quais são as 4 vitais?"},
		},
	},
}

// TotalSteps is the total number of wizard steps
const TotalSteps = 12

// GetFrameworkStep returns the framework step for a given step number
func GetFrameworkStep(step int) *FrameworkStep {
	if step < 0 || step >= len(FrameworkOrder) {
		return nil
	}
	return &FrameworkOrder[step]
}

// GetFrameworkByCode returns the framework step for a given code
func GetFrameworkByCode(code string) *FrameworkStep {
	for _, fw := range FrameworkOrder {
		if fw.Code == code {
			return &fw
		}
	}
	return nil
}

// GetAllFrameworks returns all framework steps
func GetAllFrameworks() []FrameworkStep {
	return FrameworkOrder
}

// GetFrameworkOrder returns the framework execution order for a challenge type.
// MVP: Returns the default order for all challenge types (all 12 frameworks).
// FUTURE: This function is the extension point for per-challenge framework selection.
func GetFrameworkOrder(challengeType string) []FrameworkStep {
	// MVP: Same order for all challenges - run all 12 frameworks
	_ = challengeType // Unused in MVP, but signature ready for future use
	return FrameworkOrder
}
