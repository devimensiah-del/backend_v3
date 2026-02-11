-- v2_043_remove_data_display_frameworks.sql
-- Remove EMPRESA, NEGOCIO, MERCADO from frameworks table
--
-- These are NOT AI frameworks - they display enrichment data from the company record.
-- The frameworks table should only contain actual AI-powered analyses.
--
-- After this migration:
-- - AI frameworks check company.enrichment_status before running
-- - Frontend handles Empresa/Negócio/Mercado tabs independently
-- - Cleaner separation: enrichment data vs AI analysis

-- Step 1: Delete dependencies that reference these frameworks
DELETE FROM framework_dependencies
WHERE framework_id IN (SELECT id FROM frameworks WHERE code IN ('empresa', 'negocio', 'mercado'))
   OR depends_on_id IN (SELECT id FROM frameworks WHERE code IN ('empresa', 'negocio', 'mercado'));

-- Step 2: Delete any results for these frameworks (shouldn't exist, but cleanup)
DELETE FROM company_framework_results
WHERE framework_id IN (SELECT id FROM frameworks WHERE code IN ('empresa', 'negocio', 'mercado'));

-- Step 3: Delete the frameworks themselves
DELETE FROM frameworks WHERE code IN ('empresa', 'negocio', 'mercado');

-- Step 4: Update layer comment
COMMENT ON COLUMN frameworks.layer IS 'Layer number for AI analysis frameworks (1=environment, 2=diagnosis, etc.)';
