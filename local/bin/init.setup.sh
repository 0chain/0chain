#!/bin/sh

mkdir -p "$(dirname "$0")"/../miner/data/rocksdb
mkdir -p "$(dirname "$0")"/../miner/log
mkdir -p "$(dirname "$0")"/../miner/config
cp "$(dirname "$0")"/../../docker.local/config/sc.yaml "$(dirname "$0")"/../miner/config/sc.yaml
cp "$(dirname "$0")"/../../docker.local/config/n2n_delay.yaml "$(dirname "$0")"/../miner/config/n2n_delay.yaml

mkdir -p "$(dirname "$0")"/../sharder/data/blocks
mkdir -p "$(dirname "$0")"/../sharder/data/rocksdb
mkdir -p "$(dirname "$0")"/../sharder/log
mkdir -p "$(dirname "$0")"/../sharder/config
cp "$(dirname "$0")"/../../docker.local/config/sc.yaml "$(dirname "$0")"/../sharder/config/sc.yaml
cp "$(dirname "$0")"/../../docker.local/config/minio_config.txt "$(dirname "$0")"/../sharder/config/minio_config.txt