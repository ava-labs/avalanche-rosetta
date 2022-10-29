#!/bin/bash

sudo ethtool -K eth0 tx off rx off

# downloading cli
curl -sSfL https://raw.githubusercontent.com/coinbase/rosetta-cli/master/scripts/install.sh | sh -s

echo "start check:construction"
./bin/rosetta-cli check:construction --configuration-file rosetta-cli-conf/devnet/config.json

