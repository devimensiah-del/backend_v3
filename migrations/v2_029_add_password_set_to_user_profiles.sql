-- Migration: v2_029_add_password_set_to_user_profiles
-- Description: Add password_set column to track users who need to set their password
-- Used for: Auto-created users via landing page submission (temporary 1-hour sessions)

-- Add password_set column
-- Default TRUE because existing users already have passwords (normal signup)
-- New users created via submission will have FALSE
ALTER TABLE user_profiles
ADD COLUMN IF NOT EXISTS password_set BOOLEAN NOT NULL DEFAULT TRUE;

-- Add comment for documentation
COMMENT ON COLUMN user_profiles.password_set IS
  'FALSE for users auto-created via submission who need to set password on return. TRUE for users who signed up normally or have set their password.';

-- Index for quick lookup of users needing password setup (optional, for admin queries)
CREATE INDEX IF NOT EXISTS idx_user_profiles_password_not_set
ON user_profiles (email)
WHERE password_set = FALSE;
