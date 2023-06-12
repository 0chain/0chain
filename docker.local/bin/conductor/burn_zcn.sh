#!/bin/bash
# burn zcn in 0chain network

./zwalletcli/zwallet create-wallet --wallet testing.json

for i in $(seq 2)
do
  ./zwalletcli/zwallet --wallet testing.json faucet --methodName pour --input "{Pay day}"
done

burn_zcn_output=$(./zwalletcli/zwallet bridge-burn-zcn --token 1 --wallet testing.json)

tx=$(echo $burn_zcn_output | sed "s/.*with txn: *\(.*\) T.*/\1/")

# get authorizers signatures
./zwalletcli/zwallet bridge-get-zcn-burn --hash $tx
