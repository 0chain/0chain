#!/bin/bash
# burn zcn in 0chain network
burn_zcn_output=$(./zwalletcli/zwallet bridge-burn-eth --amount 1 --wallet testing.json)

tx=$(echo $burn_zcn_output | sed -n 's/.*burn \[OK\]: \(.*\)/\1/p')

# get authorizers signatures
./zwalletcli/zwallet bridge-mint-zcn --hash $tx --wallet testing.json
