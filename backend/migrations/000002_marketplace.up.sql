-- ==========================================
-- NENAV Migration 002: Marketplace Schema
-- Comprehensive tables for bookings, profs,
-- payments, matching, disputes, and workpass.
-- ==========================================

-- ==========================================
-- 1. Service Categories & Trades
-- ==========================================
CREATE TABLE IF NOT EXISTS categories (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  code VARCHAR(50) NOT NULL UNIQUE,
  name_en VARCHAR(100) NOT NULL,
  name_te VARCHAR(100) NOT NULL,
  name_hi VARCHAR(100) NOT NULL,
  icon VARCHAR(100) NOT NULL DEFAULT 'wrench',
  sort_order INT NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS trades (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
  code VARCHAR(50) NOT NULL UNIQUE,
  name_en VARCHAR(100) NOT NULL,
  name_te VARCHAR(100) NOT NULL,
  name_hi VARCHAR(100) NOT NULL,
  base_hourly_rate INT NOT NULL DEFAULT 199,    -- in INR
  base_half_day_rate INT NOT NULL DEFAULT 699,  -- 4 hours
  base_full_day_rate INT NOT NULL DEFAULT 1299, -- 8 hours
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================
-- 2. Customer Saved Addresses
-- ==========================================
CREATE TABLE IF NOT EXISTS addresses (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  label VARCHAR(50) NOT NULL DEFAULT 'home', -- 'home', 'work', 'site', 'other'
  address_line1 TEXT NOT NULL,
  address_line2 TEXT,
  landmark VARCHAR(255),
  city VARCHAR(100) NOT NULL DEFAULT 'Hyderabad',
  state VARCHAR(100) NOT NULL DEFAULT 'Telangana',
  pincode VARCHAR(10) NOT NULL,
  latitude NUMERIC(10,7) NOT NULL,
  longitude NUMERIC(10,7) NOT NULL,
  location GEOMETRY(Point, 4326),
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================
-- 3. Professional Profiles (PROFs)
-- ==========================================
CREATE TABLE IF NOT EXISTS prof_profiles (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
  trade_id UUID REFERENCES trades(id) ON DELETE SET NULL,
  kyc_status VARCHAR(20) NOT NULL DEFAULT 'pending'
    CHECK (kyc_status IN ('pending', 'verified', 'rejected')),
  kyc_document_type VARCHAR(50),
  kyc_document_number VARCHAR(100),
  experience_years INT NOT NULL DEFAULT 1,
  rating NUMERIC(3,2) NOT NULL DEFAULT 5.00,
  total_reviews INT NOT NULL DEFAULT 0,
  total_jobs INT NOT NULL DEFAULT 0,
  is_available BOOLEAN NOT NULL DEFAULT TRUE,
  current_latitude NUMERIC(10,7) DEFAULT 17.385044,
  current_longitude NUMERIC(10,7) DEFAULT 78.486671,
  location GEOMETRY(Point, 4326),
  badge_police_verified BOOLEAN NOT NULL DEFAULT FALSE,
  badge_skill_certified BOOLEAN NOT NULL DEFAULT FALSE,
  badge_nenav_pro BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================
-- 4. Bookings Table (Authoritative State)
-- ==========================================
CREATE TABLE IF NOT EXISTS bookings (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  booking_reference VARCHAR(30) NOT NULL UNIQUE,
  customer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  trade_id UUID REFERENCES trades(id),
  assigned_prof_id UUID REFERENCES prof_profiles(id) ON DELETE SET NULL,
  status VARCHAR(30) NOT NULL DEFAULT 'draft'
    CHECK (status IN (
      'draft',
      'pending_payment',
      'searching',
      'assigned',
      'en_route',
      'arrived',
      'in_progress',
      'completed',
      'cancelled',
      'disputed'
    )),
  duration_type VARCHAR(20) NOT NULL DEFAULT 'hourly'
    CHECK (duration_type IN ('hourly', 'half_day', 'full_day', 'multi_day', 'recurring')),
  scheduled_date DATE NOT NULL,
  time_slot VARCHAR(50) NOT NULL,
  hours INT NOT NULL DEFAULT 2,
  worker_count INT NOT NULL DEFAULT 1,
  address_id UUID REFERENCES addresses(id) ON DELETE SET NULL,
  job_description TEXT,
  checkin_otp VARCHAR(6) NOT NULL DEFAULT '4829',
  quote_breakdown JSONB NOT NULL DEFAULT '{}',
  payment_status VARCHAR(20) NOT NULL DEFAULT 'pending'
    CHECK (payment_status IN ('pending', 'escrowed', 'completed', 'refunded', 'failed')),
  payment_method VARCHAR(20) NOT NULL DEFAULT 'upi'
    CHECK (payment_method IN ('upi', 'card', 'netbanking', 'cash')),
  customer_confirmed_completion BOOLEAN NOT NULL DEFAULT FALSE,
  prof_confirmed_completion BOOLEAN NOT NULL DEFAULT FALSE,
  cancelled_at TIMESTAMPTZ,
  cancellation_reason TEXT,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================
-- 5. Booking Assignment Broadcasts
-- ==========================================
CREATE TABLE IF NOT EXISTS booking_assignments (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
  prof_id UUID NOT NULL REFERENCES prof_profiles(id) ON DELETE CASCADE,
  status VARCHAR(20) NOT NULL DEFAULT 'offered'
    CHECK (status IN ('offered', 'accepted', 'rejected', 'timeout', 'cancelled')),
  offered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  responded_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================
-- 6. Job Lifecycle Audit Events
-- ==========================================
CREATE TABLE IF NOT EXISTS job_events (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
  actor_id UUID,
  event_type VARCHAR(50) NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================
-- 7. Payments & Invoices
-- ==========================================
CREATE TABLE IF NOT EXISTS payments (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  amount INT NOT NULL, -- in INR
  currency VARCHAR(3) NOT NULL DEFAULT 'INR',
  method VARCHAR(20) NOT NULL DEFAULT 'upi',
  status VARCHAR(20) NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'escrowed', 'captured', 'failed', 'refunded')),
  idempotency_key VARCHAR(100) UNIQUE,
  gateway_order_id VARCHAR(100),
  gateway_payment_id VARCHAR(100),
  cash_customer_confirmed BOOLEAN NOT NULL DEFAULT FALSE,
  cash_prof_confirmed BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================
-- 8. Refunds Table
-- ==========================================
CREATE TABLE IF NOT EXISTS refunds (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
  booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
  amount INT NOT NULL,
  reason TEXT NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'initiated'
    CHECK (status IN ('initiated', 'processed', 'failed')),
  gateway_refund_id VARCHAR(100),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================
-- 9. Disputes & Support Cases
-- ==========================================
CREATE TABLE IF NOT EXISTS disputes (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
  raised_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason VARCHAR(50) NOT NULL
    CHECK (reason IN ('late_arrival', 'no_show', 'conduct', 'damage', 'incomplete', 'skill_mismatch', 'payment_dispute')),
  description TEXT NOT NULL,
  evidence_urls JSONB NOT NULL DEFAULT '[]',
  status VARCHAR(20) NOT NULL DEFAULT 'open'
    CHECK (status IN ('open', 'under_review', 'resolved', 'dismissed')),
  resolution_notes TEXT,
  resolved_by UUID REFERENCES users(id) ON DELETE SET NULL,
  resolved_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================
-- 10. Bilateral Ratings
-- ==========================================
CREATE TABLE IF NOT EXISTS ratings (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
  from_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  to_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role VARCHAR(20) NOT NULL CHECK (role IN ('customer', 'prof')),
  score INT NOT NULL CHECK (score >= 1 AND score <= 5),
  tags JSONB NOT NULL DEFAULT '[]',
  comment TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================
-- 11. WorkPass (Recurring Subscriptions)
-- ==========================================
CREATE TABLE IF NOT EXISTS workpasses (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  prof_id UUID REFERENCES prof_profiles(id) ON DELETE SET NULL,
  trade_id UUID REFERENCES trades(id) ON DELETE SET NULL,
  plan_type VARCHAR(20) NOT NULL CHECK (plan_type IN ('weekly', 'monthly')),
  schedule_matrix JSONB NOT NULL DEFAULT '{}',
  status VARCHAR(20) NOT NULL DEFAULT 'active'
    CHECK (status IN ('active', 'paused', 'expired', 'cancelled')),
  starts_at DATE NOT NULL,
  expires_at DATE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================
-- Indexes & Triggers
-- ==========================================
CREATE INDEX IF NOT EXISTS idx_prof_profiles_trade ON prof_profiles(trade_id);
CREATE INDEX IF NOT EXISTS idx_prof_profiles_status ON prof_profiles(kyc_status);
CREATE INDEX IF NOT EXISTS idx_prof_profiles_avail ON prof_profiles(is_available);
CREATE INDEX IF NOT EXISTS idx_prof_profiles_location ON prof_profiles USING GIST (location);

CREATE INDEX IF NOT EXISTS idx_addresses_user ON addresses(user_id);
CREATE INDEX IF NOT EXISTS idx_addresses_location ON addresses USING GIST (location);

CREATE INDEX IF NOT EXISTS idx_bookings_customer ON bookings(customer_id);
CREATE INDEX IF NOT EXISTS idx_bookings_prof ON bookings(assigned_prof_id);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);
CREATE INDEX IF NOT EXISTS idx_bookings_ref ON bookings(booking_reference);

CREATE INDEX IF NOT EXISTS idx_payments_booking ON payments(booking_id);
CREATE INDEX IF NOT EXISTS idx_payments_idempotency ON payments(idempotency_key);
CREATE INDEX IF NOT EXISTS idx_disputes_booking ON disputes(booking_id);
CREATE INDEX IF NOT EXISTS idx_ratings_booking ON ratings(booking_id);

CREATE TRIGGER trg_prof_profiles_updated_at
  BEFORE UPDATE ON prof_profiles
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_bookings_updated_at
  BEFORE UPDATE ON bookings
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_payments_updated_at
  BEFORE UPDATE ON payments
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at();
