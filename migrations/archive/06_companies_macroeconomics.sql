-- =============================================================================
-- ARCHIVE: Companies & Macroeconomics (Migrations 017-020, 022, 024, 025, 027)
-- Original files: 017_macroeconomics.sql, 018_macro_indicators_expansion.sql,
--                 019_companies_table.sql, 020_company_verification.sql,
--                 022_company_owner.sql, 024_field_verification.sql,
--                 025_digital_maturity_scale.sql, 027_enrichments_company_id.sql
-- Status: APPLIED TO PRODUCTION
-- =============================================================================
-- This file is for historical reference only. DO NOT RUN.
-- =============================================================================

-- Migration 017: Created macroeconomics tables
-- - macro_data_sources (BCB, IBGE, etc.)
-- - macro_indicator_types (SELIC, IPCA, USD/BRL)
-- - macro_indicator_sources (linking)
-- - macro_indicator_values (actual data)
-- - macro_fetch_logs (API tracking)

-- Migration 018: Expanded macro indicators
-- - Added more indicator types and sources

-- Migration 019: Created companies table
-- - Verified company records with detailed fields
-- - company_submissions linking table
-- - company_data_history for change tracking

-- Migration 020: Added company verification workflow
-- - is_verified boolean
-- - allowed_users UUID array

-- Migration 022: Added owner_id to companies
-- - Links company to owning user

-- Migration 024: Added field-level verification
-- - company_field_verifications table
-- - field_deprecation_config table

-- Migration 025: Changed digital_maturity to 1-10 scale
-- - Previously was percentage, now integer 1-10

-- Migration 027: Added company_id to enrichments
-- - Links enrichments to verified companies
