-- =============================================================================
-- Migration: v2_001_frameworks_table
-- Purpose: Create frameworks table for dynamic framework configuration
-- Date: 2024-12-02
-- Replaces: 032_create_frameworks_table.sql, 033_seed_frameworks.sql
-- =============================================================================

-- Create frameworks table
CREATE TABLE IF NOT EXISTS frameworks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    name_pt VARCHAR(100) NOT NULL,
    description TEXT,
    description_pt TEXT,
    category VARCHAR(50) NOT NULL,
    layer_order INTEGER NOT NULL DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    requires_enrichment BOOLEAN DEFAULT true,
    timeout_seconds INTEGER DEFAULT 60,
    prompt_template TEXT NOT NULL,
    output_schema JSONB NOT NULL,
    preferred_model VARCHAR(50),
    temperature DECIMAL(3,2) DEFAULT 0.7,
    depends_on TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_frameworks_code ON frameworks(code);
CREATE INDEX IF NOT EXISTS idx_frameworks_category ON frameworks(category);
CREATE INDEX IF NOT EXISTS idx_frameworks_active ON frameworks(is_active) WHERE is_active = true;

COMMENT ON TABLE frameworks IS 'Dynamic framework configuration for analysis execution';
COMMENT ON COLUMN frameworks.code IS 'Unique identifier used in code (e.g., pestel, swot)';
COMMENT ON COLUMN frameworks.category IS 'One of: environment, positioning, strategy, execution';
COMMENT ON COLUMN frameworks.layer_order IS 'Execution order within category (1-based)';
COMMENT ON COLUMN frameworks.prompt_template IS 'LLM prompt with {{variable}} placeholders';
COMMENT ON COLUMN frameworks.output_schema IS 'JSON Schema for structured LLM output';
COMMENT ON COLUMN frameworks.depends_on IS 'Array of framework codes that must complete first';

-- =============================================================================
-- SEED DATA: 11 Strategic Analysis Frameworks
-- =============================================================================

-- Layer 1: Environment Analysis
INSERT INTO frameworks (code, name, name_pt, description, description_pt, category, layer_order, prompt_template, output_schema, depends_on) VALUES
('pestel', 'PESTEL Analysis', 'Análise PESTEL',
 'Analyze Political, Economic, Social, Technological, Environmental, and Legal factors',
 'Analisa fatores Políticos, Econômicos, Sociais, Tecnológicos, Ambientais e Legais',
 'environment', 1,
 'See llm/prompts.go:FrameworkPESTELPrompt',
 '{"type": "object", "required": ["political", "economic", "social", "technological", "environmental", "legal", "summary"]}'::jsonb,
 '{}'),

('porter', 'Porter''s 7 Forces', 'Forças de Porter (7)',
 'Modern Porter''s Forces analysis including AI/Data disruption and Partnerships/Ecosystems',
 'Análise das Forças de Porter modernizada incluindo disrupção por IA/Dados e Parcerias/Ecossistemas',
 'environment', 2,
 'See llm/prompts.go:FrameworkPorterPrompt',
 '{"type": "object", "required": ["competitive_rivalry", "supplier_power", "buyer_power", "threat_new_entrants", "threat_substitutes", "power_partnerships_ecosystems", "disruption_ai_data", "summary"]}'::jsonb,
 '{}'),

('tam_sam_som', 'TAM-SAM-SOM Analysis', 'Análise TAM-SAM-SOM',
 'Total Addressable Market, Serviceable Available Market, and Serviceable Obtainable Market sizing',
 'Dimensionamento de Mercado Total, Disponível e Obtível',
 'environment', 3,
 'See llm/prompts.go:FrameworkTamSamSomPrompt',
 '{"type": "object", "required": ["tam", "sam", "som", "assumptions", "summary"]}'::jsonb,
 '{}');

-- Layer 2: Positioning Analysis
INSERT INTO frameworks (code, name, name_pt, description, description_pt, category, layer_order, prompt_template, output_schema, depends_on) VALUES
('swot', 'SWOT Analysis', 'Análise SWOT',
 'Strengths, Weaknesses, Opportunities, and Threats with confidence levels',
 'Forças, Fraquezas, Oportunidades e Ameaças com níveis de confiança',
 'positioning', 1,
 'See llm/prompts.go:FrameworkSWOTPrompt',
 '{"type": "object", "required": ["strengths", "weaknesses", "opportunities", "threats", "summary"]}'::jsonb,
 ARRAY['pestel', 'porter']),

('benchmarking', 'Competitive Benchmarking', 'Benchmarking Competitivo',
 'Competitive analysis identifying performance gaps and best practices',
 'Análise competitiva identificando gaps de performance e melhores práticas',
 'positioning', 2,
 'See llm/prompts.go:FrameworkBenchmarkingPrompt',
 '{"type": "object", "required": ["competitors_analyzed", "performance_gaps", "best_practices", "summary"]}'::jsonb,
 '{}');

-- Layer 3: Strategy Development
INSERT INTO frameworks (code, name, name_pt, description, description_pt, category, layer_order, prompt_template, output_schema, depends_on) VALUES
('blue_ocean', 'Blue Ocean Strategy', 'Estratégia Oceano Azul',
 'Blue Ocean Strategy using ERRC framework (Eliminate-Reduce-Raise-Create)',
 'Estratégia Oceano Azul usando framework ERRC (Eliminar-Reduzir-Elevar-Criar)',
 'strategy', 1,
 'See llm/prompts.go:FrameworkBlueOceanPrompt',
 '{"type": "object", "required": ["eliminate", "reduce", "raise", "create", "new_value_curve", "summary"]}'::jsonb,
 ARRAY['porter']),

('growth_hacking', 'Growth Hacking Loops', 'Loops de Growth Hacking',
 'Growth hacking strategies using LEAP (Acquisition) and SCALE (Monetization) loops',
 'Estratégias de growth hacking usando loops LEAP (Aquisição) e SCALE (Monetização)',
 'strategy', 2,
 'See llm/prompts.go:FrameworkGrowthHackingPrompt',
 '{"type": "object", "required": ["leap_loop", "scale_loop", "summary"]}'::jsonb,
 ARRAY['swot', 'tam_sam_som']),

('scenarios', 'Scenario Planning', 'Planejamento de Cenários',
 'Future scenario analysis with optimistic, realistic, and pessimistic projections',
 'Análise de cenários futuros com projeções otimista, realista e pessimista',
 'strategy', 3,
 'See llm/prompts.go:FrameworkScenariosPrompt',
 '{"type": "object", "required": ["optimistic", "realist", "pessimistic", "mitigation_tactics", "summary"]}'::jsonb,
 ARRAY['pestel']);

-- Layer 4: Execution Planning
INSERT INTO frameworks (code, name, name_pt, description, description_pt, category, layer_order, prompt_template, output_schema, depends_on) VALUES
('okrs', 'OKRs 90-Day Plan', 'OKRs Plano 90 Dias',
 'Objectives and Key Results with 90-day execution plan',
 'Objetivos e Resultados-Chave com plano de execução de 90 dias',
 'execution', 1,
 'See llm/prompts.go:FrameworkOKRsPrompt',
 '{"type": "object", "required": ["plan_90_days", "total_investment", "success_metrics", "summary"]}'::jsonb,
 ARRAY['blue_ocean', 'decision_matrix']),

('bsc', 'Balanced Scorecard', 'Balanced Scorecard',
 'Balanced Scorecard with Financial, Customer, Internal Processes, and Learning perspectives',
 'Balanced Scorecard com perspectivas Financeira, Cliente, Processos Internos e Aprendizado',
 'execution', 2,
 'See llm/prompts.go:FrameworkBSCPrompt',
 '{"type": "object", "required": ["financial", "customer", "internal_processes", "learning_growth", "summary"]}'::jsonb,
 ARRAY['blue_ocean']),

('decision_matrix', 'Decision Matrix', 'Matriz de Decisão',
 'Multi-criteria decision matrix with prioritized recommendations',
 'Matriz de decisão multicriterial com recomendações priorizadas',
 'execution', 3,
 'See llm/prompts.go:FrameworkDecisionMatrixPrompt',
 '{"type": "object", "required": ["alternatives", "criteria", "recommended_option", "priority_recommendations", "summary"]}'::jsonb,
 ARRAY['scenarios']);
