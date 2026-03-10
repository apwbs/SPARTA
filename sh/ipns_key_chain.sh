#!/usr/bin/env bash
set -euo pipefail

TARGET="${1:-}"

if [[ -z "$TARGET" ]]; then
  echo "Usage: $0 ccu1|ccu2"
  exit 1
fi

case "$TARGET" in
  ccu1)
    BASE_DIR="../src/teeserver"
    ;;
  ccu2)
    BASE_DIR="../src/tee"
    ;;
  *)
    echo "Error: invalid target '$TARGET'. Use ccu1 or ccu2."
    exit 1
    ;;
esac

if [[ ! -f "$BASE_DIR/.env" ]]; then
  echo "Error: $BASE_DIR/.env not found"
  exit 1
fi

command -v ego-go >/dev/null 2>&1 || { echo "Error: ego-go not found in PATH"; exit 1; }
command -v ego >/dev/null 2>&1 || { echo "Error: ego not found in PATH"; exit 1; }

cd "$BASE_DIR"

echo "[1/3] Building enclave binary in $BASE_DIR..."
ego-go build main.go

echo "[2/3] Signing enclave binary..."
ego sign main

echo "[3/3] Running enclave (blockchain mode)..."
exec ego run main -blockchain