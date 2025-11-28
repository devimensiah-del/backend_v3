-- Migration 017: Macroeconomics Domain
-- Creates tables for storing persistent economic indicators (SELIC, IPCA, USD/BRL)
-- Updated via scheduled cron jobs for enrichment consumption

-- ============================================================================
-- Table 1: Indicator Types (Registry)
-- ============================================================================
CREATE TABLE IF NOT EXISTS macro_indicator_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL UNIQUE,           -- 'selic', 'ipca', 'usd_brl'
    name VARCHAR(255) NOT NULL,                 -- 'Taxa SELIC'
    category VARCHAR(50) NOT NULL,              -- 'interest_rate', 'inflation', 'exchange_rate'
    unit VARCHAR(20),                           -- '%', 'BRL'
    frequency VARCHAR(20) NOT NULL,             -- 'daily', 'monthly', 'quarterly'
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_indicator_types_code ON macro_indicator_types(code);
CREATE INDEX IF NOT EXISTS idx_indicator_types_active ON macro_indicator_types(is_active) WHERE is_active = true;

-- ============================================================================
-- Table 2: Data Sources (Registry)
-- ============================================================================
CREATE TABLE IF NOT EXISTS macro_data_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL UNIQUE,           -- 'bcb_api', 'ibge_sidra', 'awesomeapi'
    name VARCHAR(255) NOT NULL,
    base_url TEXT,
    priority SMALLINT DEFAULT 1,                -- Lower = higher priority (fallback order)
    is_authoritative BOOLEAN DEFAULT false,     -- Official government source
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_data_sources_code ON macro_data_sources(code);

-- ============================================================================
-- Table 3: Indicator-Source Mapping (Many-to-Many with Config)
-- ============================================================================
CREATE TABLE IF NOT EXISTS macro_indicator_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    indicator_type_id UUID NOT NULL REFERENCES macro_indicator_types(id) ON DELETE CASCADE,
    source_id UUID NOT NULL REFERENCES macro_data_sources(id) ON DELETE CASCADE,
    priority SMALLINT DEFAULT 1,                -- Per-indicator priority (fallback order)
    endpoint_config JSONB NOT NULL DEFAULT '{}', -- API endpoint details (series codes, paths)
    is_active BOOLEAN DEFAULT true,
    last_success_at TIMESTAMPTZ,
    consecutive_failures SMALLINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(indicator_type_id, source_id)
);

CREATE INDEX IF NOT EXISTS idx_indicator_sources_lookup
    ON macro_indicator_sources(indicator_type_id, priority)
    WHERE is_active = true;

-- ============================================================================
-- Table 4: Indicator Values (Time-Series Data)
-- ============================================================================
CREATE TABLE IF NOT EXISTS macro_indicator_values (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    indicator_type_id UUID NOT NULL REFERENCES macro_indicator_types(id) ON DELETE CASCADE,
    source_id UUID NOT NULL REFERENCES macro_data_sources(id) ON DELETE CASCADE,
    value NUMERIC NOT NULL,
    effective_date DATE NOT NULL,               -- Date the value is effective for
    reference_period VARCHAR(20),               -- '2024-11' for monthly, NULL for daily
    raw_response JSONB,                         -- Full API response for debugging
    fetched_at TIMESTAMPTZ DEFAULT now(),
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Fast lookup for latest value per indicator
CREATE INDEX IF NOT EXISTS idx_values_latest
    ON macro_indicator_values(indicator_type_id, effective_date DESC);

-- Fast lookup for specific indicator by code (via join)
CREATE INDEX IF NOT EXISTS idx_values_type_date
    ON macro_indicator_values(indicator_type_id, effective_date);

-- Prevent duplicate entries for same indicator/source/date/period
CREATE UNIQUE INDEX IF NOT EXISTS idx_values_unique
    ON macro_indicator_values(indicator_type_id, source_id, effective_date, COALESCE(reference_period, ''));

-- ============================================================================
-- Table 5: Fetch Logs (Audit Trail)
-- ============================================================================
CREATE TABLE IF NOT EXISTS macro_fetch_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    indicator_source_id UUID REFERENCES macro_indicator_sources(id) ON DELETE SET NULL,
    indicator_code VARCHAR(50),                 -- Denormalized for queries after source deletion
    source_code VARCHAR(50),                    -- Denormalized for queries after source deletion
    status VARCHAR(20) NOT NULL,                -- 'success', 'failed', 'partial'
    records_inserted INTEGER DEFAULT 0,
    records_updated INTEGER DEFAULT 0,
    error_message TEXT,
    response_time_ms INTEGER,
    triggered_by VARCHAR(20) DEFAULT 'scheduler', -- 'scheduler', 'manual', 'startup'
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_fetch_logs_recent
    ON macro_fetch_logs(indicator_source_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_fetch_logs_status
    ON macro_fetch_logs(status, created_at DESC);

-- ============================================================================
-- Seed Data: Indicator Types
-- ============================================================================
INSERT INTO macro_indicator_types (code, name, category, unit, frequency, metadata) VALUES
('selic', 'Taxa SELIC', 'interest_rate', '%', 'daily',
 '{"description": "Taxa básica de juros da economia brasileira", "bcb_series": "4189"}'),
('ipca', 'IPCA - Inflação', 'inflation', '%', 'monthly',
 '{"description": "Índice de Preços ao Consumidor Amplo", "ibge_table": "1737"}'),
('usd_brl', 'Dólar (USD/BRL)', 'exchange_rate', 'BRL', 'daily',
 '{"description": "Cotação do dólar americano em reais", "bcb_series": "1"}')
ON CONFLICT (code) DO NOTHING;

-- ============================================================================
-- Seed Data: Data Sources
-- ============================================================================
INSERT INTO macro_data_sources (code, name, base_url, priority, is_authoritative) VALUES
('bcb_api', 'Banco Central do Brasil', 'https://api.bcb.gov.br', 1, true),
('ibge_sidra', 'IBGE SIDRA', 'https://api.ibge.gov.br', 1, true),
('awesomeapi', 'AwesomeAPI', 'https://economia.awesomeapi.com.br', 2, false),
('exchangerate_api', 'ExchangeRate-API', 'https://api.exchangerate-api.com', 3, false)
ON CONFLICT (code) DO NOTHING;

-- ============================================================================
-- Seed Data: Indicator-Source Mappings
-- ============================================================================

-- SELIC: BCB API (primary, authoritative)
INSERT INTO macro_indicator_sources (indicator_type_id, source_id, priority, endpoint_config)
SELECT t.id, s.id, 1, '{"series": "4189", "format": "json"}'::jsonb
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'selic' AND s.code = 'bcb_api'
ON CONFLICT (indicator_type_id, source_id) DO NOTHING;

-- IPCA: IBGE SIDRA (primary, authoritative)
INSERT INTO macro_indicator_sources (indicator_type_id, source_id, priority, endpoint_config)
SELECT t.id, s.id, 1, '{"table": "1737", "variable": "63", "format": "json"}'::jsonb
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'ipca' AND s.code = 'ibge_sidra'
ON CONFLICT (indicator_type_id, source_id) DO NOTHING;

-- USD/BRL: BCB API (primary, authoritative)
INSERT INTO macro_indicator_sources (indicator_type_id, source_id, priority, endpoint_config)
SELECT t.id, s.id, 1, '{"series": "1", "format": "json"}'::jsonb
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'usd_brl' AND s.code = 'bcb_api'
ON CONFLICT (indicator_type_id, source_id) DO NOTHING;

-- USD/BRL: AwesomeAPI (fallback)
INSERT INTO macro_indicator_sources (indicator_type_id, source_id, priority, endpoint_config)
SELECT t.id, s.id, 2, '{"endpoint": "/last/USD-BRL", "format": "json"}'::jsonb
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'usd_brl' AND s.code = 'awesomeapi'
ON CONFLICT (indicator_type_id, source_id) DO NOTHING;

-- USD/BRL: ExchangeRate-API (tertiary fallback)
INSERT INTO macro_indicator_sources (indicator_type_id, source_id, priority, endpoint_config)
SELECT t.id, s.id, 3, '{"base": "USD", "target": "BRL"}'::jsonb
FROM macro_indicator_types t, macro_data_sources s
WHERE t.code = 'usd_brl' AND s.code = 'exchangerate_api'
ON CONFLICT (indicator_type_id, source_id) DO NOTHING;

-- ============================================================================
-- Helper function: Get latest value for an indicator
-- ============================================================================
CREATE OR REPLACE FUNCTION get_latest_indicator_value(p_code VARCHAR(50))
RETURNS TABLE (
    indicator_code VARCHAR(50),
    value NUMERIC,
    effective_date DATE,
    reference_period VARCHAR(20),
    fetched_at TIMESTAMPTZ,
    source_code VARCHAR(50)
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        t.code AS indicator_code,
        v.value,
        v.effective_date,
        v.reference_period,
        v.fetched_at,
        s.code AS source_code
    FROM macro_indicator_values v
    JOIN macro_indicator_types t ON v.indicator_type_id = t.id
    JOIN macro_data_sources s ON v.source_id = s.id
    WHERE t.code = p_code
    ORDER BY v.effective_date DESC, v.fetched_at DESC
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Comments
-- ============================================================================
COMMENT ON TABLE macro_indicator_types IS 'Registry of economic indicator types (SELIC, IPCA, USD/BRL)';
COMMENT ON TABLE macro_data_sources IS 'Registry of data source APIs (BCB, IBGE, AwesomeAPI)';
COMMENT ON TABLE macro_indicator_sources IS 'Mapping of indicators to sources with fallback priorities';
COMMENT ON TABLE macro_indicator_values IS 'Time-series storage of indicator values';
COMMENT ON TABLE macro_fetch_logs IS 'Audit trail of fetch operations for debugging and monitoring';
