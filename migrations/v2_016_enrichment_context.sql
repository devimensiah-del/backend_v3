-- Migration v2_016: Add enrichment context columns and simple macro_indicators table
-- This enables richer company enrichment data storage and a simple macro indicators system

-- Part 1: Simple macro_indicators table (manually updated via Supabase UI)
CREATE TABLE IF NOT EXISTS macro_indicators (
    code        TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    value       NUMERIC NOT NULL,
    unit        TEXT NOT NULL,
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_by  TEXT
);

-- Seed with current Brazilian macro data (December 2024)
INSERT INTO macro_indicators (code, name, value, unit, updated_by) VALUES
('selic', 'Taxa SELIC', 12.25, '% a.a.', 'manual'),
('ipca_12m', 'IPCA 12 meses', 4.83, '%', 'manual'),
('usd_brl', 'Dólar/Real', 6.05, 'R$', 'manual'),
('pib_growth', 'Crescimento PIB', 2.9, '%', 'manual'),
('unemployment', 'Taxa de Desemprego', 6.2, '%', 'manual')
ON CONFLICT (code) DO NOTHING;

-- Part 2: New enrichment context columns on companies table
ALTER TABLE companies ADD COLUMN IF NOT EXISTS industry_growth_rate TEXT;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS industry_trends JSONB DEFAULT '[]';
ALTER TABLE companies ADD COLUMN IF NOT EXISTS regulatory_context TEXT;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS market_concentration TEXT;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS enrichment_sources JSONB DEFAULT '[]';

-- Add comments for documentation
COMMENT ON TABLE macro_indicators IS 'Simple macro indicators table - manually updated via Supabase UI';
COMMENT ON COLUMN companies.industry_growth_rate IS 'Industry growth rate e.g. "+12% CAGR"';
COMMENT ON COLUMN companies.industry_trends IS 'Array of industry trends e.g. ["AI adoption", "Sustainability"]';
COMMENT ON COLUMN companies.regulatory_context IS 'Key regulatory information for the industry';
COMMENT ON COLUMN companies.market_concentration IS 'Market concentration e.g. "Fragmentado", "Concentrado"';
COMMENT ON COLUMN companies.enrichment_sources IS 'Array of URLs used as sources during enrichment';
