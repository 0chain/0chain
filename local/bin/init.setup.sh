#!/bin/sh

mkdir -p miner/data/rocksdb
mkdir -p miner/log
mkdir -p miner/config
cp config/0chain.yaml miner/config/0chain.yaml
cp config/sc.yaml miner/config/sc.yaml
cp config/b0magicBlock_2_miners_1_sharder.json miner/config/b0magicBlock_2_miners_1_sharder.json

mkdir -p sharder/data/blocks
mkdir -p sharder/data/rocksdb
mkdir -p sharder/log
mkdir -p sharder/config
cp config/0chain.yaml sharder/config/0chain.yaml
cp config/sc.yaml sharder/config/sc.yaml
cp config/b0magicBlock_2_miners_1_sharder.json sharder/config/b0magicBlock_2_miners_1_sharder.json
