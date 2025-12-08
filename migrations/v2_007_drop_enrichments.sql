-- =============================================================================
-- Migration: v2_008_drop_enrichments
-- Purpose: Drop enrichments table - enrichment now happens inline during company creation
-- Date: 2024-12-04
-- =============================================================================
--
-- CONTEXT:
-- Enrichment is being simplified from a separate table/entity to a one-time
-- inline process during company creation. The enrichment workflow will now
-- directly update the companies table instead of storing data in a separate
-- enrichments table.
--
-- NEW DESIGN:
-- - No enrichments table - data goes directly to companies table
-- - Enrichment is triggered once at company creation, never re-triggered
-- - Only Perplexity is used (no Gemini, no macro data integration)
-- - Enrichment status is tracked directly on the companies table
--
-- =============================================================================

BEGIN;

-- =============================================================================
-- PART 1: Add enrichment tracking columns to companies table
-- =============================================================================

-- Add enrichment status tracking to companies (if not already present)
ALTER TABLE companies ADD COLUMN IF NOT EXISTS enrichment_status VARCHAR(20) DEFAULT 'pending'
    CHECK (enrichment_status IN ('pending', 'processing', 'completed', 'failed'));

ALTER TABLE companies ADD COLUMN IF NOT EXISTS enrichment_completed_at TIMESTAMPTZ;

ALTER TABLE companies ADD COLUMN IF NOT EXISTS enrichment_error TEXT;

-- Add comments for documentation
COMMENT ON COLUMN companies.enrichment_status IS 'One-time enrichment status: pending, processing, completed, or failed';
COMMENT ON COLUMN companies.enrichment_completed_at IS 'Timestamp when enrichment finished (success or failure)';
COMMENT ON COLUMN companies.enrichment_error IS 'Error message if enrichment failed';

-- =============================================================================
-- PART 2: Drop foreign key constraints referencing enrichments table
-- =============================================================================

-- Drop the foreign key from analyses table that references enrichments
ALTER TABLE analyses DROP CONSTRAINT IF EXISTS analyses_enrichment_id_fkey;

-- Drop the foreign key from enrichments to companies (reverse relationship)
ALTER TABLE enrichments DROP CONSTRAINT IF EXISTS fk_enrichments_company;

-- Drop the foreign key from enrichments to submissions
ALTER TABLE enrichments DROP CONSTRAINT IF EXISTS enrichments_submission_id_fkey;

-- =============================================================================
-- PART 3: Drop indexes related to enrichments
-- =============================================================================

DROP INDEX IF EXISTS idx_enrichments_submission_id;
DROP INDEX IF EXISTS idx_enrichments_status;
DROP INDEX IF EXISTS idx_enrichments_company_id;

-- Drop index on analyses.enrichment_id (will be dropped with the column)
DROP INDEX IF EXISTS idx_analyses_enrichment_id;

-- =============================================================================
-- PART 4: Drop enrichment_id column from analyses table
-- =============================================================================

-- Remove the enrichment_id column from analyses (no longer needed)
ALTER TABLE analyses DROP COLUMN IF EXISTS enrichment_id;

-- =============================================================================
-- PART 5: Drop the enrichments table
-- =============================================================================

DROP TABLE IF EXISTS enrichments CASCADE;

-- =============================================================================
-- PART 6: Create index for new enrichment_status column
-- =============================================================================

CREATE INDEX IF NOT EXISTS idx_companies_enrichment_status ON companies(enrichment_status);

COMMIT;

-- =============================================================================
-- DOWN MIGRATION (Rollback)
-- =============================================================================
--
-- This DOWN migration recreates the enrichments table structure as it existed
-- in the baseline schema. Note: Data will NOT be restored, only structure.
--
-- To rollback, run the following:
--
-- BEGIN;
--
-- -- Recreate enrichments table
-- CREATE TABLE IF NOT EXISTS enrichments (
--     id UUID NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
--     submission_id UUID NOT NULL UNIQUE,
--     status VARCHAR NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'approved', 'failed')),
--     progress INTEGER NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
--     current_step TEXT,
--     sources_status JSONB,
--     sources_used TEXT[],
--     error_message TEXT,
--     retry_count INTEGER NOT NULL DEFAULT 0,
--     max_retries INTEGER NOT NULL DEFAULT 3,
--     started_at TIMESTAMPTZ,
--     completed_at TIMESTAMPTZ,
--     is_locked BOOLEAN DEFAULT false,
--     data JSONB DEFAULT '{}',
--     auto_trigger_analysis BOOLEAN NOT NULL DEFAULT false,
--     approved_at TIMESTAMPTZ,
--     company_id UUID,
--     raw_ai_response JSONB,
--     processing_time_ms INTEGER,
--     created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
--     updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
-- );
--
-- -- Recreate foreign keys
-- ALTER TABLE enrichments ADD CONSTRAINT enrichments_submission_id_fkey
--     FOREIGN KEY (submission_id) REFERENCES submissions(id);
--
-- ALTER TABLE enrichments ADD CONSTRAINT fk_enrichments_company
--     FOREIGN KEY (company_id) REFERENCES companies(id);
--
-- -- Recreate indexes
-- CREATE INDEX IF NOT EXISTS idx_enrichments_submission_id ON enrichments(submission_id);
-- CREATE INDEX IF NOT EXISTS idx_enrichments_status ON enrichments(status);
-- CREATE INDEX IF NOT EXISTS idx_enrichments_company_id ON enrichments(company_id);
--
-- -- Add enrichment_id back to analyses table
-- ALTER TABLE analyses ADD COLUMN IF NOT EXISTS enrichment_id UUID;
--
-- -- Recreate foreign key from analyses to enrichments
-- ALTER TABLE analyses ADD CONSTRAINT analyses_enrichment_id_fkey
--     FOREIGN KEY (enrichment_id) REFERENCES enrichments(id);
--
-- -- Recreate index on analyses.enrichment_id
-- CREATE INDEX IF NOT EXISTS idx_analyses_enrichment_id ON analyses(enrichment_id);
--
-- -- Drop enrichment tracking from companies table
-- DROP INDEX IF EXISTS idx_companies_enrichment_status;
-- ALTER TABLE companies DROP COLUMN IF EXISTS enrichment_status;
-- ALTER TABLE companies DROP COLUMN IF EXISTS enrichment_completed_at;
-- ALTER TABLE companies DROP COLUMN IF EXISTS enrichment_error;
--
-- COMMIT;
