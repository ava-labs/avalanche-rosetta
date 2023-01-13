#!/bin/bash

export CAMINO_NETWORK=${CAMINO_NETWORK:-testnet}
export CAMINO_CHAIN=${CAMINO_CHAIN:-501}
export CAMINO_MODE=${CAMINO_MODE:-online}
export CAMINO_GENESIS_HASH=${CAMINO_GENESIS_HASH:-"0x31ced5b9beb7f8782b014660da0cb18cc409f121f408186886e1ca3e8eeca96b"}

cat <<EOF > /app/caminogo-config.json
{
  "network-id": "$CAMINO_NETWORK",
  "http-host": "0.0.0.0",
  "api-keystore-enabled": false,
  "api-admin-enabled": false,
  "api-ipcs-enabled": false,
  "api-keystore-enabled": false,
  "db-dir": "/data",
  "chain-config-dir": "/app/configs/chains",
  "network-require-validator-to-connect": true
}
EOF

mkdir -p /app/configs/chains/C

cat <<EOF > /app/configs/chains/C/config.json
{
  "snowman-api-enabled": false,
  "coreth-admin-api-enabled": false,
  "rpc-gas-cap": 2500000000,
  "rpc-tx-fee-cap": 100,
  "eth-apis": ["internal-public-eth","internal-public-blockchain","internal-public-transaction-pool","internal-public-tx-pool","internal-public-debug","internal-private-debug","debug-tracer","web3","public-eth","public-eth-filter","public-debug","private-debug","net"],
  "pruning-enabled": false
}
EOF

cat <<EOF > /app/rosetta-config.json
{
  "mode": "$CAMINO_MODE",
  "rpc_endpoint": "http://localhost:9650",
  "listen_addr": "0.0.0.0:8080",
  "network_id": 1,
  "network_name": "$CAMINO_NETWORK",
  "chain_id": $CAMINO_CHAIN,
  "genesis_block_hash": "$CAMINO_GENESIS_HASH"
}
EOF

# Execute a custom command instead of default on
if [ -n "$@" ]; then
  exec $@
fi

exec /app/rosetta-runner \
  -mode $CAMINO_MODE \
  -camino-bin /app/camino-node \
  -camino-config /app/caminogo-config.json \
  -rosetta-bin /app/rosetta-server \
  -rosetta-config rosetta-config.json
