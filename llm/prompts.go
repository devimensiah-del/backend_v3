package llm

// System prompts implementing the 10XMentorAI Cognitive Frameworks.
// All outputs are strictly JSON in PT-BR.
// OPTIMIZED FOR: PDF Layout Constraints (16 Pages, No Scroll) & Strategic Density.

const (
	// Layer3InferencePrompt - AI-powered strategic intelligence (Layer 3)
	Layer3InferencePrompt = `Você é um Analista de Inteligência Estratégica Sênior.
Seu objetivo é gerar insights estratégicos profundos com base em dados coletados.

Empresa Alvo: {{COMPANY_NAME}}

{{CONTEXT_DATA}}

**MISSÃO: ANÁLISE ESTRATÉGICA EM 3 DIMENSÕES**

**1. BUSINESS SUMMARY (Resumo do Negócio)**
- Descrição clara e concisa do negócio (2-3 frases)
- Proposta de valor central
- Mercado-alvo primário
- Principais produtos/serviços (3-5)
- Tom de marca (profissional, inovador, tradicional, etc.)
- Fatores únicos de diferenciação

**2. DIGITAL MATURITY (Maturidade Digital)**
Avalie de 1-10 cada dimensão:
- Qualidade do Website (design, UX, conteúdo)
- Presença SEO (visibilidade em buscas)
- Redes Sociais (engajamento e alcance)
- Avaliações Online (reviews, reputação)
- Capacidade de E-commerce (se aplicável)
- Score Geral de Maturidade Digital
- Observações principais sobre a presença digital

**3. COMPETITIVE INTELLIGENCE (Inteligência Competitiva)**
- Classificação detalhada da indústria
- 3-5 principais concorrentes
- Posição de mercado (líder, desafiador, nicho)
- Principal diferenciador competitivo
- Nível de ameaça competitiva (alto, médio, baixo)

**4. STRATEGIC GAPS (Lacunas Estratégicas)**
Identifique lacunas e oportunidades:
- Gaps Operacionais (processos, capacidades)
- Gaps Tecnológicos (infraestrutura, ferramentas)
- Gaps de Marketing (branding, posicionamento)
- Gaps de Talento (habilidades, equipe) - se relevante
- Oportunidades de Vitória Rápida (quick wins)
- Ações Prioritárias (top 3-5 recomendações)

**IMPORTANTE:**
- Use dados fornecidos das Layer 1 e Layer 2 como base
- Infira de forma inteligente quando dados faltarem
- Seja específico e acionável nas recomendações
- Foque em insights que gerem valor estratégico

Retorne um JSON válido no seguinte formato:
{
  "businessSummary": {
    "description": "Descrição do negócio em 2-3 frases",
    "valueProposition": "Proposta de valor central",
    "targetMarket": "Segmento de clientes primário",
    "keyProducts": ["Produto/Serviço 1", "Produto/Serviço 2", "Produto/Serviço 3"],
    "brandTone": "profissional/inovador/tradicional",
    "uniqueFactors": ["Diferenciador 1", "Diferenciador 2"]
  },
  "digitalMaturity": {
    "overallScore": 6,
    "websiteQuality": 7,
    "seoPresence": 5,
    "socialMedia": 6,
    "onlineReviews": 5,
    "ecommerceCapability": 4,
    "observations": ["Observação 1", "Observação 2", "Observação 3"]
  },
  "competitiveIntel": {
    "industry": "Classificação detalhada da indústria",
    "competitors": ["Concorrente 1", "Concorrente 2", "Concorrente 3"],
    "marketPosition": "líder/desafiador/nicho",
    "keyDifferentiator": "Principal vantagem competitiva",
    "threatLevel": "alto/médio/baixo"
  },
  "strategicGaps": {
    "operationalGaps": ["Gap operacional 1", "Gap 2"],
    "technologyGaps": ["Gap tecnológico 1", "Gap 2"],
    "marketingGaps": ["Gap marketing 1", "Gap 2"],
    "talentGaps": ["Gap talento 1"],
    "opportunities": ["Oportunidade 1", "Oportunidade 2", "Oportunidade 3"],
    "priorityActions": ["Ação prioritária 1", "Ação 2", "Ação 3"]
  }
}`

	// StrategicEnrichmentPrompt (Legacy - kept for backward compatibility)
	StrategicEnrichmentPrompt = `Você é um Consultor de Estratégia Sênior.
Seu objetivo é construir um perfil estratégico profundo de uma empresa com base em sua pegada digital.
Empresa Alvo: {{COMPANY_NAME}}

{{USER_CONTEXT}}

**INSTRUÇÃO DE PESQUISA:**
1. Se o usuário forneceu um Website ou LinkedIn, acesse esses links PRIMEIRO.
2. Use a busca para validar informações e encontrar concorrentes ou notícias recentes.
3. SE NÃO CONSEGUIR ACESSAR FONTES ONLINE, use os dados fornecidos e INFIRA baseado no setor e desafio de negócio.

Fase 1: DESCOBERTA FATUAL (Pesquisar e Verificar - ou Inferir se pesquisa falhar)
- Website, LinkedIn (se fornecido, tente acessar)
- Sede, Ano de Fundação, Contagem de Funcionários (inferir baseado no setor se não encontrar)
- Estimativa de Receita (inferir com base no setor e região)
- Principais Produtos/Serviços (inferir do nome e desafio)

Fase 2: INFERÊNCIA ESTRATÉGICA (SEMPRE execute esta fase, mesmo sem dados externos)
1. Arquétipo de Valor (Baixo Custo vs Premium vs Diferenciação)
2. Segmentação de Clientes (Enterprise vs SMB vs B2C)
3. Maturidade Digital (Score 1-10 baseado em presença online descrita)
4. Principais Forças e Fraquezas (inferir do desafio de negócio fornecido)
5. Posicionamento de Mercado (líder, desafiador, nicho)

Fase 3: CONTEXTO COMPETITIVO
- Liste 3-5 principais concorrentes prováveis no setor
- Diferenciais competitivos (inferir do desafio)

IMPORTANTE:
- Retorne SEMPRE um JSON completo, mesmo se algumas informações forem inferências.
- Marque claramente dados verificados vs inferidos.
- Use o DESAFIO DE NEGÓCIO fornecido como principal fonte de insights.

Retorne um JSON válido no seguinte formato:
{
  "overview": {
    "description": "Descrição da empresa",
    "sources": ["fonte1", "fonte2"] ou null se inferido
  },
  "digitalPresence": {
    "websiteUrl": "url ou vazio",
    "recentNews": ["notícia1"] ou null
  },
  "marketPosition": {
    "industry": "setor detalhado",
    "keyDifferentiator": "diferencial principal",
    "competitors": ["concorrente1", "concorrente2"]
  },
  "strategicInference": {
    "brandTone": "profissional/inovador/tradicional",
    "digitalMaturity": 5,
    "valueArchetype": "Premium/Custo/Diferenciação",
    "customerSegment": "Enterprise/SMB/B2C",
    "strengths": ["força1", "força2"],
    "weaknesses": ["fraqueza1", "fraqueza2"]
  }
}`

	// 1. PESTEL - Modelo SCAN [cite: 416] - CONSTRAINED
	FrameworkPESTELPrompt = `Realize uma análise PESTEL (Modelo SCAN) priorizada para relatório executivo.
Contexto: {{COMPANY_DATA}}
Inteligência: {{ENRICHMENT_DATA}}

REGRAS DE FORMATAÇÃO (CRÍTICO):
1. Selecione EXATAMENTE 3 fatores de maior impacto por dimensão. Não liste mais que 3.
2. Cada fator deve ter no MÁXIMO 120 caracteres. Seja direto e denso.
3. Priorize "Sinais Fracos" que representam oportunidades ou ameaças reais.

Retorne JSON (valores em PT-BR):
{
  "political": ["Fator Crítico 1 (max 120 chars)", "Fator 2", "Fator 3"],
  "economic": ["...", "...", "..."],
  "social": ["...", "...", "..."],
  "technological": ["...", "...", "..."],
  "environmental": ["...", "...", "..."],
  "legal": ["...", "...", "..."],
  "summary": "Resumo executivo conciso (max 250 chars)"
}`

	// 2. PORTER - Modelo RACE [cite: 131] - CONSTRAINED
	FrameworkPorterPrompt = `Analise as 5 Forças de Porter usando o framework RACE.
Contexto: {{COMPANY_DATA}}
Inteligência: {{ENRICHMENT_DATA}}

REGRAS DE FORMATAÇÃO (CRÍTICO):
1. Para cada força, escreva um parágrafo único de MÁXIMO 250 caracteres.
2. Não descreva a teoria. Descreva o IMPACTO direto na empresa.
3. Use linguagem executiva e densa (Ex: "Alta rivalidade devido a X, exigindo diferenciação via Y").

Retorne JSON (valores em PT-BR):
{
  "competitive_rivalry": "Análise de impacto (max 250 chars)",
  "supplier_power": "...",
  "buyer_power": "...",
  "threat_new_entrants": "...",
  "threat_substitutes": "...",
  "overall_attractiveness": "Alta/Média/Baixa",
  "summary": "Resumo executivo conciso"
}`

	// 3. TAM-SAM-SOM - Modelo RACE/AIM [cite: 658, 690] - CONSTRAINED
	FrameworkTamSamSomPrompt = `Dimensione o mercado usando o modelo RACE/AIM (Assess, Infer, Model).
Contexto: {{COMPANY_DATA}}
Inteligência: {{ENRICHMENT_DATA}}

REGRAS:
1. Liste EXATAMENTE 3 premissas críticas.
2. Máximo 100 caracteres por premissa.

Retorne JSON (valores em PT-BR):
{
  "tam": "Valor Total (Ex: R$ 50B)",
  "sam": "Valor Endereçável (Ex: R$ 10B)",
  "som": "Valor Alcançável (Ex: R$ 100M)",
  "assumptions": ["Premissa crítica 1 (max 100 chars)", "Premissa 2", "Premissa 3"],
  "cagr": "Estimativa de crescimento (Ex: +15% a.a.)",
  "summary": "Síntese de potencial (max 200 chars)"
}`

	// 4. SWOT - Modelo LIFT [cite: 317] - CONSTRAINED
	FrameworkSWOTPrompt = `Realize uma análise SWOT (Modelo LIFT) focada em itens acionáveis.
Contexto Interno: {{COMPANY_DATA}}
Contexto Externo (Layer 1): {{PESTEL_INSIGHTS}} | {{PORTER_INSIGHTS}}

REGRAS DE FORMATAÇÃO (CRÍTICO):
1. Liste EXATAMENTE 4 itens por quadrante. Nem mais, nem menos.
2. Cada item deve ter no MÁXIMO 100 caracteres.
3. Ordene por impacto: O item 1 deve ser o mais crítico para a estratégia.

Retorne JSON (valores em PT-BR):
{
  "strengths": ["Força crítica 1 (max 100 chars)", "Força 2", "Força 3", "Força 4"],
  "weaknesses": ["...", "...", "...", "..."],
  "opportunities": ["...", "...", "...", "..."],
  "threats": ["...", "...", "...", "..."],
  "summary": "Resumo da posição estratégica (max 200 chars)"
}`

	// 5. BENCHMARKING - Modelo COMPARE [cite: 920] - CONSTRAINED
	FrameworkBenchmarkingPrompt = `Realize um Benchmarking Competitivo (Modelo COMPARE).
Contexto: {{COMPANY_DATA}}
Inteligência: {{ENRICHMENT_DATA}}

REGRAS:
1. Liste exatamente 3 Gaps de Performance e 3 Melhores Práticas.
2. Máximo 120 caracteres por item.

Retorne JSON (valores em PT-BR):
{
  "competitors_analyzed": ["Empresa A", "Empresa B"],
  "performance_gaps": ["Gap crítico 1 (max 120 chars)", "Gap 2", "Gap 3"],
  "best_practices": ["Prática 1 (max 120 chars)", "Prática 2", "Prática 3"],
  "summary": "Resumo comparativo"
}`

	// 6. BLUE OCEAN - Modelo CREATE [cite: 516] - CONSTRAINED
	FrameworkBlueOceanPrompt = `Desenvolva uma Estratégia do Oceano Azul (Modelo CREATE/ERRC).
Contexto: {{COMPANY_DATA}}
Oceano Vermelho Atual: {{PORTER_INSIGHTS}}

REGRAS DE FORMATAÇÃO (CRÍTICO):
1. Liste EXATAMENTE 3 ações para cada categoria (Eliminar, Reduzir, Elevar, Criar).
2. Cada ação deve ser curta e direta (MÁXIMO 90 caracteres).
3. Focar em diferenciação radical.

Retorne JSON (valores em PT-BR):
{
  "eliminate": ["Ação 1", "Ação 2", "Ação 3"],
  "reduce": ["...", "...", "..."],
  "raise": ["...", "...", "..."],
  "create": ["...", "...", "..."],
  "new_value_curve": "Descrição curta da nova curva de valor (max 150 chars)",
  "summary": "Síntese da inovação"
}`

	// 7. GROWTH HACKING - Modelo LEAP [cite: 1164] - CONSTRAINED
	FrameworkGrowthHackingPrompt = `Crie uma estratégia de Growth Hacking (Modelo LEAP).
Contexto: {{COMPANY_DATA}}
Meta: Acelerar tração.

REGRAS:
1. Liste 3 Hipóteses de alto impacto.
2. Liste 3 Experimentos práticos.
3. Máximo 140 caracteres por item.

Retorne JSON (valores em PT-BR):
{
  "hypotheses": ["Hipótese 1 (max 140 chars)", "Hipótese 2", "Hipótese 3"],
  "experiments": ["Experimento 1 (max 140 chars)", "Experimento 2", "Experimento 3"],
  "key_metrics": ["Métrica 1", "Métrica 2", "Métrica 3"],
  "summary": "Resumo da estratégia de crescimento"
}`

	// 8. CENÁRIOS - Modelo FUTUREMAP [cite: 1426] - CONSTRAINED
	FrameworkScenariosPrompt = `Realize uma Análise de Cenários (Modelo FUTUREMAP).
Contexto: {{COMPANY_DATA}}
Incertezas: {{PESTEL_INSIGHTS}}

REGRAS DE FORMATAÇÃO (CRÍTICO):
1. Para cada cenário, escreva um texto denso de MÁXIMO 450 caracteres (aprox. 60 palavras).
2. Seja narrativo mas direto. Descreva o "mundo" e a "implicância".

Retorne JSON (valores em PT-BR):
{
  "scenario_optimistic": "Texto do cenário otimista (max 450 chars)",
  "scenario_realist": "Texto do cenário realista (max 450 chars)",
  "scenario_pessimistic": "Texto do cenário pessimista (max 450 chars)",
  "early_warning_signals": ["Sinal 1", "Sinal 2", "Sinal 3"],
  "summary": "Resumo de riscos"
}`

	// 9. OKRs - Modelo FOCUS [cite: 1041] - CONSTRAINED
	FrameworkOKRsPrompt = `Defina OKRs Estratégicos (Modelo FOCUS).
Contexto: {{COMPANY_DATA}}
Estratégia: {{BLUE_OCEAN_INSIGHTS}}

REGRAS:
1. Crie EXATAMENTE 2 Objetivos Estratégicos.
2. Para cada Objetivo, defina EXATAMENTE 3 Key Results (KRs).
3. Use números nos KRs (Ex: "Aumentar receita em 20%"). Max 100 chars por KR.

Retorne JSON (valores em PT-BR):
{
  "objectives": [
    {
      "title": "Objetivo 1 (Curto)",
      "key_results": ["KR 1 (max 100 chars)", "KR 2", "KR 3"]
    },
    {
      "title": "Objetivo 2 (Curto)",
      "key_results": ["KR 1", "KR 2", "KR 3"]
    }
  ],
  "summary": "Roteiro de implementação"
}`

	// 10. BSC - Modelo ALIGN [cite: 774] - CONSTRAINED
	FrameworkBSCPrompt = `Estruture um Balanced Scorecard (Modelo ALIGN).
Contexto: {{COMPANY_DATA}}
Estratégia: {{BLUE_OCEAN_INSIGHTS}}

REGRAS:
1. Liste EXATAMENTE 2 Objetivos/Métricas para cada perspectiva.
2. Máximo 100 caracteres por item.

Retorne JSON (valores em PT-BR):
{
  "financial": ["Objetivo Financeiro 1", "Objetivo Financeiro 2"],
  "customer": ["Objetivo Cliente 1", "Objetivo Cliente 2"],
  "internal_processes": ["Objetivo Proc. 1", "Objetivo Proc. 2"],
  "learning_growth": ["Objetivo Aprend. 1", "Objetivo Aprend. 2"],
  "summary": "Síntese do alinhamento"
}`

	// 11. MATRIZ DECISÃO - Modelo ANALYTICA [cite: 1303] - CONSTRAINED
	FrameworkDecisionMatrixPrompt = `Crie uma Matriz de Decisão Multicriterial (Modelo ANALYTICA).
Dilema: "Qual a melhor estratégia de crescimento?"
Contexto: {{COMPANY_DATA}}
Cenários: {{SCENARIO_INSIGHTS}}

REGRAS:
1. Liste 3 Alternativas claras.
2. Liste 3 Critérios de decisão (ex: ROI, Risco, Tempo).
3. Recomendação final deve ser um parágrafo curto (max 300 chars).

Retorne JSON (valores em PT-BR):
{
  "alternatives": ["Alternativa A", "Alternativa B", "Alternativa C"],
  "criteria": ["Critério 1", "Critério 2", "Critério 3"],
  "final_recommendation": "A melhor opção é... (max 300 chars)",
  "summary": "Justificativa"
}`

	// SYNTHESIS - Modelo PAR [cite: 200] - CONSTRAINED
	SynthesisPrompt = `Você é um Líder Exponencial. Sintetize os 11 frameworks em um Resumo Executivo.
Contexto: {{COMPANY_DATA}}
Resultados: {{ALL_FRAMEWORK_SUMMARIES}}

REGRAS DE FORMATAÇÃO (CRÍTICO PARA PDF):
1. Executive Summary: Máximo 400 caracteres. Um parágrafo de impacto.
2. Key Findings: Exatamente 3 insights estratégicos (max 150 chars cada).
3. Strategic Priorities: Exatamente 3 prioridades (max 100 chars cada).
4. Roadmap: Exatamente 3 passos macro (Curto, Médio, Longo Prazo).

Retorne JSON (valores em PT-BR):
{
  "executive_summary": "Texto de alto impacto (max 400 chars)",
  "key_findings": ["Insight 1", "Insight 2", "Insight 3"],
  "strategic_priorities": ["Prioridade 1", "Prioridade 2", "Prioridade 3"],
  "roadmap": ["Fase 1: ...", "Fase 2: ...", "Fase 3: ..."],
  "overall_recommendation": "Conclusão final curta"
}`
)
