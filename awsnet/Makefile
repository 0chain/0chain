#!make
# Check that given variables are set and all have non-empty values,
# die with an error otherwise.
#
# Params:
#   1. Variable name(s) to test.
#   2. (optional) Error message to print.
check_defined = \
    $(strip $(foreach 1,$1, \
        $(call __check_defined,$1,$(strip $(value 2)))))

__check_defined = \
    $(if $(value $1),, \
        $(error Undefined $1$(if $2, ($2))$(if $(value @), \
                required by target `$@')))

# Check that a variable specified through the stem is defined and has
# a non-empty value, die with an error otherwise.
#
#   %: The name of the variable to test.
#
check-defined-% : __check_defined_FORCE
	@:$(call check_defined, $*, target-specific)

# Since pattern rules can't be listed as prerequisites of .PHONY,
# we use the old-school and hackish FORCE workaround.
# You could go without this, but otherwise a check can be missed
# in case a file named like `check-defined-...` exists in the root
# directory, e.g. left by an accidental `make -t` invocation.
.PHONY :

#Base directory of the AWSNET infrastructure. This directory would be a standalone directory
#that could be installed on an independent machine to install AWSNET.
AWSNET_BASE?=$(CURDIR)

COOKBOOK=$(AWSNET_BASE)/cookbook
CONTEXT=$(COOKBOOK)/context

include $(CONTEXT)

ZCHAIN_REPO=$(abspath $(AWSNET_BASE)/..)

ENVIRONMENT?=testnet

INVENTORY=$(COOKBOOK)/inventory
export EC2_INI_PATH=$(INVENTORY)/ec2.ini

#Credentials directory will store the actual key/access values.
CREDENTIAL=$(AWSNET_BASE)/credentials

CONFIG=$(AWSNET_BASE)/config

CONFIG_DIR=$(CONFIG)/network

CONFIG_FILE=$(CONFIG_DIR)/config.yml

#Destination directoy to store local assests
ZCHAIN_ARTIFACT?=$(AWSNET_BASE)/testnet/$(ZCHAIN_TESTNET)

NETWORK_TARGETS=network-setup network-destroy

REGIONS=regions-display

CONTROL_DIR=control-dir-setup


ZCHAIN=zchain-show

TARGETS:=$(CONTROL_DIR) $(SERVER) $(REGIONS) $(KEY_GOALS) $(EIP_TARGETS) $(ZCHAIN) $(ZCHAIN_SERVER)

0CHAIN_TARGETS:=

.PHONY: $(TARGETS) show __check_defined_FORCE
__check_defined_FORCE:

#The AWS_ACCESS_KEY and other related variables can be installed using aws-vault.
#This ensures all the access keys are kept safe.
export AWS_ACCESS_KEY_ID:=$(shell aws-vault exec 0chain -- env | grep AWS_ACCESS_KEY_ID | cut -d'=' -f2)
export AWS_SECRET_ACCESS_KEY:=$(shell aws-vault exec 0chain -- env | grep AWS_SECRET_ACCESS_KEY|cut -d'=' -f2)
export AWS_SESSION_TOKEN:=$(shell aws-vault exec 0chain -- env | grep AWS_SESSION_TOKEN | cut -d'=' -f2)
export AWS_SECURITY_TOKEN:=$(shell aws-vault exec 0chain -- env | grep AWS_SECURITY_TOKEN|cut -d'=' -f2)

PLAYBOOK=ansible-playbook

ROLES:=artifacts keyaws instance istart storage docker gitrepo clientid agentbuild agentrole istop

ROLE_TARGETS_LOCAL:=$(foreach role,$(ROLES),$(role)-assemble-local $(role)-teardown-local)
ROLE_TARGETS_REMOTE:=$(foreach role,$(ROLES),$(role)-assemble-remote $(role)-teardown-remote)
ROLE_TARGETS_ZCHAIN:=$(foreach role,$(ROLES),$(role)-assemble-zchain $(role)-teardown-zchain)


#Different variables set in Makefile are communicated to ansible.
EXTRA_VARS="aws_access_key_id=$(AWS_ACCESS_KEY_ID) \
	aws_secret_access_key=$(AWS_SECRET_ACCESS_KEY) \
	aws_session_token=$(AWS_SESSION_TOKEN) \
	aws_security_token=$(AWS_SECURITY_TOKEN) \
	network=$(ZCHAIN_TESTNET) \
	credential=$(CREDENTIAL) \
	repo=$(ZCHAIN_REPO) \
	zchain_artifact=$(ZCHAIN_ARTIFACT)"

ADMIN_PLAY=@$(PLAYBOOK) -i $(INVENTORY) $(COOKBOOK)/cook-admin.yml --extra-vars=$(EXTRA_VARS) --tags $@
LOCAL_PLAY=@$(PLAYBOOK) -i $(INVENTORY) $(COOKBOOK)/cook-local.yml --extra-vars=$(EXTRA_VARS) --tags $@
RAW_PLAY=@$(PLAYBOOK) -i $(INVENTORY) $(COOKBOOK)/cook-raw.yml --extra-vars=$(EXTRA_VARS) --tags $@
REMOTE_PLAY=@$(PLAYBOOK) -i $(INVENTORY) $(COOKBOOK)/cook-remote.yml --extra-vars=$(EXTRA_VARS) --tags $@
ZCHAIN_PLAY=@$(PLAYBOOK) -i $(INVENTORY) $(COOKBOOK)/cook-zchain.yml --extra-vars=$(EXTRA_VARS) --tags $@

zchain_testnet:|check-defined-ZCHAIN_TESTNET
zchain_testnet:|check-defined-AWS_ACCESS_KEY_ID
zchain_testnet:|check-defined-AWS_SECRET_ACCESS_KEY
zchain_testnet:|check-defined-AWS_SESSION_TOKEN
zchain_testnet:|check-defined-AWS_SECURITY_TOKEN


$(ROLE_TARGETS_LOCAL) $(ROLE_TARGETS_REMOTE): zchain_testnet

#Artifacts:
artifacts-assemble-local:
	$(LOCAL_PLAY)

artifacts-tear-down-local:
	$(LOCAL_PLAY)

#Key Role related targets.
#ADMIN - These are super-destructive roles. They should be executed only once
#when setting up the keys for the cluster. Once the keys have been installed,
#then don't update the keys.
keyaws-assemble-local:
	$(ADMIN_PLAY)

keyaws-teardown-local:
	$(ADMIN_PLAY)

#keygit-assemble-local keygit-teardown-local:
#	$(ADMIN_PLAY)


#Create an AWS instance.
instance-assemble-local:
	$(LOCAL_PLAY)

instance-teardown-local:
	$(LOCAL_PLAY)

#Instance the packages such as ansible, python, tree etc...
instance-assemble-remote:
	$(RAW_PLAY)

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
	$(ZCHAIN_PLAY)

gitrepo-teardown-zchain:
	$(ZCHAIN_PLAY)

#Create the local copy of the nodes file. Route 53 gets updated here as well.
agent-clientid-assemble-local:
	$(LOCAL_PLAY)

#Transport the nodes file and client keys file to the remote machines.
agent-clientid-assemble-zchain:
	$(ZCHAIN_PLAY)

#Remove records from Route53
agent-clientid-teardown-local:
	$(LOCAL_PLAY)

#Copy the configuration file.
agent-config-assemble-zchain:
	$(ZCHAIN_PLAY)

agent-stage-cluster: \
	gitrepo-assemble-local \
	gitrepo-assemble-zchain \
	agent-clientid-assemble-local \
	agent-clientid-assemble-zchain \
	agent-build-assemble-zchain

#Build the zchain_base, miner and sharder
agent-build-assemble-zchain:
	$(ZCHAIN_PLAY)

#Start the sharder and miner
agent-role-assemble-zchain:
	$(ZCHAIN_PLAY)

# Stop the sharder, miner
agent-role-teardown-zchain:
	$(ZCHAIN_PLAY)

#Instance management - start and stop
istop-teardown-local:
	$(LOCAL_PLAY)


$(TARGETS): show
	@echo "Make running target=$@"
	@$(PLAYBOOK) -i $(INVENTORY) $(CONFIG_FILE) --extra-vars="access=$(ACCESS) network=$(network) repo=$(ZCHAIN_REPO) network_dir=$(NETWORK_DIR)" --tags $@


workon_chinook:
	echo  "ZCHAIN_TESTNET=chinook" > $(CONTEXT)
workon_shasta:
	echo  "ZCHAIN_TESTNET=shasta" > $(CONTEXT)
workon_whitney:
	echo  "ZCHAIN_TESTNET=whitney" > $(CONTEXT)

show:
	@echo "ZCHAIN_TESTNET.......=$(ZCHAIN_TESTNET)"
	@echo "AWS_ACCESS_KEY_ID....=$(AWS_ACCESS_KEY_ID)"
	@echo "AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY)"
	@echo "AWS_SESSION_TOKEN....=$(AWS_SESSION_TOKEN)"
	@echo "AWS_SECURITY_TOKEN...=$(AWS_SECURITY_TOKEN)"
	@echo "ZCHAIN_REPO..........=$(ZCHAIN_REPO)"
	@echo "AWSNET_BASE..........=$(AWSNET_BASE)"
	@echo "COOKBOOK.............=$(COOKBOOK)"
	@echo "CREDENTIAL...........=$(CREDENTIAL)"
	@echo "EXTRA_VARS...........=$(EXTRA_VARS)"