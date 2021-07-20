#!/bin/bash

sharder=sharder$1
cassandra=cassandra$1

root=$(pwd)

[ -d $root/data/$sharder/cassandra ] || mkdir -p $root/data/$sharder/cassandra

num=$(docker ps -a --filter "name=^${cassandra}$" | wc -l)


echo -n "[1/5] remove $cassandra: "
[ $num -eq 2 ] && docker rm $cassandra --force || echo " SKIPPED"

echo -n "[2/5] install $cassandra: " && \
docker run --name $cassandra \
--restart always -p 904$1:9042 \
-v  $root/data/$sharder/cassandra:/var/lib/cassandra/data \
-d cassandra:3.11.4



[ -d $root/data/$sharder/bin ] && rm -rf $root/data/$sharder/bin
[ -d $root/data/$sharder/sql ] && rm -rf $root/data/$sharder/sql

cp -rf ../bin $root/data/$sharder/
cp -rf ../sql $root/data/$sharder/


echo -n "[3/5] remove cassandra_init: "
num=$(docker ps -a --filter "name=^cassandra_init$" | wc -l)

[ $num -eq 2 ] && docker rm cassandra_init --force || echo "[SKIP]"



echo "[4/5] install cassandra_init"
docker run --name cassandra_init \
--link $cassandra:cassandra \
-v  $root/data/$sharder/bin:/0chain/bin \
-v  $root/data/$sharder/sql:/0chain/sql \
cassandra:3.11.4 bash /0chain/bin/cassandra-init.sh


echo -n "[5/5] remove cassandra_init: "
docker rm cassandra_init --force

