# NENAV Engineering — Release Policy & Submodule Coordination

## 1. Versioning Standard
All applications and contracts follow [Semantic Versioning 2.0.0](https://semver.org/):
`MAJOR.MINOR.PATCH`
- `MAJOR`: Breaking changes (e.g. backward-incompatible API changes requiring client migration).
- `MINOR`: Backward-compatible new features.
- `PATCH`: Backward-compatible bug fixes.

---

## 2. Release Compatibility Rules
- **Mobile Client Backward Compatibility:** Mobile app store updates can take days or weeks for end users to install. **The Core Backend MUST support the previous N-2 mobile versions at all times.**
- **Submodule Pinning:** The root `NENAV` repository pins tested, compatible commit SHAs for each submodule. Submodules are never configured to track floating branches for production releases.
- **Rollback Procedure:** If a production defect occurs:
  1. Revert the submodule commit reference in `NENAV` to the last known stable SHA.
  2. Trigger automated rollback deployment pipelines.
