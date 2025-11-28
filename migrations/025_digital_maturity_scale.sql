-- Migration 025: Update digital maturity scale from 1-5 to 1-10
-- Purpose: Provide more granular digital maturity scoring

-- Remove old constraint if exists
ALTER TABLE companies DROP CONSTRAINT IF EXISTS chk_digital_maturity_range;

-- Add new constraint for 1-10 scale
ALTER TABLE companies ADD CONSTRAINT chk_digital_maturity_range
    CHECK (digital_maturity IS NULL OR (digital_maturity >= 1 AND digital_maturity <= 10));
