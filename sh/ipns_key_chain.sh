#!/usr/bin/env bash
set -euo pipefail

if [[ ! -f ../src/teeserver/.env ]]; then
  echo "Error: ../src/teeserver/.env not found"
  exit 1
fi

command -v ego-go >/dev/null 2>&1 || { echo "Error: ego-go not found in PATH"; exit 1; }
command -v ego >/dev/null 2>&1 || { echo "Error: ego not found in PATH"; exit 1; }

cd ../src/teeserver

echo "[1/3] Building enclave binary..."
ego-go build main.go

echo "[2/3] Signing enclave binary..."
ego sign main

echo "[3/3] Running enclave (blockchain mode)..."
exec ego run main -blockchain