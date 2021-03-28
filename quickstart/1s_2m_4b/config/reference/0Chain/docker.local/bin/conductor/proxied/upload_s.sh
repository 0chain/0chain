#!/bin/sh

set -e

head -c 1024 < /dev/urandom > s1.bin

./zboxcli/zbox --wallet testing.json upload \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=s1.bin \
    --remotepath=/remote/s1.bin
