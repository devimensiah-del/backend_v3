-- Migration: v2_027_cnpj_duplicates_review
-- Description: Create table for tracking CNPJ duplicate companies needing admin review
-- Date: 2024-12-21

-- NOTE: References company_core (not companies view) for foreign keys

-- Create table for tracking duplicate CNPJ pairs
CREATE TABLE IF NOT EXISTS cnpj_duplicates_review (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cnpj_normalized VARCHAR(14) NOT NULL,
    older_company_id UUID NOT NULL REFERENCES company_core(id) ON DELETE CASCADE,
    newer_company_id UUID NOT NULL REFERENCES company_core(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    reviewed_by UUID REFERENCES user_profiles(id),
    action_taken VARCHAR(20) CHECK (action_taken IN ('merged', 'kept_separate', 'deleted_newer')),
    notes TEXT,

    -- Prevent duplicate review entries for same pair
    CONSTRAINT unique_duplicate_pair UNIQUE (older_company_id, newer_company_id)
);

-- Index for finding unreviewed duplicates
CREATE INDEX IF NOT EXISTS idx_cnpj_duplicates_review_pending
ON cnpj_duplicates_review (created_at)
WHERE reviewed_at IS NULL;

-- Index for finding duplicates by CNPJ
CREATE INDEX IF NOT EXISTS idx_cnpj_duplicates_review_cnpj
ON cnpj_duplicates_review (cnpj_normalized);

-- NOTE: We do NOT auto-populate duplicates here.
-- Use v2_028_merge_cnpj_duplicates.sql to merge existing duplicates instead.
