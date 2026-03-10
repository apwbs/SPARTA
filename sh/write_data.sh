#!/usr/bin/env bash
set -euo pipefail

# Assumes this script is run from SPARTA/sh

usage() {
  echo "Usage: $0 --function <function_name> --input_file <path> --ipnsKey <key>"
  echo
  echo "Example:"
  echo "  $0 --function WritePatientData --input_file ../inputfiles/patient.json --ipnsKey Patient"
  exit 1
}

FU=""
INPUT_FILE=""
IPNS_KEY=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --function)
      [[ $# -ge 2 ]] || usage
      FU="$2"
      shift 2
      ;;
    --input_file)
      [[ $# -ge 2 ]] || usage
      INPUT_FILE="$2"
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
[[ -n "$INPUT_FILE" ]] || usage
[[ -n "$IPNS_KEY" ]] || usage

command -v ego >/dev/null 2>&1 || { echo "Error: ego not found in PATH"; exit 1; }

cd ../src/client

SIGNER_ID="$(ego signerid client)"
MEASUREMENT="$(ego uniqueid ../teeserver/main)"
CERTIFICATE="../client/certificate/user_cert.pem"

exec ./client \
  -set_function \
  -signer_id "$SIGNER_ID" \
  -measurement "$MEASUREMENT" \
  -certificate "$CERTIFICATE" \
  -function "$FU" \
  -input_file "$INPUT_FILE" \
  -ipnsKey "$IPNS_KEY"