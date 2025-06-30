#!/usr/bin/make -f

BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')

# don't override user values
ifeq (,$(VERSION))
  VERSION := $(shell git describe --exact-match 2>/dev/null)
  # if VERSION is empty, then populate it with branch's name and raw commit hash
  ifeq (,$(VERSION))
    VERSION := $(BRANCH)-$(COMMIT)
  endif
endif

NAME := xpla
APPNAME := xplad
LEDGER_ENABLED ?= true
TM_VERSION := $(shell go list -m github.com/cometbft/cometbft | sed 's:.* ::') # grab everything after the space in "github.com/cometbft/cometbft v0.34.7"
BUILDDIR ?= $(CURDIR)/build
GO_SYSTEM_VERSION = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1-2)
REQUIRE_GO_VERSION = 1.23
GO_VERSION := $(shell cat go.mod | grep -E 'go [0-9].[0-9]+' | cut -d ' ' -f 2)

# for dockerized protobuf tools
DOCKER := $(shell which docker)
BUF_IMAGE=bufbuild/buf@sha256:3cb1f8a4b48bd5ad8f09168f10f607ddc318af202f5c057d52a45216793d85e5 #v1.4.0
DOCKER_BUF := $(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(BUF_IMAGE)

export GO111MODULE = on

# process build tags

build_tags = netgo
ifeq ($(LEDGER_ENABLED),true)
  ifeq ($(OS),Windows_NT)
    GCCEXE = $(shell where gcc.exe 2> NUL)
    ifeq ($(GCCEXE),)
      $(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
    else
      build_tags += ledger cgo
    endif
  else
    UNAME_S = $(shell uname -s)
    ifeq ($(UNAME_S),OpenBSD)
      $(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
    else
      GCC = $(shell command -v gcc 2> /dev/null)
      ifeq ($(GCC),)
        $(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
      else
        build_tags += ledger cgo
      endif
    endif
  endif
endif

ifeq (cleveldb,$(findstring cleveldb,$(CUSTOM_BUILD_OPTIONS)))
  build_tags += gcc cleveldb
endif
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
whitespace += $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

# process linker flags

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=$(NAME) \
	  -X github.com/cosmos/cosmos-sdk/version.AppName=$(APPNAME) \
	  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
	  -X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)" \
	  -X github.com/cometbft/cometbft/version.TMCoreSemVer=$(TM_VERSION)

ifeq (cleveldb,$(findstring cleveldb,$(CUSTOM_BUILD_OPTIONS)))
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=cleveldb
endif
ifeq (,$(findstring nostrip,$(CUSTOM_BUILD_OPTIONS)))
  ldflags += -w -s
endif
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'
# check for nostrip option
ifeq (,$(findstring nostrip,$(CUSTOM_BUILD_OPTIONS)))
  BUILD_FLAGS += -trimpath
endif

check_version:
ifneq ($(shell [ "$(GO_SYSTEM_VERSION)" \< "$(REQUIRE_GO_VERSION)" ] && echo true),)
	@echo "ERROR: Minumum Go version $(REQUIRE_GO_VERSION) is required for $(VERSION) of xpla, but system has $(GO_SYSTEM_VERSION)."
	exit 1
endif

all: install

.PHONY: install
install: check_version go.sum
	go install -mod=readonly $(BUILD_FLAGS) ./cmd/xplad

.PHONY: build
build: check_version go.sum
	go build -mod=readonly $(BUILD_FLAGS) -o build/xplad ./cmd/xplad

build-release: build/linux/amd64 build/linux/arm64

build-release-amd64: go.sum $(BUILDDIR)/
	$(DOCKER) buildx create --name xpla-builder || true
	$(DOCKER) buildx use xpla-builder
	$(DOCKER) buildx build \
		--build-arg GO_VERSION=$(GO_VERSION) \
		--build-arg GIT_VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(COMMIT) \
		--build-arg GOOS=linux \
		--build-arg GOARCH=amd64 \
		--platform linux/amd64 \
		-t xpla:local-amd64 \
		--load \
		-f Dockerfile .
	$(DOCKER) rm -f xpla-builder || true
	$(DOCKER) create -ti --name xpla-builder xpla:local-amd64
	$(DOCKER) cp xpla-builder:/usr/bin/xplad $(BUILDDIR)/release/xplad
	tar -czvf $(BUILDDIR)/release/xpla_$(VERSION)_Linux_x86_64.tar.gz -C $(BUILDDIR)/release/ xplad
	rm $(BUILDDIR)/release/xplad
	$(DOCKER) rm -f xpla-builder

build-release-arm64: go.sum $(BUILDDIR)/
	$(DOCKER) buildx create --name xpla-builder  || true
	$(DOCKER) buildx use xpla-builder 
	$(DOCKER) buildx build \
		--build-arg GO_VERSION=$(GO_VERSION) \
		--build-arg GIT_VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(COMMIT) \
		--build-arg GOOS=linux \
		--build-arg GOARCH=arm64 \
		--platform linux/arm64 \
		-t xpla:local-arm64 \
		--load \
		-f Dockerfile .
	$(DOCKER) rm -f xpla-builder || true
	$(DOCKER) create -ti --name xpla-builder xpla:local-arm64
	$(DOCKER) cp xpla-builder:/usr/bin/xplad $(BUILDDIR)/release/xplad 
	tar -czvf $(BUILDDIR)/release/xpla_$(VERSION)_Linux_arm64.tar.gz -C $(BUILDDIR)/release/ xplad 
	rm $(BUILDDIR)/release/xplad
	$(DOCKER) rm -f xpla-builder

.PHONY: test
test: go.sum
	go clean -testcache
	go test -short -p 1 ./...

go.sum: go.mod
	@go mod verify
	@go mod tidy

###############################################################################
###                                Protobuf                                 ###
###############################################################################
PROTO_VERSION=0.13.0
PROTO_BUILDER_IMAGE=ghcr.io/cosmos/proto-builder:$(PROTO_VERSION)
PROTO_FORMATTER_IMAGE=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace --user 0 $(PROTO_BUILDER_IMAGE)

proto-all: proto-format proto-lint proto-gen

proto-gen:
	@echo "Generating Protobuf files"
	$(PROTO_FORMATTER_IMAGE) sh ./scripts/protocgen.sh

proto-format:
	@echo "Formatting Protobuf files"
	$(PROTO_FORMATTER_IMAGE) \
	find ./ -name *.proto -exec clang-format -i {} \;

proto-swagger-gen:
	@./scripts/protoc-swagger-openapi-gen.sh

proto-lint:
	$(PROTO_FORMATTER_IMAGE) buf lint --error-format=json

proto-update-deps:
	@echo "Updating Protobuf dependencies"
	$(DOCKER) run --rm -v $(CURDIR)/proto:/workspace --workdir /workspace $(PROTO_BUILDER_IMAGE) buf mod update

###############################################################################
###                          Precompiled contract                           ###
###############################################################################

# TODO: precompiled interface should be changed as a NPM package
abi-gen:
	cd precompile && \
	solc --abi --pretty-json --overwrite -o auth auth/IAuth.sol && \
	solc --abi --pretty-json --overwrite -o bank bank/IBank.sol && \
	solc --abi --pretty-json --overwrite -o distribution distribution/IDistribution.sol && \
	solc --abi --pretty-json --overwrite -o staking staking/IStaking.sol && \
	solc --abi --pretty-json --overwrite -o wasm wasm/IWasm.sol && \
	cd ../x/bank/keeper && \
	solc --abi --pretty-json --overwrite -o . ./IERC20.sol

###############################################################################
###                                Docker                                   ###
###############################################################################
get-heighliner:
	go install github.com/strangelove-ventures/heighliner@latest
local-image:
ifeq (,$(shell which heighliner))
	echo 'heighliner' binary not found. Consider running `make get-heighliner`
else
	DOCKER_BUILDKIT=1 heighliner build -c $(NAME) --local --no-cache --dockerfile cosmos --build-target "make install" --pre-build "apk add --update --no-cache binutils-gold && ln -s /lib/libwasmvm_muslc.aarch64.a /lib/libwasmvm.aarch64.a" --binaries "/go/bin/xplad"
endif
