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
TM_VERSION := $(shell go list -m github.com/tendermint/tendermint | sed 's:.* ::') # grab everything after the space in "github.com/tendermint/tendermint v0.34.7"

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
      build_tags += ledger
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
        build_tags += ledger
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
	  -X github.com/tendermint/tendermint/version.TMCoreSemVer=$(TM_VERSION)

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

all: install

install: go.sum
	go install -mod=readonly $(BUILD_FLAGS) ./cmd/xplad

build: go.sum
	go build -mod=readonly $(BUILD_FLAGS) -o build/xplad ./cmd/xplad

.PHONY: test
test: go.sum
	go test -short ./...

go.sum: go.mod
	@go mod verify
	@go mod tidy

###############################################################################
###                                Protobuf                                 ###
###############################################################################
PROTO_BUILDER_IMAGE=tendermintdev/sdk-proto-gen:v0.7
PROTO_FORMATTER_IMAGE=tendermintdev/docker-build-proto@sha256:aabcfe2fc19c31c0f198d4cd26393f5e5ca9502d7ea3feafbfe972448fee7cae

proto-all: proto-format proto-lint proto-gen

proto-gen:
	@echo "Generating Protobuf files"
	$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(PROTO_BUILDER_IMAGE) sh ./scripts/protocgen.sh

proto-format:
	@echo "Formatting Protobuf files"
	$(DOCKER) run --rm -v $(CURDIR):/workspace \
	--workdir /workspace $(PROTO_FORMATTER_IMAGE) \
	find ./ -name *.proto -exec clang-format -i {} \;

proto-swagger-gen:
	@./scripts/protoc-swagger-gen.sh

proto-lint:
	@$(DOCKER_BUF) lint --error-format=json