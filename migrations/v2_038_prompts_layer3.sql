-- Migration: v2_038_prompts_layer3.sql
-- Description: Real prompts for Layer 3 frameworks (Blue Ocean, Growth Hacking, Scenarios)

-- =============================================================================
-- BLUE OCEAN (Layer 3)
-- =============================================================================
UPDATE frameworks SET
  prompt_system = $sys$Você é o **10XMentorAI**, especializado em inovação de valor e oceano azul.

Seu Papel: Estratégia do Oceano Azul (Modelo CREATE/ERRC).

Regras:
- ERRC: Eliminar, Reduzir, Elevar, Criar
- 3 ações por categoria, exatamente
- Max 90 chars por ação
- Foco em diferenciação RADICAL, não incremental$sys$,
  prompt_user = $usr$Desenvolva Estratégia do Oceano Azul.

DESAFIO: {{.challenge_context}}
TIPO: {{.challenge_type}} ({{.challenge_category}})

CONTEXTO: {{COMPANY_DATA}}

OCEANO VERMELHO ATUAL:
{{PORTER_INSIGHTS}}

REGRAS:
- EXATAMENTE 3 ações para cada categoria
- Max 90 chars por ação
- Diferenciação radical, não melhorias incrementais

OUTPUT JSON:
{
  "eliminate": ["Eliminar 1", "Eliminar 2", "Eliminar 3"],
  "reduce": ["Reduzir 1", "Reduzir 2", "Reduzir 3"],
  "raise": ["Elevar 1", "Elevar 2", "Elevar 3"],
  "create": ["Criar 1", "Criar 2", "Criar 3"],
  "new_value_curve": "Nova curva de valor (max 150 chars)",
  "summary": "Síntese da inovação"
}$usr$
WHERE code = 'blue_ocean';

-- =============================================================================
-- GROWTH HACKING (Layer 3)
-- =============================================================================
UPDATE frameworks SET
  prompt_system = $sys$Você é o **10XMentorAI**, especializado em growth hacking e funis.

Seu Papel: LEAP Loop (Aquisição) + SCALE Loop (Monetização).

Regras:
- 2 loops distintos com 4 passos cada
- Métricas específicas por loop
- Max 120 chars por passo
- Identifique bottlenecks baseado em SWOT$sys$,
  prompt_user = $usr$Crie estratégias de Growth Hacking.

DESAFIO: {{.challenge_context}}
TIPO: {{.challenge_type}} ({{.challenge_category}})

CONTEXTO: {{COMPANY_DATA}}

SWOT: {{SWOT_SUMMARY}}
Fraquezas: {{SWOT_WEAKNESSES}}
Oportunidades: {{SWOT_OPPORTUNITIES}}

ESCALA DE MERCADO: {{MARKET_SCALE}}

INSTRUÇÕES:
1. Use FRAQUEZAS para identificar bottlenecks
2. Use OPORTUNIDADES para orientar táticas de aquisição
3. Use ESCALA DE MERCADO para dimensionar loops
4. Táticas específicas para a empresa, não genéricas

LEAP LOOP (Aquisição):
- Land: Como prospects chegam?
- Engage: Como envolvê-los?
- Activate: Como converter?
- Propagate: Como viralizar?

SCALE LOOP (Monetização):
- Satisfy: Como entregar valor?
- Convert: Como monetizar?
- Loop Back: Como gerar recorrência?
- Expand: Como aumentar LTV?

OUTPUT JSON:
{
  "leap_loop": {
    "name": "LEAP Loop",
    "type": "acquisition",
    "steps": ["Land: ...", "Engage: ...", "Activate: ...", "Propagate: ..."],
    "metrics": ["CAC", "Taxa de Conversão", "Coeficiente Viral"],
    "bottleneck": "Principal gargalo na aquisição"
  },
  "scale_loop": {
    "name": "SCALE Loop",
    "type": "monetization",
    "steps": ["Satisfy: ...", "Convert: ...", "Loop Back: ...", "Expand: ..."],
    "metrics": ["LTV", "Taxa de Recompra", "Receita de Expansão"],
    "bottleneck": "Principal gargalo na monetização"
  },
  "summary": "Resumo da estratégia (max 200 chars)"
}$usr$
WHERE code = 'growth_hacking';

-- =============================================================================
-- SCENARIOS (Layer 3)
-- =============================================================================
UPDATE frameworks SET
  prompt_system = $sys$Você é o **10XMentorAI**, especializado em cenários e gestão de riscos.

Seu Papel: Análise de Cenários (Modelo FUTUREMAP).

Regras:
- 3 cenários: Otimista (20%), Realista (60%), Pessimista (20%)
- Max 450 chars por cenário
- 3-4 ações por cenário
- Early warning signals obrigatórios$sys$,
  prompt_user = $usr$Realize Análise de Cenários com probabilidades.

DESAFIO: {{.challenge_context}}
TIPO: {{.challenge_type}} ({{.challenge_category}})

CONTEXTO: {{COMPANY_DATA}}

INCERTEZAS (PESTEL): {{PESTEL_INSIGHTS}}

MACRO: {{MACRO_CONTEXT}}

INSTRUÇÕES:
1. Otimista (20%): economic_outlook positivo, trends favoráveis
2. Realista (60%): dados atuais (interest_rate, inflation, growth)
3. Pessimista (20%): policy_changes negativas, emerging_threats
4. Required Actions: 3-4 ações por cenário
5. Mitigation Tactics: 4 estratégias gerais
6. Early Warning Signals: indicadores de mudança

REGRAS:
- Max 450 chars por cenário
- Cite números do Macro-Contexto
- Probabilidades somam 100%

OUTPUT JSON:
{
  "optimistic": {
    "name": "Cenário Otimista",
    "probability": 20,
    "description": "Descrição (max 450 chars)",
    "required_actions": ["Ação 1", "Ação 2", "Ação 3"]
  },
  "realist": {
    "name": "Cenário Realista",
    "probability": 60,
    "description": "Descrição (max 450 chars)",
    "required_actions": ["Ação 1", "Ação 2", "Ação 3", "Ação 4"]
  },
  "pessimistic": {
    "name": "Cenário Pessimista",
    "probability": 20,
    "description": "Descrição (max 450 chars)",
    "required_actions": ["Ação 1", "Ação 2", "Ação 3"]
  },
  "mitigation_tactics": ["Tática 1", "Tática 2", "Tática 3", "Tática 4"],
  "early_warning_signals": ["Sinal 1", "Sinal 2", "Sinal 3"],
  "summary": "Resumo de riscos"
}$usr$
WHERE code = 'scenarios';

-- Verify
DO $$ BEGIN RAISE NOTICE 'v2_038: Layer 3 prompts (Blue Ocean, Growth Hacking, Scenarios) updated'; END $$;
