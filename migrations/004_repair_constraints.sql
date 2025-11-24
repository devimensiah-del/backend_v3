-- 004_fix_constraints_idempotent.sql
-- Repairs and aligns constraints. Safe to re-run (Idempotent).

BEGIN;

-- 1. Prerequisites
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
        CREATE TYPE user_role AS ENUM ('user', 'admin', 'super_admin', 'service_role');
    END IF;
END$$;

-- 2. Submissions
-- We drop 'submissions_status_check' (auto-gen), 'valid_status' (old), AND 'chk_submissions_status_received' (current target)
ALTER TABLE IF EXISTS submissions
    DROP CONSTRAINT IF EXISTS submissions_status_check,
    DROP CONSTRAINT IF EXISTS valid_status,
    DROP CONSTRAINT IF EXISTS chk_submissions_status_received,
    ADD CONSTRAINT chk_submissions_status_received CHECK (status = 'received');

-- 3. Enrichments
-- Drop auto-gen, old, and current target
ALTER TABLE IF EXISTS enrichments
    DROP CONSTRAINT IF EXISTS enrichments_status_check,
    DROP CONSTRAINT IF EXISTS valid_enrichment_status,
    DROP CONSTRAINT IF EXISTS chk_enrichments_status,
    ADD CONSTRAINT chk_enrichments_status CHECK (status IN ('pending', 'processing', 'completed', 'approved', 'failed'));

-- 4. Analyses
-- Drop auto-gen, old, and current target
ALTER TABLE IF EXISTS analyses
    DROP CONSTRAINT IF EXISTS analyses_status_check,
    DROP CONSTRAINT IF EXISTS valid_analysis_status,
    DROP CONSTRAINT IF EXISTS chk_analyses_status,
    ADD CONSTRAINT chk_analyses_status CHECK (status IN ('pending', 'processing', 'completed', 'approved', 'sent', 'failed'));

-- 5. Reports
-- Drop auto-gen, old, and current target for both status columns
ALTER TABLE IF EXISTS reports
    DROP CONSTRAINT IF EXISTS reports_status_check,
    DROP CONSTRAINT IF EXISTS valid_report_status,
    DROP CONSTRAINT IF EXISTS chk_reports_status,
    ADD CONSTRAINT chk_reports_status CHECK (status IN ('pending', 'processing', 'completed', 'failed'));

ALTER TABLE IF EXISTS reports
    DROP CONSTRAINT IF EXISTS reports_pdf_generation_status_check,
    DROP CONSTRAINT IF EXISTS valid_pdf_status,
    DROP CONSTRAINT IF EXISTS chk_reports_pdf_status,
    ADD CONSTRAINT chk_reports_pdf_status CHECK (pdf_generation_status IN ('pending', 'processing', 'completed', 'failed'));

COMMIT;