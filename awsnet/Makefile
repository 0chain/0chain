#Base directory of the AWSNET infrastructure. This directory would be a standalone directory
#that could be installed on an independent machine to install AWSNET.
AWSNET_BASE?=$(CURDIR)
COOKBOOK=$(AWSNET_BASE)/cookbook
ZCHAIN_REPO=$(abspath $(AWSNET_BASE)/..)

ENVIRONMENT?=testnet

INVENTORY=$(COOKBOOK)/inventory
export EC2_INI_PATH=$(INVENTORY)/ec2.ini

#Credentials directory will store the actual key/access values.
CREDENTIAL=$(AWSNET_BASE)/credentials

#Create a testnet with the following name
network?=whitney

CONFIG=$(AWSNET_BASE)/config

CONFIG_DIR=$(CONFIG)/network

CONFIG_FILE=$(CONFIG_DIR)/config.yml

#Destination directoy to store local assests
ZCHAIN_ARTIFACT?=$(AWSNET_BASE)/testnet/$(network)

NETWORK_TARGETS=network-setup network-destroy

REGIONS=regions-display

CONTROL_DIR=control-dir-setup

ADMIN_KEY=keyadmin-aws-create keyadmin-aws-delete
#ADMIN_INSTANCE=instance-assemble instance-teardown
#Create the key for the different subsystems
keyadmin-assemble: keyadmin-aws-create
keyadmin-teardown: keyadmin-aws-delete


#Create the instances...
testnet-assemble: keyadmin-assemble instance-assemble instance-prime docker-install docker-hello
testnet-teardown: instance-terminate keyadmin-teardown

REMOTE_DOCKER_GOAL=docker-assemble docker-install docker-hello

LOCAL_ARTIFACTS=artificats-assemble-local artifacts-teardown-local
REMOTE_ARTIFACTS=artifacts-assemble-remote artifacts-teardown-remote

docker-assemble: docker-install

#All instance setup goals are done using the local connection.
EXTRA_VARS="credential=$(CREDENTIAL) network=$(network) repo=$(ZCHAIN_REPO) zchain_artifact=$(ZCHAIN_ARTIFACT)"

LOCAL_INSTANCE_GOALS=$(ADMIN_KEY) $(ADMIN_INSTANCE) artifacts-local
$(LOCAL_INSTANCE_GOALS): show
	@echo "Make target=$@"
	$(PLAYBOOK) -i $(INVENTORY) $(COOKBOOK)/config-local-instance.yml -l localhost --connection=local --extra-vars=$(EXTRA_VARS) --tags $@

REMOTE_INSTANCE_GOALS=instance-prime $(REMOTE_DOCKER_GOAL)
$(REMOTE_INSTANCE_GOALS): show
	@echo "Make target=$@"
	@$(PLAYBOOK) -i $(INVENTORY) $(COOKBOOK)/config-remote-instance.yml --extra-vars=$(EXTRA_VARS) --tags $@

ZCHAIN_INSTANCE_GOALS=artifacts-zchain
$(ZCHAIN_INSTANCE_GOALS): show
	@echo "Make target=$@"
	@$(PLAYBOOK) -i $(INVENTORY) $(COOKBOOK)/config-zchain-instance.yml --extra-vars=$(EXTRA_VARS) --tags $@


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

ROLES=artifacts gitrepo instances clientid

ROLE_TARGETS_LOCAL:=$(foreach role,$(ROLES),$(role)-assemble-local $(role)-teardown-local)
ROLE_TARGETS_REMOTE:=$(foreach role,$(ROLES),$(role)-assemble-remote $(role)-teardown-remote)
ROLE_TARGETS_ZCHAIN:=$(foreach role,$(ROLES),$(role)-assemble-zchain $(role)-teardown-zchain)

$(ROLE_TARGETS_LOCAL):
	@$(PLAYBOOK) -i $(INVENTORY) $(COOKBOOK)/cook-local.yml --extra-vars=$(EXTRA_VARS) --tags $@

$(ROLE_TARGETS_REMOTE):
	@$(PLAYBOOK) -i $(INVENTORY) $(COOKBOOK)/cook-remote.yml --extra-vars=$(EXTRA_VARS) --tags $@

$(ROLE_TARGETS_ZCHAIN):
	@$(PLAYBOOK) -i $(INVENTORY) $(COOKBOOK)/cook-zchain.yml --extra-vars=$(EXTRA_VARS) --tags $@

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
	@echo "CREDENTIAL....=$(CREDENTIAL)"
	@echo "ROLE_TARGETS..=$(ROLE_TARGETS)"