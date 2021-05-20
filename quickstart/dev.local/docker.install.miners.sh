#!/bin/bash
set -e

. ./env.sh



cd $zChain/docker.local/bin




for i in $(seq 1 2)
do
    echo ""
    echo "  - miner$i"
    echo ""

    VOLUMES_CONFIG="$zCurrent/config/0chain" VOLUMES_DATA="$zCurrent/data" MINER=$i docker-compose -p miner$i -f ../build.miner/dev-docker-compose.yml up -d
    #open http://127.0.0.1:707$i/_diagnostics

done







