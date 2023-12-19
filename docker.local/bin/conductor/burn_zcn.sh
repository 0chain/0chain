#!/bin/bash
# burn zcn in 0chain network

./zwalletcli/zwallet create-wallet --wallet testing.json

./zwalletcli/zwallet --wallet testing.json faucet --methodName pour --input "{Pay day}" --tokens 100

burn_zcn_output=$(./zwalletcli/zwallet bridge-burn-zcn --token 1 --wallet testing.json)

tx=$(echo $burn_zcn_output | sed "s/.*with txn: *\(.*\) T.*/\1/")

# get authorizers signatures
./zwalletcli/zwallet bridge-get-zcn-burn --hash $tx
