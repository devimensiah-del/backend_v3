-- Migration 006: Add indexes and constraints for reports table
-- Purpose: Improve query performance and enforce one-to-one submission-report relationship
-- Author: Claude Code
-- Date: 2025-11-24

-- Add unique constraint on submission_id to enforce one-to-one relationship
-- This enables the ON CONFLICT clause in the Upsert method
ALTER TABLE reports
ADD CONSTRAINT reports_submission_id_unique UNIQUE (submission_id);

-- Add index on submission_id for faster lookups (if constraint doesn't create it automatically)
-- Some databases create indexes automatically for unique constraints, but we'll be explicit
CREATE INDEX IF NOT EXISTS idx_reports_submission_id ON reports(submission_id);

-- Add index on analysis_id for potential foreign key lookups
CREATE INDEX IF NOT EXISTS idx_reports_analysis_id ON reports(analysis_id);

-- Add index on status for filtering reports by status
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status);

-- Add index on pdf_generation_status for tracking PDF generation progress
CREATE INDEX IF NOT EXISTS idx_reports_pdf_generation_status ON reports(pdf_generation_status);

-- Add composite index for common query patterns (submission + status)
CREATE INDEX IF NOT EXISTS idx_reports_submission_status ON reports(submission_id, status);

-- Add index on created_at for chronological sorting
CREATE INDEX IF NOT EXISTS idx_reports_created_at ON reports(created_at DESC);

-- Comment explaining the indexes
COMMENT ON INDEX idx_reports_submission_id IS 'Performance: Fast lookups by submission_id in GetBySubmissionID()';
COMMENT ON INDEX idx_reports_analysis_id IS 'Performance: Fast lookups by analysis_id for potential joins';
COMMENT ON INDEX idx_reports_status IS 'Performance: Filter reports by status (pending, completed, failed)';
COMMENT ON INDEX idx_reports_pdf_generation_status IS 'Performance: Track PDF generation progress';
COMMENT ON INDEX idx_reports_submission_status IS 'Performance: Composite index for submission + status queries';
COMMENT ON INDEX idx_reports_created_at IS 'Performance: Chronological sorting in List() method';
COMMENT ON CONSTRAINT reports_submission_id_unique ON reports IS 'Integrity: Enforce one-to-one relationship between submission and report';
