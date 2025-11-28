-- Migration: Company Owner Field
-- Purpose: Add owner_id to companies for user management permissions
-- Only the owner (or admin) can add/remove users from allowed_users

-- ============================================
-- 1. Add owner_id column to companies table
-- ============================================
ALTER TABLE companies ADD COLUMN IF NOT EXISTS owner_id UUID;

-- Foreign key to users table (Supabase auth.users)
-- Note: Using uuid type without FK since auth.users is in auth schema
-- If using public users table, add: REFERENCES users(id) ON DELETE SET NULL

-- ============================================
-- 2. Create index for efficient owner lookups
-- ============================================
CREATE INDEX IF NOT EXISTS idx_companies_owner ON companies (owner_id) WHERE owner_id IS NOT NULL;

-- ============================================
-- 3. Backfill owner_id from existing data
-- Priority: first user in allowed_users, then user_id from primary submission
-- ============================================
UPDATE companies c
SET owner_id = COALESCE(
    -- First, try the first user in allowed_users array
    c.allowed_users[1],
    -- Then, try the user_id from the primary submission
    (
        SELECT s.user_id
        FROM company_submissions cs
        JOIN submissions s ON s.id = cs.submission_id
        WHERE cs.company_id = c.id
          AND cs.is_primary = true
          AND s.user_id IS NOT NULL
        ORDER BY s.created_at ASC
        LIMIT 1
    )
)
WHERE c.owner_id IS NULL;

-- ============================================
-- 4. Add documentation
-- ============================================
COMMENT ON COLUMN companies.owner_id IS 'User who owns this company and can manage allowed_users. Admins bypass this check.';
