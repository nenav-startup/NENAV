-- ==========================================
-- NENAV Migration 001: Foundation Schema (Rollback)
-- ==========================================

DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at();

DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS otp_requests;
DROP TABLE IF EXISTS users;

DROP EXTENSION IF EXISTS "postgis";
DROP EXTENSION IF EXISTS "uuid-ossp";
