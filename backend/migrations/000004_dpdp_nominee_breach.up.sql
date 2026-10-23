-- ==========================================
-- NENAV Migration 004: DPDP Nominee & Breach Ledger Schema
-- Digital Personal Data Protection Act (2023) Compliance
-- 1. Section 14: Data Principal Nominee Designations
-- 2. Section 8(6): Personal Data Breach Notification Ledger
-- ==========================================

-- ==========================================
-- 1. Nominees (Section 14)
-- ==========================================
CREATE TABLE IF NOT EXISTS nominees (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  nominee_name VARCHAR(100) NOT NULL,
  relationship VARCHAR(50) NOT NULL,
  phone VARCHAR(15) NOT NULL,
  email VARCHAR(255),
  status VARCHAR(20) NOT NULL DEFAULT 'active'
    CHECK (status IN ('active', 'revoked')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT uq_user_nominee UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS idx_nominees_user ON nominees(user_id);
CREATE INDEX IF NOT EXISTS idx_nominees_status ON nominees(status);

-- ==========================================
-- 2. Personal Data Breaches (Section 8(6))
-- ==========================================
CREATE TABLE IF NOT EXISTS data_breaches (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  incident_reference VARCHAR(50) NOT NULL UNIQUE,
  nature_of_breach TEXT NOT NULL,
  affected_count INT NOT NULL DEFAULT 0,
  severity VARCHAR(20) NOT NULL DEFAULT 'medium'
    CHECK (severity IN ('low', 'medium', 'high', 'critical')),
  mitigation_taken TEXT NOT NULL,
  reported_to_dpbi_at TIMESTAMPTZ,
  notified_principals_at TIMESTAMPTZ,
  status VARCHAR(20) NOT NULL DEFAULT 'investigating'
    CHECK (status IN ('investigating', 'contained', 'reported', 'closed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_breaches_ref ON data_breaches(incident_reference);
CREATE INDEX IF NOT EXISTS idx_breaches_status ON data_breaches(status);
CREATE INDEX IF NOT EXISTS idx_breaches_created ON data_breaches(created_at);

-- ==========================================
-- Triggers
-- ==========================================
CREATE TRIGGER trg_nominees_updated_at
  BEFORE UPDATE ON nominees
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_data_breaches_updated_at
  BEFORE UPDATE ON data_breaches
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at();
