package llm

// System prompts implementing the 10XMentorAI Cognitive Frameworks.
// All outputs are strictly JSON in PT-BR.
// OPTIMIZED FOR: PDF Layout Constraints (16 Pages, No Scroll) & Strategic Density.

const (
	// CRITICAL: Output is fed into UnifiedEnrichmentPrompt as {{PRE_SEARCH_CONTEXT}}
	PreSearchPrompt = `Você é um Agente de Inteligência de Mercado especializado em pesquisa corporativa e macroeconômica.
Sua missão: Coletar TODOS os dados públicos disponíveis sobre a empresa e seu contexto de mercado.

DADOS DO USUÁRIO:
Nome da Empresa: {{COMPANY_NAME}}
{{USER_CONTEXT}}

--- TAREFA COMPLETA (2 PARTES) ---

## PARTE 1: IDENTIFICAÇÃO CORPORATIVA
O nome "{{COMPANY_NAME}}" pode corresponder a múltiplas empresas. BUSQUE ATIVAMENTE:
- "[{{COMPANY_NAME}}] empresa Brasil site oficial"
- "[{{COMPANY_NAME}}] CNPJ razão social Receita Federal"
- "[{{COMPANY_NAME}}] LinkedIn company page Brasil"
- "[{{COMPANY_NAME}}] fundação história"

Dados a coletar:
- Razão social oficial, nome fantasia
- CNPJ (se público)
- Website oficial
- LinkedIn, Twitter/X
- Ano de fundação
- Sede (cidade, estado, país)
- Setor de atuação (CNAE)
- Porte estimado (funcionários, faturamento)

## PARTE 2: TENDÊNCIAS DO SETOR
Baseado no setor identificado, BUSQUE:
- "[Setor] mercado Brasil 2025 tendências"
- "[Setor] principais players concorrentes Brasil"
- "[Setor] regulamentação leis 2025"
- "[Setor] tecnologia inovação disrupção"

Dados a coletar:
- Crescimento do setor (CAGR)
- Principais concorrentes
- Novas regulamentações
- Tendências tecnológicas
- Barreiras de entrada

---

Retorne JSON estrito (PT-BR):
{
  "identified_company": {
    "nome_oficial": "Razão Social Completa",
    "nome_fantasia": "Nome Fantasia (se diferente)",
    "website_oficial": "https://...",
    "cnpj": "XX.XXX.XXX/XXXX-XX ou null",
    "linkedin_url": "https://linkedin.com/company/...",
    "twitter_handle": "@handle ou null",
    "fundacao_ano": "YYYY",
    "sede": {
      "cidade": "São Paulo",
      "estado": "SP",
      "pais": "Brasil"
    },
    "setor_confirmado": "Tecnologia / Agronegócio / etc",
    "cnae": "Código CNAE se encontrado",
    "porte": "Micro/Pequena/Média/Grande",
    "funcionarios_estimativa": "10-50 / 50-200 / etc",
    "faturamento_estimativa": "R$ X - Y (se encontrado)"
  },
  "industry_context": {
    "sector": "Setor da empresa",
    "growth_rate": "+X% CAGR",
    "market_size_estimate": "R$ XXB ou TAM estimado",
    "key_players": ["Concorrente 1", "Concorrente 2", "Concorrente 3"],
    "technology_trends": ["Trend 1", "Trend 2"],
    "regulatory_changes": ["Regulação 1", "Lei 2"],
    "barriers_to_entry": "Alta/Média/Baixa - descrição"
  },
  "technical_validation": {
    "website_active": true,
    "social_presence": ["LinkedIn", "Twitter", "Instagram"],
    "digital_maturity": "Alta/Média/Baixa",
    "online_reputation": "Resumo da presença online"
  },
  "confidence_score": 85,
  "disambiguation_notes": "Se houver empresas similares, liste aqui",
  "sources_used": ["URL1", "URL2", "URL3"],
  "search_quality": "high/medium/low",
  "data_freshness": "2025-11-26"
}`

	UnifiedEnrichmentPrompt = `Você é um Agente de Inteligência Corporativa (CIA).
Sua missão: Criar um JSON de Perfil Corporativo Perfeito, fundindo dados do usuário, dados técnicos coletados, contexto macroeconômico REAL em tempo real, e suas próprias pesquisas na web.

--- FONTES DE DADOS ---

1. O QUE O USUÁRIO DISSE:
{{USER_CONTEXT}}

2. PRÉ-IDENTIFICAÇÃO DA EMPRESA (PERPLEXITY - Alta Precisão):
{{PRE_SEARCH_CONTEXT}}

INSTRUÇÕES CRÍTICAS PARA PRÉ-IDENTIFICAÇÃO:
- Se os dados acima contiverem uma empresa identificada com alta confiança (confidence_score > 70), USE ESSES DADOS como fonte primária
- O nome oficial, CNPJ, website e dados da sede devem ser priorizados se presentes
- Se houver "disambiguation_notes" sobre empresas similares, use para evitar confusão
- Se a pré-identificação falhou, continue com busca ativa normal

3. O QUE NOSSOS ROBÔS ENCONTRARAM (Dados Técnicos):
{{TECHNICAL_CONTEXT}}

4. CONTEXTO MACROECONÔMICO BRASILEIRO (APIs OFICIAIS BCB + IBGE):
{{REAL_TIME_MACRO_DATA}}

INSTRUÇÕES CRÍTICAS PARA DADOS MACRO (OBRIGATÓRIO):
- Os dados acima foram obtidos diretamente das APIs oficiais do Banco Central do Brasil (BCB) e IBGE
- USE ESTES VALORES EXATOS para SELIC, IPCA e câmbio USD/BRL - NÃO PESQUISE NOVOS VALORES
- Estes são dados autoritativos de alta confiança (100%) - cite a fonte oficial em suas análises
- APENAS se os dados estiverem marcados como "Indisponível", então busque na web como fallback
- Para outros indicadores não incluídos (PIB, desemprego, etc.), você pode buscar na web

5. O QUE FALTA (Sua prioridade de busca):
{{MISSING_FIELDS}}

--- INSTRUÇÕES DE EXECUÇÃO ---

1. **RESOLUÇÃO DE CONFLITOS:**
   - Se os "Dados Técnicos" contradizem o Usuário (ex: usuário diz "Tech", scraper diz "Padaria"), confie no usuario.
   - Se o usuário não informou Localização, busque a sede da empresa.

2. **BUSCA ATIVA - DADOS PÚBLICOS DA EMPRESA (DESCOBERTA):**
   Busque e preencha a seção "discovered_data" com informações públicas que o usuário NÃO forneceu:

   A. **CNPJ e RAZÃO SOCIAL:**
      - Busque: "[Empresa] CNPJ consulta"
      - Busque: "[Empresa] razão social receita federal"

   B. **LINKEDIN DA EMPRESA:**
      - Busque: "[Empresa] linkedin company page"
      - URL típica: linkedin.com/company/[nome-empresa]

   C. **TWITTER/X DA EMPRESA:**
      - Busque: "[Empresa] twitter X perfil oficial"

   D. **WEBSITE OFICIAL:**
      - Se não fornecido, busque: "[Empresa] site oficial"

   E. **ANO DE FUNDAÇÃO:**
      - Busque: "[Empresa] fundação história ano criação"

   F. **TAMANHO DA EMPRESA (funcionários):**
      - Busque: "[Empresa] número funcionários glassdoor linkedin"

   G. **FATURAMENTO ESTIMADO:**
      - Busque: "[Empresa] faturamento receita valor econômico"

   H. **SETOR/INDÚSTRIA:**
      - Busque: "[Empresa] setor atuação CNAE"

3. **BUSCA ATIVA - PERFIL DA EMPRESA:**
   - Para cada item faltante listado acima, use a ferramenta de busca.
   - Exemplo: Se falta faturamento, busque "Faturamento [Empresa] valor econômico".
   - Exemplo: Se falta localização, busque "[Empresa] sede address".

4. **BUSCA ATIVA - CONTEXTO MACROECONÔMICO (CRÍTICO):**
   Esta é a seção mais importante para garantir análises precisas. Busque dados atualizados sobre:

   A. **INDICADORES ECONÔMICOS DO PAÍS:**
      - Busque: "[País] PIB crescimento 2025"
      - Busque: "[País] inflação IPCA taxa juros SELIC 2025"
      - Busque: "[País] câmbio dólar real 2025"
      - Busque: "[País] reformas políticas mudanças regulatórias 2025"

   B. **TENDÊNCIAS DO SETOR:**
      - Busque: "[Setor da empresa] crescimento tendências Brasil 2025"
      - Busque: "[Setor] tecnologia inovação adoção Brasil"
      - Busque: "[Setor] fusões aquisições M&A Brasil 2025"
      - Busque: "[Setor] principais players concentração mercado Brasil"

   C. **AMBIENTE REGULATÓRIO:**
      - Busque: "[Setor] novas leis regulamentações Brasil 2025"
      - Busque: "[Setor] compliance normas padrões ISO Brasil"

   D. **SINAIS DE MERCADO:**
      - Busque: "[Insumos relevantes] preços commodities Brasil 2025" (ex: "aço preços Brasil 2025" se for indústria)
      - Busque: "[Concorrentes identificados] novos produtos lançamentos 2025"

5. **ESTRUTURA DE RETORNO (JSON OBRIGATÓRIO):**
Preencha TODOS os campos. Se não achar exato, estime e marque como "estimated".

{
  "discovered_data": {
    "cnpj": "CNPJ encontrado (ou null se não encontrado)",
    "website": "Website oficial descoberto (ou null)",
    "linkedin_url": "URL do LinkedIn da empresa (ou null)",
    "twitter_handle": "@handle do Twitter/X (ou null)",
    "industry": "Setor/CNAE descoberto (ou null)",
    "company_size": "Faixa de funcionários descoberta (ou null)",
    "location": "Sede/Localização descoberta (ou null)",
    "foundation_year": "Ano de fundação descoberto (ou null)",
    "funding_stage": "Estágio de financiamento descoberto (ou null)",
    "annual_revenue_estimate": "Faturamento estimado descoberto (ou null)",
    "target_market": "Mercado alvo descoberto (ou null)"
  },
  "profile_overview": {
    "legal_name": "Razão Social Completa",
    "website": "URL Verificada",
    "foundation_year": "Ano (ou N/A)",
    "headquarters": "Cidade, Estado, País (Essencial)"
  },
  "market_position": {
    "sector": "Setor Específico (ex: SaaS Fintech)",
    "target_audience": "Descrição do ICP",
    "value_proposition": "Proposta de valor principal"
  },
  "financials": {
    "employees_range": "Ex: 10-50",
    "revenue_estimate": "Ex: R$ 2M - 5M/ano",
    "business_model": "Ex: B2B Recorrente"
  },
  "competitive_landscape": {
    "competitors": ["Concorrente A", "Concorrente B", "Concorrente C"],
    "market_share_status": "Líder / Desafiador / Nicho"
  },
  "strategic_assessment": {
    "digital_maturity": 7,
    "strengths": ["Força 1", "Força 2"],
    "weaknesses": ["Fraqueza 1", "Fraqueza 2"]
  },
  "macro_context": {
    "economic_indicators": {
      "country": "Brasil",
      "gdp_growth": "Ex: +2.5% (2024) ou null se indisponível",
      "inflation_rate": "Ex: IPCA 4.83% a.a. ou null se indisponível",
      "interest_rate": "Ex: SELIC 12.25% a.a. ou null se indisponível",
      "exchange_rate": "Ex: USD/BRL 5.85 ou null se indisponível",
      "unemployment_rate": "Ex: 6.5% ou null se indisponível",
      "political_stability": "Ex: Moderada - reformas em andamento",
      "economic_outlook": "Síntese do cenário econômico atual",
      "recent_policy_changes": ["Reforma Tributária 2025", "Plano Safra 2025/26"]
    },
    "industry_trends": {
      "industry_sector": "Agronegócio Tecnológico",
      "growth_rate": "+12% CAGR (2024-2028)",
      "key_trends": ["Adoção de IoT", "Foco em sustentabilidade", "Transformação digital"],
      "technology_adoption": "Alta adoção de cloud/IA no setor",
      "market_concentration": "Fragmentado - sem player dominante",
      "barriers_to_entry": "Médio - requer expertise técnica e capital",
      "mergers_acquisitions": ["Empresa X adquiriu Y em 2024", "Consolidação em andamento"]
    },
    "regulatory_landscape": {
      "recent_regulations": ["Lei do Agro 2025", "Nova conformidade ambiental"],
      "upcoming_changes": ["Proposta de imposto sobre carbono", "Atualização lei de dados"],
      "compliance_requirements": "ISO 9001, rastreabilidade obrigatória",
      "industry_standards": ["ISO 9001", "ISO 14001", "Rastreabilidade MAPA"]
    },
    "market_signals": {
      "commodity_prices": ["Aço +12% YoY", "Cobre estável"],
      "supply_chain_status": "Atrasos moderados em componentes eletrônicos",
      "consumer_sentiment": "Cauteloso devido a inflação",
      "competitor_activity": ["Concorrente X lançou novo produto", "Concorrente Y expandiu para nova região"],
      "emerging_threats": ["Novo entrante low-cost da China", "Tecnologia substituta ganhando tração"]
    },
    "data_sources": ["url1", "url2", "url3"],
    "last_updated": "2025-11-22"
  }
}`

	// 1. PESTEL - Modelo SCAN [cite: 416] - CONSTRAINED
	FrameworkPESTELPrompt = `Realize uma análise PESTEL (Modelo SCAN) priorizada para relatório executivo.
Contexto: {{COMPANY_DATA}}
Inteligência: {{ENRICHMENT_DATA}}
Macro-Contexto: {{MACRO_CONTEXT}}

INSTRUÇÕES CRÍTICAS:
1. Use os dados do Macro-Contexto para fundamentar CADA fator com evidências concretas e atualizadas.
2. Para Political/Economic: Cite números específicos (ex: "SELIC 11.75%", "Reforma tributária 2025").
3. Para Technological/Social: Correlacione tendências do setor com a posição da empresa.
4. Para Environmental/Legal: Cite regulamentações específicas do Macro-Contexto.

REGRAS DE FORMATAÇÃO (CRÍTICO):
1. Selecione EXATAMENTE 3 fatores de maior impacto por dimensão. Não liste mais que 3.
2. Cada fator deve ter no MÁXIMO 120 caracteres. Seja direto e denso.
3. Priorize "Sinais Fracos" que representam oportunidades ou ameaças reais.
4. SEMPRE baseie-se nos dados reais do macro_context, NÃO em conhecimento genérico.

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

	// 2. PORTER 7 FORCES (2025+) - Modelo RACE [cite: 131] - CONSTRAINED
	FrameworkPorterPrompt = `Analise as 7 Forças de Porter (2025+) usando o framework RACE.
Contexto: {{COMPANY_DATA}}
Inteligência: {{ENRICHMENT_DATA}}
Macro-Contexto: {{MACRO_CONTEXT}}

INSTRUÇÕES CRÍTICAS - Use Macro-Contexto para fundamentar cada força:
1. **Supplier Power**: Use commodity_prices e supply_chain_status do macro_context.
2. **Buyer Power**: Use consumer_sentiment e economic_indicators (inflação, desemprego).
3. **Competitive Rivalry**: Use competitor_activity, market_concentration, e M&A data.
4. **Threat of New Entrants**: Use barriers_to_entry e regulatory_landscape.
5. **Threat of Substitutes**: Use emerging_threats e technology_adoption do setor.
6. **Power of Partnerships/Ecosystems** (NOVA): Use industry trends, collaborative platforms, strategic alliances.
7. **AI/Data Disruption** (NOVA): Use technology_adoption, emerging_threats relacionados a IA/dados.

INTENSIDADE: Para cada força, classifique como "Alta", "Média" ou "Baixa".

IMPLICAÇÕES ESTRATÉGICAS: Liste 4 ações estratégicas baseadas nas forças.

REGRAS DE FORMATAÇÃO (CRÍTICO):
1. Para cada força, escreva um parágrafo único de MÁXIMO 250 caracteres.
2. Não descreva a teoria. Descreva o IMPACTO direto na empresa com dados reais.
3. Use linguagem executiva e densa (Ex: "Alta rivalidade devido a X, exigindo diferenciação via Y").
4. Cite números e fatos específicos do Macro-Contexto sempre que possível.

Retorne JSON (valores em PT-BR):
{
  "competitive_rivalry": "Análise de impacto (max 250 chars)",
  "supplier_power": "...",
  "buyer_power": "...",
  "threat_new_entrants": "...",
  "threat_substitutes": "...",
  "power_partnerships_ecosystems": "Análise de parcerias e ecossistemas (max 250 chars)",
  "disruption_ai_data": "Análise de disrupção por IA/Dados (max 250 chars)",
  "competitive_rivalry_intensity": "Alta|Média|Baixa",
  "supplier_power_intensity": "Alta|Média|Baixa",
  "buyer_power_intensity": "Alta|Média|Baixa",
  "threat_new_entrants_intensity": "Alta|Média|Baixa",
  "threat_substitutes_intensity": "Alta|Média|Baixa",
  "power_partnerships_ecosystems_intensity": "Alta|Média|Baixa",
  "disruption_ai_data_intensity": "Alta|Média|Baixa",
  "strategic_implications": ["Implicação 1", "Implicação 2", "Implicação 3", "Implicação 4"],
  "overall_attractiveness": "Alta/Média/Baixa",
  "summary": "Resumo executivo conciso"
}`

	// 3. TAM-SAM-SOM com Estimativa Obrigatória - Modelo RACE/AIM [cite: 658, 690] - CONSTRAINED
	FrameworkTamSamSomPrompt = `Dimensione o mercado usando ESTIMATIVA OBRIGATÓRIA com o modelo RACE/AIM.
Contexto: {{COMPANY_DATA}}
Inteligência: {{ENRICHMENT_DATA}}
Macro-Contexto: {{MACRO_CONTEXT}}

REGRA CRÍTICA: SEMPRE forneça estimativas em FAIXA DE VALORES. NUNCA retorne "Dados insuficientes".
Seja TRANSPARENTE sobre a base da estimativa usando confidence_level e data_sources_used.

METODOLOGIA DE ESTIMATIVA (use na ordem de preferência):
1. **Dados Diretos**: Se encontrar tamanho de mercado do setor, use diretamente
2. **Proxy por Concorrentes**: Se souber receita de concorrentes, extrapole pelo market share
3. **Proxy por Funcionários**: Funcionários × receita média por funcionário do setor
4. **Benchmarks Setoriais Brasil** (use se não houver dados específicos):
   - Agronegócio/AgTech: TAM ~R$ 1.2T, crescimento +8-12% a.a.
   - Tecnologia/SaaS B2B: TAM ~R$ 50-80B, crescimento +18-25% a.a.
   - Serviços Financeiros: TAM ~R$ 400-600B, crescimento +10-15% a.a.
   - Varejo/E-commerce: TAM ~R$ 1.5-2T, crescimento +12-18% a.a.
   - Saúde/HealthTech: TAM ~R$ 300-500B, crescimento +15-20% a.a.
   - Educação/EdTech: TAM ~R$ 150-250B, crescimento +20-25% a.a.
   - Logística: TAM ~R$ 200-350B, crescimento +10-15% a.a.

CÁLCULOS:
- TAM: Mercado total do setor no Brasil (ajuste para região se especificado)
- SAM: TAM × % do segmento-alvo (considere constraints geográficos/regulatórios)
- SOM: SAM × market share realista (1-5% startups, 5-15% empresas estabelecidas)
- CAGR: Use growth_rate do setor do macro_context, ajuste ±3% baseado em trends

NÍVEIS DE CONFIANÇA (confidence_level 0-100):
- 80-100: Dados diretos do setor/empresa encontrados
- 50-79: Proxy calculations com dados parciais
- 30-49: Principalmente extrapolação de benchmarks setoriais
- <30: Estimativa especulativa (adicione caveat_message)

MODELAGEM DE CENÁRIOS (OBRIGATÓRIO):
Para cada métrica (TAM, SAM, SOM), forneça 3 cenários:

1. CONSERVADOR (probabilidade 30%):
   - Premissas pessimistas: execução com desafios, mercado competitivo
   - Market share: Startups 0.5-2%, Empresas 3-8%
   - Use este como BASELINE para planejamento

2. REALISTA (probabilidade 50%):
   - Premissas moderadas: boa execução, condições normais
   - Market share: Startups 2-5%, Empresas 8-15%
   - Principal referência para decisões

3. OTIMISTA (probabilidade 20%):
   - Premissas favoráveis: execução excepcional, vantagens competitivas
   - Market share: Startups 5-10%, Empresas 15-25%
   - Apenas se condições ideais se confirmarem

REGRA CRÍTICA: O cenário CONSERVADOR deve ser viável e sustentável.
Se o cenário conservador não gera valor, reavalie a estratégia.

VALIDAÇÃO DE REALISMO:
- Compare SOM com receitas típicas do setor no Brasil
- Associações setoriais: R$ 2-10M/ano é teto realista
- Startups early-stage: R$ 500k-5M/ano nos primeiros 3 anos
- Empresas estabelecidas PME: R$ 5-50M/ano

REGRAS:
1. Liste EXATAMENTE 3 premissas críticas com fonte/cálculo.
2. Máximo 100 caracteres por premissa.
3. SEMPRE forneça faixa de valores (ex: "R$ 800M - 1.2B"), NUNCA valor único.
4. Se confidence_level < 50, adicione caveat_message com aviso.

Retorne JSON (valores em PT-BR):
{
  "tam": "R$ X - Y (faixa obrigatória, ex: R$ 800M - 1.2B)",
  "sam": "R$ X - Y",
  "som": "R$ X - Y",
  "assumptions": ["Premissa 1 com fonte/cálculo", "Premissa 2", "Premissa 3"],
  "cagr": "+X% a.a. (baseado em: fonte)",
  "confidence_level": 65,
  "estimation_method": "dados_diretos|proxy_concorrentes|proxy_funcionarios|benchmark_setorial",
  "calculation_notes": "Explicação breve do cálculo usado (max 200 chars)",
  "data_sources_used": {
    "direct_data": ["Dado direto encontrado (se houver)"],
    "proxy_calculations": ["Cálculo proxy usado (se houver)"],
    "benchmark_extrapolations": ["Benchmark aplicado (se houver)"]
  },
  "tam_scenarios": {
    "conservative": {"value_range": "R$ X-Y", "probability": 30},
    "realistic": {"value_range": "R$ X-Y", "probability": 50},
    "optimistic": {"value_range": "R$ X-Y", "probability": 20}
  },
  "som_scenarios": {
    "conservative": {"value_range": "R$ X-Y", "market_share": "X%", "probability": 30},
    "realistic": {"value_range": "R$ X-Y", "market_share": "X%", "probability": 50},
    "optimistic": {"value_range": "R$ X-Y", "market_share": "X%", "probability": 20}
  },
  "baseline_recommendation": "conservative ou realistic (qual usar como base)",
  "caveat_message": "⚠️ Aviso se confidence < 50 (opcional)",
  "summary": "Síntese de potencial (max 200 chars)"
}`

	// 4. SWOT com Confidence Levels - Modelo LIFT [cite: 317] - CONSTRAINED
	FrameworkSWOTPrompt = `Realize uma análise SWOT (Modelo LIFT) focada em itens acionáveis com níveis de confiança e fonte.
Contexto Interno: {{COMPANY_DATA}}
Contexto Externo (Layer 1): {{PESTEL_INSIGHTS}} | {{PORTER_INSIGHTS}}

REGRAS DE FORMATAÇÃO (CRÍTICO):
1. Liste EXATAMENTE 4 itens por quadrante. Nem mais, nem menos.
2. Cada item deve ter no MÁXIMO 100 caracteres no campo "content".
3. Ordene por impacto: O item 1 deve ser o mais crítico para a estratégia.
4. Para cada item, adicione:
   - confidence: "Alta" | "Média" | "Baixa" (baseado na qualidade dos dados)
   - source: "fato" | "análise de mercado" | "estimativa" | "feedback de clientes"

Retorne JSON (valores em PT-BR):
{
  "strengths": [
    {"content": "Força crítica 1 (max 100 chars)", "confidence": "Alta", "source": "fato"},
    {"content": "Força 2", "confidence": "Média", "source": "análise de mercado"},
    {"content": "Força 3", "confidence": "Alta", "source": "feedback de clientes"},
    {"content": "Força 4", "confidence": "Baixa", "source": "estimativa"}
  ],
  "weaknesses": [
    {"content": "...", "confidence": "Alta|Média|Baixa", "source": "fato|análise de mercado|estimativa|feedback de clientes"},
    {"content": "...", "confidence": "...", "source": "..."},
    {"content": "...", "confidence": "...", "source": "..."},
    {"content": "...", "confidence": "...", "source": "..."}
  ],
  "opportunities": [
    {"content": "...", "confidence": "...", "source": "..."},
    {"content": "...", "confidence": "...", "source": "..."},
    {"content": "...", "confidence": "...", "source": "..."},
    {"content": "...", "confidence": "...", "source": "..."}
  ],
  "threats": [
    {"content": "...", "confidence": "...", "source": "..."},
    {"content": "...", "confidence": "...", "source": "..."},
    {"content": "...", "confidence": "...", "source": "..."},
    {"content": "...", "confidence": "...", "source": "..."}
  ],
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

	// 7. GROWTH HACKING - LEAP + SCALE Loops [cite: 1164] - CONSTRAINED
	FrameworkGrowthHackingPrompt = `Crie estratégias de Growth Hacking com LEAP Loop (Aquisição) e SCALE Loop (Monetização).
Contexto: {{COMPANY_DATA}}
Meta: Acelerar tração e maximizar valor do cliente.

LEAP LOOP (Aquisição):
1. **Land** (Aterrissar): Como prospects chegam?
2. **Engage** (Engajar): Como envolvê-los?
3. **Activate** (Ativar): Como converter em usuários?
4. **Propagate** (Propagar): Como gerar viralidade?

SCALE LOOP (Monetização):
1. **Satisfy** (Satisfazer): Como entregar valor?
2. **Convert** (Converter): Como monetizar?
3. **Loop Back** (Retornar): Como gerar recorrência?
4. **Expand** (Expandir): Como aumentar LTV?

REGRAS:
1. Para cada loop, defina os 4 passos com clareza (max 120 chars cada).
2. Identifique métricas-chave para cada loop.
3. Identifique o principal bottleneck (gargalo) de cada loop.

Retorne JSON (valores em PT-BR):
{
  "leap_loop": {
    "name": "LEAP Loop",
    "type": "acquisition",
    "steps": ["Land: Como chegam (max 120 chars)", "Engage: Como engajar", "Activate: Como converter", "Propagate: Como viralizar"],
    "metrics": ["CAC", "Taxa de Conversão", "Coeficiente Viral"],
    "bottleneck": "Principal gargalo no funil de aquisição"
  },
  "scale_loop": {
    "name": "SCALE Loop",
    "type": "monetization",
    "steps": ["Satisfy: Como entregar valor", "Convert: Como monetizar", "Loop Back: Como gerar recorrência", "Expand: Como aumentar LTV"],
    "metrics": ["LTV", "Taxa de Recompra", "Receita de Expansão"],
    "bottleneck": "Principal gargalo na monetização"
  },
  "summary": "Resumo da estratégia de crescimento (max 200 chars)"
}`

	// 8. CENÁRIOS com Probabilidades - Modelo FUTUREMAP [cite: 1426] - CONSTRAINED
	FrameworkScenariosPrompt = `Realize uma Análise de Cenários (Modelo FUTUREMAP) com probabilidades e táticas de mitigação.
Contexto: {{COMPANY_DATA}}
Incertezas: {{PESTEL_INSIGHTS}}
Macro-Contexto: {{MACRO_CONTEXT}}

INSTRUÇÕES CRÍTICAS - Use Macro-Contexto para criar cenários realistas:
1. **Cenário Otimista (20%)**: Baseie em economic_outlook positivo, industry trends favoráveis.
2. **Cenário Realista (60%)**: Use dados atuais (interest_rate, inflation_rate, growth_rate).
3. **Cenário Pessimista (20%)**: Considere recent_policy_changes negativas, emerging_threats, economic risks.
4. **Required Actions**: Para cada cenário, liste 3-4 ações específicas a tomar se materializar.
5. **Mitigation Tactics**: Liste 4 estratégias para mitigar riscos gerais.
6. **Early Warning Signals**: Liste indicadores concretos do macro_context que sinalizam mudanças.

REGRAS DE FORMATAÇÃO (CRÍTICO):
1. Para cada cenário, escreva um texto denso de MÁXIMO 450 caracteres (aprox. 60 palavras).
2. Seja narrativo mas direto. Descreva o "mundo" e a "implicância".
3. Cite números e eventos específicos do Macro-Contexto (ex: "Se SELIC cair para 9%...", "Se reforma tributária for aprovada...").
4. Probabilidades devem somar 100% (tipicamente 20% otimista, 60% realista, 20% pessimista).

Retorne JSON (valores em PT-BR):
{
  "optimistic": {
    "name": "Cenário Otimista",
    "probability": 20,
    "description": "Texto do cenário otimista (max 450 chars)",
    "required_actions": ["Ação específica 1 se cenário ocorrer", "Ação 2", "Ação 3"]
  },
  "realist": {
    "name": "Cenário Realista",
    "probability": 60,
    "description": "Texto do cenário realista (max 450 chars)",
    "required_actions": ["Ação específica 1 se cenário ocorrer", "Ação 2", "Ação 3", "Ação 4"]
  },
  "pessimistic": {
    "name": "Cenário Pessimista",
    "probability": 20,
    "description": "Texto do cenário pessimista (max 450 chars)",
    "required_actions": ["Ação específica 1 se cenário ocorrer", "Ação 2", "Ação 3"]
  },
  "mitigation_tactics": ["Tática de mitigação 1", "Tática 2", "Tática 3", "Tática 4"],
  "early_warning_signals": ["Sinal 1", "Sinal 2", "Sinal 3"],
  "summary": "Resumo de riscos e preparação estratégica"
}`

	// 9. OKRs Plano 90 Dias - Modelo FOCUS [cite: 1041] - CONSTRAINED
	// IMPORTANT: Use Decision Matrix recommendations as input for OKR alignment
	FrameworkOKRsPrompt = `Defina um Plano Estratégico de 90 Dias (Modelo FOCUS) com marcos mensais.
Contexto: {{COMPANY_DATA}}
Estratégia: {{BLUE_OCEAN_INSIGHTS}}
Recomendações Priorizadas: {{DECISION_MATRIX_RECOMMENDATIONS}}

REGRA CRÍTICA: Os OKRs devem estar ALINHADOS com as Recomendações Priorizadas da Matriz de Decisão.
Cada objetivo mensal deve contribuir diretamente para implementar as recomendações estratégicas.

FORMATO 90 DIAS COM MARCOS MENSAIS:
- Os 90 dias são divididos em 3 meses (Mês 1, Mês 2, Mês 3)
- Cada mês tem foco progressivo: Fundação → Crescimento → Consolidação
- Use datas relativas (não hardcoded como "Q1 2025")

ESTRUTURA DE FASES ESTENDIDA (OBRIGATÓRIO):
Além do plano de 90 dias, forneça visão de 12 meses em 3 fases:

FASE 1 - FUNDAÇÃO (Meses 1-3):
Objetivo: Estabelecer bases sólidas antes de escalar
- Ajustes estruturais, jurídicos, organizacionais
- Validação de premissas críticas
- Quick wins de baixo risco
- NÃO prometa resultados de mercado nesta fase

FASE 2 - VALIDAÇÃO (Meses 4-6):
Objetivo: Testar hipóteses com escopo limitado
- Pilotos controlados (1-3 casos)
- Coleta de feedback real
- Ajustes baseados em aprendizado
- Métricas de validação, não de escala

FASE 3 - ESCALA (Meses 7-12):
Objetivo: Expandir o que foi validado
- Somente após validação bem-sucedida
- Crescimento gradual e sustentável
- Métricas de crescimento realistas

AVALIAÇÃO DE CAPACIDADE (OBRIGATÓRIO):
Antes de definir KRs, avalie:
- Equipe disponível: quantidade, skills, dedicação
- Orçamento realista: não assuma recursos ilimitados
- Dependências externas: parcerias, aprovações, integrações
- Resistência interna: stakeholders que podem bloquear

Se capacidade < ambição → REDUZA o escopo, não o prazo.

REGRAS DE REALISMO:
- Cada KR deve ter responsável identificável
- Prazos devem considerar férias, feriados, imprevistos
- Adicione 30% de buffer para contingências
- Prefira "completar X" sobre "aumentar Y%" quando incerto

REGRAS:
1. Crie EXATAMENTE 3 blocos mensais (Mês 1, Mês 2, Mês 3).
2. Para cada mês, defina 1 Objetivo e EXATAMENTE 3 Key Results (KRs).
3. Use números nos KRs (Ex: "Aumentar receita em 20%"). Max 100 chars por KR.
4. Adicione estimativa de investimento e timeline para cada mês.
5. Progressão lógica: Mês 1 = fundação, Mês 2 = crescimento, Mês 3 = consolidação/escala.
6. Vincule cada objetivo às recomendações da Matriz de Decisão (campo "aligned_recommendation").

Retorne JSON (valores em PT-BR):
{
  "plan_90_days": [
    {
      "month": "Mês 1",
      "focus": "Fundação",
      "objective": "Objetivo do Mês 1 (foco em fundação/validação)",
      "key_results": ["KR1 (max 100 chars)", "KR2", "KR3"],
      "investment": "R$ 10-20 mil (ou estimativa apropriada)",
      "aligned_recommendation": "Título da recomendação prioritária #1 ou #2 que este mês implementa"
    },
    {
      "month": "Mês 2",
      "focus": "Crescimento",
      "objective": "Objetivo do Mês 2 (foco em crescimento)",
      "key_results": ["KR1", "KR2", "KR3"],
      "investment": "R$ 20-40 mil",
      "aligned_recommendation": "Título da recomendação que este mês implementa"
    },
    {
      "month": "Mês 3",
      "focus": "Consolidação",
      "objective": "Objetivo do Mês 3 (foco em consolidação/escala)",
      "key_results": ["KR1", "KR2", "KR3"],
      "investment": "R$ 30-50 mil",
      "aligned_recommendation": "Título da recomendação que este mês implementa"
    }
  ],
  "total_investment": "R$ 60-110 mil (soma dos 3 meses)",
  "success_metrics": ["Métrica de sucesso 1 para os 90 dias", "Métrica 2", "Métrica 3"],
  "execution_phases": {
    "foundation": {
      "duration": "Meses 1-3",
      "focus": "Estruturação",
      "objectives": ["Objetivo de fundação 1", "Objetivo 2"],
      "key_results": ["KR mensurável 1", "KR 2"],
      "investment_range": "R$ X-Y",
      "capacity_requirements": {
        "team": "X pessoas",
        "skills_needed": ["Skill 1", "Skill 2"],
        "dependencies": ["Dependência 1", "Dependência 2"]
      }
    },
    "validation": {
      "duration": "Meses 4-6",
      "focus": "Pilotos",
      "prerequisites": ["Fase 1 concluída", "Aprovações obtidas"],
      "objectives": ["Objetivo de validação 1", "Objetivo 2"],
      "key_results": ["KR de validação 1", "KR 2"],
      "investment_range": "R$ X-Y"
    },
    "scale": {
      "duration": "Meses 7-12",
      "focus": "Expansão controlada",
      "prerequisites": ["Validação bem-sucedida", "Métricas atingidas"],
      "objectives": ["Objetivo de escala 1", "Objetivo 2"],
      "key_results": ["KR de escala 1", "KR 2"],
      "investment_range": "R$ X-Y"
    }
  },
  "capacity_assessment": {
    "team_readiness": "Alta|Média|Baixa",
    "budget_adequacy": "Suficiente|Limitado|Insuficiente",
    "stakeholder_alignment": "Alinhado|Parcial|Resistência",
    "blockers": ["Bloqueio identificado 1", "Bloqueio 2 (se houver)"]
  },
  "summary": "Roteiro de implementação 90 dias (max 200 chars)"
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

	// 11. MATRIZ DECISÃO com Recomendações Priorizadas - Modelo ANALYTICA [cite: 1303] - CONSTRAINED
	FrameworkDecisionMatrixPrompt = `Crie uma Matriz de Decisão Multicriterial (Modelo ANALYTICA) com recomendações priorizadas e ciclo de revisão.
Dilema: "Qual a melhor estratégia de crescimento?"
Contexto: {{COMPANY_DATA}}
Cenários: {{SCENARIO_INSIGHTS}}

AVALIAÇÃO DE VIABILIDADE JURÍDICA (OBRIGATÓRIO):
Para cada recomendação priorizada, avalie:

1. VIABILIDADE LEGAL:
   - Requer alteração estatutária? (Alto risco se sim)
   - Conflita com regulamentação setorial?
   - Implica riscos de responsabilidade civil?
   - Exige licenças/autorizações específicas?

2. COMPLEXIDADE INSTITUCIONAL:
   - Depende de aprovação de assembleia?
   - Envolve múltiplas partes (federação, regionais)?
   - Há precedentes similares no setor?

3. CLASSIFICAÇÃO DE RISCO JURÍDICO:
   - BAIXO: Operacional, não requer mudanças estruturais
   - MÉDIO: Requer ajustes contratuais ou regimentais
   - ALTO: Requer alteração estatutária ou envolve litígio potencial
   - CRÍTICO: Pode invalidar decisões ou gerar responsabilização

REGRA: Recomendações de risco ALTO ou CRÍTICO devem incluir:
- Parecer jurídico recomendado antes de execução
- Plano de mitigação específico
- Alternativa de menor risco

GATILHOS EXTRAORDINÁRIOS (adicionar):
- Mudança regulatória que exija >10% de ajuste de custo
- Nova exigência de licenciamento que bloqueie operação
- Decisão judicial que afete modelo de negócio
- Falha de compliance LGPD/setorial

REGRAS:
1. Liste 3 Alternativas claras.
2. Liste 3 Critérios de decisão (ex: ROI, Risco, Tempo).
3. Defina a melhor opção com score (ex: "7.3/10") e comparação (ex: "+23% acima da segunda opção").
4. Crie 3 Recomendações Prioritárias (prioridade 1, 2, 3) com timeline, budget E legal_feasibility.
5. Defina ciclo de revisão (frequência e gatilhos extraordinários).
6. Liste 5-7 métricas para monitorar execução.

Retorne JSON (valores em PT-BR):
{
  "alternatives": ["Alternativa A", "Alternativa B", "Alternativa C"],
  "criteria": ["Critério 1", "Critério 2", "Critério 3"],
  "recommended_option": "Nome da melhor alternativa",
  "score": "7.3/10",
  "score_comparison": "+23% acima da segunda opção",
  "priority_recommendations": [
    {
      "priority": 1,
      "title": "Recomendação #1",
      "description": "Descrição detalhada da recomendação",
      "timeline": "9-12 meses",
      "budget": "R$150-250k",
      "legal_feasibility": {
        "risk_level": "Baixo|Médio|Alto|Crítico",
        "requires_statutory_change": false,
        "requires_legal_opinion": false,
        "regulatory_dependencies": ["Licença X", "Aprovação Y (se houver)"],
        "mitigation_plan": "Plano de mitigação se risco Alto/Crítico"
      }
    },
    {
      "priority": 2,
      "title": "Recomendação #2",
      "description": "Descrição detalhada",
      "timeline": "6-9 meses",
      "budget": "R$80-120k",
      "legal_feasibility": {
        "risk_level": "Baixo|Médio|Alto|Crítico",
        "requires_statutory_change": false,
        "requires_legal_opinion": false,
        "regulatory_dependencies": [],
        "mitigation_plan": ""
      }
    },
    {
      "priority": 3,
      "title": "Recomendação #3",
      "description": "Descrição detalhada",
      "timeline": "3-6 meses",
      "budget": "R$40-60k",
      "legal_feasibility": {
        "risk_level": "Baixo|Médio|Alto|Crítico",
        "requires_statutory_change": false,
        "requires_legal_opinion": false,
        "regulatory_dependencies": [],
        "mitigation_plan": ""
      }
    }
  ],
  "review_cycle": {
    "frequency": "Trimestral",
    "extraordinary_triggers": ["Mudança regulatória >10% custo", "Bloqueio de licenciamento", "Decisão judicial relevante", "Falha de compliance LGPD/setorial", "Entrada de competidor relevante"]
  },
  "monitoring_metrics": ["Métrica 1", "Métrica 2", "Métrica 3", "Métrica 4", "Métrica 5"],
  "final_recommendation": "A melhor opção é... (max 300 chars)",
  "summary": "Justificativa e próximos passos"
}`

	// SYNTHESIS com Desafio Central - Modelo PAR [cite: 200] - CONSTRAINED
	SynthesisPrompt = `Você é um Líder Exponencial. Sintetize os 11 frameworks em um Resumo Executivo estruturado.
Contexto: {{COMPANY_DATA}}
Resultados: {{ALL_FRAMEWORK_SUMMARIES}}

VALIDAÇÃO DE CONSISTÊNCIA (OBRIGATÓRIO):
Antes de finalizar, verifique:

1. ALINHAMENTO FINANCEIRO:
   - OKRs são atingíveis com o SOM conservador?
   - Investimento proposto < receita projetada no cenário realista?
   - Se não → SINALIZAR INCONSISTÊNCIA

2. ALINHAMENTO DE CAPACIDADE:
   - Equipe disponível suporta os OKRs definidos?
   - Há skills críticos faltando?
   - Se não → AJUSTAR ESCOPO ou SINALIZAR

3. ALINHAMENTO JURÍDICO:
   - Recomendações de alto risco têm mitigação?
   - Há bloqueios legais não endereçados?
   - Se sim → DESTACAR COMO RISCO CRÍTICO

4. REALISMO GERAL:
   - Este plano seria aprovado por um conselho experiente?
   - As premissas são defensáveis?
   - Se dúvida → SER CONSERVADOR

REGRAS DE FORMATAÇÃO (CRÍTICO PARA PDF):
1. **Central Challenge**: Identifique O desafio estratégico central (max 200 chars).
2. **Executive Summary**: Máximo 400 caracteres. Um parágrafo de impacto.
3. **Main Findings**: 4 insights principais baseados no SWOT (Strength, Weakness, Opportunity, Threat).
4. **Important Notes**: 2-3 observações críticas ou warnings (max 120 chars cada).
5. **Key Findings**: Exatamente 3 insights estratégicos (max 150 chars cada).
6. **Strategic Priorities**: Exatamente 3 prioridades (max 100 chars cada).
7. **Roadmap**: Exatamente 3 passos macro (Curto, Médio, Longo Prazo).

Retorne JSON (valores em PT-BR):
{
  "central_challenge": "O principal desafio estratégico que a empresa enfrenta (max 200 chars)",
  "executive_summary": "Texto de alto impacto (max 400 chars)",
  "main_findings": [
    "Força principal identificada (do SWOT)",
    "Fraqueza principal identificada (do SWOT)",
    "Oportunidade principal identificada (do SWOT)",
    "Ameaça principal identificada (do SWOT)"
  ],
  "important_notes": [
    "Observação crítica 1 (max 120 chars)",
    "Observação crítica 2 (max 120 chars)",
    "Observação crítica 3 (max 120 chars, opcional)"
  ],
  "key_findings": ["Insight estratégico 1 (max 150 chars)", "Insight 2", "Insight 3"],
  "strategic_priorities": ["Prioridade 1 (max 100 chars)", "Prioridade 2", "Prioridade 3"],
  "roadmap": ["Curto Prazo (3-6 meses): ...", "Médio Prazo (6-12 meses): ...", "Longo Prazo (12+ meses): ..."],
  "overall_recommendation": "Conclusão final curta e acionável",
  "consistency_validation": {
    "financial_alignment": true,
    "capacity_alignment": true,
    "legal_alignment": true,
    "overall_realism_score": 8,
    "flags": ["Flag de inconsistência 1 (se houver)", "Flag 2 (se houver)"]
  }
}`
)
