# Install 0Chain databases

## Table of contents

- [Introduction](#introduction)
- [redis](#redis)
- [cassandra](#cassandra)

## Introduction

0Chain makes use of there different types of database,
- rocksDB
- redis
- cassandra

rocksDB's usage will all be handled by the application, and
its installation was outlined in the 
[bulding 0chain document](https://github.com/0chain/0chain/blob/debug_builds/local/build_0chain.md#install-rocksdb)
We will outline installing redis and cassandra here.

## redis

If you are going to be running miners on the machine you need to install redis.
```shell
sudo apt update
sudo apt install -y redis-server
```
todo: need to set up `/etc/systemd/system/redis.service` so that
both redis instances run with custom configure files. Just using
` redis-server "path/to/redis.conf"` fails.

## cassandra

If you intend to run a sharder you need to install cassandra. 

Cassandra requires java, so if you don't have java installed yet
```shell
sudo apt update
sudo apt install -y openjdk-14-jdk
```
`java -verion` should now sho 14.0.2. Now to install cassandra
```shell
wget -q -O - https://www.apache.org/dist/cassandra/KEYS | sudo apt-key add --
sudo sh -c 'echo "deb http://www.apache.org/dist/cassandra/debian 311x main" \
> /etc/apt/sources.list.d/cassandra.sources.list'
sudo apt update
sudo apt install -y cassandra
```

