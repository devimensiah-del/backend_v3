-- 1. Reset any invalid statuses to 'pending' so the constraint doesn't fail
UPDATE public.enrichments
SET status = 'pending'
WHERE status NOT IN ('pending', 'finished', 'approved');

-- 2. Drop the old constraint
ALTER TABLE public.enrichments 
DROP CONSTRAINT IF EXISTS enrichments_status_check;

-- 3. Add the strict new constraint
ALTER TABLE public.enrichments
ADD CONSTRAINT enrichments_status_check 
CHECK (status IN ('pending', 'finished', 'approved'));

-- 4. Ensure default is correct
ALTER TABLE public.enrichments
ALTER COLUMN status SET DEFAULT 'pending';