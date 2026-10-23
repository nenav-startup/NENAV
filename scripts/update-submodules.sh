#!/usr/bin/env bash
set -euo pipefail

echo "============================================="
echo "   NENAV Submodule Synchronizer & Auditor"
echo "============================================="

MODE="check"
TARGET="all"

for arg in "$@"; do
    case "$arg" in
        --apply|--sync)
            MODE="apply"
            ;;
        --check|--dry-run)
            MODE="check"
            ;;
        *)
            TARGET="$arg"
            ;;
    esac
done

if [ "$MODE" = "check" ]; then
    echo "[i] Mode: READ-ONLY CHECK (Dry Run)"
    echo "    To apply updates and move references, re-run with: ./scripts/update-submodules.sh --apply"
    echo ""
    echo "[+] Current Pinned Submodule Commits:"
    git submodule status
    echo ""
    echo "[+] Checking for upstream updates on remote branches..."
    git submodule foreach --quiet '
        git fetch origin main --quiet
        LOCAL=$(git rev-parse HEAD)
        REMOTE=$(git rev-parse origin/main)
        if [ "$LOCAL" != "$REMOTE" ]; then
            echo "  [UPDATE AVAILABLE] $name: Pinned ($LOCAL) -> Upstream ($REMOTE)"
        else
            echo "  [UP TO DATE]       $name is at latest origin/main ($LOCAL)"
        fi
    '
    exit 0
fi

echo "[!] Mode: APPLYING UPDATES (Modifying Submodule Pointers)"
echo "[+] Updating submodules..."

if [ "$TARGET" = "all" ]; then
    git submodule update --remote --merge
else
    git submodule update --remote --merge "repositories/$TARGET"
fi

echo ""
echo "[+] Submodule status after update:"
git submodule status

echo ""
echo "[i] Reminder: To pin new versions in NENAV, review the diffs and commit the updated references:"
echo "    git diff repositories/"
echo "    git add repositories/ .gitmodules"
echo "    git commit -m 'chore(submodules): bump submodule references to latest reviewed commits'"
