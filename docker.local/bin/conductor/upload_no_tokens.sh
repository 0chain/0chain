#!/bin/sh

set -e

# create random file
head -c 52428800 < /dev/urandom > random.bin

# upload it
./zboxcli/zbox --wallet testing.json upload \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=upload.bin \
    --remotepath=/remote/upload.bin
