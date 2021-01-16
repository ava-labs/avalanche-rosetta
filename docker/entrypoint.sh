#!/bin/bash

export AVALANCHE_NETWORK=${AVALANCHE_NETWORK:-testnet}
export AVALANCHE_CHAIN=${AVALANCHE_CHAIN:-43113}
export AVALANCHE_MODE=${AVALANCHE_MODE:-online}
export AVALANCHE_GENESIS_HASH=${AVALANCHE_GENESIS_HASH:-"0x31ced5b9beb7f8782b014660da0cb18cc409f121f408186886e1ca3e8eeca96b"}

cat <<EOF > /app/avalanchego-config.json
{
  "network-id": "$AVALANCHE_NETWORK",
  "http-host": "0.0.0.0",
  "api-keystore-enabled": false,
  "api-admin-enabled": false,
  "api-ipcs-enabled": false,
  "coreth-config": {
    "snowman-api-enabled": true,
    "coreth-admin-api-enabled": true,
    "net-api-enabled": true,
    "rpc-gas-cap": 2500000000,
    "rpc-tx-fee-cap": 100,
    "eth-api-enabled": true,
    "personal-api-enabled": true,
    "tx-pool-api-enabled": true,
    "debug-api-enabled": true,
    "web3-api-enabled": true
  }
}
EOF

cat <<EOF > /app/rosetta-config.json
{
  "mode": "$AVALANCHE_MODE",
  "rpc_endpoint": "http://localhost:9650",
  "listen_addr": "0.0.0.0:8080",
  "network_id": 1,
  "network_name": "$AVALANCHE_NETWORK",
  "chain_id": $AVALANCHE_CHAIN,
  "genesis_block_hash": "$AVALANCHE_GENESIS_HASH"
}
EOF

# Configure prefunded account for Rosetta Construction check if running Testnet
if [ "$AVALANCHE_CHAIN" -eq "43113" ]; then
  if ([ -n "$ROSETTA_PREFUNDED_ACCOUNT_KEY" ] && [ -n "$ROSETTA_PREFUNDED_ACCOUNT_ADDRESS" ]); then
    query=".construction.prefunded_accounts += [{\"privkey\": \"$ROSETTA_PREFUNDED_ACCOUNT_KEY\",\"account_identifier\": {\"address\": \"$ROSETTA_PREFUNDED_ACCOUNT_ADDRESS\"},\"curve_type\": \"secp256k1\",\"currency\": {\"symbol\": \"AVAX\",\"decimals\": 18}}]"
    cat <<< $(/app/jq $query ./rosetta-cli-conf/testnet/config.json) > ./rosetta-cli-conf/testnet/config.json
  fi
fi

# Execute a custom command instead of default on
if [ -n "$@" ]; then
  exec $@
fi

exec /app/rosetta-runner \
  -mode $AVALANCHE_MODE \
  -avalanche-bin /app/avalanchego \
  -avalanche-config /app/avalanchego-config.json \
  -rosetta-bin /app/rosetta-server \
  -rosetta-config rosetta-config.json