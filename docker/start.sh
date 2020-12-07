#!/bin/bash

export AVALANCHE_NETWORK=${AVALANCHE_NETWORK:-testnet}
export AVALANCHE_CHAIN=${AVALANCHE_CHAIN:-43113}
export AVALANCHE_MODE=${AVALANCHE_CHAIN:online}

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
  "chain_id": $AVALANCHE_CHAIN
}
EOF

/app/rosetta-runner \
  -mode $AVALANCHE_MODE \
  -avalanche-bin /app/avalanchego \
  -avalanche-config /app/avalanchego-config.json \
  -rosetta-bin /app/rosetta-server \
  -rosetta-config rosetta-config.json