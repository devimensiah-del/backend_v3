-- Preview: Show what will be merged by v2_028_merge_cnpj_duplicates.sql
-- Run this BEFORE the migration to verify the merge plan
-- NOTE: Uses company_core (actual table), not companies (view)

WITH ranked_companies AS (
    SELECT
        id,
        name,
        cnpj,
        cnpj_normalized,
        created_at,
        enrichment_status,
        ROW_NUMBER() OVER (
            PARTITION BY cnpj_normalized
            ORDER BY created_at DESC  -- Most recent first
        ) as rn,
        COUNT(*) OVER (PARTITION BY cnpj_normalized) as duplicate_count
    FROM company_core
    WHERE cnpj_normalized IS NOT NULL
      AND cnpj_normalized != ''
      AND (deleted_at IS NULL)
),
merge_plan AS (
    SELECT
        id as company_id,
        name as company_name,
        cnpj,
        cnpj_normalized,
        created_at,
        enrichment_status,
        rn,
        duplicate_count,
        CASE WHEN rn = 1 THEN 'KEEP (newest)' ELSE 'MERGE INTO NEWEST' END as action,
        FIRST_VALUE(id) OVER (
            PARTITION BY cnpj_normalized
            ORDER BY created_at DESC
        ) as target_company_id,
        FIRST_VALUE(name) OVER (
            PARTITION BY cnpj_normalized
            ORDER BY created_at DESC
        ) as target_company_name
    FROM ranked_companies
    WHERE duplicate_count > 1
)
SELECT
    cnpj_normalized,
    cnpj,
    company_id,
    company_name,
    created_at,
    enrichment_status,
    action,
    CASE
        WHEN action = 'KEEP (newest)' THEN NULL
        ELSE target_company_name
    END as will_merge_into,
    (SELECT COUNT(*) FROM company_submissions cs WHERE cs.company_id = merge_plan.company_id) as submission_count,
    (SELECT COUNT(*) FROM challenges ch WHERE ch.company_id = merge_plan.company_id) as challenge_count,
    (SELECT COUNT(*) FROM analyses a WHERE a.company_id = merge_plan.company_id) as analysis_count
FROM merge_plan
ORDER BY cnpj_normalized, created_at DESC;

-- Summary
SELECT
    'Summary' as report,
    COUNT(DISTINCT cnpj_normalized) as duplicate_cnpj_count,
    COUNT(*) FILTER (WHERE rn > 1) as companies_to_be_merged,
    COUNT(*) FILTER (WHERE rn = 1) as companies_to_be_kept
FROM (
    SELECT
        cnpj_normalized,
        ROW_NUMBER() OVER (PARTITION BY cnpj_normalized ORDER BY created_at DESC) as rn
    FROM company_core
    WHERE cnpj_normalized IS NOT NULL
      AND cnpj_normalized != ''
      AND deleted_at IS NULL
) ranked
WHERE cnpj_normalized IN (
    SELECT cnpj_normalized
    FROM company_core
    WHERE cnpj_normalized IS NOT NULL AND cnpj_normalized != '' AND deleted_at IS NULL
    GROUP BY cnpj_normalized
    HAVING COUNT(*) > 1
);
