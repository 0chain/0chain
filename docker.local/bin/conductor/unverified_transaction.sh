#!/bin/sh

set -e

rm -rf ~/.zcn/testing.json

./zwalletcli/zwallet --wallet testing.json faucet --methodName pour --input “{Pay day}”
