#!/bin/bash

sharder=sharder$1
postgres=postgres$1

root=$(pwd)

[ -d $root/data/$sharder/postgres ] || mkdir -p $root/data/$sharder/postgres

num=$(docker ps -a --filter "name=^${postgres}$" | wc -l)


echo -n "[1/5] remove $postgres: "
[ $num -eq 2 ] && docker rm $postgres --force || echo " SKIPPED"

echo -n "[2/5] install $postgres: " && \

docker run --name $postgres \
--restart always -p 553$1:5432 \
-e POSTGRES_PASSWORD=postgres \
-e POSTGRES_PORT=5432 \
-e POSTGRES_HOST=postgres \
-e POSTGRES_USER=postgres  \
-e POSTGRES_PASSWORD=postgres \
-e POSTGRES_HOST_AUTH_METHOD=trust \
-v $root/data/$sharder/postgres:/var/lib/postgresql/data \
-v $root/data/$sharder/sql_script/:/docker-entrypoint-initdb.d/ \
-d postgres:14


[ -d $root/data/$sharder/bin ] && rm -rf $root/data/$sharder/bin
[ -d $root/data/$sharder/sql ] && rm -rf $root/data/$sharder/sql

cp -rf ../bin $root/data/$sharder/
cp -rf ../sql $root/data/$sharder/


echo -n "[3/5] remove zchain_postgres_init: "
num=$(docker ps -a --filter "name=^zchain_postgres_init$" | wc -l)

[ $num -eq 2 ] && docker rm zchain_postgres_init --force || echo "[SKIP]"



echo "[4/5] install zchain_postgres_init"

docker run --name zchain_postgres_init \
--link $postgres:postgres \
-e  POSTGRES_PORT=5432 \
-e  POSTGRES_HOST=postgres \
-e  POSTGRES_USER=postgres  \
-e  POSTGRES_PASSWORD=postgres \
-v  $root/data/$sharder/bin:/zchain/bin \
-v  $root/data/$sharder/sql:/zchain/sql \
postgres:14 bash /zchain/bin/postgres-entrypoint.sh 

echo -n "[5/5] remove zchain_postgres_init: "
docker rm zchain_postgres_init --force
