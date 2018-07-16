AWSNET_BASE?=$(CURDIR)

ZCHAIN_BASE=$(abspath $(AWSNET_BASE)/..)

network?=whitney

APB?=ansible-playbook

CONFIG=$(AWSNET_BASE)/config

ACCESS=$(AWSNET_BASE)/access

CONFIG_DIR=$(CONFIG)/network

CONFIG_FILE=$(CONFIG_DIR)/config.yml

NETWORK_DIR=$(AWSNET_BASE)/$(network)

NETWORK_TARGETS=network-setup network-destroy

REGIONS=regions-display

CONTROL_DIR=control-dir-setup

SERVER_LAUNCH=server-launch

SERVER_TERMINATE=server-terminate

SERVER:=$(SERVER_LAUNCH) $(SERVER_TERMINATE)

ZCHAIN_SERVER=zchain-motd

ACCESS_TARGETS=key-create key-region-add key-region-remove key-file-delete

EIP_TARGETS=eip-create eip-list-all

ZCHAIN=zchain-show

TARGETS:=$(CONTROL_DIR) $(SERVER) $(REGIONS) $(ACCESS_TARGETS) $(EIP_TARGETS) $(ZCHAIN) $(ZCHAIN_SERVER)

keys-install: key-create key-region-add

keys-uninstall: key-region-remove key-file-delete

.PHONY: $(TARGETS) netgen clean show

$(TARGETS): show
	@echo "Make running target=$@"
	@$(APB) -i $(CONFIG_DIR)/hosts $(CONFIG_FILE) --extra-vars="access=$(ACCESS) network=$(network) zchain_base=$(ZCHAIN_BASE) network_dir=$(NETWORK_DIR)" --tags $@

netgen: show
	@echo "Generate configuration files for 0chain network=$(network)..."
	@$(APB) -i $(CONFIG_DIR)/hosts $(CONFIG_FILE) --extra-vars

clean:


show:
	@echo "0CHAIN_BASE=$(ZCHAIN_BASE)"
	@echo "AWSNET_BASE=$(AWSNET_BASE)"
	@echo "CONFIG=$(CONFIG)"
	@echo "ACCESS=$(ACCESS)"
	@echo "CONFIG_FILE=$(CONFIG_FILE)"
	@echo "NETWORK_DIR=$(NETWORK_DIR)"
