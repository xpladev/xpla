#!/bin/sh
# MONIKER=validator1|validator2|validator3|validator4 sh /opt/tests/e2e/entrypoint.sh

# 1. chain init
xplad init $MONIKER --chain-id localtest_1-1 --home $XPLAHOME

# 2. copy the node setting files to the node home dir
cp -r /opt/tests/e2e/$MONIKER/* /opt/.xpla/config

# 3. register my validator & users keyfile
xplad keys add validator1 --recover --home $XPLAHOME < /opt/tests/e2e/test_keys/$MONIKER.mnemonics
xplad keys add user1 --recover --home $XPLAHOME < /opt/tests/e2e/test_keys/user1.mnemonics
xplad keys add user2 --recover --home $XPLAHOME < /opt/tests/e2e/test_keys/user2.mnemonics

# 4. check genesis.json
xplad genesis validate-genesis --home $XPLAHOME

# 5. start daemon
xplad start --home=$XPLAHOME
