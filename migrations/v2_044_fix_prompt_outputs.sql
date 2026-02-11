-- Migration: v2_044_fix_prompt_outputs.sql
-- Description: Fix Porter and SWOT Cross prompts to output JSON structures matching frontend editors
-- Problem: Porter prompt outputs flat structure, frontend expects { forces: [] } array
--          SWOT Cross prompt says "Retorne em JSON estruturado" without specifying structure

-- =============================================================================
-- 1. Fix PORTER prompt output structure
-- =============================================================================
-- Current: flat structure with competitive_rivalry, supplier_power_intensity, etc.
-- Required: { forces: [{ force, intensity, description }], overallAttractiveness, summary }

UPDATE frameworks SET
  prompt_user = $usr$Analise as 7 Forças de Porter (2025+).

DESAFIO: {{.challenge_context}}
TIPO: {{.challenge_type}} ({{.challenge_category}})

CONTEXTO: {{COMPANY_DATA}}
MACRO: {{MACRO_CONTEXT}}

AS 7 FORÇAS:
1. Rivalidade Competitiva
2. Poder dos Fornecedores
3. Poder dos Compradores
4. Ameaça de Novos Entrantes
5. Ameaça de Substitutos
6. Parcerias/Ecossistemas (NOVA)
7. Disrupção IA/Dados (NOVA)

REGRAS:
- Max 250 chars por força
- Intensity: "Muito Baixa" | "Baixa" | "Moderada" | "Alta" | "Muito Alta"
- Descreva IMPACTO direto com dados reais
- Use informacoes do MACRO_CONTEXT quando relevante

OUTPUT JSON (EXATAMENTE este formato):
{
  "forces": [
    { "force": "Rivalidade Competitiva", "intensity": "Alta", "description": "Analise rivalidade (max 250 chars)" },
    { "force": "Poder dos Fornecedores", "intensity": "Media", "description": "Analise fornecedores (max 250 chars)" },
    { "force": "Poder dos Compradores", "intensity": "Alta", "description": "Analise compradores (max 250 chars)" },
    { "force": "Ameaca de Novos Entrantes", "intensity": "Baixa", "description": "Analise barreiras de entrada (max 250 chars)" },
    { "force": "Ameaca de Substitutos", "intensity": "Moderada", "description": "Analise substitutos (max 250 chars)" },
    { "force": "Parcerias/Ecossistemas", "intensity": "Moderada", "description": "Analise parcerias estrategicas (max 250 chars)" },
    { "force": "Disrupcao IA/Dados", "intensity": "Alta", "description": "Analise impacto de IA e dados (max 250 chars)" }
  ],
  "overallAttractiveness": "Alta",
  "summary": "Resumo executivo da analise de Porter (max 300 chars)"
}$usr$
WHERE code = 'porter';

-- =============================================================================
-- 2. Fix SWOT_CROSS prompt output structure
-- =============================================================================
-- Current: "Retorne em JSON estruturado" without specifying structure
-- Required: { so_strategies: [], wo_strategies: [], st_strategies: [], wt_strategies: [], summary }

UPDATE frameworks SET
  prompt_user = $usr$Realize uma analise SWOT Cross (Matriz TOWS) baseada na analise SWOT.

## CONTEXTO DA EMPRESA
{{.Company}}

## ANALISE SWOT
{{.SWOT}}

## INSTRUCOES
Crie estrategias cruzadas para cada quadrante:

1. **SO (Maxi-Maxi)**: Como usar FORCAS para aproveitar OPORTUNIDADES?
2. **WO (Mini-Maxi)**: Como superar FRAQUEZAS aproveitando OPORTUNIDADES?
3. **ST (Maxi-Mini)**: Como usar FORCAS para minimizar AMEACAS?
4. **WT (Mini-Mini)**: Como minimizar FRAQUEZAS e evitar AMEACAS?

REGRAS:
- 2-3 estrategias por quadrante
- Cada estrategia precisa de: descricao, elementos_swot, prioridade, prazo, recursos
- prioridade: "alta" | "media" | "baixa"
- prazo: "curto" | "medio" | "longo"

OUTPUT JSON (EXATAMENTE este formato):
{
  "so_strategies": [
    {
      "descricao": "Estrategia SO clara (max 200 chars)",
      "elementos_swot": ["Forca X", "Oportunidade Y"],
      "prioridade": "alta",
      "prazo": "curto",
      "recursos": "Recursos necessarios"
    }
  ],
  "wo_strategies": [
    {
      "descricao": "Estrategia WO clara (max 200 chars)",
      "elementos_swot": ["Fraqueza A", "Oportunidade B"],
      "prioridade": "media",
      "prazo": "medio",
      "recursos": "Recursos necessarios"
    }
  ],
  "st_strategies": [
    {
      "descricao": "Estrategia ST clara (max 200 chars)",
      "elementos_swot": ["Forca C", "Ameaca D"],
      "prioridade": "alta",
      "prazo": "curto",
      "recursos": "Recursos necessarios"
    }
  ],
  "wt_strategies": [
    {
      "descricao": "Estrategia WT clara (max 200 chars)",
      "elementos_swot": ["Fraqueza E", "Ameaca F"],
      "prioridade": "baixa",
      "prazo": "longo",
      "recursos": "Recursos necessarios"
    }
  ],
  "summary": "Resumo geral das estrategias cruzadas (max 300 chars)"
}$usr$,
  prompt_update = $usr$Atualize as estrategias SWOT Cross baseado nas mudancas da analise SWOT.

## CONTEXTO DA EMPRESA
{{.Company}}

## ANALISE SWOT ATUALIZADA
{{.SWOT}}

## SWOT CROSS ATUAL
{{.CurrentResult}}

INSTRUCOES:
1. Revise cada quadrante de estrategias (SO, WO, ST, WT)
2. Atualize estrategias afetadas pelas mudancas no SWOT
3. MANTENHA estrategias ainda validas
4. REMOVA ou AJUSTE estrategias obsoletas

OUTPUT JSON (mesmo formato, adicione changes_summary):
{
  "so_strategies": [...],
  "wo_strategies": [...],
  "st_strategies": [...],
  "wt_strategies": [...],
  "summary": "Resumo atualizado",
  "changes_summary": "Descricao das alteracoes feitas"
}$usr$
WHERE code = 'swot_cross';

-- Verify updates
DO $$
DECLARE
  porter_updated BOOLEAN;
  swot_cross_updated BOOLEAN;
BEGIN
  SELECT EXISTS(SELECT 1 FROM frameworks WHERE code = 'porter' AND prompt_user LIKE '%"forces":%') INTO porter_updated;
  SELECT EXISTS(SELECT 1 FROM frameworks WHERE code = 'swot_cross' AND prompt_user LIKE '%so_strategies%') INTO swot_cross_updated;

  IF porter_updated AND swot_cross_updated THEN
    RAISE NOTICE 'v2_044: Porter and SWOT Cross prompt outputs fixed successfully';
  ELSE
    RAISE EXCEPTION 'v2_044: Migration verification failed - porter: %, swot_cross: %', porter_updated, swot_cross_updated;
  END IF;
END $$;
