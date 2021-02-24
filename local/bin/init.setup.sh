#!/bin/sh

for i in $(seq 1 8)
do
  mkdir -p local/miner$i/data/rocksdb
  mkdir -p local/miner$i/log
  mkdir -p local/miner$i/config
  cp local/config/0chain.yaml local/miner$i/config/0chain.yaml
  cp local/config/sc.yaml local/miner$i/config/sc.yaml
  cp local/config/b0mnode1_keys.txt local/miner$i/config/b0mnode1_keys.txt
  cp local/config/n2n_delay.yaml local/miner$i/config/n2n_delay.yaml
  cp local/config/b0magicBlock_2_miners_1_sharder.json local/miner$i/config/b0magicBlock_2_miners_1_sharder.json
done

for i in $(seq 1 3)
do
  mkdir -p local/sharder$i/data/blocks
  mkdir -p local/sharder$i/data/rocksdb
  mkdir -p local/sharder$i/log
  mkdir -p local/sharder$i/config
  cp local/config/0chain.yaml local/sharder$i/config/0chain.yaml
  cp local/config/sc.yaml local/sharder$i/config/sc.yaml
  cp local/config/b0magicBlock_2_miners_1_sharder.json local/sharder$i/config/b0magicBlock_2_miners_1_sharder.json
done
