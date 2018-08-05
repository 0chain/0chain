# TestNet Setup with AWS and Docker Containers

## Table of Contents

- [Initial Setup](#initial-setup)
	- [Directory Setup for Miners & Sharders](#directory-setup-for-miners-&-sharders)
	- [Setup Network](#setup-network)
- [Building and Starting the Nodes](#building-and-starting-the-nodes)
- [Generating Test Transactions](#generating-test-transactions)
- [Troubleshooting](#troubleshooting)
- [Debugging](#debugging)
- [Miscellaneous](#miscellaneous)
	- [Cleanup](#cleanup)

## Initial Setup

### Context Setup for Cluster

Each cluster has a name such as chinook, eddy, shasta or whitney. Setup the context for future commands by using the workon_<cluster-name> target. If you are going to manage "shasta", then issue "workon_shasta". This command will create a file called "context" that will export the environment variable ZCHAIN_TESTNET. Here are the contents of the file when workon_shasta is invoked.

ZCHAIN_TESTNET=shasta

```
$ make workon_shasta
```

### Stop the sharder and miner
The sharder and miner are addressed as agents in the Makefile and the cookbook. To stop all the existing agents, issue the command agent-role-teardown-zchain.

```
$ make agent-role-teardown-zchain
```

## Upload git repository, nodes files and start the cluster

The git branch or tag version should be updated in the respective "blueprint.yaml" under awsnet/cook/anchor/<cluster-name>/blueprint.yaml.


```
$ make agent-stage-cluster
```

The "agent-stage-cluster" is comprised of following steps.

gitrepo-assemble-local
artifacts-assemble-remote
gitrepo-assemble-zchain
agent-clientid-assemble-local
agent-clientid-assemble-zchain
agent-config-assemble-zchain
agent-build-assemble-zchain

## GITREPO-ASSEMBLE-LOCAL

```
$ make gitrepo-assemble-local 
```
## ARTIFACTS-ASSEMBLE-REMOTE
```
$ make gitrepo-assemble-local 
```

## GITREPO-ASSEMBLE-ZCHAIN
```
$ make gitrepo-assemble-local 
```

## AGENT-CLIENTID-ASSEMBlE-LOCAL
```
$ make agent-clientid-assemble-local
```
## AGENT-CLIENTID-ASSEMBLE-ZCHAIN
```
$ make agent-clientid-assemble-zchain
```
## AGENT-CONFIG-ASSEMBLE-ZCHAIN
```
$ make agent-config-assemble-zchain
```
## AGENT-BUILD-ASSEMBlE-ZCHAIN
```
$ make agent-build-assemble-zchain
```

