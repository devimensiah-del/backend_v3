-- Migration 028: Move PDF storage from reports table to analyses
-- Purpose: Simplify architecture - analysis IS the report, PDF is just the document

-- Step 1: Add PDF columns to analyses
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS pdf_url TEXT;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS pdf_generated_at TIMESTAMPTZ;
