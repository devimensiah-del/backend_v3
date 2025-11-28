-- Migration: Add auto_trigger_analysis flag to enrichments
-- Purpose: Support "enrich and analyze" workflow where analysis auto-triggers after enrichment approval

-- Add column to track if analysis should auto-trigger after this enrichment is approved
ALTER TABLE enrichments ADD COLUMN IF NOT EXISTS auto_trigger_analysis BOOLEAN NOT NULL DEFAULT false;

-- Add comment for documentation
COMMENT ON COLUMN enrichments.auto_trigger_analysis IS 'When true, analysis job is automatically enqueued when this enrichment is approved';
