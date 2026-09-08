#
# xpla localnet
#
# build:
#   docker build --force-rm -t xpladev/xpla .
# run:
#   docker run --rm -it --env-file=path/to/.env --name xpla-localnet xpladev/xpla

### BUILD
FROM golang:1.23-alpine AS build

# Install required binaries
RUN apk add --update --no-cache zip git make cmake build-base linux-headers musl-dev libc-dev binutils-gold

WORKDIR /
RUN git clone --depth 1 --branch v3.0.1 https://github.com/microsoft/mimalloc; cd mimalloc; mkdir build; cd build; cmake ..; make -j$(nproc); make install
ENV MIMALLOC_RESERVE_HUGE_OS_PAGES=4

WORKDIR /workspace
# Copy source files
COPY . .
ARG GIT_VERSION=dev
ARG GIT_COMMIT=unknown
# Download dependencies and CosmWasm libwasmvm if found.
RUN set -eux; \
    export ARCH=$(uname -m); \
    WASM_VERSION=$(go list -mod=readonly -m -f '{{if .Replace}}replaced{{end}}{{.Version}}' github.com/CosmWasm/wasmvm/v2); \
    if [ ! -z "${WASM_VERSION}" ]; then \
      wget -O /lib/libwasmvm_muslc.${ARCH}.a https://github.com/CosmWasm/wasmvm/releases/download/${WASM_VERSION}/libwasmvm_muslc.${ARCH}.a; \
    fi; \
    go mod download

# Build executable
RUN LEDGER_ENABLED=false BUILD_TAGS=muslc LDFLAGS='-linkmode=external -extldflags "-L/mimalloc/build -lmimalloc -Wl,-z,muldefs -static"' \
    make build VERSION="$GIT_VERSION" COMMIT="$GIT_COMMIT"

FROM alpine:3.22 AS publish
RUN apk add --no-cache ca-certificates
COPY --from=build /workspace/build/xplad /bin/xplad
EXPOSE 9090 8545 26656 26657
# Alpine's default shell command must not become the node image's default.
CMD []
