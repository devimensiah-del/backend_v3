-- =============================================================================
-- Migration: v2_009_challenge_entity
-- Purpose: Introduce Challenge entity, refactor submission/analysis relationships
-- Date: 2024-12-04
-- Updated: 2024-12-05 (removed contact fields - they stay on Submission)
-- =============================================================================
--
-- CONTEXT:
-- Separates "challenge definition" from "submission entry point". A company can
-- have multiple challenges over time. Analysis now links to Challenge (required)
-- instead of Submission (historical reference only).
--
-- NEW RELATIONSHIPS:
-- Company (1) → Challenge (many) → Analysis (many)
-- Submission (entry form) creates Company + first Challenge
--
-- NOTE: Contact info stays on Submission (the entry point). Challenge is just
-- the problem definition - for re-analyze scenarios there's no "requester".
--
-- =============================================================================

BEGIN;

-- =============================================================================
-- PART 1: Create challenges table
-- =============================================================================

CREATE TABLE IF NOT EXISTS challenges (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    company_id UUID NOT NULL,

    -- Challenge definition (required)
    challenge_category VARCHAR(50) NOT NULL CHECK (challenge_category IN (
        'growth', 'transform', 'transition', 'compete', 'funding'
    )),
    challenge_type VARCHAR(50) NOT NULL CHECK (challenge_type IN (
        -- Growth types
        'growth_organic', 'growth_geographic', 'growth_segment', 'growth_product', 'growth_channel',
        -- Transform types
        'transform_digital', 'transform_model', 'transform_culture', 'transform_operational',
        -- Transition types
        'transition_succession', 'transition_exit', 'transition_merger', 'transition_turnaround',
        -- Compete types
        'compete_differentiate', 'compete_defend', 'compete_reposition',
        -- Funding types
        'funding_raise', 'funding_debt', 'funding_ipo'
    )),
    business_challenge TEXT NOT NULL,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT challenges_company_id_fkey FOREIGN KEY (company_id)
        REFERENCES companies(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_challenges_company_id ON challenges(company_id);
CREATE INDEX IF NOT EXISTS idx_challenges_category ON challenges(challenge_category);
CREATE INDEX IF NOT EXISTS idx_challenges_type ON challenges(challenge_type);
CREATE INDEX IF NOT EXISTS idx_challenges_deleted_at ON challenges(deleted_at);

-- Comments for documentation
COMMENT ON TABLE challenges IS 'Strategic business challenges for companies - defines the problem to solve';
COMMENT ON COLUMN challenges.challenge_category IS 'Top-level: growth, transform, transition, compete, funding';
COMMENT ON COLUMN challenges.challenge_type IS 'Specific type: growth_organic, transition_exit, etc.';
COMMENT ON COLUMN challenges.business_challenge IS 'Free-text description of the challenge context';

-- =============================================================================
-- PART 2: Create default challenges for ALL companies
-- =============================================================================

-- Create a challenge for EVERY company (not just those with submissions)
-- This ensures all analyses can be linked to a challenge
INSERT INTO challenges (
    id,
    company_id,
    challenge_category,
    challenge_type,
    business_challenge,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid() as id,
    c.id as company_id,
    'growth' as challenge_category,
    'growth_organic' as challenge_type,
    COALESCE(
        (SELECT s.business_challenge
         FROM submissions s
         INNER JOIN company_submissions cs ON s.id = cs.submission_id
         WHERE cs.company_id = c.id AND cs.is_primary = true
         LIMIT 1),
        'Challenge created during v2 migration - please update via admin'
    ) as business_challenge,
    c.created_at,
    c.updated_at
FROM companies c
WHERE NOT EXISTS (SELECT 1 FROM challenges ch WHERE ch.company_id = c.id)
ON CONFLICT DO NOTHING;

-- =============================================================================
-- PART 3: Update analyses table to link to challenges
-- =============================================================================

-- Add challenge_id column (nullable initially for migration)
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS challenge_id UUID;

-- Strategy 1: Populate via submission → company_submissions → company → challenge
UPDATE analyses a
SET challenge_id = ch.id
FROM challenges ch
INNER JOIN company_submissions cs ON ch.company_id = cs.company_id
WHERE a.submission_id = cs.submission_id
  AND cs.is_primary = true
  AND a.challenge_id IS NULL;

-- Strategy 2: Populate via direct company_id on analyses
UPDATE analyses a
SET challenge_id = ch.id
FROM challenges ch
WHERE a.company_id = ch.company_id
  AND a.challenge_id IS NULL;

-- Strategy 3: For any remaining orphans, create a company + challenge
-- First, identify orphan analyses and create placeholder companies
INSERT INTO companies (id, name, created_at, updated_at)
SELECT
    gen_random_uuid(),
    COALESCE(s.company_name, 'Orphan Company - Analysis ' || a.id::text),
    a.created_at,
    a.updated_at
FROM analyses a
LEFT JOIN submissions s ON a.submission_id = s.id
WHERE a.challenge_id IS NULL
  AND a.company_id IS NULL
ON CONFLICT DO NOTHING;

-- Link orphan analyses to their new companies and create challenges
DO $$
DECLARE
    orphan RECORD;
    new_company_id UUID;
    new_challenge_id UUID;
BEGIN
    FOR orphan IN
        SELECT a.id as analysis_id, a.submission_id, a.created_at, s.company_name
        FROM analyses a
        LEFT JOIN submissions s ON a.submission_id = s.id
        WHERE a.challenge_id IS NULL
    LOOP
        -- Create company if needed
        INSERT INTO companies (name, created_at, updated_at)
        VALUES (
            COALESCE(orphan.company_name, 'Orphan Company - ' || orphan.analysis_id::text),
            orphan.created_at,
            NOW()
        )
        RETURNING id INTO new_company_id;

        -- Create challenge for the company
        INSERT INTO challenges (company_id, challenge_category, challenge_type, business_challenge)
        VALUES (new_company_id, 'growth', 'growth_organic', 'Orphan analysis migrated - please review')
        RETURNING id INTO new_challenge_id;

        -- Update analysis with company and challenge
        UPDATE analyses
        SET company_id = new_company_id, challenge_id = new_challenge_id
        WHERE id = orphan.analysis_id;
    END LOOP;
END $$;

-- Verify no NULLs remain before applying constraint
DO $$
DECLARE
    null_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO null_count FROM analyses WHERE challenge_id IS NULL;
    IF null_count > 0 THEN
        RAISE EXCEPTION 'v2_008 ABORT: % analyses still have NULL challenge_id', null_count;
    END IF;
END $$;

-- Now safe to make NOT NULL
ALTER TABLE analyses ALTER COLUMN challenge_id SET NOT NULL;

-- Add foreign key constraint
ALTER TABLE analyses ADD CONSTRAINT analyses_challenge_id_fkey
    FOREIGN KEY (challenge_id) REFERENCES challenges(id) ON DELETE CASCADE;

-- Make submission_id nullable (historical reference only)
ALTER TABLE analyses ALTER COLUMN submission_id DROP NOT NULL;

-- Create index for challenge_id lookups
CREATE INDEX IF NOT EXISTS idx_analyses_challenge_id ON analyses(challenge_id);

COMMENT ON COLUMN analyses.challenge_id IS 'Required: The challenge this analysis addresses';
COMMENT ON COLUMN analyses.submission_id IS 'Optional: Historical reference to originating submission';

-- =============================================================================
-- PART 4: Clean up submissions table
-- =============================================================================

-- Drop status column if it exists (always "received", useless)
-- NOTE: These columns may not exist if running on a fresh schema
ALTER TABLE submissions DROP COLUMN IF EXISTS status;

-- Drop legacy challenge columns if they exist (moved to challenges table)
-- These were added by some older migrations but never used in Go code
ALTER TABLE submissions DROP COLUMN IF EXISTS challenge_category;
ALTER TABLE submissions DROP COLUMN IF EXISTS challenge_type;
ALTER TABLE submissions DROP COLUMN IF EXISTS business_challenge;

-- Drop indexes related to dropped columns (safe even if they don't exist)
DROP INDEX IF EXISTS idx_submissions_challenge_category;
DROP INDEX IF EXISTS idx_submissions_challenge_type;
DROP INDEX IF EXISTS idx_submissions_status;

-- Update comments
COMMENT ON TABLE submissions IS 'Public submission form - entry point that creates Company + first Challenge';
COMMENT ON COLUMN submissions.contact_email IS 'Contact from form - grants read access to linked company';

-- =============================================================================
-- PART 5: Add unique constraint on submission → company (one submission per company)
-- =============================================================================

-- This ensures submissions table is truly the entry point (1:1 with company after migration)
-- Note: This might fail if data has duplicates - handle manually if needed
-- CREATE UNIQUE INDEX IF NOT EXISTS idx_company_submissions_unique_submission
--     ON company_submissions(submission_id) WHERE is_primary = true;

COMMIT;

-- =============================================================================
-- DOWN MIGRATION (Rollback)
-- =============================================================================
--
-- To rollback this migration, run the following:
--
-- BEGIN;
--
-- -- Remove unique constraint
-- DROP INDEX IF EXISTS idx_company_submissions_unique_submission;
--
-- -- Add columns back to submissions
-- ALTER TABLE submissions ADD COLUMN IF NOT EXISTS status VARCHAR NOT NULL DEFAULT 'received'
--     CHECK (status = 'received');
-- ALTER TABLE submissions ADD COLUMN IF NOT EXISTS challenge_category VARCHAR(50);
-- ALTER TABLE submissions ADD COLUMN IF NOT EXISTS challenge_type VARCHAR(50);
-- ALTER TABLE submissions ADD COLUMN IF NOT EXISTS business_challenge TEXT;
--
-- -- Repopulate challenge fields from challenges table
-- UPDATE submissions s
-- SET
--     challenge_category = c.challenge_category,
--     challenge_type = c.challenge_type,
--     business_challenge = c.business_challenge
-- FROM challenges c
-- INNER JOIN company_submissions cs ON c.company_id = cs.company_id
-- WHERE s.id = cs.submission_id
--   AND cs.is_primary = true;
--
-- -- Make business_challenge NOT NULL again
-- ALTER TABLE submissions ALTER COLUMN business_challenge SET NOT NULL;
--
-- -- Recreate indexes
-- CREATE INDEX IF NOT EXISTS idx_submissions_challenge_category ON submissions(challenge_category);
-- CREATE INDEX IF NOT EXISTS idx_submissions_challenge_type ON submissions(challenge_type);
-- CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions(status);
--
-- -- Revert analyses table changes
-- ALTER TABLE analyses ALTER COLUMN submission_id SET NOT NULL;
-- ALTER TABLE analyses DROP CONSTRAINT IF EXISTS analyses_challenge_id_fkey;
-- DROP INDEX IF EXISTS idx_analyses_challenge_id;
-- ALTER TABLE analyses DROP COLUMN IF EXISTS challenge_id;
--
-- -- Drop challenges table
-- DROP INDEX IF EXISTS idx_challenges_deleted_at;
-- DROP INDEX IF EXISTS idx_challenges_type;
-- DROP INDEX IF EXISTS idx_challenges_category;
-- DROP INDEX IF EXISTS idx_challenges_company_id;
-- DROP TABLE IF EXISTS challenges CASCADE;
--
-- COMMIT;
