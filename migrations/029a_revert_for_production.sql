-- Add 'approved' back to allowed statuses temporarily
  ALTER TABLE enrichments DROP CONSTRAINT IF EXISTS enrichments_status_check;
  ALTER TABLE enrichments ADD CONSTRAINT enrichments_status_check
      CHECK (status IN ('pending', 'completed', 'failed', 'approved'));

  -- Same for analyses
  ALTER TABLE analyses DROP CONSTRAINT IF EXISTS analyses_status_check;
  ALTER TABLE analyses ADD CONSTRAINT analyses_status_check
      CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'approved', 'sent'));