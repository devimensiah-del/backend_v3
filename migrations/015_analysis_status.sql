-- 1. Reset any invalid statuses to 'pending'
UPDATE public.analyses
SET status = 'pending'
WHERE status NOT IN ('pending', 'completed', 'approved', 'sent');

-- 2. Drop the old constraint
ALTER TABLE public.analyses 
DROP CONSTRAINT IF EXISTS analyses_status_check;

-- 3. Add the strict new constraint
ALTER TABLE public.analyses
ADD CONSTRAINT analyses_status_check 
CHECK (status IN ('pending', 'completed', 'approved', 'sent'));

-- 4. Ensure default is correct
ALTER TABLE public.analyses
ALTER COLUMN status SET DEFAULT 'pending';