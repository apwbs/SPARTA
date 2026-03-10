#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 --ccu ccu1|ccu2 --sender_role ccu1|ccu2"
  echo
  echo "Examples:"
  echo "  $0 --ccu ccu1 --sender_role ccu1"
  echo "  $0 --ccu ccu2 --sender_role ccu1"
  echo
  echo "Notes:"
  echo "  - Run from SPARTA/sh"
  echo "  - Use the same --sender_role on both terminals"
  echo "  - Assumes both CCUs were already prepared with prepare_ccu.sh"
  exit 1
}

CCU=""
SENDER_ROLE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --ccu)
      [[ $# -ge 2 ]] || usage
      CCU="$2"
      shift 2
      ;;
    --sender_role)
      [[ $# -ge 2 ]] || usage
      SENDER_ROLE="$2"
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
[[ -n "$SENDER_ROLE" ]] || { echo "Error: --sender_role is required"; usage; }

case "$CCU" in
  ccu1)
    BASE_DIR="../src/teeserver"
    PEER_MAIN="../tee/main"
    ;;
  ccu2)
    BASE_DIR="../src/tee"
    PEER_MAIN="../teeserver/main"
    ;;
  *)
    echo "Error: --ccu must be ccu1 or ccu2"
    exit 1
    ;;
esac

case "$SENDER_ROLE" in
  ccu1|ccu2)
    ;;
  *)
    echo "Error: --sender_role must be ccu1 or ccu2"
    exit 1
    ;;
esac

command -v ego >/dev/null 2>&1 || { echo "Error: ego not found in PATH"; exit 1; }

cd "$BASE_DIR"

if [[ ! -f main ]]; then
  echo "Error: main not found in $BASE_DIR"
  echo "Run prepare_ccu.sh first."
  exit 1
fi

if [[ ! -f "$PEER_MAIN" ]]; then
  echo "Error: peer binary not found at $PEER_MAIN"
  echo "Make sure the other CCU has also been prepared."
  exit 1
fi

echo "Computing peer measurement from $PEER_MAIN..."
PEER_MEASUREMENT="$(ego uniqueid "$PEER_MAIN")"

echo "Starting exchange_seed on $CCU with sender_role=$SENDER_ROLE..."
exec ego run main -measurement "$PEER_MEASUREMENT" -exchange_seed -sender_role "$SENDER_ROLE"