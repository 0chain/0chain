# TestNet Setup with Docker Containers

## Table of Contents

- [Initial Setup](#initial-setup) - [Directory Setup for Miners & Sharders](#directory-setup-for-miners-&-sharders) - [Setup Network](#setup-network)
- [Building and Starting the Nodes](#building-and-starting-the-nodes)
- [Generating Test Transactions](#generating-test-transactions)
- [Troubleshooting](#troubleshooting)
- [Debugging](#debugging)
- [Unit tests](#unittests)
- [Creating The Magic Block](#creating-the-magic-block)
- [Miscellaneous](#miscellaneous) - [Cleanup](#cleanup) - [View Change](docs/viewchange.md) - [Minio Setup](#minio)

## Initial Setup

### Directory Setup for Miners & Sharders

In the git/0chain run the following command

```
$ ./docker.local/bin/init.setup.sh
```

### Setup Network

Setup a network called testnet0 for each of these node containers to talk to each other.

**_Note: The config file should be providing the IP address of the nodes as per the IP addresses in this network._**

```
$ ./docker.local/bin/setup_network.sh
```

## Building the Nodes

1. Open 5 terminal tabs. Use the first one for building the containers by being in git/0chain directory. Use the next 3 for 3 miners and be in the respective miner directories created above in docker.local. Use the 5th terminal and be in the sharder1 directory.

1.1) First build the base containers, zchain_build_base and zchain_run_base

```
$ ./docker.local/bin/build.base.sh
```

2. Building the miners and sharders. From the git/0chain directory use

2.1) To build the miner containers

```
$ ./docker.local/bin/build.miners.sh
```

2.2) To build the sharder containers

```
$ ./docker.local/bin/build.sharders.sh
```

for building the 1 sharder.

2.3) Syncing time (the host and the containers are being offset by a few seconds that throws validation errors as we accept transactions that are within 5 seconds of creation). This step is needed periodically when you see the validation error.

```
$ ./docker.local/bin/sync_clock.sh
```

## Configuring the nodes

1. Use ./docker.local/config/0chain.yaml to configure the blockchain properties. The default options are setup for running the blockchain fast in development.

1.1) If you want the logs to appear on the console - change logging.console from false to true

1.2) If you want the debug statements in the logs to appear - change logging.level from 'info' to 'debug'

1.3) If you want to change the block size, set the value of server_chain.block.size

1.4) If you want to adjust the network relay time, set the value of network.relay_time

## Starting the nodes

1. Starting the nodes. On each of the miner terminals use the commands (note the .. at the beginning. This is because, these commands are run from within the docker.local/<miner/sharder|i> directories and the bin is one level above relative to these directories)

On the sharder terminal, use

```
$ ../bin/start.b0sharder.sh
```

Wait till the cassandra is started and the sharder is ready to listen to requests.

On the respective miner terminal, use

```
$ ../bin/start.b0miner.sh
```

## Setting up Cassandra Schema

The following is no longer required as the schema is automatically loaded.

Start the sharder service that also brings up the cassandra service. To run commands on cassandra, use the following command

```
$ ../bin/run.sharder.sh cassandra cqlsh
```

1. To create zerochain keyspace, do the following

```
$ ../bin/run.sharder.sh cassandra cqlsh -f /0chain/sql/zerochain_keyspace.sql
```

2. To create the tables, do the following

```
$ ../bin/run.sharder.sh cassandra cqlsh -k zerochain -f /0chain/sql/txn_summary.sql
```

3. When you want to truncate existing data (use caution), do the following

```
$ ../bin/run.sharder.sh cassandra cqlsh -k zerochain -f /0chain/sql/truncate_tables.sql
```

## Generating Test Transactions

There is no need to generate the test data separately. In development mode, the transaction data is automatically generated at a certain rate based on the block size.

However, you can use the <a href='https://github.com/0chain/block-explorer'>block explorer</a> to submit transactions, view the blocks and confirm the transactions.

## Monitoring the progress

1. Use <a href='https://github.com/0chain/block-explorer'>block explorer</a> to see the progress of the block chain.

2. In addition, use the '/\_diagnostics' link on any node to view internal details of the blockchain and the node.

## Troubleshooting

1. Ensure the port mapping is all correct:

```
$ docker ps
```

This should display a few containers and should include containers with images miner1_miner, miner2_miner and miner3_miner and they should have the ports mapped like "0.0.0.0:7071->7071/tcp"

2. Confirming the servers are up and running. From a browser, visit

- http://localhost:7071/_diagnostics

- http://localhost:7072/_diagnostics

- http://localhost:7073/_diagnostics

to see the status of the miners.

Similarly, following links can be used to see the status of the sharders

- http://localhost:7171/_diagnostics

- http://localhost:7172/_diagnostics

- http://localhost:7173/_diagnostics

3. Connecting to redis servers running within the containers (you are within the appropriate miner directories)

Default redis (used for clients and state):

```
$ ../bin/run.miner.sh redis redis-cli
```

Redis used for transactions:

```
$ ../bin/run.miner.sh redis_txns redis-cli
```

4. Connecting to cassandra used in the sharder (you are within the appropriate sharder directories)

```
$ ../bin/run.sharder.sh cassandra cqlsh
```

## Debugging

The logs of the nodes are stored in log directory (/0chain/log on the container and docker.local/miner|sharder[n]/log in the host). The 0chain.log contains all the logs related to the protocol and the n2n.log contains all the node to node communication logs. The typical issues that need to be debugged is errors in the log, why certain things have not happeend which requires reviewing the timestamp of a sequence of events in the network. Here is an example set of commands to do some debugging.

Find arrors in all the miner nodes (from git/0chain)

```
$ grep ERROR docker.local/miner*/log/0chain.log
```

This gives a set of errors in the log. Say an error indicates a problem for a specific block, say abc, then

```
$ grep abc docker.local/miner*/log/0chain.log
```

gives all the logs related to block 'abc'

To get the start time of all the rounds

```
$ grep 'starting round' docker.local/miner*/log/0chain.log
```

This gives the start timestamps that can be used to correlate the events and their timings.

## Unit tests

Unit tests can be run with `go test` outside of Docker if you have the correct C++ dependencies installed on your system.

```
$ cd code/go/src/0chain.net/my-pkg
$ go test
```

Otherwise, we have a Docker image which takes care of installing the build dependencies for you in an environment identical to our other Docker builds.

First build the base image.

```
$ ./docker.local/bin/build.base.sh
```

Then run the tests.

```
$ ./docker.local/bin/unit_test.sh [<packages>]
```

The list of packages is optional, and if provided runs only the tests from those packages. If no packages are specified, all unit tests are run.

## Creating The Magic Block

First build the magic block image.

```
$ ./docker.local/bin/build.magic_block.sh
```

Next, set the configuration file. To do this edit the docker.local/build.magicBlock/docker-compose.yml file. On line 13 is a flag "--config_file" set it to the magic block configuration file you want to use.

To create the magic block.

```
$ ./docker.local/bin/create.magic_block.sh
```

The magic block json file will appear in the docker.local/config under the name given in the configuration file.

## Miscellaneous

### Cleanup

1. If you want to restart the blockchain from the beginning

```
$ ./docker.local/bin/clean.sh
```

This cleans up the directories within docker.local/miner* and docker.local/sharder*

**_Note: this script can take a while if the blockchain generated a lot of blocks as the script deletes
the databases and also all the blocks that are stored by the sharders. Since each block is stored as a
separate file, deleting thousands of such files will take some time._**

2. If you want to get rid of old unused docker resources:

```
$ docker system prune
```

### Minio

- You can use the inbuild minio support to store blocks on cloud

You have to update minio_config file with the cloud creds data, The file can found at `docker.local/config/minio_config.txt`.
The following order is used for the content :

```
CONNECTION_URL
ACCESS_KEY_ID
SECRET_ACCESS_KEY
BUCKET_NAME
REGION
```

- Your minio config file is then used in the docker-compose while starting the sharder node

```
--minio_file config/minio_config.txt
```

- You can either update the setting in the same file which is given above or create a new one with you config and use that as

```
--minio_file config/your_new_minio_config_file.txt
```

\*\*\_Note: Do not forget to put the file in the same config folder OR mount your new folder.

- Apart from private connection config, There are other options as well in the 0chain.yaml file to manage minio settings.

Sample config

```
minio:
  # Enable or disable minio backup, Do not enable with deep scan ON
  enabled: false 
  # In Seconds, The frequency at which the worker should look for files, Ex: 3600 means it will run every 3600 seconds
  worker_frequency: 3600 
  # Number of workers to run in parallel, Just to make execution faster we can have mutiple workers running simultaneously
  num_workers: 5 
  # Use SSL for connection or not
  use_ssl: false 
  # How old the block should be to be considered for moving to cloud
  old_block_round_range: 20000000 
  # Delete local copy of block once it's moved to cloud
  delete_local_copy: true
```

# Integration tests

### Introduction

Integration tests uses RPC server which controls the launch and behavior of nodes.
The server called 'conductor' and placed in code/go/0chain.net/conductor/conductor.

All integration tests described in docker.local/config/conductor.yml. Comment out
some tests to disable them. Add more tests by needs.

To build miners and sharders for integration tests use

```
./docker.local/bin/build.sharders-integration-tests.sh && ./docker.local/bin/build.miners-integration-tests.sh
```

Note:

> Don't forget to rebuild miners and sharders again to return them to
> non-testing state after the tests.
>
> ```
> ./docker.local/bin/build.sharders.sh && ./docker.local/bin/build.miners.sh
> ```

### Start the tests

- Check out tests cases and configurations ./docker.local/config/conductor.yml.
- Check out ./docker.local/config/0chain.yml the 'integration_tests' part.
- Build miners and sharders, following instruction above.

#### Start test View Change 1

Test view change (part I).

```
./docker.local/bin/start.conductor.sh view-change-1
```

#### Start test View Change 2

Test view change (part II).

```
./docker.local/bin/start.conductor.sh view-change-2
```

#### Start test View Change 3

Test view change (part III).

```
./docker.local/bin/start.conductor.sh view-change-3
```

#### Blockchain 1

Test blockchain (part I, miners).

```
./docker.local/bin/start.conductor.sh blockchain-1
```

#### Blockchain 2

Test blockchain, (part II, sharders).

```
./docker.local/bin/start.conductor.sh blockchain-2
```

#### Blobber 1

##### Note

It's not recommended to start automated blobber tests, since they are unstable.
Sometimes, zwalelt/zbox transactions can't be confirmed due to, probably, some
problems in block worker, or another side.

The problem is zbox and zwallet commands sometimes fails with
```
<an error message> consensus_failed: consensus failed on sharders
```
without a real reason.

##### Prepare all.

Directories tree should be:

```
0chain/
blobber/
zboxcli/
zwalletcli/
blockWorker/
```

Otherwise, it requires corrections in tests, configurations, tests scripts and
following steps.

##### Prepare block worker.

The blockWorker should be patched. Go to the blockWorker directory and
apply patch provided in the 0chain repository. Make sure your haven't changes
you haven't commit yet in the blockWorker repository. Since, you can loose the
changes.

```
git apply --check ../0chain/docker.local/bin/conductor/block-worker-local.patch # check first
git apply ../0chain/docker.local/bin/conductor/block-worker-local.patch # and apply
```

To revert blockWorker repository to its latest commit state use
```
git reset --hard
git clean -f
```
That removes all changes and all new files.

##### Prepare blobbers.

Use the same approach to patch blobbers with
`../0chain/docker.local/bin/conductor/blobber-tests.patch`. And the same
approach to revert it.

##### Build all.

Build blockWorker, zbox and zwallet as usual. Blobbers will be build
automatically but after tests don't forget to rebuild them as usual.

##### Start tests.

Test blobbers. Note: the tests requires cleaning up blobbers and the blockWorker
that requires `sudo` password entering sometimes.

```
./docker.local/bin/start.conductor.sh blobber-1
```

And

```
./docker.local/bin/start.conductor.sh blobber-2
```

#### After all

Don't forget to rollback changes, clean and rebuild applications for
regular usage.
