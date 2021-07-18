#!/bin/bash

sharder=sharder$1
docker=cassandra$1
echo $sharder
echo $docker
root=$(pwd)

num=$(docker ps -a --filter "name=$docker" | wc -l)
echo $num

[ $num -eq 2 ] && docker rm $docker --force

# -eq 1, only column header
[ $num -eq 1 ] && \
docker run --name $docker \
--restart always -p 904$1:9042 \
-v  $root/data/$sharder/cassandra:/var/lib/cassandra/data \
-d cassandra:3.11.4


echo Initializing cassandra

[ -d $root/data/$sharder ] || mkdir -p $root/data/$sharder

[ -d $root/data/$sharder/bin ] && rm -rf $root/data/$sharder/bin
[ -d $root/data/$sharder/sql ] && rm -rf $root/data/$sharder/sql

cp -rf ../bin $root/data/$sharder/
cp -rf ../sql $root/data/$sharder/


[ "$(docker ps -a | grep cassandra_init)" ] && docker rm cassandra_init --force


docker run --name cassandra_init \
--link $docker:cassandra \
-v  $root/data/$sharder/bin:/0chain/bin \
-v  $root/data/$sharder/sql:/0chain/sql \
cassandra:3.11.4 bash /0chain/bin/cassandra-init.sh

docker rm cassandra_init --force

