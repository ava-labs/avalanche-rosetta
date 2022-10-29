#!/bin/bash

sudo ethtool -K eth0 tx off rx off

make build
nohup ./rosetta-server -config=./scripts/config.json > /dev/null 2>&1 &

sleep 15

curl -s --location --request POST 'http://localhost:8085/network/list' \
--header 'Content-Type: application/json' \
--data-raw '{
    "metadata" : {}
}'


