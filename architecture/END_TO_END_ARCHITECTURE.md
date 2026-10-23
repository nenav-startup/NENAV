# NENAV — End-to-End Architecture & Core Lifecycles

This document describes the end-to-end data flow for the primary user journey in NENAV: service discovery, booking dispatch, execution, payment settlement, dispute escalation, and admin audit.

---

## 1. End-to-End Booking Lifecycle

```mermaid
sequenceDiagram
    autonumber
    actor Customer as Customer (NENAV-USER)
    actor Pro as Professional (NENAV-PROF)
    participant Gateway as API Gateway
    participant Backend as Core Backend (Go)
    participant DB as Postgres + PostGIS
    actor Admin as Admin (NENAV-ADMIN)
    actor Support as Support (NENAV-SUPPORT)

    Note over Customer,DB: 1. Discovery & Search
    Customer->>Gateway: GET /services/nearby?lat=...&lng=...&radius=10
    Gateway->>Backend: Forward authorized query
    Backend->>DB: ST_DWithin(service_area, ST_MakePoint(lng, lat))
    DB-->>Backend: Matching active professionals
    Backend-->>Customer: 200 OK (Services & Ratings)

    Note over Customer,Pro: 2. Booking Creation & Dispatch
    Customer->>Gateway: POST /bookings (service_id, slot, coordinates)
    Gateway->>Backend: Validate & authenticate
    Backend->>DB: INSERT booking (status: REQUESTED)
    Backend-->>Pro: Push notification (New Job Request)
    Backend-->>Customer: 201 Created (Booking ID, Status: REQUESTED)

    Note over Pro,Customer: 3. Acceptance & Execution
    Pro->>Gateway: PATCH /bookings/{id}/status (status: CONFIRMED)
    Gateway->>Backend: Verify pro role & ownership
    Backend->>DB: UPDATE status=CONFIRMED
    Backend-->>Customer: Push notification (Booking Confirmed)
    
    Pro->>Gateway: PATCH /bookings/{id}/status (status: IN_PROGRESS)
    Pro->>Gateway: PATCH /bookings/{id}/status (status: COMPLETED)

    Note over Customer,Support: 4. Payment & Potential Dispute
    Customer->>Gateway: POST /payments/capture (booking_id)
    Gateway->>Backend: Charge customer card & place funds in escrow
    
    opt Dispute Raised
        Customer->>Gateway: POST /support/tickets (booking_id, "Incomplete job")
        Gateway->>Support: Create dispute ticket with read-only booking snapshot
        Support->>Backend: Review audit log & resolve
        Admin->>Backend: Approve partial refund if required
    end
```

---

## 2. Invariant Rules Across the Ecosystem

1. **No Client Trust:** Booking amounts, platform fees, and pro earnings are calculated server-side only.
2. **Deterministic State Transitions:** Bookings cannot transition backwards or skip mandatory checkpoints (e.g. `REQUESTED` $\rightarrow$ `CONFIRMED` $\rightarrow$ `IN_PROGRESS` $\rightarrow$ `COMPLETED`).
3. **Escrow Hold:** Customer funds are captured into escrow upon booking confirmation and released to the professional's payout balance only upon successful OTP/job completion or verified resolution.
