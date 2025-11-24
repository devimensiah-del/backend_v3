-- Remove deprecated report columns that are no longer used by the backend or frontend.
-- Safe to re-run thanks to IF EXISTS.

BEGIN;

ALTER TABLE reports
    DROP COLUMN IF EXISTS pestel_page,
    DROP COLUMN IF EXISTS decision_matrix_page,
    DROP COLUMN IF EXISTS bcg_matrix_page,
    DROP COLUMN IF EXISTS value_proposition_page,
    DROP COLUMN IF EXISTS strategic_priorities_page,
    DROP COLUMN IF EXISTS risks_mitigation_page;

COMMIT;
