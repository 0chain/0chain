# TestNet Setup with Docker Containers

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

## Building and Starting the Nodes

1) Open 5 terminal tabs. Use the first one for building the containers by being in git/0chain directory. Use the next 3 for 3 miners and be in the respective miner<i> directories created above in docker.local. Use the 5th terminal and be in the sharder1 directory.

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

3) Starting the nodes. On each of the miner terminals use the commands (note the .. at the beginning. This is because, these commands are run from within the docker.local/<miner/sharder|i> directories and the bin is one level above relative to these directories)


```
$ ../bin/start.miner.sh block-size
```
If block-size is not specified, a default of 5000 is used. Block size argument only works in test mode

On the sharder terminal, use
```
$ ../bin/start.sharder.sh
```
## Generating Test Transactions

1) To build the miner_stress program from git/0chain directory
```$ ./docker.local/bin/build_txns_generator.sh```
2) To run the miner_stress program after starting the 3 miners
```$ ./docker.local/bin/generate_txns.sh num-txns```
If num-txns is not specified, then 25000 transactions are generated for each miner

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

The logs of the nodes are going to be stored in a file (currently appLogs). The typical issues that need to be debugged is errors in the log, why certain things have not happeend which requires reviewing the timestamp of a sequence of events in the network. Here is an example set of commands to do some debugging.

Find arrors in all the miner nodes (from git/0chain)
```
$ docker.local/bin/run_all.miner.sh grep ERROR appLogs
```
This gives a set of errors in the log. Say an error indicates a problem for a specific block, then
```
$ docker.local/bin/run_all.miner.sh grep block-id appLogs
```
gives all the logs related to that block-id (which is the specific hash you got from the earlier command)

To get the start time of all the rounds
```
$ docker.local/bin/run_all.miner.sh grep 'starting round' appLogs
```
This gives the start timestamps that can be used to correlate the events and their timings.

## Miscellaneous

### Cleanup

Get rid of old unused docker resources:
```
$ docker system prune
```
