#!/bin/sh
# MONIKER=validator1|validator2|validator3|validator4 sh /opt/integration_test/entrypoint.sh

# File is not atomic and somestimes secondary nodes copies the existing old genesisfile
# It should be cleared
rm -f /genesis/*

# 1. chain init
/usr/bin/xplad init $MONIKER --chain-id localtest_1-1 --home $XPLAHOME

# 2. Register the keys
# xpla1z2k85n48ydfvzslrugwzl4j2u7vtdyf3xvucmc
/usr/bin/xplad keys add validator1 --recover --keyring-backend test --home $XPLAHOME < /opt/integration_test/test_keys/validator1.mnemonics
# xpla16wx7ye3ce060tjvmmpu8lm0ak5xr7gm2dp0kpt
/usr/bin/xplad keys add validator2 --recover --keyring-backend test --home $XPLAHOME < /opt/integration_test/test_keys/validator2.mnemonics
# xpla1pe9mc2q72u94sn2gg52ramrt26x5efw6hr5gt4
/usr/bin/xplad keys add validator3 --recover --keyring-backend test --home $XPLAHOME < /opt/integration_test/test_keys/validator3.mnemonics
# xpla1luqjvjyns9e92h06tq6zqtw76k8xtegfcerzjr
/usr/bin/xplad keys add validator4 --recover --keyring-backend test --home $XPLAHOME < /opt/integration_test/test_keys/validator4.mnemonics
# xpla1y6gnay0pv49asun56la09jcmhg2kc949mpftvt
/usr/bin/xplad keys add user1 --recover --keyring-backend test --home $XPLAHOME < /opt/integration_test/test_keys/user1.mnemonics
# xpla1u27snswkjpenlscgvszcfjmz8uy2y5qacx0826
/usr/bin/xplad keys add user2 --recover --keyring-backend test --home $XPLAHOME < /opt/integration_test/test_keys/user2.mnemonics
# xpla14xprgqlnuep23kmvsk5utd9pw7w27yeyjlcmcz -- validator5_experimental
/usr/bin/xplad keys add validator5_experimental --recover --keyring-backend test --home $XPLAHOME < /opt/integration_test/test_keys/validator5_experimental.mnemonics
# xpla1ur90l8ecppc55gll7k57dk2tvs2w5m9jzptpcj -- volunteer validator1
/usr/bin/xplad keys add volunteer_validator1 --recover --keyring-backend test --home $XPLAHOME < /opt/integration_test/test_keys/volunteer_validator1.mnemonics
# xpla1yct6tmmm0twn2wz637lt0yz62xwtqhyqa84uu5 -- volunteer validator2
/usr/bin/xplad keys add volunteer_validator2 --recover --keyring-backend test --home $XPLAHOME < /opt/integration_test/test_keys/volunteer_validator2.mnemonics
# xpla1unq7rvf4jkcpmqww09j0u8k37qkgjxm43llwx5 -- volunteer validator3
/usr/bin/xplad keys add volunteer_validator3 --recover --keyring-backend test --home $XPLAHOME < /opt/integration_test/test_keys/volunteer_validator3.mnemonics

# 3. Add the genesis accounts
/usr/bin/xplad add-genesis-account $(/usr/bin/xplad keys show validator1 -a --keyring-backend test --home $XPLAHOME) 100000000000000000000axpla --keyring-backend test --home $XPLAHOME
/usr/bin/xplad add-genesis-account $(/usr/bin/xplad keys show validator2 -a --keyring-backend test --home $XPLAHOME) 100000000000000000000axpla --keyring-backend test --home $XPLAHOME
/usr/bin/xplad add-genesis-account $(/usr/bin/xplad keys show validator3 -a --keyring-backend test --home $XPLAHOME) 100000000000000000000axpla --keyring-backend test --home $XPLAHOME
/usr/bin/xplad add-genesis-account $(/usr/bin/xplad keys show validator4 -a --keyring-backend test --home $XPLAHOME) 100000000000000000000axpla --keyring-backend test --home $XPLAHOME
/usr/bin/xplad add-genesis-account $(/usr/bin/xplad keys show user1 -a --keyring-backend test --home $XPLAHOME) 100000000000000000000axpla --keyring-backend test --home $XPLAHOME
/usr/bin/xplad add-genesis-account $(/usr/bin/xplad keys show user2 -a --keyring-backend test --home $XPLAHOME) 100000000000000000000axpla --keyring-backend test --home $XPLAHOME

/usr/bin/xplad add-genesis-account $(/usr/bin/xplad keys show volunteer_validator1 -a --keyring-backend test --home $XPLAHOME) 100000000000000000000axpla --keyring-backend test --home $XPLAHOME
/usr/bin/xplad add-genesis-account $(/usr/bin/xplad keys show volunteer_validator2 -a --keyring-backend test --home $XPLAHOME) 100000000000000000000axpla --keyring-backend test --home $XPLAHOME
/usr/bin/xplad add-genesis-account $(/usr/bin/xplad keys show volunteer_validator3 -a --keyring-backend test --home $XPLAHOME) 100000000000000000000axpla --keyring-backend test --home $XPLAHOME

# -- experimentals
/usr/bin/xplad add-genesis-account $(/usr/bin/xplad keys show validator5_experimental -a --keyring-backend test --home $XPLAHOME) 100000000000000000000axpla --keyring-backend test --home $XPLAHOME

# 4. Get the node keys and create gentxs
for IDX in 1 2 3 4
do
    # 1) Copy the credentials
    cp /opt/integration_test/validator$IDX/node_key.json $XPLAHOME/config
    cp /opt/integration_test/validator$IDX/priv_validator_key.json $XPLAHOME/config

    # 2) Execute a gentx
    /usr/bin/xplad gentx validator$IDX 9000000000000000000axpla  \
        --chain-id="localtest_1-1" \
        --pubkey=$(xplad tendermint show-validator --home $XPLAHOME) \
        --min-self-delegation=1 \
        --moniker=validator$IDX \
        --commission-rate=0.1 \
        --commission-max-rate=0.2 \
        --commission-max-change-rate=0.01 \
        --ip="192.167.100.$IDX" \
        --keyring-backend test \
        --home $XPLAHOME

done

# 5. Do collect gentxs
/usr/bin/xplad collect-gentxs --home $XPLAHOME

# 6. Replace params
sed -i 's/"bond_denom": "stake"/"bond_denom": "axpla"/g' $XPLAHOME/config/genesis.json
sed -i 's/"evm_denom": "aphoton",/"evm_denom": "axpla",/g' $XPLAHOME/config/genesis.json
sed -i 's/"mint_denom": "stake",/"mint_denom": "axpla",/g' $XPLAHOME/config/genesis.json
sed -i 's/"denom": "stake",/"denom": "axpla",/g' $XPLAHOME/config/genesis.json
sed -i 's/"max_gas": "-1",/"max_gas": "5000000",/g' $XPLAHOME/config/genesis.json
sed -i 's/"no_base_fee": false,/"no_base_fee": true,/g' $XPLAHOME/config/genesis.json
sed -i 's/"inflation": "0.[0-9]\+",/"inflation": "0.000000000000000000",/g' $XPLAHOME/config/genesis.json
sed -i 's/"inflation_rate_change": "0.[0-9]\+",/"inflation_rate_change": "0.000000000000000000",/g' $XPLAHOME/config/genesis.json
sed -i 's/"inflation_min": "0.[0-9]\+",/"inflation_min": "0.000000000000000000",/g' $XPLAHOME/config/genesis.json
sed -i 's/"unbonding_time": "[0-9]\+s"/"unbonding_time": "4s"/' $XPLAHOME/config/genesis.json

# reduce blocktime around 2 sec
sed -i 's/"blocks_per_year": "[0-9]\+"/"blocks_per_year": "19000000"/g' $XPLAHOME/config/genesis.json

# slashing params
# downtime_jail_duration reduces into 20s, while is restricted over 60s
sed -i 's/"downtime_jail_duration": "[0-9]\+s"/"downtime_jail_duration": "20s"/g' $XPLAHOME/config/genesis.json
# signed_blocks_window reduces into 5, while the valus is restricted over 10, by the restriction of Cosmos SDK
sed -i 's/"signed_blocks_window": "[0-9]\+"/"signed_blocks_window": "5"/g' $XPLAHOME/config/genesis.json

# gov params
sed -i 's/"max_deposit_period": "[0-9]\+s"/"max_deposit_period": "4s"/' $XPLAHOME/config/genesis.json
sed -i 's/"voting_period": "[0-9]\+s"/"voting_period": "4s"/' $XPLAHOME/config/genesis.json

/usr/bin/xplad validate-genesis --home $XPLAHOME

# 7. Copy to the shared folder
cp $XPLAHOME/config/genesis.json /genesis

### ALL DONE FOR GENESIS CONFIG
### followings are for validator setting

# 1. Copy the node setting files to the node home dir
cp -r /opt/integration_test/$MONIKER/* $XPLAHOME/config
cp -r /opt/integration_test/common_configs/* $XPLAHOME/config

# 2. Get genesis.json from the shared
cp /genesis/genesis.json $XPLAHOME/config

# 4. check genesis.json
# downtime_jail_duration will raise an error
# /usr/bin/xplad validate-genesis --home $XPLAHOME
cat $XPLAHOME/config/genesis.json

# 5. start daemon
/usr/bin/xplad tendermint unsafe-reset-all --home=$XPLAHOME
/usr/bin/xplad start --home=$XPLAHOME
