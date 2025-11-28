-- Migration 024: Field-level verification for Company
-- Purpose: Track which company fields are verified and protected from re-enrichment (smart merge)

CREATE TABLE IF NOT EXISTS company_field_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    field_name TEXT NOT NULL,
    verified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    verified_by UUID,

    UNIQUE(company_id, field_name)
);

-- Index for looking up all verified fields for a company
CREATE INDEX IF NOT EXISTS idx_cfv_company ON company_field_verifications(company_id);

-- Index for checking if specific field is verified
CREATE INDEX IF NOT EXISTS idx_cfv_company_field ON company_field_verifications(company_id, field_name);

COMMENT ON TABLE company_field_verifications IS 'Tracks which company fields are verified and protected from re-enrichment';
COMMENT ON COLUMN company_field_verifications.field_name IS 'Column name from companies table (e.g., cnpj, industry, sector)';
COMMENT ON COLUMN company_field_verifications.verified_by IS 'User ID of admin who verified this field';
