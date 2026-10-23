# NENAV Engineering — Testing Standards

## 1. Testing Pyramid
We maintain balanced test automation across all layers:
- **Unit Tests (70%):** Fast, isolated tests for pure functions, state machines, and business logic.
- **Integration Tests (20%):** Database interactions (pgx / SQL queries), PostGIS spatial queries, and API route handlers.
- **Contract & End-to-End Tests (10%):** Cross-repository OpenAPI validation and critical user journeys.

---

## 2. Test Execution Requirements
- **Determinism:** Tests must be deterministic. Never use real timers or uncontrolled network calls; use test doubles, clock abstractions, or testcontainers.
- **Test Database Isolation:** Integration tests run against an ephemeral PostgreSQL + PostGIS container. Each test suite runs in an isolated transaction or truncated test schema.
- **Minimum Coverage:** Target $\ge 80\%$ test coverage on core business domains (Bookings state machine, Payment escrow calculations).
