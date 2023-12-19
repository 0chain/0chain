#!/bin/sh

set -e

# create random file
head -c 5M < /dev/urandom > upload.bin

remotepath=$1

echo "REMOTEPATH = $remotepath"

# upload it
./zboxcli/zbox --wallet testing.json upload \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=upload.bin \
    --remotepath=$remotepath
