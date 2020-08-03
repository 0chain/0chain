#!/bin/sh

rm -f got.bin

./zboxcli/zbox --wallet testing.json list \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --remotepath /remote
