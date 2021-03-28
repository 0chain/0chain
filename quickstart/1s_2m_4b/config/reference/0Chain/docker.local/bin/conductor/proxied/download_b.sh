#!/bin/sh

set -e

rm -f gb1.bin

./zboxcli/zbox --wallet testing.json download \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=gb1.bin \
    --remotepath=/remote/b1.bin
