#!/bin/sh

set -e

head -c 1024 < /dev/urandom > s2.bin

./zboxcli/zbox --wallet testing.json update \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=s2.bin \
    --remotepath=/remote/s1.bin
