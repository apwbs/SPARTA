#!/usr/bin/env bash
set -euo pipefail

# Assumes this script is run from the sh/ folder

command -v ipfs >/dev/null 2>&1 || { echo "Error: ipfs not found in PATH"; exit 1; }

usage() {
  echo "Usage: $0 --key_name <name>"
  echo "Example: $0 --key_name Patient"
  exit 1
}

KEY_NAME=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --key_name|-k)
      [[ $# -ge 2 ]] || usage
      KEY_NAME="$2"
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

if [[ -z "$KEY_NAME" ]]; then
  usage
fi

UPPER_NAME="$(echo "$KEY_NAME" | tr '[:lower:]' '[:upper:]')"
VAR_MAIN="IPNS_KEY_${UPPER_NAME}"
VAR_LIGHT="IPNS_KEY_${UPPER_NAME}_LIGHT"

# Generate and capture IDs.
# If key already exists, fall back to reading it from `ipfs key list -l`.
gen_or_get_id() {
  local name="$1"
  local id=""

  if id="$(ipfs key gen "$name" 2>/dev/null)"; then
    echo "$id"
    return 0
  fi

  id="$(ipfs key list -l | awk -v k="$name" '$2==k {print $1; exit}')"
  if [[ -z "$id" ]]; then
    echo "Error: could not generate or resolve key id for $name" >&2
    return 1
  fi
  echo "$id"
}

KEY_ID="$(gen_or_get_id "$KEY_NAME")"
LIGHT_ID="$(gen_or_get_id "${KEY_NAME}Light")"

upsert_env() {
  local env_file="$1"
  mkdir -p "$(dirname "$env_file")"
  touch "$env_file"

  # Ensure file ends with a newline
  if [[ -s "$env_file" ]] && [[ "$(tail -c 1 "$env_file")" != "" ]]; then
    echo >> "$env_file"
  fi

  local has_main=false
  local has_light=false
  grep -q "^${VAR_MAIN}=" "$env_file" && has_main=true
  grep -q "^${VAR_LIGHT}=" "$env_file" && has_light=true

  # If both are missing, add ONE blank line before appending (only if file already has content)
  if [[ "$has_main" == false && "$has_light" == false ]]; then
    if [[ -s "$env_file" ]]; then
      if [[ -n "$(tail -n 1 "$env_file")" ]]; then
        echo >> "$env_file"
      fi
    fi
  fi

  # Main
  if [[ "$has_main" == true ]]; then
    sed -i.bak "s|^${VAR_MAIN}=.*$|${VAR_MAIN}=\"${KEY_ID}\"|" "$env_file"
  else
    echo "${VAR_MAIN}=\"${KEY_ID}\"" >> "$env_file"
  fi

  # Light
  if [[ "$has_light" == true ]]; then
    sed -i.bak "s|^${VAR_LIGHT}=.*$|${VAR_LIGHT}=\"${LIGHT_ID}\"|" "$env_file"
  else
    echo "${VAR_LIGHT}=\"${LIGHT_ID}\"" >> "$env_file"
  fi

  echo "Updated $env_file:"
  echo "  $VAR_MAIN=\"$KEY_ID\""
  echo "  $VAR_LIGHT=\"$LIGHT_ID\""
}

for ENV_FILE in ../src/teeserver/.env ../src/tee/.env; do
  upsert_env "$ENV_FILE"
done