-- Migration: Company Verification System
-- Purpose: Add verification status and user access control to companies
-- When is_verified=true AND cnpj IS NOT NULL, CNPJ becomes unique (no duplicate submissions)

-- ============================================
-- 1. Add new columns to companies table
-- ============================================

-- is_verified: Admin-controlled flag. When true + CNPJ present = CNPJ is locked
ALTER TABLE companies ADD COLUMN IF NOT EXISTS is_verified BOOLEAN NOT NULL DEFAULT false;

-- allowed_users: Array of user UUIDs who can access this company's data
-- Empty array means only admin can access (or original submitter via submission link)
ALTER TABLE companies ADD COLUMN IF NOT EXISTS allowed_users UUID[] NOT NULL DEFAULT '{}';

-- ============================================
-- 2. Create partial unique index on CNPJ
-- Only enforced when company is verified AND has a CNPJ
-- This prevents new submissions with the same CNPJ
-- ============================================

CREATE UNIQUE INDEX IF NOT EXISTS idx_companies_verified_cnpj
ON companies (cnpj)
WHERE is_verified = true AND cnpj IS NOT NULL;

-- ============================================
-- 3. Add indexes for efficient lookups
-- ============================================

-- Fast lookup for verified companies
CREATE INDEX IF NOT EXISTS idx_companies_verified ON companies (is_verified) WHERE is_verified = true;

-- Fast lookup by allowed users (GIN for array containment)
CREATE INDEX IF NOT EXISTS idx_companies_allowed_users ON companies USING GIN (allowed_users);

-- ============================================
-- 4. Add comments for documentation
-- ============================================

COMMENT ON COLUMN companies.is_verified IS 'Admin-verified company. When true with CNPJ, the CNPJ becomes unique and blocked for new submissions.';
COMMENT ON COLUMN companies.allowed_users IS 'Array of user UUIDs who can access this company data. Empty means admin-only access.';
COMMENT ON INDEX idx_companies_verified_cnpj IS 'Enforces CNPJ uniqueness only for verified companies with a CNPJ.';
