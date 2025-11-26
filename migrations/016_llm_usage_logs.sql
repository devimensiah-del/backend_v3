-- Migration: Add LLM usage tracking table
-- This table logs all LLM API calls for cost monitoring and analysis

CREATE TABLE IF NOT EXISTS llm_usage_logs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id uuid REFERENCES submissions(id) ON DELETE SET NULL,
    framework_name text NOT NULL,              -- e.g., 'pestel', 'porter', 'enrichment', 'synthesis'
    model_used text NOT NULL,                  -- e.g., 'google/gemini-2.5-flash'
    input_tokens int NOT NULL DEFAULT 0,
    output_tokens int NOT NULL DEFAULT 0,
    total_tokens int NOT NULL DEFAULT 0,
    cost_usd decimal(12,8) NOT NULL DEFAULT 0, -- 8 decimal places for micro-pricing
    latency_ms int,                            -- Response time in milliseconds
    is_fallback boolean NOT NULL DEFAULT false, -- True if fallback model was used
    error_message text,                        -- If call failed
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Index for querying by submission
CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_submission_id ON llm_usage_logs(submission_id);

-- Index for time-based queries (cost reports)
CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_created_at ON llm_usage_logs(created_at);

-- Index for model analysis
CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_model ON llm_usage_logs(model_used);

-- Index for framework analysis
CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_framework ON llm_usage_logs(framework_name);

COMMENT ON TABLE llm_usage_logs IS 'Tracks all LLM API calls for cost monitoring, usage analysis, and debugging';
