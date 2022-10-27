#!/bin/bash

make build
./rosetta-server -config=./script/config.json
# nohup make run-devnet > /dev/null 2>&1 &

# sleep 15

# curl -s --location --request POST 'http://localhost:8080/network/list' \
# --header 'Content-Type: application/json' \
# --data-raw '{
#     "metadata" : {}
# }'

