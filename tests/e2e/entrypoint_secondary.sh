#!/bin/sh
# MONIKER=validator1|validator2|validator3|validator4 sh /opt/tests/e2e/entrypoint.sh

# 1. chain init
xplad init $MONIKER --chain-id localtest_1-1 --home $XPLAHOME
xplad config set client chain-id localtest_1-1 --home $XPLAHOME

# 2. copy the node setting files to the node home dir
cp -r /opt/tests/e2e/$MONIKER/* $XPLAHOME/config
cp -r /opt/tests/e2e/common_configs/* $XPLAHOME/config

sed -i "s/moniker = \"validator1\"/moniker = \"$MONIKER\"/g" $XPLAHOME/config/config.toml

# 3. register my validator & users keyfile
xplad keys add $MONIKER --recover --keyring-backend test --home $XPLAHOME < /opt/tests/e2e/test_keys/$MONIKER.mnemonics
xplad keys add user1 --recover --keyring-backend test --home $XPLAHOME < /opt/tests/e2e/test_keys/user1.mnemonics
xplad keys add user2 --recover --keyring-backend test --home $XPLAHOME < /opt/tests/e2e/test_keys/user2.mnemonics

# 4. get genesis.json from the shared folder
# "depends_on" in docker-compose.yml means that it depends on Dockerfile image, not completely wait for the entrypoint exeuction.
sleep 10s
cp /genesis/genesis.json $XPLAHOME/config

# 4. check genesis.json
xplad genesis validate-genesis --home $XPLAHOME

# 5. start daemon
xplad comet unsafe-reset-all --home=$XPLAHOME
xplad start --home=$XPLAHOME
