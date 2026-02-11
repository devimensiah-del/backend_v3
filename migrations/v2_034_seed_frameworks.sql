-- Migration: v2_034_seed_frameworks.sql
-- Description: Seed the 12 existing frameworks with their configurations
-- Note: prompt_user values are simplified placeholders - full prompts loaded from llm/prompts.go in code

-- Insert all 12 frameworks with layer assignments matching current workflow.go
INSERT INTO frameworks (code, name, description, layer, is_base, prompt_user, model_config) VALUES

-- Layer 1: Environment (parallel execution)
('pestel', 'PESTEL Analysis', 'Macro-environmental analysis: Political, Economic, Social, Technological, Environmental, Legal factors', 1, true,
 'Realize uma análise PESTEL (Modelo SCAN) priorizada para relatório executivo. [Full prompt in llm/prompts.go:FrameworkPESTELPrompt]',
 '{"model": "google/gemini-2.5-flash", "temperature": 0.5, "max_tokens": 8000, "fallback_model": "openai/gpt-4.1-mini"}'),

('porter', 'Porter 7 Forces', 'Industry structure analysis: 5 traditional forces + partnerships + AI disruption', 1, true,
 'Analise as 7 Forças de Porter (2025+) usando o framework RACE. [Full prompt in llm/prompts.go:FrameworkPorterPrompt]',
 '{"model": "google/gemini-2.5-flash", "temperature": 0.5, "max_tokens": 8000, "fallback_model": "openai/gpt-4.1-mini"}'),

('tam_sam_som', 'TAM-SAM-SOM', 'Market sizing with 3-tier scenario modeling', 1, true,
 'Dimensione o mercado usando ESTIMATIVA OBRIGATÓRIA com o modelo RACE/AIM. [Full prompt in llm/prompts.go:FrameworkTamSamSomPrompt]',
 '{"model": "google/gemini-2.5-flash", "temperature": 0.5, "max_tokens": 8000, "fallback_model": "openai/gpt-4.1-mini"}'),

-- Layer 2: Positioning (depends on Layer 1)
('swot', 'SWOT Analysis', 'Internal strengths/weaknesses + external opportunities/threats with confidence levels', 2, true,
 'Realize uma análise SWOT (Modelo LIFT) focada em itens acionáveis com níveis de confiança e fonte. [Full prompt in llm/prompts.go:FrameworkSWOTPrompt]',
 '{"model": "google/gemini-2.5-flash", "temperature": 0.5, "max_tokens": 8000, "fallback_model": "openai/gpt-4.1-mini"}'),

('benchmarking', 'Benchmarking', 'Competitive benchmarking against key competitors', 2, false,
 'Realize um Benchmarking Competitivo (Modelo COMPARE). [Full prompt in llm/prompts.go:FrameworkBenchmarkingPrompt]',
 '{"model": "google/gemini-2.5-flash", "temperature": 0.5, "max_tokens": 8000, "fallback_model": "openai/gpt-4.1-mini"}'),

-- Layer 3: Strategy (depends on Layer 2)
('blue_ocean', 'Blue Ocean Strategy', 'Value innovation via ERRC framework (Eliminate, Reduce, Raise, Create)', 3, false,
 'Desenvolva uma Estratégia do Oceano Azul (Modelo CREATE/ERRC). [Full prompt in llm/prompts.go:FrameworkBlueOceanPrompt]',
 '{"model": "google/gemini-2.5-flash", "temperature": 0.5, "max_tokens": 8000, "fallback_model": "openai/gpt-4.1-mini"}'),

('growth_hacking', 'Growth Hacking', 'LEAP Loop (acquisition) + SCALE Loop (monetization)', 3, false,
 'Crie estratégias de Growth Hacking com LEAP Loop (Aquisição) e SCALE Loop (Monetização). [Full prompt in llm/prompts.go:FrameworkGrowthHackingPrompt]',
 '{"model": "google/gemini-2.5-flash", "temperature": 0.5, "max_tokens": 8000, "fallback_model": "openai/gpt-4.1-mini"}'),

('scenarios', 'Scenario Analysis', 'Future scenarios with probabilities and mitigation tactics', 3, false,
 'Realize uma Análise de Cenários (Modelo FUTUREMAP) com probabilidades e táticas de mitigação. [Full prompt in llm/prompts.go:FrameworkScenariosPrompt]',
 '{"model": "google/gemini-2.5-flash", "temperature": 0.5, "max_tokens": 8000, "fallback_model": "openai/gpt-4.1-mini"}'),

-- Layer 4: Decision Making (depends on Layer 3)
('decision_matrix', 'Decision Matrix', 'Multi-criteria decision analysis with prioritized recommendations', 4, false,
 'Crie uma Matriz de Decisão Multicriterial (Modelo ANALYTICA) com recomendações priorizadas. [Full prompt in llm/prompts.go:FrameworkDecisionMatrixPrompt]',
 '{"model": "google/gemini-2.5-flash", "temperature": 0.5, "max_tokens": 8000, "fallback_model": "openai/gpt-4.1-mini"}'),

-- Layer 5: Execution (depends on Layer 4)
('okrs', 'OKRs 90-Day Plan', 'Objectives and Key Results with monthly milestones', 5, false,
 'Defina um Plano Estratégico de 90 Dias (Modelo FOCUS) com marcos mensais. [Full prompt in llm/prompts.go:FrameworkOKRsPrompt]',
 '{"model": "google/gemini-2.5-flash", "temperature": 0.5, "max_tokens": 8000, "fallback_model": "openai/gpt-4.1-mini"}'),

('bsc', 'Balanced Scorecard', 'Strategic alignment across financial, customer, process, and learning perspectives', 5, false,
 'Estruture um Balanced Scorecard (Modelo ALIGN). [Full prompt in llm/prompts.go:FrameworkBSCPrompt]',
 '{"model": "google/gemini-2.5-flash", "temperature": 0.5, "max_tokens": 8000, "fallback_model": "openai/gpt-4.1-mini"}'),

-- Layer 6: Synthesis (depends on all)
('synthesis', 'Executive Synthesis', 'Final executive summary combining all framework insights', 6, true,
 'Você é um Líder Exponencial. Sintetize os 11 frameworks em um Resumo Executivo estruturado. [Full prompt in llm/prompts.go:SynthesisPrompt]',
 '{"model": "google/gemini-2.5-pro-preview", "temperature": 0.5, "max_tokens": 6000, "fallback_model": "openai/gpt-4.1"}')

ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    layer = EXCLUDED.layer,
    is_base = EXCLUDED.is_base,
    prompt_user = EXCLUDED.prompt_user,
    model_config = EXCLUDED.model_config,
    updated_at = NOW();

-- =============================================================================
-- FRAMEWORK DEPENDENCIES
-- Only direct dependencies (transitive resolved at runtime via topological sort)
-- =============================================================================

-- Layer 2 depends on Layer 1 frameworks
-- SWOT depends on PESTEL and Porter for external context
INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_swot.id, f_pestel.id, 'required'
FROM frameworks f_swot, frameworks f_pestel
WHERE f_swot.code = 'swot' AND f_pestel.code = 'pestel'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_swot.id, f_porter.id, 'required'
FROM frameworks f_swot, frameworks f_porter
WHERE f_swot.code = 'swot' AND f_porter.code = 'porter'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

-- Benchmarking depends on TAM-SAM-SOM for market scale context
INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_bench.id, f_tam.id, 'required'
FROM frameworks f_bench, frameworks f_tam
WHERE f_bench.code = 'benchmarking' AND f_tam.code = 'tam_sam_som'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

-- Layer 3 depends on Layer 2
-- Blue Ocean depends on Porter for competitive context
INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_blue.id, f_porter.id, 'required'
FROM frameworks f_blue, frameworks f_porter
WHERE f_blue.code = 'blue_ocean' AND f_porter.code = 'porter'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

-- Growth Hacking depends on SWOT and TAM-SAM-SOM
INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_growth.id, f_swot.id, 'required'
FROM frameworks f_growth, frameworks f_swot
WHERE f_growth.code = 'growth_hacking' AND f_swot.code = 'swot'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_growth.id, f_tam.id, 'required'
FROM frameworks f_growth, frameworks f_tam
WHERE f_growth.code = 'growth_hacking' AND f_tam.code = 'tam_sam_som'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

-- Scenarios depends on PESTEL for macro context
INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_scen.id, f_pestel.id, 'required'
FROM frameworks f_scen, frameworks f_pestel
WHERE f_scen.code = 'scenarios' AND f_pestel.code = 'pestel'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

-- Layer 4: Decision Matrix depends on Scenarios
INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_dm.id, f_scen.id, 'required'
FROM frameworks f_dm, frameworks f_scen
WHERE f_dm.code = 'decision_matrix' AND f_scen.code = 'scenarios'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

-- Layer 5: OKRs depends on Decision Matrix and Blue Ocean
INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_okr.id, f_dm.id, 'required'
FROM frameworks f_okr, frameworks f_dm
WHERE f_okr.code = 'okrs' AND f_dm.code = 'decision_matrix'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_okr.id, f_blue.id, 'required'
FROM frameworks f_okr, frameworks f_blue
WHERE f_okr.code = 'okrs' AND f_blue.code = 'blue_ocean'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

-- BSC depends on Blue Ocean for strategy context
INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_bsc.id, f_blue.id, 'required'
FROM frameworks f_bsc, frameworks f_blue
WHERE f_bsc.code = 'bsc' AND f_blue.code = 'blue_ocean'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

-- Layer 6: Synthesis depends on all other frameworks
-- Note: This will be handled dynamically at runtime since synthesis needs ALL outputs
-- We don't need to store all 11 dependencies - the code knows synthesis runs last
INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_syn.id, f_dm.id, 'required'
FROM frameworks f_syn, frameworks f_dm
WHERE f_syn.code = 'synthesis' AND f_dm.code = 'decision_matrix'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_syn.id, f_okr.id, 'required'
FROM frameworks f_syn, frameworks f_okr
WHERE f_syn.code = 'synthesis' AND f_okr.code = 'okrs'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

INSERT INTO framework_dependencies (framework_id, depends_on_id, dependency_type)
SELECT f_syn.id, f_bsc.id, 'required'
FROM frameworks f_syn, frameworks f_bsc
WHERE f_syn.code = 'synthesis' AND f_bsc.code = 'bsc'
ON CONFLICT (framework_id, depends_on_id) DO NOTHING;

-- Verify seed completed
DO $$
DECLARE
    fw_count INTEGER;
    dep_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO fw_count FROM frameworks;
    SELECT COUNT(*) INTO dep_count FROM framework_dependencies;
    RAISE NOTICE 'Seeded % frameworks and % dependencies', fw_count, dep_count;
END $$;
