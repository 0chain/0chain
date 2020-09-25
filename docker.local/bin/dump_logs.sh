#!/bin/sh

mkdir -pv docker.local/log_dumps

tar -czvf "docker.local/log_dumps/$(date -u '+%Y.%m.%d-%H.%M.%S').logs.tar.gz" \
    docker.local/{miner,sharder}{1,2,3,4,5,6,7,8}/log/{0chain,n2n}.log 2>/dev/null
