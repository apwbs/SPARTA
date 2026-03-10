#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 --ccu ccu1|ccu2"
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

[[ -n "$CCU" ]] || usage

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

cd "$BASE_DIR"
exec ego run main