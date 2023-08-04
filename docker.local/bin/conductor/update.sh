#!/bin/sh

set -e

# create random file
head -c 5M < /dev/urandom > update.bin

# update it
./zboxcli/zbox --wallet testing.json update \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=update.bin \
    --remotepath=/remote/upload.bin