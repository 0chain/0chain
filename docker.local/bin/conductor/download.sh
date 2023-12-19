#!/bin/sh

set -e

remote_path=$1

rm -f got.bin

./zboxcli/zbox --wallet testing.json download \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=got.bin \
    --remotepath "$remote_path"
