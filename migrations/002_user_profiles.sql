-- ==================== USER PROFILES ====================
-- Extensions of the Supabase auth.users table
-- This handles the "Admin vs User" logic required by your PRD

-- Create an Enum for roles to ensure strict typing
CREATE TYPE user_role AS ENUM ('user', 'admin', 'super_admin');

CREATE TABLE IF NOT EXISTS user_profiles (
    id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    
    -- Identity
    email VARCHAR(255) NOT NULL, -- Copied from auth.users for easier querying
    full_name VARCHAR(255),
    avatar_url TEXT,
    
    -- Authorization & Access
    role user_role NOT NULL DEFAULT 'user',
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Business Context (Optional, useful if one user manages multiple submissions)
    organization_name VARCHAR(255),
    job_title VARCHAR(255),

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indices
CREATE INDEX idx_profiles_role ON user_profiles(role);
CREATE INDEX idx_profiles_email ON user_profiles(email);

-- Trigger for updated_at
CREATE TRIGGER update_profiles_updated_at BEFORE UPDATE ON user_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ==================== FOREIGN KEY FIX ====================
-- Now that user_profiles exists, we link submissions to it.
-- This ensures if a user is deleted, their submissions are handled correctly.

ALTER TABLE submissions 
ADD CONSTRAINT fk_submissions_user 
FOREIGN KEY (user_id) REFERENCES user_profiles(id);

-- Function to handle new user signup
CREATE OR REPLACE FUNCTION public.handle_new_user() 
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO public.user_profiles (id, email, full_name, role)
  VALUES (
    NEW.id, 
    NEW.email, 
    NEW.raw_user_meta_data->>'full_name', -- Grabs name from Google/Auth metadata
    'user' -- Default role is always 'user'
  );
  RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- The Trigger
CREATE OR REPLACE TRIGGER on_auth_user_created
  AFTER INSERT ON auth.users
  FOR EACH ROW EXECUTE FUNCTION public.handle_new_user();