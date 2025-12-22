-- Migration: v2_037_prompts_layer2.sql
-- Description: Real prompts for Layer 2 frameworks (SWOT, Benchmarking)

-- =============================================================================
-- SWOT (Layer 2)
-- =============================================================================
UPDATE frameworks SET
  prompt_system = $sys$Você é o **10XMentorAI**, especializado em diagnóstico organizacional.

Seu Papel: Análise SWOT (Modelo LIFT) com níveis de confiança.

Regras:
- EXATAMENTE 4 itens por quadrante
- Max 100 chars por item
- Confidence + Source obrigatórios para cada item
- Ordenado por impacto (item 1 = mais crítico)$sys$,
  prompt_user = $usr$Realize análise SWOT focada em itens acionáveis.

DESAFIO: {{.challenge_context}}
TIPO: {{.challenge_type}} ({{.challenge_category}})

CONTEXTO INTERNO: {{COMPANY_DATA}}

CONTEXTO EXTERNO (Layer 1):
{{PESTEL_INSIGHTS}}
{{PORTER_INSIGHTS}}

REGRAS:
- EXATAMENTE 4 itens por quadrante
- Max 100 chars no campo "content"
- Ordene por impacto
- Cada item precisa de:
  - confidence: "Alta" | "Média" | "Baixa"
  - source: "fato" | "análise de mercado" | "estimativa" | "feedback de clientes"

OUTPUT JSON:
{
  "strengths": [
    {"content": "Força 1 (max 100 chars)", "confidence": "Alta", "source": "fato"},
    {"content": "Força 2", "confidence": "Média", "source": "análise de mercado"},
    {"content": "Força 3", "confidence": "Alta", "source": "feedback de clientes"},
    {"content": "Força 4", "confidence": "Baixa", "source": "estimativa"}
  ],
  "weaknesses": [
    {"content": "Fraqueza 1", "confidence": "Alta", "source": "fato"},
    {"content": "Fraqueza 2", "confidence": "Média", "source": "análise de mercado"},
    {"content": "Fraqueza 3", "confidence": "Baixa", "source": "estimativa"},
    {"content": "Fraqueza 4", "confidence": "Média", "source": "feedback de clientes"}
  ],
  "opportunities": [
    {"content": "Oportunidade 1", "confidence": "Alta", "source": "análise de mercado"},
    {"content": "Oportunidade 2", "confidence": "Média", "source": "fato"},
    {"content": "Oportunidade 3", "confidence": "Baixa", "source": "estimativa"},
    {"content": "Oportunidade 4", "confidence": "Média", "source": "análise de mercado"}
  ],
  "threats": [
    {"content": "Ameaça 1", "confidence": "Alta", "source": "fato"},
    {"content": "Ameaça 2", "confidence": "Média", "source": "análise de mercado"},
    {"content": "Ameaça 3", "confidence": "Alta", "source": "fato"},
    {"content": "Ameaça 4", "confidence": "Baixa", "source": "estimativa"}
  ],
  "summary": "Resumo da posição estratégica (max 200 chars)"
}$usr$
WHERE code = 'swot';

-- =============================================================================
-- BENCHMARKING (Layer 2)
-- =============================================================================
UPDATE frameworks SET
  prompt_system = $sys$Você é o **10XMentorAI**, especializado em análise competitiva e benchmarking.

Seu Papel: Benchmarking Competitivo (Modelo COMPARE).

Regras:
- Use concorrentes reais do COMPANY_DATA
- 3 gaps + 3 práticas, exatamente
- Max 120 chars por item
- Foco em diferenciais acionáveis$sys$,
  prompt_user = $usr$Realize Benchmarking Competitivo.

DESAFIO: {{.challenge_context}}
TIPO: {{.challenge_type}} ({{.challenge_category}})

CONTEXTO: {{COMPANY_DATA}}

REGRAS:
- Exatamente 3 Gaps de Performance
- Exatamente 3 Melhores Práticas
- Max 120 chars por item
- Baseie-se nos concorrentes do COMPANY_DATA

OUTPUT JSON:
{
  "competitors_analyzed": ["Concorrente A", "Concorrente B"],
  "performance_gaps": [
    "Gap crítico 1 (max 120 chars)",
    "Gap 2",
    "Gap 3"
  ],
  "best_practices": [
    "Prática 1 (max 120 chars)",
    "Prática 2",
    "Prática 3"
  ],
  "summary": "Resumo comparativo"
}$usr$
WHERE code = 'benchmarking';

-- Verify
DO $$ BEGIN RAISE NOTICE 'v2_037: Layer 2 prompts (SWOT, Benchmarking) updated'; END $$;
