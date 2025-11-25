-- Migration: Add CNPJ column to submissions table
-- CNPJ is Brazilian company registration number, needed for form alignment

ALTER TABLE submissions
ADD COLUMN IF NOT EXISTS cnpj VARCHAR(18);

COMMENT ON COLUMN submissions.cnpj IS 'CNPJ - Brazilian company registration number (XX.XXX.XXX/XXXX-XX format)';

-- Create index for potential lookups
CREATE INDEX IF NOT EXISTS idx_submissions_cnpj ON submissions(cnpj) WHERE cnpj IS NOT NULL;
