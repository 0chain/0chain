#!/bin/sh
num_txns=${1:-1000}
echo Submitting $num_txns transactions to each miner
docker.local/bin/generate_txns.sh $num_txns

echo Clearing the sharder state
curl http://localhost:7171/_start -o -

echo Starting the block generation process
curl http://localhost:7071/_start -o -
