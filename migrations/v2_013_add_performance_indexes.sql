-- v2_013_add_performance_indexes.sql
-- Performance indexes for common query patterns
--
-- NOTE: Using regular CREATE INDEX (not CONCURRENTLY) because migration tools
-- run inside transactions. For zero-downtime on high-traffic production tables,
-- run these manually with CONCURRENTLY outside a transaction:
--   CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_submissions_email_lower ON submissions (lower(contact_email));

-- Case-insensitive email lookup on submissions
CREATE INDEX IF NOT EXISTS idx_submissions_email_lower
ON submissions (lower(contact_email));

-- Challenge lookups by company
CREATE INDEX IF NOT EXISTS idx_challenges_company_id
ON challenges (company_id) WHERE deleted_at IS NULL;

-- Company-submission join optimization
CREATE INDEX IF NOT EXISTS idx_company_submissions_submission_id
ON company_submissions (submission_id);
