# End-to-end Test

This module is basically for the automated end-to-end test. But if you want to execute by yourself for testing (test testing) purpose, this tests are also executable on your local. Please follow the step below:

## Prerequisites

- Docker >= 20.10
- Docker compose >= 2.12

## How to run

```bash
# 1. From the repo root, move to the tests/e2e, and execute docker compose
cd tests/e2e
docker-compose up -d

# 2. Wait for building. Once done without error, you may check the nodes running
docker ps

#CONTAINER ID   IMAGE                    COMMAND                  CREATED          STATUS          PORTS                                                                             NAMES
#648b7146ce8c   e2e-node1   "sh -c 'MONIKER=vali…"   44 minutes ago   Up 44 minutes   0.0.0.0:8545->8545/tcp, 0.0.0.0:9090->9090/tcp, 0.0.0.0:26656->26656/tcp          xpla-localnet-validator1
#97279d567135   e2e-node2   "sh -c 'MONIKER=vali…"   44 minutes ago   Up 44 minutes   8545/tcp, 9090/tcp, 0.0.0.0:9100->9100/tcp, 26656/tcp, 0.0.0.0:26666->26666/tcp   xpla-localnet-validator2
#f604f68c3f82   e2e-node3   "sh -c 'MONIKER=vali…"   44 minutes ago   Up 44 minutes   8545/tcp, 9090/tcp, 0.0.0.0:9110->9110/tcp, 26656/tcp, 0.0.0.0:26676->26676/tcp   xpla-localnet-validator3
#c3d0d9daefd2   e2e-node4   "sh -c 'MONIKER=vali…"   44 minutes ago   Up 44 minutes   8545/tcp, 9090/tcp, 0.0.0.0:9120->9120/tcp, 26656/tcp, 0.0.0.0:26686->26686/tcp   xpla-localnet-validator4

# 3. Execute tests
go test

# Do not execute short mode
# (X) go test -short

# ...
# PASS
# ok      github.com/xpladev/xpla/tests/e2e        29.365s

# If you see the pass sign, you may down the nodes
docker-compose down
```

## Test scenario

### WASM

- Send `delegation` tx
- Upload the contract binary and get a code ID
- With the code ID above, try to instantiate the contract
- Execute the contract
- Assert from querying the contract in each test step by assertion
