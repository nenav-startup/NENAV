-- ==========================================
-- NENAV Migration 003: DPDP & Privacy Engineering Schema
-- Digital Personal Data Protection Act (2023) Compliance
-- 1. Consent Ledger (Immutable append-only history)
-- 2. Data Subject Rights (DSR - Export, Erasure 72h SLA, Correction)
-- 3. Grievance Redressal (DPO & DPBI Escalation Tracking)
-- 4. Privacy Access Logs (Sensitive data access tracking)
-- ==========================================

-- ==========================================
-- 1. Consents Ledger (Immutable Audit Trail)
-- ==========================================
CREATE TABLE IF NOT EXISTS consents (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  purpose VARCHAR(50) NOT NULL
    CHECK (purpose IN ('analytics', 'marketing', 'precise_location', 'service_delivery', 'ai_processing')),
  status VARCHAR(20) NOT NULL
    CHECK (status IN ('granted', 'revoked')),
  notice_version VARCHAR(20) NOT NULL DEFAULT 'v1.0.0',
  ip_address INET,
  user_agent TEXT,
  metadata JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index to quickly resolve the latest state for any user + purpose
CREATE INDEX IF NOT EXISTS idx_consents_user_purpose ON consents(user_id, purpose, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_consents_created_at ON consents(created_at);

-- ==========================================
-- 2. Data Subject Requests (DSR)
-- ==========================================
CREATE TABLE IF NOT EXISTS data_subject_requests (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  request_type VARCHAR(30) NOT NULL
    CHECK (request_type IN ('access_export', 'correction', 'erasure', 'nominee', 'grievance')),
  status VARCHAR(20) NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'processing', 'completed', 'rejected')),
  details JSONB NOT NULL DEFAULT '{}',
  sla_deadline TIMESTAMPTZ NOT NULL, -- e.g., 72 hours for erasure under DPDP regulations
  completed_at TIMESTAMPTZ,
  rejection_reason TEXT,
  export_file_url TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dsr_user ON data_subject_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_dsr_status_sla ON data_subject_requests(status, sla_deadline);
CREATE INDEX IF NOT EXISTS idx_dsr_created_at ON data_subject_requests(created_at);

-- ==========================================
-- 3. Grievances (DPO Redressal Mechanism)
-- ==========================================
CREATE TABLE IF NOT EXISTS grievances (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  ticket_number VARCHAR(30) NOT NULL UNIQUE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  subject VARCHAR(200) NOT NULL,
  description TEXT NOT NULL,
  dpo_name VARCHAR(100) NOT NULL DEFAULT 'NENAV Data Protection Officer',
  status VARCHAR(30) NOT NULL DEFAULT 'open'
    CHECK (status IN ('open', 'under_investigation', 'resolved', 'escalated_to_dpbi')),
  resolution_notes TEXT,
  resolved_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_grievances_user ON grievances(user_id);
CREATE INDEX IF NOT EXISTS idx_grievances_ticket ON grievances(ticket_number);
CREATE INDEX IF NOT EXISTS idx_grievances_status ON grievances(status);

-- ==========================================
-- 4. Privacy Access Logs (Data Access Auditing)
-- ==========================================
CREATE TABLE IF NOT EXISTS privacy_access_logs (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  accessor_id UUID,
  accessor_role VARCHAR(20),
  target_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  accessed_fields JSONB NOT NULL DEFAULT '[]',
  reason TEXT NOT NULL,
  ip_address INET,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_privacy_access_target ON privacy_access_logs(target_user_id);
CREATE INDEX IF NOT EXISTS idx_privacy_access_created ON privacy_access_logs(created_at);

-- ==========================================
-- Triggers
-- ==========================================
CREATE TRIGGER trg_dsr_updated_at
  BEFORE UPDATE ON data_subject_requests
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_grievances_updated_at
  BEFORE UPDATE ON grievances
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at();
