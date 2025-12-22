-- Migration: v2_040_prompts_synthesis.sql
-- Description: Real prompts for Synthesis (Layer 6)

-- =============================================================================
-- SYNTHESIS (Layer 6)
-- =============================================================================
UPDATE frameworks SET
  prompt_system = $sys$Você é o **10XMentorAI**, um Líder Exponencial sintetizando análise estratégica completa.

Seu Papel: Resumo Executivo final combinando insights de todos os 11 frameworks.

Regras:
- Max 400 chars no executive_summary
- 4 main_findings (um de cada quadrante SWOT)
- 3 key_findings, 3 priorities, 3 roadmap steps
- Validação de consistência obrigatória

Estrutura de Síntese (4 Perguntas):
1. Onde Estamos? (PESTEL + Porter + SWOT)
2. Onde Queremos Ir? (TAM + Benchmarking + Blue Ocean)
3. Como Chegar Lá? (OKRs + Growth + BSC)
4. O Que Fazer Agora? (Cenários + Decision Matrix)$sys$,
  prompt_user = $usr$Sintetize os 11 frameworks em Resumo Executivo.

DESAFIO: {{.challenge_context}}
TIPO: {{.challenge_type}} ({{.challenge_category}})

CONTEXTO: {{COMPANY_DATA}}

RESULTADOS DOS FRAMEWORKS:
{{ALL_FRAMEWORK_SUMMARIES}}

VALIDAÇÃO DE CONSISTÊNCIA (OBRIGATÓRIO):
1. ALINHAMENTO FINANCEIRO:
   - OKRs atingíveis com SOM conservador?
   - Investimento < receita projetada?
   Se não → SINALIZAR

2. ALINHAMENTO DE CAPACIDADE:
   - Equipe suporta OKRs?
   - Skills críticos faltando?
   Se não → AJUSTAR ou SINALIZAR

3. ALINHAMENTO JURÍDICO:
   - Recomendações alto risco têm mitigação?
   - Bloqueios legais não endereçados?
   Se sim → DESTACAR RISCO CRÍTICO

4. REALISMO GERAL:
   - Aprovado por conselho experiente?
   - Premissas defensáveis?
   Se dúvida → SER CONSERVADOR

REGRAS DE FORMATAÇÃO (PDF):
1. Central Challenge: desafio do cliente
2. Executive Summary: max 400 chars, parágrafo de impacto
3. Main Findings: 4 insights do SWOT
4. Important Notes: 2-3 observações (max 120 chars cada)
5. Key Findings: 3 insights (max 150 chars cada)
6. Strategic Priorities: 3 prioridades (max 100 chars cada)
7. Roadmap: 3 passos (Curto, Médio, Longo Prazo)

NARRATIVA ESTRATÉGICA (4 Perguntas):
- Parte I - Onde Estamos?: PESTEL + Porter + SWOT (max 200 chars)
- Parte II - Onde Queremos Ir?: TAM + Benchmarking + Blue Ocean (max 200 chars)
- Parte III - Como Chegar Lá?: OKRs + Growth + BSC (max 200 chars)
- Parte IV - O Que Fazer Agora?: Cenários + Decision Matrix (max 200 chars)

OUTPUT JSON:
{
  "central_challenge": "Desafio do cliente",
  "executive_summary": "Texto de impacto (max 400 chars)",
  "main_findings": [
    "Força principal (SWOT)",
    "Fraqueza principal (SWOT)",
    "Oportunidade principal (SWOT)",
    "Ameaça principal (SWOT)"
  ],
  "important_notes": [
    "Observação 1 (max 120 chars)",
    "Observação 2 (max 120 chars)",
    "Observação 3 (opcional)"
  ],
  "key_findings": ["Insight 1 (max 150)", "Insight 2", "Insight 3"],
  "strategic_priorities": ["Prioridade 1 (max 100)", "Prioridade 2", "Prioridade 3"],
  "roadmap": [
    "Curto Prazo (3-6 meses): ...",
    "Médio Prazo (6-12 meses): ...",
    "Longo Prazo (12+ meses): ..."
  ],
  "overall_recommendation": "Conclusão final acionável",
  "strategic_narrative": {
    "parte_1_onde_estamos": "Síntese situação (max 200 chars)",
    "parte_2_onde_queremos_ir": "Síntese posicionamento (max 200 chars)",
    "parte_3_como_chegar_la": "Síntese execução (max 200 chars)",
    "parte_4_o_que_fazer_agora": "Síntese ações (max 200 chars)"
  },
  "consistency_validation": {
    "financial_alignment": true,
    "capacity_alignment": true,
    "legal_alignment": true,
    "overall_realism_score": 8,
    "flags": ["Flag inconsistência (se houver)"]
  }
}$usr$
WHERE code = 'synthesis';

-- =============================================================================
-- VERIFICATION
-- =============================================================================
DO $$
DECLARE
    empty_prompts INTEGER;
    placeholder_prompts INTEGER;
BEGIN
    SELECT COUNT(*) INTO empty_prompts
    FROM frameworks
    WHERE prompt_user IS NULL OR LENGTH(prompt_user) < 100;

    SELECT COUNT(*) INTO placeholder_prompts
    FROM frameworks
    WHERE prompt_user LIKE '%[Full prompt in llm/prompts.go%';

    IF empty_prompts > 0 THEN
        RAISE WARNING '⚠️ Found % frameworks with empty/short prompts', empty_prompts;
    END IF;

    IF placeholder_prompts > 0 THEN
        RAISE WARNING '⚠️ Found % frameworks still with placeholder text', placeholder_prompts;
    ELSE
        RAISE NOTICE '✅ All 12 frameworks have real prompts populated';
    END IF;
END $$;
