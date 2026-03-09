#!/bin/sh
set -eu

# Resolve repo root = parent of this script's folder (repo/sh/...)
SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$REPO_ROOT"

CA_FILE="./src/certauth/ca.go"
if [ ! -f "$CA_FILE" ]; then
  echo "[start_ca] ERROR: cannot find $CA_FILE (are you in the right repo?)" >&2
  exit 1
fi

echo "[start_ca] Running CA..."
exec go run "$CA_FILE" -start_ca "$@"