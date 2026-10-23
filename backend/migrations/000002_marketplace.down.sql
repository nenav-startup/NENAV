-- ==========================================
-- NENAV Migration 002: Marketplace Schema (Down)
-- ==========================================

DROP TABLE IF EXISTS workpasses CASCADE;
DROP TABLE IF EXISTS ratings CASCADE;
DROP TABLE IF EXISTS disputes CASCADE;
DROP TABLE IF EXISTS refunds CASCADE;
DROP TABLE IF EXISTS payments CASCADE;
DROP TABLE IF EXISTS job_events CASCADE;
DROP TABLE IF EXISTS booking_assignments CASCADE;
DROP TABLE IF EXISTS bookings CASCADE;
DROP TABLE IF EXISTS prof_profiles CASCADE;
DROP TABLE IF EXISTS addresses CASCADE;
DROP TABLE IF EXISTS trades CASCADE;
DROP TABLE IF EXISTS categories CASCADE;
