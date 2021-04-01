#!/bin/sh

set -e

head -c 32428800 < /dev/urandom > b1.bin

# upload a file to download it
./zboxcli/zbox --wallet testing.json upload \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=b1.bin \
    --remotepath=/remote/b1.bin

rm -f gb1.bin

./zboxcli/zbox --wallet testing.json download \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=gb1.bin \
    --remotepath /remote/b1.bin
