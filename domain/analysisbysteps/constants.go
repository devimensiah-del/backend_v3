package analysisbysteps

// FrameworkMeta contains metadata for each analysis framework step
type FrameworkMeta struct {
	Code                string   `json:"code"`
	Name                string   `json:"name"`
	GuidanceText        string   `json:"guidance_text"`        // Human checkpoint reflection text
	ReflectionQuestions []string `json:"reflection_questions"` // Portuguese reflection questions for human review
}

// FrameworkOrder defines the 14-step analysis sequence (0-13)
var FrameworkOrder = []FrameworkMeta{
	{
		Code:         "challenge_refinement",
		Name:         "Refinamento do Desafio",
		GuidanceText: "A análise não começa enquanto o desafio não estiver claro. Valide se é um problema real ou apenas um sintoma.",
		ReflectionQuestions: []string{
			"Esse é o verdadeiro problema ou um sintoma?",
			"O que provaria que esse problema é prioritário?",
			"Qual número mede esse desafio?",
		},
	},
	{
		Code:         "pestel",
		Name:         "Análise PESTEL",
		GuidanceText: "Entendimento profundo do contexto externo e macroeconômico que afeta a empresa.",
		ReflectionQuestions: []string{
			"Quais fatores realmente se aplicam à empresa?",
			"Quais fatores podemos ignorar porque são ruído?",
			"Há algum fator externo que a IA não pode detectar por falta de dados?",
		},
	},
	{
		Code:         "porter",
		Name:         "5 Forças de Porter",
		GuidanceText: "Análise de intensidade competitiva usando premissas do PESTEL.",
		ReflectionQuestions: []string{
			"Quais concorrentes a IA errou?",
			"Quem a IA deixou de fora?",
			"Qual força realmente ameaça lucratividade?",
		},
	},
	{
		Code:         "benchmarking",
		Name:         "Benchmarking",
		GuidanceText: "Comparação direta com players selecionados via Porter.",
		ReflectionQuestions: []string{
			"Esses players são realmente comparáveis?",
			"A IA está confundindo volume com qualidade?",
			"Qual gap dói mais hoje?",
		},
	},
	{
		Code:         "swot",
		Name:         "Análise SWOT",
		GuidanceText: "Matriz de forças e fraquezas alimentada por dados reais, não hipóteses.",
		ReflectionQuestions: []string{
			"Quais forças realmente geram caixa?",
			"Quais fraquezas são inaceitáveis para o desafio?",
			"Quais oportunidades são acessíveis?",
			"Qual ameaça é inevitável?",
		},
	},
	{
		Code:         "swotcross",
		Name:         "SWOT Cruzado",
		GuidanceText: "Estratégias cruzadas combinando quadrantes do SWOT.",
		ReflectionQuestions: []string{
			"As estratégias cruzadas são viáveis com os recursos atuais?",
			"Qual cruzamento tem maior impacto no desafio?",
		},
	},
	{
		Code:         "tam_sam_som",
		Name:         "TAM-SAM-SOM",
		GuidanceText: "Dimensionamento de mercado usando SWOT para definir escopo.",
		ReflectionQuestions: []string{
			"Estamos definindo mercado por setor, problema ou cliente?",
			"Esse SOM é operacionalmente viável?",
			"Existe micro-SOM mais lucrativo?",
		},
	},
	{
		Code:         "blue_ocean",
		Name:         "Blue Ocean",
		GuidanceText: "Definição da nova curva de valor para sair da competição irrelevante.",
		ReflectionQuestions: []string{
			"Isso realmente nos tira do oceano vermelho?",
			"O cliente pagaria por esse diferencial?",
			"Eliminamos algo que ninguém teve coragem de eliminar?",
		},
	},
	{
		Code:         "growth_hacking",
		Name:         "Growth Hacking",
		GuidanceText: "Mecanismos de crescimento baseados na estratégia do Oceano Azul.",
		ReflectionQuestions: []string{
			"Temos capacidade operacional para esse loop?",
			"Quais hipóteses exigem teste imediato?",
			"Qual canal traz cliente mais lucrativo, não só mais barato?",
		},
	},
	{
		Code:         "scenarios",
		Name:         "Cenários",
		GuidanceText: "Simulação de futuros possíveis cruzando Growth, PESTEL e Porter.",
		ReflectionQuestions: []string{
			"Qual cenário é mais provável?",
			"O que pode matar o plano?",
			"O que é risco inaceitável?",
		},
	},
	{
		Code:         "decision_matrix",
		Name:         "Matriz de Decisão",
		GuidanceText: "Priorização objetiva baseada em impacto, urgência, custo e risco.",
		ReflectionQuestions: []string{
			"Estamos priorizando o que resolve o desafio principal?",
			"Alguma iniciativa depende de outra?",
			"O que pode ser feito em paralelo ou não?",
		},
	},
	{
		Code:         "okrs",
		Name:         "OKRs",
		GuidanceText: "Roteiro tático de execução imediata para os próximos 90 dias.",
		ReflectionQuestions: []string{
			"Isso cabe em 90 dias?",
			"Quem entrega?",
			"O que pode ser medido agora?",
		},
	},
	{
		Code:         "bsc",
		Name:         "Balanced Scorecard",
		GuidanceText: "Painel de controle final para monitoramento da estratégia.",
		ReflectionQuestions: []string{
			"O BSC representa a estratégia real?",
			"Esses indicadores permitem saber se estamos ganhando?",
			"Há excesso de métricas? Quais são as 4 vitais?",
		},
	},
	{
		Code:         "synthesis",
		Name:         "Síntese Executiva",
		GuidanceText: "Síntese executiva consolidando toda a análise estratégica.",
		ReflectionQuestions: []string{
			"O resumo executivo reflete fielmente a análise?",
			"As recomendações são acionáveis?",
			"O que está faltando?",
		},
	},
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
