# TestNet Setup with AWS and Docker Containers

## Table of Contents

-   [Step One: Context Setup for Cluster](#context-setup-for-cluster)
-   [Step Two: Stop Agents - Sharder + Miner](#stop-agents---sharder-+-miner)
-   [Step Three: Stage the Cluster](#upload-git-repository,-nodes-files-and-start-the-cluster)
-   [Step Four: Start the Agents](#start-the-agents)
-   [Step Five: Start Block Explore](#start-block-explorer-for-the-respective-cluster-and-issue-few-transactions)

## Context Setup for Cluster

Each cluster in the testnet configuration has a name and associated directory under awsnet/cookbook/anchor. 

Under the cluster specific directory there are three files:
- 0chain.yaml - 0chain configuration parameters
- blueprint.yml - Allocation of regions, miner, sharders, instance types etc.
- clientid.yml - Client Id, Public Key and Private key. This file is generated using the keygen util in the awsnet/util directory.

If you need to update the git/branch to be replicated check the blueprint.yaml.  

Setup the context for future commands by using the workon_<cluster-name> target. 

For example:

If you are going to manage "shasta", then issue "make workon_shasta". 

This command will create a file called "context", that will export the environment variable ZCHAIN_TESTNET=shasta for all future commands.

ZCHAIN_TESTNET=shasta

```
$ make workon_shasta
```

## STOP AGENTS - SHARDER + MINER

Before upgrading the cluster, ensure all the agents - ie sharder and miner are stopped. 

Issue the following command to stop all the agents and their containers.

```
$ make agent-role-teardown-zchain
```

## Upload git repository, nodes files and start the cluster

The command 'make agent-stage-cluster' will do the following actions:

   
- gitrepo-assemble-local: Create a zip file from the repository. The git branch/tag version is listed under "blueprint.yml"
- artifacts-assemble-remote - Create the necessary directories on the remote cluster under "/0chain"
- gitrepo-assemble-zchain - Copy the repository tar file to the clusters and untar them
- agent-clientid-assemble-local - Create a master copy of the clientid and nodes file. Update AWS Route53 with IP and DNS names
- agent-clientid-assemble-zchain - Install nodes.txt and clientid files on the remote cluster
- agent-config-assemble-zchain - Install cluster specific configuration file 0chain.yaml file. 
- agent-build-assemble-zchain - Build zchain_base, miner and sharder docker images. <br> This step is the longest step. It can take upwards of 45 minutes for this 
step to complete for the first time.

```
$ make agent-stage-cluster
```

## Start the agents
The command 'agent-role-asemble-zchain' will start the sharder and miner. It runs the scripts listed under docker.aws.

```
$ make agent-role-assemble-zchain
```

## Start Block explorer for the respective cluster and issue few transactions

Download and run the block-explore with the settings file for that cluster. 

For example to run block-explorer for shasta, run the following command

```
$ meteor --settings ./.deploy/shasta-settings.json
```

