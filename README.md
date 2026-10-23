# NENAV — Ecosystem Architecture & Core Registry

Welcome to the central coordination repository for **NENAV**. This repository acts as the single source of truth for the entire NENAV ecosystem architecture, shared API contracts, security governance, infrastructure definitions, and repository coordination.

---

## 1. Ecosystem Topology

NENAV operates as six independent Git repositories, with the five functional repositories integrated into this main repository via **Git submodules** under `repositories/`.

```text
NENAV ECOSYSTEM
│
├── NENAV/                              [MAIN COORDINATION REPOSITORY]
│   ├── architecture/                   [System, Data, Security Architecture]
│   ├── contracts/                      [OpenAPI Specs, Event Schemas, Error Codes]
│   ├── engineering/                    [Standards, Branching, Testing, Release Policies]
│   ├── infrastructure/                 [Dev, Staging, Prod Infra & Docker Compose]
│   ├── security/                       [Threat Models, Data Classification, Secret Management]
│   ├── decisions/ADR/                  [Architecture Decision Records]
│   ├── scripts/                        [Ecosystem Setup & Submodule Sync Tooling]
│   │
│   └── repositories/
│       ├── NENAV-ADMIN/                [Git Submodule: Admin & Governance Web]
│       ├── NENAV-USER/                 [Git Submodule: Customer iOS, Android & Web]
│       ├── NENAV-PROF/                 [Git Submodule: Professional iOS, Android & Web]
│       ├── NENAV-SUPPORT/              [Git Submodule: Customer & Pro Support Portal]
│       └── NENAV-PRODUCT-MANAGEMENT/   [Git Submodule: Confidential Strategy & Operations]
```

---

## 2. Repository Registry

| Repository | Scope & Purpose | Tech Stack | Remote URL |
|---|---|---|---|
| **NENAV** | Orchestration, Contracts & Architecture | Markdown, OpenAPI, Shell, Docker | [`nenav-startup/NENAV`](https://github.com/nenav-startup/NENAV) |
| **NENAV-ADMIN** | Administrative console, KYC & auditing | React / Next.js, Go API | [`nenav-startup/NENAV-ADMIN`](https://github.com/nenav-startup/NENAV-ADMIN) |
| **NENAV-USER** | Customer mobile apps & web client | Flutter (iOS/Android), React Web | [`nenav-startup/NENAV-USER`](https://github.com/nenav-startup/NENAV-USER) |
| **NENAV-PROF** | Service partner mobile apps & web client | Flutter (iOS/Android), React Web | [`nenav-startup/NENAV-PROF`](https://github.com/nenav-startup/NENAV-PROF) |
| **NENAV-SUPPORT** | Support desk, ticketing & disputes | React, Support API client | [`nenav-startup/NENAV-SUPPORT`](https://github.com/nenav-startup/NENAV-SUPPORT) |
| **NENAV-PRODUCT-MANAGEMENT** | Strategy, Financials, Business Docs | Markdown, Specs (Private) | [`nenav-startup/NENAV-PRODUCT-MANAGEMENT`](https://github.com/nenav-startup/NENAV-PRODUCT-MANAGEMENT) |

---

## 3. Developer Onboarding & Submodule Setup

### Cloning the Full Ecosystem
To clone the entire ecosystem including all submodules in one command:
```bash
git clone --recurse-submodules https://github.com/nenav-startup/NENAV.git
```

### Initializing in an Existing Clone
If you already cloned `NENAV` without submodules:
```bash
./scripts/setup.sh
# or manually:
git submodule update --init --recursive
```

### Synchronizing Submodule Updates
To pull and verify latest submodule commits:
```bash
./scripts/update-submodules.sh
```

---

## 4. Architectural Invariants

1. **Independent Repositories:** Each repository retains its own Git history, pull request workflows, CI pipelines, and version tags.
2. **Authoritative Backend:** Core business logic (Identity, Booking State Machine, PostGIS geospatial matching, Payments, Notifications) is managed as a high-performance **Modular Monolith in Go** using Chi and PostgreSQL.
3. **Contract-First Integration:** All cross-repository API communication conforms strictly to versioned OpenAPI specifications in `contracts/openapi/`.
4. **Least-Privilege Security:** Role-based access control (RBAC) is enforced at the server-side API Gateway. Frontend permission flags are for UX only.

---

## 5. Directory Structure Navigation

- [`architecture/`](architecture/): Deep-dive architectural blueprints.
- [`contracts/`](contracts/): Shared OpenAPI specifications, JSON schemas, and error taxonomy.
- [`engineering/`](engineering/): Engineering standards, Git commit conventions, and release policies.
- [`infrastructure/`](infrastructure/): Local development Docker Compose stacks and deployment configurations.
- [`security/`](security/): Threat models, data classification matrices, and secret handling policies.
- [`decisions/ADR/`](decisions/ADR/): Architecture Decision Records documenting ecosystem decisions.
