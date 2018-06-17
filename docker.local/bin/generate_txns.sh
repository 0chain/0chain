#!/bin/sh

TXNS=${1:-25000}
for i in $(seq 1 3);
do
  code/go/test/miner_stress --address "127.0.0.1:707$i" --num_clients 400 --num_txns $TXNS
done

