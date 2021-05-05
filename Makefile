# Makefile
# Specific features to manage automations in development & build processes.
#
# For usage on Windows see [Chocolatey CLI Documentation](https://docs.chocolatey.org/en-us/choco/setup)
# Then execute `choco install make` command in shell, now you will be able to use `make` on Windows.

make_path := $(abspath $(lastword $(MAKEFILE_LIST)))
root_path := $(patsubst %/, %, $(dir $(make_path)))

.PHONY: pre-push go-mod check-commit
pre-push: go-mod check-commit

go-mod:
	@echo "Prepare Go mod files..."
	@cd $(root_path)/code/go/0chain.net && go mod tidy -v
	@cd $(root_path)/code/go/0chain.net && go mod download -x
	@echo "Go mod files completed."

check-commit:
	@echo "Load dependencies..."
	@cd $(root_path)/code/go/0chain.net && go get -d ./...
	@echo "Dependencies loaded."
	@echo "Start testing..."
	@cd $(root_path)/code/go/0chain.net && go test ./...
	@echo "Tests completed."
