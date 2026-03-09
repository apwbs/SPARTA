#!/usr/bin/env bash
set -euo pipefail

# Assumes this script is run from the sh/ folder
cd ../src/genstruct

usage() {
  echo "Usage: $0 --struct_name <Name> --input_file <json_path> [--decision_inputs <xml_path>]"
  echo "Example:"
  echo "  $0 --struct_name Patient --input_file ../data/patients.json --decision_inputs ../dmnparser/xmls/running_example.xml"
  exit 1
}

STRUCT_NAME=""
INPUT_FILE=""
DECISION_INPUTS=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --struct_name|-s)
      [[ $# -ge 2 ]] || usage
      STRUCT_NAME="$2"
      shift 2
      ;;
    --input_file|-i)
      [[ $# -ge 2 ]] || usage
      INPUT_FILE="$2"
      shift 2
      ;;
    --decision_inputs|-d)
      [[ $# -ge 2 ]] || usage
      DECISION_INPUTS="$2"
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

if [[ -z "$STRUCT_NAME" ]]; then
  echo "Error: --struct_name is required"
  usage
fi

if [[ -z "$INPUT_FILE" ]]; then
  echo "Error: --input_file is required"
  usage
fi

# Resolve paths to absolute (since we cd into src/structgen)
if [[ "$INPUT_FILE" != /* ]]; then
  INPUT_FILE="$(cd ../../sh && pwd)/$INPUT_FILE"
fi
if [[ -n "$DECISION_INPUTS" && "$DECISION_INPUTS" != /* ]]; then
  DECISION_INPUTS="$(cd ../../sh && pwd)/$DECISION_INPUTS"
fi

if [[ ! -f "$INPUT_FILE" ]]; then
  echo "Error: JSON input file not found: $INPUT_FILE"
  exit 1
fi

if [[ -n "$DECISION_INPUTS" && ! -f "$DECISION_INPUTS" ]]; then
  echo "Error: XML decision_inputs file not found: $DECISION_INPUTS"
  exit 1
fi

CMD=(go run json_to_struct_parser.go -struct_name "$STRUCT_NAME" -input_file "$INPUT_FILE")
if [[ -n "$DECISION_INPUTS" ]]; then
  CMD+=(-decision_inputs "$DECISION_INPUTS")
fi

exec "${CMD[@]}"