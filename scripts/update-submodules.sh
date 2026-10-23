#!/usr/bin/env bash
set -euo pipefail

echo "============================================="
echo "   NENAV Submodule Synchronizer"
echo "============================================="

# Fetch latest commits from configured tracking branches
echo "[+] Fetching remote updates for all submodules..."
git submodule update --remote --merge

echo ""
echo "[+] Submodule status after update:"
git submodule status

echo ""
echo "[i] Reminder: To pin new versions in NENAV, commit the updated submodule references:"
echo "    git add repositories/ .gitmodules"
echo "    git commit -m 'chore: bump submodule references'"
