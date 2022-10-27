#!/bin/bash

make build
ls
./rosetta-server -config=./scripts/config.json
# nohup make run-devnet > /dev/null 2>&1 &

# sleep 15

# curl -s --location --request POST 'http://localhost:8080/network/list' \
# --header 'Content-Type: application/json' \
# --data-raw '{
#     "metadata" : {}
# }'

