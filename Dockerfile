#
# xpla localnet
#
# build:
#   docker build --force-rm -t xpladev/xpla .
# run:
#   docker run --rm -it --env-file=path/to/.env --name xpla-localnet xpladev/xpla

### BUILD
FROM golang:1.23.0-alpine3.20 AS build

# Create appuser.
RUN adduser -D -g '' valiuser
# Install required binaries
RUN apk add --update --no-cache zip git make cmake build-base linux-headers musl-dev libc-dev

WORKDIR /
RUN git clone --depth 1 https://github.com/microsoft/mimalloc; cd mimalloc; mkdir build; cd build; cmake ..; make -j$(nproc); make install
ENV MIMALLOC_RESERVE_HUGE_OS_PAGES=4


WORKDIR /workspace
# Copy source files
COPY . .
# Download dependencies and CosmWasm libwasmvm if found.
RUN set -eux; \    
    export ARCH=$(uname -m); \
    WASM_VERSION=$(go list -mod=readonly -m all | grep github.com/CosmWasm/wasmvm | awk '{print $2}'); \
    if [ ! -z "${WASM_VERSION}" ]; then \
      wget -O /lib/libwasmvm_muslc.a https://github.com/CosmWasm/wasmvm/releases/download/${WASM_VERSION}/libwasmvm_muslc.${ARCH}.a; \      
    fi; \
    go mod download;

# Build executable
RUN LEDGER_ENABLED=false BUILD_TAGS=muslc LDFLAGS='-linkmode=external -extldflags "-L/mimalloc/build -lmimalloc -Wl,-z,muldefs -static"' make build

# --------------------------------------------------------
FROM alpine:3.20 AS runtime

COPY --from=build /workspace/build/xplad /usr/bin/xplad
COPY --from=build /workspace/tests/e2e /opt/tests/e2e

# Expose Cosmos ports
EXPOSE 9090
EXPOSE 8545
EXPOSE 26656
#EXPOSE 26657

# Set entry point
CMD [ "/usr/bin/xplad", "version" ]
