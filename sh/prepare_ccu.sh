#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 --ccu ccu1|ccu2"
  echo
  echo "Examples:"
  echo "  $0 --ccu ccu1"
  echo "  $0 --ccu ccu2"
  exit 1
}

CCU=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --ccu)
      [[ $# -ge 2 ]] || usage
      CCU="$2"
      shift 2
      ;;
    -h|--help)
      usage
      ;;
    *)
      echo "Unknown option: $1"
      usage
      ;;
  esac
done

[[ -n "$CCU" ]] || { echo "Error: --ccu is required"; usage; }

case "$CCU" in
  ccu1)
    BASE_DIR="../src/teeserver"
    ;;
  ccu2)
    BASE_DIR="../src/tee"
    ;;
  *)
    echo "Error: --ccu must be ccu1 or ccu2"
    exit 1
    ;;
esac

command -v ego-go >/dev/null 2>&1 || { echo "Error: ego-go not found in PATH"; exit 1; }
command -v ego >/dev/null 2>&1 || { echo "Error: ego not found in PATH"; exit 1; }

cd "$BASE_DIR"

echo "[1/2] Building in $BASE_DIR..."
ego-go build main.go

echo "[2/2] Signing in $BASE_DIR..."
ego sign main

echo "Preparation completed for $CCU."