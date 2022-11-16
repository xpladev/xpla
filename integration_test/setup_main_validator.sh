#!/bin/sh

# MONIKER=validator1|validator2|validator3|validator4 ./setup.sh

echo "1. Node init"
xplad init $MONIKER --chain-id localtest-47_1

echo "2. Create wallets"
# xpla1z2k85n48ydfvzslrugwzl4j2u7vtdyf3xvucmc
xplad keys add validator1 --recover < ./test_keys/validator1.mnemonics
# xpla16wx7ye3ce060tjvmmpu8lm0ak5xr7gm2dp0kpt
xplad keys add validator2 --recover < ./test_keys/validator2.mnemonics
# xpla16lx5m49s22kd6mzd2pu7fgdh73crf63kpe0vrg
xplad keys add validator3 --recover < ./test_keys/validator3.mnemonics
# xpla1chr502n2f7ghmjrcm6930hzt8vzr68yhyn0rcm
xplad keys add validator4 --recover < ./test_keys/validator4.mnemonics
# xpla14dwqhzq47ce6jzs4wlz0kn2en6re7re7jd5rqg
xplad keys add user1 --recover < ./test_keys/user1.mnemonics
# xpla152ur5h83g0zs75u3zmemqt2ccrlwl7ykplp0cv
xplad keys add user2 --recover < ./test_keys/user2.mnemonics

echo "3. Add genesis account"
xplad add-genesis-account $(xplad keys show validator1 -a) 100000000000000000000axpla
xplad add-genesis-account $(xplad keys show validator2 -a) 100000000000000000000axpla
xplad add-genesis-account $(xplad keys show validator3 -a) 100000000000000000000axpla
xplad add-genesis-account $(xplad keys show validator4 -a) 100000000000000000000axpla
xplad add-genesis-account $(xplad keys show user1 -a) 100000000000000000000axpla
xplad add-genesis-account $(xplad keys show user2 -a) 100000000000000000000axpla

echo "4. Gentx"
xplad gentx validator1 10000000000000000000axpla  \
    --chain-id="localtest_1-1" \
    --pubkey=$(xplad tendermint show-validator --home ~/.xpla) \
    --min-self-delegation=1 \
    --moniker=validator1 \
    --commission-rate=0.1 \
    --commission-max-rate=0.2 \
    --commission-max-change-rate=0.01 \
    --ip=192.167.100.1

echo "5. Collect Gentx"
xplad collect-gentxs

echo "6. Validator Gentx"
xplad validate-genesis
