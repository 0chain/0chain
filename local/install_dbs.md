# Install 0Chain databases

## Table of contents

- [Introduction](#Introduction)
- [Redis](#Redis)
- [Cassandra](#Cassandra)

## Introduction

0Chain makes use of there different types of databases:
- RocksDB
- Redis
- Cassandra

rocksDB's usage will all be handled by the application, and
its installation was outlined in the 
[bulding 0chain document](https://github.com/0chain/0chain/blob/debug_builds/local/build_environment.md#install-rocksdb)
We will outline installing Redis and Cassandra here.

## Redis

If you are going to be running miners on the machine you need to install Redis.
```shell
sudo apt update
sudo apt install -y redis-server
```
TODO: need to set up `/etc/systemd/system/redis.service` so that
both Redis instances run with custom configure files. Just using
` redis-server "path/to/redis.conf"` fails.

## Cassandra

If you intend to run a sharder you need to install Cassandra. 

Cassandra requires java-8, so if you don't have java 8 set up. 
```shell
sudo apt update
sudo apt install -y openjdk-8-jdk
```
`java -version` might not show 8. Not a problem you will need
to run `sudo update-alternatives --config java` sometime before
you start running 0Chain.

You will probably want to update your `.profile` file with
```shell
export JAVA_HOME=/usr/lib/jvm/java-1.8.0-openjdk-amd64
export PATH=$PATH:$JAVA_HOME/bin
```
Now to install Cassandra
```shell
wget -q -O - https://www.apache.org/dist/cassandra/KEYS | sudo apt-key add --
sudo sh -c 'echo "deb http://www.apache.org/dist/cassandra/debian 311x main" \
> /etc/apt/sources.list.d/cassandra.sources.list'
sudo apt update
sudo apt install -y cassandra
```
Use the `cassandra.yaml` file provided by 0chain.
```shell
sudo mv /etc/cassandra/cassandra.yaml /etc/cassandra/cassandra.yaml.backup
sudo cp 0chain/docker.local/config/cassandra/cassandra.yaml /etc/cassandra/cassandra.yaml
```
Cassandra tools require python2 to run, however recent versions of Ubuntu have only python3 
installed by default; if this is the case check out
[how to install python 2](https://linuxconfig.org/install-python-2-on-ubuntu-20-04-focal-fossa-linux),
in any case you might want to consider using a
[Python version switch manager](https://linuxconfig.org/ubuntu-20-04-python-version-switch-manager)

