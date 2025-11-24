-- 1. Update all "finished" to "completed"
UPDATE enrichments SET status = 'completed' WHERE status = 'finished';
UPDATE analyses SET status = 'completed' WHERE status = 'finished';

-- 2. Update status constraints
ALTER TABLE enrichments DROP CONSTRAINT IF EXISTS enrichments_status_check;
ALTER TABLE enrichments ADD CONSTRAINT enrichments_status_check
CHECK (status IN ('pending', 'completed', 'approved'));

ALTER TABLE analyses DROP CONSTRAINT IF EXISTS analyses_status_check;
ALTER TABLE analyses ADD CONSTRAINT analyses_status_check
CHECK (status IN ('pending', 'completed', 'approved', 'sent'));

-- 3. Add progress tracking to enrichments
ALTER TABLE enrichments
ADD COLUMN IF NOT EXISTS progress INTEGER DEFAULT 0 CHECK (progress >= 0 AND progress <= 100);

ALTER TABLE enrichments
ADD COLUMN IF NOT EXISTS current_step TEXT DEFAULT '';

-- 4. Ensure timestamp triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_submissions_updated_at ON submissions;
CREATE TRIGGER update_submissions_updated_at BEFORE UPDATE
    ON submissions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_enrichments_updated_at ON enrichments;
CREATE TRIGGER update_enrichments_updated_at BEFORE UPDATE
    ON enrichments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_analyses_updated_at ON analyses;
CREATE TRIGGER update_analyses_updated_at BEFORE UPDATE
    ON analyses FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 5. Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_enrichments_status ON enrichments(status);
CREATE INDEX IF NOT EXISTS idx_enrichments_submission_id ON enrichments(submission_id);
CREATE INDEX IF NOT EXISTS idx_analyses_status ON analyses(status);
CREATE INDEX IF NOT EXISTS idx_analyses_submission_id ON analyses(submission_id);
