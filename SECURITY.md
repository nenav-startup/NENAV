# Security Policy — NENAV Ecosystem

Security and trust are fundamental to NENAV. This policy defines vulnerability reporting, data classification, and credential handling across all six repositories.

---

## 1. Reporting a Vulnerability

If you discover a security vulnerability in any NENAV repository, service, or infrastructure component:

- **DO NOT** create a public GitHub issue.
- Email our security team directly at **security@nenav.com** (or open a private GitHub Security Advisory).
- Include:
  - Affected repository and commit SHA / version.
  - Detailed steps to reproduce or proof-of-concept (PoC).
  - Assessment of potential impact.

We commit to acknowledging your disclosure within 24 hours and providing remediation timelines.

---

## 2. Secrets & Credential Quarantine

- **Zero Tolerance for Committed Secrets:** Never commit API keys, private keys, database passwords, JWT signing secrets, or payment credentials into any Git repository.
- **Gitguardian & Secret Scanning:** Automated pre-commit hooks and GitHub Secret Scanning are enabled. Any commit containing detected tokens will be rejected at push.
- **Remediation:** If a secret is accidentally committed, consider it compromised immediately:
  1. Revoke and rotate the credential immediately.
  2. Rewrite the repository history or purge the commit using `git-filter-repo`.
  3. File an incident report in `security/INCIDENT_RESPONSE.md`.

---

## 3. Principle of Least Privilege

- Access to `NENAV-PRODUCT-MANAGEMENT` is restricted to authorized founders and senior leadership.
- Production database credentials and Cloud KMS keys are inaccessible to regular developer roles and isolated within environment secret stores (e.g., AWS Secrets Manager, GCP Secret Manager, or HashiCorp Vault).
- Server-side role-based access control (RBAC) validates every request at the API Gateway.
