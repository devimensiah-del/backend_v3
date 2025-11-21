-- Enable RLS
ALTER TABLE user_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE submissions ENABLE ROW LEVEL SECURITY;

-- 1. Users can view their own profile
CREATE POLICY "Users can view own profile" 
ON user_profiles FOR SELECT 
USING (auth.uid() = id);

-- 2. Users can update their own profile
CREATE POLICY "Users can update own profile" 
ON user_profiles FOR UPDATE 
USING (auth.uid() = id);

-- 3. Admins can view ALL profiles
CREATE POLICY "Admins can view all profiles" 
ON user_profiles FOR SELECT 
USING (
  EXISTS (
    SELECT 1 FROM user_profiles 
    WHERE id = auth.uid() AND role IN ('admin', 'super_admin')
  )
);

-- 4. Users can only see their own Submissions
CREATE POLICY "Users see own submissions" 
ON submissions FOR SELECT 
USING (auth.uid() = user_id);

-- 5. Admins can see ALL Submissions
CREATE POLICY "Admins see all submissions" 
ON submissions FOR SELECT 
USING (
  EXISTS (
    SELECT 1 FROM user_profiles 
    WHERE id = auth.uid() AND role IN ('admin', 'super_admin')
  )
);

-- ==================== RLS FOR CHILD TABLES ====================

-- Enable RLS on all child tables
ALTER TABLE enrichments ENABLE ROW LEVEL SECURITY;
ALTER TABLE analyses ENABLE ROW LEVEL SECURITY;
ALTER TABLE reports ENABLE ROW LEVEL SECURITY;

-- Policy: Users can read Enrichments if they own the parent Submission
CREATE POLICY "Users read own enrichments" ON enrichments
FOR SELECT USING (
    EXISTS (
        SELECT 1 FROM submissions 
        WHERE submissions.id = enrichments.submission_id 
        AND submissions.user_id = auth.uid()
    )
);

-- Policy: Users can read Analyses if they own the parent Submission
CREATE POLICY "Users read own analyses" ON analyses
FOR SELECT USING (
    EXISTS (
        SELECT 1 FROM submissions 
        WHERE submissions.id = analyses.submission_id 
        AND submissions.user_id = auth.uid()
    )
);

-- Policy: Users can read Reports if they own the parent Submission
CREATE POLICY "Users read own reports" ON reports
FOR SELECT USING (
    EXISTS (
        SELECT 1 FROM submissions 
        WHERE submissions.id = reports.submission_id 
        AND submissions.user_id = auth.uid()
    )
);

-- Policy: Admins have full access to child tables
-- (Repeat similar logic for Admins or rely on Service Role in Go backend)
CREATE POLICY "Admins access all child data" ON enrichments
FOR ALL USING (
    EXISTS (
        SELECT 1 FROM user_profiles 
        WHERE id = auth.uid() AND role IN ('admin', 'super_admin')
    )
);
-- (Apply same Admin policy to analyses and reports)