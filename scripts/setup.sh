#!/usr/bin/env bash
set -euo pipefail

echo "============================================="
echo "   NENAV Ecosystem Workspace Setup"
echo "============================================="

# Ensure git is installed
if ! command -v git &> /dev/null; then
    echo "[-] Error: git is not installed or not in PATH."
    exit 1
fi

echo "[+] Initializing and fetching Git submodules..."
git submodule update --init --recursive --progress

echo "[+] Verifying submodule status..."
git submodule status

echo "[+] Setting executable permissions for helper scripts..."
chmod +x scripts/*.sh

echo ""
echo "============================================="
echo "   NENAV Ecosystem Initialized Successfully!"
echo "============================================="
echo "Submodules available under: ./repositories/"
ls -ld repositories/*
