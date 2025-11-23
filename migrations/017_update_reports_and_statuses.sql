-- 1. Update Reports Table
-- Add new columns for the 24-page report structure
ALTER TABLE reports 
ADD COLUMN IF NOT EXISTS divider_part1_page text,
ADD COLUMN IF NOT EXISTS pestel_pes_page text,
ADD COLUMN IF NOT EXISTS pestel_tel_page text,
ADD COLUMN IF NOT EXISTS divider_part2_page text,
ADD COLUMN IF NOT EXISTS divider_part3_page text,
ADD COLUMN IF NOT EXISTS growth_loops_page text,
ADD COLUMN IF NOT EXISTS divider_part4_page text,
ADD COLUMN IF NOT EXISTS recommendations_page text,
ADD COLUMN IF NOT EXISTS roadmap_page text;

-- Update default total_pages
ALTER TABLE reports ALTER COLUMN total_pages SET DEFAULT 24;

-- 2. Update Enrichments Table
-- Standardize status 'finished' -> 'completed'
UPDATE enrichments SET status = 'completed' WHERE status = 'finished';

-- Update constraint to remove 'processing', 'failed' and use 'completed'
ALTER TABLE enrichments DROP CONSTRAINT IF EXISTS enrichments_status_check;
ALTER TABLE enrichments ADD CONSTRAINT enrichments_status_check 
    CHECK (status::text = ANY (ARRAY['pending'::character varying, 'completed'::character varying, 'approved'::character varying]::text[]));

-- 3. Update Analyses Table
-- Update constraint to remove 'processing', 'failed'
ALTER TABLE analyses DROP CONSTRAINT IF EXISTS analyses_status_check;
ALTER TABLE analyses ADD CONSTRAINT analyses_status_check 
    CHECK (status::text = ANY (ARRAY['pending'::character varying, 'completed'::character varying, 'approved'::character varying, 'sent'::character varying]::text[]));
