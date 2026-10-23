#!/usr/bin/env bash
set -euo pipefail

echo "============================================="
echo "   NENAV Ecosystem Integrity Validator"
echo "============================================="

FAILED=0

check_file() {
    if [ -f "$1" ]; then
        echo "  [PASS] File exists: $1"
    else
        echo "  [FAIL] Missing file: $1"
        FAILED=1
    fi
}

check_dir() {
    if [ -d "$1" ]; then
        echo "  [PASS] Directory exists: $1"
    else
        echo "  [FAIL] Missing directory: $1"
        FAILED=1
    fi
}

echo "1. Checking Core Ecosystem Architecture Files..."
check_file "README.md"
check_file ".gitmodules"
check_file ".gitignore"
check_file "CONTRIBUTING.md"
check_file "SECURITY.md"
check_file "contracts/openapi/nenav-api.v1.yaml"
check_file "infrastructure/development/docker-compose.yml"

echo ""
echo "2. Checking Registered Submodule Repositories..."
for sub in NENAV-ADMIN NENAV-USER NENAV-PROF NENAV-SUPPORT NENAV-PRODUCT-MANAGEMENT; do
    check_dir "repositories/$sub"
    if [ -d "repositories/$sub/.git" ] || [ -f "repositories/$sub/.git" ]; then
        echo "  [PASS] Submodule gitlink valid: repositories/$sub"
    else
        echo "  [FAIL] Submodule gitlink invalid: repositories/$sub"
        FAILED=1
    fi
done

echo ""
if [ "$FAILED" -eq 0 ]; then
    echo ">>> All ecosystem integrity checks PASSED!"
    exit 0
else
    echo ">>> Ecosystem integrity validation FAILED."
    exit 1
fi
