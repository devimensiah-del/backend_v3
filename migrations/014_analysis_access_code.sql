-- Migration 014: Add access_code columns for shareable public links
-- This enables public sharing of analysis reports via unique codes

-- Add access code columns
ALTER TABLE analyses ADD COLUMN access_code VARCHAR(12);
ALTER TABLE analyses ADD COLUMN access_code_created_at TIMESTAMP WITH TIME ZONE;

-- Partial unique index: only enforce uniqueness on non-null values
-- This allows multiple NULL values while ensuring active codes are unique
CREATE UNIQUE INDEX idx_analyses_access_code ON analyses(access_code) WHERE access_code IS NOT NULL;
