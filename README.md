# TestNet Setup with Docker Containers

## Table of Contents

- [Initial Setup](#initial-setup) 
  - [Directory Setup for Miners & Sharders](#directory-setup-for-miners-and-sharders) 
  - [Setup Network](#setup-network)
- [Building and Starting the Nodes](#building-the-nodes)
- [Building the Nodes](#building-the-nodes)  
- [Generating Test Transactions](#generating-test-transactions)
- [Troubleshooting](#troubleshooting)
- [Debugging](#debugging)
- [Unit tests](#unit-tests)
- [Creating The Magic Block](#creating-the-magic-block)
- [Initial states](#initial-states)
- [Miscellaneous](#miscellaneous) 
  - [Cleanup](#cleanup)
  - [Minio Setup](#minio)
- [Run 0chain on ec2 / vm / bare metal](https://github.com/0chain/0chain/blob/master/docker.aws/README.md)
- [Run 0chain on ec2 / vm / bare metal over https](https://github.com/0chain/0chain/blob/master/https/README.md)

## Initial Setup

### Directory Setup for Miners and Sharders

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

1. Use `./docker.local/config/0chain.yaml` to configure the blockchain properties. The default options are setup for running the blockchain fast in development.

1.1) If you want the logs to appear on the console - change `logging.console` from `false` to `true`

1.2) If you want the debug statements in the logs to appear - change `logging.level` from `"info"` to `"debug"`

1.3) If you want to change the block size, set the value of `server_chain.block.size`

1.4) If you want to adjust the network relay time, set the value of `network.relay_time`

**_Note: Remove sharder72 and miner75 from docker.local/config/b0snode2_keys.txt and docker.local/config/b0mnode5_keys.txt respectively if you are joining to local network._**

## Starting the nodes

1. Starting the nodes. On each of the miner terminals use the commands (note the `..` at the beginning. This is because, these commands are run from within the `docker.local/<miner/sharder|i>` directories and the `bin` is one level above relative to these directories)

Start sharder first because miners need the genesis magic block. On the sharder terminal, use

```
$ ../bin/start.b0sharder.sh
```

Wait till the cassandra is started and the sharder is ready to listen to requests.

On the respective miner terminal, use

```
$ ../bin/start.b0miner.sh
```

## Re-starting the nodes

To reflect a change in config files 0chain.yaml and sc.yaml, just restart the miner or sharder to take the new configuration. If you're doing a code change locally or pulling updates from GitHub, you need to build.
```
git pull
docker.local/bin/build.base.sh && docker.local/bin/build.sharders.sh && docker.local/bin/build.miners.sh
```
For existing code and if you have tried running once, make sure there are no previous files and processes.
```
docker stop $(docker ps -a -q)
docker.local/bin/clean.sh
docker.local/bin/init.setup.sh
docker.local/bin/sync_clock.sh
```
Then go to individual miner/sharder:
```
../bin/start.b0sharder.sh (start sharders first!)
../bin/start.b0miner.sh
```
### Running on systems with SELinux enabled

Library by `herumi` for working with BLS threshold signatures requires this flag turned on:

```
setsebool -P selinuxuser_execheap 1
```

If you are curious about the reasons for this, this thread sheds some light on the topic:

https://github.com/herumi/xbyak/issues/9

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

## Dependencies for local compilation

You need to install `rocksdb` and `herumi/bls`, refer to `docker.local/build.base/Dockerfile.build_base` for necessary steps.

For local compilation it should be enough of `go build` from a submodule folder, e.g.
```
cd code/go/0chain.net/miner
go build
```

You can pass tag `development` if you want to simulate n2n delays.
And you also need tag `bn256` to build the same code as in production:
```
go build -tags "bn256 development"
```

## Debugging

### Debug builds of 0chain

If you want to run a debug 0chain build you can follow the details contained in the 
[`0chain/local` folder](https://github.com/0chain/0chain/blob/debug_builds/local/README.md).

Only one miner and one sharder can be run on any single machine, so you will need at least
three machines to for a working 0chain.

### Log files

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
$ cd code/go/0chain.net/my-pkg
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

## Initial states

The balance for the various nodes is setup in a `initial_state.yaml` file. 
This file is a list of node ids and token amounts.

The initial state yaml file is entered as a command line argument when 
running a sharder or miner, falling that the `0chain.yaml` 
`network.inital_states` entry is used to find the initial state file.

An example, that can be used with the preset ids, can be found at 
[0chian/docker.local/config/inital_state.yaml`](https://github.com/0chain/0chain/blob/master/docker.local/config/initial_state.yaml)

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
- In minio the folders do not get deleted and will cause a slight increase in volume over time.

# Integration tests

Refer to [conductor documentation](https://github.com/0chain/0chain/blob/master/code/go/0chain.net/conductor/README.md)