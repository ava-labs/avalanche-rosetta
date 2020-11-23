#!/bin/bash

export AVALANCHE_NETWORK=${AVALANCHE_NETWORK:-testnet}
export AVALANCHE_CHAIN=${AVALANCHE_CHAIN:-43113}

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
  "mode": "online",
  "rpc_endpoint": "http://localhost:9650",
  "listen_addr": "0.0.0.0:8081",
  "network_id": 1,
  "network_name": "$AVALANCHE_NETWORK",
  "chain_id": $AVALANCHE_CHAIN
}
EOF

/app/avalanchego --config-file=/app/avalanchego-config.json & \
/app/rosetta -config=/app/rosetta-config.json
