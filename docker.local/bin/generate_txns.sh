#!/bin/sh

TXNS=${1:-25000}
code/go/test/miner_stress --address "127.0.0.1:7071" --num_clients 400 --num_txns $TXNS
code/go/test/miner_stress --address "127.0.0.1:7072" --num_clients 400 --num_txns $TXNS 
code/go/test/miner_stress --address "127.0.0.1:7073" --num_clients 400 --num_txns $TXNS

