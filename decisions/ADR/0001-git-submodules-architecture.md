# ADR 0001: Git Submodules Architecture for Multi-Repository Coordination

## Status
Accepted

## Context
NENAV requires strict operational separation among six functional business domains:
1. `NENAV` (Ecosystem Coordination & Architecture)
2. `NENAV-ADMIN` (Administrative Controls & KYC Governance)
3. `NENAV-USER` (Customer Mobile & Web Applications)
4. `NENAV-PROF` (Service Partner Mobile & Web Applications)
5. `NENAV-SUPPORT` (Customer & Pro Support Operations)
6. `NENAV-PRODUCT-MANAGEMENT` (Confidential Strategy, Financials & Operations)

Each repository requires independent Git commit histories, branch protection rules, dedicated deployment lifecycles, and role-based developer permissions (e.g., contract mobile developers must not have read access to financial strategy or administrative source code).

Simultaneously, a unified local engineering workflow is required to allow principal engineers and CI/CD pipelines to checkout the ecosystem cohesively.

## Decision
We establish six independent GitHub repositories under `@nenav-startup`. The five functional repositories are registered inside the root `NENAV` repository as Git submodules under `repositories/`.

## Consequences
- **Positive:** Full permission isolation; clean git histories; atomic release tracking via pinned submodule commit SHAs.
- **Negative:** Requires developers to understand `git submodule update` workflows.
- **Mitigation:** Automate submodule synchronization using `./scripts/setup.sh` and `./scripts/update-submodules.sh`.
