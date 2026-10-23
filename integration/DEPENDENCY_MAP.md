# NENAV Ecosystem — Dependency & Integration Map

This matrix documents the service boundaries, data dependencies, and cross-repository integration interfaces across the ecosystem.

```mermaid
graph LR
    UserApp["NENAV-USER<br/>(Customer Client)"] -->|Consumes OpenAPI / Auth| CoreAPI["Core Backend API<br/>(Go + Chi)"]
    ProfApp["NENAV-PROF<br/>(Professional Client)"] -->|Consumes OpenAPI / Auth| CoreAPI
    AdminWeb["NENAV-ADMIN<br/>(Governance Web)"] -->|Admin Ops & Audit| CoreAPI
    SupportWeb["NENAV-SUPPORT<br/>(Support Portal)"] -->|Queries Bookings & Users| CoreAPI
    
    CoreAPI --> Postgres[("PostgreSQL 16 + PostGIS")]
    CoreAPI --> Redis[("Redis 7")]
    CoreAPI --> SMS["SMS / OTP Provider"]
    CoreAPI --> Payments["Stripe / Razorpay Payment Gateway"]
    CoreAPI --> Maps["Google Maps Platform (Places / Routing)"]

    MainRepo["NENAV Main Orchestrator"] -.->|Governs Specs| UserApp
    MainRepo -.->|Governs Specs| ProfApp
    MainRepo -.->|Governs Specs| AdminWeb
    MainRepo -.->|Governs Specs| SupportWeb
```

---

## Service Contracts & Dependency Invariants
1. **Unidirectional Dependency:** Client applications (`NENAV-USER`, `NENAV-PROF`, `NENAV-ADMIN`, `NENAV-SUPPORT`) depend on the Core API contracts governed by `NENAV`. The Core API never depends on specific client implementations.
2. **Third-Party Gateways:**
   - **Payments:** Handled exclusively via server-side webhook reconciliation (Stripe / Razorpay). Clients never transmit raw card numbers or payment credentials.
   - **Geocoding & Navigation:** Customer app searches locations via Google Maps Places API; server computes route matching and radius filtering via PostGIS.
