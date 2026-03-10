#!/usr/bin/env bash
set -euo pipefail

# Assumes this script is run from SPARTA/sh

usage() {
  echo "Usage: $0 --function <function_name> --ipnsKey <key>"
  echo
  echo "Example:"
  echo "  $0 --function PriorityRE --ipnsKey 2026"
  exit 1
}

FU=""
IPNS_KEY=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --function)
      [[ $# -ge 2 ]] || usage
      FU="$2"
      shift 2
      ;;
    --ipnsKey)
      [[ $# -ge 2 ]] || usage
      IPNS_KEY="$2"
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

[[ -n "$FU" ]] || usage
[[ -n "$IPNS_KEY" ]] || usage

command -v ego >/dev/null 2>&1 || { echo "Error: ego not found in PATH"; exit 1; }

cd ../src/client

SIGNER_ID="$(ego signerid client)"
MEASUREMENT="$(ego uniqueid ../teeserver/main)"
CERTIFICATE="../client/certificate/user_cert.pem"

exec ./client \
  -decision_function \
  -signer_id "$SIGNER_ID" \
  -measurement "$MEASUREMENT" \
  -certificate "$CERTIFICATE" \
  -function "$FU" \
  -ipnsKey "$IPNS_KEY"