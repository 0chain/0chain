#!/bin/sh

for i in $(seq 1 8)
do
  echo "deleting miner$i rocksdb db for dkg"
  rm -rf docker.local/miner$i/data/rocksdb/dkg*
done