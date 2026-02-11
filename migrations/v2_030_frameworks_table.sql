-- Migration: v2_030_frameworks_table.sql
-- Description: Create frameworks table for database-driven framework configuration
-- This replaces hardcoded constants in domain/analysis/constants.go

-- Frameworks table - defines all available strategic frameworks
CREATE TABLE IF NOT EXISTS frameworks (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,

    -- Identity
    code VARCHAR(50) NOT NULL UNIQUE,         -- e.g., "pestel", "porter", "swot"
    name VARCHAR(100) NOT NULL,               -- e.g., "PESTEL Analysis"
    description TEXT,                         -- Human-readable description

    -- Execution configuration
    layer INTEGER NOT NULL,                   -- Execution order (1-6, where 6 is synthesis)
    is_base BOOLEAN NOT NULL DEFAULT false,   -- Required for all analyses (cannot be skipped)
    is_active BOOLEAN NOT NULL DEFAULT true,  -- Can be disabled without deletion

    -- Prompt configuration
    prompt_system TEXT,                       -- System prompt (optional override)
    prompt_user TEXT NOT NULL,                -- Main prompt template (required)
    prompt_json_template TEXT,                -- Expected JSON schema documentation

    -- Model configuration (JSONB for flexibility)
    model_config JSONB NOT NULL DEFAULT '{
        "model": "google/gemini-2.5-flash",
        "temperature": 0.5,
        "max_tokens": 8000,
        "fallback_model": "openai/gpt-4.1-mini"
    }',

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_frameworks_layer ON frameworks(layer);
CREATE INDEX IF NOT EXISTS idx_frameworks_is_active ON frameworks(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_frameworks_code ON frameworks(code);

-- Add comment for documentation
COMMENT ON TABLE frameworks IS 'Database-driven framework configuration for strategic analysis. Replaces hardcoded constants.go and prompts.go.';
COMMENT ON COLUMN frameworks.code IS 'Unique string identifier (e.g., pestel, porter, swot). Used for URLs and code references.';
COMMENT ON COLUMN frameworks.layer IS 'Execution layer: 1=Environment, 2=Positioning, 3=Strategy, 4=Decision, 5=Execution, 6=Synthesis';
COMMENT ON COLUMN frameworks.is_base IS 'Base frameworks are required for every analysis and cannot be skipped.';
