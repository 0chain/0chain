#!/bin/sh

mkdir -p ../miner/data/rocksdb
mkdir -p ../miner/log
mkdir -p ../miner/config
cp ../../docker.local/config/sc.yaml ../miner/config/sc.yaml

mkdir -p ../sharder/data/blocks
mkdir -p ../sharder/data/rocksdb
mkdir -p ../sharder/log
mkdir -p ../sharder/config
cp ../../docker.local/config/sc.yaml ../sharder/config/sc.yaml