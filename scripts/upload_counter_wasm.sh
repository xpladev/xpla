#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# scripts -> revert_cases -> suites -> solidity -> tests -> repo root (5 levels)
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../../.." && pwd)"
DEFAULT_WASM="$REPO_ROOT/tests/solidity/suites/misc/counter_submsg.wasm"
WASM_PATH="${1:-$DEFAULT_WASM}"
if [[ ! -f "$WASM_PATH" ]]; then
  echo "Error: WASM file not found: $WASM_PATH" >&2
  echo "Usage: $0 [WASM_PATH]" >&2
  exit 1
fi

CHAINDIR="${CHAINDIR:-$HOME/.xpla}"
CHAIN_ID="${CHAIN_ID:-xpla-1}"
KEYRING="${KEYRING:-test}"
FROM="${FROM:-dev0}"
LABEL_PRECOMPILE="${LABEL_PRECOMPILE:-wasm_counter_precompile}"
LABEL_REVERT="${LABEL_REVERT:-wasm_counter_revert}"
GAS_PRICES="${GAS_PRICES:-280000000000axpla}"
GAS="${GAS:-2000000}"

# Store code
TX_OUT=$(xplad tx wasm store "$WASM_PATH" --from "$FROM" --chain-id "$CHAIN_ID" \
  --home "$CHAINDIR" --keyring-backend "$KEYRING" --gas-prices "$GAS_PRICES" --gas "$GAS" --broadcast-mode sync -y --output json 2>/dev/null || true)
if [[ -z "$TX_OUT" ]]; then
  echo "Error: wasm store failed. Is the chain running and xplad in PATH?" >&2
  exit 1
fi

TX_HASH=$(echo "$TX_OUT" | jq -r '.txhash')

# Poll until store tx is in a block (same as instantiate; needed when run from Node)
CODE_ID=""
for attempt in $(seq 1 15); do
  sleep 2
  TX_RESPONSE=$(xplad query tx "$TX_HASH" --home "$CHAINDIR" --output json 2>&1) || true
  CODE_ID=$(echo "$TX_RESPONSE" | jq -r '.events[]? | select(.type=="store_code") | .attributes[]? | select(.key=="code_id") | .value' 2>/dev/null | head -1)
  if [[ -z "$CODE_ID" || "$CODE_ID" == "null" ]]; then
    CODE_ID=$(echo "$TX_RESPONSE" | jq -r '(.logs[0].events // [])[]? | select(.type=="store_code") | .attributes[]? | select(.key=="code_id") | .value' 2>/dev/null | head -1)
  fi
  [[ -n "$CODE_ID" && "$CODE_ID" != "null" ]] && break
done
if [[ -z "$CODE_ID" || "$CODE_ID" == "null" ]]; then
  echo "Error: could not parse code_id from store output." >&2
  echo "TX_RESPONSE (last attempt): $TX_RESPONSE" >&2
  exit 1
fi

# Helper: query tx until instantiate address is found (block may not be committed yet)
# Optional second arg: max_attempts (default 20)
parse_instantiate_address() {
  local tx_hash="$1"
  local max_attempts="${2:-20}"
  [[ -z "$tx_hash" || "$tx_hash" == "null" ]] && { echo "Error: empty tx_hash" >&2; return 1; }
  for attempt in $(seq 1 "$max_attempts"); do
    sleep 2
    # Capture stdout+stderr so we see "tx not found" etc. when second tx not yet in block
    TX_RESPONSE=$(xplad query tx "$tx_hash" --home "$CHAINDIR" --output json 2>&1) || true
    CONTRACT_ADDRESS=$(echo "$TX_RESPONSE" | jq -r '.events[]? | select(.type=="instantiate") | .attributes[]? | select(.key=="_contract_address") | .value' 2>/dev/null | head -1)
    if [[ -z "$CONTRACT_ADDRESS" || "$CONTRACT_ADDRESS" == "null" ]]; then
      CONTRACT_ADDRESS=$(echo "$TX_RESPONSE" | jq -r '(.logs[0].events // [])[]? | select(.type=="instantiate") | .attributes[]? | select(.key=="_contract_address") | .value' 2>/dev/null | head -1)
    fi
    if [[ -n "$CONTRACT_ADDRESS" && "$CONTRACT_ADDRESS" != "null" ]]; then
      return 0
    fi
  done
  echo "Error: could not get instantiate contract address for tx $tx_hash" >&2
  return 1
}

# Instantiate (1) for precompiles/test/wasm
INSTANTIATE_MSG='{}'
INST_OUT=$(xplad tx wasm instantiate $CODE_ID $INSTANTIATE_MSG --from "$FROM" --no-admin \
  --label "$LABEL_PRECOMPILE" --chain-id "$CHAIN_ID" --home "$CHAINDIR" --keyring-backend "$KEYRING" --gas-prices "$GAS_PRICES" --gas "$GAS" --broadcast-mode sync -y --output json 2>/dev/null || true)
if [[ -z "$INST_OUT" ]]; then
  echo "Error: wasm instantiate (1) failed." >&2
  exit 1
fi
TX_HASH=$(echo "$INST_OUT" | jq -r '.txhash')
if ! parse_instantiate_address "$TX_HASH"; then
  echo "Error: could not parse contract address from instantiate (1) output." >&2
  exit 1
fi
COUNTER_WASM_ADDRESS="$CONTRACT_ADDRESS"

# Brief pause before second instantiate (we already confirmed first tx is in a block via query tx).
sleep 5

# Instantiate (2) for revert_cases (broadcast-mode sync; poll for tx in block).
# If this tx stays in mempool and never gets into a block, the cause is likely node-side
# (mempool/block builder logic or chain config); check node logs or mempool config.
INST_OUT=$(xplad tx wasm instantiate $CODE_ID $INSTANTIATE_MSG --from "$FROM" --no-admin \
  --label "$LABEL_REVERT" --chain-id "$CHAIN_ID" --home "$CHAINDIR" --keyring-backend "$KEYRING" --gas-prices "$GAS_PRICES" --gas "$GAS" --broadcast-mode sync -y --output json 2>/dev/null || true)
if [[ -z "$INST_OUT" ]]; then
  echo "Error: wasm instantiate (2) failed." >&2
  exit 1
fi
sleep 5
TX_HASH=$(echo "$INST_OUT" | jq -r '.txhash')
if ! parse_instantiate_address "$TX_HASH"; then
  echo "Error: could not parse contract address from instantiate (2) output." >&2
  exit 1
fi
REVERT_COUNTER_WASM_ADDRESS="$CONTRACT_ADDRESS"

# Single JSON line for test-helper.js to parse (-c = compact, one line)
jq -c -n --arg c "$COUNTER_WASM_ADDRESS" --arg r "$REVERT_COUNTER_WASM_ADDRESS" \
  '{COUNTER_WASM_ADDRESS:$c,REVERT_COUNTER_WASM_ADDRESS:$r}'
