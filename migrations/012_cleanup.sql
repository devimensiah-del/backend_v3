-- 1. CLEAN UP SUBMISSIONS
-- The submission status was trying to track the whole lifecycle. Let's simplify.
ALTER TABLE public.submissions 
DROP CONSTRAINT IF EXISTS submissions_status_check;

ALTER TABLE public.submissions
ADD CONSTRAINT submissions_status_check 
CHECK (status IN ('received', 'archived'));

-- Set everything to received for now to fix legacy data
UPDATE public.submissions SET status = 'received';


-- 2. REFACTOR ENRICHMENTS (The big change)
-- We are removing the specific columns and relying on a master 'data' jsonb column
ALTER TABLE public.enrichments
DROP COLUMN IF EXISTS company_info,
DROP COLUMN IF EXISTS market_data,
DROP COLUMN IF EXISTS competitors,
DROP COLUMN IF EXISTS financial_data,
DROP COLUMN IF EXISTS news,
DROP COLUMN IF EXISTS social_media,
DROP COLUMN IF EXISTS technologies,
DROP COLUMN IF EXISTS key_people,
DROP COLUMN IF EXISTS funding,
DROP COLUMN IF EXISTS partnerships,
DROP COLUMN IF EXISTS products,
DROP COLUMN IF EXISTS enriched_data; -- Dropping the old one to recreate clean

ALTER TABLE public.enrichments
ADD COLUMN data jsonb DEFAULT '{}'::jsonb; -- This holds EVERYTHING from Layer 1, 2, and 3

-- Fix the Status Enum to match your flow EXACTLY
ALTER TABLE public.enrichments 
DROP CONSTRAINT IF EXISTS enrichments_status_check;

ALTER TABLE public.enrichments
ADD CONSTRAINT enrichments_status_check 
CHECK (status IN ('pending', 'processing', 'finished', 'approved', 'failed'));


-- 3. REFACTOR ANALYSES (Versioning logic)
-- Ensure we have the versioning columns and the correct status flow
ALTER TABLE public.analyses 
DROP CONSTRAINT IF EXISTS analyses_status_check;

ALTER TABLE public.analyses
ADD CONSTRAINT analyses_status_check 
CHECK (status IN ('pending', 'processing', 'completed', 'approved', 'sent', 'failed'));

-- Ensure versioning columns are present (based on your schema they were there, but let's be safe)
-- If you need to "Lock" a previous version, we use `is_latest` boolean.