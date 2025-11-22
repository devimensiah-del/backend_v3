-- ==================== 1. ADD CNPJ COLUMN ====================
-- Add CNPJ (Brazilian company registration number) to submissions table
-- Format: XX.XXX.XXX/XXXX-XX (18 characters with formatting)
ALTER TABLE submissions
ADD COLUMN IF NOT EXISTS cnpj VARCHAR(18);

-- Add index for CNPJ lookups (useful for duplicate detection)
CREATE INDEX IF NOT EXISTS idx_submissions_cnpj ON submissions(cnpj) WHERE cnpj IS NOT NULL AND deleted_at IS NULL;

-- ==================== 2. UPDATE STATUS CONSTRAINT ====================
-- Drop the old constraint
ALTER TABLE submissions
DROP CONSTRAINT IF EXISTS valid_status;

-- Add new constraint with additional statuses to match frontend SubmissionStatus type
-- New statuses: 'processing' (generic processing state), 'ready_for_review' (waiting for admin publish)
ALTER TABLE submissions
ADD CONSTRAINT valid_status CHECK (status IN (
    -- Workflow States
    'pending',              -- Initial submission
    'processing',           -- Generic processing state
    'enriching',            -- Worker 1 Active
    'enriched',             -- Worker 1 Done
    'analyzing',            -- Worker 2 Active
    'analyzed',             -- Worker 2 Done (Internal)
    'ready_for_review',     -- Waiting for Admin Publish
    'generating_report',    -- PDF generation in progress
    'completed',            -- PDF Generated & Email Sent
    -- Failure States
    'enrichment_failed',
    'analysis_failed',
    'report_failed',
    'failed'                -- Generic Failure
));

-- ==================== 3. COMMENT FOR DOCUMENTATION ====================
COMMENT ON COLUMN submissions.cnpj IS 'Brazilian CNPJ (Cadastro Nacional da Pessoa Jurídica) - Company registration number';
COMMENT ON CONSTRAINT valid_status ON submissions IS 'Valid submission workflow statuses matching frontend SubmissionStatus type';
