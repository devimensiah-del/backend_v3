-- Migration 030: Field deprecation system for automatic un-verification
-- Purpose: Track deprecation categories for company fields so verified fields expire based on type
-- Categories: 'financial' (6mo), 'strategic' (1yr), 'core' (2yr), 'static' (never)

-- Add deprecation category to field verifications
-- Default to 'static' (never expires) for backwards compatibility
ALTER TABLE company_field_verifications
ADD COLUMN IF NOT EXISTS deprecation_category TEXT NOT NULL DEFAULT 'static';

COMMENT ON COLUMN company_field_verifications.deprecation_category IS 'Field category for deprecation: financial (6mo), strategic (1yr), core (2yr), static (never)';

-- Create deprecation config table that defines expiration periods per field
CREATE TABLE IF NOT EXISTS field_deprecation_config (
    field_name TEXT PRIMARY KEY,
    category TEXT NOT NULL,
    deprecation_months INT NOT NULL,
    description TEXT
);

-- Populate with all company fields and their deprecation rules
INSERT INTO field_deprecation_config (field_name, category, deprecation_months, description) VALUES
-- Financial fields (6 months) - Change frequently
('revenue_estimate', 'financial', 6, 'Receita estimada da empresa'),
('funding_stage', 'financial', 6, 'Estágio de financiamento'),
('employees_range', 'financial', 6, 'Faixa de funcionários'),
('annual_revenue_min', 'financial', 6, 'Receita anual mínima'),
('annual_revenue_max', 'financial', 6, 'Receita anual máxima'),

-- Strategic fields (12 months) - Change moderately
('competitors', 'strategic', 12, 'Lista de concorrentes'),
('market_share_status', 'strategic', 12, 'Status de participação de mercado'),
('digital_maturity', 'strategic', 12, 'Nível de maturidade digital'),
('strengths', 'strategic', 12, 'Pontos fortes identificados'),
('weaknesses', 'strategic', 12, 'Pontos fracos identificados'),
('value_proposition', 'strategic', 12, 'Proposta de valor'),

-- Core fields (24 months) - Change rarely
('industry', 'core', 24, 'Indústria/setor principal'),
('sector', 'core', 24, 'Setor de atuação'),
('business_model', 'core', 24, 'Modelo de negócio'),
('target_audience', 'core', 24, 'Público alvo'),
('target_market', 'core', 24, 'Mercado alvo'),

-- Static fields (never expire) - Rarely or never change
('name', 'static', 0, 'Nome da empresa'),
('cnpj', 'static', 0, 'CNPJ'),
('legal_name', 'static', 0, 'Razão social'),
('foundation_year', 'static', 0, 'Ano de fundação'),
('headquarters', 'static', 0, 'Sede/matriz'),
('website', 'static', 0, 'Website'),
('linkedin_url', 'static', 0, 'URL do LinkedIn'),
('twitter_handle', 'static', 0, 'Handle do Twitter'),
('location', 'static', 0, 'Localização'),
('company_size', 'static', 0, 'Porte da empresa')
ON CONFLICT (field_name) DO UPDATE SET
    category = EXCLUDED.category,
    deprecation_months = EXCLUDED.deprecation_months,
    description = EXCLUDED.description;

-- Create function to check and expire old field verifications
CREATE OR REPLACE FUNCTION expire_field_verifications(p_company_id UUID DEFAULT NULL)
RETURNS INTEGER AS $$
DECLARE
    expired_count INTEGER;
BEGIN
    WITH expired_fields AS (
        SELECT cfv.id
        FROM company_field_verifications cfv
        JOIN field_deprecation_config fdc ON cfv.field_name = fdc.field_name
        WHERE fdc.deprecation_months > 0  -- Skip static fields (0 months = never expires)
        AND cfv.verified_at < NOW() - (fdc.deprecation_months || ' months')::INTERVAL
        AND (p_company_id IS NULL OR cfv.company_id = p_company_id)
    )
    DELETE FROM company_field_verifications
    WHERE id IN (SELECT id FROM expired_fields);

    GET DIAGNOSTICS expired_count = ROW_COUNT;
    RETURN expired_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION expire_field_verifications IS 'Removes expired field verifications based on deprecation_months config. Pass company_id to check specific company, or NULL for all.';

-- Index for looking up deprecation config
CREATE INDEX IF NOT EXISTS idx_fdc_category ON field_deprecation_config(category);

COMMENT ON TABLE field_deprecation_config IS 'Configuration table defining deprecation periods for each company field type';
