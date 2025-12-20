-- Migration: v2_021_enrichment_3_steps.sql
-- Description: Create company_enrichment table for 3-step enrichment tracking
-- Date: 2024-12-20

-- =============================================================================
-- NEW TABLE: company_enrichment
-- Tracks 3-step enrichment status per company
-- =============================================================================

CREATE TABLE IF NOT EXISTS company_enrichment (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

  -- Step 1: Basic Info (automatic at company creation)
  -- Fields: CNPJ, razão social, fundação, sede, funcionários, website, redes sociais, executivos
  step1_status VARCHAR(20) DEFAULT 'pending' CHECK (step1_status IN ('pending', 'processing', 'completed', 'failed')),
  step1_completed_at TIMESTAMPTZ,
  step1_error TEXT,
  step1_data JSONB,  -- Store raw Step 1 result for context building in subsequent steps

  -- Step 2: Business Model (human-triggered after Step 1)
  -- Fields: modelo de negócio, produtos/serviços, público alvo, proposta de valor, região geográfica
  step2_status VARCHAR(20) DEFAULT 'pending' CHECK (step2_status IN ('pending', 'processing', 'completed', 'failed')),
  step2_completed_at TIMESTAMPTZ,
  step2_error TEXT,
  step2_data JSONB,  -- Store raw Step 2 result for context building in Step 3

  -- Step 3: Competitive Intelligence (human-triggered after Step 2)
  -- Fields: concorrentes, informações do setor, reputação, notícias recentes
  step3_status VARCHAR(20) DEFAULT 'pending' CHECK (step3_status IN ('pending', 'processing', 'completed', 'failed')),
  step3_completed_at TIMESTAMPTZ,
  step3_error TEXT,
  step3_data JSONB,

  -- Timestamps
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),

  -- Constraint: One enrichment record per company
  UNIQUE(company_id)
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_company_enrichment_company_id ON company_enrichment(company_id);
CREATE INDEX IF NOT EXISTS idx_company_enrichment_step1_status ON company_enrichment(step1_status);
CREATE INDEX IF NOT EXISTS idx_company_enrichment_step2_status ON company_enrichment(step2_status);
CREATE INDEX IF NOT EXISTS idx_company_enrichment_step3_status ON company_enrichment(step3_status);

-- =============================================================================
-- ALTER TABLE: companies
-- Add geographic_regions field (output from Step 2)
-- =============================================================================

ALTER TABLE companies
  ADD COLUMN IF NOT EXISTS geographic_regions JSONB DEFAULT '[]';

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE company_enrichment IS 'Tracks 3-step enrichment status and data per company';
COMMENT ON COLUMN company_enrichment.step1_data IS 'Raw JSON result from Step 1 enrichment (basic info)';
COMMENT ON COLUMN company_enrichment.step2_data IS 'Raw JSON result from Step 2 enrichment (business model)';
COMMENT ON COLUMN company_enrichment.step3_data IS 'Raw JSON result from Step 3 enrichment (competitive intel)';
COMMENT ON COLUMN companies.geographic_regions IS 'Geographic regions where company operates (from Step 2 enrichment)';
