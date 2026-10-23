-- ==========================================
-- NENAV Migration 001: Foundation Schema
-- Core tables for authentication and audit
-- ==========================================

-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";

-- ==========================================
-- Users Table
-- ==========================================
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  phone VARCHAR(15) NOT NULL,
  phone_country_code VARCHAR(5) NOT NULL DEFAULT '+91',
  email VARCHAR(255),
  display_name VARCHAR(100),
  role VARCHAR(20) NOT NULL DEFAULT 'user'
    CHECK (role IN ('user', 'prof', 'business', 'admin', 'support')),
  status VARCHAR(20) NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'active', 'suspended', 'deactivated')),
  locale VARCHAR(10) NOT NULL DEFAULT 'en-IN',
  timezone VARCHAR(50) NOT NULL DEFAULT 'Asia/Kolkata',
  country_code VARCHAR(3) NOT NULL DEFAULT 'IN',
  currency_code VARCHAR(3) NOT NULL DEFAULT 'INR',
  metadata JSONB DEFAULT '{}',
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Unique phone per country code
  CONSTRAINT uq_users_phone UNIQUE (phone_country_code, phone)
);

-- ==========================================
-- OTP Table (short-lived, for authentication)
-- ==========================================
CREATE TABLE otp_requests (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  phone VARCHAR(15) NOT NULL,
  phone_country_code VARCHAR(5) NOT NULL DEFAULT '+91',
  otp_hash VARCHAR(255) NOT NULL,
  purpose VARCHAR(20) NOT NULL DEFAULT 'login'
    CHECK (purpose IN ('login', 'verify', 'reset', 'change_phone')),
  attempts INT NOT NULL DEFAULT 0,
  max_attempts INT NOT NULL DEFAULT 3,
  verified BOOLEAN NOT NULL DEFAULT FALSE,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================
-- Sessions Table
-- ==========================================
CREATE TABLE sessions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash VARCHAR(255) NOT NULL,
  device_info JSONB DEFAULT '{}',
  ip_address INET,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================
-- Audit Log Table (append-only)
-- ==========================================
CREATE TABLE audit_log (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  actor_id UUID,
  actor_role VARCHAR(20),
  action VARCHAR(100) NOT NULL,
  entity_type VARCHAR(50),
  entity_id UUID,
  metadata JSONB DEFAULT '{}',
  ip_address INET,
  request_id VARCHAR(100),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================
-- Indexes
-- ==========================================
CREATE INDEX idx_users_phone ON users(phone_country_code, phone);
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_status ON users(status);

CREATE INDEX idx_otp_phone ON otp_requests(phone_country_code, phone);
CREATE INDEX idx_otp_expires ON otp_requests(expires_at);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

CREATE INDEX idx_audit_actor ON audit_log(actor_id);
CREATE INDEX idx_audit_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_action ON audit_log(action);
CREATE INDEX idx_audit_created ON audit_log(created_at);

-- ==========================================
-- Updated_at Trigger Function
-- ==========================================
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated_at
  BEFORE UPDATE ON users
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at();
