-- Drop triggers
DROP TRIGGER IF EXISTS trg_grievances_updated_at ON grievances;
DROP TRIGGER IF EXISTS trg_dsr_updated_at ON data_subject_requests;

-- Drop tables in reverse order
DROP TABLE IF EXISTS privacy_access_logs CASCADE;
DROP TABLE IF EXISTS grievances CASCADE;
DROP TABLE IF EXISTS data_subject_requests CASCADE;
DROP TABLE IF EXISTS consents CASCADE;
