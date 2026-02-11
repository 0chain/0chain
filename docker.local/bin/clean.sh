#!/bin/sh

# Kill any running background scripts
for script in vc.sh chaos.sh monitor.sh; do
  pids=$(ps aux | grep "[/]$script" | awk '{print $2}')
  if [ -n "$pids" ]; then
    echo "killing $script (PIDs: $pids)"
    echo "$pids" | xargs kill 2>/dev/null
  fi
done

rm -rf docker.local/sql/*

for i in $(seq 1 8)
do
  echo "deleting miner$i logs"
  rm -rf docker.local/miner"$i"/log/*
  echo "deleting miner$i redis db"
  rm -rf docker.local/miner"$i"/data/redis/state/*
  rm -rf docker.local/miner"$i"/data/redis/transactions/*
  echo "deleting miner$i rocksdb db"
  rm -rf docker.local/miner"$i"/data/rocksdb/config*
  rm -rf docker.local/miner"$i"/data/rocksdb/mb*
  rm -rf docker.local/miner"$i"/data/rocksdb/state*
  rm -rf docker.local/miner"$i"/data/rocksdb/dkg*
done

for i in $(seq 1 4)
do
  echo "deleting sharder$i logs"
  rm -rf docker.local/sharder"$i"/log/*
  echo "deleting sharder$i cassandra db"
  rm -rf docker.local/sharder"$i"/data/cassandra/*
  echo "deleting sharder$i rocksdb db"
  rm -rf docker.local/sharder"$i"/data/rocksdb/*
  echo "delete sharder$i postgres db"
  rm -rf docker.local/sharder"$i"/data/postgresql/*
  rm -rf docker.local/sharder"$i"/data/postgresql2/*
done

for i in $(seq 1 4)
do
  echo "deleting sharder$i blocks on the file system"
  rm -rf docker.local/sharder"$i"/data/blocks/*
done

echo "deleting kafka config"
rm -rf docker.local/kafka/config/*
echo "deleting kafka data"
rm -rf docker.local/kafka/data/*