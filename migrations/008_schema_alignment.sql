-- Migration 008: Schema Alignment with Application Models
-- Purpose: Add missing columns required by Go models and repositories
-- Author: Schema Analysis
-- Date: 2025-11-24
--
-- CRITICAL FIXES:
-- 1. submissions.cnpj - Model expects this column, used in CREATE/UPDATE
-- 2. reports.financial_projections_page - Model expects this column, used in all queries
-- 3. reports.risk_assessment_page - Model expects this column, used in all queries
--
-- PERFORMANCE IMPROVEMENTS:
-- 4. Add indexes on frequently queried columns

BEGIN;

-- =============================================================================
-- 1. CRITICAL: Add missing cnpj column to submissions
-- =============================================================================
-- Referenced in:
--   - domain/submission/model.go:19 (CNPJ *string `db:"cnpj"`)
--   - domain/submission/repository.go:58,64 (INSERT)
--   - domain/submission/repository.go:246 (UPDATE)
--
-- Without this column, all submission CREATE and UPDATE operations FAIL.

ALTER TABLE submissions
    ADD COLUMN IF NOT EXISTS cnpj character varying;

COMMENT ON COLUMN submissions.cnpj IS 'Brazilian company registration number (CNPJ)';

-- =============================================================================
-- 2. CRITICAL: Add missing financial_projections_page column to reports
-- =============================================================================
-- Referenced in:
--   - domain/report/model.go:36 (FinancialProjectionsPage string `db:"financial_projections_page"`)
--   - domain/report/repository.go:42,82,121,168,212,244,276 (all CRUD operations)
--
-- Without this column, all report operations FAIL.

ALTER TABLE reports
    ADD COLUMN IF NOT EXISTS financial_projections_page text;

COMMENT ON COLUMN reports.financial_projections_page IS 'HTML content for financial projections page (Page 20)';

-- =============================================================================
-- 3. CRITICAL: Add missing risk_assessment_page column to reports
-- =============================================================================
-- Referenced in:
--   - domain/report/model.go:38 (RiskAssessmentPage string `db:"risk_assessment_page"`)
--   - domain/report/repository.go:43,83,123,170,213,245,277 (all CRUD operations)
--
-- Without this column, all report operations FAIL.

ALTER TABLE reports
    ADD COLUMN IF NOT EXISTS risk_assessment_page text;

COMMENT ON COLUMN reports.risk_assessment_page IS 'HTML content for risk assessment page (Page 22)';

-- =============================================================================
-- 4. CRITICAL: Add missing UNIQUE constraint on enrichments.submission_id
-- =============================================================================
-- Required for ON CONFLICT (submission_id) DO NOTHING in ReserveEnrichment()
-- Migration 005 was supposed to add this but it's missing from the database.
--
-- Referenced in:
--   - domain/submission/repository.go: ReserveEnrichment() uses ON CONFLICT
--   - domain/submission/workflow.go:82: Idempotency guard for job enqueue

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'enrichments_submission_id_key'
    ) THEN
        ALTER TABLE enrichments
            ADD CONSTRAINT enrichments_submission_id_key UNIQUE (submission_id);
    END IF;
END $$;

-- =============================================================================
-- 5. PERFORMANCE: Add missing indexes
-- =============================================================================
-- These indexes improve query performance for common operations:
--   - Status filtering (worker processing, API filtering)
--   - Foreign key lookups (joins in analytics queries)

-- Index for analyses status (used in analytics and filtering)
CREATE INDEX IF NOT EXISTS idx_analyses_status
    ON analyses(status);

-- Index for analyses enrichment_id (foreign key lookups)
CREATE INDEX IF NOT EXISTS idx_analyses_enrichment_id
    ON analyses(enrichment_id);

-- Index for enrichments status (worker processing queries)
CREATE INDEX IF NOT EXISTS idx_enrichments_status
    ON enrichments(status);

-- Note: UNIQUE constraints on submission_id in enrichments/analyses
-- automatically create indexes, but we document them here for clarity.

-- Add index comments
COMMENT ON INDEX idx_analyses_status IS 'Performance: Filter analyses by status in List() and analytics';
COMMENT ON INDEX idx_analyses_enrichment_id IS 'Performance: Foreign key lookups for enrichment data';
COMMENT ON INDEX idx_enrichments_status IS 'Performance: Worker status filtering for pending/processing enrichments';

COMMIT;
