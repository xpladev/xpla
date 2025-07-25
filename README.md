# XPLA

## Build

```sh
make install
```

## Build docker image

```bash
make local-image
```

## Test

### Unit test & Integration test
```bash
make test
```

### End-to-end (e2e) test

#### 1. Local e2e test
See detailed instructions in [Local e2e test](./tests/e2e/README.md)

#### 2. e2e test with interchaintest
**Prerequisites:**
- Docker image must be built first: [Build docker image](#build-docker-image)

**Run test:**
```bash
cd tests/e2e/multichain
go test ./...
```

## License

XPLA is licensed under the Apache License 2.0
