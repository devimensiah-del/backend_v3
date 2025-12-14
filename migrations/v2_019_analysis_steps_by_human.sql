-- v2_019_analysis_steps_by_human.sql
-- IAH-2: Step-by-step analysis with human editing

CREATE TABLE IF NOT EXISTS analysis_steps_v2 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analysis_id UUID NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,
    framework_code TEXT NOT NULL,
    step_number INTEGER NOT NULL,
    ai_output TEXT,
    human_edited TEXT,
    visible BOOLEAN NOT NULL DEFAULT true,
    status TEXT NOT NULL DEFAULT 'pending',
    generated_at TIMESTAMPTZ,
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(analysis_id, framework_code)
);

CREATE INDEX idx_analysis_steps_v2_analysis ON analysis_steps_v2(analysis_id);
CREATE INDEX idx_analysis_steps_v2_status ON analysis_steps_v2(status);

-- Rollback: DROP TABLE IF EXISTS analysis_steps_v2 CASCADE;
