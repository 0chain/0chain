# TestNet Setup with Docker Containers

## Table of Contents

- [Initial Setup](#initial-setup)
	- [Directory Setup for Miners & Sharders](#directory-setup-for-miners-&-sharders)
	- [Setup Network](#setup-network)
- [Building and Starting the Nodes](#building-and-starting-the-nodes)
- [Generating Test Transactions](#generating-test-transactions)
- [Troubleshooting](#troubleshooting)
- [Debugging](#debugging)
- [Unit tests](#unittests)
- [Miscellaneous](#miscellaneous)
	- [Cleanup](#cleanup)

## Initial Setup

### Directory Setup for Miners & Sharders

In the git/0chain run the following command
```
$ ./docker.local/bin/init.setup.sh
```

### Setup Network

Setup a network called testnet0 for each of these node containers to talk to each other.

***Note: The config file should be providing the IP address of the nodes as per the IP addresses in this network.***
```
$ ./docker.local/bin/setup_network.sh
```

## Building the Nodes

1) Open 5 terminal tabs. Use the first one for building the containers by being in git/0chain directory. Use the next 3 for 3 miners and be in the respective miner<i> directories created above in docker.local. Use the 5th terminal and be in the sharder1 directory.

1.1) First build the base container, zchain_base
```
$ ./docker.local/bin/build.base.sh
```

2) Building the miners and sharders. From the git/0chain directory use

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
1) Use ./docker.local/config/0chain.yaml to configure the blockchain properties. The default options are setup for running the blockchain fast in development.

1.1) If you want the logs to appear on the console - change logging.console from false to true

1.2) If you want the debug statements in the logs to appear - change logging.level from 'info' to 'debug'

1.3) If you want to change the block size, set the value of server_chain.block.size

1.4) If you want to adjust the network relay time, set the value of network.relay_time

## Starting the nodes
1) Starting the nodes. On each of the miner terminals use the commands (note the .. at the beginning. This is because, these commands are run from within the docker.local/<miner/sharder|i> directories and the bin is one level above relative to these directories)

On the sharder terminal, use
```
$ ../bin/start.sharder.sh
```
Wait till the cassandra is started and the sharder is ready to listen to requests.

On the respective miner terminal, use
```
$ ../bin/start.miner.sh
```



## Setting up Cassandra Schema

The following is no longer required as the schema is automatically loaded.

Start the sharder service that also brings up the cassandra service. To run commands on cassandra, use the following command

```
$ ../bin/run.sharder.sh cassandra cqlsh
```

1) To create zerochain keyspace, do the following

```
$ ../bin/run.sharder.sh cassandra cqlsh -f /0chain/sql/zerochain_keyspace.sql
```

2) To create the tables, do the following

```
$ ../bin/run.sharder.sh cassandra cqlsh -k zerochain -f /0chain/sql/txn_summary.sql
```

3) When you want to truncate existing data (use caution), do the following

```
$ ../bin/run.sharder.sh cassandra cqlsh -k zerochain -f /0chain/sql/truncate_tables.sql
```

## Generating Test Transactions

There is no need to generate the test data separately. In development mode, the transaction data is automatically generated at a certain rate based on the block size.

However, you can use the <a href='https://github.com/0chain/block-explorer'>block explorer</a> to submit transactions, view the blocks and confirm the transactions.

## Monitoring the progress

1) Use <a href='https://github.com/0chain/block-explorer'>block explorer</a> to see the progress of the block chain.

2) In addition, use the '/_diagnostics' link on any node to view internal details of the blockchain and the node.

## Troubleshooting

1) Ensure the port mapping is all correct:
```
$ docker ps
```
This should display a few containers and should include containers with images miner1_miner, miner2_miner and miner3_miner and they should have the ports mapped like "0.0.0.0:7071->7071/tcp"

2) Confirming the servers are up and running. From a browser, visit

- http://localhost:7071/

- http://localhost:7072/

- http://localhost:7073/

to see the status of the miners.

3) Connecting to redis servers running within the containers (you are within the appropriate miner directories)

Default redis (used for clients and state):
```
$ ../bin/run.miner.sh redis redis-cli
```
Redis used for transactions:
```
$ ../bin/run.miner.sh redis_txns redis-cli
```
4) Connecting to cassandra used in the sharder (you are within the appropriate sharder directories)
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

## Miscellaneous

### Cleanup

Get rid of old unused docker resources:
```
$ docker system prune
```
