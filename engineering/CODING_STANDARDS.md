# NENAV Engineering — Coding Standards

## 1. General Principles
- **Readability & Simplicity:** Code should be self-documenting. Prefer explicit logic over clever abstractions.
- **Fail Fast & Explicitly:** Handle errors immediately at the boundary. Do not swallow errors or return silent failures.
- **Immutability:** Favor immutable data structures and pure functions where practical.

---

## 2. Go (Backend Services)
- Follow official [Effective Go](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).
- **Linter:** Use `golangci-lint` with standard presets (`govet`, `staticcheck`, `errcheck`, `gosec`).
- **Error Handling:** Wrap errors with context (`fmt.Errorf("service_name.operation: %w", err)`). Never use `panic()` for control flow.
- **Context Propagation:** The first argument of any I/O bound function must be `ctx context.Context`.
- **Database Access:** Use prepared statements and parameterized queries. Never concatenate strings into SQL queries.

---

## 3. Flutter / Dart (Mobile Applications)
- Follow [Effective Dart](https://dart.dev/effective-dart) style guide.
- **State Management:** Use a clean, reactive state management pattern (Bloc / Riverpod) with strict separation between UI widgets and business logic.
- **Design System:** Use central theme tokens (`ThemeData`, typography, color schemes) rather than hardcoded hex colors or padding values.
- **Null Safety:** Strict sound null safety is required across all packages.

---

## 4. TypeScript / React (Web Applications)
- Strict mode enabled (`"strict": true` in `tsconfig.json`).
- Avoid `any`. Use strict interfaces or zod-validated schemas.
- **Components:** Functional components with typed props only. Avoid monolithic components; break UI into focused, reusable elements.
- **Accessibility:** Semantic HTML elements with appropriate ARIA roles and keyboard navigability.
