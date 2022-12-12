#!/bin/sh
# MONIKER=validator1|validator2|validator3|validator4 sh /opt/integration_test/entrypoint.sh

# 1. chain init
/usr/bin/xplad init $MONIKER --chain-id localtest_1-1 --home $XPLAHOME

# 2. copy the node setting files to the node home dir
cp -r /opt/integration_test/$MONIKER/* /opt/.xpla/config

# 3. register my validator & users keyfile
/usr/bin/xplad keys add validator1 --recover --home $XPLAHOME < /opt/integration_test/test_keys/$MONIKER.mnemonics
/usr/bin/xplad keys add user1 --recover --home $XPLAHOME < /opt/integration_test/test_keys/user1.mnemonics
/usr/bin/xplad keys add user2 --recover --home $XPLAHOME < /opt/integration_test/test_keys/user2.mnemonics

# 4. check genesis.json
/usr/bin/xplad validate-genesis --home $XPLAHOME

# 5. start daemon
/usr/bin/xplad start --home=$XPLAHOME
