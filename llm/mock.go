package llm

import (
	"encoding/json"
	"fmt"
)

// MockResponses contains canned LLM responses for each framework
// Used when LLM_MOCK_MODE=true for fast testing and frontend development
var MockResponses = map[string]interface{}{
	"challenge_refinement": map[string]interface{}{
		"refined_context": "[MOCK] O desafio central da empresa é aumentar a participação de mercado em 20% nos próximos 6 meses. A análise preliminar indica que a falta de diferenciação no mercado é a causa raiz principal. A empresa possui uma equipe técnica qualificada e tecnologia proprietária como forças, mas enfrenta desafios em presença digital e processos internos. O mercado está favorável com crescimento de 15% ao ano e oportunidades claras em transformação digital. Para resolver este desafio, precisamos focar em: (1) desenvolver uma proposta de valor única baseada em tecnologia, (2) acelerar a presença digital, e (3) estabelecer parcerias estratégicas. O sucesso será medido pelo market share trimestral, com meta de atingir 20% em 180 dias.",
	},

	"pestel": map[string]interface{}{
		"political":      []string{"[MOCK] Regulamentação favorável ao setor", "[MOCK] Incentivos fiscais disponíveis"},
		"economic":       []string{"[MOCK] Crescimento do PIB projetado em 2.5%", "[MOCK] Taxa de juros em tendência de queda"},
		"social":         []string{"[MOCK] Mudança no comportamento do consumidor", "[MOCK] Aumento da demanda por sustentabilidade"},
		"technological":  []string{"[MOCK] Adoção acelerada de IA", "[MOCK] Transformação digital em andamento"},
		"environmental":  []string{"[MOCK] Pressão por práticas ESG", "[MOCK] Regulamentações ambientais mais rígidas"},
		"legal":          []string{"[MOCK] LGPD em plena vigência", "[MOCK] Novas normas setoriais"},
		"summary":        "[MOCK] Ambiente macro favorável com oportunidades em tecnologia e sustentabilidade.",
	},

	"porter": map[string]interface{}{
		"forces": []map[string]interface{}{
			{"force": "Rivalidade Competitiva", "intensity": "High", "description": "[MOCK] Competição intensa com 5 players principais no mercado"},
			{"force": "Poder dos Fornecedores", "intensity": "Medium", "description": "[MOCK] Poder moderado dos fornecedores com alternativas disponíveis"},
			{"force": "Poder dos Compradores", "intensity": "High", "description": "[MOCK] Alto poder de barganha dos clientes devido a múltiplas opções"},
			{"force": "Ameaça de Novos Entrantes", "intensity": "Medium", "description": "[MOCK] Barreiras médias à entrada no mercado"},
			{"force": "Ameaça de Substitutos", "intensity": "Low", "description": "[MOCK] Ameaça baixa de substitutos diretos no curto prazo"},
			{"force": "Poder de Parcerias/Ecossistemas", "intensity": "Medium", "description": "[MOCK] Ecossistema de parcerias em desenvolvimento"},
			{"force": "Disrupção por IA/Dados", "intensity": "Very High", "description": "[MOCK] Alto potencial de disrupção via IA e análise de dados"},
		},
		"overallAttractiveness": "Média-Alta",
		"summary":               "[MOCK] Indústria com competição intensa mas oportunidades claras de diferenciação via tecnologia e parcerias.",
	},

	"benchmarking": map[string]interface{}{
		"competitorsAnalyzed": []string{"[MOCK] Concorrente A - líder de mercado", "[MOCK] Concorrente B - preço competitivo", "[MOCK] Concorrente C - tecnologia avançada"},
		"performanceGaps":     []string{"[MOCK] Experiência mobile inferior", "[MOCK] Velocidade de entrega", "[MOCK] Presença digital limitada"},
		"bestPractices":       []string{"[MOCK] Atendimento omnichannel", "[MOCK] Precificação dinâmica", "[MOCK] Fidelização gamificada"},
		"summary":             "[MOCK] Gaps claros identificados em experiência digital e logística.",
	},

	"swot": map[string]interface{}{
		"strengths": []map[string]interface{}{
			{"content": "[MOCK] Equipe técnica qualificada", "confidence": "Alta", "source": "fato"},
			{"content": "[MOCK] Base de clientes fiel", "confidence": "Alta", "source": "análise de mercado"},
			{"content": "[MOCK] Tecnologia proprietária", "confidence": "Média", "source": "fato"},
		},
		"weaknesses": []map[string]interface{}{
			{"content": "[MOCK] Falta de presença digital", "confidence": "Alta", "source": "análise de mercado"},
			{"content": "[MOCK] Processos internos lentos", "confidence": "Média", "source": "feedback de clientes"},
		},
		"opportunities": []map[string]interface{}{
			{"content": "[MOCK] Expansão para novos mercados", "confidence": "Média", "source": "análise de mercado"},
			{"content": "[MOCK] Parcerias estratégicas", "confidence": "Alta", "source": "estimativa"},
		},
		"threats": []map[string]interface{}{
			{"content": "[MOCK] Novos entrantes digitais", "confidence": "Alta", "source": "análise de mercado"},
			{"content": "[MOCK] Mudanças regulatórias", "confidence": "Média", "source": "estimativa"},
		},
		"summary": "[MOCK] Empresa com forças técnicas sólidas mas gaps em presença digital.",
	},

	"swotcross": map[string]interface{}{
		"so_strategies": []string{
			"[MOCK] Usar tecnologia proprietária para expandir para novos mercados geográficos",
			"[MOCK] Aproveitar equipe qualificada para desenvolver soluções inovadoras em segmentos em crescimento",
			"[MOCK] Capitalizar base de clientes fiéis para testar novos produtos antes do lançamento amplo",
		},
		"wo_strategies": []string{
			"[MOCK] Desenvolver presença digital via parcerias estratégicas com empresas de tecnologia",
			"[MOCK] Automatizar processos internos aproveitando o momento de transformação digital do mercado",
			"[MOCK] Contratar talentos digitais atraídos pelo potencial de crescimento do setor",
		},
		"st_strategies": []string{
			"[MOCK] Fortalecer diferenciação técnica para criar barreiras contra novos entrantes",
			"[MOCK] Usar relacionamento com clientes para blindar contra competidores digitais",
			"[MOCK] Antecipar mudanças regulatórias usando expertise interna",
		},
		"wt_strategies": []string{
			"[MOCK] Acelerar digitalização para não perder terreno para entrantes nativos digitais",
			"[MOCK] Simplificar processos para reduzir vulnerabilidade a mudanças de mercado",
			"[MOCK] Desenvolver plano de contingência para cenários de pressão competitiva intensa",
		},
		"summary": "[MOCK] Estratégias cruzadas identificadas com foco em digitalização e diferenciação técnica como prioridades principais.",
	},

	"tam_sam_som": map[string]interface{}{
		"tam":               "R$ 50 bilhões - Mercado total endereçável no Brasil",
		"sam":               "R$ 5 bilhões - Segmento acessível com produto atual",
		"som":               "R$ 500 milhões - Mercado capturável em 3 anos",
		"assumptions":       []string{"[MOCK] Crescimento de 15% ao ano", "[MOCK] Digitalização acelerada", "[MOCK] Regulamentação favorável"},
		"cagr":              "15%",
		"confidence_level":  75,
		"estimation_method": "[MOCK] Bottom-up baseado em dados de mercado",
		"calculation_notes": "[MOCK] Estimativas baseadas em relatórios de mercado e análise competitiva",
		"summary":           "[MOCK] Mercado de R$ 50B com SOM de R$ 500M capturável em 3 anos.",
	},

	"blue_ocean": map[string]interface{}{
		"eliminate":      []string{"[MOCK] Burocracia excessiva", "[MOCK] Intermediários desnecessários"},
		"reduce":         []string{"[MOCK] Custos operacionais", "[MOCK] Tempo de entrega"},
		"raise":          []string{"[MOCK] Experiência do cliente", "[MOCK] Personalização"},
		"create":         []string{"[MOCK] Modelo de assinatura", "[MOCK] Comunidade de usuários"},
		"newValueCurve":  "[MOCK] Curva de valor focada em experiência digital e comunidade",
		"summary":        "[MOCK] Oportunidade de oceano azul via modelo de assinatura e comunidade.",
	},

	"growth_hacking": map[string]interface{}{
		"leap_loop": map[string]interface{}{
			"name":       "[MOCK] Loop de Aquisição Viral",
			"type":       "viral",
			"steps":      []string{"[MOCK] Usuário experimenta produto", "[MOCK] Convida amigos via referral", "[MOCK] Amigos se cadastram", "[MOCK] Ciclo se repete"},
			"metrics":    []string{"[MOCK] K-factor > 1", "[MOCK] Viral coefficient", "[MOCK] Tempo de ciclo"},
			"bottleneck": "[MOCK] Taxa de conversão no convite",
		},
		"scale_loop": map[string]interface{}{
			"name":       "[MOCK] Loop de Retenção e Receita",
			"type":       "retention",
			"steps":      []string{"[MOCK] Onboarding gamificado", "[MOCK] Uso recorrente", "[MOCK] Upsell automatizado", "[MOCK] Expansão de conta"},
			"metrics":    []string{"[MOCK] DAU/MAU > 40%", "[MOCK] Churn < 5%", "[MOCK] LTV/CAC > 3"},
			"bottleneck": "[MOCK] Engajamento na primeira semana",
		},
		"summary": "[MOCK] Dois loops de crescimento identificados: viral para aquisição e retenção para escala.",
	},

	"scenarios": map[string]interface{}{
		"optimistic": map[string]interface{}{
			"name":             "[MOCK] Cenário Otimista",
			"probability":      30,
			"description":      "[MOCK] Crescimento acelerado do mercado com adoção rápida",
			"required_actions": []string{"[MOCK] Escalar operações", "[MOCK] Investir em capacidade", "[MOCK] Expandir equipe"},
		},
		"realist": map[string]interface{}{
			"name":             "[MOCK] Cenário Realista",
			"probability":      50,
			"description":      "[MOCK] Crescimento moderado seguindo tendências atuais",
			"required_actions": []string{"[MOCK] Execução disciplinada", "[MOCK] Otimização contínua", "[MOCK] Monitoramento de KPIs"},
		},
		"pessimistic": map[string]interface{}{
			"name":             "[MOCK] Cenário Pessimista",
			"probability":      20,
			"description":      "[MOCK] Contração do mercado com desafios regulatórios",
			"required_actions": []string{"[MOCK] Reduzir custos", "[MOCK] Diversificar receita", "[MOCK] Contingência de caixa"},
		},
		"mitigation_tactics":    []string{"[MOCK] Diversificação de receita", "[MOCK] Parcerias estratégicas", "[MOCK] Reserva de contingência"},
		"early_warning_signals": []string{"[MOCK] Queda no pipeline", "[MOCK] Aumento de churn", "[MOCK] Mudanças regulatórias"},
		"summary":               "[MOCK] Três cenários mapeados com 50% de probabilidade para cenário realista.",
	},

	"decision_matrix": map[string]interface{}{
		"alternatives":         []string{"[MOCK] Expansão digital", "[MOCK] Novos mercados", "[MOCK] Parcerias estratégicas"},
		"criteria":             []string{"[MOCK] Impacto financeiro", "[MOCK] Viabilidade técnica", "[MOCK] Alinhamento estratégico", "[MOCK] Risco"},
		"final_recommendation": "[MOCK] Expansão digital é a recomendação prioritária com score 7.6/10",
		"recommended_option":   "[MOCK] Expansão Digital",
		"score":                "7.6",
		"score_comparison":     "[MOCK] Expansão (7.6) > Parcerias (7.4) > Novos mercados (6.5)",
		"priority_recommendations": []map[string]interface{}{
			{"priority": 1, "title": "[MOCK] Expansão Digital", "description": "[MOCK] Desenvolver plataforma digital completa", "timeline": "6 meses", "budget": "R$ 200k"},
			{"priority": 2, "title": "[MOCK] Parcerias Estratégicas", "description": "[MOCK] Firmar 3 parcerias-chave", "timeline": "3 meses", "budget": "R$ 50k"},
			{"priority": 3, "title": "[MOCK] Novos Mercados", "description": "[MOCK] Expandir para região Sul", "timeline": "12 meses", "budget": "R$ 300k"},
		},
		"review_cycle": map[string]interface{}{
			"frequency":              "Mensal",
			"extraordinary_triggers": []string{"[MOCK] Mudança > 20% em KPIs", "[MOCK] Novo concorrente", "[MOCK] Mudança regulatória"},
		},
		"monitoring_metrics": []string{"[MOCK] ROI por iniciativa", "[MOCK] Milestone completion", "[MOCK] Budget variance"},
		"summary":            "[MOCK] Matriz decisória aponta expansão digital como prioridade.",
	},

	"okrs": map[string]interface{}{
		"plan_90_days": []map[string]interface{}{
			{
				"month":                  "Mês 1",
				"focus":                  "Fundação",
				"objective":              "[MOCK] Estabelecer bases para expansão digital",
				"key_results":            []string{"[MOCK] KR1: Definir roadmap", "[MOCK] KR2: Contratar equipe", "[MOCK] KR3: Setup infraestrutura"},
				"investment":             "R$ 50k",
				"aligned_recommendation": "Expansão Digital",
			},
			{
				"month":                  "Mês 2",
				"focus":                  "Crescimento",
				"objective":              "[MOCK] Desenvolver MVP e validar mercado",
				"key_results":            []string{"[MOCK] KR1: MVP funcional", "[MOCK] KR2: 100 usuários beta", "[MOCK] KR3: NPS > 40"},
				"investment":             "R$ 75k",
				"aligned_recommendation": "Expansão Digital",
			},
			{
				"month":                  "Mês 3",
				"focus":                  "Consolidação",
				"objective":              "[MOCK] Lançamento e primeiros clientes pagantes",
				"key_results":            []string{"[MOCK] KR1: Lançamento público", "[MOCK] KR2: 50 clientes pagantes", "[MOCK] KR3: Receita R$ 25k"},
				"investment":             "R$ 50k",
				"aligned_recommendation": "Expansão Digital",
			},
		},
		"total_investment": "R$ 175k",
		"success_metrics":  []string{"[MOCK] Receita recorrente", "[MOCK] CAC < R$ 100", "[MOCK] Churn < 5%"},
		"summary":          "[MOCK] Plano de 90 dias com investimento de R$ 175k para expansão digital.",
	},

	"bsc": map[string]interface{}{
		"financial": []string{
			"[MOCK] Aumentar receita em 25%",
			"[MOCK] Reduzir custos operacionais em 10%",
			"[MOCK] Atingir margem EBITDA de 15%",
		},
		"customer": []string{
			"[MOCK] NPS > 50",
			"[MOCK] Taxa de retenção > 90%",
			"[MOCK] Tempo de resposta < 2h",
		},
		"internal_processes": []string{
			"[MOCK] Automatizar 80% dos processos",
			"[MOCK] Reduzir tempo de ciclo em 30%",
			"[MOCK] Zero downtime crítico",
		},
		"learning_growth": []string{
			"[MOCK] 40h de treinamento/funcionário",
			"[MOCK] Turnover < 10%",
			"[MOCK] Índice de inovação > 20%",
		},
		"summary": "[MOCK] BSC balanceado com foco em crescimento sustentável.",
	},

	"synthesis": map[string]interface{}{
		"executiveSummary": "[MOCK] A análise estratégica completa revela uma oportunidade clara de expansão digital com ROI estimado de 3x em 12 meses. O mercado está favorável com crescimento de 15% ao ano, e a empresa possui forças técnicas sólidas que podem ser alavancadas. A recomendação principal é acelerar a transformação digital para capturar market share antes dos concorrentes, com investimento de R$ 175k em 90 dias.",
		"keyFindings": []string{
			"[MOCK] Mercado em crescimento de 15% ao ano com oportunidades claras em transformação digital",
			"[MOCK] Gaps significativos em presença digital comparado aos concorrentes principais",
			"[MOCK] Tecnologia proprietária e equipe qualificada são diferenciais competitivos subutilizados",
			"[MOCK] Base de clientes fiel representa oportunidade de expansão via cross-sell",
			"[MOCK] Cenário regulatório favorável para inovação no setor",
		},
		"strategicPriorities": []string{
			"[MOCK] Expansão digital acelerada como prioridade máxima nos próximos 90 dias",
			"[MOCK] Estabelecimento de parcerias estratégicas para complementar capacidades internas",
			"[MOCK] Programa de fidelização e retenção para proteger base de clientes",
			"[MOCK] Automatização de processos internos para ganho de eficiência",
		},
		"roadmap": []string{
			"[MOCK] Mês 1: Definir roadmap de produto digital, contratar equipe, setup de infraestrutura",
			"[MOCK] Mês 2: Desenvolver MVP, recrutar 100 usuários beta, validar proposta de valor",
			"[MOCK] Mês 3: Lançamento público, primeiros 50 clientes pagantes, atingir R$ 25k em receita",
			"[MOCK] Mês 4-6: Escalar operações, firmar parcerias-chave, expandir para novos segmentos",
			"[MOCK] Mês 7-12: Consolidar posição de mercado, atingir 20% de market share, ROI 3x",
		},
		"overallRecommendation": "[MOCK] Recomendamos fortemente a aprovação do plano de expansão digital com investimento de R$ 175k. A análise demonstra que o mercado está favorável, a empresa possui os ativos necessários, e o timing é crítico para capturar a oportunidade antes dos concorrentes. O principal risco é a inação, que resultaria em perda estimada de R$ 500k/ano em receita. Com execução disciplinada, projetamos ROI de 3x em 12 meses e atingimento da meta de 20% de market share.",
	},
}

// GetMockResponse returns a mock JSON response for the given framework
// Returns the JSON string that would normally come from the LLM
func GetMockResponse(frameworkCode string) (string, error) {
	response, exists := MockResponses[frameworkCode]
	if !exists {
		return "", fmt.Errorf("no mock response defined for framework: %s", frameworkCode)
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal mock response: %w", err)
	}

	return string(jsonBytes), nil
}

// GetMockResponseStruct unmarshals the mock response directly into the target struct
func GetMockResponseStruct(frameworkCode string, target interface{}) error {
	response, exists := MockResponses[frameworkCode]
	if !exists {
		return fmt.Errorf("no mock response defined for framework: %s", frameworkCode)
	}

	// Marshal to JSON then unmarshal to target (to handle type conversion properly)
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal mock response: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal mock response: %w", err)
	}

	return nil
}
