#!/bin/sh

echo "cleaning 8 miners..."
for i in $(seq 1 8)
do
  echo "deleting miner$i logs"
  rm -rf ./miner$i/log/*
  echo "deleting miner$i redis db"
  rm -rf ./miner$i/data/redis/state/*
  rm -rf ./miner$i/data/redis/transactions/*
  echo "deleting miner$i rocksdb db"
  rm -rf ./miner$i/data/rocksdb/*
done

echo "cleaning 4 sharders..."
for i in $(seq 1 4)
do
  echo "deleting sharder$i logs"
  rm -rf ./sharder$i/log/*
  echo "deleting sharder$i cassandra db"
  rm -rf ./sharder$i/data/cassandra/*
  echo "deleting sharder$i rocksdb db"
  rm -rf ./sharder$i/data/rocksdb/*
  echo "deleting sharder$i blocks on the file system"
  rm -rf ./sharder$i/data/blocks/*
  echo "deleting sharder$i postgresql db"
  rm -rf ./sharder$i/data/postgresql/*
done

echo "cleaned up"
