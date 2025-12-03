-- Migration 033: Seed frameworks table with 11 strategic analysis frameworks
-- Purpose: Populate framework configuration for dynamic analysis execution
-- Date: 2025-12-02

-- Layer 1: Environment Analysis
INSERT INTO frameworks (code, name, name_pt, description, description_pt, category, layer_order, prompt_template, output_schema, depends_on) VALUES
('pestel', 'PESTEL Analysis', 'Análise PESTEL',
 'Analyze Political, Economic, Social, Technological, Environmental, and Legal factors affecting the business environment',
 'Analisa fatores Políticos, Econômicos, Sociais, Tecnológicos, Ambientais e Legais que afetam o ambiente de negócios',
 'environment', 1,
 'See llm/prompts.go:FrameworkPESTELPrompt for full template',
 '{
   "type": "object",
   "required": ["political", "economic", "social", "technological", "environmental", "legal", "summary"],
   "properties": {
     "political": {"type": "array", "items": {"type": "string"}, "minItems": 3, "maxItems": 3},
     "economic": {"type": "array", "items": {"type": "string"}, "minItems": 3, "maxItems": 3},
     "social": {"type": "array", "items": {"type": "string"}, "minItems": 3, "maxItems": 3},
     "technological": {"type": "array", "items": {"type": "string"}, "minItems": 3, "maxItems": 3},
     "environmental": {"type": "array", "items": {"type": "string"}, "minItems": 3, "maxItems": 3},
     "legal": {"type": "array", "items": {"type": "string"}, "minItems": 3, "maxItems": 3},
     "summary": {"type": "string", "maxLength": 250}
   }
 }'::jsonb,
 '{}'),

('porter', 'Porter''s 7 Forces', 'Forças de Porter (7)',
 'Modern Porter''s Forces analysis including AI/Data disruption and Partnerships/Ecosystems forces',
 'Análise das Forças de Porter modernizada incluindo disrupção por IA/Dados e força de Parcerias/Ecossistemas',
 'environment', 2,
 'See llm/prompts.go:FrameworkPorterPrompt for full template',
 '{
   "type": "object",
   "required": ["competitive_rivalry", "supplier_power", "buyer_power", "threat_new_entrants", "threat_substitutes", "power_partnerships_ecosystems", "disruption_ai_data", "competitive_rivalry_intensity", "supplier_power_intensity", "buyer_power_intensity", "threat_new_entrants_intensity", "threat_substitutes_intensity", "power_partnerships_ecosystems_intensity", "disruption_ai_data_intensity", "strategic_implications", "overall_attractiveness", "summary"],
   "properties": {
     "competitive_rivalry": {"type": "string", "maxLength": 250},
     "supplier_power": {"type": "string", "maxLength": 250},
     "buyer_power": {"type": "string", "maxLength": 250},
     "threat_new_entrants": {"type": "string", "maxLength": 250},
     "threat_substitutes": {"type": "string", "maxLength": 250},
     "power_partnerships_ecosystems": {"type": "string", "maxLength": 250},
     "disruption_ai_data": {"type": "string", "maxLength": 250},
     "competitive_rivalry_intensity": {"type": "string", "enum": ["Alta", "Média", "Baixa"]},
     "supplier_power_intensity": {"type": "string", "enum": ["Alta", "Média", "Baixa"]},
     "buyer_power_intensity": {"type": "string", "enum": ["Alta", "Média", "Baixa"]},
     "threat_new_entrants_intensity": {"type": "string", "enum": ["Alta", "Média", "Baixa"]},
     "threat_substitutes_intensity": {"type": "string", "enum": ["Alta", "Média", "Baixa"]},
     "power_partnerships_ecosystems_intensity": {"type": "string", "enum": ["Alta", "Média", "Baixa"]},
     "disruption_ai_data_intensity": {"type": "string", "enum": ["Alta", "Média", "Baixa"]},
     "strategic_implications": {"type": "array", "items": {"type": "string"}, "minItems": 4, "maxItems": 4},
     "overall_attractiveness": {"type": "string"},
     "summary": {"type": "string"}
   }
 }'::jsonb,
 '{}'),

('tam_sam_som', 'TAM-SAM-SOM Analysis', 'Análise TAM-SAM-SOM',
 'Total Addressable Market, Serviceable Available Market, and Serviceable Obtainable Market sizing with scenario modeling',
 'Dimensionamento de Mercado Total, Disponível e Obtível com modelagem de cenários',
 'environment', 3,
 'See llm/prompts.go:FrameworkTamSamSomPrompt for full template',
 '{
   "type": "object",
   "required": ["tam", "sam", "som", "assumptions", "cagr", "confidence_level", "estimation_method", "summary"],
   "properties": {
     "tam": {"type": "string"},
     "sam": {"type": "string"},
     "som": {"type": "string"},
     "assumptions": {"type": "array", "items": {"type": "string", "maxLength": 100}, "minItems": 3, "maxItems": 3},
     "cagr": {"type": "string"},
     "confidence_level": {"type": "integer", "minimum": 0, "maximum": 100},
     "estimation_method": {"type": "string"},
     "calculation_notes": {"type": "string", "maxLength": 200},
     "data_sources_used": {
       "type": "object",
       "properties": {
         "direct_data": {"type": "array", "items": {"type": "string"}},
         "proxy_calculations": {"type": "array", "items": {"type": "string"}},
         "benchmark_extrapolations": {"type": "array", "items": {"type": "string"}}
       }
     },
     "tam_scenarios": {
       "type": "object",
       "properties": {
         "conservative": {"type": "object", "properties": {"value_range": {"type": "string"}, "probability": {"type": "integer"}}},
         "realistic": {"type": "object", "properties": {"value_range": {"type": "string"}, "probability": {"type": "integer"}}},
         "optimistic": {"type": "object", "properties": {"value_range": {"type": "string"}, "probability": {"type": "integer"}}}
       }
     },
     "som_scenarios": {
       "type": "object",
       "properties": {
         "conservative": {"type": "object", "properties": {"value_range": {"type": "string"}, "market_share": {"type": "string"}, "probability": {"type": "integer"}}},
         "realistic": {"type": "object", "properties": {"value_range": {"type": "string"}, "market_share": {"type": "string"}, "probability": {"type": "integer"}}},
         "optimistic": {"type": "object", "properties": {"value_range": {"type": "string"}, "market_share": {"type": "string"}, "probability": {"type": "integer"}}}
       }
     },
     "baseline_recommendation": {"type": "string"},
     "caveat_message": {"type": "string"},
     "summary": {"type": "string", "maxLength": 200}
   }
 }'::jsonb,
 '{}');

-- Layer 2: Positioning Analysis
INSERT INTO frameworks (code, name, name_pt, description, description_pt, category, layer_order, prompt_template, output_schema, depends_on) VALUES
('swot', 'SWOT Analysis', 'Análise SWOT',
 'Strengths, Weaknesses, Opportunities, and Threats analysis with confidence levels and source attribution',
 'Análise de Forças, Fraquezas, Oportunidades e Ameaças com níveis de confiança e fonte de dados',
 'positioning', 1,
 'See llm/prompts.go:FrameworkSWOTPrompt for full template',
 '{
   "type": "object",
   "required": ["strengths", "weaknesses", "opportunities", "threats", "summary"],
   "properties": {
     "strengths": {
       "type": "array",
       "items": {
         "type": "object",
         "required": ["content", "confidence", "source"],
         "properties": {
           "content": {"type": "string", "maxLength": 100},
           "confidence": {"type": "string", "enum": ["Alta", "Média", "Baixa"]},
           "source": {"type": "string", "enum": ["fato", "análise de mercado", "estimativa", "feedback de clientes"]}
         }
       },
       "minItems": 4,
       "maxItems": 4
     },
     "weaknesses": {
       "type": "array",
       "items": {
         "type": "object",
         "required": ["content", "confidence", "source"],
         "properties": {
           "content": {"type": "string", "maxLength": 100},
           "confidence": {"type": "string", "enum": ["Alta", "Média", "Baixa"]},
           "source": {"type": "string", "enum": ["fato", "análise de mercado", "estimativa", "feedback de clientes"]}
         }
       },
       "minItems": 4,
       "maxItems": 4
     },
     "opportunities": {
       "type": "array",
       "items": {
         "type": "object",
         "required": ["content", "confidence", "source"],
         "properties": {
           "content": {"type": "string", "maxLength": 100},
           "confidence": {"type": "string", "enum": ["Alta", "Média", "Baixa"]},
           "source": {"type": "string", "enum": ["fato", "análise de mercado", "estimativa", "feedback de clientes"]}
         }
       },
       "minItems": 4,
       "maxItems": 4
     },
     "threats": {
       "type": "array",
       "items": {
         "type": "object",
         "required": ["content", "confidence", "source"],
         "properties": {
           "content": {"type": "string", "maxLength": 100},
           "confidence": {"type": "string", "enum": ["Alta", "Média", "Baixa"]},
           "source": {"type": "string", "enum": ["fato", "análise de mercado", "estimativa", "feedback de clientes"]}
         }
       },
       "minItems": 4,
       "maxItems": 4
     },
     "summary": {"type": "string", "maxLength": 200}
   }
 }'::jsonb,
 ARRAY['pestel', 'porter']),

('benchmarking', 'Competitive Benchmarking', 'Benchmarking Competitivo',
 'Competitive analysis identifying performance gaps and best practices',
 'Análise competitiva identificando gaps de performance e melhores práticas',
 'positioning', 2,
 'See llm/prompts.go:FrameworkBenchmarkingPrompt for full template',
 '{
   "type": "object",
   "required": ["competitors_analyzed", "performance_gaps", "best_practices", "summary"],
   "properties": {
     "competitors_analyzed": {"type": "array", "items": {"type": "string"}},
     "performance_gaps": {"type": "array", "items": {"type": "string", "maxLength": 120}, "minItems": 3, "maxItems": 3},
     "best_practices": {"type": "array", "items": {"type": "string", "maxLength": 120}, "minItems": 3, "maxItems": 3},
     "summary": {"type": "string"}
   }
 }'::jsonb,
 '{}');

-- Layer 3: Strategy Development
INSERT INTO frameworks (code, name, name_pt, description, description_pt, category, layer_order, prompt_template, output_schema, depends_on) VALUES
('blue_ocean', 'Blue Ocean Strategy', 'Estratégia Oceano Azul',
 'Blue Ocean Strategy using ERRC framework (Eliminate-Reduce-Raise-Create)',
 'Estratégia Oceano Azul usando framework ERRC (Eliminar-Reduzir-Elevar-Criar)',
 'strategy', 1,
 'See llm/prompts.go:FrameworkBlueOceanPrompt for full template',
 '{
   "type": "object",
   "required": ["eliminate", "reduce", "raise", "create", "new_value_curve", "summary"],
   "properties": {
     "eliminate": {"type": "array", "items": {"type": "string", "maxLength": 90}, "minItems": 3, "maxItems": 3},
     "reduce": {"type": "array", "items": {"type": "string", "maxLength": 90}, "minItems": 3, "maxItems": 3},
     "raise": {"type": "array", "items": {"type": "string", "maxLength": 90}, "minItems": 3, "maxItems": 3},
     "create": {"type": "array", "items": {"type": "string", "maxLength": 90}, "minItems": 3, "maxItems": 3},
     "new_value_curve": {"type": "string", "maxLength": 150},
     "summary": {"type": "string"}
   }
 }'::jsonb,
 ARRAY['porter']),

('growth_hacking', 'Growth Hacking Loops', 'Loops de Growth Hacking',
 'Growth hacking strategies using LEAP (Acquisition) and SCALE (Monetization) loops',
 'Estratégias de growth hacking usando loops LEAP (Aquisição) e SCALE (Monetização)',
 'strategy', 2,
 'See llm/prompts.go:FrameworkGrowthHackingPrompt for full template',
 '{
   "type": "object",
   "required": ["leap_loop", "scale_loop", "summary"],
   "properties": {
     "leap_loop": {
       "type": "object",
       "required": ["name", "type", "steps", "metrics", "bottleneck"],
       "properties": {
         "name": {"type": "string"},
         "type": {"type": "string", "enum": ["acquisition"]},
         "steps": {"type": "array", "items": {"type": "string", "maxLength": 120}, "minItems": 4, "maxItems": 4},
         "metrics": {"type": "array", "items": {"type": "string"}},
         "bottleneck": {"type": "string"}
       }
     },
     "scale_loop": {
       "type": "object",
       "required": ["name", "type", "steps", "metrics", "bottleneck"],
       "properties": {
         "name": {"type": "string"},
         "type": {"type": "string", "enum": ["monetization"]},
         "steps": {"type": "array", "items": {"type": "string", "maxLength": 120}, "minItems": 4, "maxItems": 4},
         "metrics": {"type": "array", "items": {"type": "string"}},
         "bottleneck": {"type": "string"}
       }
     },
     "summary": {"type": "string", "maxLength": 200}
   }
 }'::jsonb,
 ARRAY['swot', 'tam_sam_som']),

('scenarios', 'Scenario Planning', 'Planejamento de Cenários',
 'Future scenario analysis with optimistic, realistic, and pessimistic projections',
 'Análise de cenários futuros com projeções otimista, realista e pessimista',
 'strategy', 3,
 'See llm/prompts.go:FrameworkScenariosPrompt for full template',
 '{
   "type": "object",
   "required": ["optimistic", "realist", "pessimistic", "mitigation_tactics", "early_warning_signals", "summary"],
   "properties": {
     "optimistic": {
       "type": "object",
       "required": ["name", "probability", "description", "required_actions"],
       "properties": {
         "name": {"type": "string"},
         "probability": {"type": "integer"},
         "description": {"type": "string", "maxLength": 450},
         "required_actions": {"type": "array", "items": {"type": "string"}, "minItems": 3}
       }
     },
     "realist": {
       "type": "object",
       "required": ["name", "probability", "description", "required_actions"],
       "properties": {
         "name": {"type": "string"},
         "probability": {"type": "integer"},
         "description": {"type": "string", "maxLength": 450},
         "required_actions": {"type": "array", "items": {"type": "string"}, "minItems": 3}
       }
     },
     "pessimistic": {
       "type": "object",
       "required": ["name", "probability", "description", "required_actions"],
       "properties": {
         "name": {"type": "string"},
         "probability": {"type": "integer"},
         "description": {"type": "string", "maxLength": 450},
         "required_actions": {"type": "array", "items": {"type": "string"}, "minItems": 3}
       }
     },
     "mitigation_tactics": {"type": "array", "items": {"type": "string"}, "minItems": 4, "maxItems": 4},
     "early_warning_signals": {"type": "array", "items": {"type": "string"}, "minItems": 3},
     "summary": {"type": "string"}
   }
 }'::jsonb,
 ARRAY['pestel']);

-- Layer 4: Execution Planning
INSERT INTO frameworks (code, name, name_pt, description, description_pt, category, layer_order, prompt_template, output_schema, depends_on) VALUES
('okrs', 'OKRs 90-Day Plan', 'OKRs Plano 90 Dias',
 'Objectives and Key Results with 90-day execution plan and phased approach',
 'Objetivos e Resultados-Chave com plano de execução de 90 dias e abordagem em fases',
 'execution', 1,
 'See llm/prompts.go:FrameworkOKRsPrompt for full template',
 '{
   "type": "object",
   "required": ["plan_90_days", "total_investment", "success_metrics", "summary"],
   "properties": {
     "plan_90_days": {
       "type": "array",
       "items": {
         "type": "object",
         "required": ["month", "focus", "objective", "key_results", "investment", "aligned_recommendation"],
         "properties": {
           "month": {"type": "string"},
           "focus": {"type": "string"},
           "objective": {"type": "string"},
           "key_results": {"type": "array", "items": {"type": "string", "maxLength": 100}, "minItems": 3, "maxItems": 3},
           "investment": {"type": "string"},
           "aligned_recommendation": {"type": "string"}
         }
       },
       "minItems": 3,
       "maxItems": 3
     },
     "total_investment": {"type": "string"},
     "success_metrics": {"type": "array", "items": {"type": "string"}, "minItems": 3},
     "execution_phases": {
       "type": "object",
       "properties": {
         "foundation": {"type": "object"},
         "validation": {"type": "object"},
         "scale": {"type": "object"}
       }
     },
     "capacity_assessment": {
       "type": "object",
       "properties": {
         "team_readiness": {"type": "string"},
         "budget_adequacy": {"type": "string"},
         "stakeholder_alignment": {"type": "string"},
         "blockers": {"type": "array", "items": {"type": "string"}}
       }
     },
     "summary": {"type": "string", "maxLength": 200}
   }
 }'::jsonb,
 ARRAY['blue_ocean', 'decision_matrix']),

('bsc', 'Balanced Scorecard', 'Balanced Scorecard',
 'Balanced Scorecard with Financial, Customer, Internal Processes, and Learning perspectives',
 'Balanced Scorecard com perspectivas Financeira, Cliente, Processos Internos e Aprendizado',
 'execution', 2,
 'See llm/prompts.go:FrameworkBSCPrompt for full template',
 '{
   "type": "object",
   "required": ["financial", "customer", "internal_processes", "learning_growth", "summary"],
   "properties": {
     "financial": {"type": "array", "items": {"type": "string", "maxLength": 100}, "minItems": 2, "maxItems": 2},
     "customer": {"type": "array", "items": {"type": "string", "maxLength": 100}, "minItems": 2, "maxItems": 2},
     "internal_processes": {"type": "array", "items": {"type": "string", "maxLength": 100}, "minItems": 2, "maxItems": 2},
     "learning_growth": {"type": "array", "items": {"type": "string", "maxLength": 100}, "minItems": 2, "maxItems": 2},
     "summary": {"type": "string"}
   }
 }'::jsonb,
 ARRAY['blue_ocean']),

('decision_matrix', 'Decision Matrix', 'Matriz de Decisão',
 'Multi-criteria decision matrix with prioritized recommendations and legal feasibility assessment',
 'Matriz de decisão multicriterial com recomendações priorizadas e avaliação de viabilidade jurídica',
 'execution', 3,
 'See llm/prompts.go:FrameworkDecisionMatrixPrompt for full template',
 '{
   "type": "object",
   "required": ["alternatives", "criteria", "recommended_option", "score", "score_comparison", "priority_recommendations", "review_cycle", "monitoring_metrics", "final_recommendation", "summary"],
   "properties": {
     "alternatives": {"type": "array", "items": {"type": "string"}, "minItems": 3, "maxItems": 3},
     "criteria": {"type": "array", "items": {"type": "string"}, "minItems": 3, "maxItems": 3},
     "recommended_option": {"type": "string"},
     "score": {"type": "string"},
     "score_comparison": {"type": "string"},
     "priority_recommendations": {
       "type": "array",
       "items": {
         "type": "object",
         "required": ["priority", "title", "description", "timeline", "budget", "legal_feasibility"],
         "properties": {
           "priority": {"type": "integer"},
           "title": {"type": "string"},
           "description": {"type": "string"},
           "timeline": {"type": "string"},
           "budget": {"type": "string"},
           "legal_feasibility": {
             "type": "object",
             "required": ["risk_level", "requires_statutory_change", "requires_legal_opinion"],
             "properties": {
               "risk_level": {"type": "string", "enum": ["Baixo", "Médio", "Alto", "Crítico"]},
               "requires_statutory_change": {"type": "boolean"},
               "requires_legal_opinion": {"type": "boolean"},
               "regulatory_dependencies": {"type": "array", "items": {"type": "string"}},
               "mitigation_plan": {"type": "string"}
             }
           }
         }
       },
       "minItems": 3,
       "maxItems": 3
     },
     "review_cycle": {
       "type": "object",
       "required": ["frequency", "extraordinary_triggers"],
       "properties": {
         "frequency": {"type": "string"},
         "extraordinary_triggers": {"type": "array", "items": {"type": "string"}}
       }
     },
     "monitoring_metrics": {"type": "array", "items": {"type": "string"}, "minItems": 5},
     "final_recommendation": {"type": "string", "maxLength": 300},
     "summary": {"type": "string"}
   }
 }'::jsonb,
 ARRAY['scenarios']);

-- Note: Synthesis is handled separately as it aggregates all frameworks
-- It is not a framework in the frameworks table, but rather a final aggregation step
