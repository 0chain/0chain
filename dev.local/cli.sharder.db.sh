#!/bin/bash

sharder=sharder$1
postgres=postgres$1
cassandra=cassandra$1

root=$(pwd)

[ -d $root/data/$sharder/postgres ] || mkdir -p "$root/data/$sharder/postgres"
[ -d $root/data/$sharder/cassandra ] || mkdir -p "$root/data/$sharder/cassandra"

[ -d $root/data/$sharder/bin ] && rm -rf $root/data/$sharder/bin
[ -d $root/data/$sharder/sql ] && rm -rf $root/data/$sharder/sql
[ -d $root/data/$sharder/sql_script ] && rm -rf $root/data/$sharder/sql_script

cp -rf ../bin $root/data/$sharder/
cp -rf ../sql $root/data/$sharder/
cp -r ../docker.local/sql_script $root/data/$sharder/


num=$(docker ps -a --filter "name=^${postgres}$" | wc -l)
echo -n "[1/9] remove $postgres: "
[ $num -eq 2 ] && docker rm $postgres --force || echo " SKIPPED"


num=$(docker ps -a --filter "name=^${cassandra}$" | wc -l)
echo -n "[2/9] remove $cassandra: "
[ $num -eq 2 ] && docker rm $cassandra --force || echo " SKIPPED"


echo -n "[3/9] remove zchain_postgres_init: "
num=$(docker ps -a --filter "name=^zchain_postgres_init$" | wc -l)

[ $num -eq 2 ] && docker rm zchain_postgres_init --force || echo "[SKIP]"

echo -n "[4/9] remove zchain_cassandra_init: "
num=$(docker ps -a --filter "name=^zchain_cassandra_init$" | wc -l)

[ $num -eq 2 ] && docker rm zchain_cassandra_init --force || echo "[SKIP]"

echo -n "[5/9] install [$postgres, $cassandra]: " 
SHARDER=$1 docker-compose -p sharder"$1" -f ./sharder.docker-compose.yml up -d


echo "[6/9] install zchain_postgres_init"

docker run --name zchain_postgres_init \
--link $postgres:postgres \
-e  POSTGRES_PORT=5432 \
-e  POSTGRES_HOST=postgres \
-e  POSTGRES_USER=postgres  \
-e  POSTGRES_PASSWORD=postgres \
-v  $root/data/$sharder/bin:/zchain/bin \
-v  $root/data/$sharder/sql_script:/zchain/sql \
postgres:14 bash /zchain/bin/postgres-entrypoint.sh 

echo -n "[7/9] remove zchain_postgres_init: "
docker rm zchain_postgres_init --force


echo "[8/9] install zchain_cassandra_init"
docker run --name zchain_cassandra_init \
--link $cassandra:cassandra \
-v  $root/data/$sharder/bin:/0chain/bin \
-v  $root/data/$sharder/sql:/0chain/sql \
cassandra:3.11.4 bash /0chain/bin/cassandra-init.sh

echo -n "[9/9] remove zchain_cassandra_init: "
docker rm zchain_cassandra_init --force