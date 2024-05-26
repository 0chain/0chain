# Makefile
# Specific features to manage automations in development & build processes.
#
# For usage on Windows see [Chocolatey CLI Documentation](https://docs.chocolatey.org/en-us/choco/setup)
# Then execute `choco install make` command in shell, now you will be able to use `make` on Windows.


ifeq ($(OS),Windows_NT)
    detected_OS := Windows
		detected_ARCH := x86_64
else
    detected_OS := $(shell sh -c 'uname 2>/dev/null || echo Unknown')
		detected_ARCH := $(shell sh -c 'uname -m 2>/dev/null || echo Unknown')
endif


make_path := $(abspath $(lastword $(MAKEFILE_LIST)))
root_path := $(patsubst %/, %, $(dir $(make_path)))
GOPATH := $(shell go env GOPATH)

.PHONY: pre-push go-mod check-commit
pre-push: go-mod check-commit

.PHONY: check-commit go-get run-test
check-commit: go-get run-test

.PHONY: install-mockery mockery install-msgp msgp build-mocks swagger

.PHONY: build-benchmark benchmark

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
	@cd $(root_path)/code/go/0chain.net && go test -tags bn256 -cover ./...
	@echo "Tests completed."

install-mockery:
	@([ -d ./tmp/mockery ] || mkdir -p ./tmp/mockery) \
	&& echo "[+]download mockery" \
	&& ([ -f ./tmp/mockery/mockery.tar.gz ] || curl -L -o ./tmp/mockery/mockery.tar.gz https://github.com/vektra/mockery/releases/download/v2.28.1/mockery_2.28.1_$(detected_OS)_$(detected_ARCH).tar.gz) \
	&& echo "[+]install mockery" \
	&& tar zxvfC ./tmp/mockery/mockery.tar.gz ./tmp/mockery \
	&& cp ./tmp/mockery/mockery $(GOPATH)/bin/ \
	&& rm -rf ./tmp
build-mocks:
	GOPATH=$(GOPATH) ./generate_mocks.sh

install-msgp:
	@echo "Install msgp..."
	./docker.local/bin/install.msgp.sh
	@echo "Msgp installed."

msgp:
	@echo "Run msgp..."
	@cd $(root_path)/code/go/0chain.net && go generate -run=msgp ./...
	@echo "Run msgp completed."

swagger-storage-sc:
	@echo "Run swagger for storage smart contract API ..."
	swagger generate spec -w code/go/0chain.net/ -c 0chain.net/smartcontract/storagesc -c 0chain.net/smartcontract/dbs -c 0chain.net/chaincore/... -c 0chain.net/core/... -x 0chain.net/chaincore/chain -m -o docs/swagger-storage-sc.yaml
	swagger generate markdown  -f docs/swagger-storage-sc.yaml --output=docs/storage-sc-api.md
	@echo "swagger documentation generated for storage smart contract API"

swagger-miner-sc:
	@echo "Run swagger for miner smart contract API ..."
	swagger generate spec -w code/go/0chain.net/ -c 0chain.net/smartcontract/minersc -c 0chain.net/smartcontract/dbs -c 0chain.net/smartcontract/rest -c 0chain.net/chaincore/... -x 0chain.net/chaincore/chain -c 0chain.net/core/... -m -o docs/swagger-miner-sc.yaml
	swagger generate markdown  -f docs/swagger-miner-sc.yaml --output=docs/miner-sc-api.md
	@echo "swagger documentation generated for miner smart contract API"

swagger-sharder:
	@echo "Run swagger for sharder API ..."
	swagger generate spec -w code/go/0chain.net/ -c 0chain.net/sharder -c 0chain.net/smartcontract/... -c 0chain.net/chaincore/... -c 0chain.net/core/... -m -o docs/swagger-sharder.yaml
	swagger generate markdown  -f docs/swagger-sharder.yaml --output=docs/swagger-sharder.md
	@echo "swagger documentation generated for sharder API"

swagger-miner:
	@echo "Run swagger for miner API ..."
	swagger generate spec -w code/go/0chain.net/ -c 0chain.net/miner -c 0chain.net/chaincore/... -m -o docs/swagger-miner.yaml
	swagger generate markdown  -f docs/swagger-miner.yaml --output=docs/swagger-miner.md
	@echo "swagger documentation generated for miner API"

swagger: swagger-sharder swagger-miner

build-benchmark:
	./docker.local/bin/build.benchmark.sh

benchmark:
	@cd $(root_path)/docker.local/benchmarks \
	&& ../bin/start.benchmarks.sh
