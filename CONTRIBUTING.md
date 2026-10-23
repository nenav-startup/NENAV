# Contributing to the NENAV Ecosystem

Thank you for contributing to NENAV! This document outlines the standards, branch conventions, and workflow rules required to maintain our six-repository architecture.

---

## 1. Repository Boundary Rules

1. **Never commit application code directly to the `NENAV` root orchestrator repository.**  
   The `NENAV` root repository is strictly for ecosystem architecture, OpenAPI contracts, CI definitions, and submodule pins.
2. **Work directly in the appropriate submodule or independent repository clone:**
   - Admin changes $\rightarrow$ `NENAV-ADMIN`
   - Customer mobile/web changes $\rightarrow$ `NENAV-USER`
   - Professional partner changes $\rightarrow$ `NENAV-PROF`
   - Support desk changes $\rightarrow$ `NENAV-SUPPORT`
   - Business & product strategy $\rightarrow$ `NENAV-PRODUCT-MANAGEMENT`

---

## 2. Branching Strategy & Git Workflow

We follow **Trunk-Based Development** with short-lived feature branches:

```text
main (protected)
  │
  ├── feature/user-auth-flow  ──> PR (passes CI & 1 review) ──> squash merge
  │
  ├── fix/booking-state-race  ──> PR ──> squash merge
  │
  └── release/v1.0.0          ──> Tagged release
```

### Branch Naming Conventions
- `feature/<issue-or-shortname>` (e.g., `feature/location-picker`)
- `fix/<issue-or-shortname>` (e.g., `fix/token-expiry`)
- `chore/<shortname>` (e.g., `chore/bump-dependencies`)
- `docs/<shortname>` (e.g., `docs/update-threat-model`)

---

## 3. Commit Message Standards

We enforce [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(<scope>): <short description in present imperative tense>

[optional body]

[optional footer(s)]
```

### Allowed Types
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation-only changes
- `style`: Changes that do not affect the meaning of the code (formatting, white-space)
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance
- `test`: Adding missing tests or correcting existing tests
- `chore`: Build tasks, package updates, repository maintenance

**Example:**
```text
feat(booking): implement postgis proximity radius filtering
```

---

## 4. Contract-First Development

When introducing or modifying API endpoints:
1. Propose changes in `NENAV/contracts/openapi/nenav-api.v1.yaml` via pull request.
2. Review for backward compatibility with published mobile client versions.
3. Once merged into `NENAV`, generate or update client SDKs in the respective consumer repositories (`NENAV-USER`, `NENAV-PROF`, `NENAV-ADMIN`).
