#!/bin/sh

set -e

head -c 32428800 < /dev/urandom > b2.bin

./zboxcli/zbox --wallet testing.json update \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=b2.bin \
    --remotepath=/remote/b1.bin
