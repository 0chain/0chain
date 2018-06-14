#!/bin/sh
num_txns=${1:-1000}
docker.local/bin/generate_txns.sh $num_txns
wget -q -O - http://localhost:7171/_start
wget -q -O - http://localhost:7071/_start
