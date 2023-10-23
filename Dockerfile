#
# xpla localnet
#
# build:
#   docker build --force-rm -t xpladev/xpla .
# run:
#   docker run --rm -it --env-file=path/to/.env --name xpla-localnet xpladev/xpla

### BUILD
FROM golang:1.19-alpine AS build
WORKDIR /localnet

# Create appuser.
RUN adduser -D -g '' valiuser
# Install required binaries
RUN apk add --update --no-cache zip git make cmake build-base linux-headers musl-dev libc-dev

# Copy source files
COPY . /localnet/

ENV LIBWASMVM_VERSION=v1.3.0

RUN git clone --depth 1 https://github.com/microsoft/mimalloc; cd mimalloc; mkdir build; cd build; cmake ..; make -j$(nproc); make install
ENV MIMALLOC_RESERVE_HUGE_OS_PAGES=4

# See https://github.com/CosmWasm/wasmvm/releases
ADD https://github.com/CosmWasm/wasmvm/releases/download/${LIBWASMVM_VERSION}/libwasmvm_muslc.aarch64.a /lib/libwasmvm_muslc.aarch64.a
ADD https://github.com/CosmWasm/wasmvm/releases/download/${LIBWASMVM_VERSION}/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.x86_64.a
RUN sha256sum /lib/libwasmvm_muslc.aarch64.a | grep b1610f9c8ad8bdebf5b8f819f71d238466f83521c74a2deb799078932e862722
RUN sha256sum /lib/libwasmvm_muslc.x86_64.a | grep b4aad4480f9b4c46635b4943beedbb72c929eab1d1b9467fe3b43e6dbf617e32

# Copy the library you want to the final location that will be found by the linker flag `-lwasmvm_muslc`
RUN cp /lib/libwasmvm_muslc.`uname -m`.a /lib/libwasmvm_muslc.a

# Build executable
RUN LEDGER_ENABLED=false BUILD_TAGS=muslc LDFLAGS='-linkmode=external -extldflags "-L/localnet/mimalloc/build -lmimalloc -Wl,-z,muldefs -static"' make build

# --------------------------------------------------------
FROM alpine:3.15 AS runtime

WORKDIR /opt
RUN [ "mkdir", "-p", "/opt/integration_test" ]

COPY --from=build /localnet/build/xplad /usr/bin/xplad
COPY --from=build /localnet/integration_test /opt/integration_test

# Expose Cosmos ports
EXPOSE 9090
EXPOSE 8545
EXPOSE 26656
#EXPOSE 26657

# Set entry point
CMD [ "/usr/bin/xplad", "version" ]
