# NENAV — System Architecture Specification

## 1. High-Level Architectural Model

NENAV employs a clean, layered architectural model separating edge client applications, API ingress & policy enforcement, core business logic, and transactional persistence.

```text
┌────────────────────────────────────────────────────────┐
│                   EDGE CLIENT APPLICATIONS             │
│                                                        │
│  NENAV-USER (iOS/Android/Web)   NENAV-PROF (iOS/Android/Web)│
│  NENAV-ADMIN (Admin Web)        NENAV-SUPPORT (Support Web) │
└───────────────────────────┬────────────────────────────┘
                            │ HTTPS / WSS / TLS 1.3
                            ▼
┌────────────────────────────────────────────────────────┐
│               INGRESS & API GATEWAY LAYER              │
│                                                        │
│  - TLS Termination & DDoS Protection (Cloudflare)     │
│  - JWT Signature & Expiration Verification            │
│  - Role-Based Access Control (RBAC) Enforcement        │
│  - Distributed Token Bucket Rate Limiting (Redis)     │
└───────────────────────────┬────────────────────────────┘
                            │ Authorized Request Pipeline
                            ▼
┌────────────────────────────────────────────────────────┐
│           AUTHORITATIVE CORE BACKEND (Go + Chi)        │
│                                                        │
│  ┌──────────────────┐  ┌──────────────────┐            │
│  │ Identity & Auth  │  │ Services Catalog │            │
│  └──────────────────┘  └──────────────────┘            │
│  ┌──────────────────┐  ┌──────────────────┐            │
│  │ Booking Engine   │  │ Payments & Escrow│            │
│  └──────────────────┘  └──────────────────┘            │
│  ┌──────────────────┐  ┌──────────────────┐            │
│  │ Notifications    │  │ Dispute Liaison  │            │
│  └──────────────────┘  └──────────────────┘            │
└─────────────┬──────────────────────────┬───────────────┘
              │ Relational Queries &     │ Transient Data &
              │ PostGIS Spatial Ops      │ Locks
              ▼                          ▼
┌──────────────────────────┐    ┌────────────────────────┐
│   PostgreSQL 16 + PostGIS│    │        Redis 7         │
│                          │    │                        │
│ - Users & Roles          │    │ - Session Tokens       │
│ - Geolocated Services    │    │ - Distributed Locks    │
│ - Booking State Machine  │    │ - OTP Temporary Store  │
│ - Financial Ledgers      │    │ - Rate Limit Counters  │
└──────────────────────────┘    └────────────────────────┘
```

---

## 2. Ingress & Traffic Management
All client requests terminate at the Ingress Gateway:
1. **SSL/TLS 1.3** is enforced with HSTS.
2. **Reverse Proxy & Routing:** Paths routed according to OpenAPI routing maps.
3. **CORS Policies:** Strict origin verification per client domain (e.g. `admin.nenav.com`, `app.nenav.com`).
4. **Security Headers:** Enforces Content Security Policy (CSP), X-Frame-Options (`DENY`), and X-Content-Type-Options (`nosniff`).

---

## 3. Asynchronous Events & Notifications
- Critical status transitions (e.g. `BOOKING_ACCEPTED`, `PROFESSIONAL_ARRIVED`) emit events to an internal worker pool.
- Background workers handle push notifications (Firebase Cloud Messaging / APNs), SMS alerts, and transactional emails asynchronously without blocking the client HTTP response loop.
