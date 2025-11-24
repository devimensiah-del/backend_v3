-- Migration 018: Fix enrichment status constraint
-- Issue: Code uses 'completed' but database constraint expects 'finished'
-- This causes constraint violations when worker tries to complete enrichments

-- 1. Update any existing 'finished' statuses to 'completed' (if any exist)
UPDATE public.enrichments
SET status = 'completed'
WHERE status = 'finished';

-- 2. Drop the old constraint that expects 'finished'
ALTER TABLE public.enrichments
DROP CONSTRAINT IF EXISTS enrichments_status_check;

-- 3. Add the correct constraint that matches the code
ALTER TABLE public.enrichments
ADD CONSTRAINT enrichments_status_check
CHECK (status IN ('pending', 'completed', 'approved'));

-- 4. Create an index on status for better query performance
CREATE INDEX IF NOT EXISTS idx_enrichments_status ON public.enrichments(status) WHERE status != 'approved';
