#!/usr/bin/env bash
set -euo pipefail

# Assumes this script is run from the sh/ folder
cd ../src/alfaparser

usage() {
  echo "Usage: $0 --policy <path_to_policy.alfa>"
  echo "Example:"
  echo "  $0 --policy ../src/alfaparser/policy.alfa"
  exit 1
}

POLICY_PATH=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --policy|-p)
      [[ $# -ge 2 ]] || usage
      POLICY_PATH="$2"
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

if [[ -z "$POLICY_PATH" ]]; then
  echo "Error: --policy is required"
  usage
fi

# Resolve to absolute path before we run Go (because we cd)
if [[ "$POLICY_PATH" != /* ]]; then
  POLICY_PATH="$(cd ../../sh && pwd)/$POLICY_PATH"
fi

if [[ ! -f "$POLICY_PATH" ]]; then
  echo "Error: policy file not found: $POLICY_PATH"
  exit 1
fi

exec go run alfa_parser.go -policy "$POLICY_PATH"