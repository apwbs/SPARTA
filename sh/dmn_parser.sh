#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 --xml <path_to_dmn_xml>"
  echo "Example:"
  echo "  $0 --xml ../src/dmnparser/xmls/running_example.xml"
  exit 1
}

XML_PATH=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --xml|-x)
      [[ $# -ge 2 ]] || usage
      XML_PATH="$2"
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

if [[ -z "$XML_PATH" ]]; then
  echo "Error: --xml is required"
  usage
fi

# Convert to absolute path while still in sh/
if [[ "$XML_PATH" != /* ]]; then
  XML_PATH="$(pwd)/$XML_PATH"
fi

if [[ ! -f "$XML_PATH" ]]; then
  echo "Error: XML file not found: $XML_PATH"
  exit 1
fi

# Now go to the parser folder
cd ../src/dmnparser/parser

exec go run . -xml "$XML_PATH"