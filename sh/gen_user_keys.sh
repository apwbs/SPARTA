#!/usr/bin/env bash
set -euo pipefail

# Always run from repo root (even if launched inside sh/)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

PUB_OUT=""
PRIV_OUT=""

usage() {
  echo "Usage: $0 --pub <public_key_path> --priv <private_key_path>"
  exit 1
}

# Parse args
while [[ $# -gt 0 ]]; do
  case "$1" in
    --pub|-p)
      [[ $# -ge 2 ]] || usage
      PUB_OUT="$2"; shift 2;;
    --priv|-k)
      [[ $# -ge 2 ]] || usage
      PRIV_OUT="$2"; shift 2;;
    -h|--help)
      usage;;
    *)
      echo "Unknown option: $1"
      usage;;
  esac
done

# Require both outputs
[[ -n "$PUB_OUT" ]] || { echo "Error: --pub is required."; usage; }
[[ -n "$PRIV_OUT" ]] || { echo "Error: --priv is required."; usage; }

# Ensure output dirs exist
mkdir -p "$(dirname "$PUB_OUT")"
mkdir -p "$(dirname "$PRIV_OUT")"

GO_FILE="src/userkeygen/gen_user_keys.go"
if [[ ! -f "$GO_FILE" ]]; then
  echo "Error: Go file not found: $GO_FILE"
  exit 1
fi

echo "Generating keys..."
echo "Public key output:  $PUB_OUT"
echo "Private key output: $PRIV_OUT"

exec go run "$GO_FILE" \
  -output_path_file "$PUB_OUT" \
  -output_path_file_private_key "$PRIV_OUT"