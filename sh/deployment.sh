#!/bin/bash
set -euo pipefail

cd ../src/blockchain

contract_address=$(
  truffle migrate --network development --reset \
  | tee /dev/tty \
  | sed -nE 's/^[[:space:]]*> contract address:[[:space:]]*(0x[a-fA-F0-9]{40}).*$/\1/p' \
  | tail -n 1
)

if [[ -z "$contract_address" ]]; then
  echo "Could not extract contract address from truffle output"
  exit 1
fi

ENV_FILE="../teeserver/.env"

if grep -q '^CONTRACT_ADDRESS_SPARTA=' "$ENV_FILE"; then
  sed -i.bak "s|^CONTRACT_ADDRESS_SPARTA=.*$|CONTRACT_ADDRESS_SPARTA=$contract_address|" "$ENV_FILE"
else
  echo "CONTRACT_ADDRESS_SPARTA=$contract_address" >> "$ENV_FILE"
fi

echo "✅ Updated $ENV_FILE with $contract_address"