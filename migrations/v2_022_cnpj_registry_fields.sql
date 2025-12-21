-- v2_022_cnpj_registry_fields.sql
-- Add CNPJ registry fields to companies table (from casadosdados.com.br)

-- CNPJ Registry data
ALTER TABLE companies ADD COLUMN IF NOT EXISTS trade_name TEXT;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS phone TEXT;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS email TEXT;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS cnae_primary TEXT;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS cnae_codes JSONB DEFAULT '[]'::jsonb;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS capital_social TEXT;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS partners JSONB DEFAULT '[]'::jsonb;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS cnpj_verified BOOLEAN DEFAULT FALSE;

-- Additional social links
ALTER TABLE companies ADD COLUMN IF NOT EXISTS instagram_url TEXT;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS facebook_url TEXT;

-- Add comments for documentation
COMMENT ON COLUMN companies.trade_name IS 'Nome fantasia from CNPJ registry';
COMMENT ON COLUMN companies.phone IS 'Corporate phone from CNPJ registry';
COMMENT ON COLUMN companies.email IS 'Corporate email from CNPJ registry';
COMMENT ON COLUMN companies.cnae_primary IS 'Primary CNAE code and description';
COMMENT ON COLUMN companies.cnae_codes IS 'All CNAE codes (primary + secondary)';
COMMENT ON COLUMN companies.capital_social IS 'Registered capital (e.g., R$ 1.000,00)';
COMMENT ON COLUMN companies.partners IS 'Partners with roles from CNPJ registry';
COMMENT ON COLUMN companies.cnpj_verified IS 'True if data fetched from official CNPJ registry';
COMMENT ON COLUMN companies.instagram_url IS 'Instagram profile URL';
COMMENT ON COLUMN companies.facebook_url IS 'Facebook page URL';
