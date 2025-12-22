-- Migration: v2_035_update_model_configs.sql
-- Description: Update all frameworks to use google/gemini-3-flash-preview with anthropic/claude-haiku-4.5 fallback

-- Update all frameworks except synthesis
UPDATE frameworks SET
    model_config = '{"model": "google/gemini-3-flash-preview", "temperature": 0.5, "max_tokens": 8000, "fallback_model": "anthropic/claude-haiku-4.5"}'::jsonb,
    updated_at = NOW()
WHERE code != 'synthesis';

-- Synthesis uses same model but with lower max_tokens
UPDATE frameworks SET
    model_config = '{"model": "google/gemini-3-flash-preview", "temperature": 0.5, "max_tokens": 6000, "fallback_model": "anthropic/claude-haiku-4.5"}'::jsonb,
    updated_at = NOW()
WHERE code = 'synthesis';

-- Verify update
DO $$
DECLARE
    updated_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO updated_count FROM frameworks
    WHERE model_config->>'model' = 'google/gemini-3-flash-preview';
    RAISE NOTICE 'Updated % frameworks to use gemini-3-flash-preview', updated_count;
END $$;
