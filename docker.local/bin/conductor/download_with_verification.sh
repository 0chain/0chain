#!/bin/sh

set -e

remotepath=$1

# create random file
head -c 100M < /dev/urandom > update.bin

# update it
./zboxcli/zbox --wallet testing.json download \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --verifydownload \
    --localpath=download.bin \
    --remotepath=$remotepath