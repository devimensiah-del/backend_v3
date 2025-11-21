-- ==================== 1. AI OVERWRITE PROTECTION ====================
-- Add a flag to enrichments so the Go Scraper knows not to overwrite user edits
ALTER TABLE enrichments 
ADD COLUMN is_locked BOOLEAN DEFAULT FALSE;

-- ==================== 2. USER UPDATE POLICIES ====================
-- Allow Users to EDIT their own Enrichment data
CREATE POLICY "Users update own enrichments" ON enrichments
FOR UPDATE 
USING (
    EXISTS (
        SELECT 1 FROM submissions 
        WHERE submissions.id = enrichments.submission_id 
        AND submissions.user_id = auth.uid()
    )
);

-- Allow Users to EDIT their own Reports (if you want them to tweak the HTML)
CREATE POLICY "Users update own reports" ON reports
FOR UPDATE 
USING (
    EXISTS (
        SELECT 1 FROM submissions 
        WHERE submissions.id = reports.submission_id 
        AND submissions.user_id = auth.uid()
    )
);

-- ==================== 3. ADMIN ACCESS (Completing the set) ====================
-- You had a comment for these, here is the actual SQL

-- Admins full access to Analyses
CREATE POLICY "Admins full access analyses" ON analyses
FOR ALL 
USING (
    EXISTS (
        SELECT 1 FROM user_profiles 
        WHERE id = auth.uid() AND role IN ('admin', 'super_admin')
    )
);

-- Admins full access to Reports
CREATE POLICY "Admins full access reports" ON reports
FOR ALL 
USING (
    EXISTS (
        SELECT 1 FROM user_profiles 
        WHERE id = auth.uid() AND role IN ('admin', 'super_admin')
    )
);