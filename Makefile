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

.PHONY: pre-push go-mod check-commit
pre-push: go-mod check-commit

.PHONY: check-commit go-get run-test
check-commit: go-get run-test

.PHONY: install-mockery mockery

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
	@[ -d ./bin/mockery ] || mkdir -p ./bin/mockery
	@echo "[+]download mockery" 
	@[ -f ./bin/mockery/mockery.tar.gz ] || curl -L -o ./bin/mockery/mockery.tar.gz https://github.com/vektra/mockery/releases/download/v2.12.2/mockery_2.12.2_$(detected_OS)_$(detected_ARCH).tar.gz
	@echo "[+]install mockery"
	@[ -f "$(GOPATH)/bin/mockery" ] || tar zxvfC ./bin/mockery/mockery.tar.gz ./bin/mockery 
	@cp ./bin/mockery/mockery $(GOPATH)/bin/
	@rm -rf ./bin/mockery/mockery

build-mocks: install-mockery
	@echo "Making mocks..."
	@echo "-------------------------------------"
	@echo "[+] core	" 
	@echo "-------------------------------------"
	@cd $(root_path)/code/go/0chain.net/core && mockery --case underscore --output=../core/mocks --all

	@echo "-------------------------------------"
	@echo "[+] miner	" 
	@echo "-------------------------------------"
	@cd $(root_path)/code/go/0chain.net/miner && mockery --case underscore --output=../miner/mocks --all

	@echo "-------------------------------------"
	@echo "[+] chaincore	" 
	@echo "-------------------------------------"
	@cd $(root_path)/code/go/0chain.net/chaincore && mockery --case underscore --output=../chaincore/mocks --all

	@echo "-------------------------------------"
	@echo "[+] conductor	" 
	@echo "-------------------------------------"
	@cd $(root_path)/code/go/0chain.net/conductor && mockery --case underscore --output=../conductor/mocks --all

	@echo "-------------------------------------"
	@echo "[+] sharder	" 
	@echo "-------------------------------------"
	@cd $(root_path)/code/go/0chain.net/sharder && mockery --case underscore --output=../sharder/mocks --all

	@echo "-------------------------------------"
	@echo "[+] smartcontract	" 
	@echo "-------------------------------------"
	@cd $(root_path)/code/go/0chain.net/smartcontract && mockery --case underscore --output=../smartcontract/mocks --all

	@echo "-------------------------------------"
	@echo "[+] ./...	" 
	@echo "-------------------------------------"
	@cd $(root_path)/code/go/0chain.net && go generate -run="mockery" ./...

	@echo "Mocks files are generated."

install-msgp:
	@echo "Install msgp..."
	./docker.local/bin/install.msgp.sh
	@echo "Msgp installed."

msgp:
	@echo "Run msgp..."
	@cd $(root_path)/code/go/0chain.net && go generate -run=msgp ./...
	@echo "Run msgp completed."

swagger:
	@echo "Run swagger..."
	swagger generate spec -w  code/go/0chain.net/sharder/sharder  -m  -o docs/swagger.yaml
	swagger generate markdown  -f docs/swagger.yaml --output=docs/swagger.md
	@echo "swagger documentation generated"

