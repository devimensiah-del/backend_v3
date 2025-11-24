-- Align status constraints with current workflow contract
-- submission: always 'received'
-- enrichment: pending | completed | approved
-- analysis: pending | completed | approved | sent
-- report: pending | processing | completed | failed

-- Normalize existing data to allowed values
UPDATE submissions SET status = 'received' WHERE status IS NULL OR status <> 'received';
UPDATE enrichments SET status = 'pending' WHERE status NOT IN ('pending', 'completed', 'approved');
UPDATE analyses SET status = 'pending' WHERE status NOT IN ('pending', 'completed', 'approved', 'sent');
UPDATE reports SET status = 'pending' WHERE status NOT IN ('pending', 'processing', 'completed', 'failed');

-- Drop old constraints if they exist
ALTER TABLE submissions DROP CONSTRAINT IF EXISTS valid_status;
ALTER TABLE enrichments DROP CONSTRAINT IF EXISTS valid_enrichment_status;
ALTER TABLE analyses DROP CONSTRAINT IF EXISTS valid_analysis_status;
ALTER TABLE reports DROP CONSTRAINT IF EXISTS valid_report_status;
ALTER TABLE reports DROP CONSTRAINT IF EXISTS valid_pdf_status;

-- Add new constraints
ALTER TABLE submissions
    ADD CONSTRAINT chk_submissions_status_received CHECK (status = 'received');

ALTER TABLE enrichments
    ADD CONSTRAINT chk_enrichments_status CHECK (status IN ('pending', 'completed', 'approved'));

ALTER TABLE analyses
    ADD CONSTRAINT chk_analyses_status CHECK (status IN ('pending', 'completed', 'approved', 'sent'));

ALTER TABLE reports
    ADD CONSTRAINT chk_reports_status CHECK (status IN ('pending', 'processing', 'completed', 'failed'));
