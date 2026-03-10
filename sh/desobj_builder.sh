#!/usr/bin/env bash
set -euo pipefail

# Assumes this script is run from the sh/ folder
cd ../src/alfaparser

usage() {
  echo "Usage: $0 --alfa_script <path_to_policy.alfa>"
  echo "Example:"
  echo "  $0 --alfa_script ../src/alfaparser/policy.alfa"
  exit 1
}

ALFA_SCRIPT_PATH=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --alfa_script)
      [[ $# -ge 2 ]] || usage
      ALFA_SCRIPT_PATH="$2"
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

if [[ -z "$ALFA_SCRIPT_PATH" ]]; then
  echo "Error: --alfa_script is required"
  usage
fi

# Resolve to absolute path before we run Go (because we cd)
if [[ "$ALFA_SCRIPT_PATH" != /* ]]; then
  ALFA_SCRIPT_PATH="$(cd ../../sh && pwd)/$ALFA_SCRIPT_PATH"
fi

if [[ ! -f "$ALFA_SCRIPT_PATH" ]]; then
  echo "Error: policy file not found: $ALFA_SCRIPT_PATH"
  exit 1
fi

exec go run alfa_parser.go -alfa_script "$ALFA_SCRIPT_PATH"