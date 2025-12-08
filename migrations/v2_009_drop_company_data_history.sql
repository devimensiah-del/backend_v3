-- Migration: v2_011_drop_company_data_history
-- Description: Drop unused company_data_history table (code that wrote to it was removed)

DROP TABLE IF EXISTS company_data_history CASCADE;
