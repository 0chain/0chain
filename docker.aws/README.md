## Initial Setup

### Directory Setup for Miners & Sharders

In the git/0chain run the following command

```
$ ./docker.local/bin/init.setup.sh
```

### Setup Network

Setup a network called testnet0 for each of these node containers to talk to each other.

```
$ ./docker.local/bin/setup_network.sh
```
## Modify Config and Keys files

1. Update `network . dns_url` in `./docker.local/config/0chain.yaml` to point to `http://<network_url>/dns`

2. Modify `docker.local/config/b0snode2_keys.txt` and replace `localhost` and `198.18.0.83` with public ip of your instance / vm.

3. Modify `docker.local/config/b0mnode5_keys.txt` and replace `localhost` and `198.18.0.83` with public ip of your instance / vm.

## Building the Nodes

1. Open 2 terminal tabs.

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

2.3) Syncing time (the host and the containers are being offset by a few seconds that throws validation errors as we accept transactions that are within 5 seconds of creation). This step is needed periodically when you see the validation error.

```
$ ./docker.local/bin/sync_clock.sh

```

## Starting the nodes

1. To start sharder container `cd docker.local/sharder2`

```
$ ../bin/start.b0sharder.sh
```

Wait till the cassandra is started and the sharder is ready to listen to requests.

2. To start sharder container `cd docker.local/miner5` in other terminal.


```
$ ../bin/start.b0miner.sh
```


