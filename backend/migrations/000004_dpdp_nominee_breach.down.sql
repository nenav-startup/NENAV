-- Drop triggers
DROP TRIGGER IF EXISTS trg_data_breaches_updated_at ON data_breaches;
DROP TRIGGER IF EXISTS trg_nominees_updated_at ON nominees;

-- Drop tables
DROP TABLE IF EXISTS data_breaches CASCADE;
DROP TABLE IF EXISTS nominees CASCADE;
