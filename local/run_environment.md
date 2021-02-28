# Run environment for 0Chain go executables

## Table of contents

- [Introduction](#introduction)
- [redis](#redis)
- [cassandra](#cassandra)

## Introduction

This document explains how to run 0Chain a sharder and miner on a machine, 
either Mac or Linux, outside a docker file. 
As the names of key items , such as database locations, 
are hardcoded, a maximum of only one sharder and one miner can run on one machine.

A working 0Chain network ideally needs at least three or four machines.
Some work could be done on less, but not if it requires fully functioning
blockchains. 

## redis

If you intend to run an 0Chain minor you will need two redis databases instances.
Assuming redis has been installed as in  
[install_dbs.md](https://github.com/0chain/0chain/blob/debug_builds/local/install_dbs.md)
then two redis severs can be started on separate terminals. 
0Chain's miner hardcodes them to be on port 6479 and 6479. 
Make sure any terminals running redis are shut down and run
```shell
sudo 0chain/local/bin/reset_redis.sh
```
This should start two terminals running on port 6478 and 6479.

## cassandra

If you intend to run an 0Chain sharder you will need a Cassandra database. 

Firstly cassandra 3 requires java-8, if you have other java implementations 
installed you will want to select java-8
```shell
sudo update-alternatives --config java
```
Now reset the cassandra cluster.
```shell
sudo 0chain/local/bin/reset_cassandra.sh
cqlsh -f 0chain/docker.local/config/cassandra/init.cql
```

### Configfile