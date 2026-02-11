-- Migration: v2_042_stale_tracking.sql
-- Description: Add stale tracking columns to company_framework_results
-- Enables dependency cascade invalidation and update suggestions

-- =============================================================================
-- 1. Add stale tracking columns
-- =============================================================================

-- Is this result stale due to upstream changes?
ALTER TABLE company_framework_results ADD COLUMN IF NOT EXISTS is_stale BOOLEAN DEFAULT FALSE;

-- Reason why it's stale (e.g., "MERCADO foi atualizado")
ALTER TABLE company_framework_results ADD COLUMN IF NOT EXISTS stale_reason TEXT;

-- When the stale status was detected
ALTER TABLE company_framework_results ADD COLUMN IF NOT EXISTS stale_detected_at TIMESTAMPTZ;

-- User acknowledged the stale warning but chose not to re-run
ALTER TABLE company_framework_results ADD COLUMN IF NOT EXISTS stale_acknowledged_at TIMESTAMPTZ;
ALTER TABLE company_framework_results ADD COLUMN IF NOT EXISTS stale_acknowledged_by UUID;

-- Previous result (for diff comparison when updating)
ALTER TABLE company_framework_results ADD COLUMN IF NOT EXISTS previous_result JSONB;
ALTER TABLE company_framework_results ADD COLUMN IF NOT EXISTS previous_version INT;

-- Upstream hash when this result was generated (for change detection)
-- Note: context_hash already exists, but we add upstream_data_hash for clarity
ALTER TABLE company_framework_results ADD COLUMN IF NOT EXISTS upstream_data_hash TEXT;

-- =============================================================================
-- 2. Add comments
-- =============================================================================

COMMENT ON COLUMN company_framework_results.is_stale IS 'True if upstream dependencies changed after this result was generated';
COMMENT ON COLUMN company_framework_results.stale_reason IS 'Human-readable reason why result is stale (e.g., "MERCADO foi atualizado")';
COMMENT ON COLUMN company_framework_results.stale_detected_at IS 'When the stale status was first detected';
COMMENT ON COLUMN company_framework_results.stale_acknowledged_at IS 'When user acknowledged stale warning but chose not to re-run';
COMMENT ON COLUMN company_framework_results.stale_acknowledged_by IS 'User who acknowledged the stale warning';
COMMENT ON COLUMN company_framework_results.previous_result IS 'Previous version result (for diff/comparison on updates)';
COMMENT ON COLUMN company_framework_results.previous_version IS 'Version number of the previous result';
COMMENT ON COLUMN company_framework_results.upstream_data_hash IS 'SHA256 hash of upstream dependency data used to generate this result';

-- =============================================================================
-- 3. Create indexes for stale queries
-- =============================================================================

-- Index for finding stale results
CREATE INDEX IF NOT EXISTS idx_cfr_stale
ON company_framework_results(company_id, is_stale)
WHERE is_stale = TRUE AND is_current = TRUE;

-- Index for results needing attention (stale but not acknowledged)
CREATE INDEX IF NOT EXISTS idx_cfr_stale_unacknowledged
ON company_framework_results(company_id)
WHERE is_stale = TRUE AND stale_acknowledged_at IS NULL AND is_current = TRUE;

-- =============================================================================
-- 4. Create function to mark downstream as stale
-- =============================================================================

CREATE OR REPLACE FUNCTION mark_downstream_stale(
    p_company_id UUID,
    p_changed_framework_code TEXT
)
RETURNS INTEGER AS $$
DECLARE
    v_changed_framework_id UUID;
    v_affected_count INTEGER := 0;
    v_downstream_id UUID;
    v_downstream_code TEXT;
BEGIN
    -- Get the framework ID
    SELECT id INTO v_changed_framework_id
    FROM frameworks
    WHERE code = p_changed_framework_code;

    IF v_changed_framework_id IS NULL THEN
        RAISE EXCEPTION 'Framework not found: %', p_changed_framework_code;
    END IF;

    -- Find all frameworks that depend on the changed framework (direct dependencies)
    -- and mark their results as stale
    FOR v_downstream_id, v_downstream_code IN
        SELECT f.id, f.code
        FROM framework_dependencies fd
        JOIN frameworks f ON f.id = fd.framework_id
        WHERE fd.depends_on_id = v_changed_framework_id
          AND f.is_active = TRUE
    LOOP
        -- Mark the downstream result as stale
        UPDATE company_framework_results
        SET
            is_stale = TRUE,
            stale_reason = p_changed_framework_code || ' foi atualizado',
            stale_detected_at = NOW(),
            stale_acknowledged_at = NULL,
            stale_acknowledged_by = NULL,
            updated_at = NOW()
        WHERE company_id = p_company_id
          AND framework_id = v_downstream_id
          AND is_current = TRUE
          AND status = 'completed';

        IF FOUND THEN
            v_affected_count := v_affected_count + 1;

            -- Recursively mark dependencies of this framework
            v_affected_count := v_affected_count + mark_downstream_stale(p_company_id, v_downstream_code);
        END IF;
    END LOOP;

    RETURN v_affected_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION mark_downstream_stale IS 'Recursively marks all downstream framework results as stale when an upstream framework is updated';

-- =============================================================================
-- 5. Create function to get stale frameworks for a company
-- =============================================================================

CREATE OR REPLACE FUNCTION get_stale_frameworks(p_company_id UUID)
RETURNS TABLE (
    framework_code TEXT,
    framework_name TEXT,
    stale_reason TEXT,
    stale_detected_at TIMESTAMPTZ,
    stale_acknowledged_at TIMESTAMPTZ,
    result_id UUID
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        f.code,
        f.name,
        cfr.stale_reason,
        cfr.stale_detected_at,
        cfr.stale_acknowledged_at,
        cfr.id
    FROM company_framework_results cfr
    JOIN frameworks f ON f.id = cfr.framework_id
    WHERE cfr.company_id = p_company_id
      AND cfr.is_stale = TRUE
      AND cfr.is_current = TRUE
    ORDER BY f.step_number, f.execution_order;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_stale_frameworks IS 'Returns all stale framework results for a company with framework details';

-- =============================================================================
-- 6. Create function to acknowledge stale status
-- =============================================================================

CREATE OR REPLACE FUNCTION acknowledge_stale(
    p_result_id UUID,
    p_user_id UUID
)
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE company_framework_results
    SET
        stale_acknowledged_at = NOW(),
        stale_acknowledged_by = p_user_id,
        updated_at = NOW()
    WHERE id = p_result_id
      AND is_stale = TRUE
      AND is_current = TRUE;

    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION acknowledge_stale IS 'Marks a stale result as acknowledged by the user';

-- =============================================================================
-- 7. Create function to clear stale status (after re-run)
-- =============================================================================

CREATE OR REPLACE FUNCTION clear_stale_status(p_result_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE company_framework_results
    SET
        is_stale = FALSE,
        stale_reason = NULL,
        stale_detected_at = NULL,
        stale_acknowledged_at = NULL,
        stale_acknowledged_by = NULL,
        updated_at = NOW()
    WHERE id = p_result_id;

    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION clear_stale_status IS 'Clears the stale status after a result is re-generated';

-- =============================================================================
-- 8. Verify migration
-- =============================================================================

DO $$
BEGIN
    -- Check columns exist
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'company_framework_results'
        AND column_name = 'is_stale'
    ) THEN
        RAISE NOTICE 'Migration v2_042 complete: stale tracking columns added';
    ELSE
        RAISE EXCEPTION 'Migration failed: is_stale column not found';
    END IF;
END $$;
