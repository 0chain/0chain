#!/bin/sh

set -e

rm -rf ~/.zcn/testing.json

# try few times to affect random sharders choose in Go SDK
for run in $(seq 1 10)
do
	./zwalletcli/zwallet --wallet testing.json faucet --methodName pour --input "{Pay day}" || exit
done
