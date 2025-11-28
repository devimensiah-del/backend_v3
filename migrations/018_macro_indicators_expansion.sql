-- Migration: 018_macro_indicators_expansion.sql
-- Description: Expand macroeconomics indicators from 3 to 14
-- Date: 2025-11-27

-- ============================================
-- ADD NEW INDICATOR TYPES (11 new)
-- ============================================
INSERT INTO macro_indicator_types (code, name, category, unit, frequency) VALUES
-- Inflation indicators
('inpc', 'INPC', 'inflation', '%', 'monthly'),
('ipca15', 'IPCA-15', 'inflation', '%', 'monthly'),
('ipp', 'IPP - Preços ao Produtor', 'inflation', '%', 'monthly'),
-- Construction
('construction_cost', 'Custo do m²', 'construction', '%', 'monthly'),
-- GDP
('gdp_growth', 'PIB - Variação Trimestral', 'gdp', '%', 'quarterly'),
('gdp_12mo', 'PIB - Acumulado 12 meses', 'gdp', '%', 'quarterly'),
('gdp_per_capita', 'PIB per capita', 'gdp', 'BRL', 'annual'),
-- Production
('industry', 'Produção Industrial', 'production', '%', 'monthly'),
('commerce', 'Volume de Vendas - Comércio', 'production', '%', 'monthly'),
('services', 'Volume de Serviços', 'production', '%', 'monthly'),
-- Employment
('unemployment', 'Taxa de Desemprego', 'employment', '%', 'quarterly')
ON CONFLICT (code) DO NOTHING;

-- ============================================
-- ADD MANUAL SEED DATA SOURCE
-- ============================================
INSERT INTO macro_data_sources (code, name, base_url, is_authoritative) VALUES
('manual_seed', 'Manual Data Entry', NULL, false)
ON CONFLICT (code) DO NOTHING;

-- ============================================
-- SEED VALUES FOR ALL INDICATORS
-- ============================================

-- IPCA (Oct 2025) - manual seed since IBGE DNS fails
INSERT INTO macro_indicator_values (indicator_type_id, source_id, value, effective_date, reference_period, fetched_at)
SELECT t.id, s.id, 0.09, '2025-10-15', '2025-10', NOW()
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'ipca' AND s.code = 'manual_seed'
ON CONFLICT DO NOTHING;

-- INPC (Oct 2025)
INSERT INTO macro_indicator_values (indicator_type_id, source_id, value, effective_date, reference_period, fetched_at)
SELECT t.id, s.id, 0.03, '2025-10-15', '2025-10', NOW()
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'inpc' AND s.code = 'manual_seed'
ON CONFLICT DO NOTHING;

-- IPCA-15 (Nov 2025)
INSERT INTO macro_indicator_values (indicator_type_id, source_id, value, effective_date, reference_period, fetched_at)
SELECT t.id, s.id, 0.20, '2025-11-15', '2025-11', NOW()
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'ipca15' AND s.code = 'manual_seed'
ON CONFLICT DO NOTHING;

-- IPP (Sep 2025)
INSERT INTO macro_indicator_values (indicator_type_id, source_id, value, effective_date, reference_period, fetched_at)
SELECT t.id, s.id, -0.25, '2025-09-15', '2025-09', NOW()
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'ipp' AND s.code = 'manual_seed'
ON CONFLICT DO NOTHING;

-- Construction Cost (Oct 2025)
INSERT INTO macro_indicator_values (indicator_type_id, source_id, value, effective_date, reference_period, fetched_at)
SELECT t.id, s.id, 0.27, '2025-10-15', '2025-10', NOW()
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'construction_cost' AND s.code = 'manual_seed'
ON CONFLICT DO NOTHING;

-- GDP Growth (Q2 2025)
INSERT INTO macro_indicator_values (indicator_type_id, source_id, value, effective_date, reference_period, fetched_at)
SELECT t.id, s.id, 2.2, '2025-06-30', '2025-Q2', NOW()
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'gdp_growth' AND s.code = 'manual_seed'
ON CONFLICT DO NOTHING;

-- GDP 12mo (Q2 2025)
INSERT INTO macro_indicator_values (indicator_type_id, source_id, value, effective_date, reference_period, fetched_at)
SELECT t.id, s.id, 3.2, '2025-06-30', '2025-Q2', NOW()
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'gdp_12mo' AND s.code = 'manual_seed'
ON CONFLICT DO NOTHING;

-- GDP per capita (2022 - latest available)
INSERT INTO macro_indicator_values (indicator_type_id, source_id, value, effective_date, reference_period, fetched_at)
SELECT t.id, s.id, 47802.02, '2022-12-31', '2022', NOW()
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'gdp_per_capita' AND s.code = 'manual_seed'
ON CONFLICT DO NOTHING;

-- Industry (Sep 2025)
INSERT INTO macro_indicator_values (indicator_type_id, source_id, value, effective_date, reference_period, fetched_at)
SELECT t.id, s.id, -0.4, '2025-09-30', '2025-09', NOW()
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'industry' AND s.code = 'manual_seed'
ON CONFLICT DO NOTHING;

-- Commerce (Sep 2025)
INSERT INTO macro_indicator_values (indicator_type_id, source_id, value, effective_date, reference_period, fetched_at)
SELECT t.id, s.id, -0.3, '2025-09-30', '2025-09', NOW()
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'commerce' AND s.code = 'manual_seed'
ON CONFLICT DO NOTHING;

-- Services (Sep 2025)
INSERT INTO macro_indicator_values (indicator_type_id, source_id, value, effective_date, reference_period, fetched_at)
SELECT t.id, s.id, 0.6, '2025-09-30', '2025-09', NOW()
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'services' AND s.code = 'manual_seed'
ON CONFLICT DO NOTHING;

-- Unemployment (Q3 2025)
INSERT INTO macro_indicator_values (indicator_type_id, source_id, value, effective_date, reference_period, fetched_at)
SELECT t.id, s.id, 5.6, '2025-09-30', '2025-Q3', NOW()
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'unemployment' AND s.code = 'manual_seed'
ON CONFLICT DO NOTHING;
