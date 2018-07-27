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
network?=chinook

CONFIG=$(AWSNET_BASE)/config

CONFIG_DIR=$(CONFIG)/network

CONFIG_FILE=$(CONFIG_DIR)/config.yml

#Destination directoy to store local assests
ZCHAIN_ARTIFACT?=$(AWSNET_BASE)/testnet/$(network)

NETWORK_TARGETS=network-setup network-destroy

REGIONS=regions-display

CONTROL_DIR=control-dir-setup

#Different variables set in Makefile are communicated to ansible.
EXTRA_VARS="credential=$(CREDENTIAL) network=$(network) repo=$(ZCHAIN_REPO) zchain_artifact=$(ZCHAIN_ARTIFACT)"

ZCHAIN=zchain-show

TARGETS:=$(CONTROL_DIR) $(SERVER) $(REGIONS) $(KEY_GOALS) $(EIP_TARGETS) $(ZCHAIN) $(ZCHAIN_SERVER)

0CHAIN_TARGETS:=

.PHONY: $(TARGETS) show

#Executable to be invoked.
export AWS_ACCESS_KEY_ID=AKIAIEP4LDUAY4FNU62A
export AWS_SECRET_ACCESS_KEY=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

PLAYBOOK=ansible-playbook

ROLE_TARGETS_LOCAL:=$(foreach role,$(ROLES),$(role)-assemble-local $(role)-teardown-local)
ROLE_TARGETS_REMOTE:=$(foreach role,$(ROLES),$(role)-assemble-remote $(role)-teardown-remote)
ROLE_TARGETS_ZCHAIN:=$(foreach role,$(ROLES),$(role)-assemble-zchain $(role)-teardown-zchain)

LOCAL_PLAY=@$(PLAYBOOK) -i $(INVENTORY) $(COOKBOOK)/cook-local.yml --extra-vars=$(EXTRA_VARS) --tags $@
REMOTE_PLAY=@$(PLAYBOOK) -i $(INVENTORY) $(COOKBOOK)/cook-remote.yml --extra-vars=$(EXTRA_VARS) --tags $@

#Artifacts:
artifacts-assemble-local artifacts-teardown-local:
	$(LOCAL_PLAY)

#Key Role related targets
keyaws-assemble-local keyaws-teardown-local keygit-assemble-local keygit-teardown-local:
	$(LOCAL_PLAY)

#Create an AWS instance.
instance-assemble-local instance-teardown-local:
	$(LOCAL_PLAY)

#Instance the packages such as ansible, python, tree etc...
instance-assemble-remote:
	$(REMOTE_PLAY)

istart-assemble-local:
	$(LOCAL_PLAY)

#Create the EBS volume and mount it for reboot.
storage-assemble-remote:
	$(REMOTE_PLAY)

#Install docker related routines on the remote machine
docker-assemble-remote:
	$(REMOTE_PLAY)

#Create the remote directories such as /0chain
artifacts-assemble-remote:
	$(REMOTE_PLAY)

#Zip the git repo locally.
gitrepo-assemble-local:
	$(LOCAL_PLAY)

#Move the entire git repo to the remote machine and install it
gitrepo-assemble-zchain:
	$(REMOTE_PLAY)

#Create the local copy of the nodes file. Route 53 gets updated here as well.
clientid-assemble-local:
	$(LOCAL_PLAY)

#Transport the nodes file and client keys file to the remote machines.
clientid-assemble-remote:
	$(REMOTE_PLAY)

#Remove records from Route53
clientid-teardown-local:
	$(LOCAL_PLAY)

#Build the zchain_base, miner and sharder
agentbuild-assemble-remote:
	$(REMOTE_PLAY)

#Start the sharder and miner
agentrole-assemble-remote:
	$(REMOTE_PLAY)

# Stop the sharder, miner
agentrole-teardown-remote:
	$(REMOTE_PLAY)

#Instance management - start and stop
istop-teardown-local:
	$(LOCAL_PLAY)


$(TARGETS): show
	@echo "Make running target=$@"
	@$(PLAYBOOK) -i $(INVENTORY) $(CONFIG_FILE) --extra-vars="access=$(ACCESS) network=$(network) repo=$(ZCHAIN_REPO) network_dir=$(NETWORK_DIR)" --tags $@


show:
	@echo "ZCHAIN_REPO..=$(ZCHAIN_REPO)"
	@echo "AWSNET_BASE..=$(AWSNET_BASE)"
	@echo "COOKBOOK.....=$(COOKBOOK)"
	@echo "CREDENTIAL....=$(CREDENTIAL)"
	@echo "ROLE_TARGETS..=$(ROLE_TARGETS)"