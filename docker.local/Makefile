OS = $(shell uname | tr A-Z a-z)
export PATH := $(abspath bin/):${PATH}
VERBOSE = 1

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: init_setup network_setup build_base build_miner build_sharder sync_clock cleanup miner sharder

init_setup: ## 1.Init setup
	@cd .. && ./docker.local/bin/init.setup.sh

network_setup: ## 2.Network setup
	@cd .. && ./docker.local/bin/setup_network.sh

build_base: ## 3.Build Base Image
	@cd .. && ./docker.local/bin/build.base.sh

build_miner: ## 4.Build miners
	@cd .. && ./docker.local/bin/build.miners.sh

build_sharder: ## 5.Build sharders
	@cd .. && ./docker.local/bin/build.sharders.sh

sync_clock: ## 6.Sync clock
	@cd .. && ./docker.local/bin/sync_clock.sh

cleanup: ## Cleanup
	@cd .. && ./docker.local/bin/clean.sh

miner: ## Run miner num (=miner number)
	@cd miner$(num) && ../bin/start.b0miner.sh

sharder: ## Run sharder num (=miner number)
	@cd sharder$(num) && ../bin/start.b0sharder.sh