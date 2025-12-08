-- =============================================================================
-- SEED ANALYSES - WITH FK CONSTRAINTS DISABLED
-- =============================================================================
-- This sets foreign key columns to NULL where data doesn't exist
-- Run AFTER v2_ROLLBACK_to_baseline.sql
-- =============================================================================

-- Option 1: Set FKs to NULL for missing references
-- This will insert the analyses but lose the relationships

-- First, let's see what the data looks like and insert with NULL FKs
-- You can update the FKs later once you have the related data

