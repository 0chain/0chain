# Makefile
# Specific features to manage automations in development & build processes.
#
# For usage on Windows see [Chocolatey CLI Documentation](https://docs.chocolatey.org/en-us/choco/setup)
# Then execute `choco install make` command in shell, now you will be able to use `make` on Windows.

make_path := $(abspath $(lastword $(MAKEFILE_LIST)))
root_path := $(patsubst %/, %, $(dir $(make_path)))

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
	@cd $(root_path)/code/go/0chain.net && go test -tags bn256 -cover ./...
	@echo "Tests completed."

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
	swagger generate markdown  -f docs/swagger.yaml
	mv markdown.md docs/swagger.md
	@echo "swagger documentation generated"

