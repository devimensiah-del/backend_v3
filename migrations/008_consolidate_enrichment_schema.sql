-- Migration: Consolidate enrichment columns into enriched_data
-- This migration consolidates the old multi-column schema into a single JSONB column
-- to match the new simplified enrichment model

BEGIN;

-- Step 1: Add the new enriched_data column if it doesn't exist
ALTER TABLE enrichments
ADD COLUMN IF NOT EXISTS enriched_data JSONB DEFAULT '{}'::jsonb;

-- Step 2: Migrate existing data from old columns to enriched_data
-- Only update rows where enriched_data is empty/null
UPDATE enrichments
SET enriched_data = jsonb_build_object(
    'company_info', COALESCE(company_info, '{}'::jsonb),
    'market_data', COALESCE(market_data, '{}'::jsonb),
    'competitors', COALESCE(competitors, '{}'::jsonb),
    'financial_data', COALESCE(financial_data, '{}'::jsonb),
    'news', COALESCE(news, '{}'::jsonb),
    'social_media', COALESCE(social_media, '{}'::jsonb),
    'technologies', COALESCE(technologies, '{}'::jsonb),
    'key_people', COALESCE(key_people, '{}'::jsonb),
    'funding', COALESCE(funding, '{}'::jsonb),
    'partnerships', COALESCE(partnerships, '{}'::jsonb),
    'products', COALESCE(products, '{}'::jsonb)
)
WHERE enriched_data = '{}'::jsonb OR enriched_data IS NULL;

-- Step 3: Drop old individual columns (optional - comment out if you want to keep them temporarily)
-- ALTER TABLE enrichments DROP COLUMN IF EXISTS company_info;
-- ALTER TABLE enrichments DROP COLUMN IF EXISTS market_data;
-- ALTER TABLE enrichments DROP COLUMN IF EXISTS competitors;
-- ALTER TABLE enrichments DROP COLUMN IF EXISTS financial_data;
-- ALTER TABLE enrichments DROP COLUMN IF EXISTS news;
-- ALTER TABLE enrichments DROP COLUMN IF EXISTS social_media;
-- ALTER TABLE enrichments DROP COLUMN IF EXISTS technologies;
-- ALTER TABLE enrichments DROP COLUMN IF EXISTS key_people;
-- ALTER TABLE enrichments DROP COLUMN IF EXISTS funding;
-- ALTER TABLE enrichments DROP COLUMN IF EXISTS partnerships;
-- ALTER TABLE enrichments DROP COLUMN IF EXISTS products;

COMMIT;
