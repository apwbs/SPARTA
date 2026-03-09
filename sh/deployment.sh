#!/usr/bin/env bash
set -euo pipefail

cd ../src/blockchain

command -v truffle >/dev/null 2>&1 || { echo "Error: truffle not found in PATH"; exit 1; }

contract_address=$(
  truffle migrate --network development --reset \
  | tee /dev/tty \
  | sed -nE 's/^[[:space:]]*> contract address:[[:space:]]*(0x[a-fA-F0-9]{40}).*$/\1/p' \
  | tail -n 1
)

if [[ -z "$contract_address" ]]; then
  echo "Error: could not extract contract address from truffle output"
  exit 1
fi

upsert_env() {
  local env_file="$1"
  mkdir -p "$(dirname "$env_file")"
  touch "$env_file"

  if grep -q '^CONTRACT_ADDRESS_SPARTA=' "$env_file"; then
    sed -i.bak "s|^CONTRACT_ADDRESS_SPARTA=.*$|CONTRACT_ADDRESS_SPARTA=$contract_address|" "$env_file"
  else
    echo "CONTRACT_ADDRESS_SPARTA=$contract_address" >> "$env_file"
  fi

  echo "Updated $env_file with $contract_address"
}

for ENV_FILE in ../teeserver/.env ../tee/.env; do
  upsert_env "$ENV_FILE"
done