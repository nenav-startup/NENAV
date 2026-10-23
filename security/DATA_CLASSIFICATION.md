# NENAV Security — Data Classification & Handling Matrix

| Tier | Classification | Examples | Storage & Encryption Requirements | Retention Policy |
|---|---|---|---|---|
| **Tier 1: Restricted (Confidential)** | Secrets, private keys, payment card data (tokenized), founder equity docs | Cloud KMS keys, payment gateway tokens, investor term sheets | AES-256 at rest, TLS 1.3 in transit. Strictly forbidden in Git repositories. | Retained per compliance / statutory rules. |
| **Tier 2: Confidential (PII)** | Customer & pro personally identifiable information | Phone numbers, residential addresses, real-time GPS coordinates, government KYC IDs | Encrypted at rest (pgcrypto / KMS). Access logged in audit trails. | Retain until user account deletion request (Right to Erasure / DPDP Act). |
| **Tier 3: Internal** | Core marketplace operational data | Booking status history, dispute tickets, aggregated ratings, service catalog | Standard database encryption. Internal authorized access only. | 5 years for financial & taxation records. |
| **Tier 4: Public** | Public discovery data | Service names, category hierarchies, public business profiles, aggregated star ratings | Publicly cacheable via CDN. No authentication required. | Indefinite. |
