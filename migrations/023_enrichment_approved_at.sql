-- Migration: Add approved_at column to enrichments table
-- This column tracks when an enrichment was approved by admin

ALTER TABLE enrichments
ADD COLUMN IF NOT EXISTS approved_at TIMESTAMP WITH TIME ZONE;

-- Create index for approved_at queries
CREATE INDEX IF NOT EXISTS idx_enrichments_approved_at ON enrichments(approved_at) WHERE approved_at IS NOT NULL;
