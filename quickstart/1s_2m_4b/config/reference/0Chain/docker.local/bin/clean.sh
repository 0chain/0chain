#!/bin/sh

for i in $(seq 1 8)
do
  echo "deleting miner$i logs"
  rm -rf docker.local/miner$i/log/*
  echo "deleting miner$i redis db"
  rm -rf docker.local/miner$i/data/redis/state/*
  rm -rf docker.local/miner$i/data/redis/transactions/*
  echo "deleting miner$i rocksdb db"
  rm -rf docker.local/miner$i/data/rocksdb/*
done

for i in $(seq 1 3)
do
  echo "deleting sharder$i logs"
  rm -rf docker.local/sharder$i/log/*
  echo "deleting sharder$i cassandra db"
  rm -rf docker.local/sharder$i/data/cassandra/*
  echo "deleting sharder$i rocksdb db"
  rm -rf docker.local/sharder$i/data/rocksdb/*
done

for i in $(seq 1 3)
do
  echo "deleting sharder$i blocks on the file system"
  rm -rf docker.local/sharder$i/data/blocks/*
done
