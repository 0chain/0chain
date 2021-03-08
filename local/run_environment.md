# Run environment for 0Chain go executables

## Table of contents

- [Introduction](#introduction)
- [Configure files](#configure-files)
  - [Node keys](#node-keys)
  - [Magic block](#magic-block)  
  - [0chain.yaml](#0chain-yaml)
- [Reset databases](#reset-databases)  
  - [Redis](#redis)
  - [Cassandra](#cassandra)

## Introduction

This document explains how to run 0Chain a sharder and miner on a machine, 
either Mac or Linux, outside a docker file. 
As the names of key items, such as database locations, 
are hardcoded, a maximum of only one sharder and one miner can run on one machine.

A working 0Chain network ideally needs at least three or four machines.
Some work could be done on less, but not if it requires fully functioning
blockchains. 

## Configure files

The structure of the network is determined by the configuration. Each machine

### Node keys

The node keys gives each node its identification. There are examples of node
key files in `0chain/docker.local/config/b0*node*_keys.txt`. One of these files
is linked to each node by the node's runtime command option `--keys_file`.

For example
```shell
miner --keys_file config/b0mnode2_keys.txt
sharder --keys_file config/b0snode1_keys.txt
```

The other configuration files will refer to a node by the `id` is as defined 
by its `keys_file`. For information purposes links between each id, public key
and private key triplet is given in `docker.local\config\magicBlock_5_miners_1_sharder.yaml` 
and `docker.local\config\magicBlock_3_miners_3_sharder.yaml`.

### Magic Block

To start the chain off we need a genesis magic block file. There are two 
templates set up as examples `b0magicBlock_4_miners_1_sharder.tmp.json` and
`b0magicBlock_3_miners_1_sharder.tmp.json`. Fill in the missing details of the 
IP address of each machine in the 0chain in the `n2n_host` fields.
Each node should have a json-identical magic block file.

Each node object needs the `n2n_host` field filled in with the IP address of the 
machine. When each miner and sharder is run, the `--keys_file` option must match
the `id` field of the corresponding node, as indicated by the 
`magicBlock_3_miners_3_sharder.yaml` file.

The `t` and `n` fields must also be consistant the number of nodes in the 0chain. 
In particular
* `n` is the number of nodes in the dkg
* `t` actual node threshold for signatures 

Simplified example
```json
{
  "miners": {
    "nodes": {
       "1": {
         "id" : 1,
         "n2n_host" : "127.0.0.77"
       },
      "2": {
        "id" : 2,
        "n2n_host" : "127.0.0.85"
      },
      "3": {
        "id" : 3,
        "n2n_host" : "127.0.0.92"
      }
    }
  },
  "sharders": {
    "nodes": {
      "1": {
        "id": 1,
        "n2n_host": "127.0.0.77"
      }
    }
  },
  "t": 2,
  "n": 3
}
```
On machine `127.0.0.77` run
```shell
miner --keys_file config/b0mnode1_keys.txt
sharder --keys_file config/b0snode1_keys.txt
```
on machine `127.0.0.85` run
```shell
miner --keys_file config/b0mnode2_keys.txt
```
and on machine `127.0.0.92` run
```shell
miner --keys_file config/b0mnode3_keys.txt
```

### 0chain yaml

0chain.yaml contains many configuration details for the chain. Each
machine should have identical copies available for each miner and sharder
in its config directory.

The only field we consider here is the `magic_block_file` entry, this should
match the name of the magic file [discussed above](#magic-block)

For example
```yaml
network:
  magic_block_file: config/b0magicBlock_4_miners_1_sharder.json
```

## Reset databases

### Redis

If you intend to run a 0Chain miner you will need two Redis databases instances.
Assuming Redis has been installed as in  
[install_dbs.md](https://github.com/0chain/0chain/blob/debug_builds/local/install_dbs.md)
then two Redis severs can be started on separate terminals. 
0Chain's miner hardcodes them to be on port 6479 and 6479. 
Make sure any terminals running Redis are shut down.
You might have to force the 6379 and 6479 ports to shut down.
```shell
sudo 0chain/local/bin/reset.redis.sh
```
This should start two terminals running on port 6478 and 6479.

### Cassandra

If you intend to run an 0Chain sharder you will need a Cassandra database. 

Firstly Cassandra 3 requires java-8, if you have other java implementations 
installed you will want to select java-8
```shell
sudo update-alternatives --config java
```
To start Cassandra
```shell
service cassandra stop
rm -rf /var/lib/cassandra/*
service cassandra start
```
You might have to wait for cqlsh to come up.
```shell
cqlsh
Connected to Test Cluster at 127.0.0.1:9042.
[cqlsh 5.0.1 | Cassandra 3.11.10 | CQL spec 3.4.4 | Native protocol v4]
Use HELP for help.
cqlsh>
```
Now run, in order, the cqlsh scripts `0chian\docker.local\config\cassandra\init.cql`,
`0chain\sql\zerochain_keyspace.sql`, `magic_block_map.sql` and `txn_summary.sql`
respectively. 
