#!/bin/bash
set -e

. ./env.sh


echo ""
echo ""
echo ""
echo ""
cd $zCurrent

echo "++++++++++++++++++++++++++++++++++++++++++++++++++++++++++"
echo "+ create folders $zCurrent/data/"
echo "++++++++++++++++++++++++++++++++++++++++++++++++++++++++++"

echo "    - 0dns"
[ -d "./data/0dns/log" ]|| mkdir -p ./data/0dns/log

for i in $(seq 1 2)
do
  echo "    - miner$i"
  [ -d "./data/miner$i/data/redis/state" ] ||  mkdir -p ./data/miner$i/data/redis/state
  [ -d "./data/miner$i/data/redis/transactions" ] || mkdir -p ./data/miner$i/data/redis/transactions
  [ -d "./data/miner$i/data/rocksdb" ] || mkdir -p ./data/miner$i/data/rocksdb
  [ -d "./data/miner$i/log" ] || mkdir -p ./data/miner$i/log
done


for i in $(seq 1 1)
do
  echo "    - sharder$i"
  [ -d "./data/sharder$i/data/blocks" ] || mkdir -p ./data/sharder$i/data/blocks
  [ -d "./data/sharder$i/data/rocksdb" ] || mkdir -p ./data/sharder$i/data/rocksdb
  [ -d "./data/sharder$i/data/cassandra" ] || mkdir -p ./data/sharder$i/data/cassandra
  [ -d "./data/sharder$i/config/cassandra" ] || mkdir -p ./data/sharder$i/config/cassandra
  cp $zChain/config/cassandra/* ./data/sharder$i/config/cassandra/.
  [ -d "./data/sharder$i/log" ] || mkdir -p ./data/sharder$i/log
done




echo ""
echo ""
echo ""
echo ""
echo "++++++++++++++++++++++++++++++++++++++++++++++++++++++++++"
echo "> sync hwclock "
echo "++++++++++++++++++++++++++++++++++++++++++++++++++++++++++"
docker run --rm --privileged alpine hwclock -s


echo ""
echo ""
echo ""
echo ""


echo "++++++++++++++++++++++++++++++++++++++++++++++++++++++++++"
echo "+ start containers:                                      +"
echo "++++++++++++++++++++++++++++++++++++++++++++++++++++++++++"
cd $zDNS
echo ""
echo "  - 0dns"
echo ""

VOLUMES_CONFIG="$zCurrent/config/0dns" VOLUMES_LOG="$zCurrent/data/0dns/log" VOLUMES_MONGO_DATA="$zCurrent/data/0dns/mongodata"  docker-compose -p 0dns -f docker.local/dev-docker-compose.yml up -d

#open http://127.0.0.1:9091/network


# cd $zChain/docker.local/bin


# for i in $(seq 1 1)
# do
#     echo ""
#     echo "  - sharder$i"
#     echo ""

#     VOLUMES_CONFIG="$zCurrent/config/0chain" VOLUMES_DATA="$zCurrent/data" SHARDER=$i docker-compose -p sharder$i -f ../build.sharder/dev-docker-compose.yml up 
#     #open http://127.0.0.1:717$i/_diagnostics

# done





# for i in $(seq 1 2)
# do
#     echo ""
#     echo "  - miner$i"
#     echo ""

#     #VOLUMES_CONFIG="$zCurrent/config/0chain" VOLUMES_DATA="$zCurrent/data" MINER=$i docker-compose -p miner$i -f ../build.miner/dev-docker-compose.yml up 
#     #open http://127.0.0.1:707$i/_diagnostics

# done







