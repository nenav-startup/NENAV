# NENAV Security — Threat Model (STRIDE Methodology)

## 1. Overview
This threat model evaluates attack surfaces across the NENAV marketplace ecosystem using the **STRIDE** methodology (Spoofing, Tampering, Repudiation, Information Disclosure, Denial of Service, Elevation of Privilege).

---

## 2. Threat Analysis & Mitigations

| Category | Threat Scenario | Primary Target | Mitigation Strategy |
|---|---|---|---|
| **Spoofing** | Adversary pretends to be an authorized customer or pro | Edge API Gateway | Strong JWT with RS256/EdDSA signing, short-lived tokens (1h), and secure refresh tokens with rotation. |
| **Tampering** | Professional tampers with booking price or job completion payload | Core Backend API | Prices and payouts are strictly computed server-side. Job completion requires customer OTP verification. |
| **Repudiation** | Pro or customer denies performing a critical action | Booking & Financials | Append-only immutable audit log in PostgreSQL recording user ID, action, timestamp, and IP address. |
| **Information Disclosure** | BOLA / IDOR vulnerability exposing other users' bookings or addresses | API Endpoints | Strict server-side ownership checks: `SELECT * FROM bookings WHERE id = $1 AND customer_id = $2`. |
| **Denial of Service** | Botnet floods location search or OTP endpoints | Ingress Layer | Distributed Redis token bucket rate limiting (10 req/min for OTP, 100 req/min for search), Cloudflare WAF. |
| **Elevation of Privilege** | Support agent attempts to trigger unauthorized financial refund | Admin & Support API | Multi-eye principle: refunds exceeding ₹5,000 require dual-supervisor approval in `NENAV-ADMIN`. |
