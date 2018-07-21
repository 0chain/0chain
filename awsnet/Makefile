#Base directory of the AWSNET infrastructure. This directory would be a standalone directory
#that could be installed on an independent machine to install AWSNET.
AWSNET_BASE?=$(CURDIR)
COOKBOOK=$(AWSNET_BASE)/cookbook
ZCHAIN_REPO=$(abspath $(AWSNET_BASE)/..)

ENVIRONMENT?=testnet

INVENTORY=$(COOKBOOK)/inventory

#Credentials directory will store the actual key/access values.
CREDENTIAL=$(AWSNET_BASE)/credentials

#Create a testnet with the following name
network?=whitney

CONFIG=$(AWSNET_BASE)/config

CONFIG_DIR=$(CONFIG)/network

CONFIG_FILE=$(CONFIG_DIR)/config.yml

NETWORK_DIR=$(AWSNET_BASE)/$(network)

NETWORK_TARGETS=network-setup network-destroy

REGIONS=regions-display

CONTROL_DIR=control-dir-setup

KEYADMIN=keyadmin-aws-create keyadmin-aws-delete keyadmin-git-create keyadmin-git-delete
#Create the key for the different subsystems
keyadmin-assemble: keyadmin-aws-create keyadmin-git-create
keyadmin-teardown: keyadmin-aws-delete keyadmin-git-delete

LOCAL_INSTANCE_GOALS=instance-assemble instance-teardown

#Create the instances...
testnet-assemble: keyadmin-assemble instance-assemble instance-prime docker-install docker-hello
testnet-teardown: instance-terminate keyadmin-teardown

REMOTE_DOCKER_GOAL=docker-assemble docker-install docker-hello

docker-assemble: docker-install

#All instance setup goals are done using the local connection.
LOCAL_INSTANCE=$(KEYADMIN) $(INSTANCE) $(DOCKER_GOAL)
$(LOCAL_INSTANCE): show
	@echo "Make target=$@"
	@$(PLAYBOOK) -i $(INVENTORY) $(COOKBOOK)/config-local-instance.yml -l localhost --connection=local --extra-vars=$(EXTRA_VARS) --tags $@

REMOTE_INSTANCE_GOALS=instance-prime $(REMOTE_DOCKER_GOAL)
$(REMOTE_INSTANCE_GOALS): show
	@echo "Make target=$@"
	@$(PLAYBOOK) -i $(INVENTORY) $(COOKBOOK)/config-remote-instance.yml --extra-vars=$(EXTRA_VARS) --tags $@



ZCHAIN_SERVER=zchain-motd

EIP_TARGETS=eip-create eip-list-all

ZCHAIN=zchain-show


#testnet-launch: key-create server-launch server-prime docker-install docker-hello
#
#testnet-reset: key-delete
#
#testnet-teardown: server-terminate

#WORKFLOW=testnet-launch testnet-reset testnet-teardown

TARGETS:=$(CONTROL_DIR) $(SERVER) $(REGIONS) $(KEY_GOALS) $(EIP_TARGETS) $(ZCHAIN) $(ZCHAIN_SERVER) $(DOCKER_SETUP) $(WORKFLOW)

0CHAIN_TARGETS:=

.PHONY: $(TARGETS) netgen clean show

#Executable to be invoked.
export AWS_ACCESS_KEY_ID=AKIAIEP4LDUAY4FNU62A
export AWS_SECRET_ACCESS_KEY=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

PLAYBOOK=ansible-playbook
EXTRA_VARS="credential=$(CREDENTIAL) network=$(network) repo=$(ZCHAIN_REPO)"

$(AWS_TARGETS): show
	@echo "Make target=$@"
	@$(PLAYBOOK) -i $(INVENTORY) $(COOKBOOK)/config-aws.yml -l localhost --connection=local --extra-vars=$(EXTRA_VARS) --tags $@

$(TARGETS): show
	@echo "Make running target=$@"
	@$(PLAYBOOK) -i $(INVENTORY) $(CONFIG_FILE) --extra-vars="access=$(ACCESS) network=$(network) repo=$(ZCHAIN_REPO) network_dir=$(NETWORK_DIR)" --tags $@


netgen: show
	@echo "Generate configuration files for 0chain network=$(network)..."
	@$(APB) -i $(CONFIG_DIR)/hosts $(CONFIG_FILE) --extra-vars

clean:


show:
	@echo "ZCHAIN_REPO..=$(ZCHAIN_REPO)"
	@echo "AWSNET_BASE..=$(AWSNET_BASE)"
	@echo "COOKBOOK.....=$(COOKBOOK)"
	@echo "CREDENTIAL...=$(CREDENTIAL)"
