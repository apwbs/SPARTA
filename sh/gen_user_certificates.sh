#!/usr/bin/env bash
set -euo pipefail

# Run from repo root (even if launched inside sh/)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

GO_FILE="src/certauth/ca.go"

usage() {
  echo "Usage: $0 --pub <public_key_pem> --attributes <attrs> [--blockchain_address <addr>]"
  echo "Example:"
  echo "  $0 --pub src/client/pubkey/publicKey.pem --attributes \"role=client\" --blockchain_address 0x1234..."
  exit 1
}

PUB_KEY=""
ATTRS=""
BC_ADDR=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --pub|-p)
      [[ $# -ge 2 ]] || usage
      PUB_KEY="$2"; shift 2;;
    --attributes|-a)
      [[ $# -ge 2 ]] || usage
      ATTRS="$2"; shift 2;;
    --blockchain_address|-b)
      [[ $# -ge 2 ]] || usage
      BC_ADDR="$2"; shift 2;;
    -h|--help)
      usage;;
    *)
      echo "Unknown option: $1"
      usage;;
  esac
done

[[ -n "$PUB_KEY" ]] || { echo "Error: --pub is required."; usage; }
[[ -n "$ATTRS" ]] || { echo "Error: --attributes is required."; usage; }
[[ -f "$PUB_KEY" ]] || { echo "Error: public key file not found: $PUB_KEY"; exit 1; }
[[ -f "$GO_FILE" ]] || { echo "Error: Go file not found: $GO_FILE"; exit 1; }

# Build command
cmd=(go run "$GO_FILE" -gen_client_cert -certificate "$PUB_KEY" -attributes "$ATTRS")

# Optional blockchain address
if [[ -n "$BC_ADDR" ]]; then
  cmd+=(-blockchain_address "$BC_ADDR")
fi

exec "${cmd[@]}"