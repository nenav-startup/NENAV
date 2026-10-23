-- ==========================================
-- NENAV Local Development Seed Data
-- Hyderabad Region Mock Seed Data (§1, §8, §18)
-- ==========================================

-- Clean existing mock seeds
DELETE FROM audit_log WHERE action LIKE 'MOCK_%';
DELETE FROM sessions WHERE device_info->>'client' = 'nenav_mock';
DELETE FROM users WHERE phone IN ('9876543210', '9848012345', '9700067890', '9988776655', '9618098765');

-- 1. Insert Consumer User (Vimal Nayak)
INSERT INTO users (
  id, phone, phone_country_code, email, display_name, role, status, locale, timezone, metadata
) VALUES (
  'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
  '9876543210',
  '+91',
  'vimal.nayak@example.com',
  'Vimal Nayak',
  'user',
  'active',
  'en-IN',
  'Asia/Kolkata',
  '{"kyc_verified": true, "city": "Hyderabad", "default_cluster": "Kondapur", "wallet_balance": 2450}'
);

-- 2. Insert Verified PROFs (Workers)
INSERT INTO users (
  id, phone, phone_country_code, email, display_name, role, status, locale, timezone, metadata
) VALUES 
-- Ravi Kumar (Plumbing & Masonry)
(
  'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
  '9848012345',
  '+91',
  'ravi.kumar@prof.nenav.in',
  'Ravi Kumar',
  'prof',
  'active',
  'te-IN',
  'Asia/Kolkata',
  '{"primary_skill": "Plumbing", "rating": 4.9, "completed_jobs": 214, "aadhaar_verified": true, "vehicle": "TS 08 AB 1234", "cluster": "Kondapur"}'
),
-- Mahesh G (Electrical Specialist)
(
  'c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33',
  '9848012346',
  '+91',
  'mahesh.g@prof.nenav.in',
  'Mahesh G',
  'prof',
  'active',
  'te-IN',
  'Asia/Kolkata',
  '{"primary_skill": "Electrical", "rating": 4.9, "completed_jobs": 230, "aadhaar_verified": true, "vehicle": "TS 07 CD 5678", "cluster": "Madhapur"}'
),
-- Suresh M (Finishing Painter)
(
  'd3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44',
  '9700067890',
  '+91',
  'suresh.m@prof.nenav.in',
  'Suresh M',
  'prof',
  'active',
  'hi-IN',
  'Asia/Kolkata',
  '{"primary_skill": "Painting", "rating": 4.9, "completed_jobs": 160, "aadhaar_verified": true, "vehicle": "TS 09 EF 9012", "cluster": "Kondapur"}'
),
-- Kistaiah Chary (Master Carpenter)
(
  'e4eebc99-9c0b-4ef8-bb6d-6bb9bd380a55',
  '9988776655',
  '+91',
  'kistaiah.c@prof.nenav.in',
  'Kistaiah Chary',
  'prof',
  'active',
  'te-IN',
  'Asia/Kolkata',
  '{"primary_skill": "Carpentry", "rating": 4.9, "completed_jobs": 310, "aadhaar_verified": true, "vehicle": "TS 10 GH 3456", "cluster": "HITEC City"}'
),
-- Anand R (Deep Cleaning Specialist)
(
  'f5eebc99-9c0b-4ef8-bb6d-6bb9bd380a66',
  '9618098765',
  '+91',
  'anand.r@prof.nenav.in',
  'Anand R',
  'prof',
  'active',
  'te-IN',
  'Asia/Kolkata',
  '{"primary_skill": "Deep Cleaning", "rating": 4.8, "completed_jobs": 128, "aadhaar_verified": true, "vehicle": "TS 08 CD 4321", "cluster": "Kukatpally"}'
);

-- 3. Insert Active Mock Session for Consumer User
INSERT INTO sessions (
  id, user_id, token_hash, device_info, ip_address, expires_at
) VALUES (
  '11111111-2222-3333-4444-555555555555',
  'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
  'mock_session_token_hash_vimal_nayak_2026',
  '{"client": "nenav_mock", "platform": "iOS", "os_version": "18.0", "app_version": "1.0.0"}',
  '127.0.0.1',
  NOW() + INTERVAL '30 days'
);

-- 4. Audit Log Verification
INSERT INTO audit_log (
  actor_id, actor_role, action, entity_type, entity_id, metadata
) VALUES (
  'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
  'user',
  'MOCK_SEED_INITIALIZED',
  'system',
  'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
  '{"seed_version": "1.0.0", "locality": "Hyderabad", "records_seeded": 6}'
);
