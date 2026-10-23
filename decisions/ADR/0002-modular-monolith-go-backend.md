# ADR 0002: Modular Monolith Go Backend vs. Premature Microservices

## Status
Accepted

## Context
A common architectural failure mode in early-stage startups is adopting microservices prematurely before domain boundaries stabilize. This introduces network latency, distributed transaction complexity (SAGA patterns), dual-write failures, and severe infrastructure overhead.

## Decision
The authoritative Core Backend for NENAV will be engineered as a **Modular Monolith in Go** using Chi router:
- Core domains (Identity, Service Catalog, Bookings, Payments, Notifications) will reside in distinct Go packages with explicit interfaces.
- Inter-module communication will be in-process memory calls rather than HTTP/gRPC network hops.
- All modules share a single transactional PostgreSQL database, enabling ACID transactions for bookings and payments.
- If a specific module (e.g., geosearch or notification dispatch) experiences high independent load later, its cleanly isolated package can be extracted into an independent microservice with minimal refactoring.
