#!/bin/sh

for i in $(seq 1 3)
do
  rm docker.local/miner$i/log/*
  rm -rf docker.local/miner$i/data/redis/state/*
  rm -rf docker.local/miner$i/data/redis/transactions/*
  rm -rf docker.local/miner$i/data/rocksdb/*
done

for i in $(seq 1 3)
do
  rm docker.local/sharder$i/log/*
  rm -rf docker.local/sharder$i/data/cassandra/*
  rm -rf docker.local/sharder$i/data/blocks/*
  rm -rf docker.local/sharder$i/data/rocksdb/*
done
