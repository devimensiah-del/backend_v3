-- Migration: v2_041_step_organization.sql
-- Description: Reorganize frameworks into the new 5-step structure
-- Steps: 1-Empresa, 2-Negócio, 3-Ambiente, 4-Diagnóstico, 5-Desafios

-- =============================================================================
-- 1. Add step organization columns to frameworks table
-- =============================================================================
ALTER TABLE frameworks ADD COLUMN IF NOT EXISTS step_number INT;
ALTER TABLE frameworks ADD COLUMN IF NOT EXISTS step_name TEXT;
ALTER TABLE frameworks ADD COLUMN IF NOT EXISTS subtab_of TEXT;
ALTER TABLE frameworks ADD COLUMN IF NOT EXISTS execution_order INT DEFAULT 1;

-- Add prompt for update mode (re-execution with context injection)
ALTER TABLE frameworks ADD COLUMN IF NOT EXISTS prompt_update TEXT;

COMMENT ON COLUMN frameworks.step_number IS 'Main step number (1-5): 1=Empresa, 2=Negócio, 3=Ambiente, 4=Diagnóstico, 5=Desafios';
COMMENT ON COLUMN frameworks.step_name IS 'Display name for the step tab';
COMMENT ON COLUMN frameworks.subtab_of IS 'If this framework is a subtab, the parent step name (e.g., ambiente, diagnostico)';
COMMENT ON COLUMN frameworks.execution_order IS 'Execution order within the step (for subtabs)';
COMMENT ON COLUMN frameworks.prompt_update IS 'Prompt used when re-running after upstream changes (includes current result injection)';

-- =============================================================================
-- 2. Insert new frameworks: EMPRESA, NEGOCIO, MERCADO, SWOT_CROSS, DESAFIOS
-- =============================================================================

-- EMPRESA (Step 1) - Company data display (no AI, just enriched data)
INSERT INTO frameworks (code, name, description, layer, is_base, step_number, step_name, subtab_of, execution_order, prompt_user, model_config)
VALUES (
    'empresa',
    'Empresa',
    'Visão geral da empresa: dados cadastrais, histórico, estrutura societária',
    0,  -- Layer 0 = data display, no AI
    true,
    1,
    'Empresa',
    NULL,
    1,
    'Dados da empresa carregados automaticamente do enrichment.',
    '{"model": "none", "temperature": 0, "max_tokens": 0, "fallback_model": "none"}'
)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    step_number = EXCLUDED.step_number,
    step_name = EXCLUDED.step_name,
    subtab_of = EXCLUDED.subtab_of,
    execution_order = EXCLUDED.execution_order,
    updated_at = NOW();

-- NEGOCIO (Step 2) - Business model display
INSERT INTO frameworks (code, name, description, layer, is_base, step_number, step_name, subtab_of, execution_order, prompt_user, model_config)
VALUES (
    'negocio',
    'Negócio',
    'Modelo de negócio: produtos/serviços, proposta de valor, clientes',
    0,
    true,
    2,
    'Negócio',
    NULL,
    1,
    'Dados do negócio carregados automaticamente do enrichment.',
    '{"model": "none", "temperature": 0, "max_tokens": 0, "fallback_model": "none"}'
)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    step_number = EXCLUDED.step_number,
    step_name = EXCLUDED.step_name,
    subtab_of = EXCLUDED.subtab_of,
    execution_order = EXCLUDED.execution_order,
    updated_at = NOW();

-- MERCADO (Step 3, subtab 1 of Ambiente)
INSERT INTO frameworks (code, name, description, layer, is_base, step_number, step_name, subtab_of, execution_order, prompt_user, prompt_update, model_config)
VALUES (
    'mercado',
    'Mercado',
    'Análise de mercado: tamanho, segmentos, tendências, competidores',
    1,
    true,
    3,
    'Ambiente',
    'ambiente',
    1,
    'Analise o mercado de atuação da empresa considerando:
- Tamanho e potencial do mercado
- Segmentos de clientes principais
- Tendências de mercado
- Principais competidores
- Barreiras de entrada

## CONTEXTO DA EMPRESA
{{.Company}}

## NEGÓCIO
{{.NEGOCIO}}

Retorne uma análise estruturada em JSON.',
    'Atualize a análise de mercado considerando as mudanças no contexto.

## CONTEXTO ATUALIZADO
{{.Company}}

## NEGÓCIO ATUALIZADO
{{.NEGOCIO}}

## ANÁLISE DE MERCADO ATUAL
{{.CurrentResult}}

Identifique o que mudou e atualize apenas os pontos necessários.
Mantenha insights válidos da análise anterior.
Retorne em JSON com campo "changes_summary" explicando as alterações.',
    '{"model": "google/gemini-2.5-flash", "temperature": 0.5, "max_tokens": 8000, "fallback_model": "openai/gpt-4.1-mini"}'
)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    step_number = EXCLUDED.step_number,
    step_name = EXCLUDED.step_name,
    subtab_of = EXCLUDED.subtab_of,
    execution_order = EXCLUDED.execution_order,
    prompt_user = EXCLUDED.prompt_user,
    prompt_update = EXCLUDED.prompt_update,
    updated_at = NOW();

-- Update PESTEL to be Step 3, subtab 2 of Ambiente
UPDATE frameworks SET
    step_number = 3,
    step_name = 'Ambiente',
    subtab_of = 'ambiente',
    execution_order = 2,
    prompt_update = 'Atualize a análise PESTEL considerando as mudanças no contexto upstream.

## CONTEXTO DA EMPRESA
{{.Company}}

## NEGÓCIO
{{.NEGOCIO}}

## MERCADO ATUALIZADO
{{.MERCADO}}

## ANÁLISE PESTEL ATUAL
{{.CurrentResult}}

INSTRUÇÕES:
1. Compare o contexto atual com a análise existente
2. Identifique quais fatores (P, E, S, T, E, L) precisam ser atualizados
3. MANTENHA os insights que ainda são válidos
4. ATUALIZE apenas o que mudou devido às alterações upstream
5. Inclua "changes_summary" explicando o que foi alterado e por quê

Retorne a análise completa em JSON.'
WHERE code = 'pestel';

-- Update PORTER to be Step 3, subtab 3 of Ambiente
UPDATE frameworks SET
    step_number = 3,
    step_name = 'Ambiente',
    subtab_of = 'ambiente',
    execution_order = 3,
    prompt_update = 'Atualize a análise das 7 Forças de Porter considerando as mudanças no contexto.

## CONTEXTO DA EMPRESA
{{.Company}}

## NEGÓCIO
{{.NEGOCIO}}

## MERCADO
{{.MERCADO}}

## PESTEL
{{.PESTEL}}

## ANÁLISE PORTER ATUAL
{{.CurrentResult}}

INSTRUÇÕES:
1. Compare o contexto atual com a análise existente
2. Identifique quais forças precisam ser atualizadas
3. MANTENHA os insights que ainda são válidos
4. ATUALIZE apenas o que mudou
5. Inclua "changes_summary" explicando as alterações

Retorne a análise completa em JSON.'
WHERE code = 'porter';

-- Update SWOT to be Step 4, subtab 1 of Diagnóstico
UPDATE frameworks SET
    step_number = 4,
    step_name = 'Diagnóstico',
    subtab_of = 'diagnostico',
    execution_order = 1,
    prompt_update = 'Atualize a análise SWOT considerando as mudanças nas análises anteriores.

## CONTEXTO DA EMPRESA
{{.Company}}

## NEGÓCIO
{{.NEGOCIO}}

## MERCADO
{{.MERCADO}}

## PESTEL
{{.PESTEL}}

## PORTER
{{.PORTER}}

## ANÁLISE SWOT ATUAL
{{.CurrentResult}}

INSTRUÇÕES:
1. Revise cada quadrante (Strengths, Weaknesses, Opportunities, Threats)
2. Atualize com base nas mudanças de MERCADO, PESTEL e PORTER
3. MANTENHA itens ainda válidos
4. ATUALIZE itens afetados pelas mudanças
5. Inclua "changes_summary" com as alterações

Retorne a análise completa em JSON.'
WHERE code = 'swot';

-- SWOT_CROSS (Step 4, subtab 2 of Diagnóstico) - NEW FRAMEWORK
INSERT INTO frameworks (code, name, description, layer, is_base, step_number, step_name, subtab_of, execution_order, prompt_user, prompt_update, model_config)
VALUES (
    'swot_cross',
    'SWOT Cross',
    'Estratégias cruzadas: SO (Strengths-Opportunities), WO (Weaknesses-Opportunities), ST (Strengths-Threats), WT (Weaknesses-Threats)',
    2,
    true,
    4,
    'Diagnóstico',
    'diagnostico',
    2,
    'Realize uma análise SWOT Cross (Matriz TOWS) baseada na análise SWOT.

## CONTEXTO DA EMPRESA
{{.Company}}

## ANÁLISE SWOT
{{.SWOT}}

## INSTRUÇÕES
Crie estratégias cruzadas para cada quadrante:

1. **SO (Maxi-Maxi)**: Como usar FORÇAS para aproveitar OPORTUNIDADES?
2. **WO (Mini-Maxi)**: Como superar FRAQUEZAS aproveitando OPORTUNIDADES?
3. **ST (Maxi-Mini)**: Como usar FORÇAS para minimizar AMEAÇAS?
4. **WT (Mini-Mini)**: Como minimizar FRAQUEZAS e evitar AMEAÇAS?

Para cada estratégia, inclua:
- Descrição clara da estratégia
- Elementos SWOT relacionados (quais S/W/O/T está cruzando)
- Prioridade (alta, média, baixa)
- Prazo estimado (curto, médio, longo prazo)
- Recursos necessários

Retorne em JSON estruturado.',
    'Atualize as estratégias SWOT Cross baseado nas mudanças da análise SWOT.

## CONTEXTO DA EMPRESA
{{.Company}}

## ANÁLISE SWOT ATUALIZADA
{{.SWOT}}

## SWOT CROSS ATUAL
{{.CurrentResult}}

INSTRUÇÕES:
1. Revise cada quadrante de estratégias (SO, WO, ST, WT)
2. Atualize estratégias afetadas pelas mudanças no SWOT
3. MANTENHA estratégias ainda válidas
4. REMOVA ou AJUSTE estratégias obsoletas
5. Inclua "changes_summary" com as alterações

Retorne a análise completa em JSON.',
    '{"model": "google/gemini-2.5-flash", "temperature": 0.5, "max_tokens": 8000, "fallback_model": "openai/gpt-4.1-mini"}'
)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    step_number = EXCLUDED.step_number,
    step_name = EXCLUDED.step_name,
    subtab_of = EXCLUDED.subtab_of,
    execution_order = EXCLUDED.execution_order,
    prompt_user = EXCLUDED.prompt_user,
    prompt_update = EXCLUDED.prompt_update,
    model_config = EXCLUDED.model_config,
    updated_at = NOW();

-- DESAFIOS (Step 5) - AI-identified strategic challenges
INSERT INTO frameworks (code, name, description, layer, is_base, step_number, step_name, subtab_of, execution_order, prompt_user, prompt_update, model_config)
VALUES (
    'desafios',
    'Desafios Estratégicos',
    'Identificação de desafios estratégicos prioritários baseados em toda a análise',
    3,
    true,
    5,
    'Desafios',
    NULL,
    1,
    'Com base em toda a análise realizada, identifique os principais desafios estratégicos da empresa.

## CONTEXTO DA EMPRESA
{{.Company}}

## NEGÓCIO
{{.NEGOCIO}}

## MERCADO
{{.MERCADO}}

## PESTEL
{{.PESTEL}}

## PORTER
{{.PORTER}}

## SWOT
{{.SWOT}}

## SWOT CROSS
{{.SWOT_CROSS}}

## INSTRUÇÕES
Identifique 3-5 desafios estratégicos prioritários que a empresa deve enfrentar.

Para cada desafio, inclua:
1. **titulo**: Nome curto e descritivo do desafio
2. **descricao**: Descrição detalhada do desafio (2-3 parágrafos)
3. **categoria**: Categoria do desafio (crescimento, operacional, mercado, financeiro, inovação, pessoas)
4. **urgencia**: Nível de urgência (alta, média, baixa)
5. **impacto_potencial**: Impacto se não for resolvido (alto, médio, baixo)
6. **frameworks_relacionados**: Quais análises (PESTEL, PORTER, SWOT, etc.) evidenciam este desafio
7. **evidencias**: Lista de evidências específicas das análises que justificam este desafio
8. **acoes_iniciais**: 2-3 ações iniciais sugeridas para começar a endereçar o desafio

Retorne em JSON com array "desafios" ordenado por prioridade (urgência + impacto).',
    'Atualize os desafios estratégicos com base nas mudanças nas análises anteriores.

## CONTEXTO ATUALIZADO
{{.Company}}
{{.NEGOCIO}}
{{.MERCADO}}
{{.PESTEL}}
{{.PORTER}}
{{.SWOT}}
{{.SWOT_CROSS}}

## DESAFIOS ATUAIS
{{.CurrentResult}}

INSTRUÇÕES:
1. Revise cada desafio identificado anteriormente
2. Verifique se ainda são válidos com o novo contexto
3. Ajuste prioridades se necessário
4. Adicione novos desafios que emergiram das mudanças
5. Remova desafios que não fazem mais sentido
6. Inclua "changes_summary" explicando as alterações

Retorne em JSON.',
    '{"model": "google/gemini-2.5-flash", "temperature": 0.5, "max_tokens": 10000, "fallback_model": "openai/gpt-4.1-mini"}'
)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    step_number = EXCLUDED.step_number,
    step_name = EXCLUDED.step_name,
    subtab_of = EXCLUDED.subtab_of,
    execution_order = EXCLUDED.execution_order,
    prompt_user = EXCLUDED.prompt_user,
    prompt_update = EXCLUDED.prompt_update,
    model_config = EXCLUDED.model_config,
    updated_at = NOW();

-- =============================================================================
-- 3. Update existing frameworks with step organization (deactivate unused ones)
-- =============================================================================

-- Deactivate frameworks not in the new 5-step flow
UPDATE frameworks SET is_active = false WHERE code IN (
    'tam_sam_som', 'benchmarking', 'blue_ocean', 'growth_hacking',
    'scenarios', 'decision_matrix', 'okrs', 'bsc', 'synthesis'
);

-- =============================================================================
-- 4. Add new dependencies for the 5-step flow
-- =============================================================================

-- NEGOCIO depends on EMPRESA
INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_neg.id, f_emp.id, 'required'
FROM frameworks f_neg, frameworks f_emp
WHERE f_neg.code = 'negocio' AND f_emp.code = 'empresa'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

-- MERCADO depends on EMPRESA and NEGOCIO
INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_merc.id, f_emp.id, 'required'
FROM frameworks f_merc, frameworks f_emp
WHERE f_merc.code = 'mercado' AND f_emp.code = 'empresa'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_merc.id, f_neg.id, 'required'
FROM frameworks f_merc, frameworks f_neg
WHERE f_merc.code = 'mercado' AND f_neg.code = 'negocio'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

-- PESTEL depends on MERCADO (and transitively on EMPRESA, NEGOCIO)
INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_pest.id, f_merc.id, 'required'
FROM frameworks f_pest, frameworks f_merc
WHERE f_pest.code = 'pestel' AND f_merc.code = 'mercado'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

-- PORTER depends on PESTEL
INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_port.id, f_pest.id, 'required'
FROM frameworks f_port, frameworks f_pest
WHERE f_port.code = 'porter' AND f_pest.code = 'pestel'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

-- SWOT depends on PORTER (and transitively on all above)
-- Note: existing dependency on pestel and porter already exists

-- SWOT_CROSS depends on SWOT
INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_cross.id, f_swot.id, 'required'
FROM frameworks f_cross, frameworks f_swot
WHERE f_cross.code = 'swot_cross' AND f_swot.code = 'swot'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

-- DESAFIOS depends on SWOT_CROSS
INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_des.id, f_cross.id, 'required'
FROM frameworks f_des, frameworks f_cross
WHERE f_des.code = 'desafios' AND f_cross.code = 'swot_cross'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

-- =============================================================================
-- 5. Create index for step-based queries
-- =============================================================================
CREATE INDEX IF NOT EXISTS idx_frameworks_step ON frameworks(step_number, execution_order) WHERE is_active = true;

-- =============================================================================
-- 6. Verify migration
-- =============================================================================
DO $$
DECLARE
    active_count INTEGER;
    step_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO active_count FROM frameworks WHERE is_active = true;
    SELECT COUNT(DISTINCT step_number) INTO step_count FROM frameworks WHERE is_active = true AND step_number IS NOT NULL;
    RAISE NOTICE 'Migration complete: % active frameworks across % steps', active_count, step_count;
END $$;
