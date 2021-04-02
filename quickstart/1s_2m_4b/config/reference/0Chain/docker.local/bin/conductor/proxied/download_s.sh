#!/bin/sh

set -e

rm -f gs1.bin

./zboxcli/zbox --wallet testing.json download \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=gs1.bin \
    --remotepath=/remote/s1.bin
