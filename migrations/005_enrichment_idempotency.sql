-- 005_enrichment_idempotency.sql
-- Enforce one-to-one submission -> enrichment, align array types, and keep worker states explicit.

BEGIN;

-- 1) Ensure submission_id is unique so we never store duplicate enrichments for the same submission.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'enrichments_submission_id_key'
    ) THEN
        ALTER TABLE IF EXISTS enrichments
            ADD CONSTRAINT enrichments_submission_id_key UNIQUE (submission_id);
    END IF;
END$$;

-- 2) Normalize sources_used to text[] with a safe cast and default empty array.
ALTER TABLE IF EXISTS enrichments
    ALTER COLUMN sources_used SET DATA TYPE text[] USING
        CASE
            WHEN sources_used IS NULL THEN NULL
            ELSE ARRAY(SELECT unnest(sources_used)::text)
        END,
    ALTER COLUMN sources_used SET DEFAULT '{}'::text[];

-- 3) Make sure worker states remain allowed at the database layer (idempotent with migration 004).
ALTER TABLE IF EXISTS enrichments
    DROP CONSTRAINT IF EXISTS chk_enrichments_status,
    ADD CONSTRAINT chk_enrichments_status CHECK (
        status IN ('pending', 'processing', 'completed', 'approved', 'failed')
    );

COMMIT;
