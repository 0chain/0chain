#!/bin/sh

rm -f got.bin

./zboxcli/zbox --wallet testing.json download \
    --allocation `cat ~/.zcn/allocation.txt` \
    --localpath=got.bin \
    --remotepath /remote/random.bin
