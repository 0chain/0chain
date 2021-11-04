# Makefile
# Specific features to manage automations in development & build processes.
#
# For usage on Windows see [Chocolatey CLI Documentation](https://docs.chocolatey.org/en-us/choco/setup)
# Then execute `choco install make` command in shell, now you will be able to use `make` on Windows.

make_path := $(abspath $(lastword $(MAKEFILE_LIST)))
root_path := $(patsubst %/, %, $(dir $(make_path)))

PROJECT := 0chain/miner # used for cache directory
HTTPS_GIT := https://github.com/0chain/0chain.git
SSH_GIT := git@github.com:0chain/0chain.git

BUF_VERSION := 1.0.0-rc4
BUF_INSTALL_FROM_SOURCE := false

UNAME_OS := $(shell uname -s)
UNAME_ARCH := $(shell uname -m)
CACHE_BASE := $(HOME)/.cache/$(PROJECT)
CACHE := $(CACHE_BASE)/$(UNAME_OS)/$(UNAME_ARCH)
CACHE_BIN := $(CACHE)/bin
CACHE_VERSIONS := $(CACHE)/versions

export PATH := $(abspath $(CACHE_BIN)):$(PATH)
export GOBIN := $(abspath $(CACHE_BIN))
export GO111MODULE := on

# BUF points to the marker file for the installed version.
#
# If BUF_VERSION is changed, the binary will be re-downloaded.
BUF := $(CACHE_VERSIONS)/buf/$(BUF_VERSION)
$(BUF):
	@rm -f $(CACHE_BIN)/buf
	@mkdir -p $(CACHE_BIN)
ifeq ($(BUF_INSTALL_FROM_SOURCE),true)
	$(eval BUF_TMP := $(shell mktemp -d))
	cd $(BUF_TMP); go get github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)
	@rm -rf $(BUF_TMP)
else
	curl -sSL \
		"https://github.com/bufbuild/buf/releases/download/v$(BUF_VERSION)/buf-$(UNAME_OS)-$(UNAME_ARCH)" \
		-o "$(CACHE_BIN)/buf"
	chmod +x "$(CACHE_BIN)/buf"
endif
	@rm -rf $(dir $(BUF))
	@mkdir -p $(dir $(BUF))
	@touch $(BUF)

.DEFAULT_GOAL := local

# deps allows us to install deps without running any checks.

.PHONY: deps
deps: $(BUF)

# local is what we run when testing locally.
# This does breaking change detection against our local git repository.

.PHONY: local
local: $(BUF)
	buf lint

# https is what we run when testing in most CI providers.
# This does breaking change detection against our remote HTTPS git repository.

.PHONY: https
https: $(BUF)
	buf lint
	buf breaking --against "$(HTTPS_GIT)#branch=master"

# ssh is what we run when testing in CI providers that provide ssh public key authentication.
# This does breaking change detection against our remote HTTPS ssh repository.
# This is especially useful for private repositories.

.PHONY: ssh
ssh: $(BUF)
	buf lint
	buf breaking --against "$(SSH_GIT)#branch=master"

# clean deletes any files not checked in and the cache for all platforms.

.PHONY: clean
clean:
	git clean -xdf
	rm -rf $(CACHE_BASE)

# For updating this repository

.PHONY: updateversion
updateversion:
ifndef VERSION
	$(error "VERSION must be set")
else
ifeq ($(UNAME_OS),Darwin)
	sed -i '' "s/BUF_VERSION := [0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/BUF_VERSION := $(VERSION)/g" Makefile
else
	sed -i "s/BUF_VERSION := [0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/BUF_VERSION := $(VERSION)/g" Makefile
endif
endif


.PHONY: pre-push go-mod check-commit
pre-push: go-mod check-commit

.PHONY: check-commit go-get run-test
check-commit: go-get run-test

go-mod:
	@echo "Prepare Go mod files..."
	@cd $(root_path)/code/go/0chain.net && go mod tidy -v
	@cd $(root_path)/code/go/0chain.net && go mod download -x
	@echo "Go mod files completed."

go-get:
	@echo "Load dependencies..."
	@cd $(root_path)/code/go/0chain.net && go get -d ./...
	@echo "Dependencies loaded."

run-test:
	@echo "Start testing..."
	@cd $(root_path)/code/go/0chain.net && go test -tags "bn256 development" -cover ./...
	@echo "Tests completed."
