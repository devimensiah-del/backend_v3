-- Migration 032: Create frameworks table for dynamic framework configuration
-- Purpose: Enable database-driven framework execution instead of hardcoded structs
-- Date: 2025-12-02

CREATE TABLE frameworks (
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

CREATE INDEX idx_frameworks_code ON frameworks(code);
CREATE INDEX idx_frameworks_category ON frameworks(category);
CREATE INDEX idx_frameworks_active ON frameworks(is_active) WHERE is_active = true;

COMMENT ON TABLE frameworks IS 'Dynamic framework configuration for analysis execution';
COMMENT ON COLUMN frameworks.code IS 'Unique identifier used in code (e.g., pestel, swot)';
COMMENT ON COLUMN frameworks.category IS 'One of: environment, positioning, strategy, execution';
COMMENT ON COLUMN frameworks.layer_order IS 'Execution order within category (1-based)';
COMMENT ON COLUMN frameworks.prompt_template IS 'LLM prompt with {{variable}} placeholders';
COMMENT ON COLUMN frameworks.output_schema IS 'JSON Schema for structured LLM output';
COMMENT ON COLUMN frameworks.depends_on IS 'Array of framework codes that must complete first';
