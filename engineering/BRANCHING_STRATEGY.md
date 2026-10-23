# NENAV Engineering — Branching Strategy

## 1. Branching Model: Trunk-Based Development
NENAV adheres to **Trunk-Based Development** across all six repositories. All developers merge frequently into the primary `main` branch through reviewed Pull Requests.

```text
main (protected, always deployable)
  │
  ├── feature/booking-flow-v1 ──> PR + CI check ──> Squash & Merge
  ├── fix/token-refresh-race   ──> PR + CI check ──> Squash & Merge
  └── release/v1.2.0           ──> Release candidate verification & tag
```

---

## 2. Branch Lifecycle
1. **Branch Longevity:** Feature branches should ideally live less than 48 hours. Break large epics into smaller, independently testable slices.
2. **Branch Protection Rules on `main`:**
   - Require status checks to pass before merging (CI tests, linter, OpenAPI validation).
   - Require at least 1 peer code review approval.
   - Require conversation resolution.
   - Restrict push/merge to authorized maintainers (no direct pushes to `main`).
   - Require linear history (Squash and Merge or Rebase).
